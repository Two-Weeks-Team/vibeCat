import AppKit
import VibeCatCore

@MainActor
final class CatPanel: NSPanel {
    private let imageView = NSImageView()
    private let emotionIndicator = NSTextField(labelWithString: "")
    private let bubbleView = ChatBubbleView()
    private let spinnerView = NSProgressIndicator()
    private let catViewModel: CatViewModel
    private let spriteAnimator: SpriteAnimator
    private var spriteSize: CGFloat = 100

    private weak var audioPlayer: AudioPlayer?
    private weak var screenAnalyzer: ScreenAnalyzer?
    private var smartHideTimer: Timer?
    private var hideCountdownTimer: Timer?
    private var bubbleDuration: TimeInterval = 2.0
    private var currentBubbleText: String?
    private var turnActive = false
    private var bubbleShownAt: Date?
    private let maxBubbleDisplayTime: TimeInterval = 15.0

    var traceLogContextProvider: (() -> String?)?
    var onBubbleDidHide: (() -> Void)?

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

        spinnerView.style = .spinning
        spinnerView.controlSize = .small
        spinnerView.isHidden = true
        spinnerView.frame = NSRect(x: 0, y: 0, width: 24, height: 24)
        contentView.addSubview(spinnerView)

        emotionIndicator.font = NSFont.systemFont(ofSize: 18)
        emotionIndicator.textColor = .white
        emotionIndicator.isHidden = true
        emotionIndicator.alphaValue = 0
        emotionIndicator.wantsLayer = true
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
            guard let self else { return }
            self.setFrame(screenFrame, display: true)
            self.layoutOverlayElements()
        }

        catViewModel.onPositionUpdate = { [weak self] localPoint in
            guard let self else { return }
            self.updateSpritePosition(localPoint)
        }
    }

    func showBubble(text: String) {
        let preview = String(text.prefix(50))
        NSLog("[BUBBLE] showBubble: %@", preview)
        let displayText = text
        let wasVisible = !bubbleView.isHidden && bubbleView.alphaValue > 0
        if let traceContext = traceLogContextProvider?() {
            NSLog("[TRACE] %@ phase=%@ text_len=%d", traceContext, wasVisible ? "bubble_update" : "bubble_show", displayText.count)
        }
        currentBubbleText = displayText
        bubbleShownAt = Date()
        bubbleDuration = 2.0
        hideCountdownTimer?.invalidate()
        hideCountdownTimer = nil
        updateBubbleFrame(for: displayText)
        if wasVisible {
            bubbleView.updateText(displayText)
        } else {
            bubbleView.show(text: displayText)
        }
        ensureSmartHidePolling()
    }

    func hideBubble() {
        if let traceContext = traceLogContextProvider?() {
            NSLog("[TRACE] %@ phase=bubble_hide", traceContext)
        }
        bubbleView.hide()
        currentBubbleText = nil
        bubbleShownAt = nil
        hideCountdownTimer?.invalidate()
        hideCountdownTimer = nil
        smartHideTimer?.invalidate()
        smartHideTimer = nil
        onBubbleDidHide?()
    }

    func setTurnActive(_ active: Bool) {
        turnActive = active
        if !active {
            evaluateBubbleHide()
        }
    }

    private func ensureSmartHidePolling() {
        guard smartHideTimer == nil else { return }
        let timer = Timer(timeInterval: 0.5, repeats: true) { [weak self] _ in
            MainActor.assumeIsolated {
                self?.evaluateBubbleHide()
            }
        }
        RunLoop.main.add(timer, forMode: .common)
        smartHideTimer = timer
    }

    private func evaluateBubbleHide() {
        let audioActive = audioPlayer?.isPlaying ?? false
        let exceededMax = bubbleShownAt.map { Date().timeIntervalSince($0) > maxBubbleDisplayTime } ?? false

        if exceededMax {
            NSLog("[BUBBLE] force-hide: exceeded %.0fs max", maxBubbleDisplayTime)
            hideBubble()
            return
        }

        if turnActive || audioActive {
            hideCountdownTimer?.invalidate()
            hideCountdownTimer = nil
            return
        }

        guard hideCountdownTimer == nil else { return }
        hideCountdownTimer = Timer.scheduledTimer(withTimeInterval: bubbleDuration, repeats: false) { [weak self] _ in
            Task { @MainActor [weak self] in
                self?.hideBubble()
            }
        }
    }

    func setEmotionIndicator(_ text: String?) {
        let trimmed = text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        NSLog("[BUBBLE] setEmotionIndicator: %@", trimmed)

        if trimmed.isEmpty {
            stopEmotionAnimation()
            NSAnimationContext.runAnimationGroup({ context in
                context.duration = 0.25
                self.emotionIndicator.animator().alphaValue = 0
            }, completionHandler: { [weak self] in
                Task { @MainActor in
                    self?.emotionIndicator.isHidden = true
                    self?.emotionIndicator.stringValue = ""
                }
            })
        } else {
            emotionIndicator.stringValue = trimmed
            emotionIndicator.isHidden = false
            emotionIndicator.alphaValue = 0
            layoutOverlayElements()

            NSAnimationContext.runAnimationGroup({ context in
                context.duration = 0.3
                self.emotionIndicator.animator().alphaValue = 1.0
            })
            startEmotionAnimation()
        }
    }

    private func startEmotionAnimation() {
        guard let layer = emotionIndicator.layer else { return }
        layer.removeAllAnimations()

        let float = CAKeyframeAnimation(keyPath: "transform.translation.y")
        float.values = [0, 3, 0, -3, 0]
        float.keyTimes = [0, 0.25, 0.5, 0.75, 1.0]
        float.duration = 2.0
        float.repeatCount = .infinity
        float.timingFunction = CAMediaTimingFunction(name: .easeInEaseOut)
        layer.add(float, forKey: "float")

        let pulse = CABasicAnimation(keyPath: "transform.scale")
        pulse.fromValue = 1.0
        pulse.toValue = 1.15
        pulse.duration = 1.8
        pulse.autoreverses = true
        pulse.repeatCount = .infinity
        pulse.timingFunction = CAMediaTimingFunction(name: .easeInEaseOut)
        layer.add(pulse, forKey: "pulse")
    }

    private func stopEmotionAnimation() {
        emotionIndicator.layer?.removeAllAnimations()
    }

    func updateEmotionForState(_ state: SpriteAnimator.AnimationState) {
        switch state {
        case .happy, .celebrating:
            setEmotionIndicator("🎉")
        case .surprised:
            setEmotionIndicator("❗")
        case .thinking:
            setEmotionIndicator("💭")
        case .frustrated:
            setEmotionIndicator("😤")
        case .idle:
            setEmotionIndicator(nil)
        }
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

    func setSmartHideReferences(audioPlayer: AudioPlayer, screenAnalyzer: ScreenAnalyzer) {
        self.audioPlayer = audioPlayer
        self.screenAnalyzer = screenAnalyzer
    }

    func beginCharacterTransition() {
        showBubble(text: "캐릭터 변경 중...")
        spinnerView.isHidden = false
        spinnerView.startAnimation(nil)
        layoutSpinner()

        NSAnimationContext.runAnimationGroup { ctx in
            ctx.duration = 0.25
            self.imageView.animator().alphaValue = 0.25
        }
    }

    func endCharacterTransition(characterName: String) {
        NSAnimationContext.runAnimationGroup { ctx in
            ctx.duration = 0.3
            self.imageView.animator().alphaValue = 1.0
        } completionHandler: { [weak self] in
            Task { @MainActor [weak self] in
                self?.spinnerView.stopAnimation(nil)
                self?.spinnerView.isHidden = true
                self?.showBubble(text: "\(characterName) 등장!")
            }
        }
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
        emotionIndicator.frame.origin = NSPoint(x: imageView.frame.maxX - 4, y: imageView.frame.maxY - 4)
        if !bubbleView.isHidden, let text = currentBubbleText {
            updateBubbleFrame(for: text)
        }
    }

    private func layoutOverlayElements() {
        let catFrame = imageView.frame
        emotionIndicator.frame.origin = NSPoint(x: catFrame.maxX - 4, y: catFrame.maxY - 4)
        layoutSpinner()
        if let text = currentBubbleText {
            updateBubbleFrame(for: text)
        }
    }

    private func layoutSpinner() {
        let catFrame = imageView.frame
        spinnerView.frame.origin = NSPoint(
            x: catFrame.midX - spinnerView.frame.width / 2,
            y: catFrame.midY - spinnerView.frame.height / 2
        )
    }

    private func updateBubbleFrame(for text: String) {
        guard let screenFrame = screen?.visibleFrame ?? NSScreen.main?.visibleFrame,
              let _ = contentView else { return }

        let size = bubbleView.preferredSize(for: text)
        let catFrame = imageView.frame

        var bubbleX = catFrame.midX - size.width / 2
        var bubbleY = catFrame.maxY + 8
        var tailDir: ChatBubbleView.TailDirection = .bottom

        let projectedTop = frame.minY + bubbleY + size.height
        if projectedTop > screenFrame.maxY - 8 {
            bubbleY = catFrame.minY - size.height - 8
            tailDir = .top
        }

        let projectedLeft = frame.minX + bubbleX
        let projectedRight = projectedLeft + size.width
        if projectedRight > screenFrame.maxX - 8 {
            bubbleX -= (projectedRight - screenFrame.maxX + 8)
        }
        if projectedLeft < screenFrame.minX + 8 {
            bubbleX += (screenFrame.minX + 8 - projectedLeft)
        }

        let tailLocalX = catFrame.midX - bubbleX
        let tailRatio = tailLocalX / size.width
        bubbleView.setTailPosition(tailRatio)
        bubbleView.setTailDirection(tailDir)
        bubbleView.frame = NSRect(x: bubbleX, y: bubbleY, width: size.width, height: size.height)
        bubbleView.layoutSubtreeIfNeeded()
    }
}
