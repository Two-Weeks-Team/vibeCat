package ws

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type navigatorContext struct {
	AppName                 string  `json:"appName"`
	BundleID                string  `json:"bundleId"`
	FrontmostBundleID       string  `json:"frontmostBundleId"`
	WindowTitle             string  `json:"windowTitle"`
	FocusedRole             string  `json:"focusedRole"`
	FocusedLabel            string  `json:"focusedLabel"`
	SelectedText            string  `json:"selectedText"`
	AXSnapshot              string  `json:"axSnapshot"`
	InputFieldHint          string  `json:"inputFieldHint"`
	LastInputDescriptor     string  `json:"lastInputFieldDescriptor"`
	Screenshot              string  `json:"screenshot,omitempty"`
	FocusStableMs           int     `json:"focusStableMs"`
	CaptureConfidence       float64 `json:"captureConfidence"`
	VisibleInputCandidates  int     `json:"visibleInputCandidateCount"`
	AccessibilityPermission string  `json:"accessibilityPermission"`
	AccessibilityTrusted    bool    `json:"accessibilityTrusted"`
	ActiveDisplayID         string  `json:"activeDisplayID,omitempty"`
	TargetDisplayID         string  `json:"targetDisplayID,omitempty"`
	ScreenshotAgeMs         int     `json:"screenshotAgeMs,omitempty"`
	ScreenshotSource        string  `json:"screenshotSource,omitempty"`
	ScreenshotCached        bool    `json:"screenshotCached,omitempty"`
	ScreenBasisID           string  `json:"screenBasisId,omitempty"`
}

type navigatorContextSnapshot struct {
	AppName          string `json:"appName,omitempty"`
	BundleID         string `json:"bundleId,omitempty"`
	WindowTitle      string `json:"windowTitle,omitempty"`
	FocusedRole      string `json:"focusedRole,omitempty"`
	FocusedLabel     string `json:"focusedLabel,omitempty"`
	SelectedTextHash string `json:"selectedTextHash,omitempty"`
	AXSnapshotHash   string `json:"axSnapshotHash,omitempty"`
}

type navigatorTargetDescriptor struct {
	Role            string  `json:"role,omitempty"`
	Label           string  `json:"label,omitempty"`
	WindowTitle     string  `json:"windowTitle,omitempty"`
	AppName         string  `json:"appName,omitempty"`
	RelativeAnchor  string  `json:"relativeAnchor,omitempty"`
	RegionHint      string  `json:"regionHint,omitempty"`
	ClickX          float64 `json:"clickX,omitempty"`
	ClickY          float64 `json:"clickY,omitempty"`
	VerificationCue string  `json:"verificationCue,omitempty"`
}

type navigatorStep struct {
	ID                 string                    `json:"id"`
	ActionType         string                    `json:"actionType"`
	TargetApp          string                    `json:"targetApp"`
	TargetDescriptor   navigatorTargetDescriptor `json:"targetDescriptor"`
	InputText          string                    `json:"inputText,omitempty"`
	ExpectedOutcome    string                    `json:"expectedOutcome"`
	Confidence         float64                   `json:"confidence"`
	IntentConfidence   float64                   `json:"intentConfidence"`
	RiskLevel          string                    `json:"riskLevel"`
	ExecutionPolicy    string                    `json:"executionPolicy"`
	FallbackPolicy     string                    `json:"fallbackPolicy"`
	URL                string                    `json:"url,omitempty"`
	Hotkey             []string                  `json:"hotkey,omitempty"`
	VerifyHint         string                    `json:"verifyHint,omitempty"`
	SystemCommand      string                    `json:"systemCommand,omitempty"`
	SystemValue        string                    `json:"systemValue,omitempty"`
	SystemAmount       int                       `json:"systemAmount,omitempty"`
	Surface            string                    `json:"surface,omitempty"`
	MacroID            string                    `json:"macroID,omitempty"`
	Narration          string                    `json:"narration,omitempty"`
	VerifyContract     *navigatorVerifyContract  `json:"verifyContract,omitempty"`
	FallbackActionType string                    `json:"fallbackActionType,omitempty"`
	FallbackHotkey     []string                  `json:"fallbackHotkey,omitempty"`
	CDPScript          string                    `json:"cdpScript,omitempty"`
	MaxLocalRetries    int                       `json:"maxLocalRetries,omitempty"`
	TimeoutMs          int                       `json:"timeoutMs,omitempty"`
	ProofLevel         string                    `json:"proofLevel,omitempty"`
}

type navigatorVerifyContract struct {
	ExpectedBundleID          string  `json:"expectedBundleId,omitempty"`
	ExpectedWindowContains    string  `json:"expectedWindowContains,omitempty"`
	ExpectedFocusedRole       string  `json:"expectedFocusedRole,omitempty"`
	ExpectedFocusedLabel      string  `json:"expectedFocusedLabel,omitempty"`
	ExpectedAXContains        string  `json:"expectedAXContains,omitempty"`
	ExpectedSelectedPrefix    string  `json:"expectedSelectedTextPrefix,omitempty"`
	RequireWritableTarget     bool    `json:"requireWritableTarget,omitempty"`
	RequireFrontmostApp       bool    `json:"requireFrontmostApp,omitempty"`
	MinCaptureConfidenceAfter float64 `json:"minCaptureConfidenceAfter,omitempty"`
	ProofStrategy             string  `json:"proofStrategy,omitempty"`
}

type navigatorIntentClass string

const (
	navigatorIntentExecuteNow   navigatorIntentClass = "execute_now"
	navigatorIntentOpenNavigate navigatorIntentClass = "open_or_navigate"
	navigatorIntentFindLookup   navigatorIntentClass = "find_or_lookup"
	navigatorIntentAnalyzeOnly  navigatorIntentClass = "analyze_only"
	navigatorIntentAmbiguous    navigatorIntentClass = "ambiguous"
	navigatorExecutionPolicyLow                      = "safe_immediate"
)

type navigatorPlan struct {
	Command          string
	IntentClass      navigatorIntentClass
	IntentConfidence float64
	ClarifyQuestion  string
	ClarifyMode      navigatorClarificationResponseMode
	RiskQuestion     string
	RiskReason       string
	Steps            []navigatorStep
}

type navigatorStepTrace struct {
	ID               string                    `json:"id"`
	ActionType       string                    `json:"actionType"`
	TargetApp        string                    `json:"targetApp,omitempty"`
	TargetDescriptor navigatorTargetDescriptor `json:"targetDescriptor,omitempty"`
	PlannedAt        time.Time                 `json:"plannedAt,omitempty"`
	ResultStatus     string                    `json:"resultStatus,omitempty"`
	ObservedOutcome  string                    `json:"observedOutcome,omitempty"`
	CompletedAt      time.Time                 `json:"completedAt,omitempty"`
}

type navigatorAttemptTrace struct {
	ID               string    `json:"id"`
	TaskID           string    `json:"taskId,omitempty"`
	Command          string    `json:"command"`
	Surface          string    `json:"surface,omitempty"`
	Route            string    `json:"route"`
	RouteReason      string    `json:"routeReason,omitempty"`
	ContextHash      string    `json:"contextHash,omitempty"`
	ScreenshotSource string    `json:"screenshotSource,omitempty"`
	ScreenshotCached bool      `json:"screenshotCached,omitempty"`
	ScreenBasisID    string    `json:"screenBasisId,omitempty"`
	ActiveDisplayID  string    `json:"activeDisplayId,omitempty"`
	TargetDisplayID  string    `json:"targetDisplayId,omitempty"`
	Outcome          string    `json:"outcome,omitempty"`
	OutcomeDetail    string    `json:"outcomeDetail,omitempty"`
	StartedAt        time.Time `json:"startedAt,omitempty"`
	CompletedAt      time.Time `json:"completedAt,omitempty"`
}

type navigatorPromptKind string

type navigatorClarificationResponseMode string

const (
	navigatorPromptConfirmIntent navigatorPromptKind = "intent_confirmation"
	navigatorPromptProvideDetail navigatorPromptKind = "provide_details"
	navigatorPromptReplace       navigatorPromptKind = "replace_task"

	navigatorClarificationConfirm       navigatorClarificationResponseMode = "confirmation"
	navigatorClarificationProvideDetail navigatorClarificationResponseMode = "provide_details"
)

type navigatorSessionState struct {
	activeTaskID                string
	activeCommand               string
	pendingClarificationKind    navigatorPromptKind
	pendingClarificationCommand string
	pendingRiskyCommand         string
	steps                       []navigatorStep
	nextStepIndex               int
	currentStepID               string
	stepRetryCount              int
	deviceID                    string
	connectionID                string
	initialContext              navigatorContextSnapshot
	initialContextHash          string
	initialAppName              string
	initialWindowTitle          string
	stepHistory                 []navigatorStepTrace
	attemptHistory              []navigatorAttemptTrace
	currentAttemptID            string
	lastVerifiedContextHash     string
	createdAt                   time.Time
	updatedAt                   time.Time
}

func (s *navigatorSessionState) startPlan(command string, steps []navigatorStep) string {
	s.activeTaskID = newNavigatorTaskID()
	s.activeCommand = command
	s.pendingClarificationKind = ""
	s.pendingClarificationCommand = ""
	s.pendingRiskyCommand = ""
	s.steps = steps
	s.nextStepIndex = 0
	s.currentStepID = ""
	s.stepRetryCount = 0
	s.initialContext = navigatorContextSnapshot{}
	s.initialContextHash = ""
	s.initialAppName = ""
	s.initialWindowTitle = ""
	s.stepHistory = nil
	s.attemptHistory = nil
	s.currentAttemptID = ""
	s.lastVerifiedContextHash = ""
	now := time.Now().UTC()
	s.createdAt = now
	s.updatedAt = now
	return s.activeTaskID
}

