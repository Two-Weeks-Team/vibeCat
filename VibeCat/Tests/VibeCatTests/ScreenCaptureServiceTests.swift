import XCTest
@testable import VibeCat

final class ScreenCaptureServiceTests: XCTestCase {
    func testWindowBoundsContainMouseUsesDirectScreenSpaceCoordinates() {
        let primaryHeight = NSScreen.screens.first?.frame.height ?? 900
        let bounds = CGRect(x: 100, y: 200, width: 300, height: 400)
        XCTAssertTrue(ScreenCaptureService.MouseWindowTargetingGeometry.windowBoundsContainMouse(bounds, mouseLocation: CGPoint(x: 250, y: primaryHeight - 450)))
        XCTAssertFalse(ScreenCaptureService.MouseWindowTargetingGeometry.windowBoundsContainMouse(bounds, mouseLocation: CGPoint(x: 250, y: primaryHeight - 650)))
    }

    func testAppKitToCGPointConvertsYAxis() {
        let primaryHeight = NSScreen.screens.first?.frame.height ?? 900
        let appKitPoint = CGPoint(x: 100, y: 200)
        let cgPoint = ScreenCaptureService.MouseWindowTargetingGeometry.appKitToCGPoint(appKitPoint)
        XCTAssertEqual(cgPoint.x, 100)
        XCTAssertEqual(cgPoint.y, primaryHeight - 200, accuracy: 0.01)
    }

    func testWindowBoundsContainMouseWithCGConversion() {
        let primaryHeight = NSScreen.screens.first?.frame.height ?? 900
        let bounds = CGRect(x: 100, y: 50, width: 300, height: 400)
        let insideAppKit = CGPoint(x: 250, y: primaryHeight - 200)
        XCTAssertTrue(ScreenCaptureService.MouseWindowTargetingGeometry.windowBoundsContainMouse(bounds, mouseLocation: insideAppKit))
        let outsideAppKit = CGPoint(x: 250, y: primaryHeight - 500)
        XCTAssertFalse(ScreenCaptureService.MouseWindowTargetingGeometry.windowBoundsContainMouse(bounds, mouseLocation: outsideAppKit))
    }
}
