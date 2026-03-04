import Foundation
import XCTest
@testable import VibeCatCore

final class AudioMessageParserTests: XCTestCase {
    func testParseInvalidJSONReturnsAudioPayload() {
        let raw = Data([0x00, 0x01, 0x02])
        let message = AudioMessageParser.parse(raw)

        guard case .audio(let data) = message else {
            XCTFail("Expected .audio for non-JSON payload")
            return
        }
        XCTAssertEqual(data, raw)
    }

    func testParseTranscriptionUsesDefaultsForMissingFields() throws {
        let payload = try makeJSON(["type": "transcription", "text": "hello"])
        let message = AudioMessageParser.parse(payload)

        guard case .transcription(let text, let finished) = message else {
            XCTFail("Expected .transcription")
            return
        }
        XCTAssertEqual(text, "hello")
        XCTAssertFalse(finished)
    }

    func testParseCompanionSpeechUsesDefaultEmotionAndUrgency() throws {
        let payload = try makeJSON(["type": "companionSpeech", "text": "great job"])
        let message = AudioMessageParser.parse(payload)

        guard case .companionSpeech(let text, let emotion, let urgency) = message else {
            XCTFail("Expected .companionSpeech")
            return
        }
        XCTAssertEqual(text, "great job")
        XCTAssertEqual(emotion, "neutral")
        XCTAssertEqual(urgency, "normal")
    }

    func testParseSetupCompleteAndError() throws {
        let setupPayload = try makeJSON(["type": "setupComplete", "sessionId": "s-123"])
        let setupMessage = AudioMessageParser.parse(setupPayload)
        guard case .setupComplete(let sessionId) = setupMessage else {
            XCTFail("Expected .setupComplete")
            return
        }
        XCTAssertEqual(sessionId, "s-123")

        let errorPayload = try makeJSON(["type": "error", "message": "boom"])
        let errorMessage = AudioMessageParser.parse(errorPayload)
        guard case .error(let code, let message) = errorMessage else {
            XCTFail("Expected .error")
            return
        }
        XCTAssertEqual(code, "UNKNOWN")
        XCTAssertEqual(message, "boom")
    }

    func testParseUnknownTypeReturnsUnknown() throws {
        let payload = try makeJSON(["type": "somethingElse"])
        let message = AudioMessageParser.parse(payload)

        guard case .unknown = message else {
            XCTFail("Expected .unknown")
            return
        }
    }

    private func makeJSON(_ object: [String: Any]) throws -> Data {
        try JSONSerialization.data(withJSONObject: object)
    }
}
