import AppKit
import VibeCatCore

@MainActor
final class DecisionOverlayHUD: NSPanel {
    private let stackView = NSStackView()
    private var hudVisible = false

    private let triggerLabel = NSTextField(labelWithString: VibeCatL10n.decisionTrigger("—"))
    private let visionLabel = NSTextField(labelWithString: VibeCatL10n.decisionVision("—"))
    private let mediatorLabel = NSTextField(labelWithString: VibeCatL10n.decisionMediator("—"))
    private let moodLabel = NSTextField(labelWithString: VibeCatL10n.decisionMood("—"))
    private let cooldownLabel = NSTextField(labelWithString: VibeCatL10n.decisionCooldown("—"))
    private let hudSize = NSSize(width: 280, height: 160)

    init() {
        let screen = NSScreen.main?.frame ?? NSRect(x: 0, y: 0, width: 1440, height: 900)
        let origin = NSPoint(x: screen.midX + 40, y: screen.midY - 80)

        super.init(
            contentRect: NSRect(origin: origin, size: hudSize),
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

    func updatePosition(catScreenPosition: CGPoint, screenFrame: NSRect) {
        guard hudVisible else { return }
        let x = min(catScreenPosition.x + 50, screenFrame.maxX - hudSize.width - 8)
        let y = max(screenFrame.minY + 8, min(catScreenPosition.y - hudSize.height / 2, screenFrame.maxY - hudSize.height - 8))
        setFrameOrigin(NSPoint(x: x, y: y))
    }

    func update(trigger: String, vision: String, mediator: String, mood: String, cooldown: String) {
        triggerLabel.stringValue = VibeCatL10n.decisionTrigger(trigger)
        visionLabel.stringValue = VibeCatL10n.decisionVision(vision)
        mediatorLabel.stringValue = VibeCatL10n.decisionMediator(mediator)
        moodLabel.stringValue = VibeCatL10n.decisionMood(mood)
        cooldownLabel.stringValue = VibeCatL10n.decisionCooldown(cooldown)
    }
}
