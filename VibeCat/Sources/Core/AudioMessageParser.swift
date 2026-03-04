import Foundation

public enum ServerMessage: Sendable {
    case audio(Data)
    case transcription(text: String, finished: Bool)
    case turnComplete
    case interrupted
    case companionSpeech(text: String, emotion: String, urgency: String)
    case setupComplete(sessionId: String)
    case error(code: String, message: String)
    case unknown
}

public enum AudioMessageParser {
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
        case "error":
            let code = json["code"] as? String ?? "UNKNOWN"
            let message = json["message"] as? String ?? ""
            return .error(code: code, message: message)
        default:
            return .unknown
        }
    }
}
