import AppKit
import ApplicationServices
import Carbon.HIToolbox
import Foundation
import VibeCatCore

struct NavigatorExecutionResult: Sendable {
    let status: String
    let observedOutcome: String
}

@MainActor
final class AccessibilityNavigator {
    func currentContext() -> NavigatorContextPayload {
        let trusted = AXIsProcessTrusted()
        let frontApp = NSWorkspace.shared.frontmostApplication
        let appName = frontApp?.localizedName ?? ""
        let bundleId = frontApp?.bundleIdentifier ?? ""

        guard trusted, let appElement = focusedApplicationElement() else {
            return NavigatorContextPayload(
                appName: appName,
                bundleId: bundleId,
                windowTitle: "",
                focusedRole: "",
                focusedLabel: "",
                selectedText: "",
                axSnapshot: "",
                accessibilityTrusted: trusted
            )
        }

        let window = focusedWindowElement(from: appElement)
        let focusedElement = focusedUIElement(from: appElement)
        let windowTitle = stringValue(for: window, attribute: kAXTitleAttribute) ?? ""
        let focusedRole = stringValue(for: focusedElement, attribute: kAXRoleAttribute) ?? ""
        let focusedLabel = bestLabel(for: focusedElement)
        let selectedText = stringValue(for: focusedElement, attribute: kAXSelectedTextAttribute)
            ?? stringValue(for: focusedElement, attribute: kAXValueAttribute)
            ?? ""
        let snapshot = summarize(window: window, focusedElement: focusedElement)

        return NavigatorContextPayload(
            appName: appName,
            bundleId: bundleId,
            windowTitle: windowTitle,
            focusedRole: focusedRole,
            focusedLabel: focusedLabel,
            selectedText: selectedText,
            axSnapshot: snapshot,
            accessibilityTrusted: trusted
        )
    }

    func execute(step: NavigatorStep) async -> NavigatorExecutionResult {
        let before = currentContext()

        switch step.actionType {
        case .focusApp:
            if focusApp(named: step.targetApp) {
                try? await Task.sleep(nanoseconds: 300_000_000)
                return verify(step: step, before: before, defaultOutcome: "Focused \(step.targetApp)")
            }
            return NavigatorExecutionResult(status: "failed", observedOutcome: "Could not focus \(step.targetApp)")

        case .openURL:
            guard let rawURL = step.url, let url = URL(string: rawURL) else {
                return NavigatorExecutionResult(status: "failed", observedOutcome: "Missing URL")
            }
            if open(url: url, targetApp: step.targetApp) {
                try? await Task.sleep(nanoseconds: 600_000_000)
                return verify(step: step, before: before, defaultOutcome: "Opened \(url.absoluteString)")
            }
            return NavigatorExecutionResult(status: "failed", observedOutcome: "Could not open URL")

        case .hotkey:
            guard AXIsProcessTrusted() else {
                return NavigatorExecutionResult(status: "guided_mode", observedOutcome: "Accessibility permission is required for hotkeys")
            }
            if sendHotkey(step.hotkey) {
                try? await Task.sleep(nanoseconds: 350_000_000)
                return verify(step: step, before: before, defaultOutcome: "Sent hotkey \(step.hotkey.joined(separator: "+"))")
            }
            return NavigatorExecutionResult(status: "failed", observedOutcome: "Could not send hotkey")

        case .pasteText:
            guard AXIsProcessTrusted() else {
                return NavigatorExecutionResult(status: "guided_mode", observedOutcome: "Accessibility permission is required for text entry")
            }
            guard let inputText = step.inputText, !inputText.isEmpty else {
                return NavigatorExecutionResult(status: "failed", observedOutcome: "Missing input text")
            }
            if pasteText(inputText) {
                try? await Task.sleep(nanoseconds: 350_000_000)
                return verify(step: step, before: before, defaultOutcome: "Inserted text")
            }
            return NavigatorExecutionResult(status: "failed", observedOutcome: "Could not insert text")

        case .copySelection:
            guard AXIsProcessTrusted() else {
                return NavigatorExecutionResult(status: "guided_mode", observedOutcome: "Accessibility permission is required for copy")
            }
            if sendHotkey(["command", "c"]) {
                try? await Task.sleep(nanoseconds: 250_000_000)
                return verify(step: step, before: before, defaultOutcome: "Copied the current selection")
            }
            return NavigatorExecutionResult(status: "failed", observedOutcome: "Could not copy the selection")

        case .pressAX:
            guard AXIsProcessTrusted() else {
                return NavigatorExecutionResult(status: "guided_mode", observedOutcome: "Accessibility permission is required for UI actions")
            }
            guard let element = resolveElement(for: step.targetDescriptor) else {
                return NavigatorExecutionResult(status: "guided_mode", observedOutcome: "I found the likely target, but I should not click blindly here.")
            }
            if AXUIElementPerformAction(element, kAXPressAction as CFString) == .success {
                try? await Task.sleep(nanoseconds: 350_000_000)
                return verify(step: step, before: before, defaultOutcome: "Pressed the target control")
            }
            return NavigatorExecutionResult(status: "guided_mode", observedOutcome: "The target was visible, but it was not safely pressable.")

        case .waitFor:
            try? await Task.sleep(nanoseconds: 600_000_000)
            return verify(step: step, before: before, defaultOutcome: "Observed the next UI state")
        }
    }

