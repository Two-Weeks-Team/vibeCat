import AppKit
import ApplicationServices
import Carbon.HIToolbox
import Foundation
import VibeCatCore

struct NavigatorExecutionResult: Sendable {
    let status: String
    let observedOutcome: String
    let failureReason: ExecutionFailureReason?
    let phase: ExecutionPhase

    init(
        status: String,
        observedOutcome: String,
        failureReason: ExecutionFailureReason? = nil,
        phase: ExecutionPhase
    ) {
        self.status = status
        self.observedOutcome = observedOutcome
        self.failureReason = failureReason
        self.phase = phase
    }

    static func success(_ observedOutcome: String, phase: ExecutionPhase) -> NavigatorExecutionResult {
        NavigatorExecutionResult(status: "success", observedOutcome: observedOutcome, phase: phase)
    }

    static func failed(_ observedOutcome: String, reason: ExecutionFailureReason, phase: ExecutionPhase) -> NavigatorExecutionResult {
        NavigatorExecutionResult(status: "failed", observedOutcome: observedOutcome, failureReason: reason, phase: phase)
    }

    static func guided(_ observedOutcome: String, reason: ExecutionFailureReason, phase: ExecutionPhase) -> NavigatorExecutionResult {
        NavigatorExecutionResult(status: "guided_mode", observedOutcome: observedOutcome, failureReason: reason, phase: phase)
    }
}

private struct PasteboardSnapshotItem: Sendable {
    let dataByType: [NSPasteboard.PasteboardType: Data]
}

