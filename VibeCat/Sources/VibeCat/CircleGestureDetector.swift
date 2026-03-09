import AppKit
import Foundation

@MainActor
final class CircleGestureDetector {
    var onCircleGesture: (() -> Void)?

    private var globalMonitor: Any?
    private var localMonitor: Any?
    private var positions: [(point: CGPoint, time: Date)] = []
    private var cumulativeAngle: Double = 0
    private var circleCount = 0
    private var lastAngle: Double?

    private let requiredCircles = 3
    private let timeWindow: TimeInterval = 6.0
    private let minRadius: CGFloat = 60
    private let maxBufferSize = 300
    private var lastMouseHandleTime: Date = .distantPast
    private let mouseThrottleInterval: TimeInterval = 1.0 / 30.0

    func startMonitoring() {
        stopMonitoring()

        globalMonitor = NSEvent.addGlobalMonitorForEvents(matching: .mouseMoved) { [weak self] _ in
            Task { @MainActor [weak self] in
                self?.handleMouseMoved()
            }
        }

        localMonitor = NSEvent.addLocalMonitorForEvents(matching: .mouseMoved) { [weak self] event in
            Task { @MainActor [weak self] in
                self?.handleMouseMoved()
            }
            return event
        }
    }

    func stopMonitoring() {
        if let globalMonitor {
            NSEvent.removeMonitor(globalMonitor)
            self.globalMonitor = nil
        }
        if let localMonitor {
            NSEvent.removeMonitor(localMonitor)
            self.localMonitor = nil
        }
        reset()
    }

    private func handleMouseMoved() {
        let now = Date()
        guard now.timeIntervalSince(lastMouseHandleTime) >= mouseThrottleInterval else { return }
        lastMouseHandleTime = now
        let point = NSEvent.mouseLocation

        positions.append((point: point, time: now))
        pruneOldPositions(before: now.addingTimeInterval(-timeWindow))
        guard positions.count >= 10 else { return }

        let centroid = computeCentroid()
        let dx = Double(point.x - centroid.x)
        let dy = Double(point.y - centroid.y)
        let radius = sqrt(dx * dx + dy * dy)
        guard radius > Double(minRadius) else {
            lastAngle = nil
            return
        }

        let angle = atan2(dy, dx)
        guard let previous = lastAngle else {
            lastAngle = angle
            return
        }

        var delta = angle - previous
        if delta > .pi { delta -= 2 * .pi }
        if delta < -.pi { delta += 2 * .pi }

        cumulativeAngle += delta
        lastAngle = angle

        let completedCircles = Int(abs(cumulativeAngle) / (2 * .pi))
        if completedCircles > circleCount {
            circleCount = completedCircles
            if circleCount >= requiredCircles {
                onCircleGesture?()
                reset()
            }
        }
    }

    private func pruneOldPositions(before cutoff: Date) {
        let previousCount = positions.count
        positions.removeAll { $0.time < cutoff }
        if positions.count > maxBufferSize {
            positions.removeFirst(positions.count - maxBufferSize)
        }

        if previousCount > 5 && positions.count < 5 {
            reset()
        }
    }

    private func computeCentroid() -> CGPoint {
        let count = CGFloat(positions.count)
        let x = positions.reduce(CGFloat.zero) { partial, entry in
            partial + entry.point.x
        }
        let y = positions.reduce(CGFloat.zero) { partial, entry in
            partial + entry.point.y
        }
        return CGPoint(x: x / count, y: y / count)
    }

    private func reset() {
        cumulativeAngle = 0
        circleCount = 0
        lastAngle = nil
        positions.removeAll()
    }
}
