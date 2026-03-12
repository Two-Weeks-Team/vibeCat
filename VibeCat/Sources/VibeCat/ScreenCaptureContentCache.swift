import Foundation

@MainActor
final class ScreenCaptureContentCache<Value> {
    private let ttl: TimeInterval
    private let now: () -> Date
    private var value: Value?
    private var cachedAt: Date = .distantPast

    init(ttl: TimeInterval, now: @escaping () -> Date = Date.init) {
        self.ttl = ttl
        self.now = now
    }

    func get(loader: () async throws -> Value) async throws -> (value: Value, cached: Bool) {
        let currentTime = now()
        if let value, currentTime.timeIntervalSince(cachedAt) <= ttl {
            return (value, true)
        }

        let loaded = try await loader()
        value = loaded
        cachedAt = currentTime
        return (loaded, false)
    }

    func invalidate() {
        value = nil
        cachedAt = .distantPast
    }
}
