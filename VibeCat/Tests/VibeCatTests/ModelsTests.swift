import Foundation
import XCTest
@testable import VibeCatCore

final class ModelsTests: XCTestCase {
    func testCharacterPresetConfigCodableRoundTripWithOptionalFields() throws {
        let original = CharacterPresetConfig(
            name: "cat",
            voice: "Zephyr",
            language: "ko",
            size: "small",
            persona: "friendly"
        )

        let data = try JSONEncoder().encode(original)
        let decoded = try JSONDecoder().decode(CharacterPresetConfig.self, from: data)

        XCTAssertEqual(decoded.name, "cat")
        XCTAssertEqual(decoded.voice, "Zephyr")
        XCTAssertEqual(decoded.language, "ko")
        XCTAssertEqual(decoded.size, "small")
        XCTAssertEqual(decoded.persona, "friendly")
    }

    func testCharacterPresetConfigCodableRoundTripWithNilOptionalFields() throws {
        let original = CharacterPresetConfig(name: "jinwoo", voice: "Kore", language: "en")

        let data = try JSONEncoder().encode(original)
        let decoded = try JSONDecoder().decode(CharacterPresetConfig.self, from: data)

        XCTAssertEqual(decoded.name, "jinwoo")
        XCTAssertEqual(decoded.voice, "Kore")
        XCTAssertEqual(decoded.language, "en")
        XCTAssertNil(decoded.size)
        XCTAssertNil(decoded.persona)
    }

    func testChatMessageInitializerUsesProvidedValues() {
        let id = UUID()
        let timestamp = Date(timeIntervalSince1970: 12345)
        let message = ChatMessage(id: id, role: .companion, text: "hello", timestamp: timestamp)

        XCTAssertEqual(message.id, id)
        XCTAssertEqual(message.role, .companion)
        XCTAssertEqual(message.text, "hello")
        XCTAssertEqual(message.timestamp, timestamp)
    }

    func testCompanionSpeechEventDefaults() {
        let event = CompanionSpeechEvent(text: "test")
        XCTAssertEqual(event.text, "test")
        XCTAssertEqual(event.emotion, .neutral)
        XCTAssertEqual(event.urgency, "normal")
    }

    func testNavigatorContextPayloadCodableRoundTripPreservesExtendedFields() throws {
        let original = NavigatorContextPayload(
            appName: "Google Chrome",
            bundleId: "com.google.Chrome",
            frontmostBundleId: "com.google.Chrome",
            windowTitle: "Google",
            focusedRole: "AXTextField",
            focusedLabel: "Search",
            selectedText: "gemini live api",
            axSnapshot: "window:Google\nfocused:input:AXTextField:Search",
            inputFieldHint: "Search",
            lastInputFieldDescriptor: "bundle=com.google.Chrome|window=Google|role=textfield|label=Search",
            screenshot: "base64-jpeg",
            focusStableMs: 420,
            captureConfidence: 0.82,
            visibleInputCandidateCount: 2,
            accessibilityPermission: "trusted",
            accessibilityTrusted: true,
            activeDisplayID: "1",
            targetDisplayID: "2",
            screenshotAgeMs: 380,
            screenshotSource: "display_context_cache",
            screenshotCached: true,
            screenBasisID: "basis-123"
        )

        let data = try JSONEncoder().encode(original)
        let decoded = try JSONDecoder().decode(NavigatorContextPayload.self, from: data)

        XCTAssertEqual(decoded, original)
    }

    func testNavigatorStepCodableRoundTripPreservesSystemActionFields() throws {
        let original = NavigatorStep(
            id: "volume_down",
            actionType: .systemAction,
            targetApp: "macOS",
            targetDescriptor: .init(appName: "macOS"),
            expectedOutcome: "System volume is lower",
            confidence: 0.9,
            intentConfidence: 0.84,
            riskLevel: "low",
            executionPolicy: "safe_immediate",
            fallbackPolicy: "guided_mode",
            systemCommand: "volume",
            systemValue: "down",
            systemAmount: 12,
            surface: .terminal,
            macroID: "terminal_volume_adjust",
            narration: "Lowering the system volume.",
            verifyContract: VerifyContract(expectedBundleId: "com.apple.Terminal", requireFrontmostApp: true),
            fallbackActionType: .hotkey,
            fallbackHotkey: ["f11"],
            maxLocalRetries: 1,
            timeoutMs: 1200,
            proofLevel: .basic
        )

        let data = try JSONEncoder().encode(original)
        let decoded = try JSONDecoder().decode(NavigatorStep.self, from: data)

        XCTAssertEqual(decoded, original)
    }

