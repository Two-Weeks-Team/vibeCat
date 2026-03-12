import AppKit
import VibeCatCore

@MainActor
final class CatPanel: NSPanel {
    private enum WindowTitleBadgeLayout {
        static let cornerRadius: CGFloat = 10
        static let horizontalPadding: CGFloat = 14
        static let verticalPadding: CGFloat = 8
        static let minimumWidth: CGFloat = 120
        static let maximumWidth: CGFloat = 280
        static let topSpacing: CGFloat = 8
    }

    private enum PrivacyBadgeLayout {
        static let cornerRadius: CGFloat = 11
        static let horizontalPadding: CGFloat = 30
        static let verticalPadding: CGFloat = 14
        static let sideSpacing: CGFloat = 4
        static let minimumInset: CGFloat = 8
        static let titleLeading: CGFloat = 24
        static let titleTopInset: CGFloat = 6
        static let detailLeading: CGFloat = 10
        static let detailHorizontalPadding: CGFloat = 16
        static let detailBottomInset: CGFloat = 5
        static let badgeBackgroundOpacity: CGFloat = 0.56
        static let dotSize = NSSize(width: 8, height: 8)
        static let dotTopInset: CGFloat = 17
    }

    private let imageView = NSImageView()
    private let emotionIndicator = NSTextField(labelWithString: "")
    private let bubbleView = ChatBubbleView()
    private let spinnerView = NSProgressIndicator()
    private let privacyBadgeView = NSVisualEffectView()
    private let privacyDotView = NSView()
    private let privacyTitleLabel = NSTextField(labelWithString: "")
    private let privacyDetailLabel = NSTextField(labelWithString: "")
    private let windowTitleBadgeView = NSVisualEffectView()
    private let windowTitleLabel = NSTextField(labelWithString: "")
    private let catViewModel: CatViewModel
    private let spriteAnimator: SpriteAnimator
    private var spriteSize: CGFloat = 100

    private weak var audioPlayer: AudioPlayer?
    private weak var screenAnalyzer: ScreenAnalyzer?
    private var smartHideTimer: Timer?
    private var hideCountdownTimer: Timer?
    private var privacyBadgeHideTimer: Timer?
    private var bubbleDuration: TimeInterval = 2.0
    private var currentBubbleText: String?
    private var currentBubbleMeta: String?
    private var bubbleShowsSpinner = false
    private var turnActive = false
    private var bubbleShownAt: Date?
    private let maxBubbleDisplayTime: TimeInterval = 15.0
    private let privacyBadgeDisplayTime: TimeInterval = 3.0

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

        privacyBadgeView.material = .hudWindow
        privacyBadgeView.blendingMode = .withinWindow
        privacyBadgeView.state = .active
        privacyBadgeView.wantsLayer = true
        privacyBadgeView.layer?.cornerRadius = PrivacyBadgeLayout.cornerRadius
        privacyBadgeView.layer?.masksToBounds = true
        privacyBadgeView.layer?.backgroundColor = NSColor.black.withAlphaComponent(PrivacyBadgeLayout.badgeBackgroundOpacity).cgColor
        privacyBadgeView.alphaValue = 0
        privacyBadgeView.isHidden = true
        contentView.addSubview(privacyBadgeView)

        privacyDotView.wantsLayer = true
        privacyDotView.layer?.cornerRadius = PrivacyBadgeLayout.dotSize.width / 2
        privacyBadgeView.addSubview(privacyDotView)

        privacyTitleLabel.font = NSFont.systemFont(ofSize: 11, weight: .semibold)
        privacyTitleLabel.textColor = .white
        privacyBadgeView.addSubview(privacyTitleLabel)

        privacyDetailLabel.font = NSFont.systemFont(ofSize: 10, weight: .regular)
        privacyDetailLabel.textColor = NSColor.white.withAlphaComponent(0.78)
        privacyBadgeView.addSubview(privacyDetailLabel)

        windowTitleBadgeView.material = .hudWindow
        windowTitleBadgeView.blendingMode = .withinWindow
        windowTitleBadgeView.state = .active
        windowTitleBadgeView.wantsLayer = true
        windowTitleBadgeView.layer?.cornerRadius = WindowTitleBadgeLayout.cornerRadius
        windowTitleBadgeView.layer?.masksToBounds = true
        windowTitleBadgeView.layer?.backgroundColor = NSColor.black.withAlphaComponent(0.52).cgColor
        windowTitleBadgeView.isHidden = true
        contentView.addSubview(windowTitleBadgeView)