func (s *navigatorSessionState) clearPlan() {
	s.activeTaskID = ""
	s.activeCommand = ""
	s.pendingClarificationKind = ""
	s.pendingClarificationCommand = ""
	s.pendingRiskyCommand = ""
	s.steps = nil
	s.nextStepIndex = 0
	s.currentStepID = ""
	s.stepRetryCount = 0
	s.initialContext = navigatorContextSnapshot{}
	s.initialContextHash = ""
	s.initialAppName = ""
	s.initialWindowTitle = ""
	s.stepHistory = nil
	s.attemptHistory = nil
	s.currentAttemptID = ""
	s.lastVerifiedContextHash = ""
	s.createdAt = time.Time{}
	s.updatedAt = time.Time{}
}

func (s *navigatorSessionState) incrementStepRetry() int {
	s.stepRetryCount++
	s.touch()
	return s.stepRetryCount
}

func (s *navigatorSessionState) resetStepRetry() {
	s.stepRetryCount = 0
	s.touch()
}

func (s *navigatorSessionState) nextStep() (navigatorStep, bool) {
	if s.nextStepIndex >= len(s.steps) {
		return navigatorStep{}, false
	}
	step := s.steps[s.nextStepIndex]
	s.recordPlannedStep(step)
	s.nextStepIndex++
	s.currentStepID = step.ID
	s.touch()
	return step, true
}

func (s *navigatorSessionState) hasRemainingSteps() bool {
	return s.nextStepIndex < len(s.steps)
}

func (s *navigatorSessionState) hasActiveTask() bool {
	return strings.TrimSpace(s.activeTaskID) != ""
}

const staleTaskThreshold = 60 * time.Second

func (s *navigatorSessionState) isStaleTask() bool {
	if !s.hasActiveTask() {
		return false
	}
	if s.updatedAt.IsZero() {
		return !s.createdAt.IsZero() && time.Since(s.createdAt) > staleTaskThreshold
	}
	return time.Since(s.updatedAt) > staleTaskThreshold
}

func (s *navigatorSessionState) activeTaskSnapshot() (string, string, string, bool) {
	if strings.TrimSpace(s.activeTaskID) == "" {
		return "", "", "", false
	}
	return s.activeTaskID, s.activeCommand, s.currentStepID, true
}

func (s *navigatorSessionState) stageClarification(kind navigatorPromptKind, command string) {
	s.pendingClarificationKind = kind
	s.pendingClarificationCommand = strings.TrimSpace(command)
	s.touch()
}

func (s *navigatorSessionState) consumeClarification(command string) (navigatorPromptKind, string) {
	kind := s.pendingClarificationKind
	pending := strings.TrimSpace(s.pendingClarificationCommand)
	s.pendingClarificationKind = ""
	s.pendingClarificationCommand = ""
	s.touch()
	if strings.TrimSpace(command) == "" {
		command = pending
	}
	return kind, strings.TrimSpace(command)
}

func (s *navigatorSessionState) acceptsRefresh(command, taskID, stepID string) bool {
	stepID = strings.TrimSpace(stepID)
	if stepID == "" || stepID != s.currentStepID {
		return false
	}
	taskID = strings.TrimSpace(taskID)
	if strings.TrimSpace(s.activeTaskID) == "" || taskID == "" || taskID != s.activeTaskID {
		return false
	}
	command = strings.TrimSpace(command)
	if command == "" || strings.TrimSpace(s.activeCommand) == "" {
		return true
	}
	return command == s.activeCommand
}

func (s *navigatorSessionState) clearCurrentStep() {
	s.currentStepID = ""
	s.touch()
}

func (s *navigatorSessionState) bindLease(deviceID, connectionID string) {
	if trimmed := strings.TrimSpace(deviceID); trimmed != "" {
		s.deviceID = trimmed
	}
	if trimmed := strings.TrimSpace(connectionID); trimmed != "" {
		s.connectionID = trimmed
	}
	s.touch()
}

func (s *navigatorSessionState) ownsLease(connectionID string) bool {
	lease := strings.TrimSpace(s.connectionID)
	if lease == "" {
		return true
	}
	return lease == strings.TrimSpace(connectionID)
}

func (s *navigatorSessionState) hasPersistableState() bool {
	return s.hasActiveTask() ||
		strings.TrimSpace(s.pendingClarificationCommand) != "" ||
		strings.TrimSpace(s.pendingRiskyCommand) != ""
}

func (s *navigatorSessionState) rememberInitialContext(ctx navigatorContext) {
	if !s.hasActiveTask() {
		return
	}
	if s.initialContextHash == "" {
		s.initialContext = snapshotNavigatorContext(ctx)
		s.initialContextHash = navigatorContextHash(ctx)
		s.initialAppName = strings.TrimSpace(ctx.AppName)
		s.initialWindowTitle = strings.TrimSpace(ctx.WindowTitle)
	}
	s.touch()
}

func (s *navigatorSessionState) markVerifiedContext(ctx navigatorContext) {
	s.lastVerifiedContextHash = navigatorContextHash(ctx)
	s.touch()
}

func (s *navigatorSessionState) recordPlannedStep(step navigatorStep) {
	trace := navigatorStepTrace{
		ID:               step.ID,
		ActionType:       step.ActionType,
		TargetApp:        step.TargetApp,
		TargetDescriptor: step.TargetDescriptor,
		PlannedAt:        time.Now().UTC(),
	}
	for idx := range s.stepHistory {
		if s.stepHistory[idx].ID != step.ID {
			continue
		}
		s.stepHistory[idx].ActionType = step.ActionType
		s.stepHistory[idx].TargetApp = step.TargetApp
		s.stepHistory[idx].TargetDescriptor = step.TargetDescriptor
		if s.stepHistory[idx].PlannedAt.IsZero() {
			s.stepHistory[idx].PlannedAt = trace.PlannedAt
		}
		return
	}
	s.stepHistory = append(s.stepHistory, trace)
}

func (s *navigatorSessionState) recordStepResult(step navigatorStep, status, observedOutcome string) {
	for idx := range s.stepHistory {
		if s.stepHistory[idx].ID != step.ID {
			continue
		}
		s.stepHistory[idx].ResultStatus = strings.TrimSpace(status)
		s.stepHistory[idx].ObservedOutcome = strings.TrimSpace(observedOutcome)
		s.stepHistory[idx].CompletedAt = time.Now().UTC()
		s.touch()
		return
	}
	s.stepHistory = append(s.stepHistory, navigatorStepTrace{
		ID:               step.ID,
		ActionType:       step.ActionType,
		TargetApp:        step.TargetApp,
		TargetDescriptor: step.TargetDescriptor,
		ResultStatus:     strings.TrimSpace(status),
		ObservedOutcome:  strings.TrimSpace(observedOutcome),
		CompletedAt:      time.Now().UTC(),
	})
	s.touch()
}

func (s *navigatorSessionState) beginAttempt(command string, ctx navigatorContext, route, routeReason string) string {
	attemptID := newTraceID("attempt")
	s.currentAttemptID = attemptID
	s.attemptHistory = append(s.attemptHistory, navigatorAttemptTrace{
		ID:               attemptID,
		Command:          strings.TrimSpace(command),
		Surface:          navigatorSurfaceFromContext(ctx),
		Route:            strings.TrimSpace(route),
		RouteReason:      strings.TrimSpace(routeReason),
		ContextHash:      navigatorContextHash(ctx),
		ScreenshotSource: strings.TrimSpace(ctx.ScreenshotSource),
		ScreenshotCached: ctx.ScreenshotCached,
		ScreenBasisID:    strings.TrimSpace(ctx.ScreenBasisID),
		ActiveDisplayID:  strings.TrimSpace(ctx.ActiveDisplayID),
		TargetDisplayID:  strings.TrimSpace(ctx.TargetDisplayID),
		StartedAt:        time.Now().UTC(),
	})
	s.touch()
	return attemptID
}

func (s *navigatorSessionState) attachAttemptTask(taskID string) {
	if strings.TrimSpace(s.currentAttemptID) == "" {
		return
	}
	for idx := len(s.attemptHistory) - 1; idx >= 0; idx-- {
		if s.attemptHistory[idx].ID != s.currentAttemptID {
			continue
		}
		s.attemptHistory[idx].TaskID = strings.TrimSpace(taskID)
		s.touch()
		return
	}
}

func (s *navigatorSessionState) completeAttempt(outcome, detail string) {
	if strings.TrimSpace(s.currentAttemptID) == "" {
		return
	}
	for idx := len(s.attemptHistory) - 1; idx >= 0; idx-- {
		if s.attemptHistory[idx].ID != s.currentAttemptID {
			continue
		}
		s.attemptHistory[idx].Outcome = strings.TrimSpace(outcome)
		if strings.TrimSpace(detail) != "" {
			s.attemptHistory[idx].OutcomeDetail = strings.TrimSpace(detail)
		}
		s.attemptHistory[idx].CompletedAt = time.Now().UTC()
		s.currentAttemptID = ""
		s.touch()
		return
	}
	currentAttemptID := s.currentAttemptID
	s.attemptHistory = append(s.attemptHistory, navigatorAttemptTrace{
		ID:            currentAttemptID,
		Outcome:       strings.TrimSpace(outcome),
		OutcomeDetail: strings.TrimSpace(detail),
		CompletedAt:   time.Now().UTC(),
	})
	s.currentAttemptID = ""
	s.touch()
}

func (s *navigatorSessionState) firstPlannedStep() (navigatorStepTrace, bool) {
	if len(s.stepHistory) == 0 {
		return navigatorStepTrace{}, false
	}
	first := s.stepHistory[0]
	if first.PlannedAt.IsZero() {
		return navigatorStepTrace{}, false
	}
	return first, true
}

