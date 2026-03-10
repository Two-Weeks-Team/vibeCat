import Foundation

public enum GeminiModels {
    public static let liveNativeAudio = "gemini-2.5-flash-native-audio-preview-12-2025"
    public static let textToSpeech = "gemini-2.5-flash-preview-tts"
    public static let defaultNonLive = "gemini-3.1-flash-lite-preview"
    public static let vision = defaultNonLive
    public static let search = defaultNonLive
    public static let liteSupport = defaultNonLive

    public static let selectableLiveModels = [
        liveNativeAudio,
    ]
}
