import XCTest
@testable import VibeCatCore

final class NavigatorSurfaceProfileTests: XCTestCase {
    func testDetectChromeFromTargetApp() {
        let profile = NavigatorSurfaceProfile.detect(targetApp: "Chrome")
        XCTAssertEqual(profile.kind, .chrome)
        XCTAssertEqual(profile.primaryBundleID, "com.google.Chrome")
    }

    func testDetectTerminalFromBundleID() {
        let profile = NavigatorSurfaceProfile.detect(targetApp: "", bundleID: "com.apple.Terminal")
        XCTAssertEqual(profile.kind, .terminal)
        XCTAssertTrue(profile.matches(bundleID: "com.apple.Terminal"))
    }

    func testDetectAntigravityFromCodexAlias() {
        let profile = NavigatorSurfaceProfile.detect(targetApp: "Codex")
        XCTAssertEqual(profile.kind, .antigravity)
        XCTAssertEqual(profile.primaryBundleID, "com.openai.codex")
        XCTAssertTrue(profile.matches(appName: "Antigravity"))
        XCTAssertTrue(profile.matches(bundleID: "com.openai.codex"))
    }

    func testDetectAntigravityFromDescriptorAppName() {
        let profile = NavigatorSurfaceProfile.detect(
            targetApp: "",
            descriptor: NavigatorTargetDescriptor(appName: "Antigravity IDE")
        )
        XCTAssertEqual(profile.kind, .antigravity)
    }

    func testUnknownSurfaceFallsBackCleanly() {
        let profile = NavigatorSurfaceProfile.detect(targetApp: "Notes")
        XCTAssertEqual(profile.kind, .unknown)
        XCTAssertNil(profile.primaryBundleID)
        XCTAssertFalse(profile.matches(appName: "Chrome"))
    }

    func testChromeSearchFieldPrefersAddressBarPreparation() {
        let profile = NavigatorSurfaceProfile.detect(
            targetApp: "Chrome",
            descriptor: NavigatorTargetDescriptor(label: "Search", appName: "Chrome")
        )
        XCTAssertEqual(profile.preferredPreparationHotkey(for: .pasteText), ["command", "l"])
    }

    func testChromeAddressFieldPrefersAddressBarPreparation() {
        let profile = NavigatorSurfaceProfile.detect(
            targetApp: "Chrome",
            descriptor: NavigatorTargetDescriptor(label: "Address", appName: "Chrome")
        )
        XCTAssertEqual(profile.preferredPreparationHotkey(for: .pressAX), ["command", "l"])
    }

    func testAntigravityFollowUpComposerHasNoExtraPreparationWhenPlannerAlreadyOpenedPrompt() {
        let profile = NavigatorSurfaceProfile.detect(
            targetApp: "Antigravity",
            descriptor: NavigatorTargetDescriptor(label: "후속 변경 사항을 부탁하세요", appName: "Codex")
        )
        XCTAssertNil(profile.preferredPreparationHotkey(for: .pasteText))
    }

    func testTerminalHasNoPreparationHotkeyForPasteText() {
        let profile = NavigatorSurfaceProfile.detect(targetApp: "Terminal")
        XCTAssertNil(profile.preferredPreparationHotkey(for: .pasteText))
    }
}
