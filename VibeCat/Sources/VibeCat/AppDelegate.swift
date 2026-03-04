import AppKit
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

    private var isPaused = false

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.accessory)

        let audio = AudioPlayer()
        let voice = CatVoice(audioPlayer: audio)
        let gateway = GatewayClient()
        let capture = ScreenCaptureService()
        let sprite = SpriteAnimator(character: AppSettings.shared.character)
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
        panel.show()

        analyzer.onSpeechEvent = { [weak panel] event in
            panel?.showBubble(text: event.text)
        }

        wireGatewayCallbacks(gateway: gateway, voice: voice, analyzer: analyzer, panel: panel)

        let sbc = StatusBarController()
        let tray = TrayIconAnimator()
        sbc.attachAnimator(tray)
        sbc.onQuit = { NSApp.terminate(nil) }
        sbc.onReconnect = { [weak self] in self?.handleReconnect() }
        sbc.onPause = { [weak self] in self?.handlePause() }
        sbc.onMute = { [weak self] in self?.handleMute() }
        self.statusBarController = sbc
        self.trayAnimator = tray

        let existingKey = try? KeychainHelper.load(forKey: "vibecat-api-key")
        if let key = existingKey, !key.isEmpty {
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
    }

    private func wireGatewayCallbacks(
        gateway: GatewayClient,
        voice: CatVoice,
        analyzer: ScreenAnalyzer,
        panel: CatPanel
    ) {
        gateway.onAudioData = { data in
            voice.enqueueAudio(data)
        }
        gateway.onStateChange = { state in
            switch state {
            case .connected:
                voice.setLiveConnected(true)
            case .disconnected, .failed:
                voice.setLiveConnected(false)
            default:
                break
            }
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

    private func showOnboarding() {
        let controller = OnboardingWindowController()
        controller.onConnect = { [weak self] key in
            guard let self else { return }
            self.onboardingController = nil
            self.gatewayClient?.connect(apiKey: key)
            self.screenAnalyzer?.start()
        }
        controller.show()
        self.onboardingController = controller
    }

    private func handleReconnect() {
        guard let key = try? KeychainHelper.load(forKey: "vibecat-api-key"), !key.isEmpty else {
            showOnboarding()
            return
        }
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
}
