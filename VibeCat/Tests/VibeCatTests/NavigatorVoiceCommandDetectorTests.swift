import XCTest
@testable import VibeCatCore

final class NavigatorVoiceCommandDetectorTests: XCTestCase {
    func testRoutesTextEntryCommandWhenVisibleInputExists() {
        let context = NavigatorContextPayload(
            appName: "Google Chrome",
            bundleId: "com.google.Chrome",
            frontmostBundleId: "com.google.Chrome",
            windowTitle: "Gemini Live Agent Challenge",
            focusedRole: "AXTextField",
            focusedLabel: "Comment",
            selectedText: "",
            axSnapshot: "focused_input:AXTextField:Comment",
            inputFieldHint: "Comment",
            lastInputFieldDescriptor: "label=Comment",
            screenshot: "",
            focusStableMs: 520,
            captureConfidence: 0.88,
            visibleInputCandidateCount: 2,
            accessibilityPermission: "trusted",
            accessibilityTrusted: true
        )

        XCTAssertTrue(
            NavigatorVoiceCommandDetector.shouldRoute(
                "여기에 \"LGTM, shipping this.\" 입력해줘",
                context: context,
                hasPendingNavigatorPrompt: false
            )
        )
    }

    func testRoutesKoreanTextEntryVariantWithTypoWhenVisibleInputExists() {
        let context = NavigatorContextPayload(
            appName: "Terminal",
            bundleId: "com.apple.Terminal",
            frontmostBundleId: "com.apple.Terminal",
            windowTitle: "Terminal",
            focusedRole: "AXTextArea",
            focusedLabel: "Shell",
            selectedText: "",
            axSnapshot: "focused_input:AXTextArea:Shell",
            inputFieldHint: "Shell",
            lastInputFieldDescriptor: "label=Shell",
            screenshot: "",
            focusStableMs: 430,
            captureConfidence: 0.91,
            visibleInputCandidateCount: 1,
            accessibilityPermission: "trusted",
            accessibilityTrusted: true
        )

        XCTAssertTrue(
            NavigatorVoiceCommandDetector.shouldRoute(
                "이 터미널에 A부터 Z까지 입력해죠.",
                context: context,
                hasPendingNavigatorPrompt: false
            )
        )
    }

    func testRoutesCommitAndPushInputPhraseWhenVisibleInputExists() {
        let context = NavigatorContextPayload(
            appName: "Terminal",
            bundleId: "com.apple.Terminal",
            frontmostBundleId: "com.apple.Terminal",
            windowTitle: "Terminal",
            focusedRole: "AXTextArea",
            focusedLabel: "Shell",
            selectedText: "",
            axSnapshot: "focused_input:AXTextArea:Shell",
            inputFieldHint: "Shell",
            lastInputFieldDescriptor: "label=Shell",
            screenshot: "",
            focusStableMs: 430,
            captureConfidence: 0.91,
            visibleInputCandidateCount: 1,
            accessibilityPermission: "trusted",
            accessibilityTrusted: true
        )

        XCTAssertTrue(
            NavigatorVoiceCommandDetector.shouldRoute(
                "커밋하고 푸시하라고 입력해줘",
                context: context,
                hasPendingNavigatorPrompt: false
            )
        )
    }

    func testRoutesPutThisHereStyleWhenVisibleInputExists() {
        let context = NavigatorContextPayload(
            appName: "Codex",
            bundleId: "com.openai.codex",
            frontmostBundleId: "com.openai.codex",
            windowTitle: "Codex",
            focusedRole: "AXTextArea",
            focusedLabel: "후속 변경 사항을 부탁하세요",
            selectedText: "",
            axSnapshot: "focused_input:AXTextArea:후속 변경 사항을 부탁하세요",
            inputFieldHint: "후속 변경 사항을 부탁하세요",
            lastInputFieldDescriptor: "label=후속 변경 사항을 부탁하세요",
            screenshot: "",
            focusStableMs: 500,
            captureConfidence: 0.88,
            visibleInputCandidateCount: 1,
            accessibilityPermission: "trusted",
            accessibilityTrusted: true
        )

        XCTAssertTrue(
            NavigatorVoiceCommandDetector.shouldRoute(
                "여기에 이 문장 넣어줘",
                context: context,
                hasPendingNavigatorPrompt: false
            )
        )
    }

