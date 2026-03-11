import AppKit
import VibeCatCore

@MainActor
final class ChatBubbleView: NSView {
    enum TailDirection {
        case bottom
        case top
    }

    enum DisplayMode {
        case speech
        case status
    }

    static let bgColor = NSColor(red: 0.1, green: 0.04, blue: 0.15, alpha: 1.0)
    static let borderColor = NSColor(red: 0.61, green: 0.49, blue: 0.78, alpha: 0.8)
    static let glowColor = NSColor(red: 0.83, green: 0.33, blue: 0.48, alpha: 0.3)

    private let primaryLabel = NSTextField(wrappingLabelWithString: "")
    private let metaLabel = NSTextField(wrappingLabelWithString: "")
    private let statusSpinner = NSProgressIndicator()

    private var tailDirection: TailDirection = .bottom
    private var tailXRatio: CGFloat = 0.5
    private var displayMode: DisplayMode = .speech
    private var primaryText = ""
    private var metaText: String?

    private let horizontalPadding: CGFloat = 14
    private let verticalPadding: CGFloat = 10
    private let textHeightCompensation: CGFloat = 6
    private let tailSize = CGSize(width: 16, height: 12)
    private let cornerRadius: CGFloat = 16
    private let metaSpacing: CGFloat = 4
    private let spinnerTextGap: CGFloat = 8
    private let spinnerSize = CGSize(width: 14, height: 14)
    private let maxBubbleWidth: CGFloat = 380
    private let maxBubbleHeight: CGFloat = 500
    private let minBubbleWidth: CGFloat = 80
    private let textWidthSlack: CGFloat = 10

    var isShowingStatus: Bool { displayMode == .status }

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

        primaryLabel.textColor = .white
        primaryLabel.font = .systemFont(ofSize: 13, weight: .medium)
        primaryLabel.backgroundColor = .clear
        primaryLabel.isBezeled = false
        primaryLabel.isEditable = false
        primaryLabel.maximumNumberOfLines = 0
        primaryLabel.lineBreakMode = .byWordWrapping
        primaryLabel.usesSingleLineMode = false
        addSubview(primaryLabel)

        metaLabel.textColor = NSColor.white.withAlphaComponent(0.72)
        metaLabel.font = .systemFont(ofSize: 11, weight: .medium)
        metaLabel.backgroundColor = .clear
        metaLabel.isBezeled = false
        metaLabel.isEditable = false
        metaLabel.maximumNumberOfLines = 2
        metaLabel.lineBreakMode = .byTruncatingTail
        metaLabel.usesSingleLineMode = false
        metaLabel.isHidden = true
        addSubview(metaLabel)

        statusSpinner.style = .spinning
        statusSpinner.controlSize = .small
        statusSpinner.isDisplayedWhenStopped = false
        statusSpinner.isHidden = true
        addSubview(statusSpinner)

