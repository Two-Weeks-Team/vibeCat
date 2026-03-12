import Foundation
import ScreenCaptureKit
import CoreGraphics
import AppKit
import VibeCatCore

/// Captures a region of the screen using ScreenCaptureKit.
/// Excludes the VibeCat panel itself from captures.
@MainActor
final class ScreenCaptureService {
    enum MouseWindowTargetingGeometry {
        /// Convert AppKit screen coordinates (Y=0 at bottom-left of primary display)
        /// to CoreGraphics/Quartz coordinates (Y=0 at top-left of primary display).
        /// Used for AXUIElementCopyElementAtPosition and CGWindowList bounds comparison.
        static func appKitToCGPoint(_ appKitPoint: CGPoint) -> CGPoint {
            let primaryHeight = NSScreen.screens.first?.frame.height ?? 0
            return CGPoint(x: appKitPoint.x, y: primaryHeight - appKitPoint.y)
        }

        static func windowBoundsContainMouse(_ bounds: CGRect, mouseLocation: CGPoint) -> Bool {
            let cgPoint = appKitToCGPoint(mouseLocation)
            return bounds.contains(cgPoint)
        }
    }

    struct CaptureSnapshot {
        enum TargetKind: String {
            case windowUnderCursor = "window_under_cursor"
            case frontmostWindow = "frontmost_window"
            case display = "display"
            case displayFallback = "display_fallback"
        }

        let image: CGImage
        let appName: String
        let appBundleID: String?
        let windowTitle: String?
        let targetKind: TargetKind
        let targetIdentity: String
        let displayID: String
        let capturedAt: Date
        let screenBasisID: String

        var contextDescription: String {
            var parts = [
                "app=\(appName)",
                "target=\(targetKind.rawValue)",
                "display=\(displayID)",
            ]
            if let appBundleID, !appBundleID.isEmpty {
                parts.append("bundle=\(appBundleID)")
            }
            if let windowTitle, !windowTitle.isEmpty {
                parts.append("window=\(windowTitle)")
            }
            return "[" + parts.joined(separator: " ") + "]"
        }
    }

    struct WindowProbeSnapshot {
        let appName: String
        let appBundleID: String?
        let windowTitle: String?
        let displayID: String
        let targetKind: CaptureSnapshot.TargetKind
    }

    private struct AXWindowProbe {
        let appName: String
        let appBundleID: String?
        let windowTitle: String?
    }

    enum CaptureResult {
        case captured(CaptureSnapshot)
        case unchanged
        case unavailable(String)
    }

    private var lastImage: CGImage?
    private var lastTargetIdentity: String?
    var probePointProvider: (() -> CGPoint)?

    private func currentProbePoint() -> CGPoint {
        probePointProvider?() ?? NSEvent.mouseLocation
    }

    // MARK: - Public API

    /// Capture the screen. Returns .unchanged if screen hasn't changed significantly.
    func captureAroundCursor() async -> CaptureResult {
        do {
            let snapshot = try await performCapture(mode: selectedCaptureMode())
            if snapshot.targetIdentity != lastTargetIdentity {
                lastImage = nil
                lastTargetIdentity = snapshot.targetIdentity
            }

            if !ImageDiffer.hasSignificantChange(from: lastImage, to: snapshot.image) {
                return .unchanged
            }
            lastImage = snapshot.image
            return .captured(snapshot)
        } catch {
            return .unavailable(error.localizedDescription)
        }
    }

    /// Capture the full frontmost window (for high-significance analysis).
    func captureFullWindow() async -> CaptureResult {
        do {
            let snapshot = try await performCapture(mode: .frontmostWindow)
            lastTargetIdentity = snapshot.targetIdentity
            lastImage = snapshot.image
            return .captured(snapshot)
        } catch {
            return .unavailable(error.localizedDescription)
        }
    }

    /// Force a capture regardless of change detection.
    func forceCapture() async -> CaptureResult {
        do {
            let snapshot = try await performCapture(mode: selectedCaptureMode())
            lastTargetIdentity = snapshot.targetIdentity
            lastImage = snapshot.image
            return .captured(snapshot)
        } catch {
            return .unavailable(error.localizedDescription)
        }
    }

