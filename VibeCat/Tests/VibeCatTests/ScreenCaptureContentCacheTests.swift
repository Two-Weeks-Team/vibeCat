import XCTest
@testable import VibeCat

@MainActor
final class ScreenCaptureContentCacheTests: XCTestCase {
    func testCacheReturnsCachedValueWithinTTL() async throws {
        let now = Date()
        let cache = ScreenCaptureContentCache<String>(ttl: 5, now: { now })
        var loadCount = 0

        let first = try await cache.get {
            loadCount += 1
            return "first"
        }
        let second = try await cache.get {
            loadCount += 1
            return "second"
        }

        XCTAssertEqual(first.value, "first")
        XCTAssertFalse(first.cached)
        XCTAssertEqual(second.value, "first")
        XCTAssertTrue(second.cached)
        XCTAssertEqual(loadCount, 1)
    }

    func testCacheReloadsAfterTTLExpires() async throws {
        var now = Date()
        let cache = ScreenCaptureContentCache<String>(ttl: 1, now: { now })
        var loadCount = 0

        _ = try await cache.get {
            loadCount += 1
            return "first"
        }
        now = now.addingTimeInterval(2)
        let refreshed = try await cache.get {
            loadCount += 1
            return "second"
        }

        XCTAssertEqual(refreshed.value, "second")
        XCTAssertFalse(refreshed.cached)
        XCTAssertEqual(loadCount, 2)
    }

    func testCacheReloadsAfterExplicitInvalidation() async throws {
        let cache = ScreenCaptureContentCache<String>(ttl: 60)
        var loadCount = 0

        _ = try await cache.get {
            loadCount += 1
            return "first"
        }
        cache.invalidate()
        let refreshed = try await cache.get {
            loadCount += 1
            return "second"
        }

        XCTAssertEqual(refreshed.value, "second")
        XCTAssertFalse(refreshed.cached)
        XCTAssertEqual(loadCount, 2)
    }
}
