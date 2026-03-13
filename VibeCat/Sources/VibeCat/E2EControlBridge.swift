import AppKit
import Foundation
import Network
import VibeCatCore

// MARK: - E2EControlBridge
//
// Test-only HTTP control bridge for automated E2E testing.
//
// Only starts when the environment variable `VIBECAT_E2E_CONTROL` is set to `"1"`.
// Listens on localhost:9876 and exposes four endpoints:
//
//   POST /e2e/command    — submit a natural language navigator command
//   GET  /e2e/status     — current navigator state (idle/executing/completed/failed)
//   GET  /e2e/events     — SSE stream of navigator lifecycle events
//   POST /e2e/screenshot — capture screen and return as base64 JPEG
//
// MUST NOT be enabled in production.
// DO NOT place bridge code in Sources/Core/ — it must stay in Sources/VibeCat/.

@MainActor
final class E2EControlBridge {

    static let shared = E2EControlBridge()

    // MARK: - Navigator State

    enum NavigatorState: String, Encodable {
        case idle
        case executing
        case completed
        case failed
    }

    struct BridgeStep: Codable {
        let id: String
        let actionType: String
        let targetApp: String
        let expectedOutcome: String
    }

    private struct BridgeStatus: Encodable {
        var state: NavigatorState = .idle
        var taskId: String?
        var command: String?
        var currentStep: BridgeStep?
        var completedSteps: [BridgeStep] = []
        var error: String?
    }

    private var navigatorStatus = BridgeStatus()

    // MARK: - Dependencies (injected via configure)

    private var gatewayClient: GatewayClient?
    private var captureService: ScreenCaptureService?
    private var contextProvider: (@MainActor () -> NavigatorContextPayload)?

    // MARK: - Networking

    private var listener: NWListener?
    private var sseConnections: [NWConnection] = []
    private let bridgeQueue = DispatchQueue(label: "vibecat.e2e.bridge", qos: .utility)

    private init() {}

    // MARK: - Public API: configuration & startup

    /// Wire the bridge to live app components. Must be called before startIfEnabled().
    func configure(
        gatewayClient: GatewayClient,
        captureService: ScreenCaptureService,
        contextProvider: @escaping @MainActor () -> NavigatorContextPayload
    ) {
        self.gatewayClient = gatewayClient
        self.captureService = captureService
        self.contextProvider = contextProvider
    }

    /// Start the bridge only when VIBECAT_E2E_CONTROL=1 is set.
    func startIfEnabled() {
        guard ProcessInfo.processInfo.environment["VIBECAT_E2E_CONTROL"] == "1" else { return }
        startServer()
    }

    // MARK: - Public API: navigator state observers (called by AppDelegate hooks)

    func notifyCommandAccepted(taskId: String?, command: String) {
        navigatorStatus = BridgeStatus(
            state: .executing,
            taskId: taskId,
            command: command,
            currentStep: nil,
            completedSteps: [],
            error: nil
        )
        emitEvent([
            "type": "navigator.commandAccepted",
            "taskId": taskId ?? "",
            "command": command
        ])
    }

    func notifyStepPlanned(taskId: String, step: NavigatorStep) {
        let bs = BridgeStep(
            id: step.id,
            actionType: step.actionType.rawValue,
            targetApp: step.targetApp,
            expectedOutcome: step.expectedOutcome
        )
        navigatorStatus.currentStep = bs
        emitEvent([
            "type": "navigator.stepPlanned",
            "taskId": taskId,
            "stepId": step.id,
            "actionType": step.actionType.rawValue,
            "expectedOutcome": step.expectedOutcome
        ])
    }

    func notifyStepVerified(taskId: String, stepId: String, observedOutcome: String) {
        if let step = navigatorStatus.currentStep {
            navigatorStatus.completedSteps.append(step)
        }
        navigatorStatus.currentStep = nil
        emitEvent([
            "type": "navigator.stepVerified",
            "taskId": taskId,
            "stepId": stepId,
            "observedOutcome": observedOutcome
        ])
    }

