import Foundation

public struct ChatMessage: Identifiable, Sendable {
    public let id: UUID
    public let role: Role
    public let text: String
    public let timestamp: Date

    public enum Role: String, Sendable {
        case user
        case companion
        case system
    }

    public init(id: UUID = UUID(), role: Role, text: String, timestamp: Date = Date()) {
        self.id = id
        self.role = role
        self.text = text
        self.timestamp = timestamp
    }
}

public struct CharacterPresetConfig: Codable, Sendable {
    public let name: String
    public let voice: String
    public let language: String
    public let size: String?
    public let persona: String?

    public init(name: String, voice: String, language: String, size: String? = nil, persona: String? = nil) {
        self.name = name
        self.voice = voice
        self.language = language
        self.size = size
        self.persona = persona
    }
}

public enum CompanionEmotion: String, Sendable {
    case neutral
    case curious
    case happy
    case surprised
    case concerned
    case celebrating
}

public enum CompanionMood: String, Sendable {
    case focused
    case frustrated
    case stuck
    case idle
}

public struct CompanionSpeechEvent: Sendable {
    public let text: String
    public let emotion: CompanionEmotion
    public let urgency: String

    public init(text: String, emotion: CompanionEmotion = .neutral, urgency: String = "normal") {
        self.text = text
        self.emotion = emotion
        self.urgency = urgency
    }
}
