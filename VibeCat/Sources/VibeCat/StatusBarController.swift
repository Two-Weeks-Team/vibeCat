import AppKit
import VibeCatCore

@MainActor
final class SessionTracker {
    private(set) var sessionStartTime: Date?
    private(set) var interactionCount = 0
    private(set) var lastInteractionTime: Date?
    private(set) var latencyMs: Int?
    private(set) var lastSeenTime: Date?

    func markConnected() {
        sessionStartTime = Date()
        interactionCount = 0
        lastInteractionTime = nil
        lastSeenTime = nil
    }

    func markDisconnected() {
        lastSeenTime = Date()
        sessionStartTime = nil
        latencyMs = nil
    }

    func recordInteraction() {
        interactionCount += 1
        lastInteractionTime = Date()
    }

    func updateLatency(_ value: Int) {
        latencyMs = max(0, value)
    }
}

@MainActor
final class RecentSpeechStore {
    enum Speaker {
        case user
        case assistant

        var label: String {
            switch self {
            case .user:
                return "You"
            case .assistant:
                return "AI"
            }
        }
    }

    struct Entry {
        let speaker: Speaker
        let text: String
        let timestamp: Date
    }

    private var entries: [Entry] = []

    func add(_ text: String, speaker: Speaker, at date: Date = Date()) {
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return }
        entries.insert(Entry(speaker: speaker, text: trimmed, timestamp: date), at: 0)
        if entries.count > 10 {
            entries = Array(entries.prefix(10))
        }
    }

    func allEntries() -> [Entry] {
        entries
    }
}

@MainActor
final class EmotionTransitionStore {
    struct Entry {
        let from: SpriteAnimator.AnimationState
        let to: SpriteAnimator.AnimationState
        let timestamp: Date
    }

    private var entries: [Entry] = []

    func add(from: SpriteAnimator.AnimationState, to: SpriteAnimator.AnimationState, at date: Date = Date()) {
        entries.insert(Entry(from: from, to: to, timestamp: date), at: 0)
        if entries.count > 8 {
            entries = Array(entries.prefix(8))
        }
    }

    func allEntries() -> [Entry] {
        entries
    }
}

@MainActor
final class StatusBarController: NSObject, NSMenuDelegate {
    enum ConnectionState: Equatable {
        case connected
        case disconnected
        case reconnecting(attempt: Int, max: Int)
    }

    private let statusItem: NSStatusItem
    private var trayAnimator: TrayIconAnimator?
    private var menu = NSMenu()
    private let statusTextItem = NSMenuItem()
    private let sessionTracker = SessionTracker()

    private var languageItems: [NSMenuItem] = []
    private var voiceItems: [NSMenuItem] = []
    private var chattinessItems: [NSMenuItem] = []
    private var characterItems: [NSMenuItem] = []
    private var modelItems: [NSMenuItem] = []
    private var captureItems: [NSMenuItem] = []
    private var captureModeItems: [NSMenuItem] = []

    private var musicItem: NSMenuItem?
    private var searchItem: NSMenuItem?
    private var proactiveItem: NSMenuItem?
    private var pauseItem: NSMenuItem?
    private var muteItem: NSMenuItem?
    private var setAPIKeyItem: NSMenuItem?
    private var recentSpeechMenu: NSMenu?
    private var emotionHistoryMenu: NSMenu?

    private var lastErrorDescription: String?
    private var apiKeyNeedsAttention = false

    private var recentSpeechStore: RecentSpeechStore?
    private var emotionTransitionStore: EmotionTransitionStore?

    var onReconnect: (() -> Void)?
    var onPause: (() -> Void)?
    var onMute: (() -> Void)?
    var onQuit: (() -> Void)?
    var onShowOnboarding: (() -> Void)?
    var onCharacterChanged: ((String) -> Void)?
    var onMusicToggled: ((Bool) -> Void)?
    var onSearchToggled: (() -> Void)?
    var onProactiveAudioToggled: (() -> Void)?

