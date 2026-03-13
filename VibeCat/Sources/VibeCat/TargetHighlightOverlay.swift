import AppKit

enum TargetHighlightGeometry {
    static func overlayFrame(for targetRect: CGRect, padding: CGFloat = 6) -> CGRect {
        targetRect.insetBy(dx: -padding, dy: -padding).integral
    }
}

@MainActor
final class TargetHighlightOverlay: NSPanel {
    private let highlightView = HighlightView()

    init() {
        super.init(
            contentRect: .zero,
            styleMask: [.borderless, .nonactivatingPanel],
            backing: .buffered,
            defer: false
        )

        level = .statusBar
        backgroundColor = .clear
        isOpaque = false
        hasShadow = false
        collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary, .stationary]
        ignoresMouseEvents = true
        isReleasedWhenClosed = false

        contentView = highlightView
        orderOut(nil)
    }

    func show(targetRect: CGRect) {
        let padded = TargetHighlightGeometry.overlayFrame(for: targetRect)
        let primaryHeight = NSScreen.screens.first(where: { $0.frame.origin == .zero })?.frame.height
            ?? NSScreen.screens.first?.frame.height ?? 0
        let appKitFrame = NSRect(
            x: padded.origin.x,
            y: primaryHeight - padded.origin.y - padded.height,
            width: padded.width,
            height: padded.height
        )
        setFrame(appKitFrame, display: true)
        orderFront(nil)
    }

    func hide() {
        orderOut(nil)
    }
}

private final class HighlightView: NSView {
    override var isOpaque: Bool { false }

    override func draw(_ dirtyRect: NSRect) {
        super.draw(dirtyRect)
        NSColor.systemYellow.withAlphaComponent(0.95).setStroke()
        let path = NSBezierPath(roundedRect: bounds.insetBy(dx: 2, dy: 2), xRadius: 8, yRadius: 8)
        path.lineWidth = 3
        path.stroke()
    }
}
