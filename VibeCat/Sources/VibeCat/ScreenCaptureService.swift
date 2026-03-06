import Foundation
import ScreenCaptureKit
import CoreGraphics
import VibeCatCore

/// Captures a region of the screen using ScreenCaptureKit.
/// Excludes the VibeCat panel itself from captures.
@MainActor
final class ScreenCaptureService {
    enum CaptureResult {
        case captured(CGImage)
        case unchanged
        case unavailable(String)
    }

    private var lastImage: CGImage?
    private var lastFrontAppBundleID: String?

    // MARK: - Public API

    /// Capture the screen. Returns .unchanged if screen hasn't changed significantly.
    func captureAroundCursor() async -> CaptureResult {
        do {
            let currentApp = NSWorkspace.shared.frontmostApplication?.bundleIdentifier
            if currentApp != lastFrontAppBundleID {
                lastImage = nil
                lastFrontAppBundleID = currentApp
            }

            let image = try await performCapture(fullWindow: false)
            if !ImageDiffer.hasSignificantChange(from: lastImage, to: image) {
                return .unchanged
            }
            lastImage = image
            return .captured(image)
        } catch {
            return .unavailable(error.localizedDescription)
        }
    }

    /// Capture the full frontmost window (for high-significance analysis).
    func captureFullWindow() async -> CaptureResult {
        do {
            let image = try await performCapture(fullWindow: true)
            return .captured(image)
        } catch {
            return .unavailable(error.localizedDescription)
        }
    }

    /// Force a capture regardless of change detection.
    func forceCapture() async -> CaptureResult {
        do {
            let image = try await performCapture(fullWindow: false)
            lastImage = image
            return .captured(image)
        } catch {
            return .unavailable(error.localizedDescription)
        }
    }

    // MARK: - Private

    private func performCapture(fullWindow: Bool) async throws -> CGImage {
        let content = try await SCShareableContent.excludingDesktopWindows(false, onScreenWindowsOnly: true)

        guard !content.displays.isEmpty else {
            throw CaptureError.noDisplay
        }

        // Find the display containing the mouse pointer (multi-monitor support)
        let mouseLocation = NSEvent.mouseLocation
        let mouseScreen = NSScreen.screens.first { NSMouseInRect(mouseLocation, $0.frame, false) }
        let mouseDisplayID = mouseScreen.flatMap {
            $0.deviceDescription[NSDeviceDescriptionKey("NSScreenNumber")] as? CGDirectDisplayID
        }
        let display = mouseDisplayID.flatMap { id in
            content.displays.first { $0.displayID == id }
        } ?? content.displays.first!

        // Exclude VibeCat's own windows
        let excludedApps = content.applications.filter { app in
            app.bundleIdentifier == Bundle.main.bundleIdentifier
        }

        let filter: SCContentFilter
        if fullWindow, let frontWindow = frontmostWindow(from: content) {
            filter = SCContentFilter(desktopIndependentWindow: frontWindow)
        } else {
            filter = SCContentFilter(display: display, excludingApplications: excludedApps, exceptingWindows: [])
        }

        let config = SCStreamConfiguration()
        config.width = display.width * 2
        config.height = display.height * 2
        config.pixelFormat = kCVPixelFormatType_32BGRA
        config.showsCursor = false
        config.capturesAudio = false

        return try await SCScreenshotManager.captureImage(contentFilter: filter, configuration: config)
    }

    private func frontmostWindow(from content: SCShareableContent) -> SCWindow? {
        let frontApp = NSWorkspace.shared.frontmostApplication
        guard let bundleID = frontApp?.bundleIdentifier,
              bundleID != Bundle.main.bundleIdentifier else {
            return content.windows.first
        }
        return content.windows.first { $0.owningApplication?.bundleIdentifier == bundleID }
    }

    enum CaptureError: Error, LocalizedError {
        case noDisplay

        var errorDescription: String? {
            "No display available for capture"
        }
    }
}
