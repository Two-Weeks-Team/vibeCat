import Foundation

public enum NavigatorIntentClass: String, Sendable, Codable {
    case executeNow = "execute_now"
    case openOrNavigate = "open_or_navigate"
    case findOrLookup = "find_or_lookup"
    case analyzeOnly = "analyze_only"
    case ambiguous
}

public enum NavigatorActionType: String, Sendable, Codable {
    case focusApp = "focus_app"
    case openURL = "open_url"
    case hotkey
    case pasteText = "paste_text"
    case copySelection = "copy_selection"
    case pressAX = "press_ax"
    case systemAction = "system_action"
    case waitFor = "wait_for"
}

public enum SurfaceKind: String, Sendable, Codable {
    case antigravity
    case chrome
    case terminal
    case unknown
}

public enum ProofLevel: String, Sendable, Codable {
    case basic
    case strong
    case strict
}

public enum ExecutionFailureReason: String, Sendable, Codable {
    case focusNotReady = "focus_not_ready"
    case wrongTarget = "wrong_target"
    case targetNotFound = "target_not_found"
    case targetNotWritable = "target_not_writable"
    case verificationInconclusive = "verification_inconclusive"
    case pasteRejected = "paste_rejected"
    case surfaceAdapterUnavailable = "surface_adapter_unavailable"
}

public enum ExecutionPhase: String, Sendable, Codable {
    case preflight
    case resolveTarget = "resolve_target"
    case activateTarget = "activate_target"
    case proveTargetReady = "prove_target_ready"
    case performAction = "perform_action"
    case readBack = "read_back"
    case verifyOutcome = "verify_outcome"
    case recoverOrFail = "recover_or_fail"
}

public enum NavigatorClarificationResponseMode: String, Sendable, Codable {
    case confirmation
    case provideDetails = "provide_details"
}

public struct NavigatorTargetDescriptor: Sendable, Codable, Equatable {
    public let role: String?
    public let label: String?
    public let windowTitle: String?
    public let appName: String?
    public let relativeAnchor: String?
    public let regionHint: String?

    public init(
        role: String? = nil,
        label: String? = nil,
        windowTitle: String? = nil,
        appName: String? = nil,
        relativeAnchor: String? = nil,
        regionHint: String? = nil
    ) {
        self.role = role
        self.label = label
        self.windowTitle = windowTitle
        self.appName = appName
        self.relativeAnchor = relativeAnchor
        self.regionHint = regionHint
    }
}

public struct VerifyContract: Sendable, Codable, Equatable {
    public let expectedBundleId: String?
    public let expectedWindowContains: String?
    public let expectedFocusedRole: String?
    public let expectedFocusedLabel: String?
    public let expectedAXContains: String?
    public let expectedSelectedTextPrefix: String?
    public let requireWritableTarget: Bool?
    public let requireFrontmostApp: Bool?
    public let minCaptureConfidenceAfter: Double?
    public let proofStrategy: String?

    public init(
        expectedBundleId: String? = nil,
        expectedWindowContains: String? = nil,
        expectedFocusedRole: String? = nil,
        expectedFocusedLabel: String? = nil,
        expectedAXContains: String? = nil,
        expectedSelectedTextPrefix: String? = nil,
        requireWritableTarget: Bool? = nil,
        requireFrontmostApp: Bool? = nil,
        minCaptureConfidenceAfter: Double? = nil,
        proofStrategy: String? = nil
    ) {
        self.expectedBundleId = expectedBundleId
        self.expectedWindowContains = expectedWindowContains
        self.expectedFocusedRole = expectedFocusedRole
        self.expectedFocusedLabel = expectedFocusedLabel
        self.expectedAXContains = expectedAXContains
        self.expectedSelectedTextPrefix = expectedSelectedTextPrefix
        self.requireWritableTarget = requireWritableTarget
        self.requireFrontmostApp = requireFrontmostApp
        self.minCaptureConfidenceAfter = minCaptureConfidenceAfter
        self.proofStrategy = proofStrategy
    }
}

public struct NavigatorStep: Sendable, Codable, Equatable, Identifiable {
    public let id: String
    public let actionType: NavigatorActionType
    public let targetApp: String
    public let targetDescriptor: NavigatorTargetDescriptor
    public let inputText: String?
    public let expectedOutcome: String
    public let confidence: Double
    public let intentConfidence: Double
    public let riskLevel: String
    public let executionPolicy: String
    public let fallbackPolicy: String
    public let url: String?
    public let hotkey: [String]
    public let verifyHint: String?
    public let systemCommand: String?
    public let systemValue: String?
    public let systemAmount: Int
    public let surface: SurfaceKind?
    public let macroID: String?
    public let narration: String?
    public let verifyContract: VerifyContract?
    public let fallbackActionType: NavigatorActionType?
    public let fallbackHotkey: [String]?
    public let maxLocalRetries: Int?
    public let timeoutMs: Int?
    public let proofLevel: ProofLevel?