        windowTitleLabel.font = NSFont.systemFont(ofSize: 10, weight: .medium)
        windowTitleLabel.textColor = .white
        windowTitleLabel.lineBreakMode = .byTruncatingMiddle
        windowTitleBadgeView.addSubview(windowTitleLabel)

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
        showSpeechBubble(text: text, meta: nil)
    }

    func showSpeechBubble(text: String, meta: String?) {
        let displayText = text
        let wasVisible = !bubbleView.isHidden && bubbleView.alphaValue > 0
        NSLog(
            "[BUBBLE] speech %@: text=%@ meta=%@",
            wasVisible && !bubbleView.isShowingStatus ? "update" : "show",
            displayText,
            meta ?? ""
        )
        if let traceContext = traceLogContextProvider?() {
            NSLog("[TRACE] %@ phase=%@ text_len=%d", traceContext, wasVisible ? "bubble_update" : "bubble_show", displayText.count)
        }
        currentBubbleText = displayText
        currentBubbleMeta = meta
        bubbleShowsSpinner = false
        bubbleShownAt = Date()
        bubbleDuration = 2.0
        hideCountdownTimer?.invalidate()
        hideCountdownTimer = nil
        updateBubbleFrame()
        if wasVisible && !bubbleView.isShowingStatus {
            bubbleView.updateSpeech(text: displayText, meta: meta)
        } else {
            bubbleView.showSpeech(text: displayText, meta: meta)
        }
        ensureSmartHidePolling()
    }

    func showStatusBubble(text: String, detail: String?) {
        NSLog("[BUBBLE] status show: text=%@ detail=%@", text, detail ?? "")
        currentBubbleText = text
        currentBubbleMeta = detail
        bubbleShowsSpinner = true
        bubbleShownAt = Date()
        bubbleDuration = 2.0
        hideCountdownTimer?.invalidate()
        hideCountdownTimer = nil
        updateBubbleFrame()
        if !spinnerView.isHidden {
            spinnerView.stopAnimation(nil)
            spinnerView.isHidden = true
        }
        if bubbleView.isHidden || bubbleView.alphaValue == 0 || !bubbleView.isShowingStatus {
            bubbleView.showStatus(text: text, detail: detail)
        } else {
            bubbleView.updateStatus(text: text, detail: detail)
        }
        ensureSmartHidePolling()
    }

    func hideStatusBubbleIfShowing() {
        guard bubbleView.isShowingStatus else { return }
        hideBubble()
    }

    func hideBubble() {
        if let traceContext = traceLogContextProvider?() {
            NSLog("[TRACE] %@ phase=bubble_hide", traceContext)
        }
        bubbleView.hide()
        currentBubbleText = nil
        currentBubbleMeta = nil
        bubbleShowsSpinner = false
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

    func updateCapturePrivacyBadge(title: String, detail: String, accentColor: NSColor) {
        privacyTitleLabel.stringValue = title
        privacyDetailLabel.stringValue = detail
        privacyDotView.layer?.backgroundColor = accentColor.cgColor
        showPrivacyBadgeTemporarily()
    }

    func updateCurrentWindowTitle(appName: String, windowTitle: String?) {
        let app = appName.trimmingCharacters(in: .whitespacesAndNewlines)
        let title = (windowTitle ?? "").trimmingCharacters(in: .whitespacesAndNewlines)

        let text: String
        if !app.isEmpty && !title.isEmpty {
            text = "\(app) - \(title)"
        } else if !title.isEmpty {
            text = title
        } else {
            text = app
        }

        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            windowTitleBadgeView.isHidden = true
            return
        }

        windowTitleLabel.stringValue = trimmed
        windowTitleBadgeView.isHidden = false
        layoutWindowTitleBadge()
    }

    func currentGlobalProbePoint() -> CGPoint {
        CGPoint(x: frame.minX + imageView.frame.midX, y: frame.minY + imageView.frame.midY)
    }

    func beginCharacterTransition() {
        showBubble(text: VibeCatL10n.characterChanging())
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
                self?.showBubble(text: VibeCatL10n.characterAppeared(characterName))
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
        if !windowTitleBadgeView.isHidden {
            layoutWindowTitleBadge()
        }
        if !bubbleView.isHidden, currentBubbleText != nil {
            updateBubbleFrame()
        }
    }

    private func layoutOverlayElements() {
        let catFrame = imageView.frame
        emotionIndicator.frame.origin = NSPoint(x: catFrame.maxX - 4, y: catFrame.maxY - 4)
        layoutSpinner()
        if !privacyBadgeView.isHidden || privacyBadgeView.alphaValue > 0 {
            layoutPrivacyBadge()
        }
        if !windowTitleBadgeView.isHidden {
            layoutWindowTitleBadge()
        }
        if currentBubbleText != nil {
            updateBubbleFrame()
        }
    }

    private func layoutSpinner() {
        let catFrame = imageView.frame
        spinnerView.frame.origin = NSPoint(
            x: catFrame.midX - spinnerView.frame.width / 2,
            y: catFrame.midY - spinnerView.frame.height / 2
        )
    }

    private func layoutPrivacyBadge() {
        let titleSize = privacyTitleLabel.intrinsicContentSize
        let detailSize = privacyDetailLabel.intrinsicContentSize
        let badgeWidth = max(titleSize.width, detailSize.width) + PrivacyBadgeLayout.horizontalPadding
        let badgeHeight = titleSize.height + detailSize.height + PrivacyBadgeLayout.verticalPadding
        let catFrame = imageView.frame

        var badgeX = catFrame.maxX + PrivacyBadgeLayout.sideSpacing
        let badgeY = max(catFrame.midY - badgeHeight / 2, PrivacyBadgeLayout.minimumInset)

        if let visibleFrame = visibleFrameContainingCat(),
           frame.minX + badgeX + badgeWidth > visibleFrame.maxX - PrivacyBadgeLayout.minimumInset {
            badgeX = max(catFrame.minX - badgeWidth - PrivacyBadgeLayout.sideSpacing, PrivacyBadgeLayout.minimumInset)
        }

        privacyBadgeView.frame = NSRect(x: badgeX, y: badgeY, width: badgeWidth, height: badgeHeight)
        privacyDotView.frame = NSRect(
            x: PrivacyBadgeLayout.detailLeading,
            y: badgeHeight - PrivacyBadgeLayout.dotTopInset,
            width: PrivacyBadgeLayout.dotSize.width,
            height: PrivacyBadgeLayout.dotSize.height
        )
        privacyTitleLabel.frame = NSRect(
            x: PrivacyBadgeLayout.titleLeading,
            y: badgeHeight - titleSize.height - PrivacyBadgeLayout.titleTopInset,
            width: badgeWidth - PrivacyBadgeLayout.horizontalPadding,
            height: titleSize.height
        )
        privacyDetailLabel.frame = NSRect(
            x: PrivacyBadgeLayout.detailLeading,
            y: PrivacyBadgeLayout.detailBottomInset,
            width: badgeWidth - PrivacyBadgeLayout.detailHorizontalPadding,
            height: detailSize.height
        )
    }

    private func showPrivacyBadgeTemporarily() {
        privacyBadgeHideTimer?.invalidate()
        privacyBadgeHideTimer = nil
        layoutPrivacyBadge()

        if privacyBadgeView.isHidden || privacyBadgeView.alphaValue == 0 {
            privacyBadgeView.isHidden = false
            privacyBadgeView.alphaValue = 0
            NSAnimationContext.runAnimationGroup { context in
                context.duration = 0.16
                self.privacyBadgeView.animator().alphaValue = 1
            }
        }

        privacyBadgeHideTimer = Timer.scheduledTimer(withTimeInterval: privacyBadgeDisplayTime, repeats: false) { [weak self] _ in
            Task { @MainActor [weak self] in
                self?.hidePrivacyBadge()
            }
        }
    }

    private func hidePrivacyBadge() {
        privacyBadgeHideTimer?.invalidate()
        privacyBadgeHideTimer = nil
        guard !privacyBadgeView.isHidden else { return }

        NSAnimationContext.runAnimationGroup({ context in
            context.duration = 0.22
            self.privacyBadgeView.animator().alphaValue = 0
        }, completionHandler: { [weak self] in
            Task { @MainActor [weak self] in
                self?.privacyBadgeView.isHidden = true
            }
        })
    }

    private func layoutWindowTitleBadge() {
        let catFrame = imageView.frame
        let constrainedSize = NSSize(width: WindowTitleBadgeLayout.maximumWidth - WindowTitleBadgeLayout.horizontalPadding * 2, height: .greatestFiniteMagnitude)
        let textSize = windowTitleLabel.attributedStringValue.boundingRect(
            with: constrainedSize,
            options: [.usesLineFragmentOrigin, .usesFontLeading]
        ).integral.size

        let badgeWidth = min(max(textSize.width + WindowTitleBadgeLayout.horizontalPadding * 2, WindowTitleBadgeLayout.minimumWidth), WindowTitleBadgeLayout.maximumWidth)
        let badgeHeight = textSize.height + WindowTitleBadgeLayout.verticalPadding * 2
        let badgeX = catFrame.midX - badgeWidth / 2
        let badgeY = max(catFrame.minY - badgeHeight - WindowTitleBadgeLayout.topSpacing, 6)

        windowTitleBadgeView.frame = NSRect(x: badgeX, y: badgeY, width: badgeWidth, height: badgeHeight)
        windowTitleLabel.frame = NSRect(
            x: WindowTitleBadgeLayout.horizontalPadding,
            y: WindowTitleBadgeLayout.verticalPadding - 1,
            width: badgeWidth - WindowTitleBadgeLayout.horizontalPadding * 2,
            height: textSize.height + 2
        )
    }

    private func updateBubbleFrame() {
        guard let text = currentBubbleText else { return }
        guard let screenFrame = visibleFrameContainingCat(),
              let _ = contentView else { return }

        let size = bubbleView.preferredSize(primary: text, meta: currentBubbleMeta, showsSpinner: bubbleShowsSpinner)
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

    private func visibleFrameContainingCat() -> CGRect? {
        let catGlobalPoint = CGPoint(x: frame.minX + imageView.frame.midX, y: frame.minY + imageView.frame.midY)
        return NSScreen.screens.first(where: { $0.frame.contains(catGlobalPoint) })?.visibleFrame
            ?? NSScreen.main?.visibleFrame
    }
}
