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

    private var analysisTimer: Timer?
    private var isRunning = false
    private var isAnalyzing = false
    private var userId: String = "local-user"

    var onSpeechEvent: ((CompanionSpeechEvent) -> Void)?

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

    func start() {
        guard !isRunning else { return }
        isRunning = true
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
        guard isRunning, !isAnalyzing else {
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
            return
        case .unavailable(let reason):
            print("ScreenAnalyzer: capture unavailable: \(reason)")
            return
        case .captured(let image):
            await sendToGateway(image: image, highSignificance: false)
        }
    }

    func forceAnalysis() async {
        let result = await captureService.forceCapture()
        if case .captured(let image) = result {
            await sendToGateway(image: image, highSignificance: true)
        }
    }

    private func sendToGateway(image: CGImage, highSignificance: Bool) async {
        let processedImage = ImageProcessor.resizeIfNeeded(image)
        guard let base64 = ImageProcessor.toBase64JPEG(processedImage) else { return }

        let context = buildContext()

        if highSignificance {
            gatewayClient.sendForceCapture(imageBase64: base64, context: context, userId: userId)
        } else {
            gatewayClient.sendScreenCapture(imageBase64: base64, context: context, userId: userId)
        }

        spriteAnimator.setState(.thinking)
    }

    private func buildContext() -> String {
        let frontApp = NSWorkspace.shared.frontmostApplication?.localizedName ?? "Unknown"
        return "Active app: \(frontApp)"
    }

    func handleCompanionSpeech(_ event: CompanionSpeechEvent) {
        let emotion = mapEmotion(event.emotion)
        spriteAnimator.setState(emotion)
        catVoice.speak(event.text)
        onSpeechEvent?(event)

        Timer.scheduledTimer(withTimeInterval: 5.0, repeats: false) { [weak self] _ in
            Task { @MainActor [weak self] in
                self?.spriteAnimator.setState(.idle)
            }
        }
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
