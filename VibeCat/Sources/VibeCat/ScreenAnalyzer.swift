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

    private var captureLoopTask: Task<Void, Never>?
    private var isRunning = false
    private(set) var isAnalyzing = false
    private var userId: String = "local-user"
    private var sessionStartTime: Date = Date()

    private var workspaceObserver: NSObjectProtocol?
    private var appSwitchDebounceTimer: Timer?

    private var lastFastPathSend: Date = .distantPast
    private var lastSmartPathSend: Date = .distantPast
    private let fastPathCooldown: TimeInterval = 1.0
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
        startCaptureLoop()
        observeAppSwitches()
    }

    func pause() {
        isRunning = false
        captureLoopTask?.cancel()
        captureLoopTask = nil
        removeAppSwitchObserver()
    }

    func resume() {
        guard !isRunning else { return }
        isRunning = true
        startCaptureLoop()
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

    private let initialStabilizationDelay: UInt64 = 10_000_000_000

    private func startCaptureLoop() {
        captureLoopTask?.cancel()
        captureLoopTask = Task { @MainActor [weak self] in
            guard let self else { return }

            NSLog("[CAPTURE] stabilization delay: waiting 10s before first capture")
            try? await Task.sleep(nanoseconds: self.initialStabilizationDelay)
            guard !Task.isCancelled else { return }
            NSLog("[CAPTURE] stabilization complete — starting capture loop")

            while self.isRunning && !Task.isCancelled {
                if !self.isAnalyzing {
                    await self.runAnalysisCycle(forceSmartPath: false)
                }

                let interval = UInt64(self.effectiveCaptureInterval * 1_000_000_000)
                try? await Task.sleep(nanoseconds: interval)
            }
        }
    }

    private var effectiveCaptureInterval: TimeInterval {
        max(1.0, AppSettings.shared.captureInterval)
    }

    private var isSpeechActive: Bool {
        let audioPlaying = audioPlayer?.isPlaying ?? false
        let ttsSpeaking = gatewayClient.isTTSSpeaking
        let recentlyStopped = Date().timeIntervalSince(gatewayClient.lastSpeechEndTime) < postSpeechCooldown
        return audioPlaying || ttsSpeaking || recentlyStopped
    }

    private func runAnalysisCycle(forceSmartPath: Bool) async {
        NSLog("[CAPTURE] runAnalysisCycle: isRunning=%d, isAnalyzing=%d, isConnected=%d",
              isRunning, isAnalyzing, gatewayClient.isConnected)
        guard isRunning, !isAnalyzing else { return }
        guard gatewayClient.isConnected else { return }
        isAnalyzing = true
        defer { isAnalyzing = false }

        let result = await captureService.captureAroundCursor()
        switch result {
        case .unchanged:
            NSLog("[CAPTURE] unchanged")
            return
        case .unavailable(let reason):
            NSLog("[CAPTURE] unavailable: %@", reason)
            return
        case .captured(let snapshot):
            NSLog("[CAPTURE] captured: %dx%d target=%@ app=%@ window=%@",
                  snapshot.image.width,
                  snapshot.image.height,
                  snapshot.targetKind.rawValue,
                  snapshot.appName,
                  snapshot.windowTitle ?? "")

            let now = Date()

            if now.timeIntervalSince(lastFastPathSend) >= fastPathCooldown {
                sendFastPath(image: snapshot.image)
                lastFastPathSend = now
            }

            let speechActive = isSpeechActive
            if !speechActive {
                let smartPathReady = forceSmartPath || now.timeIntervalSince(lastSmartPathSend) >= smartPathCooldown
                if smartPathReady {
                    await sendSmartPath(snapshot: snapshot, highSignificance: forceSmartPath)
                    lastSmartPathSend = now
                }
            }
        }
    }

    func forceAnalysis() async {
        guard gatewayClient.isConnected else { return }
        let result = await captureService.captureFullWindow()
        if case .captured(let snapshot) = result {
            sendFastPath(image: snapshot.image)
            lastFastPathSend = Date()
            if !isSpeechActive {
                await sendSmartPath(snapshot: snapshot, highSignificance: true)
                lastSmartPathSend = Date()
            }
        }
    }

    // MARK: - Fast Path (video frame → Gemini Live API)

    private func sendFastPath(image: CGImage) {
        let img = image
        let client = gatewayClient
        DispatchQueue.global(qos: .userInitiated).async {
            guard let jpegData = ImageProcessor.toFastPathJPEG(img) else { return }
            NSLog("[CAPTURE] Fast Path: sending %d bytes JPEG to Live API", jpegData.count)
            DispatchQueue.main.async { client.sendVideoFrame(jpegData) }
        }
    }

    // MARK: - Smart Path (base64 image → ADK orchestrator)

    private func sendSmartPath(snapshot: ScreenCaptureService.CaptureSnapshot, highSignificance: Bool) async {
        guard gatewayClient.isConnected else { return }

        let character = AppSettings.shared.character
        let soul = spriteAnimator.loadPreset(for: character).soul

        let img = snapshot.image
        let base64: String? = await withCheckedContinuation { continuation in
            DispatchQueue.global(qos: .userInitiated).async {
                let result = ImageProcessor.toBase64JPEG(img)
                continuation.resume(returning: result)
            }
        }
        guard let base64 else { return }

        NSLog("[CAPTURE] Smart Path: base64=%d bytes, character=%@, context=%@",
              base64.count, character, snapshot.contextDescription)

        if highSignificance {
            gatewayClient.sendForceCapture(
                imageBase64: base64,
                context: snapshot.contextDescription,
                userId: userId,
                character: character,
                soul: soul,
                activityMinutes: activityMinutes
            )
        } else {
            gatewayClient.sendScreenCapture(
                imageBase64: base64,
                context: snapshot.contextDescription,
                userId: userId,
                character: character,
                soul: soul,
                activityMinutes: activityMinutes
            )
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
