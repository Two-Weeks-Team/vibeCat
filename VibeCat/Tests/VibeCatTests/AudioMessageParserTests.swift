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

    func testParseTurnStateUsesDefaultsForMissingFields() throws {
        let payload = try makeJSON(["type": "turnState"])
        let message = AudioMessageParser.parse(payload)

        guard case .turnState(let state, let source) = message else {
            XCTFail("Expected .turnState")
            return
        }
        XCTAssertEqual(state, "idle")
        XCTAssertEqual(source, "live")
    }

    func testParseTraceEventIncludesElapsedAndDetail() throws {
        let payload = try makeJSON([
            "type": "traceEvent",
            "flow": "proactive",
            "traceId": "cap_123",
            "phase": "turn_started",
            "elapsedMs": 1842,
            "detail": "target=frontmost_window"
        ])
        let message = AudioMessageParser.parse(payload)

        guard case .traceEvent(let flow, let traceId, let phase, let elapsedMs, let detail) = message else {
            XCTFail("Expected .traceEvent")
            return
        }
        XCTAssertEqual(flow, "proactive")
        XCTAssertEqual(traceId, "cap_123")
        XCTAssertEqual(phase, "turn_started")
        XCTAssertEqual(elapsedMs, 1842)
        XCTAssertEqual(detail, "target=frontmost_window")
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

    // MARK: - parseEmotionTag

    func testParseEmotionTagHappy() {
        let result = AudioMessageParser.parseEmotionTag(from: "[happy] That looks right")
        XCTAssertNotNil(result)
        XCTAssertEqual(result?.emotion, .happy)
        XCTAssertEqual(result?.cleanText, "That looks right")
    }

    func testParseEmotionTagThinkingMapsToCurious() {
        let result = AudioMessageParser.parseEmotionTag(from: "[thinking] Hmm interesting")
        XCTAssertNotNil(result)
        XCTAssertEqual(result?.emotion, .curious)
        XCTAssertEqual(result?.cleanText, "Hmm interesting")
    }

    func testParseEmotionTagConcerned() {
        let result = AudioMessageParser.parseEmotionTag(from: "[concerned] That error looks bad")
        XCTAssertNotNil(result)
        XCTAssertEqual(result?.emotion, .concerned)
        XCTAssertEqual(result?.cleanText, "That error looks bad")
    }

    func testParseEmotionTagIdleMapsToNeutral() {
        let result = AudioMessageParser.parseEmotionTag(from: "[idle] ")
        XCTAssertNotNil(result)
        XCTAssertEqual(result?.emotion, .neutral)
        XCTAssertEqual(result?.cleanText, "")
    }

    func testParseEmotionTagNoTagReturnsNil() {
        let result = AudioMessageParser.parseEmotionTag(from: "No tag here")
        XCTAssertNil(result)
    }

    func testParseEmotionTagMiddlePositionReturnsNil() {
        let result = AudioMessageParser.parseEmotionTag(from: "Text [happy] in middle")
        XCTAssertNil(result)
    }

    func testParseEmotionTagSurprised() {
        let result = AudioMessageParser.parseEmotionTag(from: "[surprised] Wow!")
        XCTAssertNotNil(result)
        XCTAssertEqual(result?.emotion, .surprised)
        XCTAssertEqual(result?.cleanText, "Wow!")
    }

    private func makeJSON(_ object: [String: Any]) throws -> Data {
        try JSONSerialization.data(withJSONObject: object)
    }
}
