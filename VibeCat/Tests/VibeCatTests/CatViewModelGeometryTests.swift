import XCTest
@testable import VibeCat

final class CatViewModelGeometryTests: XCTestCase {
    func testCombinedScreenBoundsUsesUnionOfAllScreens() {
        let union = CatViewModel.combinedBounds([
            CGRect(x: 0, y: 0, width: 100, height: 100),
            CGRect(x: 100, y: 0, width: 200, height: 150),
            CGRect(x: -50, y: -25, width: 50, height: 75),
        ])
        XCTAssertEqual(union, CGRect(x: -50, y: -25, width: 350, height: 175))
    }
}