    public init(
        id: String,
        actionType: NavigatorActionType,
        targetApp: String,
        targetDescriptor: NavigatorTargetDescriptor = .init(),
        inputText: String? = nil,
        expectedOutcome: String,
        confidence: Double,
        intentConfidence: Double,
        riskLevel: String,
        executionPolicy: String,
        fallbackPolicy: String,
        url: String? = nil,
        hotkey: [String] = [],
        verifyHint: String? = nil,
        systemCommand: String? = nil,
        systemValue: String? = nil,
        systemAmount: Int = 0,
        surface: SurfaceKind? = nil,
        macroID: String? = nil,
        narration: String? = nil,
        verifyContract: VerifyContract? = nil,
        fallbackActionType: NavigatorActionType? = nil,
        fallbackHotkey: [String]? = nil,
        maxLocalRetries: Int? = nil,
        timeoutMs: Int? = nil,
        proofLevel: ProofLevel? = nil
    ) {
        self.id = id
        self.actionType = actionType
        self.targetApp = targetApp
        self.targetDescriptor = targetDescriptor
        self.inputText = inputText
        self.expectedOutcome = expectedOutcome
        self.confidence = confidence
        self.intentConfidence = intentConfidence
        self.riskLevel = riskLevel
        self.executionPolicy = executionPolicy
        self.fallbackPolicy = fallbackPolicy
        self.url = url
        self.hotkey = hotkey
        self.verifyHint = verifyHint
        self.systemCommand = systemCommand
        self.systemValue = systemValue
        self.systemAmount = systemAmount
        self.surface = surface
        self.macroID = macroID
        self.narration = narration
        self.verifyContract = verifyContract
        self.fallbackActionType = fallbackActionType
        self.fallbackHotkey = fallbackHotkey
        self.maxLocalRetries = maxLocalRetries
        self.timeoutMs = timeoutMs
        self.proofLevel = proofLevel
    }
}

public enum GroundingSource: String, Sendable {
    case ax
    case vision
    case hotkey
    case system

    public var badge: String {
        switch self {
        case .ax: return "AX"
        case .vision: return "Vision"
        case .hotkey: return "Hotkey"
        case .system: return "System"
        }
    }
}

extension NavigatorActionType {
    public func groundingSource(step: NavigatorStep) -> GroundingSource {
        switch self {
        case .pressAX, .pasteText, .copySelection:
            return .ax
        case .hotkey:
            return .hotkey
        case .focusApp, .openURL:
            return .system
        case .systemAction:
            return .system
        case .waitFor:
            return step.targetDescriptor.label != nil ? .ax : .system
        }
    }

    public var icon: String {
        switch self {
        case .focusApp: return "\u{1F3AF}"
        case .hotkey: return "\u{2328}\u{FE0F}"
        case .pasteText: return "\u{2328}\u{FE0F}"
        case .openURL: return "\u{1F310}"
        case .pressAX: return "\u{1F446}"
        case .copySelection: return "\u{1F4CB}"
        case .systemAction: return "\u{2699}\u{FE0F}"
        case .waitFor: return "\u{23F3}"
        }
    }
}

public struct NavigatorContextPayload: Sendable, Codable, Equatable {
    public let appName: String
    public let bundleId: String
    public let frontmostBundleId: String
    public let windowTitle: String
    public let focusedRole: String
    public let focusedLabel: String
    public let selectedText: String
    public let axSnapshot: String
    public let inputFieldHint: String
    public let lastInputFieldDescriptor: String
    public let screenshot: String
    public let focusStableMs: Int
    public let captureConfidence: Double
    public let visibleInputCandidateCount: Int
    public let accessibilityPermission: String
    public let accessibilityTrusted: Bool
    public let activeDisplayID: String
    public let targetDisplayID: String
    public let screenshotAgeMs: Int
    public let screenshotSource: String
    public let screenshotCached: Bool
    public let screenBasisID: String

