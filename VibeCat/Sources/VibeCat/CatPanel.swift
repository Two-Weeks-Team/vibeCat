import AppKit
import VibeCatCore

@MainActor
final class CatPanel: NSPanel {
    private let imageView = NSImageView()
    private let bubbleView = ChatBubbleView()
    private let catViewModel: CatViewModel
    private let spriteAnimator: SpriteAnimator
    private var catSize: CGFloat = 100
    private let horizontalPadding: CGFloat = 10
    private let bottomPadding: CGFloat = 10
    private let topPadding: CGFloat = 50

    init(catViewModel: CatViewModel, spriteAnimator: SpriteAnimator) {
        self.catViewModel = catViewModel
        self.spriteAnimator = spriteAnimator

        let size = Self.panelSize(for: 100)
        let screen = NSScreen.main?.frame ?? NSRect(x: 0, y: 0, width: 1440, height: 900)
        let origin = NSPoint(x: screen.maxX - size.width - 20, y: screen.minY + 20)

        super.init(
            contentRect: NSRect(origin: origin, size: size),
            styleMask: [.borderless, .nonactivatingPanel],
            backing: .buffered,
            defer: false
        )

        level = .floating
        backgroundColor = .clear
        isOpaque = false
        hasShadow = false
        collectionBehavior = [.canJoinAllSpaces, .stationary, .ignoresCycle]
        ignoresMouseEvents = true
        isReleasedWhenClosed = false

        setupViews()
        wireAnimator()
        wireViewModel()
    }

    private func setupViews() {
        guard let contentView else { return }

        imageView.frame = imageFrame(for: catSize, in: contentView.bounds)
        imageView.imageScaling = .scaleProportionallyUpOrDown
        contentView.addSubview(imageView)

        bubbleView.frame = NSRect(x: -180, y: topPadding + catSize - 8, width: 180, height: 50)
        contentView.addSubview(bubbleView)
    }

    private func wireAnimator() {
        spriteAnimator.onFrameUpdate = { [weak self] image in
            self?.imageView.image = image
        }
    }

    private func wireViewModel() {
        catViewModel.onPositionUpdate = { [weak self] point in
            guard let self else { return }
            let currentFrame = self.frame
            let newOrigin = NSPoint(x: point.x - currentFrame.width / 2, y: point.y - currentFrame.height / 2)
            self.setFrameOrigin(newOrigin)
        }
    }

    func showBubble(text: String) {
        updateBubbleFrame(for: text)
        bubbleView.show(text: text)
    }

    func setCatSize(_ presetSize: String?) {
        switch presetSize?.lowercased() {
        case "medium":
            catSize = 120
        case "large":
            catSize = 150
        default:
            catSize = 100
        }

        let oldFrame = frame
        let newSize = Self.panelSize(for: catSize)
        let centeredOrigin = NSPoint(
            x: oldFrame.midX - newSize.width / 2,
            y: oldFrame.midY - newSize.height / 2
        )
        setFrame(NSRect(origin: centeredOrigin, size: newSize), display: true)

        guard let contentView else { return }
        imageView.frame = imageFrame(for: catSize, in: contentView.bounds)
    }

    func show() {
        orderFront(nil)
    }

    private static func panelSize(for catSize: CGFloat) -> NSSize {
        NSSize(width: catSize + 20, height: catSize + 60)
    }

    private func imageFrame(for catSize: CGFloat, in contentBounds: NSRect) -> NSRect {
        NSRect(
            x: (contentBounds.width - catSize) / 2,
            y: bottomPadding,
            width: catSize,
            height: catSize
        )
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
