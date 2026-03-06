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
    }

    func pause() {
        isRunning = false
        analysisTimer?.invalidate()
        analysisTimer = nil
    }

    func resume() {
        guard !isRunning else { return }
        isRunning = true
        scheduleNextCapture()
    }

    private func scheduleNextCapture() {
        let interval = AppSettings.shared.captureInterval
        analysisTimer = Timer.scheduledTimer(withTimeInterval: interval, repeats: false) { [weak self] _ in
            Task { @MainActor [weak self] in
                await self?.runAnalysisCycle()
            }
        }
    }

    private func runAnalysisCycle() async {
        let audioActive = audioPlayer?.isPlaying ?? false
        NSLog("[CAPTURE] runAnalysisCycle: start, isRunning=%d, isAnalyzing=%d, isConnected=%d, audioActive=%d", isRunning, isAnalyzing, gatewayClient.isConnected, audioActive)
        guard isRunning, !isAnalyzing else {
            if isRunning { scheduleNextCapture() }
            return
        }
        if audioActive {
            NSLog("[CAPTURE] runAnalysisCycle: audio playing, capturing but won't interrupt speech")
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

        let result = await captureService.captureAroundCursor()
        switch result {
        case .unchanged:
            NSLog("[CAPTURE] runAnalysisCycle: result=unchanged")
            return
        case .unavailable(let reason):
            NSLog("[CAPTURE] runAnalysisCycle: result=unavailable, reason=%@", reason)
            return
        case .captured(let image):
            NSLog("[CAPTURE] runAnalysisCycle: result=captured, width=%zu, height=%zu", image.width, image.height)
            await sendToGateway(image: image, highSignificance: false)
        }
    }

    func forceAnalysis() async {
        guard gatewayClient.isConnected else { return }
        let result = await captureService.forceCapture()
        if case .captured(let image) = result {
            await sendToGateway(image: image, highSignificance: true)
        }
    }

    private func sendToGateway(image: CGImage, highSignificance: Bool) async {
        guard gatewayClient.isConnected else { return }
        guard let base64 = ImageProcessor.toBase64JPEG(image) else { return }

        let context = buildContext()
        let character = AppSettings.shared.character
        let soul = spriteAnimator.loadPreset(for: character).soul

        NSLog("[CAPTURE] sendToGateway: base64Length=%lu, character=%@, context=%@, highSignificance=%d", base64.count, character, context, highSignificance)

        if highSignificance {
            gatewayClient.sendForceCapture(imageBase64: base64, context: context, userId: userId, character: character, soul: soul, activityMinutes: activityMinutes)
        } else {
            gatewayClient.sendScreenCapture(imageBase64: base64, context: context, userId: userId, character: character, soul: soul, activityMinutes: activityMinutes)
        }

        spriteAnimator.setState(.thinking)
    }

    private func buildContext() -> String {
        let frontApp = NSWorkspace.shared.frontmostApplication?.localizedName ?? "Unknown"
        return "Active app: \(frontApp)"
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
        // Sprite idle is managed by TTS lifecycle (ttsEnd → delayed idle in AppDelegate)
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