type navigatorTaskSnapshot struct {
	TaskID                  string
	Command                 string
	Surface                 string
	InitialAppName          string
	InitialWindowTitle      string
	InitialContextHash      string
	LastVerifiedContextHash string
	StartedAt               time.Time
	CompletedAt             time.Time
	Steps                   []navigatorStepTrace
	Attempts                []navigatorAttemptTrace
}

func (s *navigatorSessionState) snapshotTask(completedAt time.Time) *navigatorTaskSnapshot {
	if strings.TrimSpace(s.activeTaskID) == "" {
		return nil
	}
	steps := append([]navigatorStepTrace(nil), s.stepHistory...)
	attempts := append([]navigatorAttemptTrace(nil), s.attemptHistory...)
	return &navigatorTaskSnapshot{
		TaskID:                  strings.TrimSpace(s.activeTaskID),
		Command:                 strings.TrimSpace(s.activeCommand),
		Surface:                 navigatorSurfaceFromState(*s),
		InitialAppName:          strings.TrimSpace(s.initialAppName),
		InitialWindowTitle:      strings.TrimSpace(s.initialWindowTitle),
		InitialContextHash:      strings.TrimSpace(s.initialContextHash),
		LastVerifiedContextHash: strings.TrimSpace(s.lastVerifiedContextHash),
		StartedAt:               s.createdAt,
		CompletedAt:             completedAt.UTC(),
		Steps:                   steps,
		Attempts:                attempts,
	}
}

func (s *navigatorSessionState) touch() {
	now := time.Now().UTC()
	if s.createdAt.IsZero() {
		s.createdAt = now
	}
	s.updatedAt = now
}

func snapshotNavigatorContext(ctx navigatorContext) navigatorContextSnapshot {
	return navigatorContextSnapshot{
		AppName:          strings.TrimSpace(ctx.AppName),
		BundleID:         strings.TrimSpace(ctx.BundleID),
		WindowTitle:      strings.TrimSpace(ctx.WindowTitle),
		FocusedRole:      strings.TrimSpace(ctx.FocusedRole),
		FocusedLabel:     strings.TrimSpace(ctx.FocusedLabel),
		SelectedTextHash: shortStableHash(ctx.SelectedText),
		AXSnapshotHash:   shortStableHash(ctx.AXSnapshot),
	}
}

func navigatorContextHash(ctx navigatorContext) string {
	payload := strings.Join([]string{
		strings.TrimSpace(ctx.AppName),
		strings.TrimSpace(ctx.BundleID),
		strings.TrimSpace(ctx.FrontmostBundleID),
		strings.TrimSpace(ctx.WindowTitle),
		strings.TrimSpace(ctx.FocusedRole),
		strings.TrimSpace(ctx.FocusedLabel),
		strings.TrimSpace(ctx.SelectedText),
		strings.TrimSpace(ctx.AXSnapshot),
		strings.TrimSpace(ctx.InputFieldHint),
		strings.TrimSpace(ctx.LastInputDescriptor),
		fmt.Sprintf("focus_stable_ms=%d", ctx.FocusStableMs),
		fmt.Sprintf("capture_confidence=%.3f", ctx.CaptureConfidence),
		fmt.Sprintf("visible_inputs=%d", ctx.VisibleInputCandidates),
		strings.TrimSpace(ctx.AccessibilityPermission),
		fmt.Sprintf("ax_trusted=%t", ctx.AccessibilityTrusted),
	}, "\n")
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func shortStableHash(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(trimmed))
	return hex.EncodeToString(sum[:])[:16]
}

func newNavigatorTaskID() string {
	return "task_" + newConnID()
}

func planNavigatorCommand(command string, ctx navigatorContext, allowRisky bool) navigatorPlan {
	intentClass, confidence, clarifyQuestion := classifyNavigatorIntent(command)
	plan := navigatorPlan{
		Command:          strings.TrimSpace(command),
		IntentClass:      intentClass,
		IntentConfidence: confidence,
		ClarifyQuestion:  clarifyQuestion,
		ClarifyMode:      navigatorClarificationConfirm,
	}

	if intentClass == navigatorIntentAmbiguous {
		return plan
	}
	if intentClass == navigatorIntentAnalyzeOnly {
		return plan
	}

	switch {
	case looksLikeSystemAction(command):
		plan.Steps = buildSystemActionSteps(command, confidence)
	case wantsTextEntry(command, ctx):
		plan.Steps = buildTextEntrySteps(command, ctx, confidence)
	case shouldUseDocsLookup(command):
		plan.Steps = buildDocsLookupSteps(command, ctx, confidence)
	case looksLikeDirectClick(command):
		if step, ok := buildAXPressStep(command, ctx, confidence); ok {
			plan.Steps = []navigatorStep{step}
		}
	case canUseTerminalCommand(command, ctx):
		plan.Steps = buildTerminalCommandSteps(command, ctx, confidence)
	case wantsAntigravityAction(command, ctx):
		plan.Steps = buildAntigravityInlineSteps(command, ctx, confidence)
	case intentClass == navigatorIntentAnalyzeOnly:
		plan.Steps = nil
	default:
		plan.ClarifyQuestion = defaultClarifyQuestion(command)
		plan.IntentClass = navigatorIntentAmbiguous
	}

	if plan.IntentClass != navigatorIntentAnalyzeOnly && plan.IntentClass != navigatorIntentAmbiguous && len(plan.Steps) == 0 {
		plan.IntentClass = navigatorIntentAmbiguous
		plan.ClarifyQuestion = clarificationForUnresolvedTarget(command, ctx)
		plan.ClarifyMode = clarificationModeForUnresolvedTarget(command, ctx)
	}

	if plan.IntentClass == navigatorIntentAnalyzeOnly || plan.IntentClass == navigatorIntentAmbiguous {
		return plan
	}

	if riskReason := planRiskReason(command, ctx, plan.Steps); riskReason != "" && !allowRisky && stepsRequireRiskConfirmation(plan.Steps) {
		plan.RiskReason = riskReason
		plan.RiskQuestion = buildRiskQuestion(command)
		plan.Steps = nil
	}

	return plan
}

func classifyNavigatorIntent(command string) (navigatorIntentClass, float64, string) {
	lowered := strings.ToLower(strings.TrimSpace(command))
	if lowered == "" {
		return navigatorIntentAmbiguous, 0, defaultClarifyQuestion(command)
	}

	executeScore := keywordScore(lowered, []string{
		"apply", "do it", "run it", "rerun", "retry", "fix", "execute", "take care of",
		"type", "enter", "paste", "fill", "write", "focus the input", "focus the field",
		"volume", "mute", "unmute", "quieter", "louder",
		"반영", "적용", "실행", "다시 돌려", "다시 실행", "수정", "해결", "처리해", "눌러", "입력", "붙여넣", "써", "쳐", "볼륨", "음량", "음소거", "소리",
	})
	openScore := keywordScore(lowered, []string{
		"open", "go to", "take me", "bring me", "jump", "navigate", "show me",
		"열어", "이동", "데려가", "가보자", "보여", "가자",
	})
	findScore := keywordScore(lowered, []string{
		"find", "look up", "search", "docs", "official", "where is", "locate", "input field", "text field", "search box",
		"찾아", "검색", "공식 문서", "위치", "어디", "입력창", "검색창", "텍스트 필드",
	})
	analyzeScore := keywordScore(lowered, []string{
		"explain", "summarize", "what is", "why", "how", "tell me about",
		"설명", "요약", "왜", "어떻게", "뭐야", "알려줘",
	})

	if strings.Contains(lowered, "test") && (strings.Contains(lowered, "again") || strings.Contains(lowered, "retry") || strings.Contains(lowered, "check")) {
		executeScore = maxFloat(executeScore, 0.78)
	}
	if strings.Contains(lowered, "테스트") && (strings.Contains(lowered, "확인") || strings.Contains(lowered, "다시")) {
		executeScore = maxFloat(executeScore, 0.78)
	}
	if strings.Contains(lowered, "반영해") || strings.Contains(lowered, "적용해") || strings.Contains(lowered, "실행해") ||
		strings.Contains(lowered, "처리해") || strings.Contains(lowered, "수정해") || strings.Contains(lowered, "눌러줘") ||
		strings.Contains(lowered, "다시 돌려") {
		executeScore = maxFloat(executeScore, 0.76)
	}
	if containsKeywordAny(lowered, "입력해 주세요", "넣어주세요", "쳐줘", "쳐 줘", "입력하자", "넣어보자") {
		executeScore = maxFloat(executeScore, 0.78)
	}
	if requestsTextInsertion(command) {
		executeScore = maxFloat(executeScore, 0.88)
	}
	if containsKeywordAny(lowered, "volume", "mute", "unmute", "quieter", "louder", "볼륨", "음량", "음소거", "소리 줄", "소리 키") {
		executeScore = maxFloat(executeScore, 0.82)
	}
	if strings.Contains(lowered, "열어줘") || strings.Contains(lowered, "데려가") || strings.Contains(lowered, "가보자") {
		openScore = maxFloat(openScore, 0.72)
	}
	if strings.Contains(lowered, "찾아줘") || strings.Contains(lowered, "찾아봐") {
		findScore = maxFloat(findScore, 0.72)
	}
	if strings.Contains(lowered, "입력창") || strings.Contains(lowered, "검색창") || strings.Contains(lowered, "input field") || strings.Contains(lowered, "search box") {
		findScore = maxFloat(findScore, 0.74)
	}
	if strings.Contains(lowered, "official docs") || strings.Contains(lowered, "공식 문서") {
		findScore = maxFloat(findScore, 0.88)
	}
	if (strings.Contains(lowered, "explain") || strings.Contains(lowered, "설명해") || strings.Contains(lowered, "알려줘") ||
		strings.Contains(lowered, "what is") || strings.Contains(lowered, "why")) &&
		executeScore < 0.4 && openScore < 0.4 && findScore < 0.4 {
		analyzeScore = maxFloat(analyzeScore, 0.72)
	}

	topClass := navigatorIntentAmbiguous
	topScore := 0.0
	secondScore := 0.0
	for _, candidate := range []struct {
		class navigatorIntentClass
		score float64
	}{
		{navigatorIntentExecuteNow, executeScore},
		{navigatorIntentOpenNavigate, openScore},
		{navigatorIntentFindLookup, findScore},
		{navigatorIntentAnalyzeOnly, analyzeScore},
	} {
		if candidate.score > topScore {
			secondScore = topScore
			topScore = candidate.score
			topClass = candidate.class
		} else if candidate.score > secondScore {
			secondScore = candidate.score
		}
	}

	if topScore < 0.58 {
		return navigatorIntentAmbiguous, topScore, defaultClarifyQuestion(command)
	}
	if topScore-secondScore < 0.12 && topClass != navigatorIntentAnalyzeOnly {
		return navigatorIntentAmbiguous, topScore, defaultClarifyQuestion(command)
	}
	return topClass, topScore, ""
}

