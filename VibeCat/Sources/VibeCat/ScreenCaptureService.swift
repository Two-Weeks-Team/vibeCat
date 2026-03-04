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

    // MARK: - Public API

    /// Capture the screen. Returns .unchanged if screen hasn't changed significantly.
    func captureAroundCursor() async -> CaptureResult {
        do {
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

        guard let display = content.displays.first else {
            throw CaptureError.noDisplay
        }

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
        config.width = fullWindow ? 1920 : 1280
        config.height = fullWindow ? 1080 : 720
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
