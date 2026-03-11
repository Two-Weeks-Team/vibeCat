import Foundation

public enum CaptureTargetMode: String, CaseIterable, Sendable {
    case windowUnderCursor = "window_under_cursor"
    case frontmostWindow = "frontmost_window"
    case display = "display"

    public var menuTitle: String {
        VibeCatL10n.captureTargetModeTitle(self)
    }
}

public final class AppSettings: @unchecked Sendable {
    public static let shared = AppSettings()

    private let defaults = UserDefaults.standard

    private enum Key: String {
        case language = "vibecat.language"
        case voice = "vibecat.voice"
        case character = "vibecat.character"
        case chattiness = "vibecat.chattiness"
        case captureInterval = "vibecat.captureInterval"
        case captureTargetMode = "vibecat.captureTargetMode"
        case liveModel = "vibecat.liveModel"
        case musicEnabled = "vibecat.musicEnabled"
        case gatewayURL = "vibecat.gatewayURL"
        case searchEnabled = "vibecat.searchEnabled"
        case proactiveAudio = "vibecat.proactiveAudio"
        case manualAnalysisOnly = "vibecat.manualAnalysisOnly"
    }

    public var language: String {
        get { AppLanguage.resolve(defaults.string(forKey: Key.language.rawValue)).rawValue }
        set { defaults.set(AppLanguage.resolve(newValue).rawValue, forKey: Key.language.rawValue) }
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
            return v >= 1.0 ? v : 1.0
        }
        set { defaults.set(newValue, forKey: Key.captureInterval.rawValue) }
    }

    public var captureTargetMode: CaptureTargetMode {
        get {
            guard let raw = defaults.string(forKey: Key.captureTargetMode.rawValue),
                  let value = CaptureTargetMode(rawValue: raw) else {
                return .windowUnderCursor
            }
            return value
        }
        set { defaults.set(newValue.rawValue, forKey: Key.captureTargetMode.rawValue) }
    }

    public var liveModel: String {
        get { defaults.string(forKey: Key.liveModel.rawValue) ?? GeminiModels.liveNativeAudio }
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
        get {
            if defaults.object(forKey: Key.searchEnabled.rawValue) == nil { return true }
            return defaults.bool(forKey: Key.searchEnabled.rawValue)
        }
        set { defaults.set(newValue, forKey: Key.searchEnabled.rawValue) }
    }

    public var proactiveAudio: Bool {
        get {
            if defaults.object(forKey: Key.proactiveAudio.rawValue) == nil { return true }
            return defaults.bool(forKey: Key.proactiveAudio.rawValue)
        }
        set { defaults.set(newValue, forKey: Key.proactiveAudio.rawValue) }
    }

    public var manualAnalysisOnly: Bool {
        get { defaults.bool(forKey: Key.manualAnalysisOnly.rawValue) }
        set { defaults.set(newValue, forKey: Key.manualAnalysisOnly.rawValue) }
    }

    private init() {}
}
