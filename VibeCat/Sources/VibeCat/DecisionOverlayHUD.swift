import AppKit
import VibeCatCore

@MainActor
final class DecisionOverlayHUD: NSPanel {
    private let stackView = NSStackView()
    private var hudVisible = false

    private let triggerLabel = NSTextField(labelWithString: "Trigger: —")
    private let visionLabel = NSTextField(labelWithString: "Vision: —")
    private let mediatorLabel = NSTextField(labelWithString: "Mediator: —")
    private let moodLabel = NSTextField(labelWithString: "Mood: —")
    private let cooldownLabel = NSTextField(labelWithString: "Cooldown: —")

    init() {
        let size = NSSize(width: 280, height: 160)
        let screen = NSScreen.main?.frame ?? NSRect(x: 0, y: 0, width: 1440, height: 900)
        let origin = NSPoint(x: screen.minX + 20, y: screen.maxY - size.height - 40)

        super.init(
            contentRect: NSRect(origin: origin, size: size),
            styleMask: [.borderless, .nonactivatingPanel],
            backing: .buffered,
            defer: false
        )

        level = .floating
        backgroundColor = NSColor.black.withAlphaComponent(0.8)
        isOpaque = false
        hasShadow = true
        collectionBehavior = [.canJoinAllSpaces, .stationary]
        ignoresMouseEvents = true
        isReleasedWhenClosed = false

        setupViews()
        orderOut(nil)
    }

    private func setupViews() {
        guard let contentView else { return }
        contentView.wantsLayer = true
        contentView.layer?.cornerRadius = 10

        stackView.orientation = .vertical
        stackView.alignment = .leading
        stackView.spacing = 6
        stackView.edgeInsets = NSEdgeInsets(top: 12, left: 12, bottom: 12, right: 12)
        stackView.translatesAutoresizingMaskIntoConstraints = false

        for label in [triggerLabel, visionLabel, mediatorLabel, moodLabel, cooldownLabel] {
            label.font = NSFont.monospacedSystemFont(ofSize: 11, weight: .regular)
            label.textColor = .white
            label.backgroundColor = .clear
            stackView.addArrangedSubview(label)
        }

        contentView.addSubview(stackView)
        NSLayoutConstraint.activate([
            stackView.leadingAnchor.constraint(equalTo: contentView.leadingAnchor),
            stackView.trailingAnchor.constraint(equalTo: contentView.trailingAnchor),
            stackView.topAnchor.constraint(equalTo: contentView.topAnchor),
            stackView.bottomAnchor.constraint(equalTo: contentView.bottomAnchor)
        ])
    }

    func toggle() {
        if hudVisible {
            orderOut(nil)
            hudVisible = false
        } else {
            orderFront(nil)
            hudVisible = true
        }
    }

    func update(trigger: String, vision: String, mediator: String, mood: String, cooldown: String) {
        triggerLabel.stringValue = "Trigger: \(trigger)"
        visionLabel.stringValue = "Vision: \(vision)"
        mediatorLabel.stringValue = "Mediator: \(mediator)"
        moodLabel.stringValue = "Mood: \(mood)"
        cooldownLabel.stringValue = "Cooldown: \(cooldown)"
    }
}
