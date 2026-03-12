import AppKit
import VibeCatCore

@MainActor
final class NavigatorOverlayPanel: NSPanel {
    private enum Layout {
        static let width: CGFloat = 260
        static let cornerRadius: CGFloat = 12
        static let padding: CGFloat = 10
        static let spacing: CGFloat = 4
        static let badgePadding: CGFloat = 6
        static let badgeCornerRadius: CGFloat = 6
        static let bottomMargin: CGFloat = 50
    }

    enum StepResult {
        case success
        case retry
        case failed
    }

    private let containerView = NSVisualEffectView()
    private let actionIconLabel = NSTextField(labelWithString: "")
    private let actionTextLabel = NSTextField(labelWithString: "")
    private let groundingBadge = NSTextField(labelWithString: "")
    private let groundingBadgeBackground = NSView()
    private let progressLabel = NSTextField(labelWithString: "")
    private let resultLabel = NSTextField(labelWithString: "")

    private var hideTask: Task<Void, Never>?
    private var currentTaskID: String?
    private var stepIndex = 0

    init() {
        let screen = NSScreen.main?.frame ?? NSRect(x: 0, y: 0, width: 1440, height: 900)
        let origin = NSPoint(
            x: screen.midX - Layout.width / 2,
            y: screen.minY + Layout.bottomMargin
        )

        super.init(
            contentRect: NSRect(origin: origin, size: NSSize(width: Layout.width, height: 60)),
            styleMask: [.borderless, .nonactivatingPanel],
            backing: .buffered,
            defer: false
        )

        level = .floating
        backgroundColor = .clear
        isOpaque = false
        hasShadow = true
        collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary, .stationary]
        ignoresMouseEvents = true
        isReleasedWhenClosed = false

