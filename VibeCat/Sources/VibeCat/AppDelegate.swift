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

    private var audioPlayer: AudioPlayer?
    private var catVoice: CatVoice?
    private var gatewayClient: GatewayClient?
    private var captureService: ScreenCaptureService?
    private var screenAnalyzer: ScreenAnalyzer?
    private var spriteAnimator: SpriteAnimator?
    private var catViewModel: CatViewModel?
    private var backgroundMusicPlayer: BackgroundMusicPlayer?
    private var speechRecognizer: SpeechRecognizer?
    private var circleGestureDetector: CircleGestureDetector?
    private var recentSpeechStore = RecentSpeechStore()
    private var emotionTransitionStore = EmotionTransitionStore()
    private var globalHotkeyMonitor: Any?
    private var localHotkeyMonitor: Any?
    private let audioConversionQueue = DispatchQueue(label: "vibecat.audio.conversion")
    nonisolated(unsafe) private var audioConverter: AVAudioConverter?
    private var chatModeActive = false
    private let wakeWords = ["vibecat", "vibe cat", "바이브캣"]

    private var isPaused = false
    private var storedAPIKey: String?

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
        let circleGestureDetector = CircleGestureDetector()
        let companionChatPanel = CompanionChatPanel()

        self.audioPlayer = audio
        self.catVoice = voice
        self.gatewayClient = gateway
        self.captureService = capture
        self.spriteAnimator = sprite
        self.catViewModel = viewModel
        self.backgroundMusicPlayer = music
        self.speechRecognizer = speechRecognizer
        self.circleGestureDetector = circleGestureDetector
        self.companionChatPanel = companionChatPanel

        let analyzer = ScreenAnalyzer(
            captureService: capture,
            gatewayClient: gateway,
            catVoice: voice,
            spriteAnimator: sprite
        )
        self.screenAnalyzer = analyzer

        let panel = CatPanel(catViewModel: viewModel, spriteAnimator: sprite)
        self.catPanel = panel
        panel.applySpriteSize(presetSize: initialPreset.size)
        panel.show()

        gateway.setSoul(initialPreset.soul)

        circleGestureDetector.onCircleGesture = { [weak self] in
            self?.catPanel?.showBubble(text: "Analyzing...")
            Task { @MainActor [weak self] in
                await self?.screenAnalyzer?.forceAnalysis()
            }
        }
        circleGestureDetector.startMonitoring()

        companionChatPanel.onTextSubmitted = { [weak self] text in
            guard let self else { return }
            self.chatModeActive = true
            self.companionChatPanel?.addUserMessage(text)
            self.companionChatPanel?.addLoadingPlaceholder()
            self.gatewayClient?.sendText(text)
            self.statusBarController?.recordInteraction()
        }
        companionChatPanel.onDismissed = { [weak self] in
            self?.deactivateChatMode()
        }

        analyzer.onSpeechEvent = { [weak self, weak panel] event in
            panel?.showBubble(text: event.text)
            self?.recentSpeechStore.add(event.text)
            self?.statusBarController?.recordInteraction()
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
        sbc.onPause = { [weak self] in self?.handlePause() }
        sbc.onMute = { [weak self] in self?.handleMute() }
        sbc.onShowOnboarding = { [weak self] in self?.showOnboarding() }
        sbc.onCharacterChanged = { [weak self] character in
            self?.handleCharacterChanged(character)
        }
        self.statusBarController = sbc
        self.trayAnimator = tray

        sprite.onStateTransition = { [weak self] from, to in
            self?.emotionTransitionStore.add(from: from, to: to)
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

        let existingKey = try? KeychainHelper.load(forKey: "vibecat-api-key")
        if let key = existingKey, !key.isEmpty {
            storedAPIKey = key
            gateway.connect(apiKey: key)
            analyzer.start()
        } else {
            showOnboarding()
        }
    }

    func applicationWillTerminate(_ notification: Notification) {
        screenAnalyzer?.pause()
        gatewayClient?.disconnect()
        speechRecognizer?.stopListening()
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
        gateway.onAudioData = { data in
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
                let attention = gateway.lastErrorDescription?.contains("API key") == true
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
        gateway.onDisconnected = { [weak statusBarController] in
            statusBarController?.updateConnectionStatus(.disconnected)
            statusBarController?.setLastErrorDescription(gateway.lastErrorDescription)
            if let error = gateway.lastErrorDescription {
                ErrorReporter.shared.report(error, context: "Gateway")
            }
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
            case .companionSpeech(let text, let emotionStr, _):
                let emotion = CompanionEmotion(rawValue: emotionStr) ?? .neutral
                let event = CompanionSpeechEvent(text: text, emotion: emotion)
                if self.chatModeActive {
                    chatPanel?.updateLastAssistantMessage(text)
                } else {
                    analyzer?.handleCompanionSpeech(event)
                }
            case .transcription(let text, let finished):
                if self.chatModeActive {
                    if finished {
                        chatPanel?.finalizeListeningText(text)
                    } else {
                        chatPanel?.updateListeningText(text)
                    }
                } else if finished {
                    panel?.showBubble(text: text)
                    self.maybeActivateChatFromWakeWord(text)
                }
            default:
                break
            }
        }
    }

    private func handleCharacterChanged(_ character: String) {
        spriteAnimator?.setCharacter(character)

        guard let preset = spriteAnimator?.loadPreset(for: character) else { return }
        AppSettings.shared.voice = preset.voice
        catPanel?.applySpriteSize(presetSize: preset.size)
        gatewayClient?.setSoul(preset.soul)
        gatewayClient?.resendSetupPayloadIfConnected()
    }

    private func showOnboarding() {
        let controller = OnboardingWindowController()
        controller.onConnect = { [weak self] key in
            guard let self else { return }
            self.onboardingController = nil
            self.storedAPIKey = key
            self.gatewayClient?.connect(apiKey: key)
            self.screenAnalyzer?.start()
        }
        controller.show()
        self.onboardingController = controller
    }

    private func handleReconnect() {
        if let key = storedAPIKey, !key.isEmpty {
            gatewayClient?.reconnect(apiKey: key)
            return
        }

        guard let key = try? KeychainHelper.load(forKey: "vibecat-api-key"), !key.isEmpty else {
            showOnboarding()
            return
        }
        storedAPIKey = key
        gatewayClient?.reconnect(apiKey: key)
    }

    private func handlePause() {
        if isPaused {
            screenAnalyzer?.resume()
            speechRecognizer?.resumeListening()
            circleGestureDetector?.startMonitoring()
            isPaused = false
        } else {
            screenAnalyzer?.pause()
            speechRecognizer?.stopListening()
            circleGestureDetector?.stopMonitoring()
            isPaused = true
        }
    }

    private func handleMute() {
        catVoice?.mute()
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
                catPanel?.showBubble(text: "Analyzing...")
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
        speechRecognizer.onAudioBufferCaptured = { [weak self] buffer in
            guard let converter = self?.audioConverter else { return }
            self?.audioConversionQueue.async { [weak self] in
                guard let data = Self.convertAudioBufferToPCM16k(buffer: buffer, converter: converter) else { return }
                Task { @MainActor [weak self] in
                    self?.gatewayClient?.sendAudio(data)
                }
            }
        }

        Task { @MainActor [weak self] in
            guard let self else { return }
            let granted = await speechRecognizer.requestPermissions()
            guard granted else {
                ErrorReporter.shared.report("Microphone permission denied", context: "SpeechRecognizer")
                return
            }

            await speechRecognizer.startListening()
            if let input = speechRecognizer.currentAudioFormat,
               let output = AVAudioFormat(commonFormat: .pcmFormatInt16, sampleRate: 16000, channels: 1, interleaved: true) {
                self.audioConverter = AVAudioConverter(from: input, to: output)
            } else {
                ErrorReporter.shared.report("Failed to initialize audio converter", context: "SpeechRecognizer")
            }
        }
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
        switch character {
        case "cat":
            return "고양이"
        case "derpy":
            return "더피"
        case "jinwoo":
            return "진우"
        case "kimjongun":
            return "김정운"
        case "saja":
            return "사자"
        case "trump":
            return "트럼프"
        default:
            return character
        }
    }
}
