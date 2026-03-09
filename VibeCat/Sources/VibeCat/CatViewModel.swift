import AppKit
import VibeCatCore

@MainActor
final class CatViewModel {
    private(set) var position: CGPoint = .zero
    private(set) var facingLeft = false
    private var targetPosition: CGPoint = .zero
    private var homePosition: CGPoint = .zero
    private var moveTimer: Timer?
    private var returnHomeTimer: Timer?
    private var screenBounds: CGRect = .zero

    var onPositionUpdate: ((CGPoint) -> Void)?
    var onScreenFrameUpdate: ((CGRect) -> Void)?
    var activeScreenFrame: CGRect { screenBounds }

    init() {
        updateScreenBounds(for: NSEvent.mouseLocation)
        homePosition = CGPoint(x: screenBounds.maxX - 120, y: screenBounds.maxY - 120)
        position = homePosition
        targetPosition = homePosition
        startMoveLoop()
    }

    func pointToward(_ screenPoint: CGPoint) {
        let clamped = clampToBounds(globalToLocal(screenPoint))
        targetPosition = clamped
        returnHomeTimer?.invalidate()
        let timer = Timer(timeInterval: 5.0, repeats: false) { [weak self] _ in
            MainActor.assumeIsolated {
                self?.returnHome()
            }
        }
        RunLoop.main.add(timer, forMode: .common)
        returnHomeTimer = timer
    }

    func returnHome() {
        targetPosition = homePosition
    }

    private func startMoveLoop() {
        let timer = Timer(timeInterval: 1.0 / 60.0, repeats: true) { [weak self] _ in
            MainActor.assumeIsolated {
                self?.updateFromMouse()
                self?.updatePosition()
            }
        }
        RunLoop.main.add(timer, forMode: .common)
        moveTimer = timer
    }

    private let catOffsetX: CGFloat = 150
    private let catOffsetY: CGFloat = 30

    private func updateFromMouse() {
        let mouseGlobal = NSEvent.mouseLocation
        updateScreenBounds(for: mouseGlobal)
        let mouseLocal = globalToLocal(mouseGlobal)
        let catTarget = CGPoint(x: mouseLocal.x + catOffsetX, y: mouseLocal.y + catOffsetY)
        targetPosition = clampToBounds(catTarget)
        facingLeft = mouseLocal.x < position.x
    }

    private func updatePosition() {
        let dx = targetPosition.x - position.x
        let dy = targetPosition.y - position.y
        let dist = sqrt(dx * dx + dy * dy)
        guard dist > 1.0 else { return }

        if dist > 500 {
            position = targetPosition
            onPositionUpdate?(position)
            return
        }

        let factor = followFactor
        position = CGPoint(x: position.x + dx * factor, y: position.y + dy * factor)
        onPositionUpdate?(position)
    }

    private var followFactor: CGFloat {
        let stored = UserDefaults.standard.object(forKey: "vibecat.followSpeed") as? Double
        let value = stored ?? 0.08
        let clamped = max(0.01, min(1.0, value))
        return CGFloat(clamped)
    }

    private func clampToBounds(_ point: CGPoint) -> CGPoint {
        let margin: CGFloat = 60
        return CGPoint(
            x: max(margin, min(screenBounds.width - margin, point.x)),
            y: max(margin, min(screenBounds.height - margin, point.y))
        )
    }

    private func updateScreenBounds(for mouseGlobal: CGPoint) {
        let allScreens = NSScreen.screens
        let newBounds = allScreens.first(where: { NSMouseInRect(mouseGlobal, $0.frame, false) })?.frame
            ?? NSScreen.main?.frame
            ?? CGRect(x: 0, y: 0, width: 1440, height: 900)

        guard newBounds != screenBounds else { return }
        screenBounds = newBounds
        homePosition = CGPoint(x: screenBounds.width - 120, y: screenBounds.height - 120)

        if position == .zero {
            position = homePosition
            targetPosition = homePosition
        }

        position = clampToBounds(position)
        targetPosition = clampToBounds(position)
        onScreenFrameUpdate?(screenBounds)
        onPositionUpdate?(position)
    }

    private func globalToLocal(_ point: CGPoint) -> CGPoint {
        CGPoint(x: point.x - screenBounds.minX, y: point.y - screenBounds.minY)
    }
}