        setupViews()
        orderOut(nil)
    }

    private func setupViews() {
        guard let contentView else { return }

        containerView.material = .hudWindow
        containerView.blendingMode = .behindWindow
        containerView.state = .active
        containerView.wantsLayer = true
        containerView.layer?.cornerRadius = Layout.cornerRadius
        containerView.layer?.masksToBounds = true
        containerView.layer?.backgroundColor = NSColor.black.withAlphaComponent(0.72).cgColor
        containerView.translatesAutoresizingMaskIntoConstraints = false
        contentView.addSubview(containerView)

        NSLayoutConstraint.activate([
            containerView.leadingAnchor.constraint(equalTo: contentView.leadingAnchor),
            containerView.trailingAnchor.constraint(equalTo: contentView.trailingAnchor),
            containerView.topAnchor.constraint(equalTo: contentView.topAnchor),
            containerView.bottomAnchor.constraint(equalTo: contentView.bottomAnchor),
        ])

        actionIconLabel.font = NSFont.systemFont(ofSize: 16)
        actionIconLabel.textColor = .white
        actionIconLabel.translatesAutoresizingMaskIntoConstraints = false

        actionTextLabel.font = NSFont.systemFont(ofSize: 12, weight: .medium)
        actionTextLabel.textColor = .white
        actionTextLabel.lineBreakMode = .byTruncatingTail
        actionTextLabel.translatesAutoresizingMaskIntoConstraints = false

        groundingBadgeBackground.wantsLayer = true
        groundingBadgeBackground.layer?.cornerRadius = Layout.badgeCornerRadius
        groundingBadgeBackground.translatesAutoresizingMaskIntoConstraints = false

        groundingBadge.font = NSFont.systemFont(ofSize: 9, weight: .bold)
        groundingBadge.textColor = .white
        groundingBadge.alignment = .center
        groundingBadge.translatesAutoresizingMaskIntoConstraints = false

        progressLabel.font = NSFont.monospacedDigitSystemFont(ofSize: 10, weight: .regular)
        progressLabel.textColor = NSColor.white.withAlphaComponent(0.7)
        progressLabel.translatesAutoresizingMaskIntoConstraints = false

        resultLabel.font = NSFont.systemFont(ofSize: 11, weight: .semibold)
        resultLabel.textColor = .white
        resultLabel.isHidden = true
        resultLabel.translatesAutoresizingMaskIntoConstraints = false

        groundingBadgeBackground.addSubview(groundingBadge)
        containerView.addSubview(actionIconLabel)
        containerView.addSubview(actionTextLabel)
        containerView.addSubview(groundingBadgeBackground)
        containerView.addSubview(progressLabel)
        containerView.addSubview(resultLabel)

        NSLayoutConstraint.activate([
            actionIconLabel.leadingAnchor.constraint(equalTo: containerView.leadingAnchor, constant: Layout.padding),
            actionIconLabel.centerYAnchor.constraint(equalTo: containerView.centerYAnchor),
            actionIconLabel.widthAnchor.constraint(equalToConstant: 22),

            actionTextLabel.leadingAnchor.constraint(equalTo: actionIconLabel.trailingAnchor, constant: Layout.spacing),
            actionTextLabel.topAnchor.constraint(equalTo: containerView.topAnchor, constant: Layout.padding),

            groundingBadgeBackground.leadingAnchor.constraint(equalTo: actionTextLabel.trailingAnchor, constant: Layout.spacing + 2),
            groundingBadgeBackground.centerYAnchor.constraint(equalTo: actionTextLabel.centerYAnchor),

            groundingBadge.leadingAnchor.constraint(equalTo: groundingBadgeBackground.leadingAnchor, constant: Layout.badgePadding),
            groundingBadge.trailingAnchor.constraint(equalTo: groundingBadgeBackground.trailingAnchor, constant: -Layout.badgePadding),
            groundingBadge.topAnchor.constraint(equalTo: groundingBadgeBackground.topAnchor, constant: 2),
            groundingBadge.bottomAnchor.constraint(equalTo: groundingBadgeBackground.bottomAnchor, constant: -2),

            progressLabel.leadingAnchor.constraint(equalTo: actionIconLabel.trailingAnchor, constant: Layout.spacing),
            progressLabel.topAnchor.constraint(equalTo: actionTextLabel.bottomAnchor, constant: 2),
            progressLabel.bottomAnchor.constraint(equalTo: containerView.bottomAnchor, constant: -Layout.padding),

            resultLabel.trailingAnchor.constraint(equalTo: containerView.trailingAnchor, constant: -Layout.padding),
            resultLabel.centerYAnchor.constraint(equalTo: progressLabel.centerYAnchor),

            actionTextLabel.trailingAnchor.constraint(lessThanOrEqualTo: groundingBadgeBackground.leadingAnchor, constant: -Layout.spacing),
            groundingBadgeBackground.trailingAnchor.constraint(lessThanOrEqualTo: containerView.trailingAnchor, constant: -Layout.padding),
        ])
    }

    func showStep(taskId: String, step: NavigatorStep, stepNumber: Int, totalSteps: Int?) {
        hideTask?.cancel()
        hideTask = nil

        if currentTaskID != taskId {
            currentTaskID = taskId
            stepIndex = 0
        }
        stepIndex = stepNumber

        let actionLabel = VibeCatL10n.navigatorActionLabel(actionType: step.actionType, step: step)
        let source = step.actionType.groundingSource(step: step)

        actionIconLabel.stringValue = step.actionType.icon
        actionTextLabel.stringValue = actionLabel
        groundingBadge.stringValue = source.badge
        applyGroundingBadgeColor(source)

        if let total = totalSteps, total > 1 {
            progressLabel.stringValue = VibeCatL10n.navigatorStepProgress(current: stepNumber, total: total)
            progressLabel.isHidden = false
        } else {
            progressLabel.stringValue = VibeCatL10n.navigatorStepProgress(current: stepNumber, total: stepNumber)
            progressLabel.isHidden = false
        }

        resultLabel.isHidden = true

        let screen = NSScreen.main?.frame ?? NSRect(x: 0, y: 0, width: 1440, height: 900)
        let panelOrigin = NSPoint(
            x: screen.midX - Layout.width / 2,
            y: screen.minY + Layout.bottomMargin
        )
        setFrame(NSRect(origin: panelOrigin, size: NSSize(width: Layout.width, height: 52)), display: true)

        if !isVisible {
            alphaValue = 0
            orderFront(nil)
            NSAnimationContext.runAnimationGroup { ctx in
                ctx.duration = 0.2
                self.animator().alphaValue = 1
            }
        }
    }

    func showResult(_ result: StepResult) {
        resultLabel.isHidden = false

        switch result {
        case .success:
            resultLabel.stringValue = VibeCatL10n.navigatorStepSuccess()
            resultLabel.textColor = NSColor.systemGreen
        case .retry:
            resultLabel.stringValue = VibeCatL10n.navigatorStepRetry()
            resultLabel.textColor = NSColor.systemOrange
        case .failed:
            resultLabel.stringValue = VibeCatL10n.navigatorStepFailed()
            resultLabel.textColor = NSColor.systemRed
        }

        scheduleHide(after: 2.5)
    }

    func showCompletion(success: Bool) {
        resultLabel.isHidden = false

        if success {
            actionIconLabel.stringValue = "\u{2705}"
            actionTextLabel.stringValue = VibeCatL10n.navigatorDoneTitle()
            resultLabel.stringValue = ""
            resultLabel.isHidden = true
        } else {
            actionIconLabel.stringValue = "\u{274C}"
            actionTextLabel.stringValue = VibeCatL10n.navigatorFailedTitle()
            resultLabel.stringValue = ""
            resultLabel.isHidden = true
        }

        progressLabel.isHidden = true
        groundingBadge.stringValue = ""
        groundingBadgeBackground.layer?.backgroundColor = NSColor.clear.cgColor

        scheduleHide(after: 2.0)
    }

    func dismiss() {
        hideTask?.cancel()
        hideTask = nil
        currentTaskID = nil
        stepIndex = 0

        NSAnimationContext.runAnimationGroup({ ctx in
            ctx.duration = 0.25
            self.animator().alphaValue = 0
        }, completionHandler: { [weak self] in
            Task { @MainActor in
                self?.orderOut(nil)
            }
        })
    }

    private func scheduleHide(after seconds: TimeInterval) {
        hideTask?.cancel()
        hideTask = Task { @MainActor [weak self] in
            try? await Task.sleep(nanoseconds: UInt64(seconds * 1_000_000_000))
            guard let self, !Task.isCancelled else { return }
            self.dismiss()
        }
    }

    private func applyGroundingBadgeColor(_ source: GroundingSource) {
        let color: NSColor
        switch source {
        case .ax:
            color = NSColor.systemBlue
        case .vision:
            color = NSColor.systemPurple
        case .hotkey:
            color = NSColor.systemGray
        case .system:
            color = NSColor.systemTeal
        }
        groundingBadgeBackground.layer?.backgroundColor = color.withAlphaComponent(0.8).cgColor
    }
}