func keywordScore(text string, keywords []string) float64 {
	score := 0.0
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			score += 0.28
		}
	}
	if score > 0.96 {
		return 0.96
	}
	return score
}

func planRiskReason(command string, ctx navigatorContext, steps []navigatorStep) string {
	// Only check the command text and the text being pasted/typed.
	// Do NOT check ctx.SelectedText — it is ambient screen context, not an action payload.
	// Including it causes false positives when variable names like "token" appear on screen.
	inputs := []string{command}
	for _, step := range steps {
		inputs = append(inputs, step.InputText)
	}
	return navigatorRiskReason(inputs...)
}

func navigatorRiskReason(inputs ...string) string {
	for _, input := range inputs {
		lowered := strings.ToLower(input)
		for _, keyword := range []string{
			"password", "token", "secret", "deploy", "production", "prod", "delete", "remove",
			"git push", "publish", "submit", "send", "rm -rf", "sudo ", "shutdown ", "reboot ",
			"비밀번호", "토큰", "배포", "삭제", "전송", "제출",
		} {
			if strings.Contains(lowered, keyword) {
				return fmt.Sprintf("blocked as a risky action because it mentions %q", keyword)
			}
		}
	}
	return ""
}

func stepsRequireRiskConfirmation(steps []navigatorStep) bool {
	for _, step := range steps {
		switch step.ActionType {
		case "paste_text", "press_ax":
			return true
		case "hotkey":
			if len(step.Hotkey) == 1 && strings.EqualFold(step.Hotkey[0], "return") {
				return true
			}
		}
	}
	return false
}

func buildRiskQuestion(command string) string {
	return "This could change something important. Do you want me to proceed, or should I only explain the next step?"
}

func defaultClarifyQuestion(command string) string {
	return "Do you want me to act on this now, or just explain the next step?"
}

func clarificationModeForUnresolvedTarget(command string, ctx navigatorContext) navigatorClarificationResponseMode {
	switch {
	case wantsTextEntry(command, ctx):
		return navigatorClarificationProvideDetail
	case looksLikeDirectClick(command):
		return navigatorClarificationProvideDetail
	default:
		return navigatorClarificationConfirm
	}
}

func clarificationForUnresolvedTarget(command string, ctx navigatorContext) string {
	lowered := strings.ToLower(strings.TrimSpace(command))
	switch {
	case wantsTextEntry(command, ctx) && requiresDerivedTextPayload(command):
		return "I could not safely read the exact text to type from the current screen yet. Do you want me to keep analyzing the visible content, or should I only explain the next step?"
	case wantsTextEntry(command, ctx) && requiresExactTextPayload(command):
		return "I can focus the input field, but I still need the exact text to type. Tell me the text, or ask me to copy it from what is visible."
	case wantsTextEntry(command, ctx) && ctx.VisibleInputCandidates > 1 && ctx.CaptureConfidence < 0.7:
		return fmt.Sprintf("I can see %d possible input fields, so I should not guess. Do you want me to focus the likely one first, or should I only explain the next step?", ctx.VisibleInputCandidates)
	case wantsTextEntry(command, ctx):
		return "I could not confirm which input field you mean. Do you want me to focus the likely field first, or should I only explain the next step?"
	case looksLikeDirectClick(command):
		return "I could not confirm which control you want me to press. Do you want me to keep looking, or should I only explain the next step?"
	case containsKeywordAny(lowered, "여기", "거기", "here", "there") && ctx.FocusStableMs < 300:
		return "The current focus is still changing. Do you want me to wait for the UI to settle and try again, or should I only explain the next step?"
	default:
		return defaultClarifyQuestion(command)
	}
}

func clarificationPromptKindForPlan(plan navigatorPlan) navigatorPromptKind {
	switch plan.ClarifyMode {
	case navigatorClarificationProvideDetail:
		return navigatorPromptProvideDetail
	default:
		return navigatorPromptConfirmIntent
	}
}

func clarificationResponseModeForPrompt(kind navigatorPromptKind) navigatorClarificationResponseMode {
	switch kind {
	case navigatorPromptProvideDetail:
		return navigatorClarificationProvideDetail
	default:
		return navigatorClarificationConfirm
	}
}

func buildTaskReplacementQuestion(activeCommand, nextCommand string) string {
	active := cleanTopic(activeCommand)
	next := cleanTopic(nextCommand)
	switch {
	case active == "" && next == "":
		return "I am already working on one action. Do you want me to stop it and switch to this new one?"
	case active == "":
		return fmt.Sprintf("I am already working on another action. Do you want me to stop it and switch to %q?", next)
	case next == "":
		return fmt.Sprintf("I am already working on %q. Do you want me to stop that and switch tasks?", active)
	default:
		return fmt.Sprintf("I am already working on %q. Do you want me to stop that and switch to %q?", active, next)
	}
}

func needsVisionCheckpoint(completedStep navigatorStep, observedOutcome string) bool {
	switch completedStep.ActionType {
	case "open_url":
		return true
	case "paste_text":
		return true
	case "hotkey":
		for _, k := range completedStep.Hotkey {
			low := strings.ToLower(k)
			if low == "return" || low == "enter" {
				return true
			}
		}
	}
	if strings.Contains(strings.ToLower(observedOutcome), "inconclusive") {
		return true
	}
	return false
}

func buildNextActionHint(toolName, text, target string) string {
	switch toolName {
	case "navigate_open_url":
		lowered := strings.ToLower(text)
		if strings.Contains(lowered, "music.youtube") {
			return "YouTube Music page is loading. Next: call navigate_type_and_submit to search for music, then navigate_hotkey with keys=[\"space\"] to ensure playback starts."
		}
		if strings.Contains(lowered, "youtube") {
			return "YouTube is open. If searching, call navigate_type_and_submit to enter the search query."
		}
		return "URL opened. If the user's task requires typing on this page, call navigate_type_and_submit. The client automatically finds the right input field."
	case "navigate_focus_app":
		return "App is now focused and frontmost. If the user requested an action in this app, call the next tool. The client automatically detects text fields."
	case "navigate_type_and_submit":
		targetLower := strings.ToLower(target)
		textLower := strings.ToLower(text)
		if strings.Contains(targetLower, "youtube") || strings.Contains(targetLower, "music") ||
			strings.Contains(textLower, "music") || strings.Contains(textLower, "음악") {
			return "Search submitted on YouTube Music. The system is automatically clicking the first playable result to start music. Do NOT send any additional tool calls — wait for playback confirmation."
		}
		return "Text was typed and submitted. If the task requires more steps (e.g., clicking play), call navigate_hotkey next."
	case "navigate_text_entry":
		return "Text was entered into the field. If submission is needed, call navigate_hotkey with keys=[\"return\"]."
	default:
		return ""
	}
}

func shouldUseDocsLookup(command string) bool {
	lowered := strings.ToLower(command)
	return strings.Contains(lowered, "official") ||
		strings.Contains(lowered, "docs") ||
		strings.Contains(lowered, "documentation") ||
		strings.Contains(lowered, "공식 문서") ||
		strings.Contains(lowered, "문서")
}

func wantsAntigravityAction(command string, ctx navigatorContext) bool {
	lowered := strings.ToLower(command)
	app := strings.ToLower(ctx.AppName)
	return strings.Contains(app, "antigravity") ||
		strings.Contains(lowered, "antigravity") ||
		strings.Contains(lowered, "apply") ||
		strings.Contains(lowered, "반영") ||
		strings.Contains(lowered, "fix") ||
		strings.Contains(lowered, "수정") ||
		strings.Contains(lowered, "move me") ||
		strings.Contains(lowered, "데려가")
}

func canUseTerminalCommand(command string, ctx navigatorContext) bool {
	if extractCommandPayload(command) != "" {
		return true
	}
	return looksLikeShellCommand(ctx.SelectedText)
}

func looksLikeDirectClick(command string) bool {
	lowered := strings.ToLower(command)
	return strings.Contains(lowered, "click ") || strings.Contains(lowered, "press ") || strings.Contains(lowered, "눌러")
}

var volumeAmountPattern = regexp.MustCompile(`(\d{1,3})`)

func looksLikeSystemAction(command string) bool {
	_, _, _, _, ok := parseSystemAction(command)
	return ok
}

func buildSystemActionSteps(command string, intentConfidence float64) []navigatorStep {
	systemCommand, systemValue, amount, expectedOutcome, ok := parseSystemAction(command)
	if !ok {
		return nil
	}
	return []navigatorStep{{
		ID:               systemCommand + "_" + systemValue,
		ActionType:       "system_action",
		TargetApp:        "macOS",
		TargetDescriptor: navigatorTargetDescriptor{AppName: "macOS"},
		ExpectedOutcome:  expectedOutcome,
		Confidence:       0.9,
		IntentConfidence: intentConfidence,
		RiskLevel:        "low",
		ExecutionPolicy:  navigatorExecutionPolicyLow,
		FallbackPolicy:   "guided_mode",
		SystemCommand:    systemCommand,
		SystemValue:      systemValue,
		SystemAmount:     amount,
		Narration:        expectedOutcome,
		TimeoutMs:        1200,
		ProofLevel:       "basic",
	}}
}

