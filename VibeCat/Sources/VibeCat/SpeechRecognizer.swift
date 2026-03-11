@preconcurrency import AVFoundation
import Foundation

enum AudioForwardMode: Sendable {
    case normal
    case bargeIn
}

@MainActor
final class SpeechRecognizer {
    var onAudioBufferCaptured: (@Sendable (AVAudioPCMBuffer, AudioForwardMode) -> Void)?
    var onBargeInDetected: (@Sendable () -> Void)?
    var onRecordingFormatChanged: ((AVAudioFormat?) -> Void)?
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
        let bargeInCallback = onBargeInDetected
        NSLog("[SPEECH] startListening: hasCallback=%d", callback != nil ? 1 : 0)

        do {
            try await Task.detached {
                try capture.start(streamingCallback: callback, bargeInCallback: bargeInCallback)
            }.value

            currentAudioFormat = capture.recordingFormat
            isListening = true
            onRecordingFormatChanged?(currentAudioFormat)
            NSLog("[SPEECH] startListening: success, format=%@, isListening=%d", String(describing: currentAudioFormat), isListening ? 1 : 0)
        } catch {
            NSLog("[SPEECH] startListening: FAILED error=%@", String(describing: error))
            audioCapture.stop()
            currentAudioFormat = nil
            isListening = false
            onRecordingFormatChanged?(nil)
        }
    }

    func stopListening() {
        guard isListening else { return }
        audioCapture.stop()
        currentAudioFormat = nil
        isListening = false
        onRecordingFormatChanged?(nil)
    }

    func resumeListening() {
        guard !isListening else { return }
        Task { await startListening() }
    }

    func setModelSpeaking(_ speaking: Bool) {
        audioCapture.modelSpeaking = speaking
    }

    func handleAudioDeviceChange() async {
        guard isListening else {
            NSLog("[SPEECH] audio device change ignored: recognizer already stopped")
            return
        }
        NSLog("[SPEECH] audio device change detected; restarting capture")
        stopListening()
        await startListening()
    }
}

enum SpeechRecognizerError: Error {
    case audioFormatCreationFailed
}

final class SpeechAudioCapture: @unchecked Sendable {
    private var engine: AVAudioEngine?
    private(set) var recordingFormat: AVAudioFormat?
    private(set) var isVoiceProcessingActive = false
    private var streamingCallback: (@Sendable (AVAudioPCMBuffer, AudioForwardMode) -> Void)?

    private let bargeInThreshold: Float = 0.04
    private let _speakingLock = NSLock()
    private var _modelSpeaking: Bool = false
    var modelSpeaking: Bool {
        get { _speakingLock.withLock { _modelSpeaking } }
        set { _speakingLock.withLock { _modelSpeaking = newValue } }
    }
    private let consecutiveThreshold: Int = 2
    private var consecutiveAboveCount: Int = 0
    private var bargeInNotified = false
    private var bargeInCallback: (@Sendable () -> Void)?

    func start(
        streamingCallback: (@Sendable (AVAudioPCMBuffer, AudioForwardMode) -> Void)? = nil,
        bargeInCallback: (@Sendable () -> Void)? = nil
    ) throws {
        self.streamingCallback = streamingCallback
        self.bargeInCallback = bargeInCallback

        let engine = AVAudioEngine()
        self.engine = engine
        let inputNode = engine.inputNode

        do {
            try inputNode.setVoiceProcessingEnabled(true)
            isVoiceProcessingActive = true
            NSLog("[SPEECH] voice processing enabled")
        } catch {
            isVoiceProcessingActive = false
            NSLog("[SPEECH] voice processing unavailable: %@", String(describing: error))
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

        inputNode.installTap(onBus: 0, bufferSize: 2048, format: recordingFormat) { [weak self] buffer, _ in
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

            let gate = self.evaluateAudioGate(rms: rms)
            if gate.shouldNotifyBargeIn {
                self.bargeInCallback?()
            }
            guard gate.shouldForward else {
                return
            }

            guard let copy = AVAudioPCMBuffer(pcmFormat: buffer.format, frameCapacity: buffer.frameLength) else { return }
            copy.frameLength = buffer.frameLength
            if let src = buffer.floatChannelData, let dst = copy.floatChannelData {
                for channel in 0..<Int(buffer.format.channelCount) {
                    dst[channel].update(from: src[channel], count: Int(buffer.frameLength))
                }
            }
            self.streamingCallback?(copy, gate.forwardMode)
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
        bargeInCallback = nil
        isVoiceProcessingActive = false
        bargeInNotified = false
        consecutiveAboveCount = 0
    }

    private func evaluateAudioGate(rms: Float) -> (shouldForward: Bool, shouldNotifyBargeIn: Bool, forwardMode: AudioForwardMode) {
        _speakingLock.withLock {
            if _modelSpeaking {
                guard rms >= bargeInThreshold else {
                    consecutiveAboveCount = 0
                    bargeInNotified = false
                    return (false, false, .normal)
                }

                consecutiveAboveCount += 1
                if consecutiveAboveCount == 1 {
                    NSLog("[SPEECH] barge-in candidate rms=%.4f threshold=%.4f", rms, bargeInThreshold)
                }
                guard consecutiveAboveCount >= consecutiveThreshold else {
                    return (false, false, .normal)
                }

                let shouldNotify = !bargeInNotified
                if shouldNotify {
                    NSLog("[SPEECH] barge-in confirmed rms=%.4f frames=%d", rms, consecutiveAboveCount)
                }
                bargeInNotified = true
                return (true, shouldNotify, .bargeIn)
            }

            consecutiveAboveCount = 0
            bargeInNotified = false
            // Gemini Live automatic VAD expects a continuous audio stream so it can
            // detect end-of-speech from trailing silence after the user stops talking.
            // If we drop silent frames here, short barge-in utterances never complete.
            return (true, false, .normal)
        }
    }
}