    private func verify(step: NavigatorStep, before: NavigatorContextPayload, defaultOutcome: String) -> NavigatorExecutionResult {
        let after = currentContext()
        let verifyHint = (step.verifyHint ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        let targetApp = step.targetApp.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        let appMatches = targetApp.isEmpty || after.appName.lowercased().contains(targetApp)

        if !verifyHint.isEmpty {
            let haystack = [
                after.windowTitle,
                after.focusedLabel,
                after.selectedText,
                after.axSnapshot
            ].joined(separator: "\n").lowercased()
            if haystack.contains(verifyHint.lowercased()) {
                return NavigatorExecutionResult(status: "success", observedOutcome: defaultOutcome)
            }
        }

        let contextChanged = after.windowTitle != before.windowTitle
            || after.focusedLabel != before.focusedLabel
            || after.selectedText != before.selectedText
            || after.axSnapshot != before.axSnapshot

        if appMatches && (contextChanged || step.actionType == .focusApp || step.actionType == .openURL) {
            return NavigatorExecutionResult(status: "success", observedOutcome: defaultOutcome)
        }

        if appMatches {
            return NavigatorExecutionResult(status: "success", observedOutcome: defaultOutcome)
        }

        return NavigatorExecutionResult(status: "guided_mode", observedOutcome: "The action ran, but I could not verify the target state safely.")
    }

    private func focusApp(named targetApp: String) -> Bool {
        let trimmed = targetApp.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return false }

        if let running = NSWorkspace.shared.runningApplications.first(where: {
            ($0.localizedName ?? "").caseInsensitiveCompare(trimmed) == .orderedSame
        }) {
            return running.activate(options: [.activateAllWindows])
        }

        let bundleIdentifier = explicitBundleIdentifier(for: targetApp) ?? explicitBundleIdentifier(for: trimmed)
        if let bundleIdentifier,
           let appURL = NSWorkspace.shared.urlForApplication(withBundleIdentifier: bundleIdentifier) {
            let config = NSWorkspace.OpenConfiguration()
            NSWorkspace.shared.openApplication(at: appURL, configuration: config) { _, _ in }
            return true
        }

        return false
    }

    private func open(url: URL, targetApp: String) -> Bool {
        if let appURL = chromeURLIfPreferred(for: targetApp) {
            let configuration = NSWorkspace.OpenConfiguration()
            NSWorkspace.shared.open([url], withApplicationAt: appURL, configuration: configuration) { _, _ in }
            return true
        }
        return NSWorkspace.shared.open(url)
    }

    private func chromeURLIfPreferred(for targetApp: String) -> URL? {
        let lowered = targetApp.lowercased()
        guard lowered.contains("chrome") || lowered.contains("browser") else { return nil }
        return NSWorkspace.shared.urlForApplication(withBundleIdentifier: "com.google.Chrome")
    }

