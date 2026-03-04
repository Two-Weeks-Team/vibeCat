import AppKit
import VibeCatCore

@MainActor
final class ChatBubbleView: NSView {
    enum TailDirection {
        case left
        case right
        case up
        case down
    }

    private let label = NSTextField(wrappingLabelWithString: "")
    private var hideTimer: Timer?
    private var tailDirection: TailDirection = .left

    private let horizontalPadding: CGFloat = 10
    private let verticalPadding: CGFloat = 8
    private let tailSize: CGFloat = 10

    override init(frame: NSRect) {
        super.init(frame: frame)
        setup()
    }

    required init?(coder: NSCoder) {
        super.init(coder: coder)
        setup()
    }

    private func setup() {
        wantsLayer = true
        layer?.backgroundColor = .clear

        label.textColor = .white
        label.font = NSFont.systemFont(ofSize: 13)
        label.backgroundColor = .clear
        label.isBezeled = false
        label.isEditable = false
        label.translatesAutoresizingMaskIntoConstraints = false
        addSubview(label)

        isHidden = true
        alphaValue = 0
    }

    override func layout() {
        super.layout()
        label.frame = labelFrame(for: bounds)
    }

    override func draw(_ dirtyRect: NSRect) {
        super.draw(dirtyRect)

        NSColor.black.withAlphaComponent(0.78).setFill()

        let radius: CGFloat = 12
        let bodyRect = bubbleBodyRect(in: bounds)
        let bubblePath = NSBezierPath(roundedRect: bodyRect, xRadius: radius, yRadius: radius)

        let tailPath = NSBezierPath()
        switch tailDirection {
        case .left:
            let midY = bodyRect.midY
            tailPath.move(to: NSPoint(x: bodyRect.minX, y: midY + 7))
            tailPath.line(to: NSPoint(x: bodyRect.minX, y: midY - 7))
            tailPath.line(to: NSPoint(x: bounds.minX, y: midY))
        case .right:
            let midY = bodyRect.midY
            tailPath.move(to: NSPoint(x: bodyRect.maxX, y: midY + 7))
            tailPath.line(to: NSPoint(x: bodyRect.maxX, y: midY - 7))
            tailPath.line(to: NSPoint(x: bounds.maxX, y: midY))
        case .up:
            let midX = bodyRect.midX
            tailPath.move(to: NSPoint(x: midX - 7, y: bodyRect.maxY))
            tailPath.line(to: NSPoint(x: midX + 7, y: bodyRect.maxY))
            tailPath.line(to: NSPoint(x: midX, y: bounds.maxY))
        case .down:
            let midX = bodyRect.midX
            tailPath.move(to: NSPoint(x: midX - 7, y: bodyRect.minY))
            tailPath.line(to: NSPoint(x: midX + 7, y: bodyRect.minY))
            tailPath.line(to: NSPoint(x: midX, y: bounds.minY))
        }
        tailPath.close()

        bubblePath.fill()
        tailPath.fill()
    }

    func preferredSize(for text: String) -> NSSize {
        let maxTextWidth: CGFloat = 250 - horizontalPadding * 2 - tailSize
        let minTextWidth: CGFloat = 80 - horizontalPadding * 2
        let constrained = NSSize(width: maxTextWidth, height: .greatestFiniteMagnitude)
        let attributes: [NSAttributedString.Key: Any] = [.font: label.font as Any]
        let rect = (text as NSString).boundingRect(
            with: constrained,
            options: [.usesLineFragmentOrigin, .usesFontLeading],
            attributes: attributes
        )

        let measuredTextWidth = ceil(rect.width)
        let measuredTextHeight = ceil(rect.height)
        let textWidth = max(minTextWidth, measuredTextWidth)
        let width = min(250, textWidth + horizontalPadding * 2 + tailExtraWidth())
        let height = measuredTextHeight + verticalPadding * 2 + tailExtraHeight()
        return NSSize(width: max(80, width), height: max(34, height))
    }

    func setTailDirection(_ direction: TailDirection) {
        tailDirection = direction
        needsDisplay = true
    }

    func show(text: String, autohideAfter seconds: TimeInterval = 6.0) {
        label.stringValue = text
        frame.size = preferredSize(for: text)
        needsLayout = true
        layoutSubtreeIfNeeded()
        isHidden = false
        alphaValue = 0

        NSAnimationContext.runAnimationGroup { ctx in
            ctx.duration = 0.18
            self.animator().alphaValue = 1.0
        }

        hideTimer?.invalidate()
        hideTimer = Timer.scheduledTimer(withTimeInterval: seconds, repeats: false) { [weak self] _ in
            Task { @MainActor [weak self] in
                self?.hide()
            }
        }
    }

    func hide() {
        NSAnimationContext.runAnimationGroup { ctx in
            ctx.duration = 0.3
            self.animator().alphaValue = 0
        } completionHandler: { [weak self] in
            Task { @MainActor [weak self] in
                self?.isHidden = true
                self?.alphaValue = 0
            }
        }
    }

    private func tailExtraWidth() -> CGFloat {
        switch tailDirection {
        case .left, .right:
            return tailSize
        case .up, .down:
            return 0
        }
    }

    private func tailExtraHeight() -> CGFloat {
        switch tailDirection {
        case .up, .down:
            return tailSize
        case .left, .right:
            return 0
        }
    }

    private func bubbleBodyRect(in bounds: NSRect) -> NSRect {
        var rect = bounds
        switch tailDirection {
        case .left:
            rect.origin.x += tailSize
            rect.size.width -= tailSize
        case .right:
            rect.size.width -= tailSize
        case .up:
            rect.size.height -= tailSize
        case .down:
            rect.origin.y += tailSize
            rect.size.height -= tailSize
        }
        return rect
    }

    private func labelFrame(for bounds: NSRect) -> NSRect {
        let bodyRect = bubbleBodyRect(in: bounds)
        return bodyRect.insetBy(dx: horizontalPadding, dy: verticalPadding)
    }
}
