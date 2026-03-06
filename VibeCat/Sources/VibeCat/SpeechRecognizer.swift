@preconcurrency import AVFoundation
import Foundation

@MainActor
final class SpeechRecognizer {
    var onAudioBufferCaptured: (@Sendable (AVAudioPCMBuffer) -> Void)?
    private(set) var currentAudioFormat: AVAudioFormat?
    private(set) var isListening = false

    private let audioCapture = SpeechAudioCapture()

    func requestPermissions() async -> Bool {
        await withCheckedContinuation { continuation in
            switch AVAudioApplication.shared.recordPermission {
            case .granted:
                continuation.resume(returning: true)
            case .denied:
                continuation.resume(returning: false)
            case .undetermined:
                AVAudioApplication.requestRecordPermission { granted in
                    continuation.resume(returning: granted)
                }
            @unknown default:
                continuation.resume(returning: false)
            }
        }
    }

    func startListening() async {
        guard !isListening else { return }

        let capture = audioCapture
        let callback = onAudioBufferCaptured

        do {
            try await Task.detached {
                try capture.start(streamingCallback: callback)
            }.value

            currentAudioFormat = capture.recordingFormat
            isListening = true
        } catch {
            audioCapture.stop()
            currentAudioFormat = nil
            isListening = false
        }
    }

    func stopListening() {
        guard isListening else { return }
        audioCapture.stop()
        currentAudioFormat = nil
        isListening = false
    }

    func resumeListening() {
        guard !isListening else { return }
        Task { await startListening() }
    }
}

enum SpeechRecognizerError: Error {
    case audioFormatCreationFailed
}

final class SpeechAudioCapture: @unchecked Sendable {
    private var engine: AVAudioEngine?
    private(set) var recordingFormat: AVAudioFormat?
    private(set) var isVoiceProcessingActive = false
    private var streamingCallback: (@Sendable (AVAudioPCMBuffer) -> Void)?

    /// RMS threshold for noise gate. Buffers with RMS below this are silently dropped.
    /// ~0.02 linear ≈ -34dB — filters keyboard clicks, mouse sounds, fan noise, ambient hum
    /// while passing through normal speech (typically RMS 0.03–0.3).
    private let rmsThreshold: Float = 0.02

    /// Number of consecutive above-threshold buffers required before streaming begins.
    /// Prevents single noise spikes (clicks, taps) from triggering speech detection.
    /// At 4096 samples / 16kHz ≈ 256ms per buffer, 2 buffers ≈ 512ms of sustained sound.
    private let consecutiveThreshold: Int = 2
    private var consecutiveAboveCount: Int = 0

    func start(streamingCallback: (@Sendable (AVAudioPCMBuffer) -> Void)? = nil) throws {
        self.streamingCallback = streamingCallback

        let engine = AVAudioEngine()
        self.engine = engine
        let inputNode = engine.inputNode

        do {
            try inputNode.setVoiceProcessingEnabled(true)
            isVoiceProcessingActive = true
        } catch {
            isVoiceProcessingActive = false
        }

        let hwFormat = inputNode.outputFormat(forBus: 0)
        guard hwFormat.sampleRate > 0 else {
            throw SpeechRecognizerError.audioFormatCreationFailed
        }

        guard let recordingFormat = AVAudioFormat(
            commonFormat: .pcmFormatFloat32,
            sampleRate: hwFormat.sampleRate,
            channels: 1,
            interleaved: false
        ) else {
            throw SpeechRecognizerError.audioFormatCreationFailed
        }

        self.recordingFormat = recordingFormat
        let threshold = rmsThreshold

        let requiredConsecutive = consecutiveThreshold
        inputNode.installTap(onBus: 0, bufferSize: 4096, format: recordingFormat) { [weak self] buffer, _ in
            guard let self else { return }
            guard let floatData = buffer.floatChannelData else { return }

            let frameCount = Int(buffer.frameLength)
            let samples = floatData[0]
            var sumOfSquares: Float = 0.0
            for i in 0..<frameCount {
                let sample = samples[i]
                sumOfSquares += sample * sample
            }
            let rms = sqrtf(sumOfSquares / Float(max(frameCount, 1)))

            if rms < threshold {
                self.consecutiveAboveCount = 0
                return
            }

            self.consecutiveAboveCount += 1
            if self.consecutiveAboveCount < requiredConsecutive {
                return
            }

            guard let copy = AVAudioPCMBuffer(pcmFormat: buffer.format, frameCapacity: buffer.frameLength) else { return }
            copy.frameLength = buffer.frameLength
            if let src = buffer.floatChannelData, let dst = copy.floatChannelData {
                for channel in 0..<Int(buffer.format.channelCount) {
                    dst[channel].update(from: src[channel], count: Int(buffer.frameLength))
                }
            }
            self.streamingCallback?(copy)
        }

        engine.prepare()
        try engine.start()
    }

    func stop() {
        engine?.inputNode.removeTap(onBus: 0)
        engine?.stop()
        engine = nil
        recordingFormat = nil
        streamingCallback = nil
        isVoiceProcessingActive = false
    }
}
