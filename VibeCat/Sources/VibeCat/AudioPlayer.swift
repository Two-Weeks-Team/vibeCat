import Foundation
import AVFoundation
import VibeCatCore

/// Streams PCM audio buffers from the Gateway through AVAudioEngine.
/// Accepts 24kHz 16-bit mono PCM from the server.
@MainActor
final class AudioPlayer {
    private let engine = AVAudioEngine()
    private let playerNode = AVAudioPlayerNode()
    private let outputFormat: AVAudioFormat

    private(set) var isPlaying = false
    private var isMuted = false

    init() {
        outputFormat = AVAudioFormat(
            commonFormat: .pcmFormatInt16,
            sampleRate: 24000,
            channels: 1,
            interleaved: true
        )!
        setupEngine()
    }

    func enqueue(_ pcmData: Data) {
        guard !isMuted else { return }
        guard let buffer = makeBuffer(from: pcmData) else { return }
        playerNode.scheduleBuffer(buffer) { [weak self] in
            Task { @MainActor [weak self] in
                self?.isPlaying = self?.playerNode.isPlaying ?? false
            }
        }
        if !playerNode.isPlaying {
            playerNode.play()
            isPlaying = true
        }
    }

    func stop() {
        playerNode.stop()
        isPlaying = false
    }

    func mute() {
        isMuted = true
        stop()
    }

    func unmute() {
        isMuted = false
    }

    private func setupEngine() {
        engine.attach(playerNode)
        engine.connect(playerNode, to: engine.mainMixerNode, format: outputFormat)
        do {
            try engine.start()
        } catch {
            print("AudioPlayer: engine start failed: \(error)")
        }
    }

    private func makeBuffer(from data: Data) -> AVAudioPCMBuffer? {
        let frameCount = AVAudioFrameCount(data.count / 2)
        guard frameCount > 0,
              let buffer = AVAudioPCMBuffer(pcmFormat: outputFormat, frameCapacity: frameCount) else {
            return nil
        }
        buffer.frameLength = frameCount
        data.withUnsafeBytes { rawPtr in
            guard let int16Ptr = rawPtr.bindMemory(to: Int16.self).baseAddress,
                  let channelData = buffer.int16ChannelData?[0] else { return }
            channelData.update(from: int16Ptr, count: Int(frameCount))
        }
        return buffer
    }
}
