import AppKit
import XCTest
@testable import VibeCat

final class CatBubbleLayoutTests: XCTestCase {
    func testStatusPlacementGoesAboveWhenNoSpeech() {
        let catFrame = NSRect(x: 120, y: 220, width: 100, height: 100)
        let bubbleSize = NSSize(width: 180, height: 64)
        let screenFrame = NSRect(x: 0, y: 0, width: 600, height: 500)

        let placement = CatBubbleLayout.placement(
            catFrame: catFrame,
            bubbleSize: bubbleSize,
            screenFrame: screenFrame,
            mode: .status,
            reservedBottomMinY: nil,
            isSpeechVisible: false
        )

        XCTAssertEqual(placement.tailDirection, .bottom)
        XCTAssertEqual(placement.frame.minY, catFrame.maxY + 8, accuracy: 0.01)
    }

    func testStatusPlacementPinsBelowCatWhenSpeechVisible() {
        let catFrame = NSRect(x: 120, y: 220, width: 100, height: 100)
        let bubbleSize = NSSize(width: 180, height: 64)
        let screenFrame = NSRect(x: 0, y: 0, width: 600, height: 500)

        let placement = CatBubbleLayout.placement(
            catFrame: catFrame,
            bubbleSize: bubbleSize,
            screenFrame: screenFrame,
            mode: .status,
            reservedBottomMinY: nil,
            isSpeechVisible: true
        )

        XCTAssertEqual(placement.tailDirection, .top)
        XCTAssertEqual(placement.frame.maxY, catFrame.minY - 8, accuracy: 0.01)
    }

    func testStatusPlacementStacksBelowWindowBadgeWhenSpeechVisible() {
        let catFrame = NSRect(x: 120, y: 260, width: 100, height: 100)
        let bubbleSize = NSSize(width: 180, height: 64)
        let screenFrame = NSRect(x: 0, y: 0, width: 600, height: 500)

        let placement = CatBubbleLayout.placement(
            catFrame: catFrame,
            bubbleSize: bubbleSize,
            screenFrame: screenFrame,
            mode: .status,
            reservedBottomMinY: 210,
            isSpeechVisible: true
        )

        XCTAssertEqual(placement.tailDirection, .top)
        XCTAssertEqual(placement.frame.maxY, 202, accuracy: 0.01)
    }

    func testSpeechPlacementPrefersAboveCatWhenRoomExists() {
        let catFrame = NSRect(x: 120, y: 120, width: 100, height: 100)
        let bubbleSize = NSSize(width: 180, height: 64)
        let screenFrame = NSRect(x: 0, y: 0, width: 600, height: 500)

        let placement = CatBubbleLayout.placement(
            catFrame: catFrame,
            bubbleSize: bubbleSize,
            screenFrame: screenFrame,
            mode: .speech,
            reservedBottomMinY: nil
        )

        XCTAssertEqual(placement.tailDirection, .bottom)
        XCTAssertEqual(placement.frame.minY, catFrame.maxY + 8, accuracy: 0.01)
    }

    func testSpeechPlacementFallsBelowWhenTopWouldOverflow() {
        let catFrame = NSRect(x: 120, y: 420, width: 100, height: 100)
        let bubbleSize = NSSize(width: 180, height: 90)
        let screenFrame = NSRect(x: 0, y: 0, width: 600, height: 500)

        let placement = CatBubbleLayout.placement(
            catFrame: catFrame,
            bubbleSize: bubbleSize,
            screenFrame: screenFrame,
            mode: .speech,
            reservedBottomMinY: nil
        )

        XCTAssertEqual(placement.tailDirection, .top)
        XCTAssertEqual(placement.frame.maxY, catFrame.minY - 8, accuracy: 0.01)
    }

    func testSpeechPlacementStaysAboveWhenScreenHasNonZeroMinY() {
        let catFrame = NSRect(x: 120, y: 300, width: 100, height: 100)
        let bubbleSize = NSSize(width: 180, height: 64)
        let screenFrame = NSRect(x: 0, y: 25, width: 600, height: 475)

        let placement = CatBubbleLayout.placement(
            catFrame: catFrame,
            bubbleSize: bubbleSize,
            screenFrame: screenFrame,
            mode: .speech,
            reservedBottomMinY: nil
        )

        XCTAssertEqual(placement.tailDirection, .bottom,
                       "Bubble should stay above cat when there's room (non-zero screenFrame.minY)")
        XCTAssertEqual(placement.frame.minY, catFrame.maxY + 8, accuracy: 0.01)
    }

    func testPlacementRespectsNonZeroScreenOriginWhenUsingLocalCatFrame() {
        let catFrame = NSRect(x: 520, y: 220, width: 100, height: 100)
        let bubbleSize = NSSize(width: 180, height: 64)
        let globalVisibleFrame = NSRect(x: 1440, y: 38, width: 1280, height: 820)
        let panelFrame = NSRect(x: 1440, y: 0, width: 1280, height: 900)
        let localVisibleFrame = globalVisibleFrame.offsetBy(dx: -panelFrame.minX, dy: -panelFrame.minY)

        let placement = CatBubbleLayout.placement(
            catFrame: catFrame,
            bubbleSize: bubbleSize,
            screenFrame: localVisibleFrame,
            mode: .status,
            reservedBottomMinY: nil,
            isSpeechVisible: true
        )

        XCTAssertGreaterThanOrEqual(placement.frame.minX, localVisibleFrame.minX + 8)
        XCTAssertLessThanOrEqual(placement.frame.maxX, localVisibleFrame.maxX - 8)
    }
}
