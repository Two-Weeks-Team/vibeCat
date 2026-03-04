import Foundation
import Network
import VibeCatCore

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
    var onReconnecting: ((Int) -> Void)?
    var onReconnectExhausted: (() -> Void)?
    var onDisconnected: (() -> Void)?
    var onLatencyUpdate: ((Int) -> Void)?

    var lastErrorDescription: String?
    private(set) var sessionHandle: String?
    var isConnected: Bool {
        if case .connected = state { return true }
        return false
    }

    private var webSocketTask: URLSessionWebSocketTask?
    private var urlSession: URLSession?
    private var heartbeatTimer: Timer?
    private var state: ConnectionState = .disconnected {
        didSet { onStateChange?(state) }
    }
    private var sessionId: String?
    private var reconnectWorkItem: DispatchWorkItem?
    private var reconnectAttempts = 0
    private var lastConnectStartedAt: Date?
    private var rapidFailureCount = 0
    private var circuitBreakerOpen = false
    private var isNetworkAvailable = true
    private var lastPongAt: Date?
    private var isManuallyDisconnected = false
    private var currentAPIKey: String?
    private var setupSoul: String?
    private var lastPingSentAt: Date?

    private let pathMonitor = NWPathMonitor()
    private let pathMonitorQueue = DispatchQueue(label: "vibecat.gateway.path-monitor")

    private let maxReconnectAttempts = 30
    private let pingInterval: TimeInterval = 15
    private let zombieTimeout: TimeInterval = 45
    private let rapidFailureWindow: TimeInterval = 5

    private let settings = AppSettings.shared

    init() {
        pathMonitor.pathUpdateHandler = { [weak self] path in
            Task { @MainActor [weak self] in
                self?.handleNetworkPath(path)
            }
        }
        pathMonitor.start(queue: pathMonitorQueue)
    }

    deinit {
        pathMonitor.cancel()
    }

    func connect(apiKey: String) {
        currentAPIKey = apiKey
        isManuallyDisconnected = false
        reconnectAttempts = 0
        rapidFailureCount = 0
        circuitBreakerOpen = false
        reconnectWorkItem?.cancel()
        reconnectWorkItem = nil
        lastErrorDescription = nil
        establishConnection(apiKey: apiKey)
    }

    func disconnect() {
        isManuallyDisconnected = true
        reconnectWorkItem?.cancel()
        reconnectWorkItem = nil
        stopHeartbeatTimer()
        closeConnection()
        lastErrorDescription = nil
        state = .disconnected
        onDisconnected?()
    }

    func reconnect(apiKey: String) {
        currentAPIKey = apiKey
        isManuallyDisconnected = false
        reconnectAttempts = 0
        rapidFailureCount = 0
        circuitBreakerOpen = false
        reconnectWorkItem?.cancel()
        reconnectWorkItem = nil
        stopHeartbeatTimer()
        closeConnection()
        lastErrorDescription = nil
        state = .disconnected
        scheduleReconnect(immediate: true)
    }

    func setSoul(_ soul: String?) {
        let trimmed = soul?.trimmingCharacters(in: .whitespacesAndNewlines)
        setupSoul = (trimmed?.isEmpty == false) ? trimmed : nil
    }

    func resendSetupPayloadIfConnected() {
        guard case .connected = state else { return }
        sendSetupPayload()
    }

    func sendAudio(_ pcmData: Data) {
        guard case .connected = state else { return }
        webSocketTask?.send(.data(pcmData)) { _ in }
    }

    func sendText(_ text: String) {
        guard case .connected = state else { return }
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return }
        let payload: [String: Any] = [
            "clientContent": [
                "turnComplete": true,
                "turns": [
                    [
                        "role": "user",
                        "parts": [["text": trimmed]]
                    ]
                ]
            ]
        ]
        sendJSON(payload)
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

    private func establishConnection(apiKey: String) {
        if case .connecting = state { return }
        if case .connected = state { return }

        state = .connecting
        lastConnectStartedAt = Date()
        sessionId = nil

        guard let url = URL(string: settings.gatewayURL) else {
            lastErrorDescription = "Gateway URL invalid — check Settings"
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
        startHeartbeatTimer()
    }

    private func sendSetupPayload() {
        var config: [String: Any] = [
            "voice": settings.voice,
            "language": settings.language,
            "liveModel": settings.liveModel,
            "proactiveAudio": settings.proactiveAudio,
            "searchEnabled": settings.searchEnabled
        ]
        if let setupSoul {
            config["soul"] = setupSoul
        }

        let payload: [String: Any] = [
            "type": "setup",
            "config": config
        ]
        var mutablePayload = payload
        if let sessionHandle, !sessionHandle.isEmpty {
            mutablePayload["sessionHandle"] = sessionHandle
        }
        sendJSON(mutablePayload)
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
                handleConnectionDropped(error: error)
                break
            }
        }
    }

    private func handleWebSocketMessage(_ message: URLSessionWebSocketTask.Message) {
        updateHeartbeatReceipt()
        switch message {
        case .data(let data):
            let parsed = AudioMessageParser.parse(data)
            switch parsed {
            case .audio(let audioData):
                onAudioData?(audioData)
            case .setupComplete(let sid):
                sessionId = sid
                reconnectAttempts = 0
                rapidFailureCount = 0
                lastPongAt = Date()
                lastErrorDescription = nil
                state = .connected(sessionId: sid)
            case .sessionResumptionUpdate(let handle):
                let trimmed = handle.trimmingCharacters(in: .whitespacesAndNewlines)
                sessionHandle = trimmed.isEmpty ? nil : trimmed
                onMessage?(parsed)
            default:
                onMessage?(parsed)
            }
        case .string(let text):
            guard let data = text.data(using: .utf8) else { return }
            let parsed = AudioMessageParser.parse(data)
            switch parsed {
            case .setupComplete(let sid):
                sessionId = sid
                reconnectAttempts = 0
                rapidFailureCount = 0
                lastPongAt = Date()
                lastErrorDescription = nil
                state = .connected(sessionId: sid)
            case .sessionResumptionUpdate(let handle):
                let trimmed = handle.trimmingCharacters(in: .whitespacesAndNewlines)
                sessionHandle = trimmed.isEmpty ? nil : trimmed
                onMessage?(parsed)
            default:
                onMessage?(parsed)
            }
        @unknown default:
            break
        }
    }

    private func startHeartbeatTimer() {
        stopHeartbeatTimer()
        heartbeatTimer = Timer.scheduledTimer(withTimeInterval: pingInterval, repeats: true) { [weak self] _ in
            Task { @MainActor [weak self] in
                self?.runHeartbeatHealthCheck()
            }
        }
    }

    private func stopHeartbeatTimer() {
        heartbeatTimer?.invalidate()
        heartbeatTimer = nil
    }

    private func runHeartbeatHealthCheck() {
        guard case .connected = state else { return }

        if let lastPongAt, Date().timeIntervalSince(lastPongAt) > zombieTimeout {
            lastErrorDescription = "Connection timed out — Reconnecting…"
            handleConnectionDropped(error: GatewayError.pongTimeout)
            return
        }

        lastPingSentAt = Date()
        let heartbeat: [String: Any] = [
            "clientContent": [
                "turnComplete": false,
                "turns": []
            ]
        ]
        sendJSON(heartbeat)
    }

    private func handleConnectionDropped(error: Error) {
        if isManuallyDisconnected { return }
        if case .disconnected = state { return }

        stopHeartbeatTimer()
        closeConnection()
        onDisconnected?()
        lastErrorDescription = friendlyErrorDescription(for: error)
        state = .failed(error)

        if let startedAt = lastConnectStartedAt,
           Date().timeIntervalSince(startedAt) <= rapidFailureWindow {
            rapidFailureCount += 1
        } else {
            rapidFailureCount = 0
        }

        if rapidFailureCount >= 3 {
            circuitBreakerOpen = true
            lastErrorDescription = "Connection keeps failing — check API key"
            onReconnectExhausted?()
            state = .disconnected
            return
        }

        state = .disconnected
        scheduleReconnect(immediate: false)
    }

    private func scheduleReconnect(immediate: Bool) {
        guard !isManuallyDisconnected else { return }
        guard !circuitBreakerOpen else {
            onReconnectExhausted?()
            return
        }
        guard isNetworkAvailable else { return }
        guard let apiKey = currentAPIKey, !apiKey.isEmpty else {
            onReconnectExhausted?()
            return
        }

        guard reconnectAttempts < maxReconnectAttempts else {
            onReconnectExhausted?()
            return
        }

        reconnectWorkItem?.cancel()
        reconnectAttempts += 1
        onReconnecting?(reconnectAttempts)

        let delay: TimeInterval
        if immediate {
            delay = 0
        } else {
            let exponential = min(pow(2.0, Double(reconnectAttempts - 1)), 60.0)
            let jitter = Double.random(in: 0...(exponential * 0.25))
            delay = exponential + jitter
        }

        let workItem = DispatchWorkItem { [weak self] in
            Task { @MainActor [weak self] in
                guard let self else { return }
                guard !self.isManuallyDisconnected else { return }
                guard self.isNetworkAvailable else { return }
                if case .connected = self.state { return }
                self.establishConnection(apiKey: apiKey)
            }
        }
        reconnectWorkItem = workItem
        DispatchQueue.main.asyncAfter(deadline: .now() + delay, execute: workItem)
    }

    private func handleNetworkPath(_ path: NWPath) {
        let wasAvailable = isNetworkAvailable
        isNetworkAvailable = path.status == .satisfied

        guard isNetworkAvailable else {
            lastErrorDescription = "No internet connection — will reconnect when available"
            if case .connected = state {
                handleConnectionDropped(error: GatewayError.networkUnavailable)
            } else {
                state = .disconnected
                onDisconnected?()
            }
            return
        }

        if !wasAvailable {
            lastErrorDescription = nil
        }
        guard !wasAvailable else { return }
        guard !isManuallyDisconnected else { return }

        switch state {
        case .disconnected, .failed:
            scheduleReconnect(immediate: true)
        default:
            break
        }
    }

    private func closeConnection() {
        webSocketTask?.cancel(with: .normalClosure, reason: nil)
        webSocketTask = nil
        urlSession = nil
    }

    private func updateHeartbeatReceipt() {
        let now = Date()
        lastPongAt = now
        if let sentAt = lastPingSentAt {
            let latency = Int((now.timeIntervalSince(sentAt) * 1000).rounded())
            onLatencyUpdate?(max(0, latency))
            lastPingSentAt = nil
        }
    }

    enum GatewayError: Error, LocalizedError {
        case invalidURL
        case pongTimeout
        case networkUnavailable

        var errorDescription: String? {
            switch self {
            case .invalidURL:
                return "Gateway URL invalid — check Settings"
            case .pongTimeout:
                return "Connection timed out — Reconnecting…"
            case .networkUnavailable:
                return "No internet connection — will reconnect when available"
            }
        }
    }

    private func friendlyErrorDescription(for error: Error) -> String {
        if let gatewayError = error as? GatewayError {
            return gatewayError.errorDescription ?? "Connection error"
        }

        let nsError = error as NSError
        if nsError.domain == NSURLErrorDomain {
            switch nsError.code {
            case NSURLErrorNotConnectedToInternet,
                 NSURLErrorNetworkConnectionLost,
                 NSURLErrorInternationalRoamingOff,
                 NSURLErrorDataNotAllowed,
                 NSURLErrorCannotFindHost,
                 NSURLErrorCannotConnectToHost:
                return "No internet connection — will reconnect when available"
            default:
                break
            }
        }
        return "Connection timed out — Reconnecting…"
    }
}