    private(set) var connectionState: ConnectionState = .disconnected {
        didSet {
            updateStatusTextItem()
            updateTooltip()
        }
    }

    override init() {
        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.squareLength)
        statusItem.button?.toolTip = "VibeCat"
        super.init()
        buildMenu()
    }

    func attachAnimator(_ animator: TrayIconAnimator) {
        trayAnimator = animator
        animator.attach(to: statusItem)
    }

    func attachRecentSpeechStore(_ store: RecentSpeechStore) {
        recentSpeechStore = store
    }

    func attachEmotionTransitionStore(_ store: EmotionTransitionStore) {
        emotionTransitionStore = store
    }

    func updateConnectionStatus(_ state: ConnectionState) {
        let oldState = connectionState
        connectionState = state
        switch state {
        case .connected:
            if oldState != .connected {
                sessionTracker.markConnected()
            }
        case .disconnected:
            if oldState != .disconnected {
                sessionTracker.markDisconnected()
            }
        case .reconnecting:
            break
        }
    }

    func updateLatency(_ ms: Int) {
        sessionTracker.updateLatency(ms)
        updateStatusTextItem()
    }

    func recordInteraction() {
        sessionTracker.recordInteraction()
        updateStatusTextItem()
    }

    func setLastErrorDescription(_ description: String?) {
        let trimmed = description?.trimmingCharacters(in: .whitespacesAndNewlines)
        lastErrorDescription = (trimmed?.isEmpty == false) ? trimmed : nil
        updateStatusTextItem()
    }

    func setAPIKeyNeedsAttention(_ needsAttention: Bool) {
        apiKeyNeedsAttention = needsAttention
        updateAPIKeyMenuItemStyle()
    }

    private func buildMenu() {
        menu = NSMenu()
        menu.delegate = self

        statusTextItem.isEnabled = false
        menu.addItem(statusTextItem)
        updateStatusTextItem()
        menu.addItem(NSMenuItem.separator())

        let languageMenu = NSMenu()
        languageItems.removeAll(keepingCapacity: true)
        for (title, code) in [("Korean", "ko"), ("English", "en"), ("Japanese", "ja")] {
            let item = NSMenuItem(title: title, action: #selector(selectLanguage(_:)), keyEquivalent: "")
            item.target = self
            item.representedObject = code
            item.state = AppSettings.shared.language == code ? .on : .off
            languageMenu.addItem(item)
            languageItems.append(item)
        }
        let languageItem = NSMenuItem(title: "Language", action: nil, keyEquivalent: "")
        languageItem.submenu = languageMenu
        menu.addItem(languageItem)

        let voiceMenu = NSMenu()
        voiceItems.removeAll(keepingCapacity: true)
        for voice in ["Zephyr", "Puck", "Kore", "Schedar", "Zubenelgenubi", "Fenrir"] {
            let item = NSMenuItem(title: voice, action: #selector(selectVoice(_:)), keyEquivalent: "")
            item.target = self
            item.representedObject = voice
            item.state = AppSettings.shared.voice == voice ? .on : .off
            voiceMenu.addItem(item)
            voiceItems.append(item)
        }
        let voiceItem = NSMenuItem(title: "Voice", action: nil, keyEquivalent: "")
        voiceItem.submenu = voiceMenu
        menu.addItem(voiceItem)

        let chattinessMenu = NSMenu()
        chattinessItems.removeAll(keepingCapacity: true)
        for (title, value) in [("Quiet", "quiet"), ("Normal", "normal"), ("Chatty", "chatty")] {
            let item = NSMenuItem(title: title, action: #selector(selectChattiness(_:)), keyEquivalent: "")
            item.target = self
            item.representedObject = value
            item.state = AppSettings.shared.chattiness == value ? .on : .off
            chattinessMenu.addItem(item)
            chattinessItems.append(item)
        }
        let chattinessItem = NSMenuItem(title: "Chattiness", action: nil, keyEquivalent: "")
        chattinessItem.submenu = chattinessMenu
        menu.addItem(chattinessItem)

        let characterMenu = NSMenu()
        characterItems.removeAll(keepingCapacity: true)
        for char in ["cat", "derpy", "jinwoo", "kimjongun", "saja", "trump"] {
            let item = NSMenuItem(title: char, action: #selector(selectCharacter(_:)), keyEquivalent: "")
            item.target = self
            item.representedObject = char
            item.state = AppSettings.shared.character == char ? .on : .off
            characterMenu.addItem(item)
            characterItems.append(item)
        }
        let characterItem = NSMenuItem(title: "Character", action: nil, keyEquivalent: "")
        characterItem.submenu = characterMenu
        menu.addItem(characterItem)

        let recentSpeechItem = NSMenuItem(title: "Recent Speech", action: nil, keyEquivalent: "")
        let recentSubmenu = NSMenu()
        recentSpeechItem.submenu = recentSubmenu
        recentSpeechMenu = recentSubmenu
        menu.addItem(recentSpeechItem)

        let emotionItem = NSMenuItem(title: "Emotion History", action: nil, keyEquivalent: "")
        let emotionSubmenu = NSMenu()
        emotionItem.submenu = emotionSubmenu
        emotionHistoryMenu = emotionSubmenu
        menu.addItem(emotionItem)

        menu.addItem(NSMenuItem.separator())

        let modelMenu = NSMenu()
        modelItems.removeAll(keepingCapacity: true)

        let liveHeader = NSMenuItem(title: "Live API Model", action: nil, keyEquivalent: "")
        liveHeader.isEnabled = false
        modelMenu.addItem(liveHeader)

        for model in GeminiModels.selectableLiveModels {
            let item = NSMenuItem(title: "  \(model)", action: #selector(selectModel(_:)), keyEquivalent: "")
            item.target = self
            item.representedObject = model
            item.state = AppSettings.shared.liveModel == model ? .on : .off
            modelMenu.addItem(item)
            modelItems.append(item)
        }

        modelMenu.addItem(NSMenuItem.separator())

        let visionHeader = NSMenuItem(title: "Backend Analysis Models", action: nil, keyEquivalent: "")
        visionHeader.isEnabled = false
        modelMenu.addItem(visionHeader)

        let visionInfoItem = NSMenuItem(title: "  Vision/Search: \(GeminiModels.vision)", action: nil, keyEquivalent: "")
        visionInfoItem.state = .on
        visionInfoItem.isEnabled = false
        modelMenu.addItem(visionInfoItem)

        let supportInfoItem = NSMenuItem(title: "  Support: \(GeminiModels.liteSupport)", action: nil, keyEquivalent: "")
        supportInfoItem.state = .off
        supportInfoItem.isEnabled = false
        modelMenu.addItem(supportInfoItem)

        let modelItem = NSMenuItem(title: "Model", action: nil, keyEquivalent: "")
        modelItem.submenu = modelMenu
        menu.addItem(modelItem)

        let captureMenu = NSMenu()
        captureItems.removeAll(keepingCapacity: true)
        for (title, value) in [("1s", 1.0), ("3s", 3.0), ("5s", 5.0), ("10s", 10.0), ("30s", 30.0)] {
            let item = NSMenuItem(title: title, action: #selector(selectCaptureInterval(_:)), keyEquivalent: "")
            item.target = self
            item.representedObject = value
            item.state = AppSettings.shared.captureInterval == value ? .on : .off
            captureMenu.addItem(item)
            captureItems.append(item)
        }
        let captureItem = NSMenuItem(title: "Capture Interval", action: nil, keyEquivalent: "")
        captureItem.submenu = captureMenu
        menu.addItem(captureItem)

        let captureTargetMenu = NSMenu()
        captureModeItems.removeAll(keepingCapacity: true)
        for mode in CaptureTargetMode.allCases {
            let item = NSMenuItem(title: mode.menuTitle, action: #selector(selectCaptureTargetMode(_:)), keyEquivalent: "")
            item.target = self
            item.representedObject = mode.rawValue
            item.state = AppSettings.shared.captureTargetMode == mode ? .on : .off
            captureTargetMenu.addItem(item)
            captureModeItems.append(item)
        }
        let captureTargetItem = NSMenuItem(title: "Capture Target", action: nil, keyEquivalent: "")
        captureTargetItem.submenu = captureTargetMenu
        menu.addItem(captureTargetItem)

        menu.addItem(NSMenuItem.separator())

        let advancedMenu = NSMenu()

        let createdMusicItem = NSMenuItem(title: "Background Music", action: #selector(toggleMusic(_:)), keyEquivalent: "")
        createdMusicItem.target = self
        createdMusicItem.state = AppSettings.shared.musicEnabled ? .on : .off
        advancedMenu.addItem(createdMusicItem)
        musicItem = createdMusicItem

        let createdSearchItem = NSMenuItem(title: "Google Search", action: #selector(toggleSearch(_:)), keyEquivalent: "")
        createdSearchItem.target = self
        createdSearchItem.state = AppSettings.shared.searchEnabled ? .on : .off
        advancedMenu.addItem(createdSearchItem)
        searchItem = createdSearchItem

        let createdProactiveItem = NSMenuItem(title: "Proactive Audio", action: #selector(toggleProactiveAudio(_:)), keyEquivalent: "")
        createdProactiveItem.target = self
        createdProactiveItem.state = AppSettings.shared.proactiveAudio ? .on : .off
        advancedMenu.addItem(createdProactiveItem)
        proactiveItem = createdProactiveItem

        let advancedItem = NSMenuItem(title: "Advanced", action: nil, keyEquivalent: "")
        advancedItem.submenu = advancedMenu
        menu.addItem(advancedItem)

        menu.addItem(NSMenuItem.separator())

        let createdSetAPIKeyItem = NSMenuItem(title: "Connect...", action: #selector(handleShowOnboarding), keyEquivalent: "")
        createdSetAPIKeyItem.target = self
        setAPIKeyItem = createdSetAPIKeyItem
        menu.addItem(createdSetAPIKeyItem)
        updateAPIKeyMenuItemStyle()

        menu.addItem(NSMenuItem.separator())

        let reconnectItem = NSMenuItem(title: "Reconnect", action: #selector(handleReconnect), keyEquivalent: "r")
        reconnectItem.target = self
        menu.addItem(reconnectItem)

        let pauseItem = NSMenuItem(title: "Pause", action: #selector(handlePause), keyEquivalent: "p")
        pauseItem.target = self
        menu.addItem(pauseItem)
        self.pauseItem = pauseItem

        let muteItem = NSMenuItem(title: "Mute", action: #selector(handleMute), keyEquivalent: "m")
        muteItem.target = self
        menu.addItem(muteItem)
        self.muteItem = muteItem

        menu.addItem(NSMenuItem.separator())

        let quitItem = NSMenuItem(title: "Quit VibeCat", action: #selector(handleQuit), keyEquivalent: "q")
        quitItem.target = self
        menu.addItem(quitItem)

        statusItem.menu = menu
    }

    private func updateStatusTextItem() {
        let attributed = NSMutableAttributedString()
        switch connectionState {
        case .connected:
            attributed.append(NSAttributedString(string: "●", attributes: [.foregroundColor: NSColor.systemGreen]))
            let latencyText = "\(sessionTracker.latencyMs ?? 0)ms"
            let sessionDuration = formatDuration(since: sessionTracker.sessionStartTime)
            let interactions = sessionTracker.interactionCount
            attributed.append(NSAttributedString(string: " Connected · \(latencyText) · \(interactions) interactions · \(sessionDuration)"))
        case .reconnecting(let attempt, let max):
            attributed.append(NSAttributedString(string: "●", attributes: [.foregroundColor: NSColor.systemYellow]))
            attributed.append(NSAttributedString(string: " Reconnecting… (\(attempt)/\(max))"))
        case .disconnected:
            attributed.append(NSAttributedString(string: "○", attributes: [.foregroundColor: NSColor.systemRed]))
            let seen = relativeTime(from: sessionTracker.lastSeenTime)
            attributed.append(NSAttributedString(string: " Disconnected · Last seen: \(seen)"))
        }

        if let error = lastErrorDescription {
            attributed.append(NSAttributedString(string: " · \(error)"))
        }

        statusTextItem.attributedTitle = attributed
    }

    @objc private func selectLanguage(_ sender: NSMenuItem) {
        guard let code = sender.representedObject as? String else { return }
        AppSettings.shared.language = code
        refreshSubmenuChecks()
    }

    @objc private func selectVoice(_ sender: NSMenuItem) {
        guard let voice = sender.representedObject as? String else { return }
        AppSettings.shared.voice = voice
        refreshSubmenuChecks()
    }

    @objc private func selectChattiness(_ sender: NSMenuItem) {
        guard let value = sender.representedObject as? String else { return }
        AppSettings.shared.chattiness = value
        refreshSubmenuChecks()
    }

    @objc private func selectCharacter(_ sender: NSMenuItem) {
        guard let char = sender.representedObject as? String else { return }
        AppSettings.shared.character = char
        onCharacterChanged?(char)
        refreshSubmenuChecks()
    }

    @objc private func selectModel(_ sender: NSMenuItem) {
        guard let model = sender.representedObject as? String else { return }
        AppSettings.shared.liveModel = model
        refreshSubmenuChecks()
    }

    @objc private func selectCaptureInterval(_ sender: NSMenuItem) {
        guard let value = sender.representedObject as? Double else { return }
        AppSettings.shared.captureInterval = value
        refreshSubmenuChecks()
    }

    @objc private func selectCaptureTargetMode(_ sender: NSMenuItem) {
        guard let rawValue = sender.representedObject as? String,
              let mode = CaptureTargetMode(rawValue: rawValue) else { return }
        AppSettings.shared.captureTargetMode = mode
        refreshSubmenuChecks()
    }

    @objc private func toggleMusic(_ sender: NSMenuItem) {
        AppSettings.shared.musicEnabled.toggle()
        sender.state = AppSettings.shared.musicEnabled ? .on : .off
        onMusicToggled?(AppSettings.shared.musicEnabled)
    }

    @objc private func toggleSearch(_ sender: NSMenuItem) {
        AppSettings.shared.searchEnabled.toggle()
        sender.state = AppSettings.shared.searchEnabled ? .on : .off
        onSearchToggled?()
    }

    @objc private func toggleProactiveAudio(_ sender: NSMenuItem) {
        AppSettings.shared.proactiveAudio.toggle()
        sender.state = AppSettings.shared.proactiveAudio ? .on : .off
        onProactiveAudioToggled?()
    }

    func updatePauseState(_ isPaused: Bool) {
        pauseItem?.title = isPaused ? "Resume" : "Pause"
    }

    func updateMuteState(_ isMuted: Bool) {
        muteItem?.title = isMuted ? "Unmute" : "Mute"
    }

    @objc private func handleReconnect() {
        onReconnect?()
    }

    @objc private func handlePause() {
        onPause?()
    }

    @objc private func handleMute() {
        onMute?()
    }

    @objc private func handleShowOnboarding() {
        onShowOnboarding?()
    }

    @objc private func handleQuit() {
        if let handler = onQuit {
            handler()
        } else {
            NSApp.terminate(nil)
        }
    }

    @objc private func handleCopyRecentSpeech(_ sender: NSMenuItem) {
        guard let value = sender.representedObject as? String else { return }
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(value, forType: .string)
    }

    private func refreshSubmenuChecks() {
        for item in languageItems {
            guard let code = item.representedObject as? String else { continue }
            item.state = AppSettings.shared.language == code ? .on : .off
        }
        for item in voiceItems {
            guard let voice = item.representedObject as? String else { continue }
            item.state = AppSettings.shared.voice == voice ? .on : .off
        }
        for item in chattinessItems {
            guard let value = item.representedObject as? String else { continue }
            item.state = AppSettings.shared.chattiness == value ? .on : .off
        }
        for item in characterItems {
            guard let value = item.representedObject as? String else { continue }
            item.state = AppSettings.shared.character == value ? .on : .off
        }
        for item in modelItems {
            guard let value = item.representedObject as? String else { continue }
            item.state = AppSettings.shared.liveModel == value ? .on : .off
        }
        for item in captureItems {
            guard let value = item.representedObject as? Double else { continue }
            item.state = AppSettings.shared.captureInterval == value ? .on : .off
        }
        for item in captureModeItems {
            guard let rawValue = item.representedObject as? String,
                  let mode = CaptureTargetMode(rawValue: rawValue) else { continue }
            item.state = AppSettings.shared.captureTargetMode == mode ? .on : .off
        }

        musicItem?.state = AppSettings.shared.musicEnabled ? .on : .off
        searchItem?.state = AppSettings.shared.searchEnabled ? .on : .off
        proactiveItem?.state = AppSettings.shared.proactiveAudio ? .on : .off
    }

    private func rebuildRecentSpeechMenu() {
        guard let recentSpeechMenu else { return }
        recentSpeechMenu.removeAllItems()
        let entries = recentSpeechStore?.allEntries() ?? []
        if entries.isEmpty {
            let empty = NSMenuItem(title: "No recent speech", action: nil, keyEquivalent: "")
            empty.isEnabled = false
            recentSpeechMenu.addItem(empty)
            return
        }

        for entry in entries {
            let relative = relativeTime(from: entry.timestamp)
            let title = "[\(entry.speaker.label)] \(entry.text) (\(relative))"
            let item = NSMenuItem(title: title, action: #selector(handleCopyRecentSpeech(_:)), keyEquivalent: "")
            item.target = self
            item.representedObject = entry.text
            recentSpeechMenu.addItem(item)
        }
    }

    private func rebuildEmotionHistoryMenu() {
        guard let emotionHistoryMenu else { return }
        emotionHistoryMenu.removeAllItems()
        let entries = emotionTransitionStore?.allEntries() ?? []
        if entries.isEmpty {
            let empty = NSMenuItem(title: "No emotion transitions", action: nil, keyEquivalent: "")
            empty.isEnabled = false
            emotionHistoryMenu.addItem(empty)
            return
        }

        for entry in entries {
            let title = "\(entry.from.rawValue) -> \(entry.to.rawValue) (\(relativeTime(from: entry.timestamp)))"
            let item = NSMenuItem(title: title, action: nil, keyEquivalent: "")
            item.isEnabled = false
            emotionHistoryMenu.addItem(item)
        }
    }

    private func updateTooltip() {
        switch connectionState {
        case .connected:
            statusItem.button?.toolTip = "VibeCat"
        case .disconnected, .reconnecting:
            statusItem.button?.toolTip = "Offline — reconnecting…"
        }
    }

    private func updateAPIKeyMenuItemStyle() {
        guard let setAPIKeyItem else { return }
        if apiKeyNeedsAttention {
            setAPIKeyItem.attributedTitle = NSAttributedString(
                string: "Connect...",
                attributes: [.foregroundColor: NSColor.systemRed]
            )
        } else {
            setAPIKeyItem.attributedTitle = NSAttributedString(string: "Connect...")
        }
    }

    private func formatDuration(since date: Date?) -> String {
        guard let date else { return "0m 0s" }
        let seconds = Int(Date().timeIntervalSince(date))
        let minutes = seconds / 60
        let remainder = seconds % 60
        return "\(minutes)m \(remainder)s"
    }

    private func relativeTime(from date: Date?) -> String {
        guard let date else { return "never" }
        let delta = Int(Date().timeIntervalSince(date))
        if delta < 10 { return "Just now" }
        if delta < 60 { return "\(delta)s ago" }
        if delta < 3600 { return "\(delta / 60)m ago" }
        return "\(delta / 3600)h ago"
    }

    func menuWillOpen(_ menu: NSMenu) {
        refreshSubmenuChecks()
        rebuildRecentSpeechMenu()
        rebuildEmotionHistoryMenu()
        updateStatusTextItem()
    }
}
