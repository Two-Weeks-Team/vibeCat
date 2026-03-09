import Foundation
import AppKit
import CoreGraphics
import VibeCatCore

@MainActor
final class ScreenAnalyzer {
    private let captureService: ScreenCaptureService
    private let gatewayClient: GatewayClient
    private let catVoice: CatVoice
    private let spriteAnimator: SpriteAnimator
    private weak var audioPlayer: AudioPlayer?

    private var analysisTimer: Timer?
    private var isRunning = false
    private(set) var isAnalyzing = false
    private var userId: String = "local-user"
    private var sessionStartTime: Date = Date()

    private var workspaceObserver: NSObjectProtocol?
    private var appSwitchDebounceTimer: Timer?

    private var lastFastPathSend: Date = .distantPast
    private var lastSmartPathSend: Date = .distantPast
    private let fastPathCooldown: TimeInterval = 5.0
    private let smartPathCooldown: TimeInterval = 15.0
    private let postSpeechCooldown: TimeInterval = 5.0

    var onSpeechEvent: ((CompanionSpeechEvent) -> Void)?
    var onBackgroundSpeech: ((String) -> Void)?

    func setAudioPlayer(_ player: AudioPlayer) {
        self.audioPlayer = player
    }

    init(
        captureService: ScreenCaptureService,
        gatewayClient: GatewayClient,
        catVoice: CatVoice,
        spriteAnimator: SpriteAnimator
    ) {
        self.captureService = captureService
        self.gatewayClient = gatewayClient
        self.catVoice = catVoice
        self.spriteAnimator = spriteAnimator
    }

    var activityMinutes: Int {
        Int(Date().timeIntervalSince(sessionStartTime) / 60)
    }

    func start() {
        guard !isRunning else { return }
        isRunning = true
        sessionStartTime = Date()
        scheduleNextCapture()
        observeAppSwitches()
    }

    func pause() {
        isRunning = false
        analysisTimer?.invalidate()
        analysisTimer = nil
        removeAppSwitchObserver()
    }

    func resume() {
        guard !isRunning else { return }
        isRunning = true
        scheduleNextCapture()
        observeAppSwitches()
    }

    // MARK: - App Switch Detection

    private func observeAppSwitches() {
        removeAppSwitchObserver()
        workspaceObserver = NSWorkspace.shared.notificationCenter.addObserver(
            forName: NSWorkspace.didActivateApplicationNotification,
            object: nil,
            queue: .main
        ) { [weak self] _ in
            Task { @MainActor [weak self] in
                self?.handleAppSwitch()
            }
        }
    }

    private func removeAppSwitchObserver() {
        if let observer = workspaceObserver {
            NSWorkspace.shared.notificationCenter.removeObserver(observer)
            workspaceObserver = nil
        }
        appSwitchDebounceTimer?.invalidate()
        appSwitchDebounceTimer = nil
    }

    private func handleAppSwitch() {
        appSwitchDebounceTimer?.invalidate()
        appSwitchDebounceTimer = Timer.scheduledTimer(withTimeInterval: 0.5, repeats: false) { [weak self] _ in
            Task { @MainActor [weak self] in
                guard let self, self.isRunning, !self.isAnalyzing else { return }
                NSLog("[CAPTURE] App switched — triggering immediate capture")
                await self.runAnalysisCycle(forceSmartPath: true)
            }
        }
    }

    // MARK: - Capture Scheduling

    private func scheduleNextCapture() {
        let interval = AppSettings.shared.captureInterval
        analysisTimer = Timer.scheduledTimer(withTimeInterval: interval, repeats: false) { [weak self] _ in
            Task { @MainActor [weak self] in
                await self?.runAnalysisCycle(forceSmartPath: false)
            }
        }
    }

    private var isSpeechActive: Bool {
        let audioPlaying = audioPlayer?.isPlaying ?? false
        let ttsSpeaking = gatewayClient.isTTSSpeaking
        let recentlyStopped = Date().timeIntervalSince(gatewayClient.lastSpeechEndTime) < postSpeechCooldown
        return audioPlaying || ttsSpeaking || recentlyStopped
    }