    func testNavigatorClarificationResponseModeCodableRoundTrip() throws {
        let original = NavigatorClarificationResponseMode.provideDetails
        let data = try JSONEncoder().encode(original)
        let decoded = try JSONDecoder().decode(NavigatorClarificationResponseMode.self, from: data)
        XCTAssertEqual(decoded, original)
    }

    func testExecutionPhaseCodableRoundTrip() throws {
        let original = ExecutionPhase.verifyOutcome
        let data = try JSONEncoder().encode(original)
        let decoded = try JSONDecoder().decode(ExecutionPhase.self, from: data)
        XCTAssertEqual(decoded, original)
    }

    func testExecutionFailureReasonCodableRoundTrip() throws {
        let original = ExecutionFailureReason.verificationInconclusive
        let data = try JSONEncoder().encode(original)
        let decoded = try JSONDecoder().decode(ExecutionFailureReason.self, from: data)
        XCTAssertEqual(decoded, original)
    }

    func testNavigatorContextPayloadWithCachedScreenshotPreservesMetadata() {
        let original = NavigatorContextPayload(
            appName: "Codex",
            bundleId: "com.openai.codex",
            frontmostBundleId: "com.openai.codex",
            windowTitle: "Codex",
            focusedRole: "AXTextArea",
            focusedLabel: "Composer",
            selectedText: "",
            axSnapshot: "snapshot",
            inputFieldHint: "Composer",
            lastInputFieldDescriptor: "label=Composer",
            screenshot: "",
            focusStableMs: 600,
            captureConfidence: 0.93,
            visibleInputCandidateCount: 1,
            accessibilityPermission: "trusted",
            accessibilityTrusted: true
        )

        let updated = original.withScreenBasis(
            screenBasisID: "basis-xyz",
            activeDisplayID: "69733056",
            targetDisplayID: "69733056",
            screenshotAgeMs: 240,
            screenshotSource: "display_context_cache",
            screenshotCached: true,
            screenshot: "base64-jpeg"
        )

        XCTAssertEqual(updated.screenshot, "base64-jpeg")
        XCTAssertEqual(updated.activeDisplayID, "69733056")
        XCTAssertEqual(updated.targetDisplayID, "69733056")
        XCTAssertEqual(updated.screenshotAgeMs, 240)
        XCTAssertEqual(updated.screenshotSource, "display_context_cache")
        XCTAssertTrue(updated.screenshotCached)
        XCTAssertEqual(updated.screenBasisID, "basis-xyz")
    }

    func testNavigatorContextPayloadWithScreenBasisPreservesMetadata() {
        let original = NavigatorContextPayload(
            appName: "Chrome",
            bundleId: "com.google.Chrome",
            frontmostBundleId: "com.google.Chrome",
            windowTitle: "Docs",
            focusedRole: "AXTextField",
            focusedLabel: "Address",
            selectedText: "",
            axSnapshot: "snapshot",
            inputFieldHint: "Address",
            lastInputFieldDescriptor: "label=Address",
            screenshot: "",
            focusStableMs: 320,
            captureConfidence: 0.8,
            visibleInputCandidateCount: 1,
            accessibilityPermission: "trusted",
            accessibilityTrusted: true
        )

        let updated = original.withScreenBasis(
            screenBasisID: "basis-456",
            activeDisplayID: "1",
            targetDisplayID: "1",
            screenshotAgeMs: 25,
            screenshotSource: "command_force_capture",
            screenshotCached: false,
            screenshot: "fresh-image"
        )

        XCTAssertEqual(updated.screenBasisID, "basis-456")
        XCTAssertEqual(updated.screenshot, "fresh-image")
        XCTAssertEqual(updated.screenshotSource, "command_force_capture")
    }
}