func parseSystemAction(command string) (string, string, int, string, bool) {
	lowered := strings.ToLower(strings.TrimSpace(command))
	if lowered == "" {
		return "", "", 0, "", false
	}
	if requestsTextInsertion(command) {
		return "", "", 0, "", false
	}

	switch {
	case containsKeywordAny(lowered, "unmute", "음소거 해제", "mute off", "소리 켜", "소리 다시 켜", "음량 다시 켜"):
		return "volume", "unmute", 0, "System audio is unmuted", true
	case containsKeywordAny(lowered, "mute", "음소거", "소리 꺼", "음량 꺼"):
		return "volume", "mute", 0, "System audio is muted", true
	case containsKeywordAny(lowered, "volume down", "turn down", "lower volume", "quieter") ||
		(containsKeywordAny(lowered, "볼륨", "소리", "음량") && containsKeywordAny(lowered, "줄", "낮")):
		return "volume", "down", parsedSystemAmount(lowered, 12), "System volume is lower", true
	case containsKeywordAny(lowered, "volume up", "turn up", "raise volume", "louder") ||
		(containsKeywordAny(lowered, "볼륨", "소리", "음량") && containsKeywordAny(lowered, "올", "높", "키")):
		return "volume", "up", parsedSystemAmount(lowered, 12), "System volume is higher", true
	default:
		return "", "", 0, "", false
	}
}

func parsedSystemAmount(command string, fallback int) int {
	if matches := volumeAmountPattern.FindStringSubmatch(command); len(matches) == 2 {
		if value, err := strconv.Atoi(matches[1]); err == nil {
			switch {
			case value < 1:
				return fallback
			case value > 100:
				return 100
			default:
				return value
			}
		}
	}
	return fallback
}

func buildDocsLookupSteps(command string, ctx navigatorContext, intentConfidence float64) []navigatorStep {
	steps := make([]navigatorStep, 0, 2)
	if !strings.Contains(strings.ToLower(ctx.AppName), "chrome") {
		steps = append(steps, navigatorStep{
			ID:               "focus_chrome",
			ActionType:       "focus_app",
			TargetApp:        "Chrome",
			TargetDescriptor: navigatorTargetDescriptor{AppName: "Chrome"},
			ExpectedOutcome:  "Chrome is ready for the docs lookup",
			Confidence:       0.96,
			IntentConfidence: intentConfidence,
			RiskLevel:        "low",
			ExecutionPolicy:  navigatorExecutionPolicyLow,
			FallbackPolicy:   "guided_mode",
			VerifyHint:       "chrome",
			Surface:          "chrome",
			MacroID:          "focus_chrome",
			Narration:        "Switching to Chrome first.",
			VerifyContract: &navigatorVerifyContract{
				ExpectedBundleID:       "com.google.Chrome",
				ExpectedWindowContains: "Chrome",
				RequireFrontmostApp:    true,
				ProofStrategy:          "frontmost_app",
			},
			TimeoutMs:  900,
			ProofLevel: "strong",
		})
	}

	query := buildDocsSearchQuery(command, ctx)
	steps = append(steps, navigatorStep{
		ID:         "open_docs_search",
		ActionType: "open_url",
		TargetApp:  "Chrome",
		TargetDescriptor: navigatorTargetDescriptor{
			AppName: "Chrome",
		},
		ExpectedOutcome:  "Chrome opens the relevant official docs search",
		Confidence:       0.93,
		IntentConfidence: intentConfidence,
		RiskLevel:        "low",
		ExecutionPolicy:  navigatorExecutionPolicyLow,
		FallbackPolicy:   "guided_mode",
		URL:              "https://www.google.com/search?q=" + url.QueryEscape(query),
		VerifyHint:       "google",
		Surface:          "chrome",
		MacroID:          "open_docs_search",
		Narration:        "Opening the official docs in Chrome.",
		VerifyContract: &navigatorVerifyContract{
			ExpectedBundleID:       "com.google.Chrome",
			ExpectedWindowContains: "Google",
			RequireFrontmostApp:    true,
			ProofStrategy:          "window_change",
		},
		FallbackActionType: "hotkey",
		FallbackHotkey:     []string{"command", "l"},
		MaxLocalRetries:    1,
		TimeoutMs:          1500,
		ProofLevel:         "strong",
	})
	return steps
}

func buildDocsSearchQuery(command string, ctx navigatorContext) string {
	topic := cleanTopic(ctx.SelectedText)
	if topic == "" {
		topic = cleanTopic(ctx.WindowTitle)
	}
	if topic == "" {
		topic = cleanTopic(command)
	}
	if topic == "" {
		topic = "Antigravity IDE documentation"
	}
	return "site:developers.google.com OR site:ai.google.dev " + topic
}

func buildAntigravityInlineSteps(command string, ctx navigatorContext, intentConfidence float64) []navigatorStep {
	steps := make([]navigatorStep, 0, 3)
	if !strings.Contains(strings.ToLower(ctx.AppName), "antigravity") {
		steps = append(steps, navigatorStep{
			ID:               "focus_antigravity",
			ActionType:       "focus_app",
			TargetApp:        "Antigravity",
			TargetDescriptor: navigatorTargetDescriptor{AppName: "Antigravity"},
			ExpectedOutcome:  "Antigravity is frontmost",
			Confidence:       0.9,
			IntentConfidence: intentConfidence,
			RiskLevel:        "low",
			ExecutionPolicy:  navigatorExecutionPolicyLow,
			FallbackPolicy:   "guided_mode",
			VerifyHint:       "antigravity",
			Surface:          "antigravity",
			MacroID:          "focus_antigravity",
			Narration:        "Switching back to Antigravity.",
			VerifyContract: &navigatorVerifyContract{
				ExpectedBundleID:       "com.openai.codex",
				ExpectedWindowContains: "Codex",
				RequireFrontmostApp:    true,
				ProofStrategy:          "frontmost_app",
			},
			TimeoutMs:  900,
			ProofLevel: "strong",
		})
	}

	prompt := buildAntigravityPrompt(command, ctx)
	steps = append(steps,
		navigatorStep{
			ID:               "open_antigravity_inline_prompt",
			ActionType:       "hotkey",
			TargetApp:        "Antigravity",
			TargetDescriptor: navigatorTargetDescriptor{AppName: "Antigravity", WindowTitle: ctx.WindowTitle},
			ExpectedOutcome:  "Antigravity is ready to receive an inline instruction",
			Confidence:       0.82,
			IntentConfidence: intentConfidence,
			RiskLevel:        "low",
			ExecutionPolicy:  navigatorExecutionPolicyLow,
			FallbackPolicy:   "guided_mode",
			Hotkey:           []string{"command", "i"},
			Surface:          "antigravity",
			MacroID:          "open_antigravity_inline_prompt",
			Narration:        "Opening Antigravity inline prompt.",
			VerifyContract: &navigatorVerifyContract{
				ExpectedBundleID:    "com.openai.codex",
				RequireFrontmostApp: true,
				ProofStrategy:       "frontmost_app",
			},
			TimeoutMs:  900,
			ProofLevel: "strong",
		},
		navigatorStep{
			ID:               "paste_antigravity_instruction",
			ActionType:       "paste_text",
			TargetApp:        "Antigravity",
			TargetDescriptor: navigatorTargetDescriptor{AppName: "Antigravity", WindowTitle: ctx.WindowTitle},
			InputText:        prompt,
			ExpectedOutcome:  "Antigravity receives the requested navigation or apply instruction",
			Confidence:       0.8,
			IntentConfidence: intentConfidence,
			RiskLevel:        "low",
			ExecutionPolicy:  navigatorExecutionPolicyLow,
			FallbackPolicy:   "guided_mode",
			Surface:          "antigravity",
			MacroID:          "paste_antigravity_instruction",
			Narration:        "Inserting the Antigravity instruction.",
			VerifyContract: &navigatorVerifyContract{
				ExpectedBundleID:          "com.openai.codex",
				RequireFrontmostApp:       true,
				RequireWritableTarget:     true,
				MinCaptureConfidenceAfter: 0.6,
				ProofStrategy:             "text_entry",
			},
			MaxLocalRetries: 1,
			TimeoutMs:       1200,
			ProofLevel:      "strict",
		},
		navigatorStep{
			ID:               "submit_antigravity_instruction",
			ActionType:       "hotkey",
			TargetApp:        "Antigravity",
			TargetDescriptor: navigatorTargetDescriptor{AppName: "Antigravity", WindowTitle: ctx.WindowTitle},
			ExpectedOutcome:  "Antigravity starts the requested action",
			Confidence:       0.76,
			IntentConfidence: intentConfidence,
			RiskLevel:        "low",
			ExecutionPolicy:  navigatorExecutionPolicyLow,
			FallbackPolicy:   "guided_mode",
			Hotkey:           []string{"return"},
			Surface:          "antigravity",
			MacroID:          "submit_antigravity_instruction",
			Narration:        "Submitting the Antigravity instruction.",
			VerifyContract: &navigatorVerifyContract{
				ExpectedBundleID:    "com.openai.codex",
				RequireFrontmostApp: true,
				ProofStrategy:       "post_submit",
			},
			TimeoutMs:  900,
			ProofLevel: "strong",
		},
	)
	return steps
}