    func notifyCompleted(taskId: String?, summary: String) {
        navigatorStatus.state = .completed
        navigatorStatus.currentStep = nil
        navigatorStatus.error = nil
        emitEvent([
            "type": "navigator.completed",
            "taskId": taskId ?? "",
            "summary": summary
        ])
    }

    func notifyFailed(taskId: String?, reason: String) {
        navigatorStatus.state = .failed
        navigatorStatus.currentStep = nil
        navigatorStatus.error = reason
        emitEvent([
            "type": "navigator.failed",
            "taskId": taskId ?? "",
            "reason": reason
        ])
    }

    // MARK: - Server Startup

    private func startServer() {
        let port: NWEndpoint.Port = 9876
        do {
            let params = NWParameters.tcp
            params.allowLocalEndpointReuse = true
            let newListener = try NWListener(using: params, on: port)
            self.listener = newListener

            newListener.newConnectionHandler = { [weak self] connection in
                Task { @MainActor [weak self] in
                    self?.acceptConnection(connection)
                }
            }

            newListener.stateUpdateHandler = { newState in
                Task { @MainActor in
                    switch newState {
                    case .ready:
                        NSLog("[E2E-BRIDGE] E2E Control Bridge started on localhost:9876")
                    case .failed(let error):
                        NSLog("[E2E-BRIDGE] Listener failed: %@", error.localizedDescription)
                    case .cancelled:
                        NSLog("[E2E-BRIDGE] Listener cancelled")
                    default:
                        break
                    }
                }
            }

            newListener.start(queue: bridgeQueue)
        } catch {
            NSLog("[E2E-BRIDGE] Failed to create NWListener: %@", error.localizedDescription)
        }
    }

    // MARK: - Connection Lifecycle

    private func acceptConnection(_ connection: NWConnection) {
        connection.start(queue: bridgeQueue)
        readHTTPRequest(from: connection)
    }

    private func readHTTPRequest(from connection: NWConnection) {
        connection.receive(minimumIncompleteLength: 1, maximumLength: 65_536) { [weak self] data, _, isComplete, error in
            Task { @MainActor [weak self] in
                guard let self else { return }
                if let data, !data.isEmpty {
                    await self.routeRequest(data: data, connection: connection)
                } else if isComplete || error != nil {
                    connection.cancel()
                }
            }
        }
    }

    // MARK: - HTTP Routing

