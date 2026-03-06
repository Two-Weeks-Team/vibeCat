import Foundation
import AVFoundation
import VibeCatCore

@MainActor
final class CatVoice {
    private let audioPlayer: AudioPlayer
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

    func flush() {
        audioPlayer.flush()
    }

    func stop() {
        audioPlayer.clear()
    }

    func mute() {
        audioPlayer.mute()
    }

    func unmute() {
        audioPlayer.unmute()
    }
}
