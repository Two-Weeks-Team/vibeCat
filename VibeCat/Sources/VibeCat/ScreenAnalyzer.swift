import Foundation
import AppKit
import CoreGraphics
import VibeCatCore

@MainActor
final class ScreenAnalyzer {
    private struct CachedDisplayContext {
        let screenBasisID: String
        let screenshotBase64: String
        let activeDisplayID: String
        let targetDisplayID: String
        let capturedAt: Date
        let source: String
    }

    private let captureService: ScreenCaptureService
    private let gatewayClient: GatewayClient
    private let catVoice: CatVoice
    private let spriteAnimator: SpriteAnimator
    private weak var audioPlayer: AudioPlayer?

    private var captureLoopTask: Task<Void, Never>?
    private var isRunning = false
    private(set) var isAnalyzing = false
    private var userId: String = GatewayClient.deviceIdentifier()
    private var sessionStartTime: Date = Date()

    private var workspaceObserver: NSObjectProtocol?
    private var appSwitchDebounceTimer: Timer?
    private var windowProbeTask: Task<Void, Never>?

    private var lastFastPathSend: Date = .distantPast
    private var lastSmartPathSend: Date = .distantPast
    private let fastPathCooldown: TimeInterval = 1.0
    private let smartPathCooldown: TimeInterval = 15.0
    private let postSpeechCooldown: TimeInterval = 5.0
    private let maxCachedCommandContextAge: TimeInterval = 20.0

    private var cachedDisplayContext: CachedDisplayContext?

