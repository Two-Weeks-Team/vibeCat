import AppKit
import VibeCatCore

/// First-launch onboarding panel for gateway connection.
@MainActor
final class OnboardingWindowController: NSObject {
    private var window: NSWindow?
    private var errorLabel: NSTextField?
    private var titleLabel: NSTextField?
    private var subtitleLabel: NSTextField?
    private var connectButton: NSButton?
    private var cancelButton: NSButton?

    var onConnect: (() -> Void)?

    func show() {
        if window == nil {
            buildWindow()
        }
        refreshLocalizedText()
        window?.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)
    }

    func hide() {
        window?.orderOut(nil)
    }

    func showError(_ message: String) {
        errorLabel?.stringValue = message
        errorLabel?.isHidden = false
    }

    func clearError() {
        errorLabel?.stringValue = ""
        errorLabel?.isHidden = true
    }

    func refreshLocalizedText() {
        window?.title = VibeCatL10n.onboardingWindowTitle()
        titleLabel?.stringValue = VibeCatL10n.onboardingTitle()
        subtitleLabel?.stringValue = VibeCatL10n.onboardingSubtitle()
        connectButton?.title = VibeCatL10n.buttonConnect()
        cancelButton?.title = VibeCatL10n.buttonCancel()
    }

    // MARK: - Window Construction

    private func buildWindow() {
        let width: CGFloat = 400
        let height: CGFloat = 190
        let rect = NSRect(x: 0, y: 0, width: width, height: height)

        let w = NSWindow(
            contentRect: rect,
            styleMask: [.titled, .closable],
            backing: .buffered,
            defer: false
        )
        w.title = VibeCatL10n.onboardingWindowTitle()
        w.level = .floating
        w.isReleasedWhenClosed = false
        w.hidesOnDeactivate = false
        w.center()

        guard let contentView = w.contentView else { return }

        let titleLabel = NSTextField(labelWithString: VibeCatL10n.onboardingTitle())
        titleLabel.font = NSFont.boldSystemFont(ofSize: 14)
        titleLabel.frame = NSRect(x: 20, y: 150, width: 360, height: 24)
        contentView.addSubview(titleLabel)
        self.titleLabel = titleLabel

        let subtitleLabel = NSTextField(labelWithString: VibeCatL10n.onboardingSubtitle())
        subtitleLabel.font = NSFont.systemFont(ofSize: 11)
        subtitleLabel.textColor = .secondaryLabelColor
        subtitleLabel.frame = NSRect(x: 20, y: 126, width: 360, height: 32)
        subtitleLabel.lineBreakMode = .byWordWrapping
        subtitleLabel.maximumNumberOfLines = 2
        contentView.addSubview(subtitleLabel)
        self.subtitleLabel = subtitleLabel

        let errLabel = NSTextField(labelWithString: "")
        errLabel.font = NSFont.systemFont(ofSize: 11)
        errLabel.textColor = .systemRed
        errLabel.frame = NSRect(x: 20, y: 82, width: 360, height: 18)
        errLabel.isHidden = true
        contentView.addSubview(errLabel)
        self.errorLabel = errLabel

        let btn = NSButton(title: VibeCatL10n.buttonConnect(), target: self, action: #selector(connectPressed))
        btn.bezelStyle = .rounded
        btn.keyEquivalent = "\r"
        btn.frame = NSRect(x: 290, y: 20, width: 90, height: 32)
        contentView.addSubview(btn)
        self.connectButton = btn

        let cancelBtn = NSButton(title: VibeCatL10n.buttonCancel(), target: self, action: #selector(cancelPressed))
        cancelBtn.bezelStyle = .rounded
        cancelBtn.frame = NSRect(x: 190, y: 20, width: 90, height: 32)
        contentView.addSubview(cancelBtn)
        self.cancelButton = cancelBtn

        self.window = w
    }

    @objc private func connectPressed() {
        clearError()
        hide()
        onConnect?()
    }

    @objc private func cancelPressed() {
        hide()
    }
}
