import XCTest
@testable import VibeCatCore

final class VibeCatTests: XCTestCase {
    private let settingsKeys = [
        "vibecat.language",
        "vibecat.voice",
        "vibecat.character",
        "vibecat.chattiness",
        "vibecat.captureInterval",
        "vibecat.liveModel",
        "vibecat.musicEnabled",
        "vibecat.gatewayURL",
        "vibecat.searchEnabled",
        "vibecat.proactiveAudio"
    ]

    override func setUp() {
        super.setUp()
        clearSettings()
    }

    override func tearDown() {
        clearSettings()
        super.tearDown()
    }

    func testAppSettingsDefaults() {
        let settings = AppSettings.shared

        XCTAssertEqual(settings.language, "ko")
        XCTAssertEqual(settings.voice, "Zephyr")
        XCTAssertEqual(settings.character, "cat")
        XCTAssertEqual(settings.chattiness, "normal")
        XCTAssertEqual(settings.captureInterval, 1.0)
        XCTAssertEqual(settings.liveModel, "gemini-2.5-flash-native-audio-latest")
        XCTAssertFalse(settings.musicEnabled)
        XCTAssertEqual(settings.gatewayURL, "wss://realtime-gateway-163070481841.asia-northeast3.run.app")
        XCTAssertTrue(settings.searchEnabled)
        XCTAssertTrue(settings.proactiveAudio)
    }

    func testAppSettingsPersistedValuesAreReturned() {
        let settings = AppSettings.shared

        settings.language = "en"
        settings.voice = "Kore"
        settings.character = "jinwoo"
        settings.chattiness = "chatty"
        settings.captureInterval = 10.0
        settings.liveModel = "gemini-2.5-flash-preview-native-audio-dialog"
        settings.musicEnabled = true
        settings.gatewayURL = "wss://example.test/ws/live"
        settings.searchEnabled = true
        settings.proactiveAudio = true

        XCTAssertEqual(settings.language, "en")
        XCTAssertEqual(settings.voice, "Kore")
        XCTAssertEqual(settings.character, "jinwoo")
        XCTAssertEqual(settings.chattiness, "chatty")
        XCTAssertEqual(settings.captureInterval, 10.0)
        XCTAssertEqual(settings.liveModel, "gemini-2.5-flash-preview-native-audio-dialog")
        XCTAssertTrue(settings.musicEnabled)
        XCTAssertEqual(settings.gatewayURL, "wss://example.test/ws/live")
        XCTAssertTrue(settings.searchEnabled)
        XCTAssertTrue(settings.proactiveAudio)
    }

    func testCaptureIntervalFallsBackToDefaultWhenNonPositive() {
        let settings = AppSettings.shared
        settings.captureInterval = 0
        XCTAssertEqual(settings.captureInterval, 1.0)

        settings.captureInterval = -4
        XCTAssertEqual(settings.captureInterval, 1.0)
    }

    private func clearSettings() {
        let defaults = UserDefaults.standard
        for key in settingsKeys {
            defaults.removeObject(forKey: key)
        }
    }
}
