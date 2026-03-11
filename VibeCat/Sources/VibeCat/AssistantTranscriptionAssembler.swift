import Foundation

struct AssistantTranscriptionAssembler {
    private(set) var currentText = ""
    private let mergeWindow: TimeInterval
    private var finalizationDeadline: Date?

    init(mergeWindow: TimeInterval = 1.25) {
        self.mergeWindow = mergeWindow
    }

    var hasPendingFinalization: Bool {
        finalizationDeadline != nil
    }

    var scheduledFinalizationDeadline: Date? {
        finalizationDeadline
    }

    mutating func ingest(_ text: String, now: Date = Date()) -> String {
        guard !text.isEmpty else { return currentText }

        if let deadline = finalizationDeadline {
            if now >= deadline {
                currentText = ""
                finalizationDeadline = nil
            } else {
                finalizationDeadline = now.addingTimeInterval(mergeWindow)
            }
        }

        currentText += text
        return currentText
    }

    @discardableResult
    mutating func markBoundary(now: Date = Date()) -> Bool {
        let trimmed = currentText.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            finalizationDeadline = nil
            return false
        }

        finalizationDeadline = now.addingTimeInterval(mergeWindow)
        return true
    }

    func remainingFinalizationDelay(now: Date = Date()) -> TimeInterval? {
        guard let deadline = finalizationDeadline else { return nil }
        return max(0, deadline.timeIntervalSince(now))
    }

    mutating func finalizeIfDue(now: Date = Date()) -> String? {
        guard let deadline = finalizationDeadline, now >= deadline else { return nil }
        return finalizeNow()
    }

    mutating func finalizeNow() -> String? {
        let finalized = currentText.trimmingCharacters(in: .whitespacesAndNewlines)
        currentText = ""
        finalizationDeadline = nil
        guard !finalized.isEmpty else { return nil }
        return finalized
    }

    mutating func discard() {
        currentText = ""
        finalizationDeadline = nil
    }
}
