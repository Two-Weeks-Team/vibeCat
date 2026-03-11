import AppKit
import VibeCatCore

private final class ChatInputPanel: NSPanel {
    override var canBecomeKey: Bool { true }
    override var canBecomeMain: Bool { true }
}

@MainActor
final class CompanionChatPanel: NSObject, NSTextFieldDelegate, NSWindowDelegate {
    var onTextSubmitted: ((String) -> Void)?
    var onDismissed: (() -> Void)?

    var isVisible: Bool { panel?.isVisible ?? false }

    private var panel: NSPanel?
    private let panelSize = NSSize(width: 360, height: 360)

    private let scrollView = NSScrollView()
    private let messagesStack = NSStackView()
    private let inputField = NSTextField()
    private var messageViews: [(isUser: Bool, textField: NSTextField, bubble: NSView)] = []

    func show(near catPosition: CGPoint, on screen: NSScreen?) {
        let targetScreen = screen ?? NSScreen.main ?? NSScreen.screens.first
        guard let targetScreen else { return }

        if panel == nil {
            createPanel()
        }

        let frame = targetScreen.frame
        let x = min(max(catPosition.x - panelSize.width / 2, frame.minX + 12), frame.maxX - panelSize.width - 12)
        let y = min(max(catPosition.y + 70, frame.minY + 12), frame.maxY - panelSize.height - 12)
        panel?.setFrame(NSRect(origin: NSPoint(x: x, y: y), size: panelSize), display: true)
        panel?.makeKeyAndOrderFront(nil)
        panel?.orderFrontRegardless()
        panel?.makeFirstResponder(inputField)
    }

    func dismiss() {
        panel?.orderOut(nil)
        clearConversation()
        onDismissed?()
    }

    func clearConversation() {
        messageViews.removeAll()
        messagesStack.arrangedSubviews.forEach { view in
            messagesStack.removeArrangedSubview(view)
            view.removeFromSuperview()
        }
        inputField.stringValue = ""
    }

    func refreshLocalizedText() {
        inputField.placeholderString = VibeCatL10n.companionInputPlaceholder()
    }

    func addUserMessage(_ text: String) {
        appendMessage(text: text, isUser: true)
    }

    func addAssistantMessage(_ text: String) {
        appendMessage(text: text, isUser: false)
    }

    func addLoadingPlaceholder() {
        appendMessage(text: "...", isUser: false)
    }

    func updateLastAssistantMessage(_ text: String) {
        guard let idx = messageViews.lastIndex(where: { !$0.isUser }) else {
            addAssistantMessage(text)
            return
        }
        let entry = messageViews[idx]
        entry.textField.stringValue = text
        layoutMessage(entry.textField, in: entry.bubble)
        scrollToBottom()
    }

    func updateListeningText(_ text: String) {
        let prefix = VibeCatL10n.companionListeningPrefix()
        let prefixed = prefix + text
        if let idx = messageViews.lastIndex(where: { $0.isUser && $0.textField.stringValue.hasPrefix(prefix) }) {
            let entry = messageViews[idx]
            entry.textField.stringValue = prefixed
            layoutMessage(entry.textField, in: entry.bubble)
        } else {
            appendMessage(text: prefixed, isUser: true)
        }
        scrollToBottom()
    }

    func finalizeListeningText(_ text: String) {
        let prefix = VibeCatL10n.companionListeningPrefix()
        if let idx = messageViews.lastIndex(where: { $0.isUser && $0.textField.stringValue.hasPrefix(prefix) }) {
            let entry = messageViews[idx]
            entry.textField.stringValue = text
            layoutMessage(entry.textField, in: entry.bubble)
            scrollToBottom()
            return
        }
        addUserMessage(text)
    }

