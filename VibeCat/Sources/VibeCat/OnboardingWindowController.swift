import AppKit
import VibeCatCore

/// First-launch onboarding panel for API key entry.
/// Stores the key in Keychain on successful submission.
@MainActor
final class OnboardingWindowController: NSObject {
    private var window: NSWindow?
    private var apiKeyField: NSSecureTextField?
    private var errorLabel: NSTextField?
    private var connectButton: NSButton?

    var onConnect: ((String) -> Void)?

    func show() {
        if window == nil {
            buildWindow()
        }
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

    // MARK: - Window Construction

    private func buildWindow() {
        let width: CGFloat = 400
        let height: CGFloat = 200
        let rect = NSRect(x: 0, y: 0, width: width, height: height)

        let w = NSWindow(
            contentRect: rect,
            styleMask: [.titled, .closable],
            backing: .buffered,
            defer: false
        )
        w.title = "VibeCat — Connect"
        w.level = .floating
        w.isReleasedWhenClosed = false
        w.hidesOnDeactivate = false
        w.center()

        guard let contentView = w.contentView else { return }

        let titleLabel = NSTextField(labelWithString: "Enter your Gemini API Key")
        titleLabel.font = NSFont.boldSystemFont(ofSize: 14)
        titleLabel.frame = NSRect(x: 20, y: 150, width: 360, height: 24)
        contentView.addSubview(titleLabel)

        let subtitleLabel = NSTextField(labelWithString: "Your key is stored securely in Keychain and never sent to third parties.")
        subtitleLabel.font = NSFont.systemFont(ofSize: 11)
        subtitleLabel.textColor = .secondaryLabelColor
        subtitleLabel.frame = NSRect(x: 20, y: 128, width: 360, height: 18)
        contentView.addSubview(subtitleLabel)

        let field = NSSecureTextField(frame: NSRect(x: 20, y: 90, width: 360, height: 28))
        field.placeholderString = "AIza..."
        field.target = self
        field.action = #selector(connectPressed)
        contentView.addSubview(field)
        self.apiKeyField = field

        let errLabel = NSTextField(labelWithString: "")
        errLabel.font = NSFont.systemFont(ofSize: 11)
        errLabel.textColor = .systemRed
        errLabel.frame = NSRect(x: 20, y: 68, width: 360, height: 18)
        errLabel.isHidden = true
        contentView.addSubview(errLabel)
        self.errorLabel = errLabel

        let btn = NSButton(title: "Connect", target: self, action: #selector(connectPressed))
        btn.bezelStyle = .rounded
        btn.keyEquivalent = "\r"
        btn.frame = NSRect(x: 290, y: 20, width: 90, height: 32)
        contentView.addSubview(btn)
        self.connectButton = btn

        let cancelBtn = NSButton(title: "Cancel", target: self, action: #selector(cancelPressed))
        cancelBtn.bezelStyle = .rounded
        cancelBtn.frame = NSRect(x: 190, y: 20, width: 90, height: 32)
        contentView.addSubview(cancelBtn)

        self.window = w
    }

    @objc private func connectPressed() {
        let key = apiKeyField?.stringValue.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !key.isEmpty else {
            showError("Please enter your API key.")
            return
        }
        clearError()
        // Store in Keychain
        do {
            try KeychainHelper.save(key, forKey: "vibecat-api-key")
        } catch {
            showError("Failed to save key: \(error.localizedDescription)")
            return
        }
        hide()
        onConnect?(key)
    }

    @objc private func cancelPressed() {
        hide()
    }
}