    private func runAnalysisCycle(forceSmartPath: Bool) async {
        let speechActive = isSpeechActive
        NSLog("[CAPTURE] runAnalysisCycle: isRunning=%d, isAnalyzing=%d, isConnected=%d, speechActive=%d",
              isRunning, isAnalyzing, gatewayClient.isConnected, speechActive)
        guard isRunning, !isAnalyzing else {
            if isRunning { scheduleNextCapture() }
            return
        }
        guard gatewayClient.isConnected else {
            if isRunning { scheduleNextCapture() }
            return
        }
        isAnalyzing = true
        defer {
            isAnalyzing = false
            if isRunning { scheduleNextCapture() }
        }

        if speechActive {
            NSLog("[CAPTURE] suppressed — speech active (audio/tts/cooldown)")
            return
        }

        let result = await captureService.captureAroundCursor()
        switch result {
        case .unchanged:
            NSLog("[CAPTURE] unchanged")
            return
        case .unavailable(let reason):
            NSLog("[CAPTURE] unavailable: %@", reason)
            return
        case .captured(let image):
            NSLog("[CAPTURE] captured: %dx%d", image.width, image.height)

            let now = Date()

            if now.timeIntervalSince(lastFastPathSend) >= fastPathCooldown {
                sendFastPath(image: image)
                lastFastPathSend = now
            }

            let smartPathReady = forceSmartPath || now.timeIntervalSince(lastSmartPathSend) >= smartPathCooldown
            if smartPathReady {
                await sendSmartPath(image: image, highSignificance: forceSmartPath)
                lastSmartPathSend = now
            }
        }
    }

    func forceAnalysis() async {
        guard gatewayClient.isConnected, !isSpeechActive else { return }
        let result = await captureService.forceCapture()
        if case .captured(let image) = result {
            sendFastPath(image: image)
            lastFastPathSend = Date()
            await sendSmartPath(image: image, highSignificance: true)
            lastSmartPathSend = Date()
        }
    }

    // MARK: - Fast Path (video frame → Gemini Live API)

    private func sendFastPath(image: CGImage) {
        guard let jpegData = ImageProcessor.toFastPathJPEG(image) else { return }
        NSLog("[CAPTURE] Fast Path: sending %d bytes JPEG to Live API", jpegData.count)
        gatewayClient.sendVideoFrame(jpegData)
    }

    // MARK: - Smart Path (base64 image → ADK orchestrator)

    private func sendSmartPath(image: CGImage, highSignificance: Bool) async {
        guard gatewayClient.isConnected else { return }
        guard let base64 = ImageProcessor.toBase64JPEG(image) else { return }

        let appName = NSWorkspace.shared.frontmostApplication?.localizedName ?? "Unknown"
        let context = "[App: \(appName)]"
        let character = AppSettings.shared.character
        let soul = spriteAnimator.loadPreset(for: character).soul

        NSLog("[CAPTURE] Smart Path: base64=%d bytes, character=%@, app=%@", base64.count, character, appName)

        if highSignificance {
            gatewayClient.sendForceCapture(imageBase64: base64, context: context, userId: userId, character: character, soul: soul, activityMinutes: activityMinutes)
        } else {
            gatewayClient.sendScreenCapture(imageBase64: base64, context: context, userId: userId, character: character, soul: soul, activityMinutes: activityMinutes)
        }

        spriteAnimator.setState(.thinking)
    }

    func handleCompanionSpeech(_ event: CompanionSpeechEvent) {
        let textPreview = String(event.text.prefix(50))
        NSLog("[CAPTURE] handleCompanionSpeech: textPreview=%@, emotion=%@", textPreview, String(describing: event.emotion))
        let emotion = mapEmotion(event.emotion)
        spriteAnimator.setState(emotion)
        onSpeechEvent?(event)
        if !NSApp.isActive {
            onBackgroundSpeech?(event.text)
        }
    }

    func handleCompanionSpeechEmotion(_ emotion: CompanionEmotion) {
        NSLog("[CAPTURE] handleCompanionSpeechEmotion: emotion=%@", String(describing: emotion))
        let state = mapEmotion(emotion)
        spriteAnimator.setState(state)
    }

    private func mapEmotion(_ emotion: CompanionEmotion) -> SpriteAnimator.AnimationState {
        switch emotion {
        case .neutral: return .idle
        case .curious: return .thinking
        case .happy: return .happy
        case .surprised: return .surprised
        case .concerned: return .frustrated
        case .celebrating: return .celebrating
        }
    }
}
