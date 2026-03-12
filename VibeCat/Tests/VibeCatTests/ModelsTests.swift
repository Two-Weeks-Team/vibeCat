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
            accessibilityTrusted: true
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
            systemAmount: 12
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
}