    private func routeRequest(data: Data, connection: NWConnection) async {
        guard let text = String(data: data, encoding: .utf8) else {
            sendResponse(connection: connection, status: 400, body: #"{"error":"bad encoding"}"#)
            return
        }

        // Parse request line (first line before \r\n)
        let requestLine = text.components(separatedBy: "\r\n").first ?? ""
        let tokens = requestLine.components(separatedBy: " ")
        guard tokens.count >= 2 else {
            sendResponse(connection: connection, status: 400, body: #"{"error":"bad request"}"#)
            return
        }

        let method = tokens[0]
        // Strip query string from path
        let path = tokens[1].components(separatedBy: "?").first ?? tokens[1]

        // Extract body (content after double CRLF separator)
        let bodyData: Data?
        if let range = data.range(of: Data("\r\n\r\n".utf8)) {
            let slice = data[range.upperBound...]
            bodyData = slice.isEmpty ? nil : Data(slice)
        } else {
            bodyData = nil
        }

        switch (method, path) {
        case ("POST", "/e2e/command"):
            handleCommand(body: bodyData, connection: connection)
        case ("GET", "/e2e/status"):
            handleStatus(connection: connection)
        case ("GET", "/e2e/events"):
            handleEvents(connection: connection)
        case ("POST", "/e2e/screenshot"):
            await handleScreenshot(connection: connection)
        default:
            sendResponse(connection: connection, status: 404, body: #"{"error":"not found"}"#)
        }
    }

    // MARK: - POST /e2e/command

    private func handleCommand(body: Data?, connection: NWConnection) {
        struct CommandRequest: Decodable {
            let command: String
            // context is optional; when absent the bridge builds it automatically
            let context: [String: String]?
        }

        guard let body, !body.isEmpty else {
            sendResponse(connection: connection, status: 400, body: #"{"error":"missing body"}"#)
            return
        }

        guard let request = try? JSONDecoder().decode(CommandRequest.self, from: body) else {
            sendResponse(connection: connection, status: 400, body: #"{"error":"invalid json"}"#)
            return
        }

        let trimmed = request.command.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            sendResponse(connection: connection, status: 400, body: #"{"error":"command is empty"}"#)
            return
        }

        // Build navigator context (sync; no fresh screenshot — gateway already has recent state)
        let context = contextProvider?() ?? emptyContext()
        gatewayClient?.sendNavigatorCommand(trimmed, context: context)

        let taskId = "e2e_" + UUID().uuidString.replacingOccurrences(of: "-", with: "").lowercased()
        NSLog("[E2E-BRIDGE] /e2e/command accepted command=%@ taskId=%@", trimmed, taskId)

        let payload: [String: Any] = ["taskId": taskId, "accepted": true]
        guard let data = try? JSONSerialization.data(withJSONObject: payload),
              let responseBody = String(data: data, encoding: .utf8) else {
            sendResponse(connection: connection, status: 500, body: #"{"error":"internal"}"#)
            return
        }
        sendResponse(connection: connection, status: 200, body: responseBody)
    }

    // MARK: - GET /e2e/status

    private func handleStatus(connection: NWConnection) {
        guard let data = try? JSONEncoder().encode(navigatorStatus),
              let body = String(data: data, encoding: .utf8) else {
            sendResponse(connection: connection, status: 500, body: #"{"error":"encode failed"}"#)
            return
        }
        sendResponse(connection: connection, status: 200, body: body)
    }

    // MARK: - GET /e2e/events (SSE)

    private func handleEvents(connection: NWConnection) {
        let headers = [
            "HTTP/1.1 200 OK",
            "Content-Type: text/event-stream",
            "Cache-Control: no-cache",
            "Connection: keep-alive",
            "Access-Control-Allow-Origin: *",
            "",
            ""
        ].joined(separator: "\r\n")

        guard let headerData = headers.data(using: .utf8) else {
            connection.cancel()
            return
        }

        connection.send(content: headerData, completion: .contentProcessed { _ in })
        sseConnections.append(connection)

        NSLog("[E2E-BRIDGE] SSE client connected (total=%d)", sseConnections.count)

        // Track disconnection so we prune the dead connection
        connection.stateUpdateHandler = { [weak self] state in
            switch state {
            case .cancelled, .failed:
                Task { @MainActor [weak self] in
                    guard let self else { return }
                    self.sseConnections.removeAll { $0 === connection }
                    NSLog("[E2E-BRIDGE] SSE client disconnected (remaining=%d)", self.sseConnections.count)
                }
            default:
                break
            }
        }
    }

    // MARK: - POST /e2e/screenshot

    private func handleScreenshot(connection: NWConnection) async {
        guard let captureService else {
            sendResponse(connection: connection, status: 503, body: #"{"error":"capture service unavailable"}"#)
            return
        }

        NSLog("[E2E-BRIDGE] /e2e/screenshot capturing…")
        let result = await captureService.forceCapture()

        switch result {
        case .captured(let snapshot):
            let cgImage = snapshot.image
            let bitmapRep = NSBitmapImageRep(cgImage: cgImage)
            guard let jpegData = bitmapRep.representation(using: .jpeg, properties: [.compressionFactor: 0.8]) else {
                sendResponse(connection: connection, status: 500, body: #"{"error":"jpeg conversion failed"}"#)
                return
            }
            let base64String = jpegData.base64EncodedString()
            let payload: [String: Any] = [
                "image": base64String,
                "width": cgImage.width,
                "height": cgImage.height,
                "displayId": snapshot.displayID
            ]
            guard let data = try? JSONSerialization.data(withJSONObject: payload),
                  let body = String(data: data, encoding: .utf8) else {
                sendResponse(connection: connection, status: 500, body: #"{"error":"json encode failed"}"#)
                return
            }
            NSLog("[E2E-BRIDGE] /e2e/screenshot captured %dx%d display=%@", cgImage.width, cgImage.height, snapshot.displayID)
            sendResponse(connection: connection, status: 200, body: body)

        case .unavailable(let msg):
            NSLog("[E2E-BRIDGE] /e2e/screenshot unavailable: %@", msg)
            // Encode error message safely to avoid broken JSON
            if let data = try? JSONSerialization.data(withJSONObject: ["error": "capture unavailable: \(msg)"]),
               let body = String(data: data, encoding: .utf8) {
                sendResponse(connection: connection, status: 503, body: body)
            } else {
                sendResponse(connection: connection, status: 503, body: #"{"error":"capture unavailable"}"#)
            }

        case .unchanged:
            // Screen unchanged since last capture — return 503 so tests can detect this
            sendResponse(connection: connection, status: 503, body: #"{"error":"screen unchanged since last capture"}"#)
        }
    }

    // MARK: - SSE Emission

    /// Emit a server-sent event to all connected SSE clients.
    private func emitEvent(_ payload: [String: String]) {
        guard !sseConnections.isEmpty else { return }

        var jsonPayload: [String: Any] = [:]
        for (k, v) in payload { jsonPayload[k] = v }

        guard let jsonData = try? JSONSerialization.data(withJSONObject: jsonPayload),
              let jsonString = String(data: jsonData, encoding: .utf8) else {
            NSLog("[E2E-BRIDGE] emitEvent: json encode failed")
            return
        }

        let sseMessage = "data: \(jsonString)\n\n"
        guard let messageData = sseMessage.data(using: .utf8) else { return }

        for connection in sseConnections {
            connection.send(content: messageData, completion: .contentProcessed { _ in })
        }
    }

    // MARK: - HTTP Response Helpers

    private func sendResponse(
        connection: NWConnection,
        status code: Int,
        body: String,
        contentType: String = "application/json"
    ) {
        let statusText: String
        switch code {
        case 200: statusText = "OK"
        case 400: statusText = "Bad Request"
        case 404: statusText = "Not Found"
        case 500: statusText = "Internal Server Error"
        case 503: statusText = "Service Unavailable"
        default:  statusText = "Unknown"
        }

        let bodyData = body.data(using: .utf8) ?? Data()
        let headerString = [
            "HTTP/1.1 \(code) \(statusText)",
            "Content-Type: \(contentType); charset=utf-8",
            "Content-Length: \(bodyData.count)",
            "Connection: close",
            "",
            ""
        ].joined(separator: "\r\n")

        var responseData = headerString.data(using: .utf8)!
        responseData.append(bodyData)

        connection.send(content: responseData, completion: .contentProcessed { _ in
            connection.cancel()
        })
    }

    // MARK: - Context Fallback

    private func emptyContext() -> NavigatorContextPayload {
        NavigatorContextPayload(
            appName: "",
            bundleId: "",
            frontmostBundleId: "",
            windowTitle: "",
            focusedRole: "",
            focusedLabel: "",
            selectedText: "",
            axSnapshot: "",
            inputFieldHint: "",
            lastInputFieldDescriptor: "",
            screenshot: "",
            focusStableMs: 0,
            captureConfidence: 0,
            visibleInputCandidateCount: 0,
            accessibilityPermission: "unknown",
            accessibilityTrusted: false
        )
    }
}