    func testRoutesPendingNavigatorPromptResponse() {
        let context = NavigatorContextPayload(
            appName: "",
            bundleId: "",
            frontmostBundleId: "",
            windowTitle: "",
            focusedRole: "",
            focusedLabel: "",
            selectedText: "",
            axSnapshot: "",
            inputFieldHint: "",
            lastInputFieldDescriptor: "",
            screenshot: "",
            focusStableMs: 0,
            captureConfidence: 0,
            visibleInputCandidateCount: 0,
            accessibilityPermission: "unknown",
            accessibilityTrusted: false
        )

        XCTAssertTrue(
            NavigatorVoiceCommandDetector.shouldRoute(
                "설명만 해줘",
                context: context,
                hasPendingNavigatorPrompt: true
            )
        )
    }

    func testDoesNotRoutePlainConversationalQuestion() {
        let context = NavigatorContextPayload(
            appName: "Google Chrome",
            bundleId: "com.google.Chrome",
            frontmostBundleId: "com.google.Chrome",
            windowTitle: "Docs",
            focusedRole: "",
            focusedLabel: "",
            selectedText: "",
            axSnapshot: "",
            inputFieldHint: "",
            lastInputFieldDescriptor: "",
            screenshot: "",
            focusStableMs: 0,
            captureConfidence: 0.91,
            visibleInputCandidateCount: 0,
            accessibilityPermission: "trusted",
            accessibilityTrusted: true
        )

        XCTAssertFalse(
            NavigatorVoiceCommandDetector.shouldRoute(
                "이 에러가 왜 나는지 설명해줘",
                context: context,
                hasPendingNavigatorPrompt: false
            )
        )
    }

    func testRoutesDocsLookupRequest() {
        let context = NavigatorContextPayload(
            appName: "Google Chrome",
            bundleId: "com.google.Chrome",
            frontmostBundleId: "com.google.Chrome",
            windowTitle: "Antigravity",
            focusedRole: "",
            focusedLabel: "",
            selectedText: "",
            axSnapshot: "",
            inputFieldHint: "",
            lastInputFieldDescriptor: "",
            screenshot: "",
            focusStableMs: 0,
            captureConfidence: 0.81,
            visibleInputCandidateCount: 0,
            accessibilityPermission: "trusted",
            accessibilityTrusted: true
        )

        XCTAssertTrue(
            NavigatorVoiceCommandDetector.shouldRoute(
                "Gemini Live API 공식 문서 찾아줘",
                context: context,
                hasPendingNavigatorPrompt: false
            )
        )
    }

    func testRoutesSystemVolumeRequest() {
        let context = NavigatorContextPayload(
            appName: "Codex",
            bundleId: "com.openai.codex",
            frontmostBundleId: "com.openai.codex",
            windowTitle: "Codex",
            focusedRole: "AXTextArea",
            focusedLabel: "후속 변경 사항을 부탁하세요",
            selectedText: "",
            axSnapshot: "",
            inputFieldHint: "후속 변경 사항을 부탁하세요",
            lastInputFieldDescriptor: "label=후속 변경 사항을 부탁하세요",
            screenshot: "",
            focusStableMs: 640,
            captureConfidence: 0.9,
            visibleInputCandidateCount: 1,
            accessibilityPermission: "trusted",
            accessibilityTrusted: true
        )

        XCTAssertTrue(
            NavigatorVoiceCommandDetector.shouldRoute(
                "지금 볼륨 조금만 줄여 줄래?",
                context: context,
                hasPendingNavigatorPrompt: false
            )
        )
    }

