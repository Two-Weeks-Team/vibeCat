import AppKit
import ApplicationServices
import Carbon.HIToolbox
import Foundation
import VibeCatCore

struct NavigatorExecutionResult: Sendable {
    let status: String
    let observedOutcome: String
}

private struct PasteboardSnapshotItem: Sendable {
    let dataByType: [NSPasteboard.PasteboardType: Data]
}

@MainActor
final class AccessibilityNavigator {
    private var lastFocusSignature = ""
    private var focusStableSince = Date()
    private var lastInputFieldHint = ""
    private var lastInputFieldDescriptor = ""
    private var lastInputFieldWindow = ""
    private var lastInputFieldBundleID = ""

    func currentContext() -> NavigatorContextPayload {
        let trusted = AXIsProcessTrusted()
        let frontApp = NSWorkspace.shared.frontmostApplication
        let appName = frontApp?.localizedName ?? ""
        let bundleId = frontApp?.bundleIdentifier ?? ""

        guard trusted, let appElement = focusedApplicationElement() else {
            return NavigatorContextPayload(
                appName: appName,
                bundleId: bundleId,
                frontmostBundleId: bundleId,
                windowTitle: "",
                focusedRole: "",
                focusedLabel: "",
                selectedText: "",
                axSnapshot: "",
                inputFieldHint: "",
                lastInputFieldDescriptor: "",
                screenshot: "",
                focusStableMs: 0,
                captureConfidence: trusted ? 0.2 : 0.05,
                visibleInputCandidateCount: 0,
                accessibilityPermission: trusted ? "trusted" : "denied",
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
        let windowTitleKey = windowTitle.trimmingCharacters(in: .whitespacesAndNewlines)
        let focusStableMs = updateFocusStability(
            bundleId: bundleId,
            windowTitle: windowTitleKey,
            focusedRole: focusedRole,
            focusedLabel: focusedLabel
        )
        let inputFieldHint = resolvedInputFieldHint(
            bundleId: bundleId,
            windowTitle: windowTitleKey,
            focusedRole: focusedRole,
            focusedLabel: focusedLabel,
            snapshot: snapshot
        )
        let lastInputFieldDescriptor = resolvedInputFieldDescriptor(
            bundleId: bundleId,
            windowTitle: windowTitleKey,
            focusedRole: focusedRole,
            focusedLabel: focusedLabel,
            inputFieldHint: inputFieldHint
        )
        let visibleInputCandidateCount = countVisibleInputCandidates(in: snapshot)
        let captureConfidence = contextCaptureConfidence(
            trusted: trusted,
            focusedRole: focusedRole,
            focusedLabel: focusedLabel,
            snapshot: snapshot,
            focusStableMs: focusStableMs,
            inputFieldHint: inputFieldHint,
            visibleInputCandidateCount: visibleInputCandidateCount
        )

        return NavigatorContextPayload(
            appName: appName,
            bundleId: bundleId,
            frontmostBundleId: bundleId,
            windowTitle: windowTitle,
            focusedRole: focusedRole,
            focusedLabel: focusedLabel,
            selectedText: selectedText,
            axSnapshot: snapshot,
            inputFieldHint: inputFieldHint,
            lastInputFieldDescriptor: lastInputFieldDescriptor,
            screenshot: "",
            focusStableMs: focusStableMs,
            captureConfidence: captureConfidence,
            visibleInputCandidateCount: visibleInputCandidateCount,
            accessibilityPermission: trusted ? "trusted" : "denied",
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
            guard targetAppMatchesFrontmost(step.targetApp, descriptorAppName: step.targetDescriptor.appName) else {
                return NavigatorExecutionResult(status: "guided_mode", observedOutcome: "The target app changed before I could send keys safely.")
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
            guard targetAppMatchesFrontmost(step.targetApp, descriptorAppName: step.targetDescriptor.appName) else {
                return NavigatorExecutionResult(status: "guided_mode", observedOutcome: "The target app changed before I could insert text safely.")
            }
            if descriptorNeedsDirectResolution(step.targetDescriptor) {
                guard let element = resolveElement(for: step.targetDescriptor),
                      focusElementForTextEntry(element) else {
                    return NavigatorExecutionResult(status: "guided_mode", observedOutcome: "I could not safely focus the input field for text entry.")
                }
                try? await Task.sleep(nanoseconds: 200_000_000)
            } else if !looksLikeTextInputRole(currentContext().focusedRole) {
                return NavigatorExecutionResult(status: "guided_mode", observedOutcome: "I could not confirm which input field should receive the text.")
            }
            guard let inputText = step.inputText, !inputText.isEmpty else {
                return NavigatorExecutionResult(status: "failed", observedOutcome: "Missing input text")
            }
            guard let snapshot = stagePasteboardText(inputText) else {
                return NavigatorExecutionResult(status: "failed", observedOutcome: "Could not prepare the text safely")
            }
            defer {
                restorePasteboardSnapshot(snapshot)
            }
            if sendHotkey(["command", "v"]) {
                try? await Task.sleep(nanoseconds: 350_000_000)
                return verify(step: step, before: before, defaultOutcome: "Inserted text")
            }
            return NavigatorExecutionResult(status: "failed", observedOutcome: "Could not insert text")

        case .copySelection:
            guard AXIsProcessTrusted() else {
                return NavigatorExecutionResult(status: "guided_mode", observedOutcome: "Accessibility permission is required for copy")
            }
            guard targetAppMatchesFrontmost(step.targetApp, descriptorAppName: step.targetDescriptor.appName) else {
                return NavigatorExecutionResult(status: "guided_mode", observedOutcome: "The target app changed before I could copy safely.")
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
            let didAct: Bool
            let defaultOutcome: String
            if looksLikeTextInputRole(step.targetDescriptor.role) || looksLikeTextInputRole(stringValue(for: element, attribute: kAXRoleAttribute)) {
                didAct = focusElementForTextEntry(element)
                defaultOutcome = "Focused the target input field"
            } else {
                didAct = AXUIElementPerformAction(element, kAXPressAction as CFString) == .success
                defaultOutcome = "Pressed the target control"
            }
            if didAct {
                try? await Task.sleep(nanoseconds: 350_000_000)
                return verify(step: step, before: before, defaultOutcome: defaultOutcome)
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

        if targetLooksWrong(after: after, descriptor: step.targetDescriptor) {
            return NavigatorExecutionResult(
                status: "guided_mode",
                observedOutcome: "The focus moved, but not to the intended target."
            )
        }

        let contextChanged = after.windowTitle != before.windowTitle
            || after.appName != before.appName
            || after.bundleId != before.bundleId
            || after.focusedLabel != before.focusedLabel
            || after.selectedText != before.selectedText
            || after.axSnapshot != before.axSnapshot

        if appMatches && contextChanged {
            return NavigatorExecutionResult(status: "success", observedOutcome: defaultOutcome)
        }

        if appMatches && step.actionType == .focusApp && !targetApp.isEmpty {
            return NavigatorExecutionResult(status: "success", observedOutcome: defaultOutcome)
        }

        return NavigatorExecutionResult(status: "guided_mode", observedOutcome: "The action ran, but I could not verify the target state safely.")
    }

    private func targetLooksWrong(after: NavigatorContextPayload, descriptor: NavigatorTargetDescriptor) -> Bool {
        let expectedLabel = normalizedMatchValue(descriptor.label) ?? normalizedMatchValue(labelFromDescriptor(after.lastInputFieldDescriptor))
        let expectedRole = normalizedMatchValue(descriptor.role)
        let observedLabel = normalizedMatchValue(after.focusedLabel)
            ?? normalizedMatchValue(after.inputFieldHint)
            ?? normalizedMatchValue(labelFromDescriptor(after.lastInputFieldDescriptor))
        let observedRole = normalizedMatchValue(after.focusedRole)

        if let expectedLabel, let observedLabel,
           !(observedLabel.contains(expectedLabel) || expectedLabel.contains(observedLabel)),
           (looksLikeTextInputRole(descriptor.role) || looksLikeTextInputRole(after.focusedRole)) {
            return true
        }

        if let expectedRole, let observedRole,
           !(observedRole.contains(expectedRole) || expectedRole.contains(observedRole)),
           looksLikeTextInputRole(descriptor.role) {
            return true
        }

        return false
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

    private func targetAppMatchesFrontmost(_ targetApp: String, descriptorAppName: String? = nil) -> Bool {
        let expectedApp = normalizedMatchValue(targetApp) ?? normalizedMatchValue(descriptorAppName)
        guard let expectedApp else {
            return false
        }
        guard let frontApp = NSWorkspace.shared.frontmostApplication else {
            return false
        }
        if let expectedBundleID = explicitBundleIdentifier(for: expectedApp),
           frontApp.bundleIdentifier == expectedBundleID {
            return true
        }
        guard let frontmostName = normalizedMatchValue(frontApp.localizedName) else {
            return false
        }
        return frontmostName.contains(expectedApp) || expectedApp.contains(frontmostName)
    }

    private func stagePasteboardText(_ text: String) -> [PasteboardSnapshotItem]? {
        let pasteboard = NSPasteboard.general
        let snapshot = capturePasteboardSnapshot(from: pasteboard)
        pasteboard.clearContents()
        guard pasteboard.setString(text, forType: .string) else {
            restorePasteboardSnapshot(snapshot)
            return nil
        }
        return snapshot
    }

    private func capturePasteboardSnapshot(from pasteboard: NSPasteboard) -> [PasteboardSnapshotItem] {
        (pasteboard.pasteboardItems ?? []).compactMap { item in
            let dataByType = item.types.reduce(into: [NSPasteboard.PasteboardType: Data]()) { partial, type in
                if let data = item.data(forType: type) {
                    partial[type] = data
                }
            }
            return dataByType.isEmpty ? nil : PasteboardSnapshotItem(dataByType: dataByType)
        }
    }

    private func restorePasteboardSnapshot(_ snapshot: [PasteboardSnapshotItem]) {
        let pasteboard = NSPasteboard.general
        pasteboard.clearContents()
        guard !snapshot.isEmpty else { return }
        let items = snapshot.map { snapshotItem in
            let item = NSPasteboardItem()
            for (type, data) in snapshotItem.dataByType {
                item.setData(data, forType: type)
            }
            return item
        }
        pasteboard.writeObjects(items)
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
        let windowTitle = normalizedMatchValue(descriptor.windowTitle)
        let desiredRole = normalizedMatchValue(descriptor.role)
        let desiredLabel = normalizedMatchValue(descriptor.label)

        guard desiredRole != nil || desiredLabel != nil else {
            return nil
        }

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

    private func descriptorNeedsDirectResolution(_ descriptor: NavigatorTargetDescriptor) -> Bool {
        normalizedMatchValue(descriptor.role) != nil || normalizedMatchValue(descriptor.label) != nil
    }

    private func looksLikeTextInputRole(_ rawRole: String?) -> Bool {
        guard let lowered = normalizedMatchValue(rawRole) else { return false }
        return lowered.contains("textfield") || lowered.contains("textarea") || lowered.contains("searchfield")
    }

    private func focusElementForTextEntry(_ element: AXUIElement) -> Bool {
        if AXUIElementSetAttributeValue(element, kAXFocusedAttribute as CFString, kCFBooleanTrue) == .success {
            return true
        }
        return AXUIElementPerformAction(element, kAXPressAction as CFString) == .success
    }

    private func normalizedMatchValue(_ raw: String?) -> String? {
        guard let raw else { return nil }
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        return trimmed.isEmpty ? nil : trimmed
    }

    private func updateFocusStability(bundleId: String, windowTitle: String, focusedRole: String, focusedLabel: String) -> Int {
        let signature = [
            bundleId.trimmingCharacters(in: .whitespacesAndNewlines),
            windowTitle.trimmingCharacters(in: .whitespacesAndNewlines),
            focusedRole.trimmingCharacters(in: .whitespacesAndNewlines),
            focusedLabel.trimmingCharacters(in: .whitespacesAndNewlines)
        ].joined(separator: "::")

        let now = Date()
        if signature != lastFocusSignature {
            lastFocusSignature = signature
            focusStableSince = now
            return 0
        }

        let elapsed = now.timeIntervalSince(focusStableSince) * 1000
        return max(0, Int(elapsed.rounded()))
    }

    private func resolvedInputFieldHint(
        bundleId: String,
        windowTitle: String,
        focusedRole: String,
        focusedLabel: String,
        snapshot: String
    ) -> String {
        let normalizedBundle = bundleId.trimmingCharacters(in: .whitespacesAndNewlines)
        let normalizedWindow = windowTitle.trimmingCharacters(in: .whitespacesAndNewlines)

        if looksLikeTextInputRole(focusedRole) {
            let hint = normalizedInputFieldHint(focusedLabel) ?? fallbackInputFieldHint(from: snapshot)
            if let hint {
                cacheInputFieldHint(hint, bundleId: normalizedBundle, windowTitle: normalizedWindow)
                return hint
            }
        }

        if let snapshotHint = fallbackInputFieldHint(from: snapshot) {
            cacheInputFieldHint(snapshotHint, bundleId: normalizedBundle, windowTitle: normalizedWindow)
            return snapshotHint
        }

        if normalizedBundle == lastInputFieldBundleID,
           normalizedWindow == lastInputFieldWindow,
           !lastInputFieldHint.isEmpty {
            return lastInputFieldHint
        }

        return ""
    }

    private func normalizedInputFieldHint(_ raw: String?) -> String? {
        guard let raw else { return nil }
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }

    private func labelFromDescriptor(_ raw: String) -> String {
        for part in raw.split(separator: "|") {
            let value = part.trimmingCharacters(in: .whitespacesAndNewlines)
            guard value.lowercased().hasPrefix("label=") else { continue }
            return String(value.dropFirst("label=".count))
        }
        return ""
    }

    private func fallbackInputFieldHint(from snapshot: String) -> String? {
        for rawLine in snapshot.split(separator: "\n") {
            let line = rawLine.trimmingCharacters(in: .whitespacesAndNewlines)
            guard line.lowercased().contains("input:") else { continue }
            let parts = line.split(separator: ":").map { String($0).trimmingCharacters(in: .whitespacesAndNewlines) }
            if let label = parts.last, !label.isEmpty, label.caseInsensitiveCompare("input") != .orderedSame {
                return label
            }
        }
        return nil
    }

    private func cacheInputFieldHint(_ hint: String, bundleId: String, windowTitle: String) {
        lastInputFieldHint = hint
        lastInputFieldBundleID = bundleId
        lastInputFieldWindow = windowTitle
    }

    private func resolvedInputFieldDescriptor(
        bundleId: String,
        windowTitle: String,
        focusedRole: String,
        focusedLabel: String,
        inputFieldHint: String
    ) -> String {
        let normalizedBundle = bundleId.trimmingCharacters(in: .whitespacesAndNewlines)
        let normalizedWindow = windowTitle.trimmingCharacters(in: .whitespacesAndNewlines)
        let normalizedRole = normalizeRoleToken(focusedRole)
        let normalizedLabel = normalizedInputFieldHint(inputFieldHint) ?? normalizedInputFieldHint(focusedLabel)

        if looksLikeTextInputRole(focusedRole), let normalizedLabel {
            let descriptor = buildInputFieldDescriptor(
                bundleId: normalizedBundle,
                windowTitle: normalizedWindow,
                role: normalizedRole.isEmpty ? "textfield" : normalizedRole,
                label: normalizedLabel
            )
            lastInputFieldDescriptor = descriptor
            lastInputFieldBundleID = normalizedBundle
            lastInputFieldWindow = normalizedWindow
            return descriptor
        }

        if normalizedBundle == lastInputFieldBundleID,
           normalizedWindow == lastInputFieldWindow,
           !lastInputFieldDescriptor.isEmpty {
            return lastInputFieldDescriptor
        }

        if let normalizedLabel {
            let descriptor = buildInputFieldDescriptor(
                bundleId: normalizedBundle,
                windowTitle: normalizedWindow,
                role: normalizedRole.isEmpty ? "textfield" : normalizedRole,
                label: normalizedLabel
            )
            lastInputFieldDescriptor = descriptor
            lastInputFieldBundleID = normalizedBundle
            lastInputFieldWindow = normalizedWindow
            return descriptor
        }

        return ""
    }

    private func buildInputFieldDescriptor(bundleId: String, windowTitle: String, role: String, label: String) -> String {
        let components = [
            "bundle=\(bundleId)",
            "window=\(windowTitle)",
            "role=\(role)",
            "label=\(label)"
        ]
        return components.joined(separator: "|")
    }

    private func normalizeRoleToken(_ rawRole: String) -> String {
        let lowered = rawRole.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        switch true {
        case lowered.contains("textarea"):
            return "textarea"
        case lowered.contains("searchfield"):
            return "searchfield"
        case lowered.contains("textfield"):
            return "textfield"
        default:
            return lowered
        }
    }

    private func countVisibleInputCandidates(in snapshot: String) -> Int {
        var uniqueCandidates = Set<String>()
        for rawLine in snapshot.split(separator: "\n") {
            var line = rawLine.trimmingCharacters(in: .whitespacesAndNewlines)
            if line.hasPrefix("focused:") {
                line.removeFirst("focused:".count)
            }
            let lowered = line.lowercased()
            guard lowered.contains("input:") || lowered.contains("axtextfield") || lowered.contains("axtextarea") || lowered.contains("axsearchfield") else {
                continue
            }
            uniqueCandidates.insert(lowered)
        }
        return uniqueCandidates.count
    }

    private func contextCaptureConfidence(
        trusted: Bool,
        focusedRole: String,
        focusedLabel: String,
        snapshot: String,
        focusStableMs: Int,
        inputFieldHint: String,
        visibleInputCandidateCount: Int
    ) -> Double {
        guard trusted else { return 0.05 }

        var score = 0.35
        if !snapshot.isEmpty {
            score += 0.2
        }
        if !focusedRole.isEmpty {
            score += 0.15
        }
        if !focusedLabel.isEmpty || !inputFieldHint.isEmpty {
            score += 0.1
        }
        if focusStableMs >= 300 {
            score += 0.15
        } else if focusStableMs > 0 {
            score += 0.05
        }
        if looksLikeTextInputRole(focusedRole) {
            score += 0.05
        }
        if visibleInputCandidateCount > 1 && !looksLikeTextInputRole(focusedRole) {
            score -= 0.15
        }

        return min(0.99, max(0.05, score))
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
        if looksLikeTextInputRole(role) {
            let parts = ["input", role, label].filter { !$0.isEmpty }
            return parts.joined(separator: ":")
        }
        let parts = [role, label].filter { !$0.isEmpty }
        return parts.joined(separator: ":")
    }

    private func bestLabel(for element: AXUIElement?) -> String {
        guard let element else { return "" }
        return stringValue(for: element, attribute: kAXTitleAttribute)
            ?? stringValue(for: element, attribute: kAXDescriptionAttribute)
            ?? stringValue(for: element, attribute: kAXPlaceholderValueAttribute)
            ?? stringValue(for: element, attribute: kAXHelpAttribute)
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
