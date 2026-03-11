import XCTest
@testable import VibeCatCore

final class VibeCatTests: XCTestCase {
    private let settingsKeys = [
        "vibecat.language",
        "vibecat.voice",
        "vibecat.character",
        "vibecat.chattiness",
        "vibecat.captureInterval",
        "vibecat.captureTargetMode",
        "vibecat.liveModel",
        "vibecat.musicEnabled",
        "vibecat.gatewayURL",
        "vibecat.searchEnabled",
        "vibecat.proactiveAudio",
        "vibecat.manualAnalysisOnly"
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
        XCTAssertEqual(settings.captureTargetMode, .windowUnderCursor)
        XCTAssertEqual(settings.liveModel, GeminiModels.liveNativeAudio)
        XCTAssertFalse(settings.musicEnabled)
        XCTAssertEqual(settings.gatewayURL, "wss://realtime-gateway-163070481841.asia-northeast3.run.app")
        XCTAssertTrue(settings.searchEnabled)
        XCTAssertTrue(settings.proactiveAudio)
        XCTAssertFalse(settings.manualAnalysisOnly)
    }

    func testAppSettingsPersistedValuesAreReturned() {
        let settings = AppSettings.shared

        settings.language = "en"
        settings.voice = "Kore"
        settings.character = "jinwoo"
        settings.chattiness = "chatty"
        settings.captureInterval = 10.0
        settings.captureTargetMode = .display
        settings.liveModel = "gemini-live-2.5-flash-preview"
        settings.musicEnabled = true
        settings.gatewayURL = "wss://example.test/ws/live"
        settings.searchEnabled = true
        settings.proactiveAudio = true
        settings.manualAnalysisOnly = true

        XCTAssertEqual(settings.language, "en")
        XCTAssertEqual(settings.voice, "Kore")
        XCTAssertEqual(settings.character, "jinwoo")
        XCTAssertEqual(settings.chattiness, "chatty")
        XCTAssertEqual(settings.captureInterval, 10.0)
        XCTAssertEqual(settings.captureTargetMode, .display)
        XCTAssertEqual(settings.liveModel, "gemini-live-2.5-flash-preview")
        XCTAssertTrue(settings.musicEnabled)
        XCTAssertEqual(settings.gatewayURL, "wss://example.test/ws/live")
        XCTAssertTrue(settings.searchEnabled)
        XCTAssertTrue(settings.proactiveAudio)
        XCTAssertTrue(settings.manualAnalysisOnly)
    }

    func testCaptureIntervalFallsBackToDefaultWhenNonPositive() {
        let settings = AppSettings.shared
        settings.captureInterval = 0
        XCTAssertEqual(settings.captureInterval, 1.0)

        settings.captureInterval = -4
        XCTAssertEqual(settings.captureInterval, 1.0)

        settings.captureInterval = 0.5
        XCTAssertEqual(settings.captureInterval, 1.0)
    }

    func testLanguageSettingNormalizesSupportedCodes() {
        let settings = AppSettings.shared

        settings.language = "Japanese"
        XCTAssertEqual(settings.language, "ja")

        settings.language = "EN"
        XCTAssertEqual(settings.language, "en")
    }

    func testLocalizationUsesSelectedLanguage() {
        let settings = AppSettings.shared

        settings.language = "en"
        XCTAssertEqual(VibeCatL10n.screenReadingTitle(), "Reading screen...")
        XCTAssertEqual(VibeCatL10n.listeningTitle(), "Listening...")
        XCTAssertEqual(VibeCatL10n.captureTargetModeTitle(.frontmostWindow), "Frontmost Window")
        XCTAssertEqual(VibeCatL10n.processingStateLabel(stage: "searching"), "Searching...")
        XCTAssertEqual(VibeCatL10n.processingStateDetail(stage: "tool_running", tool: "maps"), "Checking Google Maps")
        XCTAssertEqual(VibeCatL10n.menuPrivacy(), "Privacy")
        XCTAssertEqual(VibeCatL10n.captureIndicatorLive(), "Screen Capture On")

        settings.language = "ja"
        XCTAssertEqual(VibeCatL10n.screenReadingTitle(), "画面を読み取り中...")
        XCTAssertEqual(VibeCatL10n.listeningDetail(), "話を聞いています")
        XCTAssertEqual(VibeCatL10n.characterName("cat"), "猫")
        XCTAssertEqual(VibeCatL10n.toolDisplayName("file_search"), "ファイル検索")
        XCTAssertEqual(VibeCatL10n.menuNoScreenshotsStored(), "スクリーンショットは保存しません")

        settings.language = "ko"
        XCTAssertEqual(VibeCatL10n.sourceCount(3), "근거 3개")
        XCTAssertEqual(VibeCatL10n.processingStateDetail(stage: "grounding", tool: "search", sourceCount: 2), "Google 검색 · 근거 2개 확인")
        XCTAssertEqual(VibeCatL10n.captureIndicatorManual(), "수동 분석 모드")
    }

    private func clearSettings() {
        let defaults = UserDefaults.standard
        for key in settingsKeys {
            defaults.removeObject(forKey: key)
        }
    }
}