    func probeWindowUnderCursor() async -> WindowProbeSnapshot? {
        let mouseLocation = currentProbePoint()
        let displayID = mouseDisplayID(mouseLocation: mouseLocation)
        if let probe = axWindowProbe(at: mouseLocation),
           probe.appBundleID != Bundle.main.bundleIdentifier {
            return WindowProbeSnapshot(
                appName: probe.appName,
                appBundleID: probe.appBundleID,
                windowTitle: probe.windowTitle,
                displayID: displayID,
                targetKind: .windowUnderCursor
            )
        }
        guard let windowInfos = CGWindowListCopyWindowInfo([.optionOnScreenOnly, .excludeDesktopElements], kCGNullWindowID) as? [[String: Any]] else {
            return nil
        }

        let bundleID = Bundle.main.bundleIdentifier

        for info in windowInfos {
            let layer = info[kCGWindowLayer as String] as? Int ?? 0
            guard layer == 0 else { continue }
            guard let boundsDict = info[kCGWindowBounds as String] as? NSDictionary else { continue }

            var bounds = CGRect.zero
            guard CGRectMakeWithDictionaryRepresentation(boundsDict, &bounds), MouseWindowTargetingGeometry.windowBoundsContainMouse(bounds, mouseLocation: mouseLocation) else {
                continue
            }

            let ownerPID = info[kCGWindowOwnerPID as String] as? pid_t ?? 0
            let runningApp = NSRunningApplication(processIdentifier: ownerPID)
            if runningApp?.bundleIdentifier == bundleID {
                continue
            }

            return WindowProbeSnapshot(
                appName: (info[kCGWindowOwnerName as String] as? String)
                    ?? runningApp?.localizedName
                    ?? "Unknown",
                appBundleID: runningApp?.bundleIdentifier,
                windowTitle: normalizedTitle(info[kCGWindowName as String] as? String),
                displayID: displayID,
                targetKind: .windowUnderCursor
            )
        }

        let frontApp = NSWorkspace.shared.frontmostApplication
        let effectiveApp: NSRunningApplication?
        if frontApp?.bundleIdentifier == Bundle.main.bundleIdentifier {
            effectiveApp = NSWorkspace.shared.runningApplications
                .first { $0.isActive && $0.bundleIdentifier != Bundle.main.bundleIdentifier }
                ?? frontApp
        } else {
            effectiveApp = frontApp
        }
        return WindowProbeSnapshot(
            appName: effectiveApp?.localizedName ?? "Unknown",
            appBundleID: effectiveApp?.bundleIdentifier,
            windowTitle: nil,
            displayID: displayID,
            targetKind: .displayFallback
        )
    }

    // MARK: - Private

    private func performCapture(mode: CaptureMode) async throws -> CaptureSnapshot {
        let content = try await SCShareableContent.excludingDesktopWindows(false, onScreenWindowsOnly: true)

        guard !content.displays.isEmpty else {
            throw CaptureError.noDisplay
        }

        let mouseLocation = currentProbePoint()
        let display = displayContainingMouse(from: content, mouseLocation: mouseLocation)
        let excludedApps = content.applications.filter { app in
            app.bundleIdentifier == Bundle.main.bundleIdentifier
        }

        let config = SCStreamConfiguration()
        config.width = display.width * 2
        config.height = display.height * 2
        config.pixelFormat = kCVPixelFormatType_32BGRA
        config.showsCursor = false
        config.capturesAudio = false

        switch mode {
        case .windowUnderCursor:
            if let target = windowUnderCursor(from: content, mouseLocation: mouseLocation) {
                let filter = SCContentFilter(desktopIndependentWindow: target.window)
                let image = try await SCScreenshotManager.captureImage(contentFilter: filter, configuration: config)
                return CaptureSnapshot(
                    image: image,
                    appName: target.appName,
                    appBundleID: target.appBundleID,
                    windowTitle: target.windowTitle,
                    targetKind: .windowUnderCursor,
                    targetIdentity: "window:\(target.window.windowID)",
                    displayID: String(display.displayID),
                    capturedAt: Date(),
                    screenBasisID: UUID().uuidString.lowercased()
                )
            }

        case .frontmostWindow:
            if let target = frontmostWindow(from: content) {
                let filter = SCContentFilter(desktopIndependentWindow: target.window)
                let image = try await SCScreenshotManager.captureImage(contentFilter: filter, configuration: config)
                return CaptureSnapshot(
                    image: image,
                    appName: target.appName,
                    appBundleID: target.appBundleID,
                    windowTitle: target.windowTitle,
                    targetKind: .frontmostWindow,
                    targetIdentity: "window:\(target.window.windowID)",
                    displayID: String(display.displayID),
                    capturedAt: Date(),
                    screenBasisID: UUID().uuidString.lowercased()
                )
            }

        case .display:
            let filter = SCContentFilter(display: display, excludingApplications: excludedApps, exceptingWindows: [])
            let image = try await SCScreenshotManager.captureImage(contentFilter: filter, configuration: config)
            let appName = NSWorkspace.shared.frontmostApplication?.localizedName ?? "Unknown"
            let appBundleID = NSWorkspace.shared.frontmostApplication?.bundleIdentifier
            return CaptureSnapshot(
                image: image,
                appName: appName,
                appBundleID: appBundleID,
                windowTitle: nil,
                targetKind: .display,
                targetIdentity: "display:\(display.displayID)",
                displayID: String(display.displayID),
                capturedAt: Date(),
                screenBasisID: UUID().uuidString.lowercased()
            )
        }

        let filter = SCContentFilter(display: display, excludingApplications: excludedApps, exceptingWindows: [])
        let image = try await SCScreenshotManager.captureImage(contentFilter: filter, configuration: config)
        let appName = NSWorkspace.shared.frontmostApplication?.localizedName ?? "Unknown"
        let appBundleID = NSWorkspace.shared.frontmostApplication?.bundleIdentifier
        return CaptureSnapshot(
            image: image,
            appName: appName,
            appBundleID: appBundleID,
            windowTitle: nil,
            targetKind: .displayFallback,
            targetIdentity: "display:\(display.displayID)",
            displayID: String(display.displayID),
            capturedAt: Date(),
            screenBasisID: UUID().uuidString.lowercased()
        )
    }

