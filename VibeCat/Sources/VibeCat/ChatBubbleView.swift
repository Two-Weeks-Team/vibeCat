import AppKit
import VibeCatCore

@MainActor
final class ChatBubbleView: NSView {
    enum TailDirection {
        case bottom
        case top
    }

    static let bgColor = NSColor(red: 0.1, green: 0.04, blue: 0.15, alpha: 1.0)
    static let borderColor = NSColor(red: 0.61, green: 0.49, blue: 0.78, alpha: 0.8)
    static let glowColor = NSColor(red: 0.83, green: 0.33, blue: 0.48, alpha: 0.3)

    private let label = NSTextField(wrappingLabelWithString: "")
    private var tailDirection: TailDirection = .bottom

    private let horizontalPadding: CGFloat = 14
    private let verticalPadding: CGFloat = 10
    private let tailSize = CGSize(width: 16, height: 12)
    private let cornerRadius: CGFloat = 16

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
        layer?.backgroundColor = NSColor.clear.cgColor
        layer?.shadowColor = Self.glowColor.cgColor
        layer?.shadowRadius = 8
        layer?.shadowOffset = CGSize(width: 0, height: -3)
        layer?.shadowOpacity = 1.0

        label.textColor = .white
        label.font = .systemFont(ofSize: 13, weight: .medium)
        label.backgroundColor = .clear
        label.isBezeled = false
        label.isEditable = false
        label.maximumNumberOfLines = 0
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

        let path = NSBezierPath()
        let kappa: CGFloat = 0.55228
        let offset = cornerRadius * kappa

        if tailDirection == .bottom {
            let bodyRect = NSRect(x: bounds.minX, y: bounds.minY + tailSize.height, width: bounds.width, height: bounds.height - tailSize.height)
            let minX = bodyRect.minX
            let maxX = bodyRect.maxX
            let minY = bodyRect.minY
            let maxY = bodyRect.maxY
            let midX = bodyRect.midX

            let tailLeft = midX - tailSize.width / 2
            let tailRight = midX + tailSize.width / 2

            path.move(to: NSPoint(x: minX + cornerRadius, y: maxY))
            path.line(to: NSPoint(x: maxX - cornerRadius, y: maxY))
            path.curve(to: NSPoint(x: maxX, y: maxY - cornerRadius),
                       controlPoint1: NSPoint(x: maxX - cornerRadius + offset, y: maxY),
                       controlPoint2: NSPoint(x: maxX, y: maxY - cornerRadius + offset))
            
            path.line(to: NSPoint(x: maxX, y: minY + cornerRadius))
            path.curve(to: NSPoint(x: maxX - cornerRadius, y: minY),
                       controlPoint1: NSPoint(x: maxX, y: minY + cornerRadius - offset),
                       controlPoint2: NSPoint(x: maxX - cornerRadius + offset, y: minY))
            
            path.line(to: NSPoint(x: tailRight, y: minY))
            
            let cp1 = NSPoint(x: tailRight - 2, y: minY - tailSize.height * 0.4)
            path.curve(to: NSPoint(x: midX, y: bounds.minY), controlPoint1: cp1, controlPoint2: cp1)
            let cp2 = NSPoint(x: tailLeft + 2, y: minY - tailSize.height * 0.4)
            path.curve(to: NSPoint(x: tailLeft, y: minY), controlPoint1: cp2, controlPoint2: cp2)
            
            path.line(to: NSPoint(x: minX + cornerRadius, y: minY))
            path.curve(to: NSPoint(x: minX, y: minY + cornerRadius),
                       controlPoint1: NSPoint(x: minX + cornerRadius - offset, y: minY),
                       controlPoint2: NSPoint(x: minX, y: minY + cornerRadius - offset))
            
            path.line(to: NSPoint(x: minX, y: maxY - cornerRadius))
            path.curve(to: NSPoint(x: minX + cornerRadius, y: maxY),
                       controlPoint1: NSPoint(x: minX, y: maxY - cornerRadius + offset),
                       controlPoint2: NSPoint(x: minX + cornerRadius - offset, y: maxY))
        } else {
            let bodyRect = NSRect(x: bounds.minX, y: bounds.minY, width: bounds.width, height: bounds.height - tailSize.height)
            let minX = bodyRect.minX
            let maxX = bodyRect.maxX
            let minY = bodyRect.minY
            let maxY = bodyRect.maxY
            let midX = bodyRect.midX

            let tailLeft = midX - tailSize.width / 2
            let tailRight = midX + tailSize.width / 2

            path.move(to: NSPoint(x: minX + cornerRadius, y: maxY))
            
            path.line(to: NSPoint(x: tailLeft, y: maxY))
            
            let cp1 = NSPoint(x: tailLeft + 2, y: maxY + tailSize.height * 0.4)
            path.curve(to: NSPoint(x: midX, y: bounds.maxY), controlPoint1: cp1, controlPoint2: cp1)
            let cp2 = NSPoint(x: tailRight - 2, y: maxY + tailSize.height * 0.4)
            path.curve(to: NSPoint(x: tailRight, y: maxY), controlPoint1: cp2, controlPoint2: cp2)
            
            path.line(to: NSPoint(x: maxX - cornerRadius, y: maxY))
            path.curve(to: NSPoint(x: maxX, y: maxY - cornerRadius),
                       controlPoint1: NSPoint(x: maxX - cornerRadius + offset, y: maxY),
                       controlPoint2: NSPoint(x: maxX, y: maxY - cornerRadius + offset))
            
            path.line(to: NSPoint(x: maxX, y: minY + cornerRadius))
            path.curve(to: NSPoint(x: maxX - cornerRadius, y: minY),
                       controlPoint1: NSPoint(x: maxX, y: minY + cornerRadius - offset),
                       controlPoint2: NSPoint(x: maxX - cornerRadius + offset, y: minY))
            
            path.line(to: NSPoint(x: minX + cornerRadius, y: minY))
            path.curve(to: NSPoint(x: minX, y: minY + cornerRadius),
                       controlPoint1: NSPoint(x: minX + cornerRadius - offset, y: minY),
                       controlPoint2: NSPoint(x: minX, y: minY + cornerRadius - offset))
            
            path.line(to: NSPoint(x: minX, y: maxY - cornerRadius))
            path.curve(to: NSPoint(x: minX + cornerRadius, y: maxY),
                       controlPoint1: NSPoint(x: minX, y: maxY - cornerRadius + offset),
                       controlPoint2: NSPoint(x: minX + cornerRadius - offset, y: maxY))
        }
        path.close()