func buildAntigravityPrompt(command string, ctx navigatorContext) string {
	contextBits := make([]string, 0, 3)
	if selected := cleanTopic(ctx.SelectedText); selected != "" {
		contextBits = append(contextBits, "Context: "+selected)
	}
	if title := cleanTopic(ctx.WindowTitle); title != "" {
		contextBits = append(contextBits, "Window: "+title)
	}
	base := fmt.Sprintf("Please help with this request inside the current project: %s", strings.TrimSpace(command))
	if len(contextBits) == 0 {
		return base
	}
	return base + "\n" + strings.Join(contextBits, "\n")
}

func buildTerminalCommandSteps(command string, ctx navigatorContext, intentConfidence float64) []navigatorStep {
	payload := extractCommandPayload(command)
	if payload == "" {
		payload = cleanTopic(ctx.SelectedText)
	}
	if payload == "" {
		return nil
	}

	steps := make([]navigatorStep, 0, 3)
	if !strings.Contains(strings.ToLower(ctx.AppName), "terminal") {
		steps = append(steps, navigatorStep{
			ID:               "focus_terminal",
			ActionType:       "focus_app",
			TargetApp:        "Terminal",
			TargetDescriptor: navigatorTargetDescriptor{AppName: "Terminal"},
			ExpectedOutcome:  "Terminal is frontmost",
			Confidence:       0.93,
			IntentConfidence: intentConfidence,
			RiskLevel:        "low",
			ExecutionPolicy:  navigatorExecutionPolicyLow,
			FallbackPolicy:   "guided_mode",
			VerifyHint:       "terminal",
			Surface:          "terminal",
			MacroID:          "focus_terminal",
			Narration:        "Switching to Terminal.",
			VerifyContract: &navigatorVerifyContract{
				ExpectedBundleID:       "com.apple.Terminal",
				ExpectedWindowContains: "Terminal",
				RequireFrontmostApp:    true,
				ProofStrategy:          "frontmost_app",
			},
			TimeoutMs:  900,
			ProofLevel: "strong",
		})
	}
	steps = append(steps,
		navigatorStep{
			ID:               "paste_terminal_command",
			ActionType:       "paste_text",
			TargetApp:        "Terminal",
			TargetDescriptor: navigatorTargetDescriptor{AppName: "Terminal", WindowTitle: ctx.WindowTitle},
			InputText:        payload,
			ExpectedOutcome:  "The command is ready in Terminal",
			Confidence:       0.88,
			IntentConfidence: intentConfidence,
			RiskLevel:        "low",
			ExecutionPolicy:  navigatorExecutionPolicyLow,
			FallbackPolicy:   "guided_mode",
			Surface:          "terminal",
			MacroID:          "paste_terminal_command",
			Narration:        "Placing the command into Terminal.",
			VerifyContract: &navigatorVerifyContract{
				ExpectedBundleID:          "com.apple.Terminal",
				RequireFrontmostApp:       true,
				RequireWritableTarget:     true,
				MinCaptureConfidenceAfter: 0.55,
				ProofStrategy:             "terminal_prompt",
			},
			MaxLocalRetries: 1,
			TimeoutMs:       1100,
			ProofLevel:      "strict",
		},
		navigatorStep{
			ID:               "submit_terminal_command",
			ActionType:       "hotkey",
			TargetApp:        "Terminal",
			TargetDescriptor: navigatorTargetDescriptor{AppName: "Terminal", WindowTitle: ctx.WindowTitle},
			ExpectedOutcome:  "Terminal runs the requested command",
			Confidence:       0.88,
			IntentConfidence: intentConfidence,
			RiskLevel:        "low",
			ExecutionPolicy:  navigatorExecutionPolicyLow,
			FallbackPolicy:   "guided_mode",
			Hotkey:           []string{"return"},
			VerifyHint:       "terminal",
			Surface:          "terminal",
			MacroID:          "submit_terminal_command",
			Narration:        "Running the command in Terminal.",
			VerifyContract: &navigatorVerifyContract{
				ExpectedBundleID:    "com.apple.Terminal",
				RequireFrontmostApp: true,
				ProofStrategy:       "post_submit",
			},
			TimeoutMs:  900,
			ProofLevel: "strong",
		},
	)
	return steps
}

func buildAXPressStep(command string, ctx navigatorContext, intentConfidence float64) (navigatorStep, bool) {
	label := extractClickLabel(command)
	if label == "" {
		return navigatorStep{}, false
	}
	return navigatorStep{
		ID:         "press_ax_target",
		ActionType: "press_ax",
		TargetApp:  ctx.AppName,
		TargetDescriptor: navigatorTargetDescriptor{
			AppName:        ctx.AppName,
			WindowTitle:    ctx.WindowTitle,
			Label:          label,
			RelativeAnchor: cleanTopic(ctx.FocusedLabel),
		},
		ExpectedOutcome:  fmt.Sprintf("Press the %s control", label),
		Confidence:       0.7,
		IntentConfidence: intentConfidence,
		RiskLevel:        "low",
		ExecutionPolicy:  navigatorExecutionPolicyLow,
		FallbackPolicy:   "guided_mode",
		VerifyHint:       strings.ToLower(label),
		Surface:          navigatorSurfaceValue(ctx.AppName),
		MacroID:          "press_ax_target",
		Narration:        "Focusing the target control.",
		VerifyContract: &navigatorVerifyContract{
			ExpectedWindowContains: ctx.WindowTitle,
			ExpectedFocusedLabel:   label,
			RequireFrontmostApp:    true,
			ProofStrategy:          "target_focus",
		},
		TimeoutMs:  800,
		ProofLevel: "strong",
	}, true
}

var quotedTextPattern = regexp.MustCompile(`"([^"]+)"|“([^”]+)”|'([^']+)'|‘([^’]+)’`)
var characterRangePattern = regexp.MustCompile(`(?i)\b([a-z0-9])\s*(?:부터|to|through|until|til|~|-)\s*([a-z0-9])(?:\s*까지)?\b`)
var characterRangeAppendLeadPattern = regexp.MustCompile(`(?is)\b([a-z0-9])\s*(?:부터|to|through|until|til|~|-)\s*([a-z0-9])(?:\s*까지)?\s*(?:뒤에|after)\s*(.*)$`)

func wantsTextEntry(command string, ctx navigatorContext) bool {
	lowered := strings.ToLower(strings.TrimSpace(command))
	if lowered == "" {
		return false
	}
	if extractTextEntryPayload(command) != "" {
		return true
	}
	for _, keyword := range []string{
		"input field", "text field", "search box", "search field", "address bar", "prompt box",
		"type ", "enter ", "paste ", "fill ", "focus the input", "focus the field",
		"입력창", "텍스트 필드", "검색창", "주소창", "프롬프트", "입력해", "붙여넣어", "써줘",
	} {
		if strings.Contains(lowered, keyword) {
			return true
		}
	}
	return hasVisibleTextInput(ctx) && containsKeywordAny(lowered, "여기에", "거기에", "here", "there")
}

func buildTextEntrySteps(command string, ctx navigatorContext, intentConfidence float64) []navigatorStep {
	descriptor, ok := buildTextEntryDescriptor(command, ctx)
	if !ok {
		return nil
	}
	return buildTextEntryStepsForDescriptor(command, ctx, intentConfidence, descriptor, 0.82, "")
}

func buildTextEntryDescriptor(command string, ctx navigatorContext) (navigatorTargetDescriptor, bool) {
	if referencesCurrentTarget(command) && ctx.FocusStableMs > 0 && ctx.FocusStableMs < 300 && !looksLikeTextInputRole(ctx.FocusedRole) {
		return navigatorTargetDescriptor{}, false
	}
	if referencesCurrentTarget(command) && ctx.VisibleInputCandidates > 1 && !looksLikeTextInputRole(ctx.FocusedRole) && ctx.CaptureConfidence < 0.7 {
		return navigatorTargetDescriptor{}, false
	}

	role := inferTextEntryRole(command, ctx)
	label := extractTextEntryFieldLabel(command, ctx)
	anchor := cleanTopic(ctx.FocusedLabel)
	if anchor == "" {
		anchor = cleanTopic(ctx.InputFieldHint)
	}
	if anchor == "" {
		anchor = cleanTopic(lastInputFieldDescriptorLabel(ctx.LastInputDescriptor))
	}

	descriptor := navigatorTargetDescriptor{
		Role:           role,
		Label:          label,
		WindowTitle:    cleanTopic(ctx.WindowTitle),
		AppName:        cleanTopic(ctx.AppName),
		RelativeAnchor: anchor,
	}
	if descriptor.Role == "" {
		descriptor.Role = "textfield"
	}
	if descriptor.Label == "" && !hasVisibleTextInput(ctx) && !looksLikeTextInputRole(ctx.FocusedRole) {
		return navigatorTargetDescriptor{}, false
	}
	return descriptor, true
}

func inferTextEntryRole(command string, ctx navigatorContext) string {
	lowered := strings.ToLower(command)
	if containsKeywordAny(lowered, "textarea", "text area", "본문", "설명", "description") {
		return "textarea"
	}
	if looksLikeTextInputRole(ctx.FocusedRole) && containsKeywordAny(lowered, "여기", "거기", "here", "there", "current") {
		return normalizeRoleToken(ctx.FocusedRole)
	}
	if descriptorRole := normalizeRoleToken(lastInputFieldDescriptorRole(ctx.LastInputDescriptor)); looksLikeTextInputRole(descriptorRole) {
		return descriptorRole
	}
	return "textfield"
}

