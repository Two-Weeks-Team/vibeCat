@preconcurrency import AVFoundation
import Foundation

enum AudioForwardMode: Sendable {
    case normal
    case bargeIn
}

@MainActor
final class SpeechRecognizer {
    enum CaptureState: Equatable, Sendable {
        case stopped
        case starting
        case recovering(attempt: Int)
        case listening
        case failed
    }

    var onAudioBufferCaptured: (@Sendable (AVAudioPCMBuffer, AudioForwardMode) -> Void)?
    var onBargeInDetected: (@Sendable () -> Void)?
    var onRecordingFormatChanged: ((AVAudioFormat?) -> Void)?
    var onCaptureStateChanged: ((CaptureState) -> Void)?
    private(set) var currentAudioFormat: AVAudioFormat?
    private(set) var isListening = false

    private let audioCapture = SpeechAudioCapture()
    private(set) var captureState: CaptureState = .stopped {
        didSet {
            guard oldValue != captureState else { return }
            onCaptureStateChanged?(captureState)
        }
    }
    private var desiredListening = false
    private var restartSequence: UInt64 = 0
    private let recoveryDelays: [UInt64] = [
        0,
        300_000_000,
        1_000_000_000,
        2_000_000_000,
    ]

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
        desiredListening = true
        await ensureListening(reason: "start", forceRestart: false)
    }

    func stopListening() {
        desiredListening = false
        restartSequence &+= 1
        teardownCapture()
        captureState = .stopped
    }

    func resumeListening() {
        desiredListening = true
        Task { await ensureListening(reason: "resume", forceRestart: false) }
    }

    func setModelSpeaking(_ speaking: Bool) {
        audioCapture.modelSpeaking = speaking
    }

    @discardableResult
    func handleAudioDeviceChange(reason: String) -> Bool {
        guard desiredListening else {
            NSLog("[SPEECH] audio device change ignored: recognizer intentionally stopped")
            return false
        }
        Task { @MainActor [weak self] in
            await self?.ensureListening(reason: reason, forceRestart: true)
        }
        return true
    }

    private func ensureListening(reason: String, forceRestart: Bool) async {
        if isListening && !forceRestart {
            captureState = .listening
            return
        }

        restartSequence &+= 1
        let sequence = restartSequence

        for (attemptIndex, delay) in recoveryDelays.enumerated() {
            guard desiredListening, sequence == restartSequence else { return }

            if delay > 0 {
                captureState = .recovering(attempt: attemptIndex + 1)
                NSLog("[SPEECH] listening recovery scheduled reason=%@ attempt=%d delay_ms=%llu", reason, attemptIndex+1, delay / 1_000_000)
                try? await Task.sleep(nanoseconds: delay)
                guard desiredListening, sequence == restartSequence else { return }
            } else {
                captureState = .starting
            }

            if forceRestart || isListening || currentAudioFormat != nil {
                teardownCapture()
            }

            if await attemptStartCapture() {
                captureState = .listening
                return
            }
        }

        guard desiredListening, sequence == restartSequence else { return }
        captureState = .failed
    }

    private func attemptStartCapture() async -> Bool {
        let capture = audioCapture
        let callback = onAudioBufferCaptured
        let bargeInCallback = onBargeInDetected
        NSLog("[SPEECH] startListening: hasCallback=%d", callback != nil ? 1 : 0)

        do {
            let format = try await capture.start(streamingCallback: callback, bargeInCallback: bargeInCallback)

            currentAudioFormat = format
            isListening = true
            onRecordingFormatChanged?(currentAudioFormat)
            NSLog("[SPEECH] startListening: success, format=%@, isListening=%d", String(describing: currentAudioFormat), isListening ? 1 : 0)
            return true
        } catch {
            NSLog("[SPEECH] startListening: FAILED error=%@", String(describing: error))
            teardownCapture()
            return false
        }
    }

    private func teardownCapture() {
        audioCapture.stop()
        currentAudioFormat = nil
        isListening = false
        onRecordingFormatChanged?(nil)
    }
}

enum SpeechRecognizerError: Error {
    case audioFormatCreationFailed
}

final class SpeechAudioCapture: @unchecked Sendable {
    private let lifecycleQueue = DispatchQueue(label: "vibecat.speech-audio-capture.lifecycle")
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
    ) async throws -> AVAudioFormat {
        try await withCheckedThrowingContinuation { continuation in
            lifecycleQueue.async { [weak self] in
                guard let self else {
                    continuation.resume(throwing: SpeechRecognizerError.audioFormatCreationFailed)
                    return
                }

                do {
                    let format = try self.startCapture(streamingCallback: streamingCallback, bargeInCallback: bargeInCallback)
                    continuation.resume(returning: format)
                } catch {
                    self.resetCaptureState()
                    continuation.resume(throwing: error)
                }
            }
        }
    }

    func stop() {
        lifecycleQueue.sync {
            resetCaptureState()
        }
    }

    private func startCapture(
        streamingCallback: (@Sendable (AVAudioPCMBuffer, AudioForwardMode) -> Void)?,
        bargeInCallback: (@Sendable () -> Void)?
    ) throws -> AVAudioFormat {
        _speakingLock.withLock {
            self.streamingCallback = streamingCallback
            self.bargeInCallback = bargeInCallback
        }

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
            let (bargeIn, streaming) = self._speakingLock.withLock {
                (self.bargeInCallback, self.streamingCallback)
            }

            if gate.shouldNotifyBargeIn {
                bargeIn?()
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
            streaming?(copy, gate.forwardMode)
        }

        engine.prepare()
        try engine.start()
        return recordingFormat
    }

    private func resetCaptureState() {
        engine?.inputNode.removeTap(onBus: 0)
        engine?.stop()
        engine = nil
        recordingFormat = nil
        isVoiceProcessingActive = false
        _speakingLock.withLock {
            streamingCallback = nil
            bargeInCallback = nil
            bargeInNotified = false
            consecutiveAboveCount = 0
        }
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
