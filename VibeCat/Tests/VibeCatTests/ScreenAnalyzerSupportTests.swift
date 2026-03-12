import CoreGraphics
import XCTest
@testable import VibeCat
import VibeCatCore

final class ScreenAnalyzerSupportTests: XCTestCase {
    func testPresetMetadataCacheLoadsOncePerCharacter() {
        let cache = ScreenAnalyzerPresetMetadataCache()
        var loadCount = 0

        let first = cache.soul(for: "cat") { character in
            loadCount += 1
            return (voice: "A", size: nil, soul: "curious")
        }
        let second = cache.soul(for: "cat") { character in
            loadCount += 1
            return (voice: "B", size: nil, soul: "different")
        }

        XCTAssertEqual(first, "curious")
        XCTAssertEqual(second, "curious")
        XCTAssertEqual(loadCount, 1)
    }

    func testPresetMetadataCacheInvalidatesSingleCharacter() {
        let cache = ScreenAnalyzerPresetMetadataCache()
        var loadCount = 0

        _ = cache.soul(for: "cat") { _ in
            loadCount += 1
            return (voice: "A", size: nil, soul: "curious")
        }
        cache.invalidate(character: "cat")
        let refreshed = cache.soul(for: "cat") { _ in
            loadCount += 1
            return (voice: "A", size: nil, soul: "refreshed")
        }

        XCTAssertEqual(refreshed, "refreshed")
        XCTAssertEqual(loadCount, 2)
    }

    func testCommandContextCacheAppliesFreshScreenBasis() {
        let cache = ScreenAnalyzerCommandContextCache(maxAge: 20)
        cache.store(
            screenBasisID: "basis-1",
            screenshotBase64: "abc123",
            activeDisplayID: "display-a",
            targetDisplayID: "display-b",
            capturedAt: Date(),
            source: "display_context_cache"
        )

        let base = NavigatorContextPayload(
            appName: "Xcode",
            bundleId: "com.apple.dt.Xcode",
            frontmostBundleId: "com.apple.dt.Xcode",
            windowTitle: "Tests.swift",
            focusedRole: "AXTextField",
            focusedLabel: "Search",
            selectedText: "",
            axSnapshot: "ax",
            inputFieldHint: "hint",
            lastInputFieldDescriptor: "field",
            screenshot: "",
            focusStableMs: 100,
            captureConfidence: 0.9,
            visibleInputCandidateCount: 1,
            accessibilityPermission: "granted",
            accessibilityTrusted: true
        )

        let updated = cache.latest(baseContext: base, includeScreenshot: true)

        XCTAssertEqual(updated.screenBasisID, "basis-1")
        XCTAssertEqual(updated.activeDisplayID, "display-a")
        XCTAssertEqual(updated.targetDisplayID, "display-b")
        XCTAssertEqual(updated.screenshotSource, "display_context_cache")
        XCTAssertEqual(updated.screenshot, "abc123")
        XCTAssertTrue(updated.screenshotCached)
    }

    func testCommandContextCacheSkipsExpiredEntry() {
        let cache = ScreenAnalyzerCommandContextCache(maxAge: 20)
        let oldDate = Date(timeIntervalSinceNow: -25)
        cache.store(
            screenBasisID: "basis-old",
            screenshotBase64: "expired",
            activeDisplayID: "display-a",
            targetDisplayID: "display-a",
            capturedAt: oldDate,
            source: "expired_cache"
        )

        let base = NavigatorContextPayload(
            appName: "Terminal",
            bundleId: "com.apple.Terminal",
            frontmostBundleId: "com.apple.Terminal",
            windowTitle: "shell",
            focusedRole: "AXTextArea",
            focusedLabel: "",
            selectedText: "",
            axSnapshot: "ax",
            inputFieldHint: "",
            lastInputFieldDescriptor: "",
            screenshot: "original",
            focusStableMs: 50,
            captureConfidence: 0.7,
            visibleInputCandidateCount: 0,
            accessibilityPermission: "granted",
            accessibilityTrusted: true
        )

        let updated = cache.latest(baseContext: base, now: Date(), includeScreenshot: true)

        XCTAssertEqual(updated, base)
    }
}