        Self.bgColor.setFill()
        path.fill()

        Self.borderColor.setStroke()
        path.lineWidth = 2
        path.stroke()
    }

    private let maxBubbleWidth: CGFloat = 320
    private let maxBubbleHeight: CGFloat = 220

    func preferredSize(for text: String) -> NSSize {
        let maxTextWidth: CGFloat = maxBubbleWidth - horizontalPadding * 2
        let minTextWidth: CGFloat = 80 - horizontalPadding * 2

        label.stringValue = text
        label.maximumNumberOfLines = 0
        label.preferredMaxLayoutWidth = maxTextWidth

        let textSize = label.intrinsicContentSize
        let textWidth = max(minTextWidth, ceil(textSize.width))
        let width = min(maxBubbleWidth, textWidth + horizontalPadding * 2)
        let rawHeight = ceil(textSize.height) + verticalPadding * 2 + tailSize.height
        let height = min(maxBubbleHeight, rawHeight)
        return NSSize(width: max(80, width), height: max(34, height))
    }

    func setTailDirection(_ direction: TailDirection) {
        guard tailDirection != direction else { return }
        tailDirection = direction
        needsLayout = true
        needsDisplay = true
    }

    func show(text: String) {
        frame.size = preferredSize(for: text)
        needsLayout = true
        layoutSubtreeIfNeeded()
        isHidden = false
        needsDisplay = true
        
        alphaValue = 0
        layer?.transform = CATransform3DMakeScale(0.3, 0.3, 1.0)

        if let layer = layer {
            let oldAnchor = layer.anchorPoint
            let newAnchor = CGPoint(x: 0.5, y: tailDirection == .bottom ? 0.0 : 1.0)
            let bounds = layer.bounds
            layer.position = CGPoint(
                x: layer.position.x + bounds.width * (newAnchor.x - oldAnchor.x),
                y: layer.position.y + bounds.height * (newAnchor.y - oldAnchor.y)
            )
            layer.anchorPoint = newAnchor
        }
        
        // Spring scale animation
        let spring = CASpringAnimation(keyPath: "transform.scale")
        spring.fromValue = 0.3
        spring.toValue = 1.0
        spring.mass = 1.0
        spring.stiffness = 322
        spring.damping = 21.5
        spring.duration = spring.settlingDuration
        spring.fillMode = .forwards
        spring.isRemovedOnCompletion = false
        layer?.add(spring, forKey: "springScale")
        layer?.transform = CATransform3DIdentity
        
        // Fade in
        NSAnimationContext.runAnimationGroup { ctx in
            ctx.duration = 0.2
            self.animator().alphaValue = 1.0
        }
    }

    func updateText(_ text: String) {
        label.stringValue = text
        needsDisplay = true
    }

    func hide() {
        NSAnimationContext.runAnimationGroup { ctx in
            ctx.duration = 0.3
            self.animator().alphaValue = 0
        } completionHandler: { [weak self] in
            Task { @MainActor [weak self] in
                self?.isHidden = true
                self?.alphaValue = 0
                self?.layer?.removeAllAnimations()
            }
        }
    }

    private func bubbleBodyRect(in bounds: NSRect) -> NSRect {
        var rect = bounds
        if tailDirection == .bottom {
            rect.origin.y += tailSize.height
            rect.size.height -= tailSize.height
        } else {
            rect.size.height -= tailSize.height
        }
        return rect
    }

    private func labelFrame(for bounds: NSRect) -> NSRect {
        let bodyRect = bubbleBodyRect(in: bounds)
        let availableWidth = bodyRect.width - horizontalPadding * 2
        return NSRect(
            x: bodyRect.minX + horizontalPadding,
            y: bodyRect.minY + verticalPadding,
            width: availableWidth,
            height: bodyRect.height - verticalPadding * 2
        )
    }
}
