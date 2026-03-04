import Foundation

@MainActor
final class ErrorReporter {
    static let shared = ErrorReporter()

    private(set) var lastError: String?
    private(set) var lastErrorTime: Date?
    var onErrorChanged: ((String?) -> Void)?

    private init() {}

    func report(_ message: String, context: String) {
        let trimmedMessage = message.trimmingCharacters(in: .whitespacesAndNewlines)
        let trimmedContext = context.trimmingCharacters(in: .whitespacesAndNewlines)
        let composed = trimmedContext.isEmpty ? trimmedMessage : "[\(trimmedContext)] \(trimmedMessage)"
        lastError = composed.isEmpty ? nil : composed
        lastErrorTime = lastError == nil ? nil : Date()
        if let lastError {
            print("[ErrorReporter] \(lastError)")
        }
        onErrorChanged?(lastError)
    }

    func clearError() {
        lastError = nil
        lastErrorTime = nil
        onErrorChanged?(nil)
    }
}
