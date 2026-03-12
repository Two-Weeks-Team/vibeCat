import XCTest
@testable import VibeCat

final class TargetHighlightOverlayTests: XCTestCase {
    func testOverlayFrameExpandsTargetRectWithPadding() {
        let rect = CGRect(x: 100, y: 200, width: 300, height: 40)
        let overlay = TargetHighlightGeometry.overlayFrame(for: rect, padding: 6)

        XCTAssertEqual(overlay.origin.x, 94)
        XCTAssertEqual(overlay.origin.y, 194)
        XCTAssertEqual(overlay.size.width, 312)
        XCTAssertEqual(overlay.size.height, 52)
    }
}