    private enum CaptureMode {
        case windowUnderCursor
        case frontmostWindow
        case display
    }

    private func selectedCaptureMode() -> CaptureMode {
        switch AppSettings.shared.captureTargetMode {
        case .windowUnderCursor:
            return .windowUnderCursor
        case .frontmostWindow:
            return .frontmostWindow
        case .display:
            return .display
        }
    }

    private struct WindowTarget {
        let window: SCWindow
        let appName: String
        let appBundleID: String?
        let windowTitle: String?
    }

    private func displayContainingMouse(from content: SCShareableContent, mouseLocation: CGPoint) -> SCDisplay {
        let mouseScreen = NSScreen.screens.first { NSMouseInRect(mouseLocation, $0.frame, false) }
        let mouseDisplayID = mouseScreen.flatMap {
            $0.deviceDescription[NSDeviceDescriptionKey("NSScreenNumber")] as? CGDirectDisplayID
        }
        return mouseDisplayID.flatMap { id in
            content.displays.first { $0.displayID == id }
        } ?? content.displays.first!
    }

    private func mouseDisplayID(mouseLocation: CGPoint) -> String {
        let mouseScreen = NSScreen.screens.first { NSMouseInRect(mouseLocation, $0.frame, false) }
        let mouseDisplayID = mouseScreen.flatMap {
            $0.deviceDescription[NSDeviceDescriptionKey("NSScreenNumber")] as? CGDirectDisplayID
        }
        return mouseDisplayID.map(String.init) ?? ""
    }

    private func windowUnderCursor(from content: SCShareableContent, mouseLocation: CGPoint) -> WindowTarget? {
        if let probe = axWindowProbe(at: mouseLocation),
           probe.appBundleID != Bundle.main.bundleIdentifier,
           let window = content.windows.first(where: { matches(window: $0, probe: probe) }) {
            return WindowTarget(
                window: window,
                appName: probe.appName,
                appBundleID: probe.appBundleID,
                windowTitle: probe.windowTitle ?? normalizedTitle(window.title)
            )
        }

        guard let windowInfos = CGWindowListCopyWindowInfo([.optionOnScreenOnly, .excludeDesktopElements], kCGNullWindowID) as? [[String: Any]] else {
            return nil
        }

        let bundleID = Bundle.main.bundleIdentifier
        let windowsByID = Dictionary(uniqueKeysWithValues: content.windows.map { (Int($0.windowID), $0) })

        for info in windowInfos {
            let layer = info[kCGWindowLayer as String] as? Int ?? 0
            guard layer == 0 else { continue }

            let windowID = info[kCGWindowNumber as String] as? Int ?? 0
            guard let window = windowsByID[windowID] else { continue }

            if let boundsDict = info[kCGWindowBounds as String] as? NSDictionary {
                var bounds = CGRect.zero
                guard CGRectMakeWithDictionaryRepresentation(boundsDict, &bounds), MouseWindowTargetingGeometry.windowBoundsContainMouse(bounds, mouseLocation: mouseLocation) else {
                    continue
                }
            } else {
                let cgMouse = MouseWindowTargetingGeometry.appKitToCGPoint(mouseLocation)
                guard window.frame.contains(cgMouse) else { continue }
            }

            let ownerPID = info[kCGWindowOwnerPID as String] as? pid_t ?? 0
            let runningApp = NSRunningApplication(processIdentifier: ownerPID)
            if runningApp?.bundleIdentifier == bundleID {
                continue
            }

            let appName = (info[kCGWindowOwnerName as String] as? String)
                ?? window.owningApplication?.applicationName
                ?? runningApp?.localizedName
                ?? "Unknown"
            let title = normalizedTitle(info[kCGWindowName as String] as? String)
            let appBundleID = window.owningApplication?.bundleIdentifier ?? runningApp?.bundleIdentifier

            return WindowTarget(
                window: window,
                appName: appName,
                appBundleID: appBundleID,
                windowTitle: title
            )
        }

        return nil
    }

