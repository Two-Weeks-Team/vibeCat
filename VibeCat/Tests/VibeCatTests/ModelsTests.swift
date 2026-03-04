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
}
