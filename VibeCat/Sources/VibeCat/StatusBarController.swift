import AppKit
import VibeCatCore

/// Root menu bar controller — creates the NSStatusItem and builds the full settings menu.
@MainActor
final class StatusBarController {
    private let statusItem: NSStatusItem
    private var trayAnimator: TrayIconAnimator?

    // Action callbacks — set by AppDelegate
    var onReconnect: (() -> Void)?
    var onPause: (() -> Void)?
    var onMute: (() -> Void)?
    var onQuit: (() -> Void)?
    var onShowOnboarding: (() -> Void)?

    init() {
        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.squareLength)
        statusItem.button?.toolTip = "VibeCat"
        buildMenu()
    }

    func attachAnimator(_ animator: TrayIconAnimator) {
        self.trayAnimator = animator
        animator.attach(to: statusItem)
    }

    // MARK: - Menu Construction

    private func buildMenu() {
        let menu = NSMenu()

        // ── Language ──────────────────────────────────────────────────────────
        let languageMenu = NSMenu()
        for (title, code) in [("Korean", "ko"), ("English", "en"), ("Japanese", "ja")] {
            let item = NSMenuItem(title: title, action: #selector(selectLanguage(_:)), keyEquivalent: "")
            item.target = self
            item.representedObject = code
            item.state = AppSettings.shared.language == code ? .on : .off
            languageMenu.addItem(item)
        }
        let languageItem = NSMenuItem(title: "Language", action: nil, keyEquivalent: "")
        languageItem.submenu = languageMenu
        menu.addItem(languageItem)

        // ── Voice ─────────────────────────────────────────────────────────────
        let voiceMenu = NSMenu()
        for voice in ["Zephyr", "Puck", "Kore", "Schedar", "Zubenelgenubi", "Fenrir"] {
            let item = NSMenuItem(title: voice, action: #selector(selectVoice(_:)), keyEquivalent: "")
            item.target = self
            item.representedObject = voice
            item.state = AppSettings.shared.voice == voice ? .on : .off
            voiceMenu.addItem(item)
        }
        let voiceItem = NSMenuItem(title: "Voice", action: nil, keyEquivalent: "")
        voiceItem.submenu = voiceMenu
        menu.addItem(voiceItem)

        // ── Chattiness ────────────────────────────────────────────────────────
        let chattinessMenu = NSMenu()
        for (title, value) in [("Quiet", "quiet"), ("Normal", "normal"), ("Chatty", "chatty")] {
            let item = NSMenuItem(title: title, action: #selector(selectChattiness(_:)), keyEquivalent: "")
            item.target = self
            item.representedObject = value
            item.state = AppSettings.shared.chattiness == value ? .on : .off
            chattinessMenu.addItem(item)
        }
        let chattinessItem = NSMenuItem(title: "Chattiness", action: nil, keyEquivalent: "")
        chattinessItem.submenu = chattinessMenu
        menu.addItem(chattinessItem)

        // ── Character ─────────────────────────────────────────────────────────
        let characterMenu = NSMenu()
        for char in ["cat", "derpy", "jinwoo", "kimjongun", "saja", "trump"] {
            let item = NSMenuItem(title: char, action: #selector(selectCharacter(_:)), keyEquivalent: "")
            item.target = self
            item.representedObject = char
            item.state = AppSettings.shared.character == char ? .on : .off
            characterMenu.addItem(item)
        }
        let characterItem = NSMenuItem(title: "Character", action: nil, keyEquivalent: "")
        characterItem.submenu = characterMenu
        menu.addItem(characterItem)

        menu.addItem(NSMenuItem.separator())

        // ── Model ─────────────────────────────────────────────────────────────
        let modelMenu = NSMenu()
        for model in ["gemini-2.0-flash-live-001", "gemini-2.5-flash-preview-native-audio-dialog"] {
            let item = NSMenuItem(title: model, action: #selector(selectModel(_:)), keyEquivalent: "")
            item.target = self
            item.representedObject = model
            item.state = AppSettings.shared.liveModel == model ? .on : .off
            modelMenu.addItem(item)
        }
        let modelItem = NSMenuItem(title: "Model", action: nil, keyEquivalent: "")
        modelItem.submenu = modelMenu
        menu.addItem(modelItem)

        // ── Capture Interval ──────────────────────────────────────────────────
        let captureMenu = NSMenu()
        for (title, value) in [("3s", 3.0), ("5s", 5.0), ("10s", 10.0), ("30s", 30.0)] {
            let item = NSMenuItem(title: title, action: #selector(selectCaptureInterval(_:)), keyEquivalent: "")
            item.target = self
            item.representedObject = value
            item.state = AppSettings.shared.captureInterval == value ? .on : .off
            captureMenu.addItem(item)
        }
        let captureItem = NSMenuItem(title: "Capture Interval", action: nil, keyEquivalent: "")
        captureItem.submenu = captureMenu
        menu.addItem(captureItem)

        menu.addItem(NSMenuItem.separator())

        // ── Advanced ──────────────────────────────────────────────────────────
        let advancedMenu = NSMenu()

        let musicItem = NSMenuItem(title: "Background Music", action: #selector(toggleMusic(_:)), keyEquivalent: "")
        musicItem.target = self
        musicItem.state = AppSettings.shared.musicEnabled ? .on : .off
        advancedMenu.addItem(musicItem)

        let searchItem = NSMenuItem(title: "Google Search", action: #selector(toggleSearch(_:)), keyEquivalent: "")
        searchItem.target = self
        searchItem.state = AppSettings.shared.searchEnabled ? .on : .off
        advancedMenu.addItem(searchItem)

        let proactiveItem = NSMenuItem(title: "Proactive Audio", action: #selector(toggleProactiveAudio(_:)), keyEquivalent: "")
        proactiveItem.target = self
        proactiveItem.state = AppSettings.shared.proactiveAudio ? .on : .off
        advancedMenu.addItem(proactiveItem)

        let advancedItem = NSMenuItem(title: "Advanced", action: nil, keyEquivalent: "")
        advancedItem.submenu = advancedMenu
        menu.addItem(advancedItem)

        menu.addItem(NSMenuItem.separator())

        // ── Actions ───────────────────────────────────────────────────────────
        let reconnectItem = NSMenuItem(title: "Reconnect", action: #selector(handleReconnect), keyEquivalent: "r")
        reconnectItem.target = self
        menu.addItem(reconnectItem)

        let pauseItem = NSMenuItem(title: "Pause", action: #selector(handlePause), keyEquivalent: "p")
        pauseItem.target = self
        menu.addItem(pauseItem)

        let muteItem = NSMenuItem(title: "Mute", action: #selector(handleMute), keyEquivalent: "m")
        muteItem.target = self
        menu.addItem(muteItem)

        menu.addItem(NSMenuItem.separator())

        let quitItem = NSMenuItem(title: "Quit VibeCat", action: #selector(handleQuit), keyEquivalent: "q")
        quitItem.target = self
        menu.addItem(quitItem)

        statusItem.menu = menu
    }

    // MARK: - Settings Actions

    @objc private func selectLanguage(_ sender: NSMenuItem) {
        guard let code = sender.representedObject as? String else { return }
        AppSettings.shared.language = code
        refreshMenuCheckmarks()
    }

    @objc private func selectVoice(_ sender: NSMenuItem) {
        guard let voice = sender.representedObject as? String else { return }
        AppSettings.shared.voice = voice
        refreshMenuCheckmarks()
    }

    @objc private func selectChattiness(_ sender: NSMenuItem) {
        guard let value = sender.representedObject as? String else { return }
        AppSettings.shared.chattiness = value
        refreshMenuCheckmarks()
    }

    @objc private func selectCharacter(_ sender: NSMenuItem) {
        guard let char = sender.representedObject as? String else { return }
        AppSettings.shared.character = char
        refreshMenuCheckmarks()
    }

    @objc private func selectModel(_ sender: NSMenuItem) {
        guard let model = sender.representedObject as? String else { return }
        AppSettings.shared.liveModel = model
        refreshMenuCheckmarks()
    }

    @objc private func selectCaptureInterval(_ sender: NSMenuItem) {
        guard let value = sender.representedObject as? Double else { return }
        AppSettings.shared.captureInterval = value
        refreshMenuCheckmarks()
    }

    @objc private func toggleMusic(_ sender: NSMenuItem) {
        AppSettings.shared.musicEnabled.toggle()
        sender.state = AppSettings.shared.musicEnabled ? .on : .off
    }

    @objc private func toggleSearch(_ sender: NSMenuItem) {
        AppSettings.shared.searchEnabled.toggle()
        sender.state = AppSettings.shared.searchEnabled ? .on : .off
    }

    @objc private func toggleProactiveAudio(_ sender: NSMenuItem) {
        AppSettings.shared.proactiveAudio.toggle()
        sender.state = AppSettings.shared.proactiveAudio ? .on : .off
    }

    // MARK: - Action Handlers

    @objc private func handleReconnect() {
        onReconnect?()
    }

    @objc private func handlePause() {
        onPause?()
    }

    @objc private func handleMute() {
        onMute?()
    }

    @objc private func handleQuit() {
        if let handler = onQuit {
            handler()
        } else {
            NSApp.terminate(nil)
        }
    }

    // MARK: - Menu Refresh

    /// Rebuild menu to reflect current settings state
    private func refreshMenuCheckmarks() {
        buildMenu()
    }
}
