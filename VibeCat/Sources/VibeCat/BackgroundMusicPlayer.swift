import Foundation
import AVFoundation
import VibeCatCore

@MainActor
final class BackgroundMusicPlayer {
    private var player: AVAudioPlayer?
    private var isEnabled: Bool

    init() {
        isEnabled = AppSettings.shared.musicEnabled
        if isEnabled { loadAndPlay() }
    }

    func setEnabled(_ enabled: Bool) {
        isEnabled = enabled
        AppSettings.shared.musicEnabled = enabled
        if enabled { loadAndPlay() } else { stop() }
    }

    private func loadAndPlay() {
        let repoRoot = findRepoRoot()
        let musicDir = repoRoot.appendingPathComponent("Assets/Music")
        guard let files = try? FileManager.default.contentsOfDirectory(
            at: musicDir, includingPropertiesForKeys: nil
        ), let first = files.first else { return }

        do {
            let p = try AVAudioPlayer(contentsOf: first)
            p.numberOfLoops = -1
            p.volume = 0.3
            p.play()
            self.player = p
        } catch {}
    }

    private func stop() {
        player?.stop()
        player = nil
    }

    private func findRepoRoot() -> URL {
        var url = URL(fileURLWithPath: FileManager.default.currentDirectoryPath)
        for _ in 0..<6 {
            if FileManager.default.fileExists(atPath: url.appendingPathComponent("Assets/Music").path) {
                return url
            }
            url = url.deletingLastPathComponent()
        }
        var bundleURL = Bundle.main.bundleURL
        for _ in 0..<6 {
            if FileManager.default.fileExists(atPath: bundleURL.appendingPathComponent("Assets/Music").path) {
                return bundleURL
            }
            bundleURL = bundleURL.deletingLastPathComponent()
        }
        return URL(fileURLWithPath: FileManager.default.currentDirectoryPath)
    }
}
