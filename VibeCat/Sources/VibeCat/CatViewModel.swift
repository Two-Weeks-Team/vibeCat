import AppKit
import VibeCatCore

@MainActor
final class CatViewModel {
    private(set) var position: CGPoint = .zero
    private var targetPosition: CGPoint = .zero
    private var homePosition: CGPoint = .zero
    private var moveTimer: Timer?
    private var screenBounds: CGRect = .zero

    var onPositionUpdate: ((CGPoint) -> Void)?

    init() {
        updateScreenBounds()
        homePosition = CGPoint(x: screenBounds.maxX - 120, y: screenBounds.maxY - 120)
        position = homePosition
        targetPosition = homePosition
        startMoveLoop()
    }

    func pointToward(_ screenPoint: CGPoint) {
        let clamped = clampToBounds(screenPoint)
        targetPosition = clamped
        Timer.scheduledTimer(withTimeInterval: 5.0, repeats: false) { [weak self] _ in
            Task { @MainActor [weak self] in
                self?.returnHome()
            }
        }
    }

    func returnHome() {
        targetPosition = homePosition
    }

    private func startMoveLoop() {
        moveTimer = Timer.scheduledTimer(withTimeInterval: 0.016, repeats: true) { [weak self] _ in
            Task { @MainActor [weak self] in
                self?.lerp()
            }
        }
    }

    private func lerp() {
        let dx = targetPosition.x - position.x
        let dy = targetPosition.y - position.y
        let dist = sqrt(dx*dx + dy*dy)
        guard dist > 1.0 else { return }
        let factor: CGFloat = 0.08
        position = CGPoint(x: position.x + dx * factor, y: position.y + dy * factor)
        onPositionUpdate?(position)
    }

    private func clampToBounds(_ point: CGPoint) -> CGPoint {
        let margin: CGFloat = 60
        return CGPoint(
            x: max(margin, min(screenBounds.maxX - margin, point.x)),
            y: max(margin, min(screenBounds.maxY - margin, point.y))
        )
    }

    private func updateScreenBounds() {
        screenBounds = NSScreen.main?.frame ?? CGRect(x: 0, y: 0, width: 1440, height: 900)
    }
}
