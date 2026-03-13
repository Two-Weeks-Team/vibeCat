import XCTest
@testable import VibeCat

final class CatMotionMathTests: XCTestCase {
    func testStepMovesTowardMouseTarget() {
        var state = CatMotionMath.initialState(
            mouseGlobal: CGPoint(x: 100, y: 100),
            screenFrames: [CGRect(x: 0, y: 0, width: 800, height: 600)]
        )

        let result = CatMotionMath.step(
            state: &state,
            mouseGlobal: CGPoint(x: 200, y: 200),
            screenFrames: [CGRect(x: 0, y: 0, width: 800, height: 600)],
            now: Date(),
            followFactor: 0.5
        )

        XCTAssertTrue(result.positionChanged)
        XCTAssertEqual(result.facingLeft, true)
    }

    func testStepChangesBoundsWhenPointerMovesToAnotherScreen() {
        var state = CatMotionMath.initialState(
            mouseGlobal: CGPoint(x: 100, y: 100),
            screenFrames: [
                CGRect(x: 0, y: 0, width: 800, height: 600),
                CGRect(x: 800, y: 0, width: 800, height: 600),
            ]
        )

        let result = CatMotionMath.step(
            state: &state,
            mouseGlobal: CGPoint(x: 900, y: 120),
            screenFrames: [
                CGRect(x: 0, y: 0, width: 800, height: 600),
                CGRect(x: 800, y: 0, width: 800, height: 600),
            ],
            now: Date(),
            followFactor: 0.08
        )

        XCTAssertTrue(result.screenBoundsChanged)
        XCTAssertEqual(result.screenBounds.origin.x, 800)
    }

    func testManualTargetOverridesMouseUntilExpiry() {
        var state = CatMotionMath.initialState(
            mouseGlobal: CGPoint(x: 100, y: 100),
            screenFrames: [CGRect(x: 0, y: 0, width: 800, height: 600)]
        )
        let now = Date()
        CatMotionMath.applyManualTarget(CGPoint(x: 300, y: 320), now: now, state: &state)

        _ = CatMotionMath.step(
            state: &state,
            mouseGlobal: CGPoint(x: 10, y: 10),
            screenFrames: [CGRect(x: 0, y: 0, width: 800, height: 600)],
            now: now.addingTimeInterval(1),
            followFactor: 1.0
        )

        XCTAssertEqual(state.targetPosition.x, 300, accuracy: 0.01)
        XCTAssertEqual(state.targetPosition.y, 320, accuracy: 0.01)
    }
}
