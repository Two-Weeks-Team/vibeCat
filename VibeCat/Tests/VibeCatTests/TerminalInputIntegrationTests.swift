import XCTest
@testable import VibeCat
@testable import VibeCatCore

final class TerminalInputIntegrationTests: XCTestCase {
    func testAccessibilityNavigatorTypesAndRunsCommandInTerminal() async throws {
        guard ProcessInfo.processInfo.environment["VIBECAT_REAL_TERMINAL_INPUT"] == "1" else {
            throw XCTSkip("VIBECAT_REAL_TERMINAL_INPUT not set — skipping real Terminal integration test")
        }

        _ = try runOSA("tell application \"Terminal\" to activate")
        _ = try runOSA("tell application \"Terminal\" to do script \"\" in front window")
        try await Task.sleep(nanoseconds: 800_000_000)

        let navigator = AccessibilityNavigator()
        let token = "VIBECAT_TERMINAL_SMOKE_OK_20260312"
        let command = "printf '\(token)\\n'"

        let focusStep = NavigatorStep(
            id: "terminal-focus-smoke",
            actionType: .focusApp,
            targetApp: "Terminal",
            expectedOutcome: "Focus Terminal",
            confidence: 0.95,
            intentConfidence: 0.95,
            riskLevel: "low",
            executionPolicy: "safe_immediate",
            fallbackPolicy: "guided_mode",
            verifyHint: "terminal",
            surface: .terminal,
            macroID: "focus_terminal",
            narration: "Switching to Terminal.",
            proofLevel: .strong
        )

        let pasteStep = NavigatorStep(
            id: "terminal-paste-smoke",
            actionType: .pasteText,
            targetApp: "Terminal",
            targetDescriptor: NavigatorTargetDescriptor(appName: "Terminal"),
            inputText: command,
            expectedOutcome: "Place the smoke command into Terminal",
            confidence: 0.95,
            intentConfidence: 0.95,
            riskLevel: "low",
            executionPolicy: "safe_immediate",
            fallbackPolicy: "guided_mode",
            surface: .terminal,
            macroID: "paste_terminal_command",
            narration: "Placing the command into Terminal.",
            proofLevel: .strict
        )
        let submitStep = NavigatorStep(
            id: "terminal-submit-smoke",
            actionType: .hotkey,
            targetApp: "Terminal",
            targetDescriptor: NavigatorTargetDescriptor(appName: "Terminal"),
            expectedOutcome: "Run the smoke command in Terminal",
            confidence: 0.95,
            intentConfidence: 0.95,
            riskLevel: "low",
            executionPolicy: "safe_immediate",
            fallbackPolicy: "guided_mode",
            hotkey: ["return"],
            verifyHint: token.lowercased(),
            surface: .terminal,
            macroID: "submit_terminal_command",
            narration: "Running the command in Terminal.",
            proofLevel: .strong
        )

        let focusResult = await navigator.execute(step: focusStep)
        let pasteResult = await navigator.execute(step: pasteStep)
        let submitResult = await navigator.execute(step: submitStep)
        try await Task.sleep(nanoseconds: 1_200_000_000)

        let terminalContents = try runOSA("tell application \"Terminal\" to get contents of selected tab of front window")

        XCTAssertNotEqual(focusResult.status, "failed")
        XCTAssertNotEqual(pasteResult.status, "failed")
        XCTAssertNotEqual(submitResult.status, "failed")
        XCTAssertTrue(terminalContents.contains(token), "expected Terminal contents to include token; got: \(terminalContents.suffix(400))")
    }

    private func runOSA(_ source: String) throws -> String {
        let process = Process()
        process.executableURL = URL(fileURLWithPath: "/usr/bin/osascript")
        process.arguments = ["-e", source]

        let stdout = Pipe()
        let stderr = Pipe()
        process.standardOutput = stdout
        process.standardError = stderr
        try process.run()
        process.waitUntilExit()

        let output = String(data: stdout.fileHandleForReading.readDataToEndOfFile(), encoding: .utf8) ?? ""
        let error = String(data: stderr.fileHandleForReading.readDataToEndOfFile(), encoding: .utf8) ?? ""

        guard process.terminationStatus == 0 else {
            throw NSError(domain: "TerminalInputIntegrationTests", code: Int(process.terminationStatus), userInfo: [NSLocalizedDescriptionKey: error])
        }

        return output
    }
}
