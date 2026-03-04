import AppKit
import VibeCatCore

@MainActor
final class CatPanel: NSPanel {
    private let imageView = NSImageView()
    private let emotionIndicator = NSTextField(labelWithString: "")
    private let bubbleView = ChatBubbleView()
    private let catViewModel: CatViewModel
    private let spriteAnimator: SpriteAnimator
    private var spriteSize: CGFloat = 100

    init(catViewModel: CatViewModel, spriteAnimator: SpriteAnimator) {
        self.catViewModel = catViewModel
        self.spriteAnimator = spriteAnimator

        let mouseGlobal = NSEvent.mouseLocation
        let screenFrame = NSScreen.screens.first(where: { NSMouseInRect(mouseGlobal, $0.frame, false) })?.frame
            ?? NSScreen.main?.frame
            ?? NSRect(x: 0, y: 0, width: 1440, height: 900)

        super.init(
            contentRect: screenFrame,
            styleMask: [.borderless, .nonactivatingPanel],
            backing: .buffered,
            defer: false
        )

        level = .statusBar
        backgroundColor = .clear
        isOpaque = false
        hasShadow = false
        collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary]
        ignoresMouseEvents = true
        isReleasedWhenClosed = false

        setupViews()
        wireAnimator()
        wireViewModel()
    }

    private func setupViews() {
        guard let contentView else { return }

        imageView.frame = NSRect(x: 0, y: 0, width: spriteSize, height: spriteSize)
        imageView.imageScaling = .scaleProportionallyUpOrDown
        contentView.addSubview(imageView)

        emotionIndicator.font = NSFont.systemFont(ofSize: 18)
        emotionIndicator.textColor = .white
        emotionIndicator.isHidden = true
        emotionIndicator.frame = NSRect(x: 0, y: 0, width: 28, height: 24)
        contentView.addSubview(emotionIndicator)

        bubbleView.frame = NSRect(x: 0, y: 0, width: 180, height: 50)
        contentView.addSubview(bubbleView)
    }

    private func wireAnimator() {
        spriteAnimator.onFrameUpdate = { [weak self] image in
            self?.imageView.image = image
        }
    }

    private func wireViewModel() {
        catViewModel.onScreenFrameUpdate = { [weak self] screenFrame in
            self?.setFrame(screenFrame, display: true)
        }

        catViewModel.onPositionUpdate = { [weak self] localPoint in
            guard let self else { return }
            self.updateSpritePosition(localPoint)
        }
    }

    func showBubble(text: String) {
        updateBubbleFrame(for: text)
        bubbleView.show(text: text)
    }

    func setEmotionIndicator(_ text: String?) {
        let trimmed = text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        emotionIndicator.stringValue = trimmed
        emotionIndicator.isHidden = trimmed.isEmpty
        layoutOverlayElements()
    }

    func applySpriteSize(presetSize: String?) {
        switch presetSize?.lowercased() {
        case "medium":
            spriteSize = 120
        case "large":
            spriteSize = 150
        default:
            spriteSize = 100
        }

        imageView.frame.size = NSSize(width: spriteSize, height: spriteSize)
        layoutOverlayElements()
    }

    func show() {
        orderFrontRegardless()
    }

    func catPositionInScreenCoordinates() -> CGPoint {
        CGPoint(x: frame.minX + imageView.frame.midX, y: frame.minY + imageView.frame.midY)
    }

    private func updateSpritePosition(_ localPoint: CGPoint) {
        imageView.frame.origin = NSPoint(
            x: localPoint.x - imageView.frame.width / 2,
            y: localPoint.y - imageView.frame.height / 2
        )
        layoutOverlayElements()
    }

    private func layoutOverlayElements() {
        let catFrame = imageView.frame
        emotionIndicator.frame.origin = NSPoint(x: catFrame.maxX + 4, y: catFrame.maxY - 20)
    }

    private func updateBubbleFrame(for text: String) {
        guard let screenFrame = screen?.visibleFrame ?? NSScreen.main?.visibleFrame,
              let contentView else { return }

        let size = bubbleView.preferredSize(for: text)
        let catFrame = imageView.frame

        var bubbleX = catFrame.maxX + 8
        var tailDirection: ChatBubbleView.TailDirection = .left
        var bubbleY = catFrame.maxY - size.height + 12

        let projectedRight = frame.minX + bubbleX + size.width
        if projectedRight > screenFrame.maxX - 8 {
            bubbleX = catFrame.minX - size.width - 8
            tailDirection = .right
        }

        let projectedTop = frame.minY + bubbleY + size.height
        if projectedTop > screenFrame.maxY - 8 {
            bubbleY = catFrame.minY - size.height - 10
            tailDirection = .up
        }

        if frame.minY + bubbleY < screenFrame.minY + 8 {
            bubbleY = catFrame.maxY - size.height + 12
            if tailDirection == .up {
                tailDirection = bubbleX < catFrame.minX ? .right : .left
            }
        }

        bubbleView.setTailDirection(tailDirection)
        bubbleView.frame = NSRect(x: bubbleX, y: bubbleY, width: size.width, height: size.height)
        contentView.needsLayout = true
    }
}