    func testRoutesKoreanHonorificVariant() {
        let context = NavigatorContextPayload(
            appName: "Google Chrome",
            bundleId: "com.google.Chrome",
            frontmostBundleId: "com.google.Chrome",
            windowTitle: "Gemini Live Agent Challenge",
            focusedRole: "AXTextField",
            focusedLabel: "Comment",
            selectedText: "",
            axSnapshot: "focused_input:AXTextField:Comment",
            inputFieldHint: "Comment",
            lastInputFieldDescriptor: "label=Comment",
            screenshot: "",
            focusStableMs: 520,
            captureConfidence: 0.88,
            visibleInputCandidateCount: 1,
            accessibilityPermission: "trusted",
            accessibilityTrusted: true
        )

        XCTAssertTrue(
            NavigatorVoiceCommandDetector.shouldRoute(
                "여기에 입력해 주세요: hello",
                context: context,
                hasPendingNavigatorPrompt: false
            )
        )
    }

    func testRoutesKoreanCasualVariant() {
        let context = NavigatorContextPayload(
            appName: "Terminal",
            bundleId: "com.apple.Terminal",
            frontmostBundleId: "com.apple.Terminal",
            windowTitle: "Terminal",
            focusedRole: "AXTextArea",
            focusedLabel: "Shell",
            selectedText: "",
            axSnapshot: "focused_input:AXTextArea:Shell",
            inputFieldHint: "Shell",
            lastInputFieldDescriptor: "label=Shell",
            screenshot: "",
            focusStableMs: 430,
            captureConfidence: 0.91,
            visibleInputCandidateCount: 1,
            accessibilityPermission: "trusted",
            accessibilityTrusted: true
        )

        XCTAssertTrue(
            NavigatorVoiceCommandDetector.shouldRoute(
                "hello world 쳐줘",
                context: context,
                hasPendingNavigatorPrompt: false
            )
        )
    }

    func testRoutesWithoutLocationReference() {
        let context = NavigatorContextPayload(
            appName: "Terminal",
            bundleId: "com.apple.Terminal",
            frontmostBundleId: "com.apple.Terminal",
            windowTitle: "Terminal",
            focusedRole: "AXTextArea",
            focusedLabel: "Shell",
            selectedText: "",
            axSnapshot: "focused_input:AXTextArea:Shell",
            inputFieldHint: "Shell",
            lastInputFieldDescriptor: "label=Shell",
            screenshot: "",
            focusStableMs: 430,
            captureConfidence: 0.91,
            visibleInputCandidateCount: 1,
            accessibilityPermission: "trusted",
            accessibilityTrusted: true
        )

        XCTAssertTrue(
            NavigatorVoiceCommandDetector.shouldRoute(
                "A부터 Z까지 입력해줘",
                context: context,
                hasPendingNavigatorPrompt: false
            )
        )
    }

    func testRoutesPostfixMarker() {
        let context = NavigatorContextPayload(
            appName: "Google Chrome",
            bundleId: "com.google.Chrome",
            frontmostBundleId: "com.google.Chrome",
            windowTitle: "Gemini Live Agent Challenge",
            focusedRole: "AXTextField",
            focusedLabel: "Comment",
            selectedText: "",
            axSnapshot: "focused_input:AXTextField:Comment",
            inputFieldHint: "Comment",
            lastInputFieldDescriptor: "label=Comment",
            screenshot: "",
            focusStableMs: 520,
            captureConfidence: 0.88,
            visibleInputCandidateCount: 1,
            accessibilityPermission: "trusted",
            accessibilityTrusted: true
        )

        XCTAssertTrue(
            NavigatorVoiceCommandDetector.shouldRoute(
                "LGTM이라고 입력해줘",
                context: context,
                hasPendingNavigatorPrompt: false
            )
        )
    }
}
