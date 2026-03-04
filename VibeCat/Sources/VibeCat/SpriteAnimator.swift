import AppKit
import VibeCatCore

@MainActor
final class SpriteAnimator {
    enum AnimationState: String {
        case idle, thinking, happy, surprised, frustrated, celebrating
    }

    private var frames: [AnimationState: [NSImage]] = [:]
    private var currentState: AnimationState = .idle
    private var currentFrame = 0
    private var timer: Timer?
    private var idleBehaviorTimer: Timer?
    private var idleOverrideTimer: Timer?
    private var character: String
    private let sessionStartedAt = Date()
    private var idleOverrideState: AnimationState?
    private var isStretchingOverride = false

    var onFrameUpdate: ((NSImage) -> Void)?
    var onStateTransition: ((AnimationState, AnimationState) -> Void)?

    init(character: String = "cat") {
        self.character = character
        loadFrames(for: character)
        startAnimation()
    }

    func setState(_ state: AnimationState) {
        guard state != currentState else { return }
        let previous = currentState
        currentState = state
        currentFrame = 0

        if state != .idle {
            clearIdleOverride()
        }

        if previous == .celebrating, state == .idle {
            applyIdleOverride(state: .happy, duration: 4.0, stretching: false)
        }

        restartAnimationTimer()
        onStateTransition?(previous, state)
    }

    func setCharacter(_ newCharacter: String) {
        guard newCharacter != character else { return }
        character = newCharacter
        loadFrames(for: newCharacter)
        currentFrame = 0
    }

    func loadPreset(for character: String) -> (voice: String, size: String?, soul: String?) {
        let repoRoot = findRepoRoot()
        let spriteDir = repoRoot.appendingPathComponent("Assets/Sprites/\(character)")
        let presetURL = spriteDir.appendingPathComponent("preset.json")

        let fallbackVoice = AppSettings.shared.voice
        var voice = fallbackVoice
        var size: String?
        var soulRef = "soul.md"

        if let data = try? Data(contentsOf: presetURL),
           let preset = try? JSONDecoder().decode(CharacterPreset.self, from: data) {
            voice = preset.voice
            size = preset.size
            if let candidate = preset.soulRef, !candidate.isEmpty {
                soulRef = candidate
            }
        }

        let soulURL = spriteDir.appendingPathComponent(soulRef)
        let soul = try? String(contentsOf: soulURL, encoding: .utf8)
        return (voice: voice, size: size, soul: soul)
    }

    private func loadFrames(for char: String) {
        frames.removeAll()
        let repoRoot = findRepoRoot()
        let spriteDir = repoRoot.appendingPathComponent("Assets/Sprites/\(char)")

        for state in AnimationState.allCases {
            var stateFrames: [NSImage] = []
            for i in 1...16 {
                let filename = "\(state.rawValue)_\(String(format: "%02d", i)).png"
                let path = spriteDir.appendingPathComponent(filename)
                if let img = NSImage(contentsOf: path) {
                    stateFrames.append(img)
                }
            }
            if !stateFrames.isEmpty {
                frames[state] = stateFrames
            }
        }

        if frames.isEmpty {
            if let fallback = NSImage(systemSymbolName: "cat.fill", accessibilityDescription: char) {
                frames[.idle] = [fallback]
            }
        }
    }

    private func startAnimation() {
        restartAnimationTimer()
        scheduleIdleBehaviorTick()
    }

    private func restartAnimationTimer() {
        timer?.invalidate()
        timer = Timer.scheduledTimer(withTimeInterval: currentFrameInterval(), repeats: true) { [weak self] _ in
            Task { @MainActor [weak self] in
                self?.advanceFrame()
            }
        }
    }

    func stop() {
        timer?.invalidate()
        timer = nil
        idleBehaviorTimer?.invalidate()
        idleBehaviorTimer = nil
        idleOverrideTimer?.invalidate()
        idleOverrideTimer = nil
    }

    private func advanceFrame() {
        let renderState = idleOverrideState ?? currentState
        let stateFrames = frames[renderState] ?? frames[.idle] ?? []
        guard !stateFrames.isEmpty else { return }
        currentFrame = (currentFrame + 1) % stateFrames.count
        onFrameUpdate?(stateFrames[currentFrame])
    }

    private func scheduleIdleBehaviorTick() {
        idleBehaviorTimer?.invalidate()
        let interval = TimeInterval.random(in: 30...60)
        idleBehaviorTimer = Timer.scheduledTimer(withTimeInterval: interval, repeats: false) { [weak self] _ in
            Task { @MainActor [weak self] in
                self?.runIdleBehaviorTick()
                self?.scheduleIdleBehaviorTick()
            }
        }
    }

    private func runIdleBehaviorTick() {
        guard currentState == .idle, idleOverrideState == nil else { return }

        let hour = Calendar.current.component(.hour, from: Date())
        let lateNight = hour >= 23 || hour < 5
        let sessionDuration = Date().timeIntervalSince(sessionStartedAt)
        let longSession = sessionDuration >= 2 * 60 * 60

        if lateNight, Double.random(in: 0...1) < 0.35 {
            applyIdleOverride(state: .frustrated, duration: 6.0, stretching: false)
            return
        }

        if longSession, Double.random(in: 0...1) < 0.35 {
            applyIdleOverride(state: .thinking, duration: 8.0, stretching: true)
        }
    }

    private func applyIdleOverride(state: AnimationState, duration: TimeInterval, stretching: Bool) {
        idleOverrideState = state
        isStretchingOverride = stretching
        currentFrame = 0
        restartAnimationTimer()

        idleOverrideTimer?.invalidate()
        idleOverrideTimer = Timer.scheduledTimer(withTimeInterval: duration, repeats: false) { [weak self] _ in
            Task { @MainActor [weak self] in
                self?.clearIdleOverride()
            }
        }
    }

    private func clearIdleOverride() {
        idleOverrideTimer?.invalidate()
        idleOverrideTimer = nil
        idleOverrideState = nil
        isStretchingOverride = false
        currentFrame = 0
        restartAnimationTimer()
    }

    private func currentFrameInterval() -> TimeInterval {
        if isStretchingOverride { return 0.2 }
        return 0.12
    }

    private func findRepoRoot() -> URL {
        var url = URL(fileURLWithPath: FileManager.default.currentDirectoryPath)
        for _ in 0..<6 {
            if FileManager.default.fileExists(atPath: url.appendingPathComponent("Assets/Sprites").path) {
                return url
            }
            url = url.deletingLastPathComponent()
        }
        var bundleURL = Bundle.main.bundleURL
        for _ in 0..<6 {
            if FileManager.default.fileExists(atPath: bundleURL.appendingPathComponent("Assets/Sprites").path) {
                return bundleURL
            }
            bundleURL = bundleURL.deletingLastPathComponent()
        }
        return URL(fileURLWithPath: FileManager.default.currentDirectoryPath)
    }
}

private struct CharacterPreset: Decodable {
    let voice: String
    let size: String?
    let soulRef: String?
}

extension SpriteAnimator.AnimationState: CaseIterable {}
