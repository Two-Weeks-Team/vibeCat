@preconcurrency import AVFoundation
import Foundation

final class AudioConversionState: @unchecked Sendable {
    private let lock = NSLock()
    private var converter: AVAudioConverter?
    private var isTransitioning = false
    private var generation: UInt64 = 0

    func beginDeviceTransition() {
        lock.withLock {
            isTransitioning = true
            generation &+= 1
        }
    }

    func finishDeviceTransition() {
        lock.withLock {
            isTransitioning = false
        }
    }

    func clearConverter() {
        lock.withLock {
            converter = nil
        }
    }

    func configureConverter(for inputFormat: AVAudioFormat) -> AVAudioConverter? {
        guard let output = AVAudioFormat(commonFormat: .pcmFormatInt16, sampleRate: 16000, channels: 1, interleaved: true),
              let converter = AVAudioConverter(from: inputFormat, to: output) else {
            return nil
        }

        lock.withLock {
            self.converter = converter
            isTransitioning = false
        }
        return converter
    }

    func snapshot() -> (converter: AVAudioConverter?, generation: UInt64, isTransitioning: Bool) {
        lock.withLock {
            (converter, generation, isTransitioning)
        }
    }

    func isCurrentGeneration(_ generation: UInt64) -> Bool {
        lock.withLock {
            self.generation == generation
        }
    }
}
