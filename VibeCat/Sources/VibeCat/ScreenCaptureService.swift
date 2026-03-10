import Foundation
import ScreenCaptureKit
import CoreGraphics
import AppKit
import VibeCatCore

/// Captures a region of the screen using ScreenCaptureKit.
/// Excludes the VibeCat panel itself from captures.
@MainActor
final class ScreenCaptureService {
    struct CaptureSnapshot {
        enum TargetKind: String {
            case windowUnderCursor = "window_under_cursor"
            case frontmostWindow = "frontmost_window"
            case displayFallback = "display_fallback"
        }

        let image: CGImage
        let appName: String
        let appBundleID: String?
        let windowTitle: String?
        let targetKind: TargetKind
        let targetIdentity: String

        var contextDescription: String {
            var parts = [
                "app=\(appName)",
                "target=\(targetKind.rawValue)",
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

    enum CaptureResult {
        case captured(CaptureSnapshot)
        case unchanged
        case unavailable(String)
    }

    private var lastImage: CGImage?
    private var lastTargetIdentity: String?

    // MARK: - Public API

    /// Capture the screen. Returns .unchanged if screen hasn't changed significantly.
    func captureAroundCursor() async -> CaptureResult {
        do {
            let snapshot = try await performCapture(mode: .windowUnderCursor)
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
            let snapshot = try await performCapture(mode: .windowUnderCursor)
            lastTargetIdentity = snapshot.targetIdentity
            lastImage = snapshot.image
            return .captured(snapshot)
        } catch {
            return .unavailable(error.localizedDescription)
        }
    }

    // MARK: - Private

    private func performCapture(mode: CaptureMode) async throws -> CaptureSnapshot {
        let content = try await SCShareableContent.excludingDesktopWindows(false, onScreenWindowsOnly: true)

        guard !content.displays.isEmpty else {
            throw CaptureError.noDisplay
        }

        let mouseLocation = NSEvent.mouseLocation
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
                    targetIdentity: "window:\(target.window.windowID)"
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
                    targetIdentity: "window:\(target.window.windowID)"
                )
            }
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
            targetIdentity: "display:\(display.displayID)"
        )
    }

    private enum CaptureMode {
        case windowUnderCursor
        case frontmostWindow
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

    private func windowUnderCursor(from content: SCShareableContent, mouseLocation: CGPoint) -> WindowTarget? {
        guard let windowInfos = CGWindowListCopyWindowInfo([.optionOnScreenOnly, .excludeDesktopElements], kCGNullWindowID) as? [[String: Any]] else {
            return nil
        }

        let bundleID = Bundle.main.bundleIdentifier
        let windowsByID = Dictionary(uniqueKeysWithValues: content.windows.map { (Int($0.windowID), $0) })

        for info in windowInfos {
            let layer = info[kCGWindowLayer as String] as? Int ?? 0
            guard layer == 0 else { continue }

            let windowID = info[kCGWindowNumber as String] as? Int ?? 0
            guard let window = windowsByID[windowID], window.frame.contains(mouseLocation) else { continue }

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

    enum CaptureError: Error, LocalizedError {
        case noDisplay

        var errorDescription: String? {
            "No display available for capture"
        }
    }
}
