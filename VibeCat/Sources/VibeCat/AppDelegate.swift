import AppKit
import UserNotifications
import VibeCatCore

@MainActor
final class AppDelegate: NSObject, NSApplicationDelegate {
    private var statusBarController: StatusBarController?
    private var trayAnimator: TrayIconAnimator?
    private var onboardingController: OnboardingWindowController?
    private var catPanel: CatPanel?

    private var audioPlayer: AudioPlayer?
    private var catVoice: CatVoice?
    private var gatewayClient: GatewayClient?
    private var captureService: ScreenCaptureService?
    private var screenAnalyzer: ScreenAnalyzer?
    private var spriteAnimator: SpriteAnimator?
    private var catViewModel: CatViewModel?
    private var backgroundMusicPlayer: BackgroundMusicPlayer?
    private var recentSpeechStore = RecentSpeechStore()
    private var emotionTransitionStore = EmotionTransitionStore()
    private var globalHotkeyMonitor: Any?
    private var localHotkeyMonitor: Any?

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

        self.audioPlayer = audio
        self.catVoice = voice
        self.gatewayClient = gateway
        self.captureService = capture
        self.spriteAnimator = sprite
        self.catViewModel = viewModel
        self.backgroundMusicPlayer = music

        let analyzer = ScreenAnalyzer(
            captureService: capture,
            gatewayClient: gateway,
            catVoice: voice,
            spriteAnimator: sprite
        )
        self.screenAnalyzer = analyzer

        let panel = CatPanel(catViewModel: viewModel, spriteAnimator: sprite)
        self.catPanel = panel
        viewModel.panel = panel
        panel.setCatSize(initialPreset.size)
        panel.show()

        gateway.setSoul(initialPreset.soul)

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
            statusBarController: sbc
        )

        setupNotifications()
        registerGlobalHotkey()

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
            case .disconnected, .failed:
                voice.setLiveConnected(false)
                statusBarController?.updateConnectionStatus(.disconnected)
                statusBarController?.setLastErrorDescription(gateway.lastErrorDescription)
                let attention = gateway.lastErrorDescription?.contains("API key") == true
                statusBarController?.setAPIKeyNeedsAttention(attention)
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
        }
        gateway.onReconnectExhausted = { [weak statusBarController] in
            statusBarController?.updateConnectionStatus(.disconnected)
            statusBarController?.setLastErrorDescription(gateway.lastErrorDescription)
            statusBarController?.setAPIKeyNeedsAttention(true)
        }
        gateway.onLatencyUpdate = { [weak statusBarController] latency in
            statusBarController?.updateLatency(latency)
        }
        gateway.onMessage = { [weak analyzer, weak panel] message in
            switch message {
            case .companionSpeech(let text, let emotionStr, _):
                let emotion = CompanionEmotion(rawValue: emotionStr) ?? .neutral
                let event = CompanionSpeechEvent(text: text, emotion: emotion)
                analyzer?.handleCompanionSpeech(event)
            case .transcription(let text, let finished):
                if finished {
                    panel?.showBubble(text: text)
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
        catPanel?.setCatSize(preset.size)
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
            isPaused = false
        } else {
            screenAnalyzer?.pause()
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
            guard flags == [.option], event.charactersIgnoringModifiers?.lowercased() == "v" else {
                return false
            }

            catPanel?.showBubble(text: "Analyzing...")
            Task { @MainActor [weak self] in
                await self?.screenAnalyzer?.forceAnalysis()
            }
            return true
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

    private func setupNotifications() {
        let center = UNUserNotificationCenter.current()
        center.requestAuthorization(options: [.alert, .sound]) { _, _ in }
    }

    private func sendCompanionNotification(text: String) {
        guard !NSApp.isActive else { return }

        let title = localizedCharacterName(AppSettings.shared.character)
        let clipped: String
        if text.count > 100 {
            clipped = String(text.prefix(100)) + "..."
        } else {
            clipped = text
        }

        let content = UNMutableNotificationContent()
        content.title = title
        content.body = clipped
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