func extractTextEntryFieldLabel(command string, ctx navigatorContext) string {
	lowered := strings.ToLower(command)
	if looksLikeTextInputRole(ctx.FocusedRole) && containsKeywordAny(lowered, "여기", "거기", "here", "there", "current") {
		return cleanTopic(ctx.FocusedLabel)
	}
	switch {
	case containsKeywordAny(lowered, "검색창", "search box", "search field", "search bar"):
		return "search"
	case containsKeywordAny(lowered, "주소창", "address bar", "url bar", "location bar"):
		return "address"
	case containsKeywordAny(lowered, "프롬프트", "prompt"):
		return "prompt"
	case containsKeywordAny(lowered, "채팅", "메시지", "message box", "chat box", "composer"):
		return "message"
	case containsKeywordAny(lowered, "입력창", "input field", "text field"):
		return cleanTopic(ctx.FocusedLabel)
	default:
		if hint := cleanTopic(ctx.InputFieldHint); hint != "" {
			return hint
		}
		return cleanTopic(lastInputFieldDescriptorLabel(ctx.LastInputDescriptor))
	}
}

func normalizeKoreanVerbSpacing(s string) string {
	r := s
	for _, pair := range [][2]string{
		{"적어 줘", "적어줘"}, {"입력해 줘", "입력해줘"}, {"쳐 줘", "쳐줘"},
		{"써 줘", "써줘"}, {"넣어 줘", "넣어줘"}, {"적어 주세요", "적어줘"},
		{"입력해 주세요", "입력해줘"}, {"쳐 주세요", "쳐줘"}, {"써 주세요", "써줘"},
		{"넣어 주세요", "넣어줘"},
	} {
		r = strings.ReplaceAll(r, pair[0], pair[1])
	}
	r = strings.TrimSuffix(r, "요")
	r = strings.TrimSuffix(r, ".")
	return r
}

func stripLocationPrefix(s string) string {
	for _, particle := range []string{"에서 ", "에 "} {
		if idx := strings.LastIndex(s, particle); idx >= 0 {
			remainder := strings.TrimSpace(s[idx+len(particle):])
			if remainder != "" {
				return remainder
			}
		}
	}
	return s
}

func extractTextEntryPayload(command string) string {
	if matches := commandLiteralPattern.FindStringSubmatch(command); len(matches) == 2 {
		return cleanTopic(matches[1])
	}
	if matches := quotedTextPattern.FindStringSubmatch(command); len(matches) > 0 {
		for _, candidate := range matches[1:] {
			if cleaned := cleanTopic(candidate); cleaned != "" {
				return cleaned
			}
		}
	}
	for _, marker := range []string{"입력해줘:", "입력해:", "입력:", "붙여넣어:", "붙여넣기:", "type:", "enter:", "paste:", "fill:"} {
		if idx := strings.Index(strings.ToLower(command), marker); idx >= 0 {
			return cleanTopic(command[idx+len(marker):])
		}
	}
	if payload := extractCharacterRangeAppendPayload(command); payload != "" {
		return payload
	}
	if payload := extractRelativeAppendPayload(command); payload != "" {
		return payload
	}
	if payload := extractCharacterRangePayload(command); payload != "" {
		return payload
	}
	if payload := extractBareDirectTextPayload(command); payload != "" {
		return payload
	}
	normalized := normalizeKoreanVerbSpacing(command)
	for _, suffix := range []string{
		"이라고 입력해줘", "이라고 입력해", "이라고 입력",
		"라고 입력해줘", "라고 입력해", "라고 입력",
		"이라고 쳐줘", "라고 쳐줘",
		"이라고 써줘", "라고 써줘",
		"이라고 넣어줘", "라고 넣어줘",
		"이라고 적어줘", "라고 적어줘",
	} {
		if idx := strings.Index(strings.ToLower(normalized), suffix); idx > 0 {
			candidate := strings.TrimSpace(normalized[:idx])
			if candidate != "" {
				return stripLocationPrefix(candidate)
			}
		}
	}
	return ""
}

func extractCharacterRangeAppendPayload(command string) string {
	matches := characterRangeAppendLeadPattern.FindStringSubmatch(strings.TrimSpace(command))
	if len(matches) != 4 {
		return ""
	}

	base := characterRangePayload(matches[1], matches[2])
	if base == "" {
		return ""
	}

	remainder := strings.TrimSpace(matches[3])
	if remainder == "" {
		return ""
	}

	leadingSpace := false
	for _, prefix := range []string{
		"한 칸 띄고", "한칸 띄고", "한 칸 띄운 다음", "한칸 띄운 다음",
		"1 칸 띄고", "1칸 띄고", "one space", "space",
	} {
		if strings.HasPrefix(strings.ToLower(remainder), prefix) {
			leadingSpace = true
			remainder = strings.TrimSpace(remainder[len(prefix):])
			break
		}
	}

	suffix := cleanTopic(trimTrailingTextEntryActionClause(remainder))
	if suffix == "" {
		return ""
	}
	if leadingSpace {
		return base + " " + suffix
	}
	return base + suffix
}

func extractRelativeAppendPayload(command string) string {
	lowered := strings.ToLower(strings.TrimSpace(command))
	if lowered == "" || !hasTextInsertionCue(lowered) {
		return ""
	}

	markerIndex := strings.LastIndex(lowered, "뒤에")
	markerLength := len("뒤에")
	if markerIndex < 0 {
		markerIndex = strings.LastIndex(lowered, "after")
		markerLength = len("after")
	}
	if markerIndex <= 0 {
		return ""
	}

	remainder := strings.TrimSpace(command[markerIndex+markerLength:])
	if remainder == "" {
		return ""
	}

	leadingSpace := false
	for _, prefix := range []string{
		"한 칸 띄고", "한칸 띄고", "한 칸 띄운 다음", "한칸 띄운 다음",
		"1 칸 띄고", "1칸 띄고", "one space", "space",
	} {
		if strings.HasPrefix(strings.ToLower(remainder), prefix) {
			leadingSpace = true
			remainder = strings.TrimSpace(remainder[len(prefix):])
			break
		}
	}

	suffix := cleanTopic(trimTrailingTextEntryActionClause(remainder))
	if suffix == "" {
		return ""
	}
	if leadingSpace {
		return " " + suffix
	}
	return suffix
}

func extractBareDirectTextPayload(command string) string {
	lowered := strings.ToLower(strings.TrimSpace(command))
	if lowered == "" || !hasTextInsertionCue(lowered) {
		return ""
	}
	if extractIntrinsicTextEntryPayload(command) != "" {
		return ""
	}
	if looksLikeScreenDerivedTextRequest(command, lowered) {
		return ""
	}

	trimmed := strings.TrimSpace(command)
	candidate := cleanTopic(trimTrailingTextEntryActionClause(command))
	if candidate == "" || candidate == cleanTopic(trimmed) {
		return ""
	}
	if containsKeywordAny(strings.ToLower(candidate),
		"뒤에", "after", "앞에", "before",
		"검색창", "입력창", "text field", "search box", "prompt", "field", "box",
		"codex", "chrome", "terminal", "여기", "거기", "here", "there",
	) {
		return ""
	}
	return candidate
}

func extractCharacterRangePayload(command string) string {
	matches := characterRangePattern.FindStringSubmatch(strings.TrimSpace(command))
	if len(matches) != 3 {
		return ""
	}

	return characterRangePayload(matches[1], matches[2])
}

func characterRangePayload(startRaw, endRaw string) string {
	start := []rune(strings.ToUpper(startRaw))
	end := []rune(strings.ToUpper(endRaw))
	if len(start) != 1 || len(end) != 1 {
		return ""
	}

	switch {
	case start[0] >= 'A' && start[0] <= 'Z' && end[0] >= 'A' && end[0] <= 'Z':
		if start[0] > end[0] {
			return ""
		}
		var b strings.Builder
		for r := start[0]; r <= end[0]; r++ {
			b.WriteRune(r)
		}
		return b.String()
	case start[0] >= '0' && start[0] <= '9' && end[0] >= '0' && end[0] <= '9':
		if start[0] > end[0] {
			return ""
		}
		var parts []string
		for r := start[0]; r <= end[0]; r++ {
			parts = append(parts, string(r))
		}
		return strings.Join(parts, "")
	default:
		return ""
	}
}

func requestsTextInsertion(command string) bool {
	lowered := strings.ToLower(strings.TrimSpace(command))
	if lowered == "" {
		return false
	}
	if extractTextEntryPayload(command) != "" {
		return true
	}
	return hasTextInsertionCue(lowered)
}

func extractIntrinsicTextEntryPayload(command string) string {
	lowered := strings.ToLower(strings.TrimSpace(command))
	if lowered == "" {
		return ""
	}
	if containsKeywordAny(lowered,
		"your name", "assistant name", "vibecat name",
		"네 이름", "너 이름", "니 이름", "자기 이름", "어시스턴트 이름", "바이브캣 이름",
	) {
		return "VibeCat"
	}
	return ""
}

func requiresDerivedTextPayload(command string) bool {
	lowered := strings.ToLower(strings.TrimSpace(command))
	if lowered == "" || !requestsTextInsertion(command) {
		return false
	}
	if extractTextEntryPayload(command) != "" || extractIntrinsicTextEntryPayload(command) != "" {
		return false
	}
	return looksLikeScreenDerivedTextRequest(command, lowered)
}

func requiresExactTextPayload(command string) bool {
	if !requestsTextInsertion(command) {
		return false
	}
	return extractTextEntryPayload(command) == "" &&
		extractIntrinsicTextEntryPayload(command) == "" &&
		!requiresDerivedTextPayload(command)
}

func resolveTextEntryPayload(command, resolvedText string) string {
	if payload := extractTextEntryPayload(command); payload != "" {
		return payload
	}
	if payload := extractIntrinsicTextEntryPayload(command); payload != "" {
		return payload
	}
	return cleanTopic(resolvedText)
}

func hasVisibleTextInput(ctx navigatorContext) bool {
	if looksLikeTextInputRole(ctx.FocusedRole) {
		return true
	}
	if cleanTopic(ctx.InputFieldHint) != "" {
		return true
	}
	if ctx.VisibleInputCandidates > 0 {
		return true
	}
	lowered := strings.ToLower(ctx.AXSnapshot)
	return containsKeywordAny(lowered, "axtextfield", "axtextarea", "input:", "focused_input:")
}

