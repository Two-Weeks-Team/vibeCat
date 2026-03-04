import AppKit
import VibeCatCore

@MainActor
final class CatPanel: NSPanel {
    private let imageView = NSImageView()
    private let bubbleView = ChatBubbleView()
    private let catViewModel: CatViewModel
    private let spriteAnimator: SpriteAnimator

    init(catViewModel: CatViewModel, spriteAnimator: SpriteAnimator) {
        self.catViewModel = catViewModel
        self.spriteAnimator = spriteAnimator

        let size = NSSize(width: 120, height: 160)
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

        imageView.frame = NSRect(x: 10, y: 40, width: 100, height: 100)
        imageView.imageScaling = .scaleProportionallyUpOrDown
        contentView.addSubview(imageView)

        bubbleView.frame = NSRect(x: -140, y: 100, width: 200, height: 50)
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
        bubbleView.show(text: text)
    }

    func show() {
        orderFront(nil)
    }
}
