import Foundation

@MainActor
final class AutomationMCPClient: Sendable {

    static let shared = AutomationMCPClient()

    private let port: Int = 3010
    private let session: URLSession = .shared
    private var sidecarProcess: Process?
    private var requestCounter = 0

    private var baseURL: URL { URL(string: "http://localhost:\(port)/stream")! }

    // MARK: - Sidecar Lifecycle

    var isRunning: Bool { sidecarProcess?.isRunning ?? false }

    func startSidecar() {
        guard !isRunning else {
            NSLog("[AMCP] sidecar already running pid=%d", sidecarProcess?.processIdentifier ?? 0)
            return
        }

        let bunPath = resolveBunPath()
        guard FileManager.default.fileExists(atPath: bunPath) else {
            NSLog("[AMCP] bun not found at %@", bunPath)
            return
        }

        let scriptPath = resolveMCPScriptPath()
        guard FileManager.default.fileExists(atPath: scriptPath) else {
            NSLog("[AMCP] automation-mcp index.ts not found at %@", scriptPath)
            return
        }

        let proc = Process()
        proc.executableURL = URL(fileURLWithPath: bunPath)
        proc.arguments = ["run", scriptPath]
        proc.currentDirectoryURL = URL(fileURLWithPath: scriptPath).deletingLastPathComponent()

        let pipe = Pipe()
        proc.standardOutput = pipe
        proc.standardError = pipe

        pipe.fileHandleForReading.readabilityHandler = { handle in
            let data = handle.availableData
            guard !data.isEmpty, let line = String(data: data, encoding: .utf8) else { return }
            NSLog("[AMCP-OUT] %@", line.trimmingCharacters(in: .whitespacesAndNewlines))
        }

        proc.terminationHandler = { [weak self] p in
            NSLog("[AMCP] sidecar terminated code=%d", p.terminationStatus)
            DispatchQueue.main.async { self?.sidecarProcess = nil }
        }

        do {
            try proc.run()
            sidecarProcess = proc
            NSLog("[AMCP] sidecar started pid=%d port=%d", proc.processIdentifier, port)
        } catch {
            NSLog("[AMCP] failed to start sidecar: %@", error.localizedDescription)
        }
    }

    func stopSidecar() {
        guard let proc = sidecarProcess, proc.isRunning else { return }
        proc.terminate()
        sidecarProcess = nil
        NSLog("[AMCP] sidecar stopped")
    }

    // MARK: - Mouse

    func mouseClick(x: Int, y: Int, button: String = "left") async -> Bool {
        let result = await callTool("mouseClick", arguments: [
            "x": x, "y": y, "button": button
        ])
        return result != nil
    }

    func mouseMove(x: Int, y: Int) async -> Bool {
        let result = await callTool("mouseMove", arguments: ["x": x, "y": y])
        return result != nil
    }

    func mouseDoubleClick(x: Int, y: Int) async -> Bool {
        let result = await callTool("mouseDoubleClick", arguments: [
            "x": x, "y": y, "button": "left"
        ])
        return result != nil
    }

    // MARK: - Keyboard

    func typeText(_ text: String) async -> Bool {
        let result = await callTool("type", arguments: ["text": text])
        return result != nil
    }

    func pressKeys(_ keys: [String]) async -> Bool {
        let result = await callTool("type", arguments: ["keys": keys.joined(separator: ",")])
        return result != nil
    }

    func systemCommand(_ command: String) async -> Bool {
        let result = await callTool("systemCommand", arguments: ["command": command])
        return result != nil
    }

    // MARK: - Window

    func focusWindow(title: String) async -> Bool {
        let result = await callTool("windowControl", arguments: [
            "action": "focus", "windowTitle": title
        ])
        return result != nil
    }

    // MARK: - JSON-RPC Transport

    private func callTool(_ name: String, arguments: [String: Any]) async -> String? {
        guard isRunning else {
            NSLog("[AMCP] sidecar not running, skipping %@", name)
            return nil
        }

        requestCounter += 1
        let reqID = requestCounter

        let body: [String: Any] = [
            "jsonrpc": "2.0",
            "id": reqID,
            "method": "tools/call",
            "params": [
                "name": name,
                "arguments": arguments
            ]
        ]

        guard let jsonData = try? JSONSerialization.data(withJSONObject: body) else {
            NSLog("[AMCP] failed to serialize request for %@", name)
            return nil
        }

        var request = URLRequest(url: baseURL)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        request.timeoutInterval = 10
        request.httpBody = jsonData

        do {
            let (data, response) = try await session.data(for: request)
            guard let http = response as? HTTPURLResponse, (200...299).contains(http.statusCode) else {
                NSLog("[AMCP] %@ failed http=%d", name, (response as? HTTPURLResponse)?.statusCode ?? -1)
                return nil
            }
            return parseToolResult(data: data, tool: name)
        } catch {
            NSLog("[AMCP] %@ error: %@", name, error.localizedDescription)
            return nil
        }
    }

    private func parseToolResult(data: Data, tool: String) -> String? {
        guard let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            return nil
        }

        if let result = json["result"] as? [String: Any],
           let content = result["content"] as? [[String: Any]],
           let first = content.first,
           let text = first["text"] as? String {
            NSLog("[AMCP] %@ ok: %@", tool, String(text.prefix(80)))
            return text
        }

        if let error = json["error"] as? [String: Any],
           let message = error["message"] as? String {
            NSLog("[AMCP] %@ rpc-error: %@", tool, message)
        }

        return nil
    }

    // MARK: - Path Resolution

    private func resolveBunPath() -> String {
        let homeDir = FileManager.default.homeDirectoryForCurrentUser.path
        let candidates = [
            "\(homeDir)/.bun/bin/bun",
            "/usr/local/bin/bun",
            "/opt/homebrew/bin/bun"
        ]
        return candidates.first { FileManager.default.fileExists(atPath: $0) } ?? candidates[0]
    }

    private func resolveMCPScriptPath() -> String {
        let bundledPath = Bundle.main.resourcePath.map { "\($0)/../../../tools/automation-mcp/index.ts" } ?? ""
        let devPath = FileManager.default.currentDirectoryPath + "/tools/automation-mcp/index.ts"

        if FileManager.default.fileExists(atPath: bundledPath) { return bundledPath }

        let repoRoot = findRepoRoot()
        let repoPath = "\(repoRoot)/tools/automation-mcp/index.ts"
        if FileManager.default.fileExists(atPath: repoPath) { return repoPath }

        return devPath
    }

    private func findRepoRoot() -> String {
        var dir = URL(fileURLWithPath: #file).deletingLastPathComponent()
        for _ in 0..<6 {
            let gitDir = dir.appendingPathComponent(".git")
            if FileManager.default.fileExists(atPath: gitDir.path) { return dir.path }
            dir = dir.deletingLastPathComponent()
        }
        return FileManager.default.currentDirectoryPath
    }
}