    private func explicitBundleIdentifier(for targetApp: String) -> String? {
        let trimmed = targetApp.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.contains(".") {
            return trimmed
        }

        switch trimmed.lowercased() {
        case "chrome", "google chrome":
            return "com.google.Chrome"
        case "terminal", "terminal.app":
            return "com.apple.Terminal"
        case "safari":
            return "com.apple.Safari"
        default:
            return nil
        }
    }

    private func pasteText(_ text: String) -> Bool {
        let pasteboard = NSPasteboard.general
        let previous = pasteboard.string(forType: .string)
        pasteboard.clearContents()
        pasteboard.setString(text, forType: .string)
        let didSend = sendHotkey(["command", "v"])
        if let previous {
            pasteboard.clearContents()
            pasteboard.setString(previous, forType: .string)
        }
        return didSend
    }

    private func sendHotkey(_ tokens: [String]) -> Bool {
        guard let keyToken = tokens.last?.lowercased(),
              let keyCode = keyCode(for: keyToken) else {
            return false
        }

        let modifiers = tokens.dropLast().reduce(CGEventFlags()) { partial, token in
            partial.union(eventFlag(for: token.lowercased()))
        }

        guard let source = CGEventSource(stateID: .combinedSessionState),
              let keyDown = CGEvent(keyboardEventSource: source, virtualKey: keyCode, keyDown: true),
              let keyUp = CGEvent(keyboardEventSource: source, virtualKey: keyCode, keyDown: false) else {
            return false
        }

        keyDown.flags = modifiers
        keyUp.flags = modifiers
        keyDown.post(tap: .cghidEventTap)
        keyUp.post(tap: .cghidEventTap)
        return true
    }

    private func eventFlag(for token: String) -> CGEventFlags {
        switch token {
        case "command", "cmd":
            return .maskCommand
        case "shift":
            return .maskShift
        case "option", "alt":
            return .maskAlternate
        case "control", "ctrl":
            return .maskControl
        default:
            return []
        }
    }

    private func keyCode(for token: String) -> CGKeyCode? {
        switch token {
        case "a": return CGKeyCode(kVK_ANSI_A)
        case "c": return CGKeyCode(kVK_ANSI_C)
        case "e": return CGKeyCode(kVK_ANSI_E)
        case "i": return CGKeyCode(kVK_ANSI_I)
        case "l": return CGKeyCode(kVK_ANSI_L)
        case "p": return CGKeyCode(kVK_ANSI_P)
        case "r": return CGKeyCode(kVK_ANSI_R)
        case "v": return CGKeyCode(kVK_ANSI_V)
        case "grave", "`": return CGKeyCode(kVK_ANSI_Grave)
        case "return", "enter": return CGKeyCode(kVK_Return)
        case "tab": return CGKeyCode(kVK_Tab)
        default: return nil
        }
    }

    private func resolveElement(for descriptor: NavigatorTargetDescriptor) -> AXUIElement? {
        guard let app = focusedApplicationElement() else { return nil }
        let roots = [focusedWindowElement(from: app), focusedUIElement(from: app), app].compactMap { $0 }
        let windowTitle = descriptor.windowTitle?.lowercased()
        let desiredRole = descriptor.role?.lowercased()
        let desiredLabel = descriptor.label?.lowercased()

        for root in roots {
            for element in breadthFirstSearch(from: root, maxDepth: 5, maxNodes: 80) {
                if let windowTitle,
                   let currentWindow = focusedWindowElement(from: app),
                   let currentTitle = stringValue(for: currentWindow, attribute: kAXTitleAttribute)?.lowercased(),
                   !currentTitle.contains(windowTitle) {
                    continue
                }
                if let desiredRole,
                   let role = stringValue(for: element, attribute: kAXRoleAttribute)?.lowercased(),
                   !role.contains(desiredRole) {
                    continue
                }
                if let desiredLabel {
                    let label = bestLabel(for: element).lowercased()
                    if !label.contains(desiredLabel) {
                        continue
                    }
                }
                return element
            }
        }

        return nil
    }

