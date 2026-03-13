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
    private var isRestarting = false
    private var pendingRestartReason: String?
    private var restartSequence: UInt64 = 0
    private let recoveryDelays: [UInt64] = [
        0,
        300_000_000,
        1_000_000_000,
        2_000_000_000,
    ]
    private var tapHealthCheckTask: Task<Void, Never>?

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

    func handleAudioDeviceChange(reason: String) {
        guard desiredListening else {
            NSLog("[SPEECH] audio device change ignored: recognizer intentionally stopped")
            return
        }
        if isRestarting {
            NSLog("[SPEECH] restart already in progress, queuing reason=%@", reason)
            pendingRestartReason = reason
            return
        }
        Task { @MainActor [weak self] in
            await self?.ensureListening(reason: reason, forceRestart: true)
        }
    }

    private func ensureListening(reason: String, forceRestart: Bool) async {
        if isListening && !forceRestart {
            captureState = .listening
            return
        }

        guard !isRestarting else {
            NSLog("[SPEECH] ensureListening skipped: restart already in progress")
            pendingRestartReason = reason
            return
        }
        isRestarting = true

        restartSequence &+= 1
        let sequence = restartSequence

        for (attemptIndex, delay) in recoveryDelays.enumerated() {
            guard desiredListening, sequence == restartSequence else { break }

            if delay > 0 {
                captureState = .recovering(attempt: attemptIndex + 1)
                NSLog("[SPEECH] listening recovery scheduled reason=%@ attempt=%d delay_ms=%llu", reason, attemptIndex+1, delay / 1_000_000)
                try? await Task.sleep(nanoseconds: delay)
                guard desiredListening, sequence == restartSequence else { break }
            } else {
                captureState = .starting
            }

            if forceRestart || isListening || currentAudioFormat != nil {
                teardownCapture()
            }

            if await attemptStartCapture() {
                captureState = .listening
                // setVoiceProcessingEnabled emits device_list_changed which
                // queues a pendingRestartReason during startup.  Re-processing
                // it would tear down the engine we just started, causing the
                // tap to go silent.  Clear without re-dispatching.
                isRestarting = false
                pendingRestartReason = nil
                return
            }
        }

        if desiredListening, sequence == restartSequence {
            captureState = .failed
        }
        drainPendingRestart()
    }

    private func drainPendingRestart() {
        isRestarting = false
        if let reason = pendingRestartReason {
            pendingRestartReason = nil
            handleAudioDeviceChange(reason: reason)
        }
    }

    private func attemptStartCapture() async -> Bool {
        let capture = audioCapture
        let callback = onAudioBufferCaptured
        let bargeInCallback = onBargeInDetected
        NSLog("[SPEECH] startListening: hasCallback=%d", callback != nil ? 1 : 0)

        capture.onEngineConfigChange = { [weak self] in
            Task { @MainActor [weak self] in
                self?.handleAudioDeviceChange(reason: "engine_config_change")
            }
        }

        do {
            try await Task.detached {
                try capture.start(streamingCallback: callback, bargeInCallback: bargeInCallback)
            }.value

            currentAudioFormat = capture.recordingFormat
            isListening = true
            onRecordingFormatChanged?(currentAudioFormat)
            NSLog("[SPEECH] startListening: success, format=%@, isListening=%d", String(describing: currentAudioFormat), isListening ? 1 : 0)
            startTapHealthCheck()
            return true
        } catch {
            NSLog("[SPEECH] startListening: FAILED error=%@", String(describing: error))
            teardownCapture()
            return false
        }
    }

    private func teardownCapture() {
        tapHealthCheckTask?.cancel()
        tapHealthCheckTask = nil
        audioCapture.stop()
        currentAudioFormat = nil
        isListening = false
        onRecordingFormatChanged?(nil)
    }

    private func startTapHealthCheck() {
        tapHealthCheckTask?.cancel()
        tapHealthCheckTask = Task { [weak self] in
            try? await Task.sleep(nanoseconds: 3_000_000_000)
            guard !Task.isCancelled else { return }
            var lastCount = self?.audioCapture.tapFireCount ?? 0
            while !Task.isCancelled {
                try? await Task.sleep(nanoseconds: 5_000_000_000)
                guard !Task.isCancelled, let self, self.isListening else { return }
                let current = self.audioCapture.tapFireCount
                if current == lastCount {
                    NSLog("[SPEECH] tap health check FAILED: count=%llu unchanged for 5s", current)
                    self.handleAudioDeviceChange(reason: "tap_health_check")
                    return
                }
                lastCount = current
            }
        }
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
    private let lifecycleLock = NSLock()
    private(set) var tapFireCount: UInt64 = 0

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
    private var configChangeObserver: NSObjectProtocol?
    var onEngineConfigChange: (@Sendable () -> Void)?

    func start(
        streamingCallback: (@Sendable (AVAudioPCMBuffer, AudioForwardMode) -> Void)? = nil,
        bargeInCallback: (@Sendable () -> Void)? = nil
    ) throws {
        lifecycleLock.lock()
        defer { lifecycleLock.unlock() }

        // Tear down existing engine before creating a new one to prevent
        // concurrent AVAudioEngine instances fighting over the audio hardware.
        stopInternal()

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
        NSLog("[SPEECH] inputNode outputFormat: sampleRate=%.0f channels=%d", hwFormat.sampleRate, hwFormat.channelCount)
        guard hwFormat.sampleRate > 0, hwFormat.channelCount > 0 else {
            throw SpeechRecognizerError.audioFormatCreationFailed
        }

        // Create 1-channel recording format for downstream consumers.
        // VP outputs 3 channels (ch0=processed voice, ch1=ambient, ch2=reserved);
        // we extract ch0 in the tap and forward mono buffers.
        guard let monoFormat = AVAudioFormat(
            commonFormat: .pcmFormatFloat32,
            sampleRate: hwFormat.sampleRate,
            channels: 1,
            interleaved: false
        ) else {
            throw SpeechRecognizerError.audioFormatCreationFailed
        }

        self.recordingFormat = monoFormat

        self.tapFireCount = 0
        // Use explicit hwFormat for the tap — production evidence (WhisperKit,
        // LiveKit) shows format:nil can silently fail to deliver buffers when
        // Voice Processing + Bluetooth creates a VPIO aggregate device.
        // hwFormat matches inputNode.outputFormat exactly, so no conversion needed.
        inputNode.installTap(onBus: 0, bufferSize: 2048, format: hwFormat) { [weak self] buffer, _ in
            guard let self else { return }
            self.tapFireCount &+= 1
            if self.tapFireCount <= 3 || self.tapFireCount % 500 == 0 {
                NSLog("[SPEECH-TAP] fired count=%llu frames=%d ch=%d rateHz=%.0f",
                      self.tapFireCount, buffer.frameLength,
                      buffer.format.channelCount, buffer.format.sampleRate)
            }
            guard let channelData = buffer.floatChannelData else {
                NSLog("[SPEECH-TAP] dropped: nil floatChannelData")
                return
            }

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

            guard let monoFmt = self.recordingFormat,
                  let copy = AVAudioPCMBuffer(pcmFormat: monoFmt, frameCapacity: buffer.frameLength) else {
                NSLog("[SPEECH-TAP] dropped: mono format nil or buffer alloc failed")
                return
            }
            copy.frameLength = buffer.frameLength
            if let dst = copy.floatChannelData {
                dst[0].update(from: samples, count: frames)
            }
            self.streamingCallback?(copy, gate.forwardMode)
        }

        engine.prepare()
        try engine.start()
        NSLog("[SPEECH] engine started isRunning=%d vpEnabled=%d tapCount=%llu format=%.0fHz/%dch",
              engine.isRunning ? 1 : 0, inputNode.isVoiceProcessingEnabled ? 1 : 0,
              self.tapFireCount, hwFormat.sampleRate, hwFormat.channelCount)

        configChangeObserver = NotificationCenter.default.addObserver(
            forName: .AVAudioEngineConfigurationChange,
            object: engine,
            queue: nil
        ) { [weak self] _ in
            guard let self else { return }
            let running = self.engine?.isRunning == true
            NSLog("[SPEECH] AVAudioEngineConfigurationChange: isRunning=%d", running ? 1 : 0)
            if !running {
                self.onEngineConfigChange?()
            }
        }
    }

    func stop() {
        lifecycleLock.lock()
        defer { lifecycleLock.unlock() }
        stopInternal()
    }

    private func stopInternal() {
        if let observer = configChangeObserver {
            NotificationCenter.default.removeObserver(observer)
            configChangeObserver = nil
        }
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
