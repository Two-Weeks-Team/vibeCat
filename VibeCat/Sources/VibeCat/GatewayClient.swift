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
    private var protocolPingTimer: Timer?
    private var awaitingPong = false
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
    private(set) var isTTSSpeaking = false
    /// Timestamp when the last TTS speech ended. Used by ScreenAnalyzer for post-speech cooldown.
    private(set) var lastSpeechEndTime: Date = .distantPast
    private var ttsEndCooldownTask: Task<Void, Never>?
    private var currentAPIKey: String?
    private var currentSessionToken: String?
    private var setupSoul: String?
    private var lastPingSentAt: Date?

    private let pathMonitor = NWPathMonitor()
    private let pathMonitorQueue = DispatchQueue(label: "vibecat.gateway.path-monitor")

    static func deviceIdentifier() -> String {
        let key = "vibecat.deviceId"
        if let existing = UserDefaults.standard.string(forKey: key) {
            return existing
        }
        let newID = UUID().uuidString
        UserDefaults.standard.set(newID, forKey: key)
        return newID
    }

    private let maxReconnectAttempts = 30
    private let appHeartbeatInterval: TimeInterval = 30
    private let protocolPingInterval: TimeInterval = 15
    private let pongTimeout: TimeInterval = 60
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
        currentSessionToken = nil
        isManuallyDisconnected = false
        reconnectAttempts = 0
        rapidFailureCount = 0
        circuitBreakerOpen = false
        reconnectWorkItem?.cancel()
        reconnectWorkItem = nil
        lastErrorDescription = nil
        Task { [weak self] in
            await self?.registerAndConnect(apiKey: apiKey)
        }
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
        currentSessionToken = nil
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
        Task { [weak self] in
            await self?.registerAndConnect(apiKey: apiKey)
        }
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
        NSLog("[GW-OUT] sendAudio: %lu bytes", pcmData.count)
        webSocketTask?.send(.data(pcmData)) { _ in }
    }

    func sendVideoFrame(_ jpegData: Data) {
        guard case .connected = state else { return }
        NSLog("[GW-OUT] sendVideoFrame: %lu bytes", jpegData.count)
        webSocketTask?.send(.data(jpegData)) { _ in }
    }

    func sendText(_ text: String) {
        guard case .connected = state else { return }
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return }
        NSLog("[GW-OUT] sendText: %@", trimmed)
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

    func sendScreenCapture(imageBase64: String, context: String, userId: String, character: String, soul: String?, activityMinutes: Int = 0) {
        guard case .connected(let sid) = state else { return }
        NSLog("[GW-OUT] sendScreenCapture: image=%lu bytes, context=%@, character=%@, activityMinutes=%d", imageBase64.count, context, character, activityMinutes)
        var payload: [String: Any] = [
            "type": "screenCapture",
            "image": imageBase64,
            "context": context,
            "sessionId": sid,
            "userId": userId,
            "character": character,
            "activityMinutes": activityMinutes
        ]
        if let soul, !soul.isEmpty {
            payload["soul"] = soul
        }
        sendJSON(payload)
    }

    func sendForceCapture(imageBase64: String, context: String, userId: String, character: String, soul: String?, activityMinutes: Int = 0) {
        guard case .connected(let sid) = state else { return }
        NSLog("[GW-OUT] sendForceCapture: image=%lu bytes, context=%@, character=%@, activityMinutes=%d", imageBase64.count, context, character, activityMinutes)
        var payload: [String: Any] = [
            "type": "forceCapture",
            "image": imageBase64,
            "context": context,
            "sessionId": sid,
            "userId": userId,
            "character": character,
            "activityMinutes": activityMinutes
        ]
        if let soul, !soul.isEmpty {
            payload["soul"] = soul
        }
        sendJSON(payload)
    }

    private func restBaseURL() -> URL? {
        guard var components = URLComponents(string: settings.gatewayURL) else { return nil }
        switch components.scheme?.lowercased() {
        case "wss":
            components.scheme = "https"
        case "ws":
            components.scheme = "http"
        case "https", "http":
            break
        default:
            return nil
        }
        components.path = ""
        components.query = nil
        components.fragment = nil
        return components.url
    }

    private func registerAndConnect(apiKey: String) async {
        guard !apiKey.isEmpty else {
            lastErrorDescription = "Connection keeps failing — check API key"
            state = .failed(GatewayError.registrationFailed)
            return
        }
        guard let baseURL = restBaseURL() else {
            lastErrorDescription = "Gateway URL invalid — check Settings"
            state = .failed(GatewayError.invalidURL)
            return
        }

        let registerURL = baseURL.appendingPathComponent("api/v1/auth/register")
        var request = URLRequest(url: registerURL)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        let body: [String: String] = ["apiKey": apiKey]
        guard let bodyData = try? JSONSerialization.data(withJSONObject: body) else {
            lastErrorDescription = "Connection keeps failing — check API key"
            state = .failed(GatewayError.registrationFailed)
            return
        }
        request.httpBody = bodyData

        do {
            let (data, response) = try await URLSession.shared.data(for: request)
            guard let httpResponse = response as? HTTPURLResponse,
                  (200...299).contains(httpResponse.statusCode) else {
                lastErrorDescription = "Connection keeps failing — check API key"
                state = .failed(GatewayError.registrationFailed)
                return
            }

            let decoded = try JSONDecoder().decode(RegisterResponse.self, from: data)
            let token = decoded.sessionToken.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !token.isEmpty else {
                lastErrorDescription = "Connection keeps failing — check API key"
                state = .failed(GatewayError.registrationFailed)
                return
            }

            currentSessionToken = token
            establishConnection(token: token)
        } catch {
            lastErrorDescription = "Connection keeps failing — check API key"
            state = .failed(GatewayError.registrationFailed)
        }
    }

    private func establishConnection(token: String) {
        if case .connecting = state { return }
        if case .connected = state { return }

        state = .connecting
        lastConnectStartedAt = Date()
        sessionId = nil

        guard var urlComponents = URLComponents(string: settings.gatewayURL) else {
            lastErrorDescription = "Gateway URL invalid — check Settings"
            state = .failed(GatewayError.invalidURL)
            return
        }

        // Ensure /ws/live path is present for WebSocket endpoint
        if urlComponents.path.isEmpty || urlComponents.path == "/" {
            urlComponents.path = "/ws/live"
        } else if !urlComponents.path.hasSuffix("/ws/live") {
            let base = urlComponents.path.hasSuffix("/") ? String(urlComponents.path.dropLast()) : urlComponents.path
            urlComponents.path = base + "/ws/live"
        }

        guard let url = urlComponents.url else {
            lastErrorDescription = "Gateway URL invalid — check Settings"
            state = .failed(GatewayError.invalidURL)
            return
        }

        NSLog("[APP] establishConnection: url=%@, hasToken=%d", url.absoluteString, !token.isEmpty)

        var request = URLRequest(url: url)
        request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")

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
            "searchEnabled": settings.searchEnabled,
            "affectiveDialog": true,
            "deviceId": Self.deviceIdentifier()
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
        let type = payload["type"] as? String ?? "unknown"
        let preview = String(text.prefix(100))
        NSLog("[GW-OUT] sendJSON: type=%@, preview=%@", type, preview)
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
            NSLog("[GW-IN] data: %lu bytes", data.count)
            let parsed = AudioMessageParser.parse(data)
            switch parsed {
            case .audio(let audioData):
                NSLog("[GW-IN] message type=audio, size=%lu bytes", audioData.count)
                onAudioData?(audioData)
            case .setupComplete(let sid):
                NSLog("[GW-IN] message type=setupComplete, sessionId=%@", sid)
                sessionId = sid
                reconnectAttempts = 0
                rapidFailureCount = 0
                lastPongAt = Date()
                lastErrorDescription = nil
                state = .connected(sessionId: sid)
            case .sessionResumptionUpdate(let handle):
                NSLog("[GW-IN] message type=sessionResumptionUpdate")
                let trimmed = handle.trimmingCharacters(in: .whitespacesAndNewlines)
                sessionHandle = trimmed.isEmpty ? nil : trimmed
                onMessage?(parsed)
            case .ttsStart(let ttsText):
                NSLog("[GW-IN] message type=ttsStart, suppressing mic, hasText=%d", ttsText != nil ? 1 : 0)
                ttsEndCooldownTask?.cancel()
                ttsEndCooldownTask = nil
                isTTSSpeaking = true
                onMessage?(parsed)
            case .ttsEnd:
                NSLog("[GW-IN] message type=ttsEnd, cooldown before resuming mic")
                onMessage?(parsed)
                ttsEndCooldownTask?.cancel()
                ttsEndCooldownTask = Task { @MainActor [weak self] in
                    try? await Task.sleep(nanoseconds: 500_000_000)
                    guard !Task.isCancelled, let self else { return }
                    self.isTTSSpeaking = false
                    self.lastSpeechEndTime = Date()
                    NSLog("[GW-IN] mic resumed after cooldown")
                }
            default:
                NSLog("[GW-IN] message type=other")
                onMessage?(parsed)
            }
        case .string(let text):
            NSLog("[GW-IN] string: %lu chars", text.count)
            guard let data = text.data(using: .utf8) else { return }
            let parsed = AudioMessageParser.parse(data)
            switch parsed {
            case .setupComplete(let sid):
                NSLog("[GW-IN] message type=setupComplete, sessionId=%@", sid)
                sessionId = sid
                reconnectAttempts = 0
                rapidFailureCount = 0
                lastPongAt = Date()
                lastErrorDescription = nil
                state = .connected(sessionId: sid)
            case .sessionResumptionUpdate(let handle):
                NSLog("[GW-IN] message type=sessionResumptionUpdate")
                let trimmed = handle.trimmingCharacters(in: .whitespacesAndNewlines)
                sessionHandle = trimmed.isEmpty ? nil : trimmed
                onMessage?(parsed)
            case .ttsStart(let ttsText):
                NSLog("[GW-IN] message type=ttsStart, suppressing mic, hasText=%d", ttsText != nil ? 1 : 0)
                ttsEndCooldownTask?.cancel()
                ttsEndCooldownTask = nil
                isTTSSpeaking = true
                onMessage?(parsed)
            case .ttsEnd:
                NSLog("[GW-IN] message type=ttsEnd, cooldown before resuming mic")
                onMessage?(parsed)
                ttsEndCooldownTask?.cancel()
                ttsEndCooldownTask = Task { @MainActor [weak self] in
                    try? await Task.sleep(nanoseconds: 500_000_000)
                    guard !Task.isCancelled, let self else { return }
                    self.isTTSSpeaking = false
                    self.lastSpeechEndTime = Date()
                    NSLog("[GW-IN] mic resumed after cooldown")
                }
            default:
                NSLog("[GW-IN] message type=other")
                onMessage?(parsed)
            }
        @unknown default:
            NSLog("[GW-IN] unknown message type")
            break
        }
    }

    private func startHeartbeatTimer() {
        stopHeartbeatTimer()
        heartbeatTimer = Timer.scheduledTimer(withTimeInterval: appHeartbeatInterval, repeats: true) { [weak self] _ in
            Task { @MainActor [weak self] in
                self?.sendAppHeartbeat()
            }
        }
        protocolPingTimer = Timer.scheduledTimer(withTimeInterval: protocolPingInterval, repeats: true) { [weak self] _ in
            Task { @MainActor [weak self] in
                self?.sendProtocolPing()
            }
        }
    }

    private func stopHeartbeatTimer() {
        heartbeatTimer?.invalidate()
        heartbeatTimer = nil
        protocolPingTimer?.invalidate()
        protocolPingTimer = nil
        awaitingPong = false
    }

    private func sendAppHeartbeat() {
        guard case .connected = state else { return }
        sendJSON(["type": "ping"])
    }

    private func sendProtocolPing() {
        guard case .connected = state, let ws = webSocketTask else { return }

        if awaitingPong, let lastPongAt, Date().timeIntervalSince(lastPongAt) > pongTimeout {
            lastErrorDescription = "Connection timed out — Reconnecting…"
            handleConnectionDropped(error: GatewayError.pongTimeout)
            return
        }

        awaitingPong = true
        lastPingSentAt = Date()
        ws.sendPing { [weak self] error in
            Task { @MainActor [weak self] in
                guard let self else { return }
                if let error {
                    NSLog("[GatewayClient] Ping failed: %@", error.localizedDescription)
                } else {
                    self.awaitingPong = false
                    self.lastPongAt = Date()
                    if let sentAt = self.lastPingSentAt {
                        let latency = Int((Date().timeIntervalSince(sentAt) * 1000).rounded())
                        self.onLatencyUpdate?(max(0, latency))
                        self.lastPingSentAt = nil
                    }
                }
            }
        }
    }

    private func handleConnectionDropped(error: Error) {
        if isManuallyDisconnected { return }
        if case .disconnected = state { return }

        NSLog("[APP] handleConnectionDropped: %@", error.localizedDescription)

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
                if let token = self.currentSessionToken, !token.isEmpty {
                    self.establishConnection(token: token)
                } else {
                    await self.registerAndConnect(apiKey: apiKey)
                }
            }
        }
        reconnectWorkItem = workItem
        DispatchQueue.main.asyncAfter(deadline: .now() + delay, execute: workItem)
    }

    private struct RegisterResponse: Decodable {
        let sessionToken: String
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
        case registrationFailed
        case pongTimeout
        case networkUnavailable

        var errorDescription: String? {
            switch self {
            case .invalidURL:
                return "Gateway URL invalid — check Settings"
            case .registrationFailed:
                return "Connection keeps failing — check API key"
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
