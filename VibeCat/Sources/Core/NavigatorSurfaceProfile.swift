import Foundation

public struct NavigatorSurfaceProfile: Sendable, Equatable {
    public let kind: SurfaceKind
    public let primaryBundleID: String?
    public let appAliases: [String]
    public let preferredTextInputKeywords: [String]
    public let preparationHotkey: [String]?

    public static func detect(
        targetApp: String,
        descriptor: NavigatorTargetDescriptor = .init(),
        appName: String? = nil,
        bundleID: String? = nil
    ) -> NavigatorSurfaceProfile {
        let candidates = [
            targetApp,
            descriptor.appName ?? "",
            descriptor.label ?? "",
            descriptor.role ?? "",
            appName ?? "",
            bundleID ?? ""
        ]
            .map { normalized($0) }
            .filter { !$0.isEmpty }

        if candidates.contains(where: { value in
            value.contains("chrome") || value == "com.google.chrome"
        }) {
            let wantsAddressBarPrep = candidates.contains(where: { value in
                value.contains("address") || value.contains("search") || value.contains("url") || value.contains("browser")
            })
            return NavigatorSurfaceProfile(
                kind: .chrome,
                primaryBundleID: "com.google.Chrome",
                appAliases: ["chrome", "google chrome", "browser"],
                preferredTextInputKeywords: ["address", "search", "검색", "url"],
                preparationHotkey: wantsAddressBarPrep ? ["command", "l"] : nil
            )
        }

        if candidates.contains(where: { value in
            value.contains("terminal") || value == "com.apple.terminal"
        }) {
            return NavigatorSurfaceProfile(
                kind: .terminal,
                primaryBundleID: "com.apple.Terminal",
                appAliases: ["terminal", "terminal.app"],
                preferredTextInputKeywords: ["prompt", "shell", "command"],
                preparationHotkey: nil
            )
        }

        if candidates.contains(where: { value in
            value.contains("antigravity") || value.contains("codex") || value == "com.openai.codex"
        }) {
            return NavigatorSurfaceProfile(
                kind: .antigravity,
                primaryBundleID: "com.openai.codex",
                appAliases: ["antigravity", "antigravity ide", "codex"],
                preferredTextInputKeywords: ["prompt", "composer", "follow-up", "후속", "reply", "입력"],
                preparationHotkey: nil
            )
        }

        return NavigatorSurfaceProfile(
            kind: .unknown,
            primaryBundleID: nil,
            appAliases: [],
            preferredTextInputKeywords: [],
            preparationHotkey: nil
        )
    }

    public func matches(appName: String?) -> Bool {
        let value = Self.normalized(appName ?? "")
        guard !value.isEmpty else { return false }
        return appAliases.contains(where: { alias in
            value.contains(alias) || alias.contains(value)
        })
    }

    public func matches(bundleID: String?) -> Bool {
        guard let primaryBundleID else { return false }
        return Self.normalized(bundleID ?? "") == Self.normalized(primaryBundleID)
    }

    public func preferredPreparationHotkey(for actionType: NavigatorActionType) -> [String]? {
        switch kind {
        case .chrome:
            if actionType == .pasteText || actionType == .pressAX {
                return preparationHotkey
            }
            return nil
        case .terminal, .antigravity, .unknown:
            return nil
        }
    }

    private static func normalized(_ raw: String) -> String {
        raw.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
    }
}