    var onSpeechEvent: ((CompanionSpeechEvent) -> Void)?
    var onBackgroundSpeech: ((String) -> Void)?
    var onScreenBasisUpdate: ((String, String?) -> Void)?

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
        refreshCapturePolicy()
    }

    func pause() {
        isRunning = false
        stopAutomaticCapture()
    }

    func resume() {
        guard !isRunning else { return }
        isRunning = true
        refreshCapturePolicy()
    }

    func reloadCapturePolicy() {
        guard isRunning else { return }
        refreshCapturePolicy()
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
        appSwitchDebounceTimer = Timer.scheduledTimer(withTimeInterval: 0.2, repeats: false) { [weak self] _ in
            Task { @MainActor [weak self] in
                guard let self, self.isRunning, !self.isAnalyzing else { return }
                NSLog("[CAPTURE] App switched — triggering immediate capture")
                await self.runAnalysisCycle(forceSmartPath: true)
            }
        }
    }

    // MARK: - Capture Scheduling

    private let initialStabilizationDelay: UInt64 = 10_000_000_000

    private var automaticCaptureEnabled: Bool {
        !AppSettings.shared.manualAnalysisOnly
    }

    private func refreshCapturePolicy() {
        stopAutomaticCapture()
        guard isRunning, automaticCaptureEnabled else { return }
        startCaptureLoop()
        startWindowProbeLoop()
        observeAppSwitches()
    }

    private func stopAutomaticCapture() {
        captureLoopTask?.cancel()
        captureLoopTask = nil
        windowProbeTask?.cancel()
        windowProbeTask = nil
        removeAppSwitchObserver()
    }

    private func startWindowProbeLoop() {
        windowProbeTask?.cancel()
        windowProbeTask = Task { @MainActor [weak self] in
            guard let self else { return }
            while self.isRunning && !Task.isCancelled {
                await self.refreshWindowProbe()
                try? await Task.sleep(nanoseconds: 120_000_000)
            }
        }
    }

    private func refreshWindowProbe() async {
        guard isRunning else { return }
        guard let snapshot = await captureService.probeWindowUnderCursor() else { return }
        if snapshot.targetKind == .displayFallback {
            onScreenBasisUpdate?("", nil)
            return
        }
        onScreenBasisUpdate?(snapshot.appName, snapshot.windowTitle)
    }

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
        max(0.3, AppSettings.shared.captureInterval)
    }

    private func newTraceID(prefix: String) -> String {
        let token = UUID().uuidString.replacingOccurrences(of: "-", with: "").lowercased()
        return "\(prefix)_\(token)"
    }

    private var isSpeechActive: Bool {
        let audioPlaying = audioPlayer?.isPlaying ?? false
        let modelTurnActive = gatewayClient.isModelTurnActive
        let recentlyStopped = Date().timeIntervalSince(gatewayClient.lastModelTurnEndTime) < postSpeechCooldown
        return audioPlaying || modelTurnActive || recentlyStopped
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
            if snapshot.targetKind == .displayFallback {
                NSLog("[CAPTURE] ignoring display fallback snapshot for proactive analysis app=%@ display=%@", snapshot.appName, snapshot.displayID)
                return
            }
            NSLog("[CAPTURE] captured: %dx%d target=%@ app=%@ window=%@",
                  snapshot.image.width,
                  snapshot.image.height,
                  snapshot.targetKind.rawValue,
                  snapshot.appName,
                  snapshot.windowTitle ?? "")

            cacheSnapshotForCommandContext(snapshot)

            let now = Date()
            let speechActive = isSpeechActive
            let smartPathReady = !speechActive && (forceSmartPath || now.timeIntervalSince(lastSmartPathSend) >= smartPathCooldown)
            let traceID = smartPathReady ? newTraceID(prefix: forceSmartPath ? "force" : "cap") : nil
            if let traceID {
                NSLog("[TRACE] flow=proactive trace=%@ phase=capture_ready force=%d target=%@ app=%@ window=%@",
                      traceID,
                      forceSmartPath,
                      snapshot.targetKind.rawValue,
                      snapshot.appName,
                      snapshot.windowTitle ?? "")
            }

            if now.timeIntervalSince(lastFastPathSend) >= fastPathCooldown {
                sendFastPath(image: snapshot.image, traceID: traceID)
                lastFastPathSend = now
            }

            if smartPathReady, let traceID {
                await sendSmartPath(snapshot: snapshot, highSignificance: forceSmartPath, traceID: traceID)
                lastSmartPathSend = now
            }
        }
    }

    func forceAnalysis() async {
        guard gatewayClient.isConnected else { return }
        let result = await captureService.forceCapture()
        if case .captured(let snapshot) = result {
            if snapshot.targetKind == .displayFallback {
                NSLog("[CAPTURE] forceAnalysis skipped display fallback app=%@ display=%@", snapshot.appName, snapshot.displayID)
                return
            }
            cacheSnapshotForCommandContext(snapshot)
            let traceID = newTraceID(prefix: "force")
            NSLog("[TRACE] flow=proactive trace=%@ phase=force_capture_ready target=%@ app=%@ window=%@",
                  traceID,
                  snapshot.targetKind.rawValue,
                  snapshot.appName,
                  snapshot.windowTitle ?? "")
            sendFastPath(image: snapshot.image, traceID: traceID)
            lastFastPathSend = Date()
            if !isSpeechActive {
                await sendSmartPath(snapshot: snapshot, highSignificance: true, traceID: traceID)
                lastSmartPathSend = Date()
            }
        }
    }

    // MARK: - Fast Path (video frame → Gemini Live API)

    private func sendFastPath(image: CGImage, traceID: String?) {
        let img = image
        let client = gatewayClient
        DispatchQueue.global(qos: .userInitiated).async {
            let startedAt = Date()
            guard let jpegData = ImageProcessor.toFastPathJPEG(img) else { return }
            NSLog("[CAPTURE] Fast Path: sending %d bytes JPEG to Live API", jpegData.count)
            if let traceID {
                let elapsedMs = Int(Date().timeIntervalSince(startedAt) * 1000)
                NSLog("[TRACE] flow=proactive trace=%@ phase=fast_path_encoded elapsed_ms=%d bytes=%d",
                      traceID, elapsedMs, jpegData.count)
            }
            DispatchQueue.main.async { client.sendVideoFrame(jpegData) }
        }
    }

    // MARK: - Smart Path (base64 image → ADK orchestrator)

    private func sendSmartPath(snapshot: ScreenCaptureService.CaptureSnapshot, highSignificance: Bool, traceID: String) async {
        guard gatewayClient.isConnected else { return }

        let character = AppSettings.shared.character
        let soul = spriteAnimator.loadPreset(for: character).soul

        let img = snapshot.image
        let encodeStart = Date()
        let base64: String? = await withCheckedContinuation { continuation in
            DispatchQueue.global(qos: .userInitiated).async {
                let result = ImageProcessor.toBase64JPEG(img)
                continuation.resume(returning: result)
            }
        }
        guard let base64 else { return }
        let encodeElapsedMs = Int(Date().timeIntervalSince(encodeStart) * 1000)

        NSLog("[CAPTURE] Smart Path: base64=%d bytes, character=%@, context=%@",
              base64.count, character, snapshot.contextDescription)
        NSLog("[TRACE] flow=proactive trace=%@ phase=smart_path_encoded elapsed_ms=%d base64_bytes=%d high_significance=%d",
              traceID, encodeElapsedMs, base64.count, highSignificance)

        if highSignificance {
            gatewayClient.sendForceCapture(
                imageBase64: base64,
                context: snapshot.contextDescription,
                userId: userId,
                character: character,
                soul: soul,
                activityMinutes: activityMinutes,
                traceID: traceID
            )
        } else {
            gatewayClient.sendScreenCapture(
                imageBase64: base64,
                context: snapshot.contextDescription,
                userId: userId,
                character: character,
                soul: soul,
                activityMinutes: activityMinutes,
                traceID: traceID
            )
        }

        NSLog("[TRACE] flow=proactive trace=%@ phase=smart_path_sent high_significance=%d", traceID, highSignificance)

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

    func latestNavigatorCommandContext(baseContext: NavigatorContextPayload) -> NavigatorContextPayload {
        latestSharedScreenBasisContext(baseContext: baseContext, includeScreenshot: true)
    }

    func latestSharedScreenBasisContext(baseContext: NavigatorContextPayload, includeScreenshot: Bool) -> NavigatorContextPayload {
        guard let cachedDisplayContext else {
            return baseContext
        }

        let ageMs = max(0, Int(Date().timeIntervalSince(cachedDisplayContext.capturedAt) * 1000))
        guard Double(ageMs) <= maxCachedCommandContextAge * 1000 else {
            return baseContext
        }

        return baseContext.withScreenBasis(
            screenBasisID: cachedDisplayContext.screenBasisID,
            activeDisplayID: cachedDisplayContext.activeDisplayID,
            targetDisplayID: cachedDisplayContext.targetDisplayID,
            screenshotAgeMs: ageMs,
            screenshotSource: cachedDisplayContext.source,
            screenshotCached: true,
            screenshot: includeScreenshot ? cachedDisplayContext.screenshotBase64 : nil
        )
    }

    func freshNavigatorCommandContext(baseContext: NavigatorContextPayload) async -> NavigatorContextPayload {
        let result = await captureService.forceCapture()
        guard case .captured(let snapshot) = result,
              snapshot.targetKind != .displayFallback,
              let screenshotBase64 = ImageProcessor.toBase64JPEG(snapshot.image) else {
            return latestNavigatorCommandContext(baseContext: baseContext)
        }

        let now = Date()
        cachedDisplayContext = CachedDisplayContext(
            screenBasisID: snapshot.screenBasisID,
            screenshotBase64: screenshotBase64,
            activeDisplayID: snapshot.displayID,
            targetDisplayID: snapshot.displayID,
            capturedAt: now,
            source: "command_force_capture"
        )

        return baseContext.withScreenBasis(
            screenBasisID: snapshot.screenBasisID,
            activeDisplayID: snapshot.displayID,
            targetDisplayID: snapshot.displayID,
            screenshotAgeMs: 0,
            screenshotSource: "command_force_capture",
            screenshotCached: false,
            screenshot: screenshotBase64
        )
    }

    private func cacheSnapshotForCommandContext(_ snapshot: ScreenCaptureService.CaptureSnapshot) {
        let image = snapshot.image
        let displayID = snapshot.displayID
        let capturedAt = snapshot.capturedAt
        DispatchQueue.global(qos: .utility).async { [weak self] in
            guard let self,
                  let screenshotBase64 = ImageProcessor.toBase64JPEG(image) else {
                return
            }

            DispatchQueue.main.async {
                self.cachedDisplayContext = CachedDisplayContext(
                    screenBasisID: snapshot.screenBasisID,
                    screenshotBase64: screenshotBase64,
                    activeDisplayID: displayID,
                    targetDisplayID: displayID,
                    capturedAt: capturedAt,
                    source: "display_context_cache"
                )
                self.onScreenBasisUpdate?(snapshot.appName, snapshot.windowTitle)
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
