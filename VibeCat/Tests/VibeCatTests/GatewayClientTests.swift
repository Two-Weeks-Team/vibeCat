import XCTest
@testable import VibeCat

private final class CountingWebSocketSessionProvider: GatewayWebSocketSessionProviding {
    private(set) var makeCount = 0
    let session = URLSession(configuration: .ephemeral)

    func makeSession() -> URLSession {
        makeCount += 1
        return session
    }
}

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

    func testMakeOrReuseWebSocketSessionReusesProviderResult() {
        let provider = CountingWebSocketSessionProvider()
        let client = GatewayClient(webSocketSessionProvider: provider)

        let first = client.makeOrReuseWebSocketSession()
        let second = client.makeOrReuseWebSocketSession()

        XCTAssertTrue(first === second)
        XCTAssertEqual(provider.makeCount, 1)
    }

    func testCloseConnectionKeepsReusableSessionForReconnect() {
        let provider = CountingWebSocketSessionProvider()
        let client = GatewayClient(webSocketSessionProvider: provider)

        let first = client.makeOrReuseWebSocketSession()
        client.closeConnection()
        let second = client.makeOrReuseWebSocketSession()

        XCTAssertTrue(first === second)
        XCTAssertEqual(provider.makeCount, 1)
    }
}
