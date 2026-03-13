import AppKit
import CoreGraphics
import Foundation
import VibeCatCore

final class CatViewModel: @unchecked Sendable {
    private let motionQueue = DispatchQueue(label: "vibecat.cat.motion", qos: .userInteractive)
    private let mouseLocationProvider: @Sendable () -> CGPoint
    private let screenFramesProvider: @Sendable () -> [CGRect]
    private var motionTimer: DispatchSourceTimer?
    private var state: CatMotionState

    private let deliveryLock = NSLock()
    private var pendingDelivery: CatMotionStepResult?
    private var deliveryScheduled = false

    var onPositionUpdate: ((CGPoint) -> Void)?
    var onScreenFrameUpdate: ((CGRect) -> Void)?

    var position: CGPoint { motionQueue.sync { state.position } }
    var facingLeft: Bool { motionQueue.sync { state.facingLeft } }
    var activeScreenFrame: CGRect { motionQueue.sync { state.screenBounds } }

    nonisolated static func combinedScreenBounds(_ screens: [NSScreen]) -> CGRect {
        combinedBounds(screens.map(\.frame))
    }

    nonisolated static func combinedBounds(_ frames: [CGRect]) -> CGRect {
        guard let first = frames.first else {
            return CGRect(x: 0, y: 0, width: 1440, height: 900)
        }
        return frames.dropFirst().reduce(first) { partial, frame in
            partial.union(frame)
        }
    }

    /// Returns the current mouse location in AppKit screen coordinates (Y=0 at bottom).
    /// Uses thread-safe CG API internally and converts to AppKit coordinates.
    nonisolated static func defaultMouseLocation() -> CGPoint {
        guard let event = CGEvent(source: nil) else { return .zero }
        let cg = event.location
        let mainHeight = CGDisplayBounds(CGMainDisplayID()).size.height
        return CGPoint(x: cg.x, y: mainHeight - cg.y)
    }

    /// Returns active display frames in AppKit screen coordinates (Y=0 at bottom).
    /// Uses thread-safe CG API internally and converts to AppKit coordinates.
    nonisolated static func activeDisplayFrames() -> [CGRect] {
        var count: UInt32 = 0
        guard CGGetActiveDisplayList(0, nil, &count) == .success, count > 0 else {
            return []
        }
        var displays = [CGDirectDisplayID](repeating: 0, count: Int(count))
        guard CGGetActiveDisplayList(count, &displays, &count) == .success else {
            return []
        }
        let mainHeight = CGDisplayBounds(CGMainDisplayID()).size.height
        return displays.prefix(Int(count)).map { display in
            let cg = CGDisplayBounds(display)
            return CGRect(
                x: cg.origin.x,
                y: mainHeight - cg.origin.y - cg.size.height,
                width: cg.size.width,
                height: cg.size.height
            )
        }
    }

    init(
        mouseLocationProvider: @escaping @Sendable () -> CGPoint = CatViewModel.defaultMouseLocation,
        screenFramesProvider: @escaping @Sendable () -> [CGRect] = CatViewModel.activeDisplayFrames
    ) {
        self.mouseLocationProvider = mouseLocationProvider
        self.screenFramesProvider = screenFramesProvider
        self.state = CatMotionMath.initialState(mouseGlobal: mouseLocationProvider(), screenFrames: screenFramesProvider())
        startMoveLoop()
    }

    deinit {
        motionQueue.sync {
            motionTimer?.setEventHandler {}
            motionTimer?.cancel()
            motionTimer = nil
        }
    }

    func pointToward(_ screenPoint: CGPoint) {
        motionQueue.async { [weak self] in
            guard let self else { return }
            CatMotionMath.applyManualTarget(screenPoint, now: Date(), state: &self.state)
        }
    }

    func returnHome() {
        motionQueue.async { [weak self] in
            guard let self else { return }
            CatMotionMath.applyReturnHome(now: Date(), state: &self.state)
        }
    }

    private func startMoveLoop() {
        motionQueue.async { [weak self] in
            guard let self, self.motionTimer == nil else { return }
            let timer = DispatchSource.makeTimerSource(queue: self.motionQueue)
            timer.schedule(deadline: .now(), repeating: .milliseconds(16), leeway: .milliseconds(2))
            timer.setEventHandler { [weak self] in
                self?.tickMotion()
            }
            self.motionTimer = timer
            timer.resume()
        }
    }

    private func tickMotion() {
        let result = CatMotionMath.step(
            state: &state,
            mouseGlobal: mouseLocationProvider(),
            screenFrames: screenFramesProvider(),
            now: Date(),
            followFactor: followFactor
        )

        guard result.positionChanged || result.screenBoundsChanged else { return }
        enqueueDelivery(result)
    }

    private var followFactor: CGFloat {
        let stored = UserDefaults.standard.object(forKey: "vibecat.followSpeed") as? Double
        let value = stored ?? 0.08
        let clamped = max(0.01, min(1.0, value))
        return CGFloat(clamped)
    }

    private func enqueueDelivery(_ result: CatMotionStepResult) {
        deliveryLock.lock()
        // Merge flags: if an earlier result had screenBoundsChanged or positionChanged
        // but main thread hasn't flushed yet, preserve those flags so setFrame isn't lost.
        if let existing = pendingDelivery {
            pendingDelivery = CatMotionStepResult(
                position: result.position,
                screenBounds: result.screenBounds,
                facingLeft: result.facingLeft,
                positionChanged: result.positionChanged || existing.positionChanged,
                screenBoundsChanged: result.screenBoundsChanged || existing.screenBoundsChanged
            )
        } else {
            pendingDelivery = result
        }
        guard !deliveryScheduled else {
            deliveryLock.unlock()
            return
        }
        deliveryScheduled = true
        deliveryLock.unlock()

        DispatchQueue.main.async { [weak self] in
            self?.flushPendingDelivery()
        }
    }

    private func flushPendingDelivery() {
        deliveryLock.lock()
        let result = pendingDelivery
        pendingDelivery = nil
        deliveryScheduled = false
        deliveryLock.unlock()

        guard let result else { return }
        if result.screenBoundsChanged {
            onScreenFrameUpdate?(result.screenBounds)
        }
        if result.positionChanged || result.screenBoundsChanged {
            onPositionUpdate?(result.position)
        }
    }
}
