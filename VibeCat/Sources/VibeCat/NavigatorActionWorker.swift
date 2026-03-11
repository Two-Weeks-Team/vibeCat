import Foundation
import VibeCatCore

@MainActor
final class NavigatorActionWorker {
    private let gatewayClient: GatewayClient
    private let navigator: AccessibilityNavigator
    private let contextProvider: @MainActor () -> NavigatorContextPayload

    private var executionTask: Task<Void, Never>?

    private(set) var activeTaskID: String?
    private(set) var activeCommand: String?
    private(set) var currentStepID: String?

    init(
        gatewayClient: GatewayClient,
        navigator: AccessibilityNavigator,
        contextProvider: @escaping @MainActor () -> NavigatorContextPayload
    ) {
        self.gatewayClient = gatewayClient
        self.navigator = navigator
        self.contextProvider = contextProvider
    }

    func beginTask(taskId: String?, command: String) {
        let normalizedTaskID = normalized(taskId)
        let normalizedCommand = normalized(command)

        if activeTaskID != normalizedTaskID {
            executionTask?.cancel()
            currentStepID = nil
        }

        activeTaskID = normalizedTaskID
        activeCommand = normalizedCommand
    }

    func clearTask(taskId: String? = nil) {
        let normalizedTaskID = normalized(taskId)
        if normalizedTaskID != nil && normalizedTaskID != activeTaskID {
            return
        }
        executionTask?.cancel()
        executionTask = nil
        activeTaskID = nil
        activeCommand = nil
        currentStepID = nil
    }

    func execute(
        taskId: String,
        step: NavigatorStep,
        onResult: @escaping @MainActor (NavigatorExecutionResult) -> Void
    ) {
        guard normalized(taskId) == activeTaskID, let command = activeCommand else {
            return
        }

        currentStepID = step.id
        executionTask?.cancel()
        executionTask = Task { [weak self] in
            guard let self else { return }
            let result = await self.navigator.execute(step: step)
            guard !Task.isCancelled else { return }
            await MainActor.run {
                guard self.activeTaskID == self.normalized(taskId),
                      self.currentStepID == step.id else {
                    return
                }
                self.gatewayClient.sendNavigatorRefresh(
                    taskId: taskId,
                    command: command,
                    step: step,
                    status: result.status,
                    observedOutcome: result.observedOutcome,
                    context: self.contextProvider()
                )
                if result.status != "success" {
                    self.currentStepID = nil
                }
                onResult(result)
            }
        }
    }

    private func normalized(_ raw: String?) -> String? {
        let trimmed = raw?.trimmingCharacters(in: .whitespacesAndNewlines)
        guard let trimmed, !trimmed.isEmpty else {
            return nil
        }
        return trimmed
    }
}
