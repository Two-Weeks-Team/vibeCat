import Foundation

public final class AppSettings: @unchecked Sendable {
    public static let shared = AppSettings()

    private let defaults = UserDefaults.standard

    private enum Key: String {
        case language = "vibecat.language"
        case voice = "vibecat.voice"
        case character = "vibecat.character"
        case chattiness = "vibecat.chattiness"
        case captureInterval = "vibecat.captureInterval"
        case liveModel = "vibecat.liveModel"
        case musicEnabled = "vibecat.musicEnabled"
        case gatewayURL = "vibecat.gatewayURL"
        case searchEnabled = "vibecat.searchEnabled"
        case proactiveAudio = "vibecat.proactiveAudio"
    }

    public var language: String {
        get { defaults.string(forKey: Key.language.rawValue) ?? "ko" }
        set { defaults.set(newValue, forKey: Key.language.rawValue) }
    }

    public var voice: String {
        get { defaults.string(forKey: Key.voice.rawValue) ?? "Zephyr" }
        set { defaults.set(newValue, forKey: Key.voice.rawValue) }
    }

    public var character: String {
        get { defaults.string(forKey: Key.character.rawValue) ?? "cat" }
        set { defaults.set(newValue, forKey: Key.character.rawValue) }
    }

    public var chattiness: String {
        get { defaults.string(forKey: Key.chattiness.rawValue) ?? "normal" }
        set { defaults.set(newValue, forKey: Key.chattiness.rawValue) }
    }

    public var captureInterval: Double {
        get {
            let v = defaults.double(forKey: Key.captureInterval.rawValue)
            return v > 0 ? v : 5.0
        }
        set { defaults.set(newValue, forKey: Key.captureInterval.rawValue) }
    }

    public var liveModel: String {
        get { defaults.string(forKey: Key.liveModel.rawValue) ?? "gemini-2.0-flash-live-001" }
        set { defaults.set(newValue, forKey: Key.liveModel.rawValue) }
    }

    public var musicEnabled: Bool {
        get { defaults.bool(forKey: Key.musicEnabled.rawValue) }
        set { defaults.set(newValue, forKey: Key.musicEnabled.rawValue) }
    }

    public var gatewayURL: String {
        get { defaults.string(forKey: Key.gatewayURL.rawValue) ?? "wss://realtime-gateway-163070481841.asia-northeast3.run.app" }
        set { defaults.set(newValue, forKey: Key.gatewayURL.rawValue) }
    }

    public var searchEnabled: Bool {
        get { defaults.bool(forKey: Key.searchEnabled.rawValue) }
        set { defaults.set(newValue, forKey: Key.searchEnabled.rawValue) }
    }

    public var proactiveAudio: Bool {
        get { defaults.bool(forKey: Key.proactiveAudio.rawValue) }
        set { defaults.set(newValue, forKey: Key.proactiveAudio.rawValue) }
    }

    private init() {}
}
