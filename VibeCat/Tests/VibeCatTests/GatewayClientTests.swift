import XCTest
@testable import VibeCat

@MainActor
final class GatewayClientTests: XCTestCase {
    func testEmptySessionHandleDoesNotOverwriteValidHandle() {
        let client = GatewayClient()

        client.applySessionHandleUpdate("valid-handle")
        XCTAssertEqual(client.sessionHandle, "valid-handle")

        client.applySessionHandleUpdate("")
        XCTAssertEqual(client.sessionHandle, "valid-handle")

        client.applySessionHandleUpdate("   ")
        XCTAssertEqual(client.sessionHandle, "valid-handle")
    }
}
