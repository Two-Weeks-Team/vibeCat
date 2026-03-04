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
    private var character: String

    var onFrameUpdate: ((NSImage) -> Void)?

    init(character: String = "cat") {
        self.character = character
        loadFrames(for: character)
        startAnimation()
    }

    func setState(_ state: AnimationState) {
        guard state != currentState else { return }
        currentState = state
        currentFrame = 0
    }

    func setCharacter(_ newCharacter: String) {
        guard newCharacter != character else { return }
        character = newCharacter
        loadFrames(for: newCharacter)
        currentFrame = 0
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
        timer = Timer.scheduledTimer(withTimeInterval: 0.12, repeats: true) { [weak self] _ in
            Task { @MainActor [weak self] in
                self?.advanceFrame()
            }
        }
    }

    func stop() {
        timer?.invalidate()
        timer = nil
    }

    private func advanceFrame() {
        let stateFrames = frames[currentState] ?? frames[.idle] ?? []
        guard !stateFrames.isEmpty else { return }
        currentFrame = (currentFrame + 1) % stateFrames.count
        onFrameUpdate?(stateFrames[currentFrame])
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

extension SpriteAnimator.AnimationState: CaseIterable {}
