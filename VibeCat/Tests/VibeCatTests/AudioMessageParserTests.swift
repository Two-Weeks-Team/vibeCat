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

    func testParseProcessingStateAndToolResult() throws {
        let processingPayload = try makeJSON([
            "type": "processingState",
            "flow": "text",
            "traceId": "text_123",
            "stage": "searching",
            "label": "검색 중...",
            "detail": "Google Search 확인 중",
            "tool": "google_search",
            "sourceCount": 3,
            "active": true
        ])
        let processingMessage = AudioMessageParser.parse(processingPayload)
        guard case .processingState(let flow, let traceId, let stage, let label, let detail, let tool, let sourceCount, let active) = processingMessage else {
            XCTFail("Expected .processingState")
            return
        }
        XCTAssertEqual(flow, "text")
        XCTAssertEqual(traceId, "text_123")
        XCTAssertEqual(stage, "searching")
        XCTAssertEqual(label, "검색 중...")
        XCTAssertEqual(detail, "Google Search 확인 중")
        XCTAssertEqual(tool, "google_search")
        XCTAssertEqual(sourceCount, 3)
        XCTAssertTrue(active)

        let toolPayload = try makeJSON([
            "type": "toolResult",
            "tool": "url_context",
            "query": "이 링크 요약해줘",
            "summary": "핵심 요약",
            "sources": ["https://ai.google.dev/gemini-api/docs/url-context"]
        ])
        let toolMessage = AudioMessageParser.parse(toolPayload)
        guard case .toolResult(let toolName, let query, let summary, let sources) = toolMessage else {
            XCTFail("Expected .toolResult")
            return
        }
        XCTAssertEqual(toolName, "url_context")
        XCTAssertEqual(query, "이 링크 요약해줘")
        XCTAssertEqual(summary, "핵심 요약")
        XCTAssertEqual(sources, ["https://ai.google.dev/gemini-api/docs/url-context"])
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

    func testParseGoAway() throws {
        let payload = try makeJSON([
            "type": "goAway",
            "reason": "session_timeout",
            "timeLeftMs": 1500
        ])
        let message = AudioMessageParser.parse(payload)
        guard case .goAway(let reason, let timeLeftMs) = message else {
            XCTFail("Expected .goAway")
            return
        }
        XCTAssertEqual(reason, "session_timeout")
        XCTAssertEqual(timeLeftMs, 1500)
    }

    func testParseNavigatorCommandAccepted() throws {
        let payload = try makeJSON([
            "type": "navigator.commandAccepted",
            "taskId": "task_123",
            "command": "take me to the docs",
            "intentClass": "find_or_lookup",
            "intentConfidence": 0.87
        ])

        let message = AudioMessageParser.parse(payload)
        guard case .navigatorCommandAccepted(let taskId, let command, let intentClass, let confidence) = message else {
            XCTFail("Expected .navigatorCommandAccepted")
            return
        }
        XCTAssertEqual(taskId, "task_123")
        XCTAssertEqual(command, "take me to the docs")
        XCTAssertEqual(intentClass, .findOrLookup)
        XCTAssertEqual(confidence, 0.87, accuracy: 0.0001)
    }

    func testParseNavigatorClarificationResponseMode() throws {
        let payload = try makeJSON([
            "type": "navigator.intentClarificationNeeded",
            "command": "검색창에 입력해줘",
            "question": "Tell me the exact text to type.",
            "responseMode": "provide_details"
        ])

        let message = AudioMessageParser.parse(payload)
        guard case .navigatorIntentClarificationNeeded(let command, let question, let responseMode) = message else {
            XCTFail("Expected .navigatorIntentClarificationNeeded")
            return
        }
        XCTAssertEqual(command, "검색창에 입력해줘")
        XCTAssertEqual(question, "Tell me the exact text to type.")
        XCTAssertEqual(responseMode, .provideDetails)
    }

    func testParseNavigatorStepPlanned() throws {
        let payload = try makeJSON([
            "type": "navigator.stepPlanned",
            "taskId": "task_abc",
            "message": "I can open the relevant docs now.",
            "step": [
                "id": "open_docs_search",
                "actionType": "open_url",
                "targetApp": "Chrome",
                "targetDescriptor": [
                    "appName": "Chrome",
                    "windowTitle": "AuthServiceTests"
                ],
                "expectedOutcome": "Chrome opens the relevant official docs search",
                "confidence": 0.91,
                "intentConfidence": 0.88,
                "riskLevel": "low",
                "executionPolicy": "safe_immediate",
                "fallbackPolicy": "guided_mode",
                "url": "https://www.google.com/search?q=auth",
                "verifyHint": "google"
            ]
        ])

        let message = AudioMessageParser.parse(payload)
        guard case .navigatorStepPlanned(let taskId, let step, let messageText) = message else {
            XCTFail("Expected .navigatorStepPlanned")
            return
        }
        XCTAssertEqual(taskId, "task_abc")
        XCTAssertEqual(messageText, "I can open the relevant docs now.")
        XCTAssertEqual(step.id, "open_docs_search")
        XCTAssertEqual(step.actionType, .openURL)
        XCTAssertEqual(step.targetApp, "Chrome")
        XCTAssertEqual(step.targetDescriptor.appName, "Chrome")
        XCTAssertEqual(step.targetDescriptor.windowTitle, "AuthServiceTests")
        XCTAssertEqual(step.url, "https://www.google.com/search?q=auth")
        XCTAssertEqual(step.verifyHint, "google")
    }

    func testParseNavigatorSystemActionStep() throws {
        let payload = try makeJSON([
            "type": "navigator.stepPlanned",
            "taskId": "task_volume",
            "message": "I can apply that macOS system change now.",
            "step": [
                "id": "volume_down",
                "actionType": "system_action",
                "targetApp": "macOS",
                "targetDescriptor": [
                    "appName": "macOS"
                ],
                "expectedOutcome": "System volume is lower",
                "confidence": 0.9,
                "intentConfidence": 0.87,
                "riskLevel": "low",
                "executionPolicy": "safe_immediate",
                "fallbackPolicy": "guided_mode",
                "systemCommand": "volume",
                "systemValue": "down",
                "systemAmount": 15
            ]
        ])

        let message = AudioMessageParser.parse(payload)
        guard case .navigatorStepPlanned(_, let step, _) = message else {
            XCTFail("Expected .navigatorStepPlanned")
            return
        }
        XCTAssertEqual(step.actionType, .systemAction)
        XCTAssertEqual(step.systemCommand, "volume")
        XCTAssertEqual(step.systemValue, "down")
        XCTAssertEqual(step.systemAmount, 15)
    }

    func testParseNavigatorRiskBlockAndCompletion() throws {
        let riskPayload = try makeJSON([
            "type": "navigator.riskyActionBlocked",
            "command": "git push",
            "question": "Do you want me to proceed?",
            "reason": "blocked as a risky action"
        ])
        let riskMessage = AudioMessageParser.parse(riskPayload)
        guard case .navigatorRiskyActionBlocked(let command, let question, let reason) = riskMessage else {
            XCTFail("Expected .navigatorRiskyActionBlocked")
            return
        }
        XCTAssertEqual(command, "git push")
        XCTAssertEqual(question, "Do you want me to proceed?")
        XCTAssertEqual(reason, "blocked as a risky action")

        let completedPayload = try makeJSON([
            "type": "navigator.completed",
            "taskId": "task_done",
            "summary": "Chrome opened the official docs and Antigravity is frontmost again."
        ])
        let completedMessage = AudioMessageParser.parse(completedPayload)
        guard case .navigatorCompleted(let taskId, let summary) = completedMessage else {
            XCTFail("Expected .navigatorCompleted")
            return
        }
        XCTAssertEqual(taskId, "task_done")
        XCTAssertTrue(summary.contains("Chrome opened"))
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