    func control(_ control: NSControl, textView: NSTextView, doCommandBy commandSelector: Selector) -> Bool {
        if commandSelector == #selector(NSResponder.insertNewline(_:)) {
            let text = inputField.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !text.isEmpty else { return true }
            inputField.stringValue = ""
            onTextSubmitted?(text)
            return true
        }
        return false
    }

    func windowWillClose(_ notification: Notification) {
        onDismissed?()
    }

    private func createPanel() {
        let panel = ChatInputPanel(
            contentRect: NSRect(origin: .zero, size: panelSize),
            styleMask: [.titled, .closable, .fullSizeContentView],
            backing: .buffered,
            defer: false
        )

        panel.isOpaque = false
        panel.backgroundColor = NSColor(calibratedWhite: 0.12, alpha: 0.92)
        panel.hasShadow = true
        panel.level = .floating
        panel.collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary]
        panel.ignoresMouseEvents = false
        panel.isMovableByWindowBackground = true
        panel.titleVisibility = .hidden
        panel.titlebarAppearsTransparent = true
        panel.delegate = self

        let content = NSView(frame: NSRect(origin: .zero, size: panelSize))
        content.wantsLayer = true
        content.layer?.cornerRadius = 12
        content.layer?.masksToBounds = true

        scrollView.frame = NSRect(x: 12, y: 56, width: panelSize.width - 24, height: panelSize.height - 68)
        scrollView.hasVerticalScroller = true
        scrollView.drawsBackground = false
        scrollView.borderType = .noBorder

        let document = NSView(frame: NSRect(x: 0, y: 0, width: scrollView.frame.width, height: 1))
        messagesStack.orientation = .vertical
        messagesStack.alignment = .leading
        messagesStack.spacing = 8
        messagesStack.translatesAutoresizingMaskIntoConstraints = false
        document.addSubview(messagesStack)

        NSLayoutConstraint.activate([
            messagesStack.leadingAnchor.constraint(equalTo: document.leadingAnchor),
            messagesStack.trailingAnchor.constraint(equalTo: document.trailingAnchor),
            messagesStack.topAnchor.constraint(equalTo: document.topAnchor),
            messagesStack.bottomAnchor.constraint(equalTo: document.bottomAnchor),
            messagesStack.widthAnchor.constraint(equalTo: document.widthAnchor)
        ])

        scrollView.documentView = document

        inputField.frame = NSRect(x: 12, y: 12, width: panelSize.width - 24, height: 32)
        refreshLocalizedText()
        inputField.focusRingType = .none
        inputField.delegate = self

        content.addSubview(scrollView)
        content.addSubview(inputField)
        panel.contentView = content
        self.panel = panel
    }

    private func appendMessage(text: String, isUser: Bool) {
        guard let document = scrollView.documentView else { return }

        let container = NSView(frame: NSRect(x: 0, y: 0, width: scrollView.frame.width, height: 1))
        container.translatesAutoresizingMaskIntoConstraints = false
        let bubble = NSView()
        bubble.wantsLayer = true
        bubble.layer?.cornerRadius = 10
        bubble.layer?.backgroundColor = (isUser ? NSColor.systemBlue : NSColor(calibratedWhite: 0.25, alpha: 1)).cgColor
        bubble.translatesAutoresizingMaskIntoConstraints = false

        let label = NSTextField(wrappingLabelWithString: text)
        label.textColor = .white
        label.font = NSFont.systemFont(ofSize: 13)
        label.translatesAutoresizingMaskIntoConstraints = false

        bubble.addSubview(label)
        container.addSubview(bubble)

        NSLayoutConstraint.activate([
            bubble.widthAnchor.constraint(lessThanOrEqualToConstant: 250),
            bubble.topAnchor.constraint(equalTo: container.topAnchor),
            bubble.bottomAnchor.constraint(equalTo: container.bottomAnchor),
            label.leadingAnchor.constraint(equalTo: bubble.leadingAnchor, constant: 10),
            label.trailingAnchor.constraint(equalTo: bubble.trailingAnchor, constant: -10),
            label.topAnchor.constraint(equalTo: bubble.topAnchor, constant: 8),
            label.bottomAnchor.constraint(equalTo: bubble.bottomAnchor, constant: -8)
        ])

        if isUser {
            bubble.trailingAnchor.constraint(equalTo: container.trailingAnchor, constant: -4).isActive = true
        } else {
            bubble.leadingAnchor.constraint(equalTo: container.leadingAnchor, constant: 4).isActive = true
        }

        messagesStack.addArrangedSubview(container)
        messageViews.append((isUser: isUser, textField: label, bubble: bubble))

        let fittingHeight = container.fittingSize.height
        var docFrame = document.frame
        docFrame.size.height += max(1, fittingHeight + messagesStack.spacing)
        document.frame = docFrame
        scrollToBottom()
    }

    private func layoutMessage(_ label: NSTextField, in bubble: NSView) {
        bubble.needsLayout = true
        label.needsLayout = true
        bubble.layoutSubtreeIfNeeded()
    }

    private func scrollToBottom() {
        guard let document = scrollView.documentView else { return }
        document.layoutSubtreeIfNeeded()
        let maxY = max(0, document.bounds.height - scrollView.contentView.bounds.height)
        scrollView.contentView.scroll(to: NSPoint(x: 0, y: maxY))
        scrollView.reflectScrolledClipView(scrollView.contentView)
    }
}
