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

        currentText = mergedTranscript(current: currentText, incoming: text)
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

    private func mergedTranscript(current: String, incoming: String) -> String {
        guard !current.isEmpty else { return incoming }
        guard !incoming.isEmpty else { return current }

        if current == incoming {
            return current
        }
        if incoming.hasPrefix(current) {
            return incoming
        }
        if current.hasPrefix(incoming) {
            return current
        }
        if current.contains(incoming) {
            return current
        }
        if incoming.contains(current) {
            return incoming
        }

        let overlap = longestSuffixPrefixOverlap(current, incoming)
        if overlap > 0 {
            return current + incoming.dropFirst(overlap)
        }
        return current + incoming
    }

    private func longestSuffixPrefixOverlap(_ current: String, _ incoming: String) -> Int {
        let currentChars = Array(current)
        let incomingChars = Array(incoming)
        let maxOverlap = min(currentChars.count, incomingChars.count)
        guard maxOverlap > 0 else { return 0 }

        for overlap in stride(from: maxOverlap, through: 1, by: -1) {
            if Array(currentChars.suffix(overlap)) == Array(incomingChars.prefix(overlap)) {
                return overlap
            }
        }
        return 0
    }
}
