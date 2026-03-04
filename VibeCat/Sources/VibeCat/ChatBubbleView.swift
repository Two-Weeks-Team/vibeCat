import AppKit
import VibeCatCore

@MainActor
final class ChatBubbleView: NSView {
    private let label = NSTextField(wrappingLabelWithString: "")
    private var hideTimer: Timer?

    override init(frame: NSRect) {
        super.init(frame: frame)
        setup()
    }

    required init?(coder: NSCoder) {
        super.init(coder: coder)
        setup()
    }

    private func setup() {
        wantsLayer = true
        layer?.backgroundColor = NSColor.black.withAlphaComponent(0.75).cgColor
        layer?.cornerRadius = 12

        label.textColor = .white
        label.font = NSFont.systemFont(ofSize: 13)
        label.backgroundColor = .clear
        label.isBezeled = false
        label.isEditable = false
        label.translatesAutoresizingMaskIntoConstraints = false
        addSubview(label)

        NSLayoutConstraint.activate([
            label.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 10),
            label.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -10),
            label.topAnchor.constraint(equalTo: topAnchor, constant: 8),
            label.bottomAnchor.constraint(equalTo: bottomAnchor, constant: -8)
        ])

        isHidden = true
    }

    func show(text: String, autohideAfter seconds: TimeInterval = 6.0) {
        label.stringValue = text
        isHidden = false
        alphaValue = 1.0

        hideTimer?.invalidate()
        hideTimer = Timer.scheduledTimer(withTimeInterval: seconds, repeats: false) { [weak self] _ in
            Task { @MainActor [weak self] in
                self?.hide()
            }
        }
    }

    func hide() {
        NSAnimationContext.runAnimationGroup { ctx in
            ctx.duration = 0.3
            self.animator().alphaValue = 0
        } completionHandler: {
            self.isHidden = true
            self.alphaValue = 1.0
        }
    }
}
