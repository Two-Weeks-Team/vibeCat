import Foundation

public enum ServerMessage: Sendable {
    case audio(Data)
    case transcription(text: String, finished: Bool)
    case inputTranscription(text: String, finished: Bool)
    case turnState(state: String, source: String)
    case traceEvent(flow: String, traceId: String, phase: String, elapsedMs: Int?, detail: String)
    case processingState(flow: String, traceId: String, stage: String, label: String, detail: String, tool: String, sourceCount: Int?, active: Bool)
    case toolResult(tool: String, query: String, summary: String, sources: [String])
    case turnComplete
    case interrupted
    case companionSpeech(text: String, emotion: String, urgency: String)
    case setupComplete(sessionId: String)
    case sessionResumptionUpdate(handle: String)
    case liveSessionReconnecting(attempt: Int, max: Int)
    case liveSessionReconnected
    case goAway(reason: String, timeLeftMs: Int)
    case ttsStart(text: String?)
    case ttsEnd
    case pong
    case navigatorCommandAccepted(taskId: String?, command: String, intentClass: NavigatorIntentClass, intentConfidence: Double)
    case navigatorIntentClarificationNeeded(command: String, question: String, responseMode: NavigatorClarificationResponseMode)
    case navigatorStepPlanned(taskId: String, step: NavigatorStep, message: String)
    case navigatorStepRunning(taskId: String, stepId: String, status: String)
    case navigatorStepVerified(taskId: String, stepId: String, status: String, observedOutcome: String)
    case navigatorRiskyActionBlocked(command: String, question: String, reason: String)
    case navigatorGuidedMode(taskId: String?, reason: String, instruction: String)
    case navigatorCompleted(taskId: String, summary: String)
    case navigatorFailed(taskId: String?, reason: String)
    case error(code: String, message: String)
    case unknown
}

public enum AudioMessageParser {
    private static let emotionTagPattern = try! NSRegularExpression(
        pattern: #"^\[(\w+)\]\s*"#
    )

    public static func parseEmotionTag(from text: String) -> (emotion: CompanionEmotion, cleanText: String)? {
        let nsRange = NSRange(text.startIndex..., in: text)
        guard let match = emotionTagPattern.firstMatch(in: text, range: nsRange),
              let tagRange = Range(match.range(at: 1), in: text) else {
            return nil
        }
        let tag = String(text[tagRange]).lowercased()
        let emotion: CompanionEmotion
        switch tag {
        case "happy", "excited", "celebrating", "joyful": emotion = .happy
        case "surprised", "shocked", "amazed": emotion = .surprised
        case "thinking", "curious", "wondering", "pondering": emotion = .curious
        case "concerned", "worried", "frustrated", "angry", "annoyed": emotion = .concerned
        case "idle", "calm", "neutral", "relaxed", "peaceful", "content": emotion = .neutral
        default: emotion = .neutral
        }
        let cleanText = emotionTagPattern.stringByReplacingMatches(
            in: text, range: nsRange, withTemplate: ""
        )
        return (emotion, cleanText)
    }

