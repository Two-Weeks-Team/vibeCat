import AppKit
import Foundation

struct CatMotionState {
    var position: CGPoint
    var targetPosition: CGPoint
    var homePosition: CGPoint
    var screenBounds: CGRect
    var facingLeft = false
    var manualTarget: CGPoint?
    var manualTargetExpiry: Date?
}

struct CatMotionStepResult {
    let position: CGPoint
    let screenBounds: CGRect
    let facingLeft: Bool
    let positionChanged: Bool
    let screenBoundsChanged: Bool
}

enum CatMotionMath {
    static let catOffsetX: CGFloat = 150
    static let catOffsetY: CGFloat = 30
    static let margin: CGFloat = 60

    static func initialState(mouseGlobal: CGPoint, screenFrames: [CGRect]) -> CatMotionState {
        let screenBounds = screenBoundsForPointer(mouseGlobal, screenFrames: screenFrames)
        let homePosition = CGPoint(x: screenBounds.width - 120, y: screenBounds.height - 120)
        return CatMotionState(
            position: homePosition,
            targetPosition: homePosition,
            homePosition: homePosition,
            screenBounds: screenBounds
        )
    }

    static func step(
        state: inout CatMotionState,
        mouseGlobal: CGPoint,
        screenFrames: [CGRect],
        now: Date,
        followFactor: CGFloat
    ) -> CatMotionStepResult {
        let previousBounds = state.screenBounds
        let nextBounds = screenBoundsForPointer(mouseGlobal, screenFrames: screenFrames)
        let boundsChanged = nextBounds != previousBounds
        if boundsChanged {
            state.screenBounds = nextBounds
            state.homePosition = CGPoint(x: nextBounds.width - 120, y: nextBounds.height - 120)
            state.position = clampToBounds(state.position, bounds: nextBounds)
            state.targetPosition = clampToBounds(state.position, bounds: nextBounds)
        }

        let mouseLocal = globalToLocal(mouseGlobal, bounds: state.screenBounds)
        let cursorTarget = clampToBounds(
            CGPoint(x: mouseLocal.x + catOffsetX, y: mouseLocal.y + catOffsetY),
            bounds: state.screenBounds
        )

        if let expiry = state.manualTargetExpiry,
           expiry > now,
           let manualTarget = state.manualTarget {
            state.targetPosition = clampToBounds(manualTarget, bounds: state.screenBounds)
        } else {
            state.manualTarget = nil
            state.manualTargetExpiry = nil
            state.targetPosition = cursorTarget
        }

        state.facingLeft = mouseLocal.x < state.position.x

        let previousPosition = state.position
        let dx = state.targetPosition.x - state.position.x
        let dy = state.targetPosition.y - state.position.y
        let dist = sqrt(dx * dx + dy * dy)
        if dist > 1.0 {
            if dist > 500 {
                state.position = state.targetPosition
            } else {
                state.position = CGPoint(
                    x: state.position.x + dx * followFactor,
                    y: state.position.y + dy * followFactor
                )
            }
        }

        return CatMotionStepResult(
            position: state.position,
            screenBounds: state.screenBounds,
            facingLeft: state.facingLeft,
            positionChanged: state.position != previousPosition,
            screenBoundsChanged: boundsChanged
        )
    }

    static func applyManualTarget(_ screenPoint: CGPoint, now: Date, state: inout CatMotionState) {
        let local = globalToLocal(screenPoint, bounds: state.screenBounds)
        state.manualTarget = clampToBounds(local, bounds: state.screenBounds)
        state.manualTargetExpiry = now.addingTimeInterval(5)
    }

    static func applyReturnHome(now: Date, state: inout CatMotionState) {
        state.manualTarget = state.homePosition
        state.manualTargetExpiry = now.addingTimeInterval(5)
    }

    static func clampToBounds(_ point: CGPoint, bounds: CGRect) -> CGPoint {
        CGPoint(
            x: max(margin, min(bounds.width - margin, point.x)),
            y: max(margin, min(bounds.height - margin, point.y))
        )
    }

    static func globalToLocal(_ point: CGPoint, bounds: CGRect) -> CGPoint {
        CGPoint(x: point.x - bounds.minX, y: point.y - bounds.minY)
    }

    static func screenBoundsForPointer(_ mouseGlobal: CGPoint, screenFrames: [CGRect]) -> CGRect {
        if let matched = screenFrames.first(where: { $0.contains(mouseGlobal) }) {
            return matched
        }
        if let first = screenFrames.first {
            return first
        }
        return CGRect(x: 0, y: 0, width: 1440, height: 900)
    }
}
