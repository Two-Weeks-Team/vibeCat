import Foundation

public enum ServerMessage: Sendable {
    case audio(Data)
    case transcription(text: String, finished: Bool)
    case turnComplete
    case interrupted
    case companionSpeech(text: String, emotion: String, urgency: String)
    case setupComplete(sessionId: String)
    case sessionResumptionUpdate(handle: String)
    case liveSessionReconnecting(attempt: Int, max: Int)
    case liveSessionReconnected
    case ttsStart(text: String?)
    case ttsEnd
    case pong
    case error(code: String, message: String)
    case unknown
}

public enum AudioMessageParser {
    private static let emotionTagPattern = try! NSRegularExpression(
        pattern: #"^\[(thinking|idle|surprised|happy|concerned)\]\s*"#
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
        case "happy": emotion = .happy
        case "surprised": emotion = .surprised
        case "thinking": emotion = .curious
        case "concerned": emotion = .concerned
        case "idle": emotion = .neutral
        default: return nil
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
        case "ttsStart":
            let text = json["text"] as? String
            return .ttsStart(text: text)
        case "ttsEnd":
            return .ttsEnd
        case "pong":
            return .pong
        case "error":
            let code = json["code"] as? String ?? "UNKNOWN"
            let message = json["message"] as? String ?? ""
            return .error(code: code, message: message)
        default:
            return .unknown
        }
    }
}