    private func frontmostWindow(from content: SCShareableContent) -> WindowTarget? {
        let frontApp = NSWorkspace.shared.frontmostApplication
        guard let bundleID = frontApp?.bundleIdentifier,
              bundleID != Bundle.main.bundleIdentifier else {
            return nil
        }

        guard let window = content.windows.first(where: { $0.owningApplication?.bundleIdentifier == bundleID }) else {
            return nil
        }

        return WindowTarget(
            window: window,
            appName: frontApp?.localizedName ?? window.owningApplication?.applicationName ?? "Unknown",
            appBundleID: bundleID,
            windowTitle: normalizedTitle(window.title)
        )
    }

    private func normalizedTitle(_ title: String?) -> String? {
        let trimmed = title?.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed?.isEmpty == false ? trimmed : nil
    }

    private func matches(window: SCWindow, probe: AXWindowProbe) -> Bool {
        if let probeBundle = probe.appBundleID,
           let bundle = window.owningApplication?.bundleIdentifier,
           bundle != probeBundle {
            return false
        }

        let probeTitle = normalizedTitle(probe.windowTitle)?.lowercased()
        let windowTitle = normalizedTitle(window.title)?.lowercased()
        if let probeTitle, !probeTitle.isEmpty {
            if let windowTitle, windowTitle == probeTitle || windowTitle.contains(probeTitle) || probeTitle.contains(windowTitle) {
                return true
            }
            return false
        }

        return true
    }

    private func axWindowProbe(at point: CGPoint) -> AXWindowProbe? {
        let cgPoint = MouseWindowTargetingGeometry.appKitToCGPoint(point)
        let systemWide = AXUIElementCreateSystemWide()
        var hit: AXUIElement?
        guard AXUIElementCopyElementAtPosition(systemWide, Float(cgPoint.x), Float(cgPoint.y), &hit) == .success,
              let hit else {
            return nil
        }

        let window = enclosingAXWindow(startingAt: hit) ?? hit
        var pid: pid_t = 0
        AXUIElementGetPid(window, &pid)
        if pid == 0 {
            AXUIElementGetPid(hit, &pid)
        }
        guard pid != 0 else {
            return nil
        }

        let app = NSRunningApplication(processIdentifier: pid)
        return AXWindowProbe(
            appName: app?.localizedName ?? "Unknown",
            appBundleID: app?.bundleIdentifier,
            windowTitle: axStringValue(window, attribute: kAXTitleAttribute)
        )
    }

    private func enclosingAXWindow(startingAt element: AXUIElement) -> AXUIElement? {
        var current: AXUIElement? = element
        for _ in 0..<8 {
            guard let node = current else { return nil }
            if axStringValue(node, attribute: kAXRoleAttribute) == kAXWindowRole as String {
                return node
            }
            current = axParent(of: node)
        }
        return nil
    }

    private func axParent(of element: AXUIElement) -> AXUIElement? {
        var value: CFTypeRef?
        guard AXUIElementCopyAttributeValue(element, kAXParentAttribute as CFString, &value) == .success,
              let value else {
            return nil
        }
        return (value as! AXUIElement)
    }

    private func axStringValue(_ element: AXUIElement, attribute: String) -> String? {
        var value: CFTypeRef?
        guard AXUIElementCopyAttributeValue(element, attribute as CFString, &value) == .success,
              let string = value as? String else {
            return nil
        }
        return string
    }

    enum CaptureError: Error, LocalizedError {
        case noDisplay

        var errorDescription: String? {
            VibeCatL10n.captureErrorNoDisplay()
        }
    }
}
