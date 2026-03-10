import AppKit
import XCTest
@testable import VibeCat

@MainActor
final class ChatBubbleViewTests: XCTestCase {
    func testSpeechLayoutVerticallyCentersSingleLineText() {
        let view = ChatBubbleView(frame: .zero)
        let size = view.preferredSize(primary: "짧은 문장", meta: nil, showsSpinner: false)
        let snapshot = view.debugLayoutSnapshot(primary: "짧은 문장", meta: nil, showsSpinner: false, size: size)

        let topGap = snapshot.body.maxY - snapshot.primary.maxY
        let bottomGap = snapshot.primary.minY - snapshot.body.minY
        XCTAssertLessThan(abs(topGap - bottomGap), 4.0)
    }

    func testSpeechLayoutKeepsMetaLabelBelowPrimaryText() {
        let view = ChatBubbleView(frame: .zero)
        let size = view.preferredSize(primary: "검색 결과입니다.", meta: "Google Search · 근거 3개", showsSpinner: false)
        let snapshot = view.debugLayoutSnapshot(primary: "검색 결과입니다.", meta: "Google Search · 근거 3개", showsSpinner: false, size: size)

        XCTAssertFalse(snapshot.meta.isEmpty)
        XCTAssertGreaterThan(snapshot.primary.minY, snapshot.meta.maxY)
    }

    func testStatusLayoutIncludesSpinnerFrame() {
        let view = ChatBubbleView(frame: .zero)
        let size = view.preferredSize(primary: "검색 중...", meta: "Google Search 확인 중", showsSpinner: true)
        let snapshot = view.debugLayoutSnapshot(primary: "검색 중...", meta: "Google Search 확인 중", showsSpinner: true, size: size)

        XCTAssertNotNil(snapshot.spinner)
        XCTAssertFalse(snapshot.meta.isEmpty)
    }
}
