import Foundation
import AVFoundation
import VibeCatCore

@MainActor
final class AudioPlayer {
    private let engine = AVAudioEngine()
    private let playerNode = AVAudioPlayerNode()
    private let outputFormat: AVAudioFormat

    private(set) var isPlaying = false
    private var isMuted = false
    private var scheduledBufferCount = 0
    private var pendingBytes = Data()
    private let coalesceThreshold = 960 // ~20ms at 24kHz 16-bit mono (480 samples * 2 bytes)

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
        pendingBytes.append(pcmData)
        guard pendingBytes.count >= coalesceThreshold else { return }
        scheduleAccumulatedSamples()
    }

    func flush() {
        guard !pendingBytes.isEmpty else { return }
        scheduleAccumulatedSamples()
    }

    func stop() {
        NSLog("[AUDIO] stop")
        pendingBytes.removeAll(keepingCapacity: true)
        playerNode.stop()
        scheduledBufferCount = 0
        isPlaying = false
    }

    func clear() {
        NSLog("[AUDIO] clear")
        pendingBytes.removeAll(keepingCapacity: true)
        playerNode.stop()
        playerNode.reset()
        scheduledBufferCount = 0
        isPlaying = false
    }

    func mute() {
        NSLog("[AUDIO] mute")
        isMuted = true
        stop()
    }

    func unmute() {
        NSLog("[AUDIO] unmute")
        isMuted = false
    }

    private func scheduleAccumulatedSamples() {
        let data = pendingBytes
        pendingBytes.removeAll(keepingCapacity: true)

        guard let buffer = makeBuffer(from: data) else { return }

        scheduledBufferCount += 1
        isPlaying = true

        playerNode.scheduleBuffer(buffer) { [weak self] in
            Task { @MainActor [weak self] in
                guard let self else { return }
                self.scheduledBufferCount -= 1
                if self.scheduledBufferCount <= 0 {
                    self.scheduledBufferCount = 0
                    self.isPlaying = false
                }
            }
        }

        if !playerNode.isPlaying {
            playerNode.play()
        }
    }

    private func setupEngine() {
        engine.attach(playerNode)
        engine.connect(playerNode, to: engine.mainMixerNode, format: outputFormat)
        do {
            try engine.start()
            NSLog("[AUDIO] setupEngine: success")
        } catch {
            NSLog("[AUDIO] setupEngine: failed - %@", error.localizedDescription)
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
