import AppKit
@preconcurrency import AVFoundation
import UserNotifications
import VibeCatCore

@MainActor
final class AppDelegate: NSObject, NSApplicationDelegate {
    private enum ChatActivationSource {
        case hotkey
        case wakeWord
    }

    private var statusBarController: StatusBarController?
    private var trayAnimator: TrayIconAnimator?
    private var onboardingController: OnboardingWindowController?
    private var catPanel: CatPanel?
    private var companionChatPanel: CompanionChatPanel?
    private var targetHighlightOverlay: TargetHighlightOverlay?

    private var audioPlayer: AudioPlayer?
    private var catVoice: CatVoice?
    private var gatewayClient: GatewayClient?
    private var captureService: ScreenCaptureService?
    private var screenAnalyzer: ScreenAnalyzer?
    private var spriteAnimator: SpriteAnimator?
    private var catViewModel: CatViewModel?
    private var backgroundMusicPlayer: BackgroundMusicPlayer?
    private var speechRecognizer: SpeechRecognizer?
    private var audioDeviceMonitor: AudioDeviceMonitor?
    private var circleGestureDetector: CircleGestureDetector?
    private var accessibilityNavigator: AccessibilityNavigator?
    private var navigatorActionWorker: NavigatorActionWorker?
    private var recentSpeechStore = RecentSpeechStore()
    private var emotionTransitionStore = EmotionTransitionStore()
    private var globalHotkeyMonitor: Any?
    private var localHotkeyMonitor: Any?
    private let audioConversionQueue = DispatchQueue(label: "vibecat.audio.conversion")
    nonisolated(unsafe) private var audioConverter: AVAudioConverter?
    private var chatModeActive = false
    private let wakeWords = ["vibecat", "vibe cat", "바이브캣"]

    private var isPaused = false
    private var isMuted = false
    private var assistantTranscription = AssistantTranscriptionAssembler()
    private var userInputTranscription = AssistantTranscriptionAssembler(mergeWindow: 0.9)
    private var pendingTranscription: String { assistantTranscription.currentText }
    private var lastSpeechEndTime: Date = .distantPast
    private let minimumSpeechGap: TimeInterval = 3.0

    private enum SpeechSource {
        case live
        case tts
    }

    private enum SpeechState: CustomStringConvertible {
        case idle
        case modelSpeaking(SpeechSource)
        case cooldown

        var isSpeaking: Bool {
            if case .modelSpeaking = self { return true }
            return false
        }

        var isCooldown: Bool {
            if case .cooldown = self { return true }
            return false
        }

        var source: SpeechSource? {
            if case .modelSpeaking(let s) = self { return s }
            return nil
        }

        var description: String {
            switch self {
            case .idle: return "idle"
            case .modelSpeaking(let s):
                switch s {
                case .live: return "modelSpeaking(live)"
                case .tts: return "modelSpeaking(tts)"
                }
            case .cooldown: return "cooldown"
            }
        }
    }
    private var speechState: SpeechState = .idle
    private var isTurnActive: Bool { speechState.isSpeaking }
    private var ttsActive: Bool { speechState.isSpeaking }
    private var bubbleLockedByTTS = false
    private var cooldownTask: Task<Void, Never>?
    private var spriteIdleTask: Task<Void, Never>?
    private var transcriptionFinalizeTask: Task<Void, Never>?
    private var listeningStatusArmed = false
    private var activeTraceContext: (flow: String, traceId: String)?
    private var traceReadyForClear = false
    private var activeProcessingTraceID: String?
    private var pendingBubbleMeta: String?
    private var activeNavigatorCommand: String?
    private var queuedVoiceNavigatorCommand: String?
    private var queuedVoiceNavigatorCommandTime: Date?
    private let voiceNavigatorQueueTimeout: TimeInterval = 8.0
    private var latestAudioDeviceSnapshot: AudioDeviceMonitor.Snapshot?
    private var audioDeviceChangeTask: Task<Void, Never>?
    private var speechCaptureState: SpeechRecognizer.CaptureState = .stopped
    private let voiceNavigatorSuppressionWindow: TimeInterval = 25

    private enum VoiceNavigatorInterceptionState {
        case idle
        case pending(command: String, deadline: Date)
        case suppressing(command: String, deadline: Date)
    }

    private enum NavigatorPromptState {
        case clarification(command: String, responseMode: NavigatorClarificationResponseMode)
        case risky(command: String)
    }

    private var navigatorPromptState: NavigatorPromptState?
    private var voiceNavigatorInterceptionState: VoiceNavigatorInterceptionState = .idle

    private func activeTraceLogContext() -> String? {
        guard let activeTraceContext else { return nil }
        return "flow=\(activeTraceContext.flow) trace=\(activeTraceContext.traceId)"
    }

    private func clearTraceIfReady() {
        guard traceReadyForClear else { return }
        activeTraceContext = nil
        traceReadyForClear = false
    }

    private func pruneVoiceNavigatorInterceptionIfExpired() {
        switch voiceNavigatorInterceptionState {
        case .idle:
            break
        case .pending(_, let deadline), .suppressing(_, let deadline):
            guard deadline <= Date() else { return }
            clearVoiceNavigatorInterception(reason: "expired")
        }
    }

