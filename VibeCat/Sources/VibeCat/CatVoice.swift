import Foundation
import AVFoundation
import VibeCatCore

@MainActor
final class CatVoice {
    private let audioPlayer: AudioPlayer
    private let synthesizer = AVSpeechSynthesizer()
    private var isLiveConnected = false

    init(audioPlayer: AudioPlayer) {
        self.audioPlayer = audioPlayer
    }

    func setLiveConnected(_ connected: Bool) {
        isLiveConnected = connected
    }

    func enqueueAudio(_ pcmData: Data) {
        audioPlayer.enqueue(pcmData)
    }

    func speak(_ text: String) {
        guard !isLiveConnected else { return }
        let utterance = AVSpeechUtterance(string: text)
        utterance.rate = 0.5
        utterance.volume = 0.9
        synthesizer.speak(utterance)
    }

    func stop() {
        audioPlayer.stop()
        synthesizer.stopSpeaking(at: .immediate)
    }

    func mute() {
        audioPlayer.mute()
        synthesizer.stopSpeaking(at: .immediate)
    }

    func unmute() {
        audioPlayer.unmute()
    }
}
