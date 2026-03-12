import Foundation
import CoreGraphics
import VibeCatCore

protocol ScreenAnalyzerEncoding: Sendable {
    func fastPathJPEG(_ image: CGImage) -> Data?
    func base64JPEG(_ image: CGImage) -> String?
}

struct DefaultScreenAnalyzerEncoder: ScreenAnalyzerEncoding, Sendable {
    func fastPathJPEG(_ image: CGImage) -> Data? {
        ImageProcessor.toFastPathJPEG(image)
    }

    func base64JPEG(_ image: CGImage) -> String? {
        ImageProcessor.toBase64JPEG(image)
    }
}

final class ScreenAnalyzerPresetMetadataCache {
    private var souls: [String: String?] = [:]

    func soul(for character: String, loader: (String) -> (voice: String, size: String?, soul: String?)) -> String? {
        if let cached = souls[character] {
            return cached
        }
        let soul = loader(character).soul
        souls[character] = soul
        return soul
    }

    func invalidate(character: String? = nil) {
        if let character {
            souls.removeValue(forKey: character)
        } else {
            souls.removeAll()
        }
    }
}

final class ScreenAnalyzerCommandContextCache {
    struct Entry {
        let screenBasisID: String
        let screenshotBase64: String
        let activeDisplayID: String
        let targetDisplayID: String
        let capturedAt: Date
        let source: String
    }

    private let maxAge: TimeInterval
    private var entry: Entry?

    init(maxAge: TimeInterval) {
        self.maxAge = maxAge
    }

    func store(
        screenBasisID: String,
        screenshotBase64: String,
        activeDisplayID: String,
        targetDisplayID: String,
        capturedAt: Date,
        source: String
    ) {
        entry = Entry(
            screenBasisID: screenBasisID,
            screenshotBase64: screenshotBase64,
            activeDisplayID: activeDisplayID,
            targetDisplayID: targetDisplayID,
            capturedAt: capturedAt,
            source: source
        )
    }

    func latest(baseContext: NavigatorContextPayload, now: Date = Date(), includeScreenshot: Bool) -> NavigatorContextPayload {
        guard let entry else {
            return baseContext
        }

        let ageMs = max(0, Int(now.timeIntervalSince(entry.capturedAt) * 1000))
        guard Double(ageMs) <= maxAge * 1000 else {
            return baseContext
        }

        return baseContext.withScreenBasis(
            screenBasisID: entry.screenBasisID,
            activeDisplayID: entry.activeDisplayID,
            targetDisplayID: entry.targetDisplayID,
            screenshotAgeMs: ageMs,
            screenshotSource: entry.source,
            screenshotCached: true,
            screenshot: includeScreenshot ? entry.screenshotBase64 : nil
        )
    }
}