    private func armVoiceNavigatorInterception(command: String) {
        let trimmed = command.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return }
        voiceNavigatorInterceptionState = .pending(
            command: trimmed,
            deadline: Date().addingTimeInterval(voiceNavigatorSuppressionWindow)
        )
        NSLog("[NAV-VOICE] intercept pending command=%@", trimmed)
    }

    private func queueVoiceNavigatorCommand(_ command: String) {
        let trimmed = command.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return }
        queuedVoiceNavigatorCommand = trimmed
        queuedVoiceNavigatorCommandTime = Date()
        NSLog("[NAV-VOICE] queued command=%@", trimmed)
    }

    private func flushQueuedVoiceNavigatorCommandIfPossible(reason: String) {
        guard let command = queuedVoiceNavigatorCommand else { return }
        if let queueTime = queuedVoiceNavigatorCommandTime,
           Date().timeIntervalSince(queueTime) > voiceNavigatorQueueTimeout {
            NSLog("[NAV-VOICE] queue expired command=%@ age=%.1fs", command,
                  Date().timeIntervalSince(queueTime))
            queuedVoiceNavigatorCommand = nil
            queuedVoiceNavigatorCommandTime = nil
            return
        }
        guard !speechState.isSpeaking, !speechState.isCooldown else {
            NSLog("[NAV-VOICE] queue held reason=%@ speech=%@", reason, speechState.description)
            return
        }
        if case .idle = voiceNavigatorInterceptionState {
        } else {
            NSLog("[NAV-VOICE] queue held reason=%@ intercept_state=pending", reason)
            return
        }
        queuedVoiceNavigatorCommand = nil
        queuedVoiceNavigatorCommandTime = nil
        NSLog("[NAV-VOICE] queue flush reason=%@ command=%@", reason, command)
        handleNavigatorTextSubmission(command, captureScreenshot: true)
    }

    @discardableResult
    private func shouldSuppressForVoiceNavigator(trigger: String) -> Bool {
        pruneVoiceNavigatorInterceptionIfExpired()
        switch voiceNavigatorInterceptionState {
        case .idle:
            return false
        case .pending(let command, let deadline):
            voiceNavigatorInterceptionState = .suppressing(command: command, deadline: deadline)
            NSLog("[NAV-VOICE] suppressing live turn trigger=%@ command=%@", trigger, command)
            return true
        case .suppressing(let command, _):
            if trigger != "audio" {
                NSLog("[NAV-VOICE] dropping live turn trigger=%@ command=%@", trigger, command)
            }
            return true
        }
    }

    private func clearVoiceNavigatorInterception(reason: String) {
        switch voiceNavigatorInterceptionState {
        case .idle:
            break
        case .pending(let command, _), .suppressing(let command, _):
            NSLog("[NAV-VOICE] intercept cleared reason=%@ command=%@", reason, command)
            voiceNavigatorInterceptionState = .idle
        }
    }

    private func shouldRouteVoiceTranscriptToNavigator(_ text: String) -> Bool {
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            return false
        }
        if navigatorPromptState != nil {
            NSLog("[NAV-VOICE] route prompt reply chatModeActive=%d text=%@", chatModeActive, trimmed)
            return true
        }
        if chatModeActive {
            NSLog("[NAV-VOICE] route chat mode command text=%@", trimmed)
            return true
        }
        let context = currentNavigatorContext()
        let shouldRoute = NavigatorVoiceCommandDetector.shouldRoute(
            trimmed,
            context: context,
            hasPendingNavigatorPrompt: navigatorPromptState != nil
        )
        NSLog(
            "[NAV-VOICE] classify chatModeActive=%d role=%@ visibleInputs=%d shouldRoute=%d text=%@",
            chatModeActive,
            context.focusedRole,
            context.visibleInputCandidateCount,
            shouldRoute,
            trimmed
        )
        return shouldRoute
    }

    private func rerouteVoiceTranscriptToNavigator(_ text: String, panel: CatPanel?) {
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return }
        armVoiceNavigatorInterception(command: trimmed)
        queueVoiceNavigatorCommand(trimmed)
        gatewayClient?.sendBargeIn()
        catVoice?.stop()
        if speechState.isSpeaking || speechState.isCooldown {
            transitionSpeech(to: .idle)
        }
        activeProcessingTraceID = nil
        clearPendingBubbleMeta()
        discardPendingTranscription(reason: "voice_navigator_reroute")
        panel?.hideBubble()
    }

    private func normalizeToolName(_ raw: String) -> String {
        VibeCatL10n.toolDisplayName(raw)
    }

    private func makeBubbleMeta(tool: String, sourceCount: Int?) -> String? {
        let normalizedTool = normalizeToolName(tool).trimmingCharacters(in: .whitespacesAndNewlines)
        let parts = [
            normalizedTool.isEmpty ? nil : normalizedTool,
            sourceCount.map { VibeCatL10n.sourceCount($0) }
        ].compactMap { $0 }
        guard !parts.isEmpty else { return nil }
        return parts.joined(separator: " · ")
    }

    private func updatePendingBubbleMeta(tool: String, sourceCount: Int?) {
        if let meta = makeBubbleMeta(tool: tool, sourceCount: sourceCount) {
            pendingBubbleMeta = meta
        }
    }

    private func clearPendingBubbleMeta() {
        pendingBubbleMeta = nil
    }

    private func statusBubbleSuppressionReason() -> String? {
        if chatModeActive {
            return "chat_mode"
        }
        if !pendingTranscription.isEmpty {
            return "assistant_transcription_pending"
        }
        if speechState.isSpeaking {
            return "assistant_speaking"
        }
        return nil
    }

    private func canShowStatusBubble() -> Bool {
        statusBubbleSuppressionReason() == nil
    }

    private func showStatusBubbleIfAllowed(panel: CatPanel?, text: String, detail: String?, context: String = "status") {
        guard let panel else {
            NSLog("[BUBBLE] status skipped (%@): panel_missing text=%@ detail=%@", context, text, detail ?? "")
            return
        }
        if let reason = statusBubbleSuppressionReason() {
            NSLog("[BUBBLE] status suppressed (%@): reason=%@ text=%@ detail=%@", context, reason, text, detail ?? "")
            return
        }
        NSLog("[BUBBLE] status show (%@): text=%@ detail=%@", context, text, detail ?? "")
        panel.showStatusBubble(text: text, detail: detail)
    }

    private func listeningStatusDetail(for text: String) -> String {
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? VibeCatL10n.listeningDetail() : trimmed
    }

    private func armListeningStatus(panel: CatPanel?) {
        listeningStatusArmed = true
        showStatusBubbleIfAllowed(
            panel: panel,
            text: VibeCatL10n.listeningTitle(),
            detail: VibeCatL10n.listeningDetail(),
            context: "listening_arm"
        )
    }

    private func disarmListeningStatus() {
        listeningStatusArmed = false
    }

    private func commitAssistantTranscription(_ text: String, reason: String) {
        recentSpeechStore.add(text, speaker: .assistant)
        statusBarController?.recordInteraction()
        NSLog("[GW-IN] transcription FINALIZED (%@): %@", reason, String(text.prefix(80)))
    }

    private func finalizePendingTranscriptionIfDue(reason: String) {
        if let finalized = assistantTranscription.finalizeIfDue() {
            commitAssistantTranscription(finalized, reason: reason)
        }
        if !assistantTranscription.hasPendingFinalization {
            transcriptionFinalizeTask = nil
        }
    }

    private func finalizePendingTranscriptionNow(reason: String) {
        transcriptionFinalizeTask?.cancel()
        transcriptionFinalizeTask = nil
        if let finalized = assistantTranscription.finalizeNow() {
            commitAssistantTranscription(finalized, reason: reason)
        }
    }

    private func discardPendingTranscription(reason: String) {
        transcriptionFinalizeTask?.cancel()
        transcriptionFinalizeTask = nil
        assistantTranscription.discard()
        NSLog("[GW-IN] transcription DISCARDED (%@)", reason)
    }

    private func appendUserInputTranscription(_ text: String) -> String {
        AssistantTranscriptionAssembler.displayText(userInputTranscription.ingest(text))
    }

    private func finalizeUserInputTranscription(fallback text: String) -> String {
        let finalized = userInputTranscription.finalizeNow()
        let resolved = finalized ?? text
        return AssistantTranscriptionAssembler.displayText(resolved)
    }

    private func discardUserInputTranscription() {
        userInputTranscription.discard()
    }

    private func schedulePendingTranscriptionFinalization() {
        transcriptionFinalizeTask?.cancel()
        guard let delay = assistantTranscription.remainingFinalizationDelay(),
              let deadline = assistantTranscription.scheduledFinalizationDeadline else {
            transcriptionFinalizeTask = nil
            return
        }
        guard delay > 0 else {
            finalizePendingTranscriptionIfDue(reason: "merge_window_elapsed")
            return
        }

        let delayNanoseconds = UInt64((delay * 1_000_000_000).rounded(.up))
        transcriptionFinalizeTask = Task { @MainActor [weak self] in
            try? await Task.sleep(nanoseconds: delayNanoseconds)
            guard let self, !Task.isCancelled else { return }
            guard self.assistantTranscription.scheduledFinalizationDeadline == deadline else { return }
            self.transcriptionFinalizeTask = nil
            self.finalizePendingTranscriptionIfDue(reason: "merge_window_elapsed")
        }
    }

    private func appendPendingTranscription(_ text: String) -> String {
        let combined = assistantTranscription.ingest(text)
        if assistantTranscription.hasPendingFinalization {
            schedulePendingTranscriptionFinalization()
        }
        return combined
    }

    private func markPendingTranscriptionBoundary(reason: String) {
        guard assistantTranscription.markBoundary() else { return }
        NSLog("[GW-IN] transcription BOUNDARY (%@): %@", reason, String(pendingTranscription.prefix(80)))
        schedulePendingTranscriptionFinalization()
    }

    private func resolveProcessingStatePresentation(
        stage: String,
        label: String,
        detail: String,
        tool: String,
        sourceCount: Int?
    ) -> (label: String, detail: String?) {
        let resolvedLabel = label.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            ? VibeCatL10n.processingStateLabel(stage: stage, tool: tool)
            : label
        let resolvedDetail = detail.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            ? (VibeCatL10n.processingStateDetail(stage: stage, tool: tool, sourceCount: sourceCount)
                ?? makeBubbleMeta(tool: tool, sourceCount: sourceCount))
            : detail
        return (resolvedLabel, resolvedDetail)
    }

    private func updateAudioConverter(for inputFormat: AVAudioFormat?) {
        guard let inputFormat,
              let output = AVAudioFormat(commonFormat: .pcmFormatInt16, sampleRate: 16000, channels: 1, interleaved: true) else {
            audioConverter = nil
            NSLog("[AUDIO-PIPE] audio converter cleared")
            return
        }
        audioConverter = AVAudioConverter(from: inputFormat, to: output)
        NSLog("[AUDIO-PIPE] audio converter configured: in=%@ out=%@", String(describing: inputFormat), String(describing: output))
    }

    private func audioInputStateLabel() -> String {
        if isPaused {
            return VibeCatL10n.audioInputPaused()
        }
        switch speechCaptureState {
        case .stopped:
            return VibeCatL10n.audioInputStopped()
        case .starting, .recovering(attempt: 1):
            return VibeCatL10n.audioInputRecovering(attempt: 1)
        case .recovering(let attempt):
            return VibeCatL10n.audioInputRecovering(attempt: attempt)
        case .failed:
            return VibeCatL10n.audioInputFailed()
        case .listening:
            return VibeCatL10n.audioInputListening()
        }
    }

    private func refreshAudioInputStatusUI() {
        let inputName = latestAudioDeviceSnapshot?.inputDeviceName ?? VibeCatL10n.audioInputUnknown()
        statusBarController?.updateAudioInputStatus(
            inputName: inputName,
            stateText: audioInputStateLabel()
        )
    }

    private func applyAudioDeviceChange(_ snapshot: AudioDeviceMonitor.Snapshot) {
        latestAudioDeviceSnapshot = snapshot
        refreshAudioInputStatusUI()
        NSLog(
            "[AUDIO-DEVICE] app handling change trigger=%@ input=%@(%u) output=%@(%u)",
            snapshot.trigger.rawValue,
            snapshot.inputDeviceName,
            snapshot.inputDeviceID,
            snapshot.outputDeviceName,
            snapshot.outputDeviceID
        )
        audioPlayer?.handleAudioDeviceChange(reason: snapshot.trigger.rawValue)
        speechRecognizer?.handleAudioDeviceChange(reason: snapshot.trigger.rawValue)
    }

    private func handleAudioDeviceChange(_ snapshot: AudioDeviceMonitor.Snapshot) {
        latestAudioDeviceSnapshot = snapshot
        refreshAudioInputStatusUI()
        audioDeviceChangeTask?.cancel()
        audioDeviceChangeTask = Task { @MainActor [weak self] in
            guard let self, !Task.isCancelled else { return }
            self.audioDeviceChangeTask = nil
            self.applyAudioDeviceChange(snapshot)
        }
    }

    private func transitionSpeech(to newState: SpeechState) {
        let old = speechState
        speechState = newState
        NSLog("[SPEECH] transition: %@ -> %@", old.description, newState.description)
        switch newState {
        case .idle:
            bubbleLockedByTTS = false
            cooldownTask?.cancel()
            spriteIdleTask?.cancel()
            speechRecognizer?.setModelSpeaking(false)
            catPanel?.setTurnActive(false)
            flushQueuedVoiceNavigatorCommandIfPossible(reason: "speech_idle")
            // Barge-in / interrupt paths can jump straight from speaking to idle.
            // Clear transient celebration/emotion UI immediately in that case.
            if old.isSpeaking {
                spriteAnimator?.setState(.idle)
                catPanel?.setEmotionIndicator(nil)
            }
        case .modelSpeaking:
            cooldownTask?.cancel()
            spriteIdleTask?.cancel()
            speechRecognizer?.setModelSpeaking(true)
            catPanel?.setTurnActive(true)
        case .cooldown:
            speechRecognizer?.setModelSpeaking(false)
            catPanel?.setTurnActive(false)
        }
    }

    private func beginSpeechCooldown() {
        lastSpeechEndTime = Date()
        transitionSpeech(to: .cooldown)
        cooldownTask = Task { @MainActor [weak self] in
            try? await Task.sleep(nanoseconds: 1_000_000_000)
            guard let self else { return }
            if case .cooldown = self.speechState {
                self.transitionSpeech(to: .idle)
            }
        }
        spriteIdleTask = Task { @MainActor [weak self] in
            try? await Task.sleep(nanoseconds: 2_000_000_000)
            guard let self else { return }
            if case .idle = self.speechState {
                self.spriteAnimator?.setState(.idle)
            }
        }
    }

    private func refreshCapturePrivacyUI() {
        guard let panel = catPanel else { return }

        let title: String
        let color: NSColor
        if isPaused {
            title = VibeCatL10n.captureIndicatorPaused()
            color = .systemGray
        } else if AppSettings.shared.navigatorModeEnabled {
            title = VibeCatL10n.captureIndicatorNavigator()
            color = .systemBlue
        } else if AppSettings.shared.manualAnalysisOnly {
            title = VibeCatL10n.captureIndicatorManual()
            color = .systemOrange
        } else {
            title = VibeCatL10n.captureIndicatorLive()
            color = .systemGreen
        }

        panel.updateCapturePrivacyBadge(
            title: title,
            detail: VibeCatL10n.captureIndicatorNoStorage(),
            accentColor: color
        )
        statusBarController?.updateCaptureIndicator(
            isPaused: isPaused,
            isManualOnly: AppSettings.shared.manualAnalysisOnly
        )
    }

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.accessory)

        let audio = AudioPlayer()
        let voice = CatVoice(audioPlayer: audio)
        let gateway = GatewayClient()
        let capture = ScreenCaptureService()
        let sprite = SpriteAnimator(character: AppSettings.shared.character)
        let initialPreset = sprite.loadPreset(for: AppSettings.shared.character)
        AppSettings.shared.voice = initialPreset.voice
        let viewModel = CatViewModel()
        let music = BackgroundMusicPlayer()
        let speechRecognizer = SpeechRecognizer()
        let audioDeviceMonitor = AudioDeviceMonitor()
        let circleGestureDetector = CircleGestureDetector()
        let companionChatPanel = CompanionChatPanel()
        let accessibilityNavigator = AccessibilityNavigator()
        let targetHighlightOverlay = TargetHighlightOverlay()

        self.audioPlayer = audio
        self.catVoice = voice
        self.gatewayClient = gateway
        self.captureService = capture
        self.spriteAnimator = sprite
        self.catViewModel = viewModel
        self.backgroundMusicPlayer = music
        self.speechRecognizer = speechRecognizer
        self.audioDeviceMonitor = audioDeviceMonitor
        self.circleGestureDetector = circleGestureDetector
        self.companionChatPanel = companionChatPanel
        self.accessibilityNavigator = accessibilityNavigator
        self.targetHighlightOverlay = targetHighlightOverlay
        self.navigatorActionWorker = NavigatorActionWorker(
            gatewayClient: gateway,
            navigator: accessibilityNavigator,
            contextProvider: { [weak self] in
                self?.currentNavigatorContext()
                    ?? NavigatorContextPayload(
                        appName: "",
                        bundleId: "",
                        frontmostBundleId: "",
                        windowTitle: "",
                        focusedRole: "",
                        focusedLabel: "",
                        selectedText: "",
                        axSnapshot: "",
                        inputFieldHint: "",
                        lastInputFieldDescriptor: "",
                        screenshot: "",
                        focusStableMs: 0,
                        captureConfidence: 0,
                        visibleInputCandidateCount: 0,
                        accessibilityPermission: "unknown",
                        accessibilityTrusted: false
                    )
            }
        )

        let analyzer = ScreenAnalyzer(
            captureService: capture,
            gatewayClient: gateway,
            catVoice: voice,
            spriteAnimator: sprite
        )
        analyzer.setAudioPlayer(audio)
        self.screenAnalyzer = analyzer

        let panel = CatPanel(catViewModel: viewModel, spriteAnimator: sprite)
        self.catPanel = panel
        capture.probePointProvider = { [weak panel] in
            panel?.currentGlobalProbePoint() ?? NSEvent.mouseLocation
        }
        panel.applySpriteSize(presetSize: initialPreset.size)
        panel.setSmartHideReferences(audioPlayer: audio, screenAnalyzer: analyzer)
        panel.traceLogContextProvider = { [weak self] in
            self?.activeTraceLogContext()
        }
        panel.onBubbleDidHide = { [weak self] in
            self?.clearPendingBubbleMeta()
            self?.clearTraceIfReady()
        }
        panel.show()
        refreshCapturePrivacyUI()
        NSLog("[AppDelegate] CatPanel shown. frame=%@ level=%d isVisible=%d", NSStringFromRect(panel.frame), panel.level.rawValue, panel.isVisible ? 1 : 0)

        gateway.setSoul(initialPreset.soul)

        circleGestureDetector.onCircleGesture = { [weak self] in
            self?.showStatusBubbleIfAllowed(
                panel: self?.catPanel,
                text: VibeCatL10n.screenReadingTitle(),
                detail: VibeCatL10n.screenReadingDetail(),
                context: "screen_capture_gesture"
            )
            Task { @MainActor [weak self] in
                await self?.screenAnalyzer?.forceAnalysis()
            }
        }
        circleGestureDetector.startMonitoring()

        audioDeviceMonitor.onChange = { [weak self] snapshot in
            Task { @MainActor [weak self] in
                self?.handleAudioDeviceChange(snapshot)
            }
        }
        audioDeviceMonitor.start()
        latestAudioDeviceSnapshot = audioDeviceMonitor.latestSnapshot ?? audioDeviceMonitor.currentSnapshot()

        companionChatPanel.onTextSubmitted = { [weak self] text in
            guard let self else { return }
            self.chatModeActive = true
            self.clearPendingBubbleMeta()
            self.companionChatPanel?.addUserMessage(text)
            self.companionChatPanel?.addLoadingPlaceholder()
            self.handleNavigatorTextSubmission(text)
            self.statusBarController?.recordInteraction()
        }
        companionChatPanel.onDismissed = { [weak self] in
            self?.deactivateChatMode()
        }

        analyzer.onSpeechEvent = { [weak self, weak panel] event in
            panel?.showSpeechBubble(text: event.text, meta: self?.pendingBubbleMeta)
            self?.recentSpeechStore.add(event.text, speaker: .assistant)
            self?.statusBarController?.recordInteraction()
        }
        analyzer.onScreenBasisUpdate = { [weak panel] appName, windowTitle in
            Task { @MainActor in
                panel?.updateCurrentWindowTitle(appName: appName, windowTitle: windowTitle)
            }
        }
        analyzer.onBackgroundSpeech = { [weak self] text in
            self?.sendCompanionNotification(text: text)
        }

        let sbc = StatusBarController()
        let tray = TrayIconAnimator()
        sbc.attachAnimator(tray)
        sbc.attachRecentSpeechStore(recentSpeechStore)
        sbc.attachEmotionTransitionStore(emotionTransitionStore)
        sbc.onQuit = { NSApp.terminate(nil) }
        sbc.onReconnect = { [weak self] in self?.handleReconnect() }
        sbc.onAnalyzeNow = { [weak self] in self?.handleAnalyzeNow() }
        sbc.onPause = { [weak self] in self?.handlePause() }
        sbc.onMute = { [weak self] in self?.handleMute() }
        sbc.onShowOnboarding = { [weak self] in self?.showOnboarding() }
        sbc.onCharacterChanged = { [weak self] character in
            self?.handleCharacterChanged(character)
        }
        sbc.onMusicToggled = { [weak self] enabled in
            self?.backgroundMusicPlayer?.setEnabled(enabled)
        }
        sbc.onSearchToggled = { [weak self] in
            self?.gatewayClient?.resendSetupPayloadIfConnected()
        }
        sbc.onProactiveAudioToggled = { [weak self] in
            self?.gatewayClient?.resendSetupPayloadIfConnected()
        }
        sbc.onNavigatorModeToggled = { [weak self] _ in
            self?.handleNavigatorModeToggle()
        }
        sbc.onManualAnalysisToggled = { [weak self] _ in
            self?.handleManualAnalysisToggle()
        }
        sbc.onLanguageChanged = { [weak self] in
            self?.gatewayClient?.resendSetupPayloadIfConnected()
            self?.onboardingController?.refreshLocalizedText()
            self?.companionChatPanel?.refreshLocalizedText()
            self?.refreshCapturePrivacyUI()
            self?.refreshAudioInputStatusUI()
        }
        self.statusBarController = sbc
        self.trayAnimator = tray
        refreshAudioInputStatusUI()

        sprite.onStateTransition = { [weak self, weak panel] from, to in
            self?.emotionTransitionStore.add(from: from, to: to)
            panel?.updateEmotionForState(to)
        }

        wireGatewayCallbacks(
            gateway: gateway,
            voice: voice,
            analyzer: analyzer,
            panel: panel,
            chatPanel: companionChatPanel,
            statusBarController: sbc
        )

        setupNotifications()
        registerGlobalHotkey()
        wireSpeechCapture(speechRecognizer: speechRecognizer, gateway: gateway)
        ErrorReporter.shared.onErrorChanged = { [weak sbc] message in
            sbc?.setLastErrorDescription(message)
        }

        gateway.connect()
        analyzer.start()
        sbc.updatePauseState(isPaused)
        sbc.updateManualAnalysisMode(AppSettings.shared.manualAnalysisOnly)
    }

    func applicationWillTerminate(_ notification: Notification) {
        screenAnalyzer?.pause()
        gatewayClient?.disconnect()
        speechRecognizer?.stopListening()
        audioDeviceMonitor?.stop()
        audioDeviceChangeTask?.cancel()
        circleGestureDetector?.stopMonitoring()
        trayAnimator?.stop()
        spriteAnimator?.stop()
        if let globalHotkeyMonitor {
            NSEvent.removeMonitor(globalHotkeyMonitor)
            self.globalHotkeyMonitor = nil
        }
        if let localHotkeyMonitor {
            NSEvent.removeMonitor(localHotkeyMonitor)
            self.localHotkeyMonitor = nil
        }
    }

    private func wireGatewayCallbacks(
        gateway: GatewayClient,
        voice: CatVoice,
        analyzer: ScreenAnalyzer,
        panel: CatPanel,
        chatPanel: CompanionChatPanel,
        statusBarController: StatusBarController
    ) {
        gateway.onAudioData = { [weak self] data in
            guard let self else { return }
            if self.shouldSuppressForVoiceNavigator(trigger: "audio") {
                return
            }
            if !self.speechState.isSpeaking && !self.speechState.isCooldown {
                self.transitionSpeech(to: .modelSpeaking(.live))
            }
            voice.enqueueAudio(data)
        }
        gateway.onStateChange = { [weak statusBarController] state in
            switch state {
            case .connected:
                voice.setLiveConnected(true)
                statusBarController?.updateConnectionStatus(.connected)
                statusBarController?.setLastErrorDescription(nil)
                statusBarController?.setAPIKeyNeedsAttention(false)
                ErrorReporter.shared.clearError()
            case .disconnected, .failed:
                voice.setLiveConnected(false)
                statusBarController?.updateConnectionStatus(.disconnected)
                statusBarController?.setLastErrorDescription(gateway.lastErrorDescription)
                let attention = gateway.lastErrorDescription != nil
                statusBarController?.setAPIKeyNeedsAttention(attention)
                if let error = gateway.lastErrorDescription {
                    ErrorReporter.shared.report(error, context: "Gateway")
                }
            default:
                break
            }
        }
        gateway.onReconnecting = { [weak statusBarController] attempt in
            statusBarController?.updateConnectionStatus(.reconnecting(attempt: attempt, max: 30))
            statusBarController?.setLastErrorDescription(gateway.lastErrorDescription)
        }
        gateway.onDisconnected = { [weak self, weak statusBarController, weak panel] in
            statusBarController?.updateConnectionStatus(.disconnected)
            statusBarController?.setLastErrorDescription(gateway.lastErrorDescription)
            if let error = gateway.lastErrorDescription {
                ErrorReporter.shared.report(error, context: "Gateway")
            }
            if let self {
                self.transitionSpeech(to: .idle)
                self.disarmListeningStatus()
                self.discardPendingTranscription(reason: "gateway_disconnected")
                self.discardUserInputTranscription()
            }
            panel?.hideBubble()
            panel?.setEmotionIndicator(nil)
        }
        gateway.onReconnectExhausted = { [weak statusBarController] in
            statusBarController?.updateConnectionStatus(.disconnected)
            statusBarController?.setLastErrorDescription(gateway.lastErrorDescription)
            statusBarController?.setAPIKeyNeedsAttention(true)
            if let error = gateway.lastErrorDescription {
                ErrorReporter.shared.report(error, context: "Gateway")
            }
        }
        gateway.onLatencyUpdate = { [weak statusBarController] latency in
            statusBarController?.updateLatency(latency)
        }
        gateway.onMessage = { [weak self, weak analyzer, weak panel, weak chatPanel] message in
            guard let self else { return }
            switch message {
            case .companionSpeech(let text, let emotionStr, let urgency):
                NSLog("[GW-IN] onMessage: companionSpeech(legacy), textLength=%lu, emotion=%@, chatModeActive=%d, state=%@", text.count, emotionStr, self.chatModeActive, self.speechState.description)
                self.disarmListeningStatus()
                if self.speechState.isSpeaking || self.speechState.isCooldown {
                    NSLog("[GW-IN] companionSpeech DROPPED: speech active (state=%@)", self.speechState.description)
                    return
                }
                let timeSinceLastSpeech = Date().timeIntervalSince(self.lastSpeechEndTime)
                if timeSinceLastSpeech < self.minimumSpeechGap {
                    NSLog("[GW-IN] companionSpeech DROPPED: too soon (%.1fs < %.1fs gap)", timeSinceLastSpeech, self.minimumSpeechGap)
                    return
                }
                let emotion = CompanionEmotion(rawValue: emotionStr) ?? .neutral
                if self.chatModeActive {
                    chatPanel?.updateLastAssistantMessage(text)
                } else {
                    let event = CompanionSpeechEvent(text: text, emotion: emotion, urgency: urgency)
                    analyzer?.handleCompanionSpeech(event)
                }
            case .transcription(let text, let finished):
                if self.shouldSuppressForVoiceNavigator(trigger: "transcription") {
                    return
                }
                NSLog("[GW-IN] onMessage: transcription, textLength=%lu, finished=%d, chatModeActive=%d, ttsActive=%d, bubbleLocked=%d", text.count, finished, self.chatModeActive, self.ttsActive, self.bubbleLockedByTTS)
                self.disarmListeningStatus()
                if self.chatModeActive {
                    if finished {
                        chatPanel?.finalizeListeningText(text)
                    } else {
                        chatPanel?.updateListeningText(text)
                    }
                } else {
                    var displayText = text
                    if let parsed = AudioMessageParser.parseEmotionTag(from: text) {
                        displayText = parsed.cleanText
                        analyzer?.handleCompanionSpeechEmotion(parsed.emotion)
                    }
                    if !displayText.isEmpty {
                        let combined = self.appendPendingTranscription(displayText)
                        if !self.bubbleLockedByTTS {
                            panel?.showSpeechBubble(text: combined, meta: self.pendingBubbleMeta)
                        }
                    }
                    if finished {
                        self.markPendingTranscriptionBoundary(reason: "output_transcription_finished")
                        NSLog("[GW-IN] transcription COMPLETE marker: %@", String(self.pendingTranscription.prefix(80)))
                        self.maybeActivateChatFromWakeWord(self.pendingTranscription.isEmpty ? displayText : self.pendingTranscription)
                    }
                }
            case .turnState(let state, let source):
                if source == "live" && self.shouldSuppressForVoiceNavigator(trigger: "turn_state_\(state)") {
                    if state != "speaking" {
                        self.clearVoiceNavigatorInterception(reason: "suppressed_turn_state_\(state)")
                    }
                    return
                }
                NSLog("[GW-IN] onMessage: turnState, state=%@, source=%@, current=%@", state, source, self.speechState.description)
                switch state {
                case "speaking":
                    self.disarmListeningStatus()
                    let speechSource: SpeechSource = source == "tts" ? .tts : .live
                    if !self.speechState.isSpeaking {
                        self.transitionSpeech(to: .modelSpeaking(speechSource))
                    }
                    self.activeProcessingTraceID = nil
                default:
                    if self.speechState.isSpeaking {
                        self.beginSpeechCooldown()
                    }
                }
            case .traceEvent(let flow, let traceId, let phase, let elapsedMs, let detail):
                if flow == "voice" && self.shouldSuppressForVoiceNavigator(trigger: "trace_\(phase)") {
                    if ["turn_complete", "turn_interrupted", "turn_failed"].contains(phase) {
                        self.clearVoiceNavigatorInterception(reason: "suppressed_\(phase)")
                    }
                    return
                }
                NSLog("[TRACE] flow=%@ trace=%@ phase=%@ elapsed_ms=%@ detail=%@",
                      flow, traceId, phase, elapsedMs.map(String.init) ?? "-", detail)
                switch phase {
                case "first_output_text", "turn_started":
                    self.activeTraceContext = (flow: flow, traceId: traceId)
                    self.traceReadyForClear = false
                    if self.activeProcessingTraceID == traceId {
                        self.activeProcessingTraceID = nil
                    }
                case "turn_complete", "turn_interrupted", "turn_failed":
                    if self.activeTraceContext?.traceId == traceId {
                        self.traceReadyForClear = true
                    }
                    if self.activeProcessingTraceID == traceId {
                        self.activeProcessingTraceID = nil
                    }
                default:
                    break
                }
            case .processingState(let flow, let traceId, let stage, let label, let detail, let tool, let sourceCount, let active):
                NSLog("[GW-IN] onMessage: processingState, flow=%@, trace=%@, stage=%@, active=%d, detail=%@, tool=%@, sources=%@",
                      flow, traceId, stage, active ? 1 : 0, detail, tool, sourceCount.map(String.init) ?? "-")
                let presentation = self.resolveProcessingStatePresentation(
                    stage: stage,
                    label: label,
                    detail: detail,
                    tool: tool,
                    sourceCount: sourceCount
                )
                self.updatePendingBubbleMeta(tool: tool, sourceCount: sourceCount)
                if active {
                    self.activeProcessingTraceID = traceId
                    self.showStatusBubbleIfAllowed(
                        panel: panel,
                        text: presentation.label,
                        detail: presentation.detail,
                        context: "processing_\(stage)"
                    )
                } else if self.activeProcessingTraceID == traceId {
                    self.activeProcessingTraceID = nil
                    if self.pendingTranscription.isEmpty {
                        panel?.hideStatusBubbleIfShowing()
                    }
                }
            case .toolResult(let tool, _, _, let sources):
                NSLog("[GW-IN] onMessage: toolResult, tool=%@, sources=%lu", tool, sources.count)
                self.updatePendingBubbleMeta(tool: tool, sourceCount: sources.count)
            case .navigatorCommandAccepted(let taskId, let command, let intentClass, let intentConfidence):
                NSLog("[GW-IN] navigator.commandAccepted task=%@ command=%@ intent=%@ confidence=%.2f", taskId ?? "-", command, intentClass.rawValue, intentConfidence)
                self.activeNavigatorCommand = command
                self.navigatorPromptState = nil
                if let taskId {
                    self.navigatorActionWorker?.beginTask(taskId: taskId, command: command)
                }
                chatPanel?.updateLastAssistantMessage(VibeCatL10n.navigatorAccepted(intent: intentClass, confidence: intentConfidence))
                self.showStatusBubbleIfAllowed(
                    panel: panel,
                    text: VibeCatL10n.navigatorActingTitle(),
                    detail: command,
                    context: "navigator_command_accepted"
                )
            case .navigatorIntentClarificationNeeded(let command, let question, let responseMode):
                NSLog(
                    "[GW-IN] navigator.intentClarificationNeeded command=%@ responseMode=%@",
                    command,
                    responseMode.rawValue
                )
                self.activeNavigatorCommand = command
                self.navigatorPromptState = .clarification(command: command, responseMode: responseMode)
                chatPanel?.updateLastAssistantMessage(question)
                self.showStatusBubbleIfAllowed(
                    panel: panel,
                    text: VibeCatL10n.navigatorClarifyingTitle(),
                    detail: question,
                    context: "navigator_clarification"
                )
            case .navigatorStepPlanned(let taskId, let step, let message):
                NSLog("[GW-IN] navigator.stepPlanned task=%@ id=%@ action=%@", taskId, step.id, step.actionType.rawValue)
                chatPanel?.updateLastAssistantMessage(message.isEmpty ? step.expectedOutcome : message)
                self.showStatusBubbleIfAllowed(
                    panel: panel,
                    text: VibeCatL10n.navigatorActingTitle(),
                    detail: step.expectedOutcome,
                    context: "navigator_step_planned"
                )
                self.executeNavigatorStep(taskId: taskId, step: step, panel: panel, chatPanel: chatPanel)
            case .navigatorStepRunning(let taskId, let stepId, let status):
                NSLog("[GW-IN] navigator.stepRunning task=%@ id=%@ status=%@", taskId, stepId, status)
            case .navigatorStepVerified(_, _, _, let observedOutcome):
                chatPanel?.updateLastAssistantMessage(observedOutcome)
            case .navigatorRiskyActionBlocked(let command, let question, let reason):
                NSLog("[GW-IN] navigator.riskyActionBlocked command=%@ reason=%@", command, reason)
                self.activeNavigatorCommand = command
                self.navigatorPromptState = .risky(command: command)
                chatPanel?.updateLastAssistantMessage(question)
                self.showStatusBubbleIfAllowed(
                    panel: panel,
                    text: VibeCatL10n.navigatorConfirmTitle(),
                    detail: reason,
                    context: "navigator_risk"
                )
            case .navigatorGuidedMode(let taskId, _, let instruction):
                self.navigatorPromptState = nil
                self.activeNavigatorCommand = nil
                if let taskId {
                    self.navigatorActionWorker?.clearTask(taskId: taskId)
                }
                chatPanel?.updateLastAssistantMessage(instruction)
                self.showStatusBubbleIfAllowed(
                    panel: panel,
                    text: VibeCatL10n.navigatorGuidedModeTitle(),
                    detail: instruction,
                    context: "navigator_guided"
                )
            case .navigatorCompleted(let taskId, let summary):
                self.navigatorPromptState = nil
                self.activeNavigatorCommand = nil
                self.navigatorActionWorker?.clearTask(taskId: taskId)
                chatPanel?.updateLastAssistantMessage(summary)
                self.showStatusBubbleIfAllowed(
                    panel: panel,
                    text: VibeCatL10n.navigatorDoneTitle(),
                    detail: summary,
                    context: "navigator_completed"
                )
            case .navigatorFailed(let taskId, let reason):
                self.navigatorPromptState = nil
                self.activeNavigatorCommand = nil
                if let taskId {
                    self.navigatorActionWorker?.clearTask(taskId: taskId)
                }
                chatPanel?.updateLastAssistantMessage(reason)
                self.showStatusBubbleIfAllowed(
                    panel: panel,
                    text: VibeCatL10n.navigatorFailedTitle(),
                    detail: reason,
                    context: "navigator_failed"
                )
            case .inputTranscription(let text, let finished):
                NSLog("[GW-IN] onMessage: inputTranscription, text=%@, finished=%d", String(text.prefix(60)), finished)
                let displayText = self.appendUserInputTranscription(text)
                if !finished,
                   !self.listeningStatusArmed,
                   !self.chatModeActive,
                   !self.speechState.isSpeaking,
                   self.activeProcessingTraceID == nil {
                    self.listeningStatusArmed = true
                    NSLog("[BUBBLE] listening auto-armed from input transcription")
                }
                if !finished && self.listeningStatusArmed {
                    if !self.bubbleLockedByTTS && !self.chatModeActive && !displayText.isEmpty {
                        panel?.showBubble(text: displayText)
                    }
                    self.showStatusBubbleIfAllowed(
                        panel: panel,
                        text: VibeCatL10n.listeningTitle(),
                        detail: self.listeningStatusDetail(for: displayText),
                        context: "input_transcription"
                    )
                }
                if finished {
                    self.disarmListeningStatus()
                    self.activeProcessingTraceID = nil
                    self.clearPendingBubbleMeta()
                    let finalUserText = self.finalizeUserInputTranscription(fallback: text)
                    if !finalUserText.isEmpty {
                        self.recentSpeechStore.add(finalUserText, speaker: .user)
                    }
                    if self.shouldRouteVoiceTranscriptToNavigator(finalUserText) {
                        self.rerouteVoiceTranscriptToNavigator(finalUserText, panel: panel)
                    }
                }
            case .audio(let data):
                NSLog("[GW-IN] onMessage: audio, dataSize=%lu", data.count)
            case .turnComplete:
                if self.shouldSuppressForVoiceNavigator(trigger: "turn_complete") {
                    self.clearVoiceNavigatorInterception(reason: "suppressed_turn_complete")
                    self.flushQueuedVoiceNavigatorCommandIfPossible(reason: "suppressed_turn_complete")
                    return
                }
                NSLog("[GW-IN] onMessage: turnComplete")
                self.catVoice?.flush()
                if self.speechState.isSpeaking {
                    self.beginSpeechCooldown()
                }
                self.activeProcessingTraceID = nil
                self.markPendingTranscriptionBoundary(reason: "turn_complete")
            case .interrupted:
                self.clearVoiceNavigatorInterception(reason: "interrupted")
                NSLog("[GW-IN] onMessage: interrupted")
                self.catVoice?.stop()
                self.transitionSpeech(to: .idle)
                self.activeProcessingTraceID = nil
                self.clearPendingBubbleMeta()
                self.discardPendingTranscription(reason: "interrupted")
                self.discardUserInputTranscription()
                panel?.hideBubble()
                self.armListeningStatus(panel: panel)
                self.flushQueuedVoiceNavigatorCommandIfPossible(reason: "interrupted")
            case .sessionResumptionUpdate(let handle):
                NSLog("[GW-IN] onMessage: sessionResumptionUpdate, handleLength=%lu", handle.count)
            case .liveSessionReconnecting(let attempt, let max):
                NSLog("[GW-IN] onMessage: liveSessionReconnecting, attempt=%d/%d", attempt, max)
                self.statusBarController?.setLastErrorDescription(VibeCatL10n.liveSessionReconnectingStatus(attempt: attempt, max: max))
                self.showStatusBubbleIfAllowed(
                    panel: panel,
                    text: VibeCatL10n.reconnectingTitle(),
                    detail: VibeCatL10n.liveSessionRecoveringDetail(),
                    context: "live_session_reconnecting"
                )
            case .liveSessionReconnected:
                NSLog("[GW-IN] onMessage: liveSessionReconnected")
                self.statusBarController?.setLastErrorDescription(nil)
                panel?.hideStatusBubbleIfShowing()
            case .setupComplete(let sessionId):
                NSLog("[GW-IN] onMessage: setupComplete, sessionId=%@", sessionId)
            case .goAway(let reason, let timeLeftMs):
                NSLog("[GW-IN] onMessage: goAway, reason=%@, timeLeftMs=%d", reason, timeLeftMs)
                self.showStatusBubbleIfAllowed(
                    panel: panel,
                    text: VibeCatL10n.reconnectingTitle(),
                    detail: VibeCatL10n.sessionResumePreparingDetail(),
                    context: "session_resume_prepare"
                )
            case .pong:
                NSLog("[GW-IN] onMessage: pong")
            case .ttsStart(let text):
                NSLog("[GW-IN] onMessage: ttsStart, hasText=%d, state=%@", text != nil ? 1 : 0, self.speechState.description)
                self.disarmListeningStatus()
                let source: SpeechSource = (text != nil && !text!.isEmpty) ? .tts : .live
                self.transitionSpeech(to: .modelSpeaking(source))
                self.catVoice?.stop()
                if source == .tts, let text, !text.isEmpty {
                    self.finalizePendingTranscriptionNow(reason: "tts_start")
                    self.bubbleLockedByTTS = true
                    panel?.showBubble(text: text)
                }
            case .ttsEnd:
                NSLog("[GW-IN] onMessage: ttsEnd")
                guard self.speechState.isSpeaking else {
                    self.discardPendingTranscription(reason: "tts_end_while_idle")
                    return
                }
                self.beginSpeechCooldown()
            case .error(let code, let message):
                NSLog("[GW-IN] onMessage: error, code=%@, message=%@", code, message)
            case .unknown:
                NSLog("[GW-IN] onMessage: unknown")
            }
        }
    }

    private func handleCharacterChanged(_ character: String) {
        catPanel?.beginCharacterTransition()

        spriteAnimator?.setCharacter(character)

        guard let preset = spriteAnimator?.loadPreset(for: character) else {
            catPanel?.endCharacterTransition(characterName: character)
            return
        }
        AppSettings.shared.voice = preset.voice
        catPanel?.applySpriteSize(presetSize: preset.size)
        gatewayClient?.setSoul(preset.soul)
        gatewayClient?.resendSetupPayloadIfConnected()

        let displayName = localizedCharacterName(character)
        Task { @MainActor [weak self] in
            try? await Task.sleep(nanoseconds: 1_200_000_000)
            self?.catPanel?.endCharacterTransition(characterName: displayName)
        }
    }

    private func showOnboarding() {
        let controller = OnboardingWindowController()
        controller.onConnect = { [weak self] in
            guard let self else { return }
            self.onboardingController = nil
            self.gatewayClient?.connect()
            self.screenAnalyzer?.start()
        }
        controller.show()
        self.onboardingController = controller
    }

    private func handleReconnect() {
        gatewayClient?.reconnect()
    }

    private func handleAnalyzeNow() {
        guard !isPaused else { return }
        showStatusBubbleIfAllowed(
            panel: catPanel,
            text: VibeCatL10n.screenReadingTitle(),
            detail: VibeCatL10n.screenReadingDetail(),
            context: "screen_capture_menu"
        )
        Task { @MainActor [weak self] in
            await self?.screenAnalyzer?.forceAnalysis()
        }
    }

    private func handlePause() {
        if isPaused {
            screenAnalyzer?.resume()
            isPaused = false
            refreshAudioInputStatusUI()
            speechRecognizer?.resumeListening()
            circleGestureDetector?.startMonitoring()
        } else {
            screenAnalyzer?.pause()
            speechRecognizer?.stopListening()
            circleGestureDetector?.stopMonitoring()
            isPaused = true
            refreshAudioInputStatusUI()
        }
        statusBarController?.updatePauseState(isPaused)
        refreshCapturePrivacyUI()
    }

    private func handleManualAnalysisToggle() {
        screenAnalyzer?.reloadCapturePolicy()
        refreshCapturePrivacyUI()
    }

    private func handleNavigatorModeToggle() {
        screenAnalyzer?.reloadCapturePolicy()
        gatewayClient?.resendSetupPayloadIfConnected()
        refreshCapturePrivacyUI()
    }

    private func handleMute() {
        isMuted.toggle()
        if isMuted {
            catVoice?.mute()
        } else {
            catVoice?.unmute()
        }
        statusBarController?.updateMuteState(isMuted)
    }

    private func registerGlobalHotkey() {
        let handler: (NSEvent) -> Bool = { [weak self] event in
            guard let self else { return false }
            guard event.type == .keyDown else { return false }
            let flags = event.modifierFlags.intersection(.deviceIndependentFlagsMask)
            guard flags == [.option] else {
                return false
            }

            switch event.charactersIgnoringModifiers?.lowercased() {
            case "v":
                showStatusBubbleIfAllowed(
                    panel: catPanel,
                    text: VibeCatL10n.screenReadingTitle(),
                    detail: VibeCatL10n.screenReadingDetail(),
                    context: "screen_capture_hotkey"
                )
                Task { @MainActor [weak self] in
                    await self?.screenAnalyzer?.forceAnalysis()
                }
                return true
            case "c":
                self.activateChatMode(source: .hotkey)
                return true
            default:
                return false
            }
        }

        globalHotkeyMonitor = NSEvent.addGlobalMonitorForEvents(matching: .keyDown) { event in
            _ = handler(event)
        }

        localHotkeyMonitor = NSEvent.addLocalMonitorForEvents(matching: .keyDown) { event in
            if handler(event) {
                return nil
            }
            return event
        }
    }

    private func wireSpeechCapture(speechRecognizer: SpeechRecognizer, gateway: GatewayClient) {
        speechRecognizer.onRecordingFormatChanged = { [weak self] format in
            self?.updateAudioConverter(for: format)
        }

        speechRecognizer.onCaptureStateChanged = { [weak self] state in
            self?.speechCaptureState = state
            self?.refreshAudioInputStatusUI()
        }

        speechRecognizer.onBargeInDetected = { [weak self] in
            Task { @MainActor [weak self] in
                self?.handleUserBargeIn()
            }
        }

        speechRecognizer.onAudioBufferCaptured = { [weak self] buffer, forwardMode in
            guard let converter = self?.audioConverter else {
                NSLog("[AUDIO-PIPE] buffer dropped: audioConverter is nil")
                return
            }
            self?.audioConversionQueue.async { [weak self] in
                guard let data = Self.convertAudioBufferToPCM16k(buffer: buffer, converter: converter) else { return }
                Task { @MainActor [weak self] in
                    guard let self else { return }
                    let serverModelTurnActive = self.gatewayClient?.isModelTurnActive == true
                    if serverModelTurnActive && self.speechState.isSpeaking && forwardMode != .bargeIn {
                        NSLog("[AUDIO-PIPE] dropped stale non-barge-in audio during model turn (%lu bytes)", data.count)
                        return
                    }
                    self.gatewayClient?.sendAudio(data)
                }
            }
        }

        Task { @MainActor [weak self] in
            guard self != nil else { return }
            let granted = await speechRecognizer.requestPermissions()
            guard granted else {
                ErrorReporter.shared.report("Microphone permission denied", context: "SpeechRecognizer")
                return
            }

            await speechRecognizer.startListening()
            if speechRecognizer.currentAudioFormat == nil {
                ErrorReporter.shared.report("Failed to initialize audio converter", context: "SpeechRecognizer")
            }
        }
    }

    private func handleUserBargeIn() {
        guard speechState.isSpeaking else { return }
        NSLog("[SPEECH] local barge-in detected")
        gatewayClient?.sendBargeIn()
        catVoice?.stop()
        discardPendingTranscription(reason: "local_barge_in")
        catPanel?.hideBubble()
        transitionSpeech(to: .idle)
        armListeningStatus(panel: catPanel)
    }

    nonisolated private static func convertAudioBufferToPCM16k(buffer: AVAudioPCMBuffer, converter: AVAudioConverter) -> Data? {
        let frameCount = AVAudioFrameCount(Double(buffer.frameLength) * 16000.0 / buffer.format.sampleRate)
        guard let converted = AVAudioPCMBuffer(pcmFormat: converter.outputFormat, frameCapacity: max(frameCount, 1024)) else {
            return nil
        }

        final class OneShotBufferBox: @unchecked Sendable {
            var buffer: AVAudioPCMBuffer?
            init(_ buffer: AVAudioPCMBuffer) {
                self.buffer = buffer
            }
        }

        var error: NSError?
        let box = OneShotBufferBox(buffer)
        let status = converter.convert(to: converted, error: &error) { _, outStatus in
            guard let input = box.buffer else {
                outStatus.pointee = .noDataNow
                return nil
            }
            box.buffer = nil
            outStatus.pointee = .haveData
            return input
        }

        guard status != .error, error == nil else { return nil }
        guard let channels = converted.int16ChannelData else { return nil }
        let sampleCount = Int(converted.frameLength)
        let pointer = channels[0]
        return Data(bytes: pointer, count: sampleCount * MemoryLayout<Int16>.size)
    }

    private func activateChatMode(source: ChatActivationSource) {
        guard let panel = companionChatPanel,
              let catPanel,
              let viewModel = catViewModel else { return }

        let activeFrame = viewModel.activeScreenFrame
        let targetScreen = NSScreen.screens.first(where: { $0.frame == activeFrame }) ?? NSScreen.main
        panel.show(near: catPanel.catPositionInScreenCoordinates(), on: targetScreen)
        if !chatModeActive {
            panel.clearConversation()
        }
        chatModeActive = true
        if source == .wakeWord {
            panel.addUserMessage("Wake word detected")
            panel.addLoadingPlaceholder()
        }
    }

    private func deactivateChatMode() {
        chatModeActive = false
        companionChatPanel?.clearConversation()
    }

    private func maybeActivateChatFromWakeWord(_ text: String) {
        let lowered = text.lowercased()
        guard wakeWords.contains(where: { lowered.contains($0) }) else { return }
        activateChatMode(source: .wakeWord)
    }

    private var notificationsAvailable = false

    private func setupNotifications() {
        guard Bundle.main.bundleIdentifier != nil else { return }
        notificationsAvailable = true
        let center = UNUserNotificationCenter.current()
        center.requestAuthorization(options: [.alert, .sound]) { _, _ in }
    }

    private func sendCompanionNotification(text: String) {
        guard notificationsAvailable else { return }
        guard !NSApp.isActive else { return }

        let title = localizedCharacterName(AppSettings.shared.character)
        let body = text.count > 100 ? String(text.prefix(100)) + "..." : text

        let content = UNMutableNotificationContent()
        content.title = title
        content.body = body
        content.sound = .default

        let request = UNNotificationRequest(
            identifier: "vibecat.speech.\(UUID().uuidString)",
            content: content,
            trigger: nil
        )
        UNUserNotificationCenter.current().add(request)
    }

    private func localizedCharacterName(_ character: String) -> String {
        VibeCatL10n.characterName(character)
    }

    private func currentNavigatorContext() -> NavigatorContextPayload {
        let baseContext = accessibilityNavigator?.currentContext()
            ?? NavigatorContextPayload(
                appName: "",
                bundleId: "",
                frontmostBundleId: "",
                windowTitle: "",
                focusedRole: "",
                focusedLabel: "",
                selectedText: "",
                axSnapshot: "",
                inputFieldHint: "",
                lastInputFieldDescriptor: "",
                screenshot: "",
                focusStableMs: 0,
                captureConfidence: 0,
                visibleInputCandidateCount: 0,
                accessibilityPermission: "unknown",
                accessibilityTrusted: false
            )
        guard let screenAnalyzer else {
            return baseContext
        }
        return screenAnalyzer.latestSharedScreenBasisContext(baseContext: baseContext, includeScreenshot: false)
    }

    private func currentNavigatorCommandContext() async -> NavigatorContextPayload {
        let baseContext = currentNavigatorContext()
        guard let screenAnalyzer else {
            return baseContext
        }
        return await screenAnalyzer.freshNavigatorCommandContext(baseContext: baseContext)
    }

    private func handleNavigatorTextSubmission(_ text: String, captureScreenshot: Bool = true) {
        Task { @MainActor in
            let context = captureScreenshot
                ? await currentNavigatorCommandContext()
                : currentNavigatorContext()
            NSLog("[NAV] submit command=%@ captureScreenshot=%d promptState=%@", text, captureScreenshot, String(describing: navigatorPromptState))
            switch navigatorPromptState {
            case .clarification(let command, let responseMode):
                NSLog("[NAV] clarification responseMode=%@", responseMode.rawValue)
                gatewayClient?.sendNavigatorClarificationResponse(
                    originalCommand: command,
                    answer: text,
                    context: context
                )
            case .risky(let command):
                gatewayClient?.sendNavigatorRiskConfirmation(
                    originalCommand: command,
                    answer: text,
                    context: context
                )
            case nil:
                activeNavigatorCommand = text
                gatewayClient?.sendNavigatorCommand(text, context: context)
            }
        }
    }

    private func executeNavigatorStep(taskId: String, step: NavigatorStep, panel: CatPanel?, chatPanel: CompanionChatPanel?) {
        if let rect = accessibilityNavigator?.highlightRect(for: step) {
            targetHighlightOverlay?.show(targetRect: rect)
        } else {
            targetHighlightOverlay?.hide()
        }
        navigatorActionWorker?.execute(taskId: taskId, step: step) { [weak self] result in
            guard let self else { return }
            self.targetHighlightOverlay?.hide()
            self.showStatusBubbleIfAllowed(
                panel: panel,
                text: VibeCatL10n.navigatorVerifyingTitle(),
                detail: result.observedOutcome,
                context: "navigator_step_result"
            )
            if result.status == "guided_mode" {
                chatPanel?.updateLastAssistantMessage(result.observedOutcome)
            }
        }
    }
}