        isHidden = true
        alphaValue = 0
    }

    override func layout() {
        super.layout()
        let layout = contentLayout(for: bounds, primary: primaryText, meta: metaText, showsSpinner: displayMode == .status)
        primaryLabel.frame = layout.primaryFrame.integral
        metaLabel.frame = layout.metaFrame.integral
        metaLabel.isHidden = layout.metaFrame.isEmpty

        if let spinnerFrame = layout.spinnerFrame {
            statusSpinner.frame = spinnerFrame.integral
            statusSpinner.isHidden = false
            statusSpinner.startAnimation(nil)
        } else {
            statusSpinner.stopAnimation(nil)
            statusSpinner.isHidden = true
        }
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
            let tailCenterX = max(bodyRect.minX + cornerRadius + tailSize.width / 2,
                                   min(bodyRect.maxX - cornerRadius - tailSize.width / 2,
                                       bodyRect.minX + bodyRect.width * tailXRatio))

            let tailLeft = tailCenterX - tailSize.width / 2
            let tailRight = tailCenterX + tailSize.width / 2

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
            path.curve(to: NSPoint(x: tailCenterX, y: bounds.minY), controlPoint1: cp1, controlPoint2: cp1)
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
            let tailCenterX = max(bodyRect.minX + cornerRadius + tailSize.width / 2,
                                   min(bodyRect.maxX - cornerRadius - tailSize.width / 2,
                                       bodyRect.minX + bodyRect.width * tailXRatio))

            let tailLeft = tailCenterX - tailSize.width / 2
            let tailRight = tailCenterX + tailSize.width / 2

            path.move(to: NSPoint(x: minX + cornerRadius, y: maxY))

            path.line(to: NSPoint(x: tailLeft, y: maxY))

            let cp1 = NSPoint(x: tailLeft + 2, y: maxY + tailSize.height * 0.4)
            path.curve(to: NSPoint(x: tailCenterX, y: bounds.maxY), controlPoint1: cp1, controlPoint2: cp1)
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

    func preferredSize(for text: String) -> NSSize {
        preferredSize(primary: text, meta: nil, showsSpinner: false)
    }

    func preferredSize(primary: String, meta: String?, showsSpinner: Bool) -> NSSize {
        let primaryFont = primaryLabel.font ?? .systemFont(ofSize: 13, weight: .medium)
        let metaFont = metaLabel.font ?? .systemFont(ofSize: 11, weight: .medium)

        let primarySingleWidth = measuredTextWidth(primary, font: primaryFont)
        let metaSingleWidth = measuredTextWidth(meta ?? "", font: metaFont)
        let estimatedContentWidth = max(
            primarySingleWidth + (showsSpinner ? spinnerSize.width + spinnerTextGap : 0),
            metaSingleWidth
        )

        let width = min(maxBubbleWidth, max(minBubbleWidth, estimatedContentWidth + horizontalPadding * 2))
        let primaryTextWidth = max(40, width - horizontalPadding * 2 - (showsSpinner ? spinnerSize.width + spinnerTextGap : 0))
        let primaryTextHeight = measuredTextHeight(primary, width: primaryTextWidth, font: primaryFont)
        let metaTextHeight = measuredTextHeight(meta ?? "", width: width - horizontalPadding * 2, font: metaFont)
        let topRowHeight = max(showsSpinner ? spinnerSize.height : 0, primaryTextHeight)
        let metaHeight = (meta?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty == false) ? metaTextHeight : 0
        let contentHeight = topRowHeight + (metaHeight > 0 ? metaSpacing + metaHeight : 0)
        let rawHeight = ceil(contentHeight + textHeightCompensation) + verticalPadding * 2 + tailSize.height
        return NSSize(width: width, height: min(maxBubbleHeight, max(34, rawHeight)))
    }

    func setTailDirection(_ direction: TailDirection) {
        guard tailDirection != direction else { return }
        tailDirection = direction
        needsLayout = true
        needsDisplay = true
    }

    func setTailPosition(_ ratio: CGFloat) {
        let clamped = max(0.15, min(0.85, ratio))
        guard abs(tailXRatio - clamped) > 0.01 else { return }
        tailXRatio = clamped
        needsDisplay = true
    }

    func show(text: String) {
        showSpeech(text: text, meta: nil)
    }

    func showSpeech(text: String, meta: String?) {
        displayMode = .speech
        applyContent(primary: text, meta: meta)
        frame.size = preferredSize(primary: text, meta: meta, showsSpinner: false)
        revealIfNeeded()
    }

    func showStatus(text: String, detail: String?) {
        displayMode = .status
        applyContent(primary: text, meta: detail)
        frame.size = preferredSize(primary: text, meta: detail, showsSpinner: true)
        revealIfNeeded()
    }

    func updateText(_ text: String) {
        updateSpeech(text: text, meta: metaText)
    }

    func updateSpeech(text: String, meta: String?) {
        displayMode = .speech
        applyContent(primary: text, meta: meta)
    }

    func updateStatus(text: String, detail: String?) {
        displayMode = .status
        applyContent(primary: text, meta: detail)
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
                self?.statusSpinner.stopAnimation(nil)
                self?.statusSpinner.isHidden = true
            }
        }
    }

    private func revealIfNeeded() {
        needsLayout = true
        layoutSubtreeIfNeeded()
        needsDisplay = true

        if !isHidden && alphaValue > 0 {
            return
        }

        isHidden = false
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

        NSAnimationContext.runAnimationGroup { ctx in
            ctx.duration = 0.2
            self.animator().alphaValue = 1.0
        }
    }

    private func applyContent(primary: String, meta: String?) {
        primaryText = primary
        metaText = meta?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty == false ? meta : nil
        primaryLabel.stringValue = primaryText
        metaLabel.stringValue = metaText ?? ""
        primaryLabel.alignment = .left
        metaLabel.alignment = .left
        needsLayout = true
        needsDisplay = true
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

    private struct ContentLayout {
        let primaryFrame: NSRect
        let metaFrame: NSRect
        let spinnerFrame: NSRect?
    }

    private func contentLayout(for bounds: NSRect, primary: String, meta: String?, showsSpinner: Bool) -> ContentLayout {
        let bodyRect = bubbleBodyRect(in: bounds)
        let primaryFont = primaryLabel.font ?? .systemFont(ofSize: 13, weight: .medium)
        let metaFont = metaLabel.font ?? .systemFont(ofSize: 11, weight: .medium)

        let primaryTextWidth = max(40, bodyRect.width - horizontalPadding * 2 - (showsSpinner ? spinnerSize.width + spinnerTextGap : 0))
        let metaTextWidth = max(40, bodyRect.width - horizontalPadding * 2)
        let primaryHeight = measuredTextHeight(primary, width: primaryTextWidth, font: primaryFont)
        let metaHeight = measuredTextHeight(meta ?? "", width: metaTextWidth, font: metaFont)
        let topRowHeight = max(showsSpinner ? spinnerSize.height : 0, primaryHeight)
        let hasMeta = meta?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty == false
        let totalContentHeight = topRowHeight + (hasMeta ? metaSpacing + metaHeight : 0)
        let startY = bodyRect.minY + max(0, floor((bodyRect.height - totalContentHeight) / 2))
        let metaFrame: NSRect
        let topRowY: CGFloat
        if hasMeta {
            metaFrame = NSRect(
                x: bodyRect.minX + horizontalPadding,
                y: startY,
                width: metaTextWidth,
                height: metaHeight
            )
            topRowY = metaFrame.maxY + metaSpacing
        } else {
            metaFrame = .zero
            topRowY = startY
        }

        if showsSpinner {
            let primarySingleWidth = measuredTextWidth(primary, font: primaryFont)
            let rowWidth = min(bodyRect.width - horizontalPadding * 2, spinnerSize.width + spinnerTextGap + max(primarySingleWidth, 40))
            let rowX = bodyRect.minX + max(horizontalPadding, floor((bodyRect.width - rowWidth) / 2))
            let spinnerFrame = NSRect(
                x: rowX,
                y: topRowY + floor((topRowHeight - spinnerSize.height) / 2),
                width: spinnerSize.width,
                height: spinnerSize.height
            )
            let primaryFrame = NSRect(
                x: spinnerFrame.maxX + spinnerTextGap,
                y: topRowY,
                width: primaryTextWidth,
                height: primaryHeight
            )
            return ContentLayout(primaryFrame: primaryFrame, metaFrame: metaFrame, spinnerFrame: spinnerFrame)
        }

        let primaryFrame = NSRect(
            x: bodyRect.minX + horizontalPadding,
            y: topRowY,
            width: primaryTextWidth,
            height: primaryHeight
        )
        return ContentLayout(primaryFrame: primaryFrame, metaFrame: metaFrame, spinnerFrame: nil)
    }

    private func measuredTextHeight(_ text: String, width: CGFloat, font: NSFont) -> CGFloat {
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return 0 }
        let textBounds = NSString(string: trimmed).boundingRect(
            with: NSSize(width: width, height: maxBubbleHeight),
            options: [.usesLineFragmentOrigin, .usesFontLeading],
            attributes: [.font: font]
        )
        return ceil(textBounds.height)
    }

    private func measuredTextWidth(_ text: String, font: NSFont) -> CGFloat {
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return 0 }
        let textBounds = NSString(string: trimmed).boundingRect(
            with: NSSize(width: maxBubbleWidth, height: maxBubbleHeight),
            options: [.usesLineFragmentOrigin, .usesFontLeading],
            attributes: [.font: font]
        )
        return ceil(textBounds.width) + textWidthSlack
    }

    func debugLayoutSnapshot(primary: String, meta: String?, showsSpinner: Bool, size: NSSize) -> (body: NSRect, primary: NSRect, meta: NSRect, spinner: NSRect?) {
        let bounds = NSRect(origin: .zero, size: size)
        let layout = contentLayout(for: bounds, primary: primary, meta: meta, showsSpinner: showsSpinner)
        return (bubbleBodyRect(in: bounds), layout.primaryFrame, layout.metaFrame, layout.spinnerFrame)
    }
}