private struct TextInputCandidate {
    let element: AXUIElement
    let role: String
    let label: String
    let position: CGPoint?
    let size: CGSize?
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
        let roots = [window, focusedElement, appElement].compactMap { $0 }
        let inputCandidates = textInputCandidates(from: roots, maxDepth: 12, maxNodes: 1200)
        let snapshot = summarize(window: window, focusedElement: focusedElement, inputCandidates: inputCandidates)
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
            snapshot: snapshot,
            inputCandidates: inputCandidates
        )
        let visibleInputCandidateCount = inputCandidates.count
        let lastInputFieldDescriptor = resolvedInputFieldDescriptor(
            bundleId: bundleId,
            windowTitle: windowTitleKey,
            focusedRole: focusedRole,
            focusedLabel: focusedLabel,
            inputFieldHint: inputFieldHint,
            visibleInputCandidateCount: visibleInputCandidateCount
        )
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
            return .failed("Could not focus \(step.targetApp)", reason: .targetNotFound, phase: .activateTarget)

        case .openURL:
            guard let rawURL = step.url, let url = URL(string: rawURL) else {
                return .failed("Missing URL", reason: .targetNotFound, phase: .preflight)
            }
            if open(url: url, targetApp: step.targetApp) {
                try? await Task.sleep(nanoseconds: 600_000_000)
                return verify(step: step, before: before, defaultOutcome: "Opened \(url.absoluteString)")
            }
            return .failed("Could not open URL", reason: .targetNotFound, phase: .performAction)

        case .hotkey:
            guard AXIsProcessTrusted() else {
                return .guided("Accessibility permission is required for hotkeys", reason: .focusNotReady, phase: .preflight)
            }
            guard targetAppMatchesFrontmost(step.targetApp, descriptorAppName: step.targetDescriptor.appName) else {
                return .guided("The target app changed before I could send keys safely.", reason: .wrongTarget, phase: .preflight)
            }
            if sendHotkey(step.hotkey) {
                try? await Task.sleep(nanoseconds: 350_000_000)
                return verify(step: step, before: before, defaultOutcome: "Sent hotkey \(step.hotkey.joined(separator: "+"))")
            }
            return .failed("Could not send hotkey", reason: .focusNotReady, phase: .performAction)

        case .pasteText:
            await prepareSurfaceForAction(step)
            guard AXIsProcessTrusted() else {
                return .guided("Accessibility permission is required for text entry", reason: .focusNotReady, phase: .preflight)
            }
            guard targetAppMatchesFrontmost(step.targetApp, descriptorAppName: step.targetDescriptor.appName) else {
                return .guided("The target app changed before I could insert text safely.", reason: .wrongTarget, phase: .preflight)
            }
            if descriptorNeedsDirectResolution(step.targetDescriptor) {
                guard let element = resolveElement(for: step.targetDescriptor),
                      await activateTextEntryElement(element, descriptor: step.targetDescriptor, targetApp: step.targetApp) else {
                    return .guided("I could not safely focus the input field for text entry.", reason: .targetNotWritable, phase: .activateTarget)
                }
            } else if !looksLikeTextInputRole(currentContext().focusedRole) {
                return .guided("I could not confirm which input field should receive the text.", reason: .wrongTarget, phase: .resolveTarget)
            }
            guard let inputText = step.inputText, !inputText.isEmpty else {
                return .failed("Missing input text", reason: .targetNotFound, phase: .preflight)
            }
            guard let snapshot = stagePasteboardText(inputText) else {
                return .failed("Could not prepare the text safely", reason: .pasteRejected, phase: .performAction)
            }
            defer {
                restorePasteboardSnapshot(snapshot)
            }
            if sendHotkey(["command", "v"]) {
                try? await Task.sleep(nanoseconds: 350_000_000)
                return verify(step: step, before: before, defaultOutcome: "Inserted text")
            }
            return .failed("Could not insert text", reason: .pasteRejected, phase: .performAction)

        case .copySelection:
            guard AXIsProcessTrusted() else {
                return .guided("Accessibility permission is required for copy", reason: .focusNotReady, phase: .preflight)
            }
            guard targetAppMatchesFrontmost(step.targetApp, descriptorAppName: step.targetDescriptor.appName) else {
                return .guided("The target app changed before I could copy safely.", reason: .wrongTarget, phase: .preflight)
            }
            if sendHotkey(["command", "c"]) {
                try? await Task.sleep(nanoseconds: 250_000_000)
                return verify(step: step, before: before, defaultOutcome: "Copied the current selection")
            }
            return .failed("Could not copy the selection", reason: .focusNotReady, phase: .performAction)

        case .pressAX:
            await prepareSurfaceForAction(step)
            guard AXIsProcessTrusted() else {
                return .guided("Accessibility permission is required for UI actions", reason: .focusNotReady, phase: .preflight)
            }
            if looksLikeTextInputRole(step.targetDescriptor.role),
               targetAppMatchesFrontmost(step.targetApp, descriptorAppName: step.targetDescriptor.appName),
               textInputFocusAlreadySafe(before, descriptor: step.targetDescriptor) {
                if let app = focusedApplicationElement(),
                   let focusedElement = focusedUIElement(from: app) {
                    _ = focusTextInputElementViaAX(focusedElement)
                }
                try? await Task.sleep(nanoseconds: 150_000_000)
                return verify(step: step, before: before, defaultOutcome: "Focused the target input field")
            }
            guard let element = resolveElement(for: step.targetDescriptor) else {
                return .guided("I found the likely target, but I should not click blindly here.", reason: .targetNotFound, phase: .resolveTarget)
            }
            let didAct: Bool
            let defaultOutcome: String
            if looksLikeTextInputRole(step.targetDescriptor.role) || looksLikeTextInputRole(stringValue(for: element, attribute: kAXRoleAttribute)) {
                didAct = await activateTextEntryElement(element, descriptor: step.targetDescriptor, targetApp: step.targetApp)
                defaultOutcome = "Focused the target input field"
            } else {
                didAct = AXUIElementPerformAction(element, kAXPressAction as CFString) == .success
                defaultOutcome = "Pressed the target control"
            }
            if didAct {
                try? await Task.sleep(nanoseconds: 350_000_000)
                return verify(step: step, before: before, defaultOutcome: defaultOutcome)
            }
            return .guided("The target was visible, but it was not safely pressable.", reason: .targetNotWritable, phase: .performAction)

        case .systemAction:
            return performSystemAction(step)

        case .waitFor:
            try? await Task.sleep(nanoseconds: 600_000_000)
            return verify(step: step, before: before, defaultOutcome: "Observed the next UI state")
        }
    }

    func highlightRect(for step: NavigatorStep) -> CGRect? {
        switch step.actionType {
        case .pressAX, .pasteText:
            guard let element = resolveElement(for: step.targetDescriptor) else {
                return nil
            }
            return rectValue(for: element)
        default:
            return nil
        }
    }

    private func verify(step: NavigatorStep, before: NavigatorContextPayload, defaultOutcome: String) -> NavigatorExecutionResult {
        let after = currentContext()
        let verifyHint = (step.verifyHint ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        let targetApp = step.targetApp.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        let lowerAppName = after.appName.lowercased()
        let appMatches = targetApp.isEmpty || lowerAppName.contains(targetApp) || targetApp.contains(lowerAppName)

        if !verifyHint.isEmpty {
            let haystack = [
                after.windowTitle,
                after.focusedLabel,
                after.selectedText,
                after.axSnapshot
            ].joined(separator: "\n").lowercased()
            if haystack.contains(verifyHint.lowercased()) {
                return .success(defaultOutcome, phase: .verifyOutcome)
            }
        }

        if targetLooksWrong(after: after, descriptor: step.targetDescriptor) {
            return .guided("The focus moved, but not to the intended target.", reason: .wrongTarget, phase: .verifyOutcome)
        }

        let contextChanged = after.windowTitle != before.windowTitle
            || after.appName != before.appName
            || after.bundleId != before.bundleId
            || after.focusedLabel != before.focusedLabel
            || after.selectedText != before.selectedText
            || after.axSnapshot != before.axSnapshot

        if appMatches && contextChanged {
            return .success(defaultOutcome, phase: .verifyOutcome)
        }

        if appMatches && step.actionType == .focusApp && !targetApp.isEmpty {
            return .success(defaultOutcome, phase: .verifyOutcome)
        }

        if appMatches,
           step.actionType == .pressAX,
           looksLikeTextInputRole(step.targetDescriptor.role),
           textInputFocusAlreadySafe(after, descriptor: step.targetDescriptor) {
            return .success(defaultOutcome, phase: .verifyOutcome)
        }

        if step.actionType == .systemAction {
            return .success(defaultOutcome, phase: .verifyOutcome)
        }

        return .guided("The action ran, but I could not verify the target state safely.", reason: .verificationInconclusive, phase: .verifyOutcome)
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

    private func prepareSurfaceForAction(_ step: NavigatorStep) async {
        let profile = NavigatorSurfaceProfile.detect(
            targetApp: step.targetApp,
            descriptor: step.targetDescriptor,
            appName: currentContext().appName,
            bundleID: currentContext().bundleId
        )
        guard let hotkey = profile.preferredPreparationHotkey(for: step.actionType) else {
            return
        }
        guard sendHotkey(hotkey) else {
            return
        }
        try? await Task.sleep(nanoseconds: 180_000_000)
    }

    private func focusApp(named targetApp: String) -> Bool {
        let trimmed = targetApp.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            NSLog("[NAV-FOCUS] empty targetApp")
            return false
        }

        if let running = NSWorkspace.shared.runningApplications.first(where: {
            ($0.localizedName ?? "").caseInsensitiveCompare(trimmed) == .orderedSame
        }) {
            let ok = running.activate(options: [.activateAllWindows])
            NSLog("[NAV-FOCUS] activate '%@' pid=%d ok=%d", trimmed, running.processIdentifier, ok ? 1 : 0)
            return ok
        }

        let bundleIdentifier = explicitBundleIdentifier(for: targetApp) ?? explicitBundleIdentifier(for: trimmed)
        if let bundleIdentifier,
           let appURL = NSWorkspace.shared.urlForApplication(withBundleIdentifier: bundleIdentifier) {
            NSLog("[NAV-FOCUS] launching '%@' via bundle=%@", trimmed, bundleIdentifier)
            let config = NSWorkspace.OpenConfiguration()
            NSWorkspace.shared.openApplication(at: appURL, configuration: config) { _, _ in }
            return true
        }

        NSLog("[NAV-FOCUS] no match for '%@'", trimmed)
        return false
    }

    private func open(url: URL, targetApp: String) -> Bool {
        if let appURL = preferredAppURL(for: targetApp) {
            let configuration = NSWorkspace.OpenConfiguration()
            NSWorkspace.shared.open([url], withApplicationAt: appURL, configuration: configuration) { _, _ in }
            return true
        }
        return NSWorkspace.shared.open(url)
    }

    private func preferredAppURL(for targetApp: String) -> URL? {
        let profile = NavigatorSurfaceProfile.detect(targetApp: targetApp)
        guard let bundleID = profile.primaryBundleID else { return nil }
        return NSWorkspace.shared.urlForApplication(withBundleIdentifier: bundleID)
    }

    private func explicitBundleIdentifier(for targetApp: String) -> String? {
        let trimmed = targetApp.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.contains(".") {
            return trimmed
        }
        return NavigatorSurfaceProfile.detect(targetApp: trimmed).primaryBundleID
    }

    private func targetAppMatchesFrontmost(_ targetApp: String, descriptorAppName: String? = nil) -> Bool {
        let profile = NavigatorSurfaceProfile.detect(targetApp: targetApp, descriptor: .init(appName: descriptorAppName))
        let expectedApp = normalizedMatchValue(targetApp) ?? normalizedMatchValue(descriptorAppName)
        guard let expectedApp else {
            NSLog("[NAV-TARGET] no expected app derived from targetApp=%@ descriptor=%@, accepting frontmost", targetApp, descriptorAppName ?? "nil")
            return true
        }
        guard let frontApp = NSWorkspace.shared.frontmostApplication else {
            NSLog("[NAV-TARGET] no frontmost application")
            return false
        }
        if profile.matches(bundleID: frontApp.bundleIdentifier) {
            return true
        }
        guard let frontmostName = normalizedMatchValue(frontApp.localizedName) else {
            NSLog("[NAV-TARGET] mismatch: expected=%@ frontName=nil bundleID=%@", expectedApp, frontApp.bundleIdentifier ?? "nil")
            return false
        }
        if profile.matches(appName: frontmostName) {
            return true
        }
        let matched = frontmostName.contains(expectedApp) || expectedApp.contains(frontmostName)
        if !matched {
            NSLog("[NAV-TARGET] mismatch: expected=%@ front=%@ bundleID=%@", expectedApp, frontmostName, frontApp.bundleIdentifier ?? "nil")
        }
        return matched
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
            NSLog("[NAV-KEY] unknown key token in %@", tokens.joined(separator: "+"))
            return false
        }

        let modifiers = tokens.dropLast().reduce(CGEventFlags()) { partial, token in
            partial.union(eventFlag(for: token.lowercased()))
        }

        guard let source = CGEventSource(stateID: .combinedSessionState),
              let keyDown = CGEvent(keyboardEventSource: source, virtualKey: keyCode, keyDown: true),
              let keyUp = CGEvent(keyboardEventSource: source, virtualKey: keyCode, keyDown: false) else {
            NSLog("[NAV-KEY] CGEvent creation failed for %@", tokens.joined(separator: "+"))
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
        // Full alphabet
        case "a": return CGKeyCode(kVK_ANSI_A)
        case "b": return CGKeyCode(kVK_ANSI_B)
        case "c": return CGKeyCode(kVK_ANSI_C)
        case "d": return CGKeyCode(kVK_ANSI_D)
        case "e": return CGKeyCode(kVK_ANSI_E)
        case "f": return CGKeyCode(kVK_ANSI_F)
        case "g": return CGKeyCode(kVK_ANSI_G)
        case "h": return CGKeyCode(kVK_ANSI_H)
        case "i": return CGKeyCode(kVK_ANSI_I)
        case "j": return CGKeyCode(kVK_ANSI_J)
        case "k": return CGKeyCode(kVK_ANSI_K)
        case "l": return CGKeyCode(kVK_ANSI_L)
        case "m": return CGKeyCode(kVK_ANSI_M)
        case "n": return CGKeyCode(kVK_ANSI_N)
        case "o": return CGKeyCode(kVK_ANSI_O)
        case "p": return CGKeyCode(kVK_ANSI_P)
        case "q": return CGKeyCode(kVK_ANSI_Q)
        case "r": return CGKeyCode(kVK_ANSI_R)
        case "s": return CGKeyCode(kVK_ANSI_S)
        case "t": return CGKeyCode(kVK_ANSI_T)
        case "u": return CGKeyCode(kVK_ANSI_U)
        case "v": return CGKeyCode(kVK_ANSI_V)
        case "w": return CGKeyCode(kVK_ANSI_W)
        case "x": return CGKeyCode(kVK_ANSI_X)
        case "y": return CGKeyCode(kVK_ANSI_Y)
        case "z": return CGKeyCode(kVK_ANSI_Z)
        // Number keys
        case "0": return CGKeyCode(kVK_ANSI_0)
        case "1": return CGKeyCode(kVK_ANSI_1)
        case "2": return CGKeyCode(kVK_ANSI_2)
        case "3": return CGKeyCode(kVK_ANSI_3)
        case "4": return CGKeyCode(kVK_ANSI_4)
        case "5": return CGKeyCode(kVK_ANSI_5)
        case "6": return CGKeyCode(kVK_ANSI_6)
        case "7": return CGKeyCode(kVK_ANSI_7)
        case "8": return CGKeyCode(kVK_ANSI_8)
        case "9": return CGKeyCode(kVK_ANSI_9)
        // Special keys
        case "grave", "`": return CGKeyCode(kVK_ANSI_Grave)
        case "minus", "-": return CGKeyCode(kVK_ANSI_Minus)
        case "equal", "=": return CGKeyCode(kVK_ANSI_Equal)
        case "leftbracket", "[": return CGKeyCode(kVK_ANSI_LeftBracket)
        case "rightbracket", "]": return CGKeyCode(kVK_ANSI_RightBracket)
        case "backslash", "\\": return CGKeyCode(kVK_ANSI_Backslash)
        case "semicolon", ";": return CGKeyCode(kVK_ANSI_Semicolon)
        case "quote", "'": return CGKeyCode(kVK_ANSI_Quote)
        case "comma", ",": return CGKeyCode(kVK_ANSI_Comma)
        case "period", ".": return CGKeyCode(kVK_ANSI_Period)
        case "slash", "/": return CGKeyCode(kVK_ANSI_Slash)
        // Navigation & control
        case "return", "enter": return CGKeyCode(kVK_Return)
        case "tab": return CGKeyCode(kVK_Tab)
        case "space": return CGKeyCode(kVK_Space)
        case "delete", "backspace": return CGKeyCode(kVK_Delete)
        case "forwarddelete": return CGKeyCode(kVK_ForwardDelete)
        case "escape", "esc": return CGKeyCode(kVK_Escape)
        // Arrow keys
        case "left", "leftarrow": return CGKeyCode(kVK_LeftArrow)
        case "right", "rightarrow": return CGKeyCode(kVK_RightArrow)
        case "up", "uparrow": return CGKeyCode(kVK_UpArrow)
        case "down", "downarrow": return CGKeyCode(kVK_DownArrow)
        // Page navigation
        case "home": return CGKeyCode(kVK_Home)
        case "end": return CGKeyCode(kVK_End)
        case "pageup": return CGKeyCode(kVK_PageUp)
        case "pagedown": return CGKeyCode(kVK_PageDown)
        // Function keys
        case "f1": return CGKeyCode(kVK_F1)
        case "f2": return CGKeyCode(kVK_F2)
        case "f3": return CGKeyCode(kVK_F3)
        case "f4": return CGKeyCode(kVK_F4)
        case "f5": return CGKeyCode(kVK_F5)
        case "f6": return CGKeyCode(kVK_F6)
        case "f7": return CGKeyCode(kVK_F7)
        case "f8": return CGKeyCode(kVK_F8)
        case "f9": return CGKeyCode(kVK_F9)
        case "f10": return CGKeyCode(kVK_F10)
        case "f11": return CGKeyCode(kVK_F11)
        case "f12": return CGKeyCode(kVK_F12)
        default: return nil
        }
    }

    private func resolveElement(for descriptor: NavigatorTargetDescriptor) -> AXUIElement? {
        guard let app = focusedApplicationElement() else { return nil }
        let context = currentContext()
        if looksLikeTextInputRole(descriptor.role),
           textInputFocusAlreadySafe(context, descriptor: descriptor),
           let focusedElement = focusedUIElement(from: app),
           let focusedRole = stringValue(for: focusedElement, attribute: kAXRoleAttribute),
           looksLikeTextInputRole(focusedRole) {
            return focusedElement
        }
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

        if looksLikeTextInputRole(descriptor.role),
           let fallback = resolveBestTextInputCandidate(
            from: roots,
            desiredRole: desiredRole,
            desiredLabel: desiredLabel,
            appName: context.appName
           ) {
            return fallback
        }

        return nil
    }

    private func resolveBestTextInputCandidate(
        from roots: [AXUIElement],
        desiredRole: String?,
        desiredLabel: String?,
        appName: String
    ) -> AXUIElement? {
        let candidates = textInputCandidates(from: roots, maxDepth: 12, maxNodes: 1500)
        guard let best = preferredTextInputCandidate(
            from: candidates,
            desiredRole: desiredRole,
            desiredLabel: desiredLabel,
            appName: appName
        ) else {
            return nil
        }
        return best.element
    }

    private func textInputRoleMatches(observed: String, expected: String) -> Bool {
        if looksLikeTextInputRole(observed), looksLikeTextInputRole(expected) {
            return true
        }
        return observed.contains(expected) || expected.contains(observed)
    }

    private func descriptorNeedsDirectResolution(_ descriptor: NavigatorTargetDescriptor) -> Bool {
        normalizedMatchValue(descriptor.role) != nil || normalizedMatchValue(descriptor.label) != nil
    }

    private func looksLikeTextInputRole(_ rawRole: String?) -> Bool {
        guard let lowered = normalizedMatchValue(rawRole) else { return false }
        return lowered.contains("textfield") || lowered.contains("textarea") || lowered.contains("searchfield")
    }

    private func focusTextInputElementViaAX(_ element: AXUIElement) -> Bool {
        if AXUIElementSetAttributeValue(element, kAXFocusedAttribute as CFString, kCFBooleanTrue) == .success {
            return true
        }
        return AXUIElementPerformAction(element, kAXPressAction as CFString) == .success
    }

    private func activateTextEntryElement(
        _ element: AXUIElement,
        descriptor: NavigatorTargetDescriptor,
        targetApp: String
    ) async -> Bool {
        if textInputElementIsReady(element, descriptor: descriptor, targetApp: targetApp) {
            return true
        }

        if focusTextInputElementViaAX(element) {
            NSLog("[NAV-AX] text input focus via AX succeeded")
            try? await Task.sleep(nanoseconds: 140_000_000)
            _ = setTextInsertionCaretToEnd(element)
            try? await Task.sleep(nanoseconds: 80_000_000)
            if textInputElementIsReady(element, descriptor: descriptor, targetApp: targetApp) {
                return true
            }
        } else {
            NSLog("[NAV-AX] text input focus via AX failed")
        }

        if setTextInsertionCaretToEnd(element) {
            NSLog("[NAV-AX] selectedTextRange caret placement succeeded")
            try? await Task.sleep(nanoseconds: 80_000_000)
            if textInputElementIsReady(element, descriptor: descriptor, targetApp: targetApp) {
                return true
            }
        }

        if let point = textInputActivationPoint(for: element) {
            let hitRole = hitTestTextInputRole(at: point)
            NSLog("[NAV-AX] clicking activation point x=%.1f y=%.1f hitRole=%@", point.x, point.y, hitRole)
            if clickTextInput(at: point) {
                try? await Task.sleep(nanoseconds: 180_000_000)
                _ = setTextInsertionCaretToEnd(element)
                try? await Task.sleep(nanoseconds: 80_000_000)
                if textInputElementIsReady(element, descriptor: descriptor, targetApp: targetApp) {
                    return true
                }
            }
        } else {
            NSLog("[NAV-AX] missing activation point for text input candidate")
        }

        return false
    }

    private func textInputElementIsReady(
        _ element: AXUIElement,
        descriptor: NavigatorTargetDescriptor,
        targetApp: String
    ) -> Bool {
        guard targetAppMatchesFrontmost(targetApp, descriptorAppName: descriptor.appName) else {
            return false
        }
        if let app = focusedApplicationElement(),
           let focused = focusedUIElement(from: app),
           sameAXElement(focused, element) {
            return looksLikeTextInputRole(stringValue(for: focused, attribute: kAXRoleAttribute))
        }
        let context = currentContext()
        return textInputFocusAlreadySafe(context, descriptor: descriptor)
    }

    private func setTextInsertionCaretToEnd(_ element: AXUIElement) -> Bool {
        let characterCount = numberValue(for: element, attribute: kAXNumberOfCharactersAttribute) ?? 0
        var range = CFRange(location: characterCount, length: 0)
        guard let value = AXValueCreate(.cfRange, &range) else {
            return false
        }
        return AXUIElementSetAttributeValue(
            element,
            kAXSelectedTextRangeAttribute as CFString,
            value
        ) == .success
    }

    private func textInputActivationPoint(for element: AXUIElement) -> CGPoint? {
        guard let position = pointValue(for: element, attribute: kAXPositionAttribute),
              let size = sizeValue(for: element, attribute: kAXSizeAttribute) else {
            return nil
        }

        let insetX = min(max(18, size.width * 0.08), max(18, size.width - 12))
        let insetY = min(max(18, size.height * 0.5), max(18, size.height - 12))
        return CGPoint(x: position.x + insetX, y: position.y + insetY)
    }

    private func clickTextInput(at point: CGPoint) -> Bool {
        let cgPoint = ScreenCaptureService.MouseWindowTargetingGeometry.appKitToCGPoint(point)
        guard let down = CGEvent(
            mouseEventSource: nil,
            mouseType: .leftMouseDown,
            mouseCursorPosition: cgPoint,
            mouseButton: .left
        ),
        let up = CGEvent(
            mouseEventSource: nil,
            mouseType: .leftMouseUp,
            mouseCursorPosition: cgPoint,
            mouseButton: .left
        ) else {
            return false
        }

        down.post(tap: .cghidEventTap)
        up.post(tap: .cghidEventTap)
        return true
    }

    private func hitTestTextInputRole(at point: CGPoint) -> String {
        let cgPoint = ScreenCaptureService.MouseWindowTargetingGeometry.appKitToCGPoint(point)
        let systemWide = AXUIElementCreateSystemWide()
        var hit: AXUIElement?
        guard AXUIElementCopyElementAtPosition(systemWide, Float(cgPoint.x), Float(cgPoint.y), &hit) == .success,
              let hit else {
            return ""
        }
        return stringValue(for: hit, attribute: kAXRoleAttribute) ?? ""
    }

    private func sameAXElement(_ lhs: AXUIElement, _ rhs: AXUIElement) -> Bool {
        CFEqual(lhs, rhs)
    }

    private func textInputFocusAlreadySafe(_ context: NavigatorContextPayload, descriptor: NavigatorTargetDescriptor) -> Bool {
        guard looksLikeTextInputRole(context.focusedRole) else { return false }
        if let expectedApp = normalizedMatchValue(descriptor.appName),
           let observedApp = normalizedMatchValue(context.appName),
           !(observedApp.contains(expectedApp) || expectedApp.contains(observedApp)) {
            return false
        }
        if let expectedWindow = normalizedMatchValue(descriptor.windowTitle),
           let observedWindow = normalizedMatchValue(context.windowTitle),
           !(observedWindow.contains(expectedWindow) || expectedWindow.contains(observedWindow)) {
            return false
        }

        guard let expectedLabel = normalizedMatchValue(descriptor.label) else {
            return true
        }

        let observedLabel = normalizedMatchValue(context.focusedLabel)
            ?? normalizedMatchValue(context.inputFieldHint)
            ?? normalizedMatchValue(labelFromDescriptor(context.lastInputFieldDescriptor))
        if let observedLabel {
            if observedLabel.contains(expectedLabel) || expectedLabel.contains(observedLabel) {
                return true
            }
        }

        return context.visibleInputCandidateCount <= 1
    }

    private func performSystemAction(_ step: NavigatorStep) -> NavigatorExecutionResult {
        let command = (step.systemCommand ?? "").trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        let value = (step.systemValue ?? "").trimmingCharacters(in: .whitespacesAndNewlines).lowercased()

        switch (command, value) {
        case ("volume", "mute"):
            guard runAppleScript(lines: ["set volume output muted true"]) != nil else {
                return .failed("Could not mute system audio", reason: .surfaceAdapterUnavailable, phase: .performAction)
            }
            return .success("Muted system audio", phase: .performAction)
        case ("volume", "unmute"):
            guard runAppleScript(lines: ["set volume output muted false"]) != nil else {
                return .failed("Could not unmute system audio", reason: .surfaceAdapterUnavailable, phase: .performAction)
            }
            return .success("Unmuted system audio", phase: .performAction)
        case ("volume", "down"), ("volume", "up"):
            let amount = max(1, min(step.systemAmount, 100))
            let direction = value == "down" ? -amount : amount
            let script = [
                "set currentSettings to get volume settings",
                "set currentVolume to output volume of currentSettings",
                "set newVolume to currentVolume + (\(direction))",
                "if newVolume < 0 then set newVolume to 0",
                "if newVolume > 100 then set newVolume to 100",
                "set volume output muted false",
                "set volume output volume newVolume",
                "return newVolume"
            ]
            guard let output = runAppleScript(lines: script), !output.isEmpty else {
                return .failed("Could not adjust system volume", reason: .surfaceAdapterUnavailable, phase: .performAction)
            }
            let directionLabel = value == "down" ? "lowered" : "raised"
            return .success("System volume \(directionLabel) to \(output)", phase: .performAction)
        default:
            return .guided("I do not support that macOS system action safely yet.", reason: .surfaceAdapterUnavailable, phase: .preflight)
        }
    }

    private func runAppleScript(lines: [String]) -> String? {
        let task = Process()
        task.executableURL = URL(fileURLWithPath: "/usr/bin/osascript")
        task.arguments = lines.flatMap { ["-e", $0] }

        let outputPipe = Pipe()
        task.standardOutput = outputPipe
        task.standardError = Pipe()

        do {
            try task.run()
            task.waitUntilExit()
        } catch {
            return nil
        }

        guard task.terminationStatus == 0 else {
            return nil
        }
        let data = outputPipe.fileHandleForReading.readDataToEndOfFile()
        return String(data: data, encoding: .utf8)?
            .trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private func normalizedMatchValue(_ raw: String?) -> String? {
        guard let raw else { return nil }
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        return trimmed.isEmpty ? nil : trimmed
    }

    private func containsKeywordAny(_ text: String, _ keywords: String...) -> Bool {
        keywords.contains { text.contains($0) }
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
        snapshot: String,
        inputCandidates: [TextInputCandidate]
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

        if let candidateLabel = preferredTextInputCandidate(
            from: inputCandidates,
            desiredRole: nil,
            desiredLabel: nil,
            appName: bundleId
        )?.label,
           !candidateLabel.isEmpty {
            cacheInputFieldHint(candidateLabel, bundleId: normalizedBundle, windowTitle: normalizedWindow)
            return candidateLabel
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
        inputFieldHint: String,
        visibleInputCandidateCount: Int
    ) -> String {
        let normalizedBundle = bundleId.trimmingCharacters(in: .whitespacesAndNewlines)
        let normalizedWindow = windowTitle.trimmingCharacters(in: .whitespacesAndNewlines)
        let normalizedRole = normalizeRoleToken(focusedRole)
        let normalizedHint = normalizedInputFieldHint(inputFieldHint)
        let normalizedFocusedLabel = normalizedInputFieldHint(focusedLabel)

        if looksLikeTextInputRole(focusedRole), let normalizedLabel = normalizedHint ?? normalizedFocusedLabel {
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

        if visibleInputCandidateCount > 0, let normalizedLabel = normalizedHint {
            let descriptor = buildInputFieldDescriptor(
                bundleId: normalizedBundle,
                windowTitle: normalizedWindow,
                role: looksLikeTextInputRole(focusedRole) ? normalizedRole : "textfield",
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

    private func summarize(window: AXUIElement?, focusedElement: AXUIElement?, inputCandidates: [TextInputCandidate]) -> String {
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
        if !inputCandidates.isEmpty {
            for candidate in inputCandidates.prefix(3) {
                let line = ["input", candidate.role, candidate.label]
                    .filter { !$0.isEmpty }
                    .joined(separator: ":")
                if !line.isEmpty, !lines.contains(line) {
                    lines.append(line)
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

    private func numberValue(for element: AXUIElement?, attribute: String) -> Int? {
        guard let element,
              let value = attributeValue(for: element, attribute: attribute) as? NSNumber else {
            return nil
        }
        return value.intValue
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

    private func textInputCandidates(from roots: [AXUIElement], maxDepth: Int, maxNodes: Int) -> [TextInputCandidate] {
        var candidates: [TextInputCandidate] = []
        var seen = Set<String>()

        for root in roots {
            for element in breadthFirstSearch(from: root, maxDepth: maxDepth, maxNodes: maxNodes) {
                guard let role = stringValue(for: element, attribute: kAXRoleAttribute),
                      looksLikeTextInputRole(role) else {
                    continue
                }
                let label = bestLabel(for: element)
                let position = pointValue(for: element, attribute: kAXPositionAttribute)
                let size = sizeValue(for: element, attribute: kAXSizeAttribute)
                let signature = [
                    normalizeRoleToken(role),
                    normalizedInputFieldHint(label) ?? "",
                    position.map { "\($0.x.rounded())x\($0.y.rounded())" } ?? "",
                    size.map { "\($0.width.rounded())x\($0.height.rounded())" } ?? ""
                ].joined(separator: "|")
                if seen.insert(signature).inserted {
                    candidates.append(TextInputCandidate(
                        element: element,
                        role: role,
                        label: label,
                        position: position,
                        size: size
                    ))
                }
            }
        }

        return candidates
    }

    private func preferredTextInputCandidate(
        from candidates: [TextInputCandidate],
        desiredRole: String?,
        desiredLabel: String?,
        appName: String
    ) -> TextInputCandidate? {
        let appLower = normalizedMatchValue(appName) ?? ""
        return candidates.max { lhs, rhs in
            let leftScore = score(textInputCandidate: lhs, desiredRole: desiredRole, desiredLabel: desiredLabel, appName: appLower)
            let rightScore = score(textInputCandidate: rhs, desiredRole: desiredRole, desiredLabel: desiredLabel, appName: appLower)
            if leftScore != rightScore {
                return leftScore < rightScore
            }
            let leftY = lhs.position?.y ?? -.greatestFiniteMagnitude
            let rightY = rhs.position?.y ?? -.greatestFiniteMagnitude
            if leftY != rightY {
                return leftY < rightY
            }
            let leftHeight = lhs.size?.height ?? 0
            let rightHeight = rhs.size?.height ?? 0
            return leftHeight < rightHeight
        }
    }

    private func score(
        textInputCandidate candidate: TextInputCandidate,
        desiredRole: String?,
        desiredLabel: String?,
        appName: String
    ) -> Int {
        let profile = NavigatorSurfaceProfile.detect(targetApp: appName, appName: appName)
        let role = normalizedMatchValue(candidate.role) ?? ""
        let label = normalizedMatchValue(candidate.label) ?? ""
        var score = 0

        if let desiredRole {
            if textInputRoleMatches(observed: role, expected: desiredRole) {
                score += 8
            } else {
                score -= 8
            }
        }

        if let desiredLabel {
            if label.contains(desiredLabel) || desiredLabel.contains(label) {
                score += 16
            } else if !label.isEmpty {
                score -= 10
            }
        }

        if role.contains("textarea") {
            score += 6
        } else if role.contains("textfield") || role.contains("searchfield") {
            score += 3
        }

        if containsKeywordAny(label, "prompt", "composer", "message", "follow-up", "후속", "메시지", "입력", "reply") {
            score += 10
        }
        if containsKeywordAny(label, "search", "검색", "address", "url") {
            score -= 6
        }
        if profile.kind == .antigravity && role.contains("textarea") {
            score += 8
        }
        if profile.kind == .antigravity && containsKeywordAny(label, "후속 변경", "follow-up", "부탁하세요") {
            score += 12
        }
        if profile.kind == .chrome,
           containsKeywordAny(label, "search", "검색", "address", "url") {
            score += 8
        }
        if profile.kind == .terminal,
           containsKeywordAny(label, "prompt", "shell", "command") {
            score += 8
        }
        if let height = candidate.size?.height, height >= 36 {
            score += 2
        }
        if let width = candidate.size?.width, width >= 240 {
            score += 2
        }
        if let y = candidate.position?.y {
            score += Int(y / 200)
        }

        return score
    }

    private func pointValue(for element: AXUIElement, attribute: String) -> CGPoint? {
        guard let value = attributeValue(for: element, attribute: attribute),
              CFGetTypeID(value) == AXValueGetTypeID() else {
            return nil
        }
        let axValue = unsafeDowncast(value, to: AXValue.self)
        guard AXValueGetType(axValue) == .cgPoint else {
            return nil
        }
        var point = CGPoint.zero
        guard AXValueGetValue(axValue, .cgPoint, &point) else {
            return nil
        }
        return point
    }

    private func currentDisplayID(window: AXUIElement?, focusedElement: AXUIElement?) -> String? {
        if let rect = rectValue(for: window) ?? rectValue(for: focusedElement) {
            return displayID(containing: rect)
        }
        return mouseDisplayID()
    }

    private func mouseDisplayID() -> String {
        guard let screen = NSScreen.screens.first(where: { NSMouseInRect(NSEvent.mouseLocation, $0.frame, false) }),
              let displayID = screen.deviceDescription[NSDeviceDescriptionKey("NSScreenNumber")] as? CGDirectDisplayID else {
            return ""
        }
        return String(displayID)
    }

    private func rectValue(for element: AXUIElement?) -> CGRect? {
        guard let element,
              let position = pointValue(for: element, attribute: kAXPositionAttribute),
              let size = sizeValue(for: element, attribute: kAXSizeAttribute) else {
            return nil
        }
        return CGRect(origin: position, size: size)
    }

    private func displayID(containing rect: CGRect) -> String? {
        let probe = CGPoint(x: rect.midX, y: rect.midY)
        guard let screen = NSScreen.screens.first(where: { $0.frame.contains(probe) }),
              let displayID = screen.deviceDescription[NSDeviceDescriptionKey("NSScreenNumber")] as? CGDirectDisplayID else {
            return nil
        }
        return String(displayID)
    }

    private func sizeValue(for element: AXUIElement, attribute: String) -> CGSize? {
        guard let value = attributeValue(for: element, attribute: attribute),
              CFGetTypeID(value) == AXValueGetTypeID() else {
            return nil
        }
        let axValue = unsafeDowncast(value, to: AXValue.self)
        guard AXValueGetType(axValue) == .cgSize else {
            return nil
        }
        var size = CGSize.zero
        guard AXValueGetValue(axValue, .cgSize, &size) else {
            return nil
        }
        return size
    }
}