    public init(
        appName: String,
        bundleId: String,
        frontmostBundleId: String,
        windowTitle: String,
        focusedRole: String,
        focusedLabel: String,
        selectedText: String,
        axSnapshot: String,
        inputFieldHint: String,
        lastInputFieldDescriptor: String,
        screenshot: String,
        focusStableMs: Int,
        captureConfidence: Double,
        visibleInputCandidateCount: Int,
        accessibilityPermission: String,
        accessibilityTrusted: Bool,
        activeDisplayID: String = "",
        targetDisplayID: String = "",
        screenshotAgeMs: Int = 0,
        screenshotSource: String = "",
        screenshotCached: Bool = false,
        screenBasisID: String = ""
    ) {
        self.appName = appName
        self.bundleId = bundleId
        self.frontmostBundleId = frontmostBundleId
        self.windowTitle = windowTitle
        self.focusedRole = focusedRole
        self.focusedLabel = focusedLabel
        self.selectedText = selectedText
        self.axSnapshot = axSnapshot
        self.inputFieldHint = inputFieldHint
        self.lastInputFieldDescriptor = lastInputFieldDescriptor
        self.screenshot = screenshot
        self.focusStableMs = focusStableMs
        self.captureConfidence = captureConfidence
        self.visibleInputCandidateCount = visibleInputCandidateCount
        self.accessibilityPermission = accessibilityPermission
        self.accessibilityTrusted = accessibilityTrusted
        self.activeDisplayID = activeDisplayID
        self.targetDisplayID = targetDisplayID
        self.screenshotAgeMs = screenshotAgeMs
        self.screenshotSource = screenshotSource
        self.screenshotCached = screenshotCached
        self.screenBasisID = screenBasisID
    }

    public func withScreenshot(_ screenshot: String) -> NavigatorContextPayload {
        NavigatorContextPayload(
            appName: appName,
            bundleId: bundleId,
            frontmostBundleId: frontmostBundleId,
            windowTitle: windowTitle,
            focusedRole: focusedRole,
            focusedLabel: focusedLabel,
            selectedText: selectedText,
            axSnapshot: axSnapshot,
            inputFieldHint: inputFieldHint,
            lastInputFieldDescriptor: lastInputFieldDescriptor,
            screenshot: screenshot,
            focusStableMs: focusStableMs,
            captureConfidence: captureConfidence,
            visibleInputCandidateCount: visibleInputCandidateCount,
            accessibilityPermission: accessibilityPermission,
            accessibilityTrusted: accessibilityTrusted,
            activeDisplayID: activeDisplayID,
            targetDisplayID: targetDisplayID,
            screenshotAgeMs: screenshotAgeMs,
            screenshotSource: screenshotSource,
            screenshotCached: screenshotCached,
            screenBasisID: screenBasisID
        )
    }

    public func withScreenBasis(
        screenBasisID: String,
        activeDisplayID: String,
        targetDisplayID: String,
        screenshotAgeMs: Int,
        screenshotSource: String,
        screenshotCached: Bool,
        screenshot: String? = nil
    ) -> NavigatorContextPayload {
        NavigatorContextPayload(
            appName: appName,
            bundleId: bundleId,
            frontmostBundleId: frontmostBundleId,
            windowTitle: windowTitle,
            focusedRole: focusedRole,
            focusedLabel: focusedLabel,
            selectedText: selectedText,
            axSnapshot: axSnapshot,
            inputFieldHint: inputFieldHint,
            lastInputFieldDescriptor: lastInputFieldDescriptor,
            screenshot: screenshot ?? self.screenshot,
            focusStableMs: focusStableMs,
            captureConfidence: captureConfidence,
            visibleInputCandidateCount: visibleInputCandidateCount,
            accessibilityPermission: accessibilityPermission,
            accessibilityTrusted: accessibilityTrusted,
            activeDisplayID: activeDisplayID,
            targetDisplayID: targetDisplayID,
            screenshotAgeMs: screenshotAgeMs,
            screenshotSource: screenshotSource,
            screenshotCached: screenshotCached,
            screenBasisID: screenBasisID
        )
    }

    public func withCachedScreenshot(
        _ screenshot: String,
        activeDisplayID: String,
        targetDisplayID: String,
        screenshotAgeMs: Int,
        screenshotSource: String,
        screenshotCached: Bool
    ) -> NavigatorContextPayload {
        NavigatorContextPayload(
            appName: appName,
            bundleId: bundleId,
            frontmostBundleId: frontmostBundleId,
            windowTitle: windowTitle,
            focusedRole: focusedRole,
            focusedLabel: focusedLabel,
            selectedText: selectedText,
            axSnapshot: axSnapshot,
            inputFieldHint: inputFieldHint,
            lastInputFieldDescriptor: lastInputFieldDescriptor,
            screenshot: screenshot,
            focusStableMs: focusStableMs,
            captureConfidence: captureConfidence,
            visibleInputCandidateCount: visibleInputCandidateCount,
            accessibilityPermission: accessibilityPermission,
            accessibilityTrusted: accessibilityTrusted,
            activeDisplayID: activeDisplayID,
            targetDisplayID: targetDisplayID,
            screenshotAgeMs: screenshotAgeMs,
            screenshotSource: screenshotSource,
            screenshotCached: screenshotCached,
            screenBasisID: screenBasisID
        )
    }
}
