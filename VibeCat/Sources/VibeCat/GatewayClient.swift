import Foundation
import VibeCatCore

/// WebSocket client connecting to the Realtime Gateway.
/// Handles setup, audio streaming, screen capture routing, and server message dispatch.
@MainActor
final class GatewayClient {
    enum ConnectionState {
        case disconnected
        case connecting
        case connected(sessionId: String)
        case failed(Error)
    }

    var onMessage: ((ServerMessage) -> Void)?
    var onStateChange: ((ConnectionState) -> Void)?
    var onAudioData: ((Data) -> Void)?

    private var webSocketTask: URLSessionWebSocketTask?
    private var urlSession: URLSession?
    private var heartbeatTimer: Timer?
    private var state: ConnectionState = .disconnected {
        didSet { onStateChange?(state) }
    }
    private var sessionId: String?

    private let settings = AppSettings.shared

    func connect(apiKey: String) {
        guard case .disconnected = state else { return }
        state = .connecting

        guard let url = URL(string: settings.gatewayURL) else {
            state = .failed(GatewayError.invalidURL)
            return
        }

        var request = URLRequest(url: url)
        request.setValue("Bearer \(apiKey)", forHTTPHeaderField: "Authorization")

        let session = URLSession(configuration: .default)
        self.urlSession = session
        let task = session.webSocketTask(with: request)
        self.webSocketTask = task
        task.resume()

        sendSetupPayload()
        startReceiveLoop()
        startHeartbeat()
    }

    func disconnect() {
        heartbeatTimer?.invalidate()
        heartbeatTimer = nil
        webSocketTask?.cancel(with: .normalClosure, reason: nil)
        webSocketTask = nil
        urlSession = nil
        state = .disconnected
    }

    func reconnect(apiKey: String) {
        disconnect()
        connect(apiKey: apiKey)
    }

    func sendAudio(_ pcmData: Data) {
        guard case .connected = state else { return }
        webSocketTask?.send(.data(pcmData)) { _ in }
    }

    func sendScreenCapture(imageBase64: String, context: String, userId: String) {
        guard case .connected(let sid) = state else { return }
        let payload: [String: Any] = [
            "type": "screenCapture",
            "image": imageBase64,
            "context": context,
            "sessionId": sid,
            "userId": userId
        ]
        sendJSON(payload)
    }

    func sendForceCapture(imageBase64: String, context: String, userId: String) {
        guard case .connected(let sid) = state else { return }
        let payload: [String: Any] = [
            "type": "forceCapture",
            "image": imageBase64,
            "context": context,
            "sessionId": sid,
            "userId": userId
        ]
        sendJSON(payload)
    }

    private func sendSetupPayload() {
        let payload: [String: Any] = [
            "type": "setup",
            "config": [
                "voice": settings.voice,
                "language": settings.language,
                "liveModel": settings.liveModel,
                "proactiveAudio": settings.proactiveAudio,
                "searchEnabled": settings.searchEnabled
            ]
        ]
        sendJSON(payload)
    }

    private func sendJSON(_ payload: [String: Any]) {
        guard let data = try? JSONSerialization.data(withJSONObject: payload),
              let text = String(data: data, encoding: .utf8) else { return }
        webSocketTask?.send(.string(text)) { _ in }
    }

    private func startReceiveLoop() {
        Task { [weak self] in
            await self?.receiveLoop()
        }
    }

    private func receiveLoop() async {
        while let task = webSocketTask {
            do {
                let message = try await task.receive()
                handleWebSocketMessage(message)
            } catch {
                if case .connected = state {
                    state = .failed(error)
                }
                break
            }
        }
    }

    private func handleWebSocketMessage(_ message: URLSessionWebSocketTask.Message) {
        switch message {
        case .data(let data):
            let parsed = AudioMessageParser.parse(data)
            switch parsed {
            case .audio(let audioData):
                onAudioData?(audioData)
            case .setupComplete(let sid):
                sessionId = sid
                state = .connected(sessionId: sid)
            default:
                onMessage?(parsed)
            }
        case .string(let text):
            guard let data = text.data(using: .utf8) else { return }
            let parsed = AudioMessageParser.parse(data)
            switch parsed {
            case .setupComplete(let sid):
                sessionId = sid
                state = .connected(sessionId: sid)
            default:
                onMessage?(parsed)
            }
        @unknown default:
            break
        }
    }

    private func startHeartbeat() {
        heartbeatTimer = Timer.scheduledTimer(withTimeInterval: 30, repeats: true) { [weak self] _ in
            Task { @MainActor [weak self] in
                self?.sendPing()
            }
        }
    }

    private func sendPing() {
        guard case .connected = state else { return }
        sendJSON(["type": "ping"])
    }

    enum GatewayError: Error, LocalizedError {
        case invalidURL

        var errorDescription: String? { "Invalid Gateway URL in settings" }
    }
}
