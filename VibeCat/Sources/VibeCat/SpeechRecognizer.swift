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
        NSLog("[SPEECH] startListening: hasCallback=%d", callback != nil ? 1 : 0)

        do {
            try await Task.detached {
                try capture.start(streamingCallback: callback)
            }.value

            currentAudioFormat = capture.recordingFormat
            isListening = true
            NSLog("[SPEECH] startListening: success, format=%@, isListening=%d", String(describing: currentAudioFormat), isListening ? 1 : 0)
        } catch {
            NSLog("[SPEECH] startListening: FAILED error=%@", String(describing: error))
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

    func setModelSpeaking(_ speaking: Bool) {
        audioCapture.modelSpeaking = speaking
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

    private let rmsThreshold: Float = 0.003

    /// Higher threshold during model speech: echo ~0.005-0.03, user speech ~0.03-0.5.
    private let bargeInThreshold: Float = 0.025

    /// Written from main thread, read from audio thread.
    /// Aligned Bool on ARM64 has natural atomicity for single-writer/single-reader.
    var modelSpeaking: Bool = false

    /// Number of consecutive above-threshold buffers required before streaming begins.
    /// At 4096 samples / 44100 Hz ≈ 93ms per buffer, 1 buffer passes immediately.
    private let consecutiveThreshold: Int = 1
    private var consecutiveAboveCount: Int = 0

    func start(streamingCallback: (@Sendable (AVAudioPCMBuffer) -> Void)? = nil) throws {
        self.streamingCallback = streamingCallback

        let engine = AVAudioEngine()
        self.engine = engine
        let inputNode = engine.inputNode

        do {
            try inputNode.setVoiceProcessingEnabled(false)
            isVoiceProcessingActive = false
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

        inputNode.installTap(onBus: 0, bufferSize: 4096, format: recordingFormat) { [weak self] buffer, _ in
            guard let self else { return }
            guard let channelData = buffer.floatChannelData else { return }

            let frames = Int(buffer.frameLength)
            let samples = channelData[0]
            var sumSquares: Float = 0
            for i in 0..<frames {
                let s = samples[i]
                sumSquares += s * s
            }
            let rms = sqrtf(sumSquares / Float(max(frames, 1)))
            // During model speech, use higher threshold to block echo while allowing barge-in
            let threshold = self.modelSpeaking ? self.bargeInThreshold : self.rmsThreshold
            guard rms >= threshold else { return }

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