    public static func parse(_ data: Data) -> ServerMessage {
        guard let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let type = json["type"] as? String else {
            return .audio(data)
        }

        switch type {
        case "transcription":
            let text = json["text"] as? String ?? ""
            let finished = json["finished"] as? Bool ?? false
            return .transcription(text: text, finished: finished)
        case "inputTranscription":
            let text = json["text"] as? String ?? ""
            let finished = json["finished"] as? Bool ?? false
            return .inputTranscription(text: text, finished: finished)
        case "turnState":
            let state = json["state"] as? String ?? "idle"
            let source = json["source"] as? String ?? "live"
            return .turnState(state: state, source: source)
        case "traceEvent":
            let flow = json["flow"] as? String ?? "unknown"
            let traceId = json["traceId"] as? String ?? ""
            let phase = json["phase"] as? String ?? "unknown"
            let elapsedMs = json["elapsedMs"] as? Int
            let detail = json["detail"] as? String ?? ""
            return .traceEvent(flow: flow, traceId: traceId, phase: phase, elapsedMs: elapsedMs, detail: detail)
        case "processingState":
            let flow = json["flow"] as? String ?? "unknown"
            let traceId = json["traceId"] as? String ?? ""
            let stage = json["stage"] as? String ?? "unknown"
            let label = json["label"] as? String ?? ""
            let detail = json["detail"] as? String ?? ""
            let tool = json["tool"] as? String ?? ""
            let sourceCount = json["sourceCount"] as? Int
            let active = json["active"] as? Bool ?? false
            return .processingState(flow: flow, traceId: traceId, stage: stage, label: label, detail: detail, tool: tool, sourceCount: sourceCount, active: active)
        case "toolResult":
            let tool = json["tool"] as? String ?? ""
            let query = json["query"] as? String ?? ""
            let summary = json["summary"] as? String ?? ""
            let sources = json["sources"] as? [String] ?? []
            return .toolResult(tool: tool, query: query, summary: summary, sources: sources)
        case "turnComplete":
            return .turnComplete
        case "interrupted":
            return .interrupted
        case "companionSpeech":
            let text = json["text"] as? String ?? ""
            let emotion = json["emotion"] as? String ?? "neutral"
            let urgency = json["urgency"] as? String ?? "normal"
            return .companionSpeech(text: text, emotion: emotion, urgency: urgency)
        case "setupComplete":
            let sessionId = json["sessionId"] as? String ?? ""
            return .setupComplete(sessionId: sessionId)
        case "sessionResumptionUpdate":
            let handle = json["sessionHandle"] as? String ?? ""
            return .sessionResumptionUpdate(handle: handle)
        case "liveSessionReconnecting":
            let attempt = json["attempt"] as? Int ?? 1
            let max = json["max"] as? Int ?? 3
            return .liveSessionReconnecting(attempt: attempt, max: max)
        case "liveSessionReconnected":
            return .liveSessionReconnected
        case "goAway":
            let reason = json["reason"] as? String ?? "unknown"
            let timeLeftMs = json["timeLeftMs"] as? Int ?? 0
            return .goAway(reason: reason, timeLeftMs: timeLeftMs)
        case "ttsStart":
            let text = json["text"] as? String
            return .ttsStart(text: text)
        case "ttsEnd":
            return .ttsEnd
        case "pong":
            return .pong
        case "navigator.commandAccepted":
            let taskId = (json["taskId"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines)
            let command = json["command"] as? String ?? ""
            let rawIntent = json["intentClass"] as? String ?? NavigatorIntentClass.ambiguous.rawValue
            let intent = NavigatorIntentClass(rawValue: rawIntent) ?? .ambiguous
            let confidence = json["intentConfidence"] as? Double ?? 0
            return .navigatorCommandAccepted(taskId: taskId?.isEmpty == true ? nil : taskId, command: command, intentClass: intent, intentConfidence: confidence)
        case "navigator.intentClarificationNeeded":
            let command = json["command"] as? String ?? ""
            let question = json["question"] as? String ?? ""
            let rawResponseMode = json["responseMode"] as? String ?? NavigatorClarificationResponseMode.confirmation.rawValue
            let responseMode = NavigatorClarificationResponseMode(rawValue: rawResponseMode) ?? .confirmation
            return .navigatorIntentClarificationNeeded(command: command, question: question, responseMode: responseMode)
        case "navigator.stepPlanned":
            let taskId = json["taskId"] as? String ?? ""
            let message = json["message"] as? String ?? ""
            guard let stepJSON = json["step"] as? [String: Any],
                  let step = parseNavigatorStep(stepJSON) else {
                return .unknown
            }
            return .navigatorStepPlanned(taskId: taskId, step: step, message: message)
        case "navigator.stepRunning":
            let taskId = json["taskId"] as? String ?? ""
            let stepId = json["stepId"] as? String ?? ""
            let status = json["status"] as? String ?? ""
            return .navigatorStepRunning(taskId: taskId, stepId: stepId, status: status)
        case "navigator.stepVerified":
            let taskId = json["taskId"] as? String ?? ""
            let stepId = json["stepId"] as? String ?? ""
            let status = json["status"] as? String ?? ""
            let observedOutcome = json["observedOutcome"] as? String ?? ""
            return .navigatorStepVerified(taskId: taskId, stepId: stepId, status: status, observedOutcome: observedOutcome)
        case "navigator.riskyActionBlocked":
            let command = json["command"] as? String ?? ""
            let question = json["question"] as? String ?? ""
            let reason = json["reason"] as? String ?? ""
            return .navigatorRiskyActionBlocked(command: command, question: question, reason: reason)
        case "navigator.guidedMode":
            let taskId = (json["taskId"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines)
            let reason = json["reason"] as? String ?? ""
            let instruction = json["instruction"] as? String ?? ""
            return .navigatorGuidedMode(taskId: taskId?.isEmpty == true ? nil : taskId, reason: reason, instruction: instruction)
        case "navigator.completed":
            let taskId = json["taskId"] as? String ?? ""
            let summary = json["summary"] as? String ?? ""
            return .navigatorCompleted(taskId: taskId, summary: summary)
        case "navigator.failed":
            let taskId = (json["taskId"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines)
            let reason = json["reason"] as? String ?? ""
            return .navigatorFailed(taskId: taskId?.isEmpty == true ? nil : taskId, reason: reason)
        case "error":
            let code = json["code"] as? String ?? "UNKNOWN"
            let message = json["message"] as? String ?? ""
            return .error(code: code, message: message)
        default:
            return .unknown
        }
    }

    private static func parseNavigatorStep(_ json: [String: Any]) -> NavigatorStep? {
        guard let id = json["id"] as? String,
              let rawActionType = json["actionType"] as? String,
              let actionType = NavigatorActionType(rawValue: rawActionType),
              let targetApp = json["targetApp"] as? String,
              let expectedOutcome = json["expectedOutcome"] as? String else {
            return nil
        }

        let descriptorJSON = json["targetDescriptor"] as? [String: Any] ?? [:]
        let targetDescriptor = NavigatorTargetDescriptor(
            role: descriptorJSON["role"] as? String,
            label: descriptorJSON["label"] as? String,
            windowTitle: descriptorJSON["windowTitle"] as? String,
            appName: descriptorJSON["appName"] as? String,
            relativeAnchor: descriptorJSON["relativeAnchor"] as? String,
            regionHint: descriptorJSON["regionHint"] as? String
        )

        let hotkey = json["hotkey"] as? [String] ?? []
        return NavigatorStep(
            id: id,
            actionType: actionType,
            targetApp: targetApp,
            targetDescriptor: targetDescriptor,
            inputText: json["inputText"] as? String,
            expectedOutcome: expectedOutcome,
            confidence: json["confidence"] as? Double ?? 0,
            intentConfidence: json["intentConfidence"] as? Double ?? 0,
            riskLevel: json["riskLevel"] as? String ?? "low",
            executionPolicy: json["executionPolicy"] as? String ?? "safe_immediate",
            fallbackPolicy: json["fallbackPolicy"] as? String ?? "guided_mode",
            url: json["url"] as? String,
            hotkey: hotkey,
            verifyHint: json["verifyHint"] as? String,
            systemCommand: json["systemCommand"] as? String,
            systemValue: json["systemValue"] as? String,
            systemAmount: json["systemAmount"] as? Int ?? 0
        )
    }
}