    private func summarize(window: AXUIElement?, focusedElement: AXUIElement?) -> String {
        var lines: [String] = []
        if let window, let title = stringValue(for: window, attribute: kAXTitleAttribute), !title.isEmpty {
            lines.append("window:\(title)")
        }
        if let focusedElement {
            let focusedSummary = summarize(element: focusedElement)
            if !focusedSummary.isEmpty {
                lines.append("focused:\(focusedSummary)")
            }
        }
        if let window {
            for element in breadthFirstSearch(from: window, maxDepth: 2, maxNodes: 18) {
                let summary = summarize(element: element)
                if !summary.isEmpty {
                    lines.append(summary)
                }
            }
        }
        return Array(lines.prefix(20)).joined(separator: "\n")
    }

    private func summarize(element: AXUIElement) -> String {
        let role = stringValue(for: element, attribute: kAXRoleAttribute) ?? ""
        let label = bestLabel(for: element)
        let parts = [role, label].filter { !$0.isEmpty }
        return parts.joined(separator: ":")
    }

    private func bestLabel(for element: AXUIElement?) -> String {
        guard let element else { return "" }
        return stringValue(for: element, attribute: kAXTitleAttribute)
            ?? stringValue(for: element, attribute: kAXDescriptionAttribute)
            ?? stringValue(for: element, attribute: kAXValueAttribute)
            ?? ""
    }

    private func focusedApplicationElement() -> AXUIElement? {
        let system = AXUIElementCreateSystemWide()
        return attributeElement(for: system, attribute: kAXFocusedApplicationAttribute)
    }

    private func focusedWindowElement(from app: AXUIElement) -> AXUIElement? {
        attributeElement(for: app, attribute: kAXFocusedWindowAttribute)
    }

    private func focusedUIElement(from app: AXUIElement) -> AXUIElement? {
        attributeElement(for: app, attribute: kAXFocusedUIElementAttribute)
    }

    private func attributeElement(for element: AXUIElement, attribute: String) -> AXUIElement? {
        guard let value = attributeValue(for: element, attribute: attribute) else {
            return nil
        }
        return axElement(from: value)
    }

    private func stringValue(for element: AXUIElement?, attribute: String) -> String? {
        guard let element,
              let value = attributeValue(for: element, attribute: attribute) else {
            return nil
        }
        if let string = value as? String {
            let trimmed = string.trimmingCharacters(in: .whitespacesAndNewlines)
            return trimmed.isEmpty ? nil : trimmed
        }
        if let attributed = value as? NSAttributedString {
            let trimmed = attributed.string.trimmingCharacters(in: .whitespacesAndNewlines)
            return trimmed.isEmpty ? nil : trimmed
        }
        if let number = value as? NSNumber {
            return number.stringValue
        }
        return nil
    }

    private func attributeValue(for element: AXUIElement, attribute: String) -> CFTypeRef? {
        var value: CFTypeRef?
        guard AXUIElementCopyAttributeValue(element, attribute as CFString, &value) == .success,
              let value else {
            return nil
        }
        return value
    }

    private func axElement(from value: CFTypeRef) -> AXUIElement? {
        guard CFGetTypeID(value) == AXUIElementGetTypeID() else {
            return nil
        }
        return unsafeDowncast(value, to: AXUIElement.self)
    }

    private func childElements(for element: AXUIElement) -> [AXUIElement] {
        guard let value = attributeValue(for: element, attribute: kAXChildrenAttribute),
              let array = value as? [Any] else {
            return []
        }
        return array.compactMap { item in
            axElement(from: item as CFTypeRef)
        }
    }

    private func breadthFirstSearch(from root: AXUIElement, maxDepth: Int, maxNodes: Int) -> [AXUIElement] {
        var queue: [(AXUIElement, Int)] = [(root, 0)]
        var output: [AXUIElement] = []

        while !queue.isEmpty && output.count < maxNodes {
            let (current, depth) = queue.removeFirst()
            output.append(current)
            guard depth < maxDepth else { continue }
            for child in childElements(for: current) {
                queue.append((child, depth + 1))
            }
        }

        return output
    }
}