func lastInputFieldDescriptorLabel(raw string) string {
	return lastInputFieldDescriptorValue(raw, "label")
}

func lastInputFieldDescriptorRole(raw string) string {
	return lastInputFieldDescriptorValue(raw, "role")
}

func lastInputFieldDescriptorValue(raw, key string) string {
	for _, part := range strings.Split(raw, "|") {
		part = strings.TrimSpace(part)
		marker := strings.ToLower(key) + "="
		if !strings.HasPrefix(strings.ToLower(part), marker) {
			continue
		}
		return strings.TrimSpace(part[len(marker):])
	}
	return ""
}

func referencesCurrentTarget(command string) bool {
	lowered := strings.ToLower(strings.TrimSpace(command))
	return containsKeywordAny(lowered, "여기", "거기", "here", "there", "current", "this")
}

func looksLikeTextInputRole(role string) bool {
	lowered := strings.ToLower(strings.TrimSpace(role))
	return containsKeywordAny(lowered, "textfield", "textarea", "searchfield")
}

func normalizeRoleToken(role string) string {
	lowered := strings.ToLower(strings.TrimSpace(role))
	switch {
	case strings.Contains(lowered, "textarea"):
		return "textarea"
	case strings.Contains(lowered, "textfield"), strings.Contains(lowered, "searchfield"):
		return "textfield"
	default:
		return cleanTopic(role)
	}
}

func hasTextInsertionCue(lowered string) bool {
	return containsKeywordAny(lowered,
		"type ", "enter ", "paste ", "fill ", "write ", "insert ",
		"입력", "붙여넣", "써", "적어", "채워", "쳐",
	)
}

func looksLikeScreenDerivedTextRequest(command, lowered string) bool {
	hasContentCue := containsKeywordAny(lowered,
		"name", "names", "text", "content", "item", "items", "label", "labels", "file", "files", "filename", "filenames",
		"log", "logs", "message", "messages", "line", "lines", "title", "titles",
		"이름", "내용", "문구", "텍스트", "항목", "파일", "파일명", "로그", "메시지", "줄", "제목",
	)
	hasScreenCue := containsKeywordAny(lowered,
		"screen", "visible", "shown", "showing", "on screen", "from screen", "here", "there", "this", "current", "listed",
		"보이는", "보여", "화면", "여기", "거기", "지금", "현재", "목록", "적혀", "되어 있", "최근",
	)
	hasExtractionCue := containsKeywordAny(lowered,
		"find", "read", "copy", "pick", "select", "list", "search", "look for",
		"찾아", "읽", "복사", "골라", "선택", "뽑", "가져와", "추려", "찾아서", "읽어서",
	)
	hasQuantityCue := containsKeywordAny(lowered,
		"one", "two", "three", "few", "first", "second", "third", "last", "recent",
		"한 개", "두 개", "세 개", "몇 개", "하나", "둘", "셋", "첫", "두 번째", "세 번째", "마지막", "최근",
	)
	return (hasContentCue && hasScreenCue) ||
		(hasExtractionCue && (hasContentCue || hasQuantityCue || hasScreenCue || referencesCurrentTarget(command)))
}

func clarificationAnswerIsSelfContained(answer string) bool {
	trimmed := strings.TrimSpace(answer)
	if trimmed == "" {
		return false
	}
	if requestsTextInsertion(trimmed) || looksLikeSystemAction(trimmed) || looksLikeDirectClick(trimmed) ||
		shouldUseDocsLookup(trimmed) || canUseTerminalCommand(trimmed, navigatorContext{}) {
		return true
	}
	intentClass, _, _ := classifyNavigatorIntent(trimmed)
	return intentClass != navigatorIntentAmbiguous
}

func mergeClarificationCommand(command, answer string, kind navigatorPromptKind) string {
	trimmedCommand := strings.TrimSpace(command)
	trimmedAnswer := strings.TrimSpace(answer)
	if trimmedCommand == "" {
		return trimmedAnswer
	}
	if trimmedAnswer == "" {
		return trimmedCommand
	}
	if kind == navigatorPromptProvideDetail && clarificationAnswerIsSelfContained(trimmedAnswer) {
		return trimmedAnswer
	}
	return strings.TrimSpace(trimmedCommand + " " + trimmedAnswer)
}

func trimTrailingTextEntryActionClause(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	lowered := strings.ToLower(trimmed)
	cut := len(trimmed)
	for _, marker := range []string{
		" 입력", "붙여넣", " type", " enter", " paste", " fill",
	} {
		if idx := strings.Index(lowered, marker); idx >= 0 && idx < cut {
			cut = idx
		}
	}
	trimmed = strings.TrimSpace(trimmed[:cut])
	trimmed = trimTrailingQuotedParticle(trimmed)
	return strings.TrimSpace(trimmed)
}

func trimTrailingQuotedParticle(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if strings.HasSuffix(trimmed, "이라고") {
		candidate := strings.TrimSpace(strings.TrimSuffix(trimmed, "이라고"))
		if endsWithHangulFinalConsonant(candidate) {
			return candidate
		}
	}
	for _, suffix := range []string{"라고", "으로", "를", "을", "이라"} {
		if strings.HasSuffix(trimmed, suffix) {
			return strings.TrimSpace(strings.TrimSuffix(trimmed, suffix))
		}
	}
	return trimmed
}

func endsWithHangulFinalConsonant(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	runes := []rune(trimmed)
	last := runes[len(runes)-1]
	if last < 0xAC00 || last > 0xD7A3 {
		return false
	}
	return (last-0xAC00)%28 != 0
}

func fallbackFieldSummary(role string, ctx navigatorContext) string {
	if label := cleanTopic(ctx.FocusedLabel); label != "" && looksLikeTextInputRole(ctx.FocusedRole) {
		return label
	}
	switch normalizeRoleToken(role) {
	case "textarea":
		return "text area"
	case "textfield":
		return "input field"
	default:
		return cleanTopic(role)
	}
}

func containsKeywordAny(text string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func cleanTopic(raw string) string {
	cleaned := strings.TrimSpace(raw)
	if cleaned == "" {
		return ""
	}
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	cleaned = strings.ReplaceAll(cleaned, "\t", " ")
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	if len(cleaned) > 120 {
		cleaned = cleaned[:120]
	}
	return strings.TrimSpace(cleaned)
}

var commandLiteralPattern = regexp.MustCompile("`([^`]+)`")

func extractCommandPayload(command string) string {
	if matches := commandLiteralPattern.FindStringSubmatch(command); len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}
	lowered := strings.ToLower(command)
	if idx := strings.Index(lowered, "run "); idx >= 0 {
		rest := strings.TrimSpace(command[idx+4:])
		if strings.Contains(rest, " ") {
			return rest
		}
	}
	return ""
}

func looksLikeShellCommand(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	for _, prefix := range []string{
		"go ", "npm ", "pnpm ", "yarn ", "swift ", "xcodebuild ", "python ", "pytest ",
		"make ", "cargo ", "git ", "rm ", "sudo ", "bash ", "sh ", "zsh ", "kubectl ",
	} {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return strings.Contains(trimmed, "./") || strings.Contains(trimmed, " --")
}

func extractClickLabel(command string) string {
	command = strings.TrimSpace(command)
	if strings.Contains(command, "\"") {
		parts := strings.Split(command, "\"")
		if len(parts) >= 3 {
			return cleanTopic(parts[1])
		}
	}
	lowered := strings.ToLower(command)
	for _, marker := range []string{"click ", "press ", "눌러 "} {
		if idx := strings.Index(lowered, marker); idx >= 0 {
			return cleanTopic(command[idx+len(marker):])
		}
	}
	return ""
}

func navigatorMessageForStep(step navigatorStep) string {
	switch step.ActionType {
	case "focus_app":
		return fmt.Sprintf("I can switch to %s now.", step.TargetApp)
	case "open_url":
		return "I can open the relevant docs now."
	case "system_action":
		return "I can apply that macOS system change now."
	case "press_ax":
		if looksLikeTextInputRole(step.TargetDescriptor.Role) {
			return "I found the input field and can focus it now."
		}
		return "I found a likely control and can act on it now."
	case "paste_text":
		if looksLikeTextInputRole(step.TargetDescriptor.Role) {
			return "I can type into that input field now."
		}
		return "I can insert the next instruction now."
	case "hotkey":
		return "I can trigger the next UI step now."
	default:
		return "I have the next step."
	}
}

func affirmativeAnswer(answer string) bool {
	lowered := strings.ToLower(strings.TrimSpace(answer))
	for _, keyword := range []string{"yes", "y", "do it", "go ahead", "proceed", "apply", "run it", "맞아", "응", "그래", "진행", "해줘", "실행"} {
		if matchesAnswerKeyword(lowered, keyword) {
			return true
		}
	}
	return false
}

func explanationAnswer(answer string) bool {
	lowered := strings.ToLower(strings.TrimSpace(answer))
	for _, keyword := range []string{"explain", "just explain", "tell me", "설명", "말만", "방법만", "안내만"} {
		if matchesAnswerKeyword(lowered, keyword) {
			return true
		}
	}
	return false
}

func matchesAnswerKeyword(answer string, keyword string) bool {
	answer = strings.TrimSpace(answer)
	keyword = strings.TrimSpace(keyword)
	if answer == "" || keyword == "" {
		return false
	}
	if answer == keyword {
		return true
	}

	for _, token := range strings.FieldsFunc(answer, func(r rune) bool {
		switch {
		case r >= 'a' && r <= 'z':
			return false
		case r >= '0' && r <= '9':
			return false
		case r >= '가' && r <= '힣':
			return false
		default:
			return true
		}
	}) {
		if token == keyword {
			return true
		}
	}

	return len(keyword) > 1 && strings.Contains(answer, keyword)
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
