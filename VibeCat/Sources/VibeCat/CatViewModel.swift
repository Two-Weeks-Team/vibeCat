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
    weak var panel: NSPanel?

    var onPositionUpdate: ((CGPoint) -> Void)?

    init() {
        updateScreenBounds(for: NSEvent.mouseLocation)
        homePosition = CGPoint(x: screenBounds.maxX - 120, y: screenBounds.maxY - 120)
        position = homePosition
        targetPosition = homePosition
        startMoveLoop()
    }

    func pointToward(_ screenPoint: CGPoint) {
        let clamped = clampToBounds(screenPoint)
        targetPosition = clamped
        returnHomeTimer?.invalidate()
        returnHomeTimer = Timer.scheduledTimer(withTimeInterval: 5.0, repeats: false) { [weak self] _ in
            Task { @MainActor [weak self] in
                self?.returnHome()
            }
        }
    }

    func returnHome() {
        targetPosition = homePosition
    }

    private func startMoveLoop() {
        moveTimer = Timer.scheduledTimer(withTimeInterval: 1.0 / 60.0, repeats: true) { [weak self] _ in
            Task { @MainActor [weak self] in
                self?.updateFromMouse()
                self?.updatePosition()
            }
        }
    }

    private func updateFromMouse() {
        let mouseGlobal = NSEvent.mouseLocation
        updateScreenBounds(for: mouseGlobal)
        targetPosition = clampToBounds(mouseGlobal)
        facingLeft = mouseGlobal.x < position.x
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
            x: max(margin, min(screenBounds.maxX - margin, point.x)),
            y: max(margin, min(screenBounds.maxY - margin, point.y))
        )
    }

    private func updateScreenBounds(for mouseGlobal: CGPoint) {
        let allScreens = NSScreen.screens
        let newBounds = allScreens.first(where: { NSMouseInRect(mouseGlobal, $0.frame, false) })?.frame
            ?? NSScreen.main?.frame
            ?? CGRect(x: 0, y: 0, width: 1440, height: 900)

        guard newBounds != screenBounds else { return }
        screenBounds = newBounds
        homePosition = CGPoint(x: screenBounds.maxX - 120, y: screenBounds.maxY - 120)

        guard let panel else { return }
        let panelFrame = panel.frame
        let clampedOrigin = NSPoint(
            x: max(screenBounds.minX, min(screenBounds.maxX - panelFrame.width, panelFrame.origin.x)),
            y: max(screenBounds.minY, min(screenBounds.maxY - panelFrame.height, panelFrame.origin.y))
        )
        panel.setFrameOrigin(clampedOrigin)
        position = CGPoint(x: clampedOrigin.x + panelFrame.width / 2, y: clampedOrigin.y + panelFrame.height / 2)
        targetPosition = clampToBounds(position)
        onPositionUpdate?(position)
    }
}
