import AppKit
import VibeCatCore

/// Animates the menu bar tray icon through 8 frames from Assets/TrayIcons_Clean/
@MainActor
final class TrayIconAnimator {
    private enum IndicatorLayout {
        static let minimumDotDiameter: CGFloat = 4.5
        static let dotSizeRatio: CGFloat = 0.32
        static let dotPadding: CGFloat = 1.5
        static let dotOutlineInset: CGFloat = 1.0
        static let dotOutlineOpacity: CGFloat = 0.32
    }

    enum CaptureIndicatorState: Hashable {
        case active
        case manual
        case paused

        var color: NSColor {
            switch self {
            case .active:
                return .systemGreen
            case .manual:
                return .systemOrange
            case .paused:
                return .systemGray
            }
        }
    }

    private var frames: [NSImage] = []
    private var currentFrame = 0
    private var timer: Timer?
    private weak var statusItem: NSStatusItem?
    private var captureIndicatorState: CaptureIndicatorState = .active
    private var compositedFrames: [CaptureIndicatorState: [NSImage]] = [:]

    private let frameInterval: TimeInterval = 0.1
    private let frameCount = 8

    init() {
        loadFrames()
    }

    /// Attach this animator to a status item and start animating
    func attach(to item: NSStatusItem) {
        self.statusItem = item
        start()
    }

    func setCaptureState(_ state: CaptureIndicatorState) {
        guard captureIndicatorState != state else { return }
        captureIndicatorState = state
        updateIcon()
    }

    /// Update animation state based on companion emotion (MVP: all emotions use same idle frames)
    func setEmotion(_ emotion: CompanionEmotion) {
        // MVP: all emotions use the same tray animation
        // Future: load emotion-specific frame sets
    }

    private func loadFrames() {
        // Try to find Assets relative to the repo root
        // During development with `swift run`, the working directory is the package root
        let repoRoot = findRepoRoot()
        for i in 1...frameCount {
            let filename = String(format: "tray_%02d.png", i)
            let path = repoRoot.appendingPathComponent("Assets/TrayIcons_Clean/\(filename)")
            if let image = NSImage(contentsOf: path) {
                image.size = NSSize(width: 18, height: 18)
                frames.append(image)
            }
        }
        // Fallback: use a system symbol if no frames loaded
        if frames.isEmpty {
            if let fallback = NSImage(systemSymbolName: "cat.fill", accessibilityDescription: "VibeCat") {
                frames = [fallback]
            } else if let fallback = NSImage(systemSymbolName: "pawprint.fill", accessibilityDescription: "VibeCat") {
                frames = [fallback]
            }
        }
        compositedFrames.removeAll()
    }

    private func findRepoRoot() -> URL {
        // Walk up from Bundle.main or current directory to find the repo root
        // (contains Assets/TrayIcons_Clean/)
        var url = URL(fileURLWithPath: FileManager.default.currentDirectoryPath)
        for _ in 0..<6 {
            let candidate = url.appendingPathComponent("Assets/TrayIcons_Clean")
            if FileManager.default.fileExists(atPath: candidate.path) {
                return url
            }
            url = url.deletingLastPathComponent()
        }
        // Last resort: use Bundle.main.bundleURL parent chain
        var bundleURL = Bundle.main.bundleURL
        for _ in 0..<6 {
            let candidate = bundleURL.appendingPathComponent("Assets/TrayIcons_Clean")
            if FileManager.default.fileExists(atPath: candidate.path) {
                return bundleURL
            }
            bundleURL = bundleURL.deletingLastPathComponent()
        }
        return URL(fileURLWithPath: FileManager.default.currentDirectoryPath)
    }

    private func start() {
        guard !frames.isEmpty else { return }
        // Show first frame immediately
        updateIcon()
        timer = Timer.scheduledTimer(withTimeInterval: frameInterval, repeats: true) { [weak self] _ in
            Task { @MainActor [weak self] in
                self?.advanceFrame()
            }
        }
    }

    func stop() {
        timer?.invalidate()
        timer = nil
    }

    private func advanceFrame() {
        currentFrame = (currentFrame + 1) % frames.count
        updateIcon()
    }

    private func updateIcon() {
        guard currentFrame < frames.count else { return }
        let images = compositedFrames[captureIndicatorState] ?? buildCompositedFrames(for: captureIndicatorState)
        guard currentFrame < images.count else { return }
        statusItem?.button?.image = images[currentFrame]
    }

    private func buildCompositedFrames(for state: CaptureIndicatorState) -> [NSImage] {
        let images = frames.map { compositedImage(for: $0, state: state) }
        compositedFrames[state] = images
        return images
    }

    private func compositedImage(for base: NSImage, state: CaptureIndicatorState) -> NSImage {
        let output = NSImage(size: base.size)
        output.lockFocus()
        defer { output.unlockFocus() }

        base.draw(in: NSRect(origin: .zero, size: base.size))

        let dotDiameter = max(
            IndicatorLayout.minimumDotDiameter,
            min(base.size.width, base.size.height) * IndicatorLayout.dotSizeRatio
        )
        let dotRect = NSRect(
            x: base.size.width - dotDiameter - IndicatorLayout.dotPadding,
            y: base.size.height - dotDiameter - IndicatorLayout.dotPadding,
            width: dotDiameter,
            height: dotDiameter
        )

        NSColor.black.withAlphaComponent(IndicatorLayout.dotOutlineOpacity).setFill()
        NSBezierPath(
            ovalIn: dotRect.insetBy(dx: -IndicatorLayout.dotOutlineInset, dy: -IndicatorLayout.dotOutlineInset)
        ).fill()

        state.color.setFill()
        NSBezierPath(ovalIn: dotRect).fill()

        return output
    }
}
