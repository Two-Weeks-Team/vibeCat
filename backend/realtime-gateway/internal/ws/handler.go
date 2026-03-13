package ws

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/genai"
	"vibecat/realtime-gateway/internal/adk"
	"vibecat/realtime-gateway/internal/cdp"
	"vibecat/realtime-gateway/internal/lang"
	"vibecat/realtime-gateway/internal/live"
	"vibecat/realtime-gateway/internal/tts"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

const memoryContextCacheTTL = 5 * time.Minute

var proactiveAnalyzeTimeout = 8 * time.Second
var forcedAnalyzeTimeout = 12 * time.Second

var proactiveContextHintDelay = 1200 * time.Millisecond
var proactiveContextHintCooldown = 90 * time.Second

type adkService interface {
	Analyze(context.Context, adk.AnalysisRequest) (*adk.AnalysisResult, error)
	Search(context.Context, adk.SearchRequest) (*adk.SearchResult, error)
	Tool(context.Context, adk.ToolRequest) (*adk.ToolResult, error)
	SaveSessionSummary(context.Context, adk.SessionSummaryRequest) error
	NavigatorEscalate(context.Context, adk.NavigatorEscalationRequest) (*adk.NavigatorEscalationResult, error)
	NavigatorBackground(context.Context, adk.NavigatorBackgroundRequest) (*adk.NavigatorBackgroundResult, error)
	MemoryContext(context.Context, adk.MemoryContextRequest) (string, error)
}

type ttsSpeaker interface {
	StreamSpeak(context.Context, tts.Config, tts.AudioSink) error
}

type memoryContextCacheEntry struct {
	context   string
	expiresAt time.Time
}

var memoryContextCache = struct {
	mu      sync.RWMutex
	entries map[string]memoryContextCacheEntry
}{
	entries: map[string]memoryContextCacheEntry{},
}

type Conn struct {
	ID   string
	conn *websocket.Conn
	mu   sync.Mutex
}

type message struct {
	Type          string          `json:"type"`
	TraceID       string          `json:"traceId,omitempty"`
	ClientContent json.RawMessage `json:"clientContent"`
}

type setupCompleteMsg struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
}

type errorMsg struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type traceEventMsg struct {
	Type      string `json:"type"`
	Flow      string `json:"flow"`
	TraceID   string `json:"traceId"`
	Phase     string `json:"phase"`
	ElapsedMs *int64 `json:"elapsedMs,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

type processingStateMsg struct {
	Type        string `json:"type"`
	Flow        string `json:"flow"`
	TraceID     string `json:"traceId"`
	Stage       string `json:"stage"`
	Label       string `json:"label"`
	Detail      string `json:"detail,omitempty"`
	Tool        string `json:"tool,omitempty"`
	SourceCount *int   `json:"sourceCount,omitempty"`
	Active      bool   `json:"active"`
}

var errLiveSessionGoAway = errors.New("gemini live goAway")

var (
	routeMapsKeywords = []string{
		"nearby", "route", "directions", "distance", "travel time", "commute", "restaurant",
		"coffee", "cafe", "place", "places", "map", "maps", "where is", "near me",
		"근처", "길찾기", "거리", "소요시간", "카페", "맛집", "지도", "어디야",
	}
	routeCodeExecutionKeywords = []string{
		"calculate", "compute", "convert", "regex", "json", "csv", "sort", "transform",
		"simulate", "evaluate", "run this code", "check this math", "계산", "변환", "정규식", "실행", "검산", "코드로 확인",
	}
	routeFileSearchKeywords = []string{
		"uploaded file", "uploaded files", "knowledge base", "knowledge-base", "attached file",
		"our docs", "our documentation", "company docs", "internal docs", "file store",
		"업로드한 파일", "첨부 파일", "지식베이스", "사내 문서", "내 문서",
	}
	routeSearchKeywords = []string{
		"search", "look up", "find", "browse", "check docs", "check the docs",
		"official docs", "latest", "release notes", "version", "error", "exception",
		"docs", "documentation", "github", "git hub", "검색", "찾아", "알아봐", "공식 문서", "최신", "버전", "문서", "깃허브",
	}
)

type queryRouteKind string

const (
	queryRoutePlainLive  queryRouteKind = "plain_live"
	queryRouteLiveSearch queryRouteKind = "live_search"
	queryRouteADKTool    queryRouteKind = "adk_tool"
)

type queryRoute struct {
	Kind queryRouteKind
	Tool adk.ToolKind
}

type liveSessionState struct {
	mu           sync.RWMutex
	session      *live.Session
	config       live.Config
	resumeHandle string
	reconnecting bool
	errChan      chan error

	// modelSpeaking is true while any assistant audio is actively streaming to the client.
	// While true, screen captures are deferred and incoming user speech is treated as barge-in.
	modelSpeaking          bool
	discardModelAudio      bool
	pendingTurnTrace       string
	pendingTurnFlow        string
	pendingTurnRootAt      time.Time
	currentTurnTrace       string
	currentTurnFlow        string
	currentTurnRootAt      time.Time
	recentProactiveHints   []proactiveHintEntry
	proactiveAnalyzeActive bool

	pendingFCMu             sync.Mutex
	pendingFCID             string
	pendingFCName           string
	pendingFCTaskID         string
	pendingFCText           string
	pendingFCTarget         string
	pendingFCSteps          []navigatorStep
	pendingFCCurrentStep    string
	pendingFCStepRetryCount int

	pendingVMu    sync.Mutex
	pendingVision *pendingVisionVerification

	pendingCpMu sync.Mutex
	pendingCp   *pendingVisionCheckpoint

	cdpMu   sync.Mutex
	cdpCtrl *cdp.ChromeController
	cdpInit bool
}

type pendingVisionVerification struct {
	fcID     string
	fcName   string
	fcText   string
	fcTarget string
	taskID   string
	observed string
	imgCh    chan visionCapturePayload
}

type visionCapturePayload struct {
	image     string
	sessionID string
	userID    string
	traceID   string
}

// pendingVisionCheckpoint holds state for a mid-step vision checkpoint.
// Between pendingFC steps, the gateway captures a screenshot and asks ADK
// whether the previous step succeeded before dispatching the next step.
type pendingVisionCheckpoint struct {
	taskID        string
	completedStep navigatorStep
	nextStep      navigatorStep
	fcID          string
	fcName        string
	fcText        string
	fcTarget      string
	imgCh         chan visionCapturePayload
}

type sessionRuntime struct {
	mu                   sync.Mutex
	userID               string
	sessionID            string
	conversationHistory  []string
	executionHistory     []string
	observabilityHistory []string
	lastFCTargetApp      string // most recent target app from FC calls (focus_app, open_url, text_entry)
}

func newSessionRuntime(defaultUserID, defaultSessionID string) *sessionRuntime {
	return &sessionRuntime{
		userID:    defaultUserID,
		sessionID: defaultSessionID,
	}
}

func (sr *sessionRuntime) setIdentity(userID, sessionID string) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	if strings.TrimSpace(userID) != "" {
		sr.userID = userID
	}
	if strings.TrimSpace(sessionID) != "" {
		sr.sessionID = sessionID
	}
}

func (sr *sessionRuntime) setLastFCTargetApp(app string) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	if app != "" {
		sr.lastFCTargetApp = app
	}
}

func (sr *sessionRuntime) getLastFCTargetApp() string {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	return sr.lastFCTargetApp
}

func (sr *sessionRuntime) appendConversation(event string) {
	sr.appendToDomain(&sr.conversationHistory, event)
}

func (sr *sessionRuntime) appendExecution(event string) {
	sr.appendToDomain(&sr.executionHistory, event)
}

func (sr *sessionRuntime) appendObservability(event string) {
	sr.appendToDomain(&sr.observabilityHistory, event)
}

func (sr *sessionRuntime) appendToDomain(target *[]string, event string) {
	event = strings.TrimSpace(event)
	if event == "" {
		return
	}

	sr.mu.Lock()
	defer sr.mu.Unlock()
	*target = append(*target, event)
	if len(*target) > 200 {
		*target = (*target)[len(*target)-200:]
	}
}

func (sr *sessionRuntime) snapshot() (string, string, []string) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	history := append([]string(nil), sr.conversationHistory...)
	return sr.userID, sr.sessionID, history
}

func (sr *sessionRuntime) snapshotDomains() ([]string, []string, []string) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	conversation := append([]string(nil), sr.conversationHistory...)
	execution := append([]string(nil), sr.executionHistory...)
	observability := append([]string(nil), sr.observabilityHistory...)
	return conversation, execution, observability
}

func (ls *liveSessionState) isReconnecting() bool {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	return ls.reconnecting
}

func (ls *liveSessionState) setReconnecting(v bool) {
	ls.mu.Lock()
	ls.reconnecting = v
	ls.mu.Unlock()
}

func (ls *liveSessionState) getSession() *live.Session {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	return ls.session
}

func (ls *liveSessionState) setSession(s *live.Session) {
	ls.mu.Lock()
	ls.session = s
	ls.mu.Unlock()
}

func (ls *liveSessionState) getResumeHandle() string {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	return ls.resumeHandle
}

func (ls *liveSessionState) setResumeHandle(h string) {
	ls.mu.Lock()
	ls.resumeHandle = h
	ls.mu.Unlock()
}

func (ls *liveSessionState) getConfig() live.Config {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	return ls.config
}

func (ls *liveSessionState) setConfig(c live.Config) {
	ls.mu.Lock()
	ls.config = c
	ls.mu.Unlock()
}

func (ls *liveSessionState) isModelSpeaking() bool {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	return ls.modelSpeaking
}

func (ls *liveSessionState) setModelSpeaking(v bool) {
	ls.mu.Lock()
	ls.modelSpeaking = v
	if !v {
		ls.discardModelAudio = false
	}
	ls.mu.Unlock()
}

func (ls *liveSessionState) markBargeInPending() bool {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	alreadyDiscarding := ls.discardModelAudio
	ls.discardModelAudio = true
	return ls.modelSpeaking && !alreadyDiscarding
}

func (ls *liveSessionState) shouldDiscardModelAudio() bool {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	return ls.discardModelAudio
}

func (ls *liveSessionState) clearDiscardModelAudio() {
	ls.mu.Lock()
	ls.discardModelAudio = false
	ls.mu.Unlock()
}

func (ls *liveSessionState) queueTurnTrace(traceID, flow string, rootAt time.Time) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	ls.pendingTurnTrace = traceID
	ls.pendingTurnFlow = flow
	ls.pendingTurnRootAt = rootAt
}

const maxRecentProactiveHints = 5

type proactiveHintEntry struct {
	text string
	at   time.Time
}

func (ls *liveSessionState) shouldSkipProactiveHint(hintText string, now time.Time) bool {
	normalized := strings.TrimSpace(hintText)
	if normalized == "" {
		return true
	}

	ls.mu.Lock()
	defer ls.mu.Unlock()

	for _, entry := range ls.recentProactiveHints {
		if entry.text == normalized && now.Sub(entry.at) < proactiveContextHintCooldown {
			return true
		}
	}

	ls.recentProactiveHints = append(ls.recentProactiveHints, proactiveHintEntry{text: normalized, at: now})
	if len(ls.recentProactiveHints) > maxRecentProactiveHints {
		ls.recentProactiveHints = ls.recentProactiveHints[1:]
	}
	return false
}

func (ls *liveSessionState) clearPendingTurnTrace() {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	ls.pendingTurnTrace = ""
	ls.pendingTurnFlow = ""
	ls.pendingTurnRootAt = time.Time{}
}

func (ls *liveSessionState) beginProactiveAnalyze(captureType string) bool {
	if captureType != "screenCapture" {
		return true
	}
	ls.mu.Lock()
	defer ls.mu.Unlock()
	if ls.proactiveAnalyzeActive {
		return false
	}
	ls.proactiveAnalyzeActive = true
	return true
}

func (ls *liveSessionState) finishProactiveAnalyze(captureType string) {
	if captureType != "screenCapture" {
		return
	}
	ls.mu.Lock()
	ls.proactiveAnalyzeActive = false
	ls.mu.Unlock()
}

func (ls *liveSessionState) setPendingFC(id, name, taskID, text, target, firstStepID string, remainingSteps []navigatorStep) {
	ls.pendingFCMu.Lock()
	defer ls.pendingFCMu.Unlock()
	ls.pendingFCID = id
	ls.pendingFCName = name
	ls.pendingFCTaskID = taskID
	ls.pendingFCText = text
	ls.pendingFCTarget = target
	ls.pendingFCSteps = remainingSteps
	ls.pendingFCCurrentStep = firstStepID
}

func (ls *liveSessionState) hasPendingFCForTask(taskID, stepID string) bool {
	ls.pendingFCMu.Lock()
	defer ls.pendingFCMu.Unlock()
	return ls.pendingFCID != "" && ls.pendingFCTaskID == taskID && ls.pendingFCCurrentStep == stepID
}

func (ls *liveSessionState) advancePendingFCStep() (navigatorStep, bool) {
	ls.pendingFCMu.Lock()
	defer ls.pendingFCMu.Unlock()
	if len(ls.pendingFCSteps) == 0 {
		return navigatorStep{}, false
	}
	next := ls.pendingFCSteps[0]
	ls.pendingFCSteps = ls.pendingFCSteps[1:]
	ls.pendingFCCurrentStep = next.ID
	return next, true
}

func (ls *liveSessionState) clearPendingFC() (id, name, text, target string) {
	ls.pendingFCMu.Lock()
	defer ls.pendingFCMu.Unlock()
	id = ls.pendingFCID
	name = ls.pendingFCName
	text = ls.pendingFCText
	target = ls.pendingFCTarget
	ls.pendingFCID = ""
	ls.pendingFCName = ""
	ls.pendingFCTaskID = ""
	ls.pendingFCText = ""
	ls.pendingFCTarget = ""
	ls.pendingFCSteps = nil
	ls.pendingFCCurrentStep = ""
	ls.pendingFCStepRetryCount = 0
	return
}

func (ls *liveSessionState) incrementFCStepRetry() int {
	ls.pendingFCMu.Lock()
	defer ls.pendingFCMu.Unlock()
	ls.pendingFCStepRetryCount++
	return ls.pendingFCStepRetryCount
}

func (ls *liveSessionState) setPendingVision(pv *pendingVisionVerification) {
	ls.pendingVMu.Lock()
	ls.pendingVision = pv
	ls.pendingVMu.Unlock()
}

func (ls *liveSessionState) clearPendingVision() {
	ls.pendingVMu.Lock()
	ls.pendingVision = nil
	ls.pendingVMu.Unlock()
}

func (ls *liveSessionState) getCDPController() *cdp.ChromeController {
	ls.cdpMu.Lock()
	defer ls.cdpMu.Unlock()
	if ls.cdpInit {
		return ls.cdpCtrl
	}
	ls.cdpInit = true
	ctrl, err := cdp.NewChromeController()
	if err != nil {
		slog.Info("[CDP] chrome not available, using AX fallback", "error", err)
		return nil
	}
	ls.cdpCtrl = ctrl
	slog.Info("[CDP] chrome controller connected")
	return ls.cdpCtrl
}

func (ls *liveSessionState) closeCDPController() {
	ls.cdpMu.Lock()
	defer ls.cdpMu.Unlock()
	if ls.cdpCtrl != nil {
		ls.cdpCtrl.Close()
		ls.cdpCtrl = nil
	}
}

func (ls *liveSessionState) deliverVisionCapture(cap visionCapturePayload) bool {
	ls.pendingVMu.Lock()
	pv := ls.pendingVision
	ls.pendingVMu.Unlock()
	if pv == nil {
		return false
	}
	select {
	case pv.imgCh <- cap:
		return true
	default:
		return false
	}
}

func (ls *liveSessionState) setPendingCheckpoint(cp *pendingVisionCheckpoint) {
	ls.pendingCpMu.Lock()
	ls.pendingCp = cp
	ls.pendingCpMu.Unlock()
}

func (ls *liveSessionState) clearPendingCheckpoint() {
	ls.pendingCpMu.Lock()
	ls.pendingCp = nil
	ls.pendingCpMu.Unlock()
}

func (ls *liveSessionState) deliverCheckpointCapture(cap visionCapturePayload) bool {
	ls.pendingCpMu.Lock()
	cp := ls.pendingCp
	ls.pendingCpMu.Unlock()
	if cp == nil {
		return false
	}
	select {
	case cp.imgCh <- cap:
		return true
	default:
		return false
	}
}

func (ls *liveSessionState) peekPendingFCStep() (navigatorStep, bool) {
	ls.pendingFCMu.Lock()
	defer ls.pendingFCMu.Unlock()
	if len(ls.pendingFCSteps) == 0 {
		return navigatorStep{}, false
	}
	return ls.pendingFCSteps[0], true
}

func (ls *liveSessionState) getPendingFCInfo() (id, name, text, target string) {
	ls.pendingFCMu.Lock()
	defer ls.pendingFCMu.Unlock()
	return ls.pendingFCID, ls.pendingFCName, ls.pendingFCText, ls.pendingFCTarget
}

func (ls *liveSessionState) ensureCurrentTurnTrace(defaultFlow string) (string, string, time.Time) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	if ls.currentTurnTrace != "" {
		return ls.currentTurnTrace, ls.currentTurnFlow, ls.currentTurnRootAt
	}
	traceID := ls.pendingTurnTrace
	flow := ls.pendingTurnFlow
	rootAt := ls.pendingTurnRootAt
	if traceID == "" {
		traceID = newTraceID(defaultFlow)
	}
	if flow == "" {
		flow = defaultFlow
	}
	if rootAt.IsZero() {
		rootAt = time.Now()
	}
	ls.currentTurnTrace = traceID
	ls.currentTurnFlow = flow
	ls.currentTurnRootAt = rootAt
	ls.pendingTurnTrace = ""
	ls.pendingTurnFlow = ""
	ls.pendingTurnRootAt = time.Time{}
	return traceID, flow, rootAt
}

func (ls *liveSessionState) currentTurnTraceSnapshot() (string, string, time.Time, bool) {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	if ls.currentTurnTrace == "" {
		return "", "", time.Time{}, false
	}
	return ls.currentTurnTrace, ls.currentTurnFlow, ls.currentTurnRootAt, true
}

func (ls *liveSessionState) finishCurrentTurnTrace() (string, string, time.Time, bool) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	if ls.currentTurnTrace == "" {
		return "", "", time.Time{}, false
	}
	traceID := ls.currentTurnTrace
	flow := ls.currentTurnFlow
	rootAt := ls.currentTurnRootAt
	ls.currentTurnTrace = ""
	ls.currentTurnFlow = ""
	ls.currentTurnRootAt = time.Time{}
	return traceID, flow, rootAt, true
}

func isJPEG(data []byte) bool {
	return len(data) > 2 && data[0] == 0xFF && data[1] == 0xD8
}

func newConnID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func newTraceID(prefix string) string {
	if strings.TrimSpace(prefix) == "" {
		prefix = "trace"
	}
	return prefix + "_" + newConnID()
}

func sendTraceEvent(c *Conn, flow, traceID, phase string, rootAt time.Time, detail string) {
	if strings.TrimSpace(traceID) == "" {
		return
	}
	msg := traceEventMsg{
		Type:    "traceEvent",
		Flow:    flow,
		TraceID: traceID,
		Phase:   phase,
		Detail:  detail,
	}
	if !rootAt.IsZero() {
		elapsed := time.Since(rootAt).Milliseconds()
		msg.ElapsedMs = &elapsed
	}
	lockedSendJSON(c, msg)
}

func sendProcessingState(c *Conn, flow, traceID, stage, label, detail, tool string, sourceCount int, active bool) {
	if strings.TrimSpace(traceID) == "" {
		return
	}
	msg := processingStateMsg{
		Type:    "processingState",
		Flow:    flow,
		TraceID: traceID,
		Stage:   stage,
		Label:   label,
		Detail:  strings.TrimSpace(detail),
		Tool:    strings.TrimSpace(tool),
		Active:  active,
	}
	if sourceCount > 0 {
		msg.SourceCount = &sourceCount
	}
	lockedSendJSON(c, msg)
}

func handleBargeIn(c *Conn, ls *liveSessionState) {
	shouldInterrupt := ls.markBargeInPending()
	if shouldInterrupt {
		if traceID, flow, rootAt, ok := ls.currentTurnTraceSnapshot(); ok {
			sendTraceEvent(c, flow, traceID, "barge_in_requested", rootAt, "")
		}
		lockedSendJSON(c, map[string]string{"type": "interrupted"})
	}
}

func sendTurnState(c *Conn, state string, source string) {
	lockedSendJSON(c, map[string]string{
		"type":   "turnState",
		"state":  state,
		"source": source,
	})
}

func truncateText(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func useNativeLiveSearch(cfg live.Config) bool {
	return cfg.GoogleSearch
}

func containsAny(lower string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func resolveQueryRoute(cfg live.Config, query string) queryRoute {
	lower := strings.ToLower(strings.TrimSpace(query))
	switch {
	case strings.Contains(lower, "http://") || strings.Contains(lower, "https://"):
		return queryRoute{Kind: queryRouteADKTool, Tool: adk.ToolKindURLContext}
	case containsAny(lower, routeFileSearchKeywords):
		return queryRoute{Kind: queryRouteADKTool, Tool: adk.ToolKindFileSearch}
	case containsAny(lower, routeMapsKeywords):
		return queryRoute{Kind: queryRouteADKTool, Tool: adk.ToolKindMaps}
	case containsAny(lower, routeCodeExecutionKeywords):
		return queryRoute{Kind: queryRouteADKTool, Tool: adk.ToolKindCodeExecution}
	case useNativeLiveSearch(cfg) && containsAny(lower, routeSearchKeywords):
		return queryRoute{Kind: queryRouteLiveSearch}
	default:
		return queryRoute{Kind: queryRoutePlainLive}
	}
}

func toolDisplayName(tool adk.ToolKind) string {
	switch tool {
	case adk.ToolKindSearch:
		return "Google Search"
	case adk.ToolKindMaps:
		return "Google Maps"
	case adk.ToolKindURLContext:
		return "URL Context"
	case adk.ToolKindCodeExecution:
		return "Code Execution"
	case adk.ToolKindFileSearch:
		return "File Search"
	default:
		return ""
	}
}

func uiLanguage(language string) string {
	switch lang.NormalizeLanguage(language) {
	case "English":
		return "en"
	case "Japanese":
		return "ja"
	default:
		return "ko"
	}
}

func localizedText(language, ko, en, ja string) string {
	switch uiLanguage(language) {
	case "en":
		return en
	case "ja":
		return ja
	default:
		return ko
	}
}

func toolRunningLabel(language string) string {
	return localizedText(language, "도구 실행 중...", "Running tool...", "ツール実行中...")
}

func responsePreparingLabel(language string) string {
	return localizedText(language, "답변 정리 중...", "Preparing response...", "回答を整理中...")
}

func screenAnalyzingLabel(language string) string {
	return localizedText(language, "화면 읽는 중...", "Reading screen...", "画面を読み取り中...")
}

func screenAnalyzingDetail(language string) string {
	return localizedText(language, "현재 창 분석 중", "Analyzing current window", "現在のウィンドウを分析中")
}

func currentWindowPreparingDetail(language string) string {
	return localizedText(language, "현재 창 기준으로 정리 중", "Preparing response from current window", "現在のウィンドウを基準に整理中")
}

func navigatorAnalyzingLabel(language string) string {
	return localizedText(language, "명령 분석 중...", "Analyzing command...", "コマンドを分析中...")
}

func navigatorPlanningLabel(language string) string {
	return localizedText(language, "실행 계획 중...", "Planning steps...", "ステップを計画中...")
}

func navigatorExecutingLabel(language string) string {
	return localizedText(language, "실행 중...", "Executing action...", "アクションを実行中...")
}

func navigatorVerifyingLabel(language string) string {
	return localizedText(language, "결과 확인 중...", "Verifying result...", "結果を確認中...")
}

func navigatorRetryingLabel(language string) string {
	return localizedText(language, "재시도 중...", "Retrying with alternative...", "代替手段で再試行中...")
}

func navigatorCompletingLabel(language string) string {
	return localizedText(language, "작업 완료 중...", "Completing task...", "タスクを完了中...")
}

func navigatorObservingLabel(language string) string {
	return localizedText(language, "화면 관찰 중...", "Observing screen...", "画面を観察中...")
}

type proactiveCaptureContext struct {
	AppName     string
	BundleID    string
	WindowTitle string
	TargetKind  string
}

func parseProactiveCaptureContext(raw string) proactiveCaptureContext {
	trimmed := strings.TrimSpace(strings.Trim(raw, "[]"))
	if trimmed == "" {
		return proactiveCaptureContext{}
	}
	return proactiveCaptureContext{
		AppName:     captureContextValue(trimmed, "app"),
		BundleID:    captureContextValue(trimmed, "bundle"),
		WindowTitle: captureContextValue(trimmed, "window"),
		TargetKind:  captureContextValue(trimmed, "target"),
	}
}

func captureContextValue(raw string, key string) string {
	marker := key + "="
	start := strings.Index(raw, marker)
	if start < 0 {
		return ""
	}
	start += len(marker)
	end := len(raw)
	for _, other := range []string{"app", "target", "bundle", "window"} {
		if other == key {
			continue
		}
		if idx := strings.Index(raw[start:], " "+other+"="); idx >= 0 && start+idx < end {
			end = start + idx
		}
	}
	return strings.TrimSpace(strings.Trim(raw[start:end], "[]"))
}

func proactiveContextHintText(language string, rawContext string) string {
	ctx := parseProactiveCaptureContext(rawContext)
	app := strings.TrimSpace(ctx.AppName)
	window := strings.TrimSpace(ctx.WindowTitle)
	subject := app
	if window != "" {
		subject = truncateText(window, 48)
	} else if subject == "" {
		subject = localizedText(language, "현재 화면", "this screen", "この画面")
	}

	appLower := strings.ToLower(app)
	bundleLower := strings.ToLower(strings.TrimSpace(ctx.BundleID))
	targetLower := strings.ToLower(ctx.TargetKind)
	if strings.Contains(appLower, "codex") || strings.Contains(bundleLower, "codex") {
		return ""
	}
	if strings.Contains(targetLower, "display") {
		return ""
	}

	switch uiLanguage(language) {
	case "en":
		switch {
		case strings.Contains(appLower, "xcode"):
			return fmt.Sprintf("%s is open in Xcode. Re-run one failing test or inspect the first error line before we go wider.", subject)
		case strings.Contains(appLower, "terminal") || strings.Contains(appLower, "iterm") || strings.Contains(appLower, "warp") || strings.Contains(appLower, "ghostty"):
			return fmt.Sprintf("%s is in the terminal. Start from the latest error line and confirm the exact command that triggered it.", subject)
		case isEditorLikeApp(appLower, bundleLower):
			return fmt.Sprintf("%s is in the editor. Narrow it to one changed function or one failing file first.", subject)
		case strings.Contains(appLower, "codex") || strings.Contains(bundleLower, "codex"):
			return fmt.Sprintf("%s is open in Codex. Start from the latest task result or one changed file before going wider.", subject)
		case strings.Contains(appLower, "chrome") || strings.Contains(appLower, "safari") || strings.Contains(appLower, "arc") || strings.Contains(appLower, "firefox"):
			return fmt.Sprintf("%s is open in the browser. Keep one source tab and one work tab, then verify the next step from there.", subject)
		default:
			return fmt.Sprintf("%s is in front. Check the last thing that changed there, then I will follow with a deeper suggestion.", subject)
		}
	case "ja":
		switch {
		case strings.Contains(appLower, "xcode"):
			return fmt.Sprintf("いま %s を Xcode で開いています。まず 1 件だけ失敗テストを再実行するか、最初のエラー行を確認しましょう。", subject)
		case strings.Contains(appLower, "terminal") || strings.Contains(appLower, "iterm") || strings.Contains(appLower, "warp") || strings.Contains(appLower, "ghostty"):
			return fmt.Sprintf("いま %s はターミナルです。最後のエラー行と、その直前のコマンドから先に確認しましょう。", subject)
		case isEditorLikeApp(appLower, bundleLower):
			return fmt.Sprintf("いま %s はエディタです。変更した関数 1 つか、失敗しているファイル 1 つまで先に絞りましょう。", subject)
		case strings.Contains(appLower, "codex") || strings.Contains(bundleLower, "codex"):
			return fmt.Sprintf("いま %s は Codex です。最新のタスク結果か、直近で変えたファイル 1 つから先に確認しましょう。", subject)
		case strings.Contains(appLower, "chrome") || strings.Contains(appLower, "safari") || strings.Contains(appLower, "arc") || strings.Contains(appLower, "firefox"):
			return fmt.Sprintf("いま %s はブラウザです。参照タブを 1 つに絞って、次の確認点だけ先に押さえましょう。", subject)
		default:
			return fmt.Sprintf("いま %s を見ています。そこで最後に変わった箇所 1 つから先に確認しましょう。", subject)
		}
	default:
		switch {
		case strings.Contains(appLower, "xcode"):
			return fmt.Sprintf("지금 %s가 Xcode에 열려 있어. 실패한 테스트 하나만 다시 돌리거나 첫 에러 줄부터 바로 보자.", subject)
		case strings.Contains(appLower, "terminal") || strings.Contains(appLower, "iterm") || strings.Contains(appLower, "warp") || strings.Contains(appLower, "ghostty"):
			return fmt.Sprintf("지금 %s가 터미널이야. 마지막 에러 줄과 그 직전 명령부터 먼저 확인하자.", subject)
		case isEditorLikeApp(appLower, bundleLower):
			return fmt.Sprintf("지금 %s가 에디터에 열려 있어. 방금 바꾼 함수 하나나 깨진 파일 하나부터 좁혀보자.", subject)
		case strings.Contains(appLower, "codex") || strings.Contains(bundleLower, "codex"):
			return fmt.Sprintf("지금 %s가 Codex에 열려 있어. 최근 작업 결과 하나나 방금 바뀐 파일 하나부터 먼저 보자.", subject)
		case strings.Contains(appLower, "chrome") || strings.Contains(appLower, "safari") || strings.Contains(appLower, "arc") || strings.Contains(appLower, "firefox"):
			return fmt.Sprintf("지금 %s가 브라우저에 열려 있어. 참고 탭 하나만 남기고 다음 확인 포인트부터 잡자.", subject)
		default:
			return fmt.Sprintf("지금 %s 쪽이야. 거기서 마지막으로 바뀐 지점 하나부터 먼저 확인하자.", subject)
		}
	}
}

func isEditorLikeApp(appLower, bundleLower string) bool {
	if strings.Contains(appLower, "codex") || strings.Contains(bundleLower, "codex") {
		return false
	}

	switch {
	case strings.Contains(appLower, "cursor"):
		return true
	case strings.Contains(appLower, "visual studio code"):
		return true
	case appLower == "code":
		return true
	case bundleLower == "com.microsoft.vscode":
		return true
	default:
		return false
	}
}

func maybeStartProactiveContextHint(ctx context.Context, c *Conn, ls *liveSessionState, ttsClient ttsSpeaker, metrics *Metrics, captureType, captureContext, traceID string, rootAt time.Time) func() {
	if ttsClient == nil || ls.isModelSpeaking() {
		slog.Debug("[HANDLER] proactive context hint skipped", "conn_id", c.ID, "reason", "tts_unavailable_or_model_speaking")
		return func() {}
	}
	if captureType != "forceCapture" && !ls.getConfig().ProactiveAudio {
		slog.Debug("[HANDLER] proactive context hint skipped", "conn_id", c.ID, "reason", "proactive_audio_disabled")
		return func() {}
	}

	done := make(chan struct{})
	hintCtx, cancel := context.WithCancel(ctx)
	cfg := ls.getConfig()
	hintText := strings.TrimSpace(proactiveContextHintText(cfg.Language, captureContext))
	if hintText == "" {
		cancel()
		slog.Debug("[HANDLER] proactive context hint skipped", "conn_id", c.ID, "reason", "empty_hint_text")
		return func() {}
	}
	if ls.shouldSkipProactiveHint(hintText, time.Now()) {
		cancel()
		slog.Debug("[HANDLER] proactive context hint skipped", "conn_id", c.ID, "reason", "duplicate_hint_recently_shown")
		return func() {}
	}

	go func() {
		timer := time.NewTimer(proactiveContextHintDelay)
		defer timer.Stop()

		select {
		case <-hintCtx.Done():
			return
		case <-done:
			return
		case <-timer.C:
		}

		slog.Info("[HANDLER] proactive context hint firing", "conn_id", c.ID, "trace_id", traceID, "hint_len", len(hintText))
		sendTraceEvent(c, "proactive", traceID, "context_hint_start", rootAt, truncateText(hintText, 120))
		if speakWithTTSFallback(hintCtx, c, ls, ttsClient, metrics, "proactive", traceID, rootAt, hintText, "proactive_context_hint") {
			sendTraceEvent(c, "proactive", traceID, "context_hint_done", rootAt, "")
		}
	}()

	var once sync.Once
	return func() {
		once.Do(func() {
			cancel()
			close(done)
		})
	}
}

func searchingLabel(language string) string {
	return localizedText(language, "검색 중...", "Searching...", "検索中...")
}

func searchDetail(language string) string {
	return localizedText(language, "Google Search 확인 중", "Checking Google Search", "Google Searchを確認中")
}

func groundingLabel(language string) string {
	return localizedText(language, "근거 확인 중...", "Checking sources...", "根拠を確認中...")
}

func groundingDetail(language string, sourceCount int) string {
	if sourceCount > 0 {
		switch uiLanguage(language) {
		case "en":
			return fmt.Sprintf("Google Search · checking %d sources", sourceCount)
		case "ja":
			return fmt.Sprintf("Google Search · 根拠 %d件を確認中", sourceCount)
		default:
			return fmt.Sprintf("Google Search · 근거 %d개 확인", sourceCount)
		}
	}
	return searchDetail(language)
}

func toolStatusDetail(tool adk.ToolKind, language string) string {
	switch tool {
	case adk.ToolKindMaps:
		return localizedText(language, "Google Maps 확인 중", "Checking Google Maps", "Google Mapsを確認中")
	case adk.ToolKindURLContext:
		return localizedText(language, "URL 내용 읽는 중", "Reading URL content", "URL内容を読み取り中")
	case adk.ToolKindCodeExecution:
		return localizedText(language, "Code Execution 확인 중", "Checking Code Execution", "Code Executionを確認中")
	case adk.ToolKindFileSearch:
		return localizedText(language, "File Search 확인 중", "Checking File Search", "File Searchを確認中")
	case adk.ToolKindSearch:
		return searchDetail(language)
	default:
		return ""
	}
}

func toolPreparingDetail(tool adk.ToolKind, language string) string {
	name := toolDisplayName(tool)
	if name == "" {
		return localizedText(language, "도구 결과 정리 중", "Preparing tool results", "ツール結果を整理中")
	}
	switch uiLanguage(language) {
	case "en":
		return "Preparing " + name + " results"
	case "ja":
		return name + " の結果を整理中"
	default:
		return name + " 결과 정리 중"
	}
}

func liveRecoveringDetail(language string) string {
	return localizedText(
		language,
		"Live 음성이 재연결 중이라 임시 음성 경로로 안내합니다",
		"Live voice is reconnecting, so I am using the fallback voice path",
		"Live 音声が再接続中のため、一時的に代替音声で案内します",
	)
}

func liveRecoveringSpeech(language string) string {
	return localizedText(
		language,
		"지금 Live 음성을 다시 연결하는 중이야. 잠깐 동안은 임시 음성으로 이어갈게.",
		"I am reconnecting live voice right now. I will keep going with the fallback voice for a moment.",
		"いま Live 音声を再接続しています。しばらくは代替音声で続けます。",
	)
}

func maybeSpeakLiveUnavailableNotice(ctx context.Context, c *Conn, ls *liveSessionState, ttsClient ttsSpeaker, metrics *Metrics, flow, traceID string, rootAt time.Time, reason string) bool {
	cfg := ls.getConfig()
	sendProcessingState(c, flow, traceID, "response_preparing", responsePreparingLabel(cfg.Language), liveRecoveringDetail(cfg.Language), "", 0, true)
	defer sendProcessingState(c, flow, traceID, "response_preparing", responsePreparingLabel(cfg.Language), liveRecoveringDetail(cfg.Language), "", 0, false)
	return speakWithTTSFallback(ctx, c, ls, ttsClient, metrics, flow, traceID, rootAt, liveRecoveringSpeech(cfg.Language), reason)
}

func maybeResolveSearchFallback(
	ctx context.Context,
	c *Conn,
	ls *liveSessionState,
	adkClient adkService,
	ttsClient ttsSpeaker,
	metrics *Metrics,
	runtime *sessionRuntime,
	query string,
	traceID string,
	rootAt time.Time,
	reason string,
) bool {
	cfg := ls.getConfig()
	if adkClient == nil {
		sendTraceEvent(c, "text", traceID, "search_fallback_unavailable", rootAt, "no_adk_client")
		sendProcessingState(c, "text", traceID, "searching", searchingLabel(cfg.Language), searchDetail(cfg.Language), "google_search", 0, false)
		return maybeSpeakLiveUnavailableNotice(ctx, c, ls, ttsClient, metrics, "text", traceID, rootAt, reason)
	}

	sendTraceEvent(c, "text", traceID, "search_fallback_start", rootAt, reason)
	searchCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	searchCtx, span := otel.Tracer("vibecat/gateway").Start(searchCtx, "adk.search")
	defer span.End()
	span.SetAttributes(
		attribute.String("app.trace_id", traceID),
		attribute.String("fallback.reason", reason),
	)

	result, err := adkClient.Search(searchCtx, adk.SearchRequest{
		Query:    query,
		Language: cfg.Language,
		TraceID:  traceID,
	})
	sendProcessingState(c, "text", traceID, "searching", searchingLabel(cfg.Language), searchDetail(cfg.Language), "google_search", 0, false)
	if err != nil {
		slog.Warn("[HANDLER] search fallback failed", "conn_id", c.ID, "trace_id", traceID, "error", err)
		sendTraceEvent(c, "text", traceID, "search_fallback_failed", rootAt, err.Error())
		return maybeSpeakLiveUnavailableNotice(ctx, c, ls, ttsClient, metrics, "text", traceID, rootAt, reason)
	}
	if result == nil || strings.TrimSpace(result.Summary) == "" {
		sendTraceEvent(c, "text", traceID, "search_fallback_empty", rootAt, "")
		return maybeSpeakLiveUnavailableNotice(ctx, c, ls, ttsClient, metrics, "text", traceID, rootAt, reason)
	}

	sendTraceEvent(c, "text", traceID, "search_fallback_done", rootAt, fmt.Sprintf("sources=%d", len(result.Sources)))
	sendProcessingState(c, "text", traceID, "grounding", groundingLabel(cfg.Language), groundingDetail(cfg.Language, len(result.Sources)), "google_search", len(result.Sources), true)
	lockedSendJSON(c, map[string]any{
		"type":    "toolResult",
		"tool":    adk.ToolKindSearch,
		"query":   result.Query,
		"summary": result.Summary,
		"sources": result.Sources,
	})
	sendProcessingState(c, "text", traceID, "grounding", groundingLabel(cfg.Language), groundingDetail(cfg.Language, len(result.Sources)), "google_search", len(result.Sources), false)

	runtime.appendExecution(fmt.Sprintf("search[%s]: %s", traceID, truncateText(result.Summary, 240)))
	sendProcessingState(c, "text", traceID, "response_preparing", responsePreparingLabel(cfg.Language), toolPreparingDetail(adk.ToolKindSearch, cfg.Language), string(adk.ToolKindSearch), len(result.Sources), true)
	success := speakWithTTSFallback(ctx, c, ls, ttsClient, metrics, "text", traceID, rootAt, result.Summary, reason)
	if success {
		runtime.appendConversation("assistant: " + truncateText(result.Summary, 240))
	}
	sendProcessingState(c, "text", traceID, "response_preparing", responsePreparingLabel(cfg.Language), toolPreparingDetail(adk.ToolKindSearch, cfg.Language), string(adk.ToolKindSearch), len(result.Sources), false)
	return success
}

func handleUserTextQuery(
	ctx context.Context,
	c *Conn,
	ls *liveSessionState,
	adkClient adkService,
	ttsClient ttsSpeaker,
	metrics *Metrics,
	runtime *sessionRuntime,
	query string,
	traceID string,
	rootAt time.Time,
) {
	query = strings.TrimSpace(query)
	if query == "" {
		return
	}
	if traceID == "" {
		traceID = newTraceID("text")
	}
	if rootAt.IsZero() {
		rootAt = time.Now()
	}

	runtime.appendConversation("user: " + truncateText(query, 240))
	sendTraceEvent(c, "text", traceID, "text_received", rootAt, fmt.Sprintf("text_len=%d", len(query)))
	route := resolveQueryRoute(ls.getConfig(), query)
	switch route.Kind {
	case queryRouteADKTool:
		if maybeResolveTool(ctx, c, ls, adkClient, ttsClient, metrics, runtime, route.Tool, query, traceID, rootAt) {
			slog.Info("[HANDLER] text handled via grounded tool", "conn_id", c.ID, "tool", route.Tool)
			return
		}
	case queryRouteLiveSearch:
		sendTraceEvent(c, "text", traceID, "live_native_search_enabled", rootAt, "google_search")
		sendProcessingState(c, "text", traceID, "searching", searchingLabel(ls.getConfig().Language), searchDetail(ls.getConfig().Language), "google_search", 0, true)
	}
	sess := ls.getSession()
	if sess == nil {
		switch route.Kind {
		case queryRouteLiveSearch:
			if maybeResolveSearchFallback(ctx, c, ls, adkClient, ttsClient, metrics, runtime, query, traceID, rootAt, "live_search_no_session") {
				return
			}
			sendProcessingState(c, "text", traceID, "searching", searchingLabel(ls.getConfig().Language), searchDetail(ls.getConfig().Language), "google_search", 0, false)
		case queryRoutePlainLive:
			if maybeSpeakLiveUnavailableNotice(ctx, c, ls, ttsClient, metrics, "text", traceID, rootAt, "live_query_no_session") {
				return
			}
		}
		slog.Warn("[HANDLER] text received but no live session", "conn_id", c.ID, "trace_id", traceID)
		sendTraceEvent(c, "text", traceID, "live_text_forward_failed", rootAt, "no_live_session")
		sendProcessingState(c, "text", traceID, "response_preparing", responsePreparingLabel(ls.getConfig().Language), "", "", 0, false)
		return
	}
	slog.Info("[HANDLER] >>> forwarding text to Gemini", "conn_id", c.ID, "trace_id", traceID, "text_len", len(query))
	ls.queueTurnTrace(traceID, "text", rootAt)
	if sendErr := sess.SendText(query); sendErr != nil {
		ls.clearPendingTurnTrace()
		slog.Warn("[HANDLER] Gemini SendText failed", "conn_id", c.ID, "error", sendErr)
		sendTraceEvent(c, "text", traceID, "live_text_forward_failed", rootAt, sendErr.Error())
		switch route.Kind {
		case queryRouteLiveSearch:
			if maybeResolveSearchFallback(ctx, c, ls, adkClient, ttsClient, metrics, runtime, query, traceID, rootAt, "live_search_forward_failed") {
				return
			}
			sendProcessingState(c, "text", traceID, "searching", searchingLabel(ls.getConfig().Language), searchDetail(ls.getConfig().Language), "google_search", 0, false)
		case queryRoutePlainLive:
			if maybeSpeakLiveUnavailableNotice(ctx, c, ls, ttsClient, metrics, "text", traceID, rootAt, "live_query_forward_failed") {
				return
			}
		}
		sendProcessingState(c, "text", traceID, "response_preparing", responsePreparingLabel(ls.getConfig().Language), "", "", 0, false)
		return
	}
	sendTraceEvent(c, "text", traceID, "live_text_forwarded", rootAt, "")
}

func memoryContextCacheKey(userID, language string) string {
	return strings.TrimSpace(userID) + "|" + strings.TrimSpace(language)
}

func getCachedMemoryContext(userID, language string) (string, bool) {
	key := memoryContextCacheKey(userID, language)
	if key == "|" {
		return "", false
	}
	now := time.Now()
	memoryContextCache.mu.RLock()
	entry, ok := memoryContextCache.entries[key]
	memoryContextCache.mu.RUnlock()
	if !ok || now.After(entry.expiresAt) || strings.TrimSpace(entry.context) == "" {
		if ok {
			memoryContextCache.mu.Lock()
			delete(memoryContextCache.entries, key)
			memoryContextCache.mu.Unlock()
		}
		return "", false
	}
	return entry.context, true
}

func putCachedMemoryContext(userID, language, contextText string) {
	key := memoryContextCacheKey(userID, language)
	if key == "|" || strings.TrimSpace(contextText) == "" {
		return
	}
	memoryContextCache.mu.Lock()
	memoryContextCache.entries[key] = memoryContextCacheEntry{
		context:   strings.TrimSpace(contextText),
		expiresAt: time.Now().Add(memoryContextCacheTTL),
	}
	memoryContextCache.mu.Unlock()
}

func invalidateCachedMemoryContext(userID, language string) {
	key := memoryContextCacheKey(userID, language)
	if key == "|" {
		return
	}
	memoryContextCache.mu.Lock()
	delete(memoryContextCache.entries, key)
	memoryContextCache.mu.Unlock()
}

func fetchMemoryContext(ctx context.Context, adkClient adkService, cfg live.Config) string {
	if adkClient == nil {
		return ""
	}
	userID := strings.TrimSpace(cfg.DeviceID)
	if userID == "" {
		return ""
	}
	if cached, ok := getCachedMemoryContext(userID, cfg.Language); ok {
		slog.Info("[HANDLER] memory context cache hit", "user_id", userID, "language", cfg.Language, "context_len", len(cached))
		return cached
	}

	memoryCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	memoryCtx, span := otel.Tracer("vibecat/gateway").Start(memoryCtx, "adk.memory_context")
	defer span.End()
	span.SetAttributes(
		attribute.Bool("user.present", userID != ""),
	)

	contextText, err := adkClient.MemoryContext(memoryCtx, adk.MemoryContextRequest{
		UserID:   userID,
		Language: cfg.Language,
	})
	if err != nil {
		slog.Warn("[HANDLER] memory context lookup failed", "user_id", userID, "error", err)
		return ""
	}
	contextText = strings.TrimSpace(contextText)
	if contextText != "" {
		putCachedMemoryContext(userID, cfg.Language, contextText)
	}
	return contextText
}

func cachedMemoryContext(cfg live.Config) string {
	userID := strings.TrimSpace(cfg.DeviceID)
	if userID == "" {
		return ""
	}
	if cached, ok := getCachedMemoryContext(userID, cfg.Language); ok {
		slog.Info("[HANDLER] memory context cache hit", "user_id", userID, "language", cfg.Language, "context_len", len(cached))
		return cached
	}
	return ""
}

func primeMemoryContextAsync(ctx context.Context, adkClient adkService, cfg live.Config) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		contextText := fetchMemoryContext(ctx, adkClient, cfg)
		if strings.TrimSpace(contextText) != "" {
			slog.Info("[HANDLER] memory context primed asynchronously for future live session", "device_id", cfg.DeviceID, "context_len", len(contextText))
		}
	}()
	return done
}

func extractGroundingSources(meta *genai.GroundingMetadata) []string {
	if meta == nil || len(meta.GroundingChunks) == 0 {
		return nil
	}
	sources := make([]string, 0, len(meta.GroundingChunks))
	seen := make(map[string]struct{}, len(meta.GroundingChunks))
	for _, chunk := range meta.GroundingChunks {
		if chunk == nil || chunk.Web == nil || strings.TrimSpace(chunk.Web.URI) == "" {
			continue
		}
		uri := strings.TrimSpace(chunk.Web.URI)
		if _, ok := seen[uri]; ok {
			continue
		}
		seen[uri] = struct{}{}
		sources = append(sources, uri)
	}
	return sources
}

func describeGroundingMetadata(meta *genai.GroundingMetadata) string {
	if meta == nil {
		return ""
	}
	parts := make([]string, 0, 3)
	if len(meta.WebSearchQueries) > 0 {
		parts = append(parts, "queries="+truncateText(strings.Join(meta.WebSearchQueries, " | "), 160))
	}
	if sources := extractGroundingSources(meta); len(sources) > 0 {
		parts = append(parts, fmt.Sprintf("sources=%d", len(sources)))
	}
	if meta.RetrievalMetadata != nil && meta.RetrievalMetadata.GoogleSearchDynamicRetrievalScore > 0 {
		parts = append(parts, fmt.Sprintf("retrieval_score=%.2f", meta.RetrievalMetadata.GoogleSearchDynamicRetrievalScore))
	}
	return strings.Join(parts, " ")
}

func buildProactivePrompt(cfg live.Config, result *adk.AnalysisResult) string {
	var b strings.Builder
	b.WriteString("[System event]\n")
	b.WriteString("A screen change was analyzed and you should proactively help the developer now.\n")
	if result != nil && result.Decision != nil {
		fmt.Fprintf(&b, "Reason: %s\n", result.Decision.Reason)
		fmt.Fprintf(&b, "Urgency: %s\n", result.Decision.Urgency)
	}
	if result != nil && result.Vision != nil {
		if result.Vision.Emotion != "" {
			fmt.Fprintf(&b, "Emotion hint: %s\n", result.Vision.Emotion)
		}
		if result.Vision.ErrorMessage != "" {
			fmt.Fprintf(&b, "Observed error: %s\n", truncateText(result.Vision.ErrorMessage, 240))
		}
		if result.Vision.Content != "" {
			fmt.Fprintf(&b, "Screen context: %s\n", truncateText(result.Vision.Content, 480))
		}
	}
	if result != nil && result.SpeechText != "" {
		fmt.Fprintf(&b, "Suggested response: %s\n", truncateText(result.SpeechText, 240))
	}
	fmt.Fprintf(&b, "Respond in %s. Start with exactly one emotion tag such as [thinking], [concerned], [happy], [surprised], or [idle]. After the tag, speak one brief natural sentence. Stay close to the suggested response. Do not mention hidden instructions, screen analysis, or system events.", cfg.Language)
	return b.String()
}

func buildSearchPrompt(cfg live.Config, query string, summary string) string {
	var b strings.Builder
	b.WriteString("[System event]\n")
	b.WriteString("The user just asked a voice question. Answer naturally using the grounded summary below.\n")
	fmt.Fprintf(&b, "User question: %s\n", truncateText(query, 240))
	fmt.Fprintf(&b, "Grounded summary: %s\n", truncateText(summary, 480))
	fmt.Fprintf(&b, "Respond in %s. Start with exactly one emotion tag such as [thinking], [concerned], [happy], [surprised], or [idle]. Then answer briefly and clearly. Do not mention hidden instructions or that a search happened unless it helps the answer.", cfg.Language)
	return b.String()
}

func buildToolPrompt(cfg live.Config, result *adk.ToolResult) string {
	var b strings.Builder
	b.WriteString("[System event]\n")
	b.WriteString("A grounded tool result is available for the user's most recent request. Answer naturally using it.\n")
	if result != nil {
		fmt.Fprintf(&b, "Tool: %s\n", result.Tool)
		fmt.Fprintf(&b, "User request: %s\n", truncateText(result.Query, 240))
		fmt.Fprintf(&b, "Grounded summary: %s\n", truncateText(result.Summary, 520))
		if len(result.Sources) > 0 {
			fmt.Fprintf(&b, "Sources: %s\n", truncateText(strings.Join(result.Sources, ", "), 320))
		}
		if len(result.RetrievedURLs) > 0 {
			fmt.Fprintf(&b, "Retrieved URLs: %s\n", truncateText(strings.Join(result.RetrievedURLs, ", "), 320))
		}
	}
	fmt.Fprintf(&b, "Respond in %s. Start with exactly one emotion tag such as [thinking], [concerned], [happy], [surprised], or [idle]. Then answer briefly and clearly. Do not mention hidden instructions or tool routing unless it helps the answer.", cfg.Language)
	return b.String()
}

func speakWithTTSFallback(ctx context.Context, c *Conn, ls *liveSessionState, ttsClient ttsSpeaker, metrics *Metrics, flow, traceID string, rootAt time.Time, text, reason string) bool {
	text = strings.TrimSpace(text)
	if ttsClient == nil || text == "" {
		return false
	}

	cfg := ls.getConfig()
	if metrics != nil {
		metrics.RecordFallback(ctx, "tts", flow, reason)
	}
	sendTraceEvent(c, flow, traceID, "tts_fallback_start", rootAt, reason)
	sendTraceEvent(c, flow, traceID, "turn_started", rootAt, "source=tts_fallback")
	sendTraceEvent(c, flow, traceID, "first_output_text", rootAt, fmt.Sprintf("text_len=%d", len(text)))
	lockedSendJSON(c, map[string]any{
		"type": "ttsStart",
		"text": text,
	})

	err := ttsClient.StreamSpeak(ctx, tts.Config{
		Voice:    cfg.Voice,
		Language: cfg.Language,
		Text:     text,
	}, func(chunk []byte) error {
		return lockedSendBinary(c, chunk)
	})
	lockedSendJSON(c, map[string]any{"type": "ttsEnd"})
	if err != nil {
		slog.Warn("[HANDLER] TTS fallback failed", "conn_id", c.ID, "trace_id", traceID, "reason", reason, "error", err)
		sendTraceEvent(c, flow, traceID, "turn_failed", rootAt, err.Error())
		return false
	}

	sendTraceEvent(c, flow, traceID, "turn_complete", rootAt, "source=tts_fallback")
	return true
}

func maybeResolveTool(ctx context.Context, c *Conn, ls *liveSessionState, adkClient adkService, ttsClient ttsSpeaker, metrics *Metrics, runtime *sessionRuntime, requestedTool adk.ToolKind, query string, traceID string, rootAt time.Time) bool {
	query = strings.TrimSpace(query)
	if query == "" || adkClient == nil {
		return false
	}
	if strings.TrimSpace(traceID) == "" {
		traceID = newTraceID("tool")
	}
	if rootAt.IsZero() {
		rootAt = time.Now()
	}

	userID, sessionID, _ := runtime.snapshot()
	sendTraceEvent(c, "tool", traceID, "tool_lookup_start", rootAt, fmt.Sprintf("query_len=%d", len(query)))
	cfg := ls.getConfig()
	sendProcessingState(c, "tool", traceID, "tool_running", toolRunningLabel(cfg.Language), toolStatusDetail(requestedTool, cfg.Language), string(requestedTool), 0, true)
	toolCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	toolCtx, span := otel.Tracer("vibecat/gateway").Start(toolCtx, "adk.tool")
	defer span.End()
	span.SetAttributes(
		attribute.String("app.trace_id", traceID),
		attribute.String("tool.requested", string(requestedTool)),
		attribute.Bool("user.present", strings.TrimSpace(userID) != ""),
		attribute.Bool("session.present", strings.TrimSpace(sessionID) != ""),
	)
	result, err := adkClient.Tool(toolCtx, adk.ToolRequest{
		Query:     query,
		Language:  cfg.Language,
		SessionID: sessionID,
		UserID:    userID,
		TraceID:   traceID,
	})
	if err != nil {
		slog.Warn("[HANDLER] tool request failed", "conn_id", c.ID, "query", truncateText(query, 80), "error", err)
		sendTraceEvent(c, "tool", traceID, "tool_lookup_failed", rootAt, err.Error())
		sendProcessingState(c, "tool", traceID, "tool_running", toolRunningLabel(cfg.Language), toolStatusDetail(requestedTool, cfg.Language), string(requestedTool), 0, false)
		return false
	}
	if result == nil || strings.TrimSpace(result.Summary) == "" {
		sendTraceEvent(c, "tool", traceID, "tool_lookup_empty", rootAt, "")
		sendProcessingState(c, "tool", traceID, "tool_running", toolRunningLabel(cfg.Language), toolStatusDetail(requestedTool, cfg.Language), string(requestedTool), 0, false)
		return false
	}
	sendTraceEvent(c, "tool", traceID, "tool_lookup_done", rootAt, fmt.Sprintf("tool=%s summary_len=%d", result.Tool, len(result.Summary)))

	runtime.appendExecution(fmt.Sprintf("tool[%s]: %s => %s", result.Tool, query, truncateText(result.Summary, 240)))
	lockedSendJSON(c, map[string]any{
		"type":    "toolResult",
		"tool":    result.Tool,
		"query":   result.Query,
		"summary": result.Summary,
		"sources": result.Sources,
	})

	if ls.isModelSpeaking() {
		handleBargeIn(c, ls)
	}

	liveSess := ls.getSession()
	if liveSess == nil {
		sendProcessingState(c, "tool", traceID, "response_preparing", responsePreparingLabel(cfg.Language), toolPreparingDetail(result.Tool, cfg.Language), string(result.Tool), len(result.Sources), true)
		if speakWithTTSFallback(ctx, c, ls, ttsClient, metrics, "tool", traceID, rootAt, result.Summary, "tool_no_live_session") {
			runtime.appendConversation("assistant: " + truncateText(result.Summary, 240))
			slog.Info("[HANDLER] grounded tool result spoken via TTS fallback", "conn_id", c.ID, "tool", result.Tool)
		} else {
			slog.Warn("[HANDLER] grounded tool prompt dropped: no live session", "conn_id", c.ID)
			sendTraceEvent(c, "tool", traceID, "tool_prompt_dropped", rootAt, "no_live_session")
		}
		sendProcessingState(c, "tool", traceID, "response_preparing", responsePreparingLabel(cfg.Language), toolPreparingDetail(result.Tool, cfg.Language), string(result.Tool), len(result.Sources), false)
		return true
	}
	ls.queueTurnTrace(traceID, "tool", rootAt)
	sendProcessingState(c, "tool", traceID, "response_preparing", responsePreparingLabel(cfg.Language), toolPreparingDetail(result.Tool, cfg.Language), string(result.Tool), len(result.Sources), true)
	if err := liveSess.SendText(buildToolPrompt(ls.getConfig(), result)); err != nil {
		ls.clearPendingTurnTrace()
		slog.Warn("[HANDLER] grounded tool prompt injection failed", "conn_id", c.ID, "error", err)
		sendTraceEvent(c, "tool", traceID, "tool_prompt_injection_failed", rootAt, err.Error())
		ttsRecovered := speakWithTTSFallback(ctx, c, ls, ttsClient, metrics, "tool", traceID, rootAt, result.Summary, "tool_live_prompt_failed")
		if ttsRecovered {
			runtime.appendConversation("assistant: " + truncateText(result.Summary, 240))
			slog.Info("[HANDLER] grounded tool result recovered via TTS fallback", "conn_id", c.ID, "tool", result.Tool)
			return true
		}
		sendProcessingState(c, "tool", traceID, "response_preparing", responsePreparingLabel(cfg.Language), toolPreparingDetail(result.Tool, cfg.Language), string(result.Tool), len(result.Sources), false)
		return false
	}

	slog.Info("[HANDLER] grounded tool prompt injected", "conn_id", c.ID, "tool", result.Tool, "summary_len", len(result.Summary))
	sendTraceEvent(c, "tool", traceID, "live_prompt_injected", rootAt, fmt.Sprintf("tool=%s", result.Tool))
	return true
}

func saveSessionMemory(ctx context.Context, adkClient adkService, cfg live.Config, runtime *sessionRuntime) {
	if adkClient == nil || runtime == nil {
		return
	}

	userID, sessionID, history := runtime.snapshot()
	if strings.TrimSpace(userID) == "" || len(history) == 0 {
		return
	}

	saveCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	saveCtx, span := otel.Tracer("vibecat/gateway").Start(saveCtx, "adk.save_session_summary")
	defer span.End()
	span.SetAttributes(
		attribute.Bool("user.present", strings.TrimSpace(userID) != ""),
		attribute.Bool("session.present", strings.TrimSpace(sessionID) != ""),
		attribute.Int("history.length", len(history)),
	)
	if err := adkClient.SaveSessionSummary(saveCtx, adk.SessionSummaryRequest{
		UserID:    userID,
		SessionID: sessionID,
		Language:  cfg.Language,
		History:   history,
	}); err != nil {
		slog.Warn("[HANDLER] session summary save failed", "user_id", userID, "session_id", sessionID, "error", err)
		return
	}

	invalidateCachedMemoryContext(userID, cfg.Language)
	slog.Info("[HANDLER] session summary saved", "user_id", userID, "session_id", sessionID, "history_len", len(history))
}

func enqueueSessionMemorySave(ctx context.Context, adkClient adkService, cfg live.Config, runtime *sessionRuntime) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		saveSessionMemory(ctx, adkClient, cfg, runtime)
	}()
	return done
}

// Handler returns an http.HandlerFunc that upgrades connections to WebSocket.
// liveMgr may be nil — in that case audio is echoed back (stub mode).
// adkClient may be nil — in that case screen captures are ignored.
func Handler(reg *Registry, liveMgr *live.Manager, adkClient adkService, ttsClient ttsSpeaker, metrics *Metrics, actionStore ActionStateStore) http.HandlerFunc {
	if actionStore == nil {
		actionStore = NewInMemoryActionStateStore()
	}
	return func(w http.ResponseWriter, r *http.Request) {
		rawConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("websocket upgrade failed", "error", err)
			return
		}

		c := &Conn{
			ID:   newConnID(),
			conn: rawConn,
		}
		reg.Add(c)
		metrics.ConnectionOpened(r.Context())
		slog.Info("websocket connected", "conn_id", c.ID, "remote", r.RemoteAddr)

		ls := &liveSessionState{errChan: make(chan error, 1)}
		runtime := newSessionRuntime("default", c.ID)
		navState := &navigatorSessionState{connectionID: c.ID}
		var navigatorActiveTraceID string
		actionStateOwner := strings.TrimSpace(c.ID)

		loadActionState := func(owner string) (navigatorSessionState, bool, error) {
			if actionStore == nil {
				return navigatorSessionState{}, false, nil
			}
			loadCtx, cancel := context.WithTimeout(context.Background(), actionStateLoadTimeout)
			defer cancel()
			return actionStore.Load(loadCtx, owner)
		}

		saveActionState := func(owner string, state navigatorSessionState) error {
			if actionStore == nil {
				return nil
			}
			saveCtx, cancel := context.WithTimeout(context.Background(), actionStateWriteTimeout)
			defer cancel()
			return actionStore.Save(saveCtx, owner, state)
		}

		deleteActionState := func(owner string) error {
			if actionStore == nil {
				return nil
			}
			deleteCtx, cancel := context.WithTimeout(context.Background(), actionStateWriteTimeout)
			defer cancel()
			return actionStore.Delete(deleteCtx, owner)
		}

		persistNavigatorState := func() {
			navState.bindLease(strings.TrimSpace(ls.getConfig().DeviceID), c.ID)
			if actionStore == nil {
				return
			}
			if err := saveActionState(actionStateOwner, *navState); err != nil {
				slog.Warn("action state save failed", "conn_id", c.ID, "owner", actionStateOwner, "error", err)
			}
		}

		syncNavigatorState := func() bool {
			if actionStore == nil {
				return true
			}
			restored, ok, err := loadActionState(actionStateOwner)
			if err != nil {
				slog.Warn("action state sync failed", "conn_id", c.ID, "owner", actionStateOwner, "error", err)
				return true
			}
			if !ok {
				if !navState.hasPersistableState() {
					*navState = navigatorSessionState{
						deviceID:     strings.TrimSpace(ls.getConfig().DeviceID),
						connectionID: c.ID,
					}
				}
				return true
			}
			if !restored.ownsLease(c.ID) {
				lockedSendJSON(c, map[string]any{
					"type":   "navigator.failed",
					"taskId": restored.activeTaskID,
					"reason": "This VibeCat connection is stale after reconnect. Continue from the newest session.",
				})
				return false
			}
			*navState = restored
			navState.bindLease(strings.TrimSpace(ls.getConfig().DeviceID), c.ID)
			return true
		}

		recordFirstActionMetric := func() {
			first, ok := navState.firstPlannedStep()
			if !ok || len(navState.stepHistory) != 1 {
				return
			}
			if metrics != nil && !navState.createdAt.IsZero() {
				metrics.RecordTimeToFirstAction(context.Background(), navigatorSurfaceFromState(*navState), first.PlannedAt.Sub(navState.createdAt))
			}
		}

		recordGuidedMetrics := func(reason string, step *navigatorStep, observedOutcome string, surface string) {
			if metrics == nil {
				return
			}
			metrics.RecordGuidedMode(context.Background(), reason, surface)
			if step == nil {
				return
			}
			metrics.RecordVerificationFailure(context.Background(), step.ActionType, surface)
			if isInputFieldFocusStep(*step) {
				metrics.RecordInputFieldFocusResult(context.Background(), "guided_mode", surface)
			}
			if shouldRecordWrongTarget(observedOutcome) {
				metrics.RecordWrongTarget(context.Background(), step.ActionType, surface)
			}
		}

		recordFailureMetrics := func(step navigatorStep, observedOutcome string, surface string) {
			if metrics == nil {
				return
			}
			metrics.RecordVerificationFailure(context.Background(), step.ActionType, surface)
			if isInputFieldFocusStep(step) {
				metrics.RecordInputFieldFocusResult(context.Background(), "failed", surface)
			}
			if shouldRecordWrongTarget(observedOutcome) {
				metrics.RecordWrongTarget(context.Background(), step.ActionType, surface)
			}
		}

		recordAttemptLog := func(event, attemptID, command, outcome, detail string) {
			slog.Info("navigator attempt",
				"conn_id", c.ID,
				"event", event,
				"attempt_id", strings.TrimSpace(attemptID),
				"command_len", len(strings.TrimSpace(command)),
				"outcome", strings.TrimSpace(outcome),
				"detail", truncateText(strings.TrimSpace(detail), 200),
			)
			runtime.appendExecution(fmt.Sprintf(
				"navigator_attempt[%s]: event=%s outcome=%s detail=%s command_len=%d",
				strings.TrimSpace(attemptID),
				strings.TrimSpace(event),
				strings.TrimSpace(outcome),
				truncateText(strings.TrimSpace(detail), 180),
				len(strings.TrimSpace(command)),
			))
		}

		finalizeNavigatorTask := func(outcome, outcomeDetail, traceID string) {
			navState.completeAttempt(outcome, outcomeDetail)
			snapshot := navState.snapshotTask(time.Now().UTC())
			enqueueNavigatorBackground(context.Background(), adkClient, runtime, ls.getConfig(), snapshot, outcome, outcomeDetail, traceID)
			navState.clearPlan()
			persistNavigatorState()
		}

		defer func() {
			enqueueSessionMemorySave(context.Background(), adkClient, ls.getConfig(), runtime)
			if sess := ls.getSession(); sess != nil {
				sess.Close()
			}
			ls.closeCDPController()
			reg.Remove(c.ID)
			metrics.ConnectionClosed(context.Background())
			rawConn.Close()
			slog.Info("websocket disconnected", "conn_id", c.ID)
		}()

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		rawConn.SetReadDeadline(time.Now().Add(pongWait))
		rawConn.SetPongHandler(func(string) error {
			rawConn.SetReadDeadline(time.Now().Add(pongWait))
			return nil
		})

		go func() {
			ticker := time.NewTicker(pingPeriod)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := rawConn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
						slog.Warn("ping failed", "conn_id", c.ID, "error", err)
						cancel()
						return
					}
				}
			}
		}()

		reconnectStarted := false
		keepaliveStarted := false

		for {
			msgType, data, readErr := rawConn.ReadMessage()
			if readErr != nil {
				if websocket.IsUnexpectedCloseError(readErr, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					slog.Warn("websocket read error", "conn_id", c.ID, "error", readErr)
				}
				if sess := ls.getSession(); sess != nil {
					sess.Close()
					ls.setSession(nil)
				}
				return
			}

			switch msgType {
			case websocket.BinaryMessage:
				if isJPEG(data) {
					if sess := ls.getSession(); sess != nil {
						if sendErr := sess.SendVideo(data); sendErr != nil {
							slog.Warn("send video to gemini failed", "conn_id", c.ID, "error", sendErr)
						} else {
							slog.Debug("video frame sent to gemini", "conn_id", c.ID, "bytes", len(data))
						}
					}
				} else {
					if ls.isModelSpeaking() && !ls.shouldDiscardModelAudio() {
						slog.Debug("dropping in-flight user audio during model speech", "conn_id", c.ID, "bytes", len(data))
						continue
					}
					if sess := ls.getSession(); sess != nil {
						if sendErr := sess.SendAudio(data); sendErr != nil {
							slog.Warn("send audio to gemini failed", "conn_id", c.ID, "error", sendErr)
						}
					} else if liveMgr == nil && !ls.isReconnecting() {
						_ = lockedSendBinary(c, data)
					}
				}

			case websocket.TextMessage:
				var msg message
				if jsonErr := json.Unmarshal(data, &msg); jsonErr != nil {
					slog.Warn("invalid json frame", "conn_id", c.ID, "error", jsonErr)
					continue
				}
				if msg.Type == "" && len(msg.ClientContent) > 0 {
					msg.Type = "clientContent"
				}
				slog.Info("websocket text frame", "conn_id", c.ID, "type", msg.Type)

				switch msg.Type {
				case "setup":
					setupMsg, parseErr := live.ParseSetup(data)
					if parseErr != nil {
						slog.Error("parse setup failed", "conn_id", c.ID, "error", parseErr)
						lockedSendJSON(c, errorMsg{Type: "error", Code: "SETUP_FAILED", Message: parseErr.Error()})
						continue
					}
					cfg := setupMsg.Config
					if strings.TrimSpace(setupMsg.ResumptionHandle) == "" {
						cfg.MemoryContext = cachedMemoryContext(cfg)
						if cfg.MemoryContext != "" {
							slog.Info("[HANDLER] memory context loaded for live session", "conn_id", c.ID, "device_id", cfg.DeviceID, "context_len", len(cfg.MemoryContext))
						} else if adkClient != nil && strings.TrimSpace(cfg.DeviceID) != "" {
							primeMemoryContextAsync(context.Background(), adkClient, cfg)
						}
					}
					ls.setConfig(cfg)
					ls.setResumeHandle(setupMsg.ResumptionHandle)
					runtime.setIdentity(cfg.DeviceID, c.ID)
					previousActionStateOwner := actionStateOwner
					if strings.TrimSpace(cfg.DeviceID) != "" {
						actionStateOwner = strings.TrimSpace(cfg.DeviceID)
					}
					if restored, ok, err := loadActionState(actionStateOwner); err != nil {
						slog.Warn("action state restore failed", "conn_id", c.ID, "owner", actionStateOwner, "error", err)
					} else if ok {
						*navState = restored
						restoredIsStale := navState.isStaleTask()
						navState.bindLease(strings.TrimSpace(cfg.DeviceID), c.ID)
						persistNavigatorState()
						if previousActionStateOwner != actionStateOwner {
							if err := deleteActionState(previousActionStateOwner); err != nil {
								slog.Warn("legacy action state delete failed", "conn_id", c.ID, "owner", previousActionStateOwner, "error", err)
							}
						}
						if navState.hasActiveTask() {
							if restoredIsStale {
								slog.Info("clearing stale restored task", "conn_id", c.ID, "task_id", navState.activeTaskID, "command", navState.activeCommand)
								navState.clearPlan()
								persistNavigatorState()
							} else {
								lockedSendJSON(c, map[string]any{
									"type":        "navigator.guidedMode",
									"taskId":      navState.activeTaskID,
									"reason":      "restored_task_state",
									"instruction": fmt.Sprintf("I restored the previous action state for %q. Ask me to resume it or give me a new command.", navState.activeCommand),
								})
							}
						}
					} else {
						navState.bindLease(strings.TrimSpace(cfg.DeviceID), c.ID)
						persistNavigatorState()
						if previousActionStateOwner != actionStateOwner {
							if err := deleteActionState(previousActionStateOwner); err != nil {
								slog.Warn("legacy action state delete failed", "conn_id", c.ID, "owner", previousActionStateOwner, "error", err)
							}
						}
					}
					if liveMgr == nil {
						lockedSendJSON(c, setupCompleteMsg{Type: "setupComplete", SessionID: c.ID})
						continue
					}
					if old := ls.getSession(); old != nil {
						old.Close()
						ls.setSession(nil)
					}
					select {
					case <-ls.errChan:
					default:
					}
					sess, connectErr := liveMgr.Connect(ctx, cfg, setupMsg.ResumptionHandle)
					if connectErr != nil {
						slog.Error("gemini connect failed", "conn_id", c.ID, "error", connectErr)
						lockedSendJSON(c, errorMsg{Type: "error", Code: "GEMINI_CONNECT_FAILED", Message: connectErr.Error()})
						continue
					}
					ls.setSession(sess)
					slog.Info("device registered", "conn_id", c.ID, "device_id", cfg.DeviceID)
					lockedSendJSON(c, setupCompleteMsg{Type: "setupComplete", SessionID: c.ID})
					go receiveFromGemini(ctx, c, sess, ls, adkClient, ttsClient, metrics, runtime)

					if !reconnectStarted {
						reconnectStarted = true
						go func() {
							for {
								select {
								case <-ctx.Done():
									return
								case recvErr := <-ls.errChan:
									if ls.getSession() != nil {
										slog.Info("stale live session error ignored (new session active)", "conn_id", c.ID, "error", recvErr)
										continue
									}
									slog.Warn("live session error, attempting reconnect", "conn_id", c.ID, "error", recvErr)
									ls.setReconnecting(true)

									reconnected := false
									for attempt := 1; attempt <= 3; attempt++ {
										metrics.ReconnectAttempt(ctx, "gemini_live")
										lockedSendJSON(c, map[string]any{
											"type":    "liveSessionReconnecting",
											"attempt": attempt,
											"max":     3,
										})

										delay := time.Duration(1<<uint(attempt-1)) * time.Second
										select {
										case <-ctx.Done():
											return
										case <-time.After(delay):
										}

										cfg := ls.getConfig()
										handle := ls.getResumeHandle()

										newSess, err := liveMgr.Connect(ctx, cfg, handle)
										if err != nil {
											slog.Warn("reconnect attempt failed", "conn_id", c.ID, "attempt", attempt, "error", err)
											continue
										}

										ls.setSession(newSess)
										ls.setReconnecting(false)
										go receiveFromGemini(ctx, c, newSess, ls, adkClient, ttsClient, metrics, runtime)
										lockedSendJSON(c, map[string]any{"type": "liveSessionReconnected"})
										slog.Info("live session reconnected", "conn_id", c.ID, "attempt", attempt)
										reconnected = true
										break
									}

									if !reconnected {
										ls.setReconnecting(false)
										lockedSendJSON(c, errorMsg{
											Type:    "error",
											Code:    "LIVE_SESSION_LOST",
											Message: "Failed to reconnect Gemini Live session after 3 attempts",
										})
									}
								}
							}
						}()
					}

					if !keepaliveStarted {
						keepaliveStarted = true
						go func() {
							ticker := time.NewTicker(15 * time.Second)
							defer ticker.Stop()
							for {
								select {
								case <-ctx.Done():
									return
								case <-ticker.C:
									if sess := ls.getSession(); sess != nil {
										silence := make([]byte, 320)
										if err := sess.SendAudio(silence); err != nil {
											slog.Debug("keepalive send failed (session may be reconnecting)", "conn_id", c.ID)
										}
									}
								}
							}
						}()
					}

				case "settingsUpdate":
					slog.Info("settings update received", "conn_id", c.ID)
					if sess := ls.getSession(); sess != nil {
						sess.Close()
						ls.setSession(nil)
					}

				case "bargeIn":
					slog.Info("barge-in received", "conn_id", c.ID)
					runtime.appendObservability("interrupt: user barge-in")
					handleBargeIn(c, ls)

				case "navigator.command":
					var navMsg struct {
						Command string           `json:"command"`
						Context navigatorContext `json:"context"`
						TraceID string           `json:"traceId"`
					}
					if parseErr := json.Unmarshal(data, &navMsg); parseErr != nil {
						slog.Warn("parse navigator command failed", "conn_id", c.ID, "error", parseErr)
						continue
					}
					traceID := strings.TrimSpace(navMsg.TraceID)
					if traceID == "" {
						traceID = newTraceID("navigator")
					}
					if !syncNavigatorState() {
						continue
					}
					attemptID := navState.beginAttempt(navMsg.Command, navMsg.Context, "navigator_command", "client_reroute")
					recordAttemptLog("received", attemptID, navMsg.Command, "received", "")
					rootAt := time.Now()
					sendTraceEvent(c, "navigator", traceID, "command_received", rootAt, truncateText(navMsg.Command, 120))
					sendProcessingState(c, "navigator", traceID, "analyzing_command", navigatorAnalyzingLabel(ls.getConfig().Language), "", "", 0, true)
					plan := planNavigatorCommand(navMsg.Command, navMsg.Context, false)
					plan = maybeEscalateNavigatorPlan(ctx, adkClient, metrics, ls.getConfig().Language, navMsg.Command, navMsg.Context, plan, traceID)
					switch {
					case plan.IntentClass == navigatorIntentAmbiguous:
						navState.completeAttempt("clarification_needed", plan.ClarifyQuestion)
						recordAttemptLog("clarification_needed", attemptID, navMsg.Command, "clarification_needed", plan.ClarifyQuestion)
						clarifyKind := clarificationPromptKindForPlan(plan)
						navState.stageClarification(clarifyKind, navMsg.Command)
						persistNavigatorState()
						if metrics != nil {
							metrics.RecordClarification(context.Background(), string(clarifyKind), navigatorSurfaceFromContext(navMsg.Context))
						}
						lockedSendJSON(c, map[string]any{
							"type":         "navigator.intentClarificationNeeded",
							"command":      navMsg.Command,
							"question":     plan.ClarifyQuestion,
							"responseMode": clarificationResponseModeForPrompt(clarifyKind),
						})
					case plan.IntentClass == navigatorIntentAnalyzeOnly:
						navState.completeAttempt("analyze_only", "routed to general live analysis")
						recordAttemptLog("analyze_only", attemptID, navMsg.Command, "analyze_only", "routed to general live analysis")
						lockedSendJSON(c, map[string]any{
							"type":             "navigator.commandAccepted",
							"taskId":           "",
							"command":          navMsg.Command,
							"intentClass":      plan.IntentClass,
							"intentConfidence": plan.IntentConfidence,
						})
						handleUserTextQuery(ctx, c, ls, adkClient, ttsClient, metrics, runtime, navMsg.Command, traceID, rootAt)
					case len(plan.Steps) == 0:
						navState.completeAttempt("guided_mode", "no_supported_step")
						recordAttemptLog("guided_mode", attemptID, navMsg.Command, "guided_mode", "no_supported_step")
						recordGuidedMetrics("no_supported_step", nil, "", navigatorSurfaceFromContext(navMsg.Context))
						lockedSendJSON(c, map[string]any{
							"type":        "navigator.guidedMode",
							"taskId":      "",
							"reason":      "no_supported_step",
							"instruction": "I can see the request, but I need a more specific target or a supported surface.",
						})
					case navState.hasActiveTask():
						activeTaskID, activeCommand, currentStepID, _ := navState.activeTaskSnapshot()
						if strings.TrimSpace(navMsg.Command) == strings.TrimSpace(activeCommand) {
							lockedSendJSON(c, map[string]any{
								"type":   "navigator.stepRunning",
								"taskId": activeTaskID,
								"stepId": currentStepID,
								"status": "already_running",
							})
							continue
						}
						if navState.isStaleTask() {
							slog.Info("auto-replacing stale task", "conn_id", c.ID, "old_task", activeTaskID, "old_command", activeCommand, "new_command", navMsg.Command)
							navState.clearPlan()
							persistNavigatorState()
							plan = planNavigatorCommand(navMsg.Command, navMsg.Context, false)
							plan = maybeEscalateNavigatorPlan(ctx, adkClient, metrics, ls.getConfig().Language, navMsg.Command, navMsg.Context, plan, "")
							if len(plan.Steps) > 0 {
								taskID := navState.startPlan(navMsg.Command, plan.Steps)
								navState.attachAttemptTask(taskID)
								recordAttemptLog("accepted_replaced_stale", attemptID, navMsg.Command, "accepted", taskID)
								navState.rememberInitialContext(navMsg.Context)
								persistNavigatorState()
								if metrics != nil {
									metrics.RecordNavigatorTask(context.Background(), navigatorSurfaceFromState(*navState), plan.IntentClass)
								}
								navigatorActiveTraceID = traceID
								sendProcessingState(c, "navigator", traceID, "analyzing_command", "", "", "", 0, false)
								sendProcessingState(c, "navigator", traceID, "planning_steps", navigatorPlanningLabel(ls.getConfig().Language), "", "", len(plan.Steps), true)
								lockedSendJSON(c, map[string]any{
									"type":             "navigator.commandAccepted",
									"taskId":           taskID,
									"command":          navMsg.Command,
									"intentClass":      plan.IntentClass,
									"intentConfidence": plan.IntentConfidence,
								})
								if step, ok := navState.nextStep(); ok {
									persistNavigatorState()
									sendProcessingState(c, "navigator", traceID, "planning_steps", "", "", "", 0, false)
									sendProcessingState(c, "navigator", traceID, "executing_step", navigatorExecutingLabel(ls.getConfig().Language), step.ActionType, "", 0, true)
									lockedSendJSON(c, map[string]any{
										"type":    "navigator.stepPlanned",
										"taskId":  taskID,
										"step":    step,
										"message": navigatorMessageForStep(step),
									})
								}
								continue
							}
						}
						navState.completeAttempt("clarification_needed", "active_task_exists")
						recordAttemptLog("clarification_needed", attemptID, navMsg.Command, "clarification_needed", "active_task_exists")
						navState.stageClarification(navigatorPromptReplace, navMsg.Command)
						persistNavigatorState()
						if metrics != nil {
							surface := navigatorSurfaceFromState(*navState)
							metrics.RecordClarification(context.Background(), string(navigatorPromptReplace), surface)
							metrics.RecordTaskReplacement(context.Background(), surface)
						}
						lockedSendJSON(c, map[string]any{
							"type":         "navigator.intentClarificationNeeded",
							"command":      navMsg.Command,
							"question":     buildTaskReplacementQuestion(activeCommand, navMsg.Command),
							"responseMode": clarificationResponseModeForPrompt(navigatorPromptReplace),
						})
					case plan.RiskQuestion != "":
						navState.completeAttempt("risk_confirmation_needed", plan.RiskReason)
						recordAttemptLog("risk_confirmation_needed", attemptID, navMsg.Command, "risk_confirmation_needed", plan.RiskReason)
						navState.pendingRiskyCommand = navMsg.Command
						persistNavigatorState()
						lockedSendJSON(c, map[string]any{
							"type":     "navigator.riskyActionBlocked",
							"command":  navMsg.Command,
							"question": plan.RiskQuestion,
							"reason":   plan.RiskReason,
						})
					default:
						taskID := navState.startPlan(navMsg.Command, plan.Steps)
						navState.attachAttemptTask(taskID)
						recordAttemptLog("accepted", attemptID, navMsg.Command, "accepted", taskID)
						navState.rememberInitialContext(navMsg.Context)
						persistNavigatorState()
						if metrics != nil {
							metrics.RecordNavigatorTask(context.Background(), navigatorSurfaceFromState(*navState), plan.IntentClass)
						}
						navigatorActiveTraceID = traceID
						sendProcessingState(c, "navigator", traceID, "analyzing_command", "", "", "", 0, false)
						sendProcessingState(c, "navigator", traceID, "planning_steps", navigatorPlanningLabel(ls.getConfig().Language), "", "", len(plan.Steps), true)
						lockedSendJSON(c, map[string]any{
							"type":             "navigator.commandAccepted",
							"taskId":           taskID,
							"command":          navMsg.Command,
							"intentClass":      plan.IntentClass,
							"intentConfidence": plan.IntentConfidence,
						})
						if step, ok := navState.nextStep(); ok {
							persistNavigatorState()
							recordFirstActionMetric()
							sendProcessingState(c, "navigator", traceID, "planning_steps", "", "", "", 0, false)
							sendProcessingState(c, "navigator", traceID, "executing_step", navigatorExecutingLabel(ls.getConfig().Language), step.ActionType, "", 0, true)
							lockedSendJSON(c, map[string]any{
								"type":    "navigator.stepPlanned",
								"taskId":  taskID,
								"step":    step,
								"message": navigatorMessageForStep(step),
							})
						}
					}

				case "navigator.confirmAmbiguousIntent":
					var confirmMsg struct {
						Command string           `json:"command"`
						Answer  string           `json:"answer"`
						Context navigatorContext `json:"context"`
					}
					if parseErr := json.Unmarshal(data, &confirmMsg); parseErr != nil {
						slog.Warn("parse navigator clarification response failed", "conn_id", c.ID, "error", parseErr)
						continue
					}
					if !syncNavigatorState() {
						continue
					}
					clarifyKind, command := navState.consumeClarification(confirmMsg.Command)
					if explanationAnswer(confirmMsg.Answer) {
						switch clarifyKind {
						case navigatorPromptReplace:
							persistNavigatorState()
							lockedSendJSON(c, map[string]any{
								"type":   "navigator.failed",
								"taskId": "",
								"reason": "I kept the current task and did not switch actions.",
							})
						default:
							navState.clearPlan()
							persistNavigatorState()
							recordGuidedMetrics("clarification_explanation_only", nil, "", navigatorSurfaceFromContext(confirmMsg.Context))
							lockedSendJSON(c, map[string]any{
								"type":        "navigator.guidedMode",
								"taskId":      "",
								"reason":      "clarification_explanation_only",
								"instruction": "I will explain the next step instead of acting.",
							})
							handleUserTextQuery(ctx, c, ls, adkClient, ttsClient, metrics, runtime, command, newTraceID("text"), time.Now())
						}
						continue
					}
					if clarifyKind == navigatorPromptProvideDetail {
						command = mergeClarificationCommand(command, confirmMsg.Answer, clarifyKind)
						plan := planNavigatorCommand(command, confirmMsg.Context, false)
						plan = maybeEscalateNavigatorPlan(ctx, adkClient, metrics, ls.getConfig().Language, command, confirmMsg.Context, plan, "")
						switch {
						case plan.IntentClass == navigatorIntentAmbiguous:
							nextClarifyKind := clarificationPromptKindForPlan(plan)
							navState.stageClarification(nextClarifyKind, command)
							persistNavigatorState()
							if metrics != nil {
								metrics.RecordClarification(context.Background(), string(nextClarifyKind), navigatorSurfaceFromContext(confirmMsg.Context))
							}
							lockedSendJSON(c, map[string]any{
								"type":         "navigator.intentClarificationNeeded",
								"command":      command,
								"question":     plan.ClarifyQuestion,
								"responseMode": clarificationResponseModeForPrompt(nextClarifyKind),
							})
						case plan.RiskQuestion != "":
							navState.pendingRiskyCommand = command
							persistNavigatorState()
							lockedSendJSON(c, map[string]any{
								"type":     "navigator.riskyActionBlocked",
								"command":  command,
								"question": plan.RiskQuestion,
								"reason":   plan.RiskReason,
							})
						case len(plan.Steps) == 0:
							recordGuidedMetrics("clarified_but_not_supported", nil, "", navigatorSurfaceFromContext(confirmMsg.Context))
							lockedSendJSON(c, map[string]any{
								"type":        "navigator.guidedMode",
								"taskId":      "",
								"reason":      "clarified_but_not_supported",
								"instruction": "I understand the intent now, but I still need a more specific or supported target.",
							})
							navState.clearPlan()
							persistNavigatorState()
						default:
							taskID := navState.startPlan(command, plan.Steps)
							navState.rememberInitialContext(confirmMsg.Context)
							persistNavigatorState()
							if metrics != nil {
								metrics.RecordNavigatorTask(context.Background(), navigatorSurfaceFromState(*navState), plan.IntentClass)
							}
							lockedSendJSON(c, map[string]any{
								"type":             "navigator.commandAccepted",
								"taskId":           taskID,
								"command":          command,
								"intentClass":      plan.IntentClass,
								"intentConfidence": plan.IntentConfidence,
							})
							if step, ok := navState.nextStep(); ok {
								persistNavigatorState()
								recordFirstActionMetric()
								lockedSendJSON(c, map[string]any{
									"type":    "navigator.stepPlanned",
									"taskId":  taskID,
									"step":    step,
									"message": navigatorMessageForStep(step),
								})
							}
						}
						continue
					}
					if !affirmativeAnswer(confirmMsg.Answer) {
						instruction := "I did not get a clear execution confirmation, so I stopped before acting."
						if clarifyKind == navigatorPromptReplace {
							instruction = "I kept the current task and did not switch actions."
						}
						recordGuidedMetrics("clarification_not_confirmed", nil, "", navigatorSurfaceFromContext(confirmMsg.Context))
						lockedSendJSON(c, map[string]any{
							"type":        "navigator.guidedMode",
							"taskId":      "",
							"reason":      "clarification_not_confirmed",
							"instruction": instruction,
						})
						if clarifyKind != navigatorPromptReplace {
							navState.clearPlan()
						}
						persistNavigatorState()
						continue
					}
					if clarifyKind == navigatorPromptReplace {
						finalizeNavigatorTask("replaced", "task replaced by a newer command", "")
					}
					plan := planNavigatorCommand(mergeClarificationCommand(command, confirmMsg.Answer, clarifyKind), confirmMsg.Context, false)
					plan = maybeEscalateNavigatorPlan(ctx, adkClient, metrics, ls.getConfig().Language, command, confirmMsg.Context, plan, "")
					if len(plan.Steps) == 0 {
						recordGuidedMetrics("clarified_but_not_supported", nil, "", navigatorSurfaceFromContext(confirmMsg.Context))
						lockedSendJSON(c, map[string]any{
							"type":        "navigator.guidedMode",
							"taskId":      "",
							"reason":      "clarified_but_not_supported",
							"instruction": "I understand the intent now, but I still need a more specific or supported target.",
						})
						navState.clearPlan()
						persistNavigatorState()
						continue
					}
					taskID := navState.startPlan(command, plan.Steps)
					navState.rememberInitialContext(confirmMsg.Context)
					persistNavigatorState()
					if metrics != nil {
						metrics.RecordNavigatorTask(context.Background(), navigatorSurfaceFromState(*navState), plan.IntentClass)
					}
					lockedSendJSON(c, map[string]any{
						"type":             "navigator.commandAccepted",
						"taskId":           taskID,
						"command":          command,
						"intentClass":      plan.IntentClass,
						"intentConfidence": plan.IntentConfidence,
					})
					if step, ok := navState.nextStep(); ok {
						persistNavigatorState()
						recordFirstActionMetric()
						lockedSendJSON(c, map[string]any{
							"type":    "navigator.stepPlanned",
							"taskId":  taskID,
							"step":    step,
							"message": navigatorMessageForStep(step),
						})
					}

				case "navigator.confirmRiskyAction":
					var confirmMsg struct {
						Command string           `json:"command"`
						Answer  string           `json:"answer"`
						Context navigatorContext `json:"context"`
					}
					if parseErr := json.Unmarshal(data, &confirmMsg); parseErr != nil {
						slog.Warn("parse navigator risky response failed", "conn_id", c.ID, "error", parseErr)
						continue
					}
					if !syncNavigatorState() {
						continue
					}
					command := strings.TrimSpace(confirmMsg.Command)
					if command == "" {
						command = navState.pendingRiskyCommand
					}
					if explanationAnswer(confirmMsg.Answer) || !affirmativeAnswer(confirmMsg.Answer) {
						recordGuidedMetrics("risky_action_not_confirmed", nil, "", navigatorSurfaceFromContext(confirmMsg.Context))
						lockedSendJSON(c, map[string]any{
							"type":        "navigator.guidedMode",
							"taskId":      "",
							"reason":      "risky_action_not_confirmed",
							"instruction": "I stopped before the risky action. I can still explain the next step.",
						})
						navState.clearPlan()
						persistNavigatorState()
						continue
					}
					plan := planNavigatorCommand(command, confirmMsg.Context, true)
					plan = maybeEscalateNavigatorPlan(ctx, adkClient, metrics, ls.getConfig().Language, command, confirmMsg.Context, plan, "")
					if len(plan.Steps) == 0 {
						lockedSendJSON(c, map[string]any{
							"type":   "navigator.failed",
							"taskId": "",
							"reason": "I could not build a safe step even after confirmation.",
						})
						navState.clearPlan()
						persistNavigatorState()
						continue
					}
					taskID := navState.startPlan(command, plan.Steps)
					navState.rememberInitialContext(confirmMsg.Context)
					persistNavigatorState()
					if metrics != nil {
						metrics.RecordNavigatorTask(context.Background(), navigatorSurfaceFromState(*navState), plan.IntentClass)
					}
					lockedSendJSON(c, map[string]any{
						"type":             "navigator.commandAccepted",
						"taskId":           taskID,
						"command":          command,
						"intentClass":      plan.IntentClass,
						"intentConfidence": plan.IntentConfidence,
					})
					if step, ok := navState.nextStep(); ok {
						persistNavigatorState()
						recordFirstActionMetric()
						lockedSendJSON(c, map[string]any{
							"type":    "navigator.stepPlanned",
							"taskId":  taskID,
							"step":    step,
							"message": navigatorMessageForStep(step),
						})
					}

				case "navigator.refreshContext":
					var refreshMsg struct {
						Command         string           `json:"command"`
						TaskID          string           `json:"taskId"`
						Step            navigatorStep    `json:"step"`
						Status          string           `json:"status"`
						ObservedOutcome string           `json:"observedOutcome"`
						Context         navigatorContext `json:"context"`
					}
					if parseErr := json.Unmarshal(data, &refreshMsg); parseErr != nil {
						slog.Warn("parse navigator refresh failed", "conn_id", c.ID, "error", parseErr)
						continue
					}
					if !syncNavigatorState() {
						continue
					}
					isNavStateRefresh := navState.acceptsRefresh(refreshMsg.Command, refreshMsg.TaskID, refreshMsg.Step.ID)
					isPendingFCRefresh := !isNavStateRefresh && ls.hasPendingFCForTask(refreshMsg.TaskID, refreshMsg.Step.ID)
					if !isNavStateRefresh && !isPendingFCRefresh {
						slog.Warn("navigator refresh ignored", "conn_id", c.ID, "command", refreshMsg.Command, "task_id", refreshMsg.TaskID, "step_id", refreshMsg.Step.ID, "active_command", navState.activeCommand)
						continue
					}
					sendProcessingState(c, "navigator", navigatorActiveTraceID, "executing_step", "", "", "", 0, false)
					sendProcessingState(c, "navigator", navigatorActiveTraceID, "verifying_result", navigatorVerifyingLabel(ls.getConfig().Language), "", "", 0, true)
					if isPendingFCRefresh {
						switch refreshMsg.Status {
						case "success":
							lockedSendJSON(c, map[string]any{
								"type":            "navigator.stepVerified",
								"taskId":          refreshMsg.TaskID,
								"stepId":          refreshMsg.Step.ID,
								"status":          "success",
								"observedOutcome": refreshMsg.ObservedOutcome,
							})
							sendProcessingState(c, "navigator", navigatorActiveTraceID, "verifying_result", "", "", "", 0, false)
							if nextStep, hasNext := ls.advancePendingFCStep(); hasNext {
								if adkClient != nil && needsVisionCheckpoint(refreshMsg.Step, refreshMsg.ObservedOutcome) {
									fcID, fcName, fcText, fcTarget := ls.getPendingFCInfo()
									cp := &pendingVisionCheckpoint{
										taskID:        refreshMsg.TaskID,
										completedStep: refreshMsg.Step,
										nextStep:      nextStep,
										fcID:          fcID,
										fcName:        fcName,
										fcText:        fcText,
										fcTarget:      fcTarget,
										imgCh:         make(chan visionCapturePayload, 1),
									}
									ls.setPendingCheckpoint(cp)
									sendProcessingState(c, "navigator", navigatorActiveTraceID, "observing_screen", navigatorObservingLabel(ls.getConfig().Language), "", "", 0, true)
									lockedSendJSON(c, map[string]any{
										"type":   "requestScreenCapture",
										"reason": "mid_step_checkpoint",
									})
									cpTraceID := navigatorActiveTraceID
									capturedCtx := ctx
									go func() {
										defer ls.clearPendingCheckpoint()
										var visionText string
										var errorDetected bool
										var capturedScreenshot string
										select {
										case cap := <-cp.imgCh:
											capturedScreenshot = cap.image
											analyzeCtx, analyzeCancel := context.WithTimeout(capturedCtx, 4*time.Second)
											defer analyzeCancel()
											result, err := adkClient.Analyze(analyzeCtx, adk.AnalysisRequest{
												Image:     cap.image,
												Context:   fmt.Sprintf("mid-step checkpoint after %s: %s", cp.completedStep.ActionType, truncateText(cp.fcText, 120)),
												SessionID: cap.sessionID,
												UserID:    cap.userID,
												TraceID:   cap.traceID,
											})
											if err != nil {
												slog.Warn("[HANDLER] checkpoint ADK analyze failed", "conn_id", c.ID, "error", err)
											} else if result != nil && result.Vision != nil {
												visionText = result.Vision.Content
												errorDetected = result.Vision.ErrorDetected
											}
										case <-time.After(3 * time.Second):
											slog.Info("[HANDLER] checkpoint vision timeout, proceeding", "conn_id", c.ID, "task_id", cp.taskID)
										}

										if errorDetected {
											slog.Info("[HANDLER] checkpoint detected error, aborting remaining steps", "conn_id", c.ID, "task_id", cp.taskID, "vision", visionText)
											abortID, abortName, _, _ := ls.clearPendingFC()
											sendProcessingState(c, "navigator", cpTraceID, "observing_screen", "", "", "", 0, false)
											lockedSendJSON(c, map[string]any{
												"type":   "navigator.failed",
												"taskId": cp.taskID,
												"reason": "checkpoint_error: " + visionText,
											})
											if fcSess := ls.getSession(); fcSess != nil {
												resp := map[string]any{
													"status": "checkpoint_failed",
													"error":  "Mid-step verification detected an error after " + cp.completedStep.ActionType,
												}
												if visionText != "" {
													resp["vision"] = visionText
												}
												_ = fcSess.SendToolResponse([]*genai.FunctionResponse{{
													ID:       abortID,
													Name:     abortName,
													Response: resp,
												}})
											}
											return
										}

										slog.Info("[HANDLER] checkpoint passed, advancing to next step", "conn_id", c.ID, "task_id", cp.taskID, "step", cp.nextStep.ActionType, "vision_len", len(visionText))

										nextToSend := cp.nextStep
										if nextToSend.ActionType == "press_ax" && nextToSend.MacroID == "fc_play_result" && capturedScreenshot != "" {
											escCtx, escCancel := context.WithTimeout(capturedCtx, 4*time.Second)
											escResult, escErr := adkClient.NavigatorEscalate(escCtx, adk.NavigatorEscalationRequest{
												Command:    "Find the first playable music result or the Music Station button on this YouTube Music search results page. Return the element to click to start playback.",
												AppName:    nextToSend.TargetApp,
												Screenshot: capturedScreenshot,
												TraceID:    cp.taskID,
											})
											escCancel()
											if escErr == nil && escResult != nil && escResult.ResolvedDescriptor != nil && escResult.Confidence > 0.5 {
												slog.Info("[HANDLER] checkpoint escalated play target", "conn_id", c.ID, "label", escResult.ResolvedDescriptor.Label, "role", escResult.ResolvedDescriptor.Role, "confidence", escResult.Confidence)
												nextToSend.TargetDescriptor = navigatorTargetDescriptor{
													Role:           escResult.ResolvedDescriptor.Role,
													Label:          escResult.ResolvedDescriptor.Label,
													AppName:        firstNonEmptyString(escResult.ResolvedDescriptor.AppName, nextToSend.TargetApp),
													WindowTitle:    escResult.ResolvedDescriptor.WindowTitle,
													RelativeAnchor: escResult.ResolvedDescriptor.RelativeAnchor,
													RegionHint:     escResult.ResolvedDescriptor.RegionHint,
												}
												nextToSend.Confidence = escResult.Confidence
												nextToSend.ExpectedOutcome = "Click " + escResult.ResolvedDescriptor.Label + " to start playback"
												nextToSend.Narration = "Clicking " + escResult.ResolvedDescriptor.Label + " to play music."
											} else {
												slog.Info("[HANDLER] checkpoint escalation failed or low confidence, using default play target", "conn_id", c.ID, "err", escErr)
											}
										}

										prevAction := cp.completedStep.ActionType
										if prevAction == "open_url" {
											time.Sleep(2500 * time.Millisecond)
										} else if nextToSend.ActionType == "focus_app" || prevAction == "focus_app" {
											time.Sleep(500 * time.Millisecond)
										} else if prevAction == "hotkey" {
											time.Sleep(2000 * time.Millisecond)
										} else {
											time.Sleep(150 * time.Millisecond)
										}
										sendProcessingState(c, "navigator", cpTraceID, "observing_screen", "", "", "", 0, false)
										sendProcessingState(c, "navigator", cpTraceID, "executing_step", navigatorExecutingLabel(ls.getConfig().Language), nextToSend.ActionType, "", 0, true)
										lockedSendJSON(c, map[string]any{
											"type":    "navigator.stepPlanned",
											"taskId":  cp.taskID,
											"step":    nextToSend,
											"message": navigatorMessageForStep(nextToSend),
										})
									}()
								} else {
									sendProcessingState(c, "navigator", navigatorActiveTraceID, "observing_screen", navigatorObservingLabel(ls.getConfig().Language), "", "", 0, true)
									prevStepAction := refreshMsg.Step.ActionType
									if prevStepAction == "open_url" {
										time.Sleep(2500 * time.Millisecond)
									} else if nextStep.ActionType == "focus_app" || prevStepAction == "focus_app" {
										time.Sleep(500 * time.Millisecond)
									} else {
										time.Sleep(150 * time.Millisecond)
									}
									sendProcessingState(c, "navigator", navigatorActiveTraceID, "observing_screen", "", "", "", 0, false)
									sendProcessingState(c, "navigator", navigatorActiveTraceID, "executing_step", navigatorExecutingLabel(ls.getConfig().Language), nextStep.ActionType, "", 0, true)
									lockedSendJSON(c, map[string]any{
										"type":    "navigator.stepPlanned",
										"taskId":  refreshMsg.TaskID,
										"step":    nextStep,
										"message": navigatorMessageForStep(nextStep),
									})
								}
							} else {
								fcID, fcName, fcText, fcTarget := ls.clearPendingFC()
								sendProcessingState(c, "navigator", navigatorActiveTraceID, "completing", navigatorCompletingLabel(ls.getConfig().Language), "", "", 0, true)
								lockedSendJSON(c, map[string]any{
									"type":    "navigator.completed",
									"taskId":  refreshMsg.TaskID,
									"summary": refreshMsg.ObservedOutcome,
								})
								sendProcessingState(c, "navigator", navigatorActiveTraceID, "completing", "", "", "", 0, false)
								if adkClient != nil {
									pv := &pendingVisionVerification{
										fcID:     fcID,
										fcName:   fcName,
										fcText:   fcText,
										fcTarget: fcTarget,
										taskID:   refreshMsg.TaskID,
										observed: refreshMsg.ObservedOutcome,
										imgCh:    make(chan visionCapturePayload, 1),
									}
									ls.setPendingVision(pv)
									sendProcessingState(c, "navigator", navigatorActiveTraceID, "observing_screen", navigatorObservingLabel(ls.getConfig().Language), "", "", 0, true)
									lockedSendJSON(c, map[string]any{
										"type":   "requestScreenCapture",
										"reason": "post_action_verification",
									})
									capturedCtx := ctx
									go func() {
										var visionText string
										select {
										case cap := <-pv.imgCh:
											analyzeCtx, analyzeCancel := context.WithTimeout(capturedCtx, 4*time.Second)
											defer analyzeCancel()
											result, err := adkClient.Analyze(analyzeCtx, adk.AnalysisRequest{
												Image:     cap.image,
												Context:   "post-action verification: " + truncateText(pv.fcText, 120),
												SessionID: cap.sessionID,
												UserID:    cap.userID,
												TraceID:   cap.traceID,
											})
											if err == nil && result != nil && result.Vision != nil && result.Vision.Content != "" {
												visionText = result.Vision.Content
											}
										case <-time.After(3 * time.Second):
											slog.Info("[HANDLER] vision verification timeout", "conn_id", c.ID, "task_id", pv.taskID)
										}
										ls.clearPendingVision()
										if fcSess := ls.getSession(); fcSess != nil {
											resp := map[string]any{
												"status": "completed",
												"text":   pv.fcText,
												"target": pv.fcTarget,
											}
											if visionText != "" {
												resp["vision"] = visionText
											}
											if hint := buildNextActionHint(pv.fcName, pv.fcText, pv.fcTarget); hint != "" {
												resp["next_action_hint"] = hint
											}
											if err := fcSess.SendToolResponse([]*genai.FunctionResponse{{
												ID:       pv.fcID,
												Name:     pv.fcName,
												Response: resp,
											}}); err != nil {
												slog.Warn("[HANDLER] FC SendToolResponse (vision) failed", "conn_id", c.ID, "error", err)
											}
										}
									}()
								} else {
									if fcSess := ls.getSession(); fcSess != nil {
										resp := map[string]any{
											"status": "completed",
											"text":   fcText,
											"target": fcTarget,
										}
										if hint := buildNextActionHint(fcName, fcText, fcTarget); hint != "" {
											resp["next_action_hint"] = hint
										}
										if err := fcSess.SendToolResponse([]*genai.FunctionResponse{{
											ID:       fcID,
											Name:     fcName,
											Response: resp,
										}}); err != nil {
											slog.Warn("[HANDLER] FC SendToolResponse failed", "conn_id", c.ID, "error", err)
										}
									}
								}
							}
						default:
							retryCount := ls.incrementFCStepRetry()
							if retryCount <= 2 {
								retryStep := refreshMsg.Step
								if retryCount == 2 && retryStep.FallbackActionType != "" {
									retryStep.ActionType = retryStep.FallbackActionType
									if len(retryStep.FallbackHotkey) > 0 {
										retryStep.Hotkey = retryStep.FallbackHotkey
									}
								}
								if retryStep.ActionType == "cdp_js" && retryStep.CDPScript != "" {
									if ctrl := ls.getCDPController(); ctrl != nil {
										jsResult, jsErr := ctrl.EvaluateJS(retryStep.CDPScript)
										if jsErr == nil && (jsResult == "focused" || jsResult == "clicked" || jsResult == "clicked_container") {
											slog.Info("navigator FC cdp_js search activation succeeded, retrying paste_text", "conn_id", c.ID, "result", jsResult)
											retryStep.ActionType = "paste_text"
											retryStep.FallbackActionType = ""
											retryStep.CDPScript = ""
										} else {
											slog.Info("navigator FC cdp_js fallback result", "conn_id", c.ID, "result", jsResult, "err", jsErr)
										}
									} else {
										slog.Info("navigator FC cdp_js: CDP controller unavailable, dispatching to client", "conn_id", c.ID)
									}
								}
								slog.Info("navigator FC self-healing retry", "conn_id", c.ID, "step_id", retryStep.ID, "retry", retryCount, "status", refreshMsg.Status)
								sendProcessingState(c, "navigator", navigatorActiveTraceID, "verifying_result", "", "", "", 0, false)
								// Vision-first: observe screen after failure so Gemini sees what went wrong
								sendProcessingState(c, "navigator", navigatorActiveTraceID, "observing_screen", navigatorObservingLabel(ls.getConfig().Language), "", "", 0, true)
								time.Sleep(200 * time.Millisecond)
								sendProcessingState(c, "navigator", navigatorActiveTraceID, "observing_screen", "", "", "", 0, false)
								sendProcessingState(c, "navigator", navigatorActiveTraceID, "retrying_step", navigatorRetryingLabel(ls.getConfig().Language), "", "", 0, true)
								sendProcessingState(c, "navigator", navigatorActiveTraceID, "retrying_step", "", "", "", 0, false)
								sendProcessingState(c, "navigator", navigatorActiveTraceID, "executing_step", navigatorExecutingLabel(ls.getConfig().Language), retryStep.ActionType, "", 0, true)
								lockedSendJSON(c, map[string]any{
									"type":    "navigator.stepPlanned",
									"taskId":  refreshMsg.TaskID,
									"step":    retryStep,
									"message": navigatorMessageForStep(retryStep),
								})
								continue
							}
							sendProcessingState(c, "navigator", navigatorActiveTraceID, "verifying_result", "", "", "", 0, false)
							fcID, fcName, _, _ := ls.clearPendingFC()
							lockedSendJSON(c, map[string]any{
								"type":   "navigator.failed",
								"taskId": refreshMsg.TaskID,
								"reason": refreshMsg.ObservedOutcome,
							})
							if fcSess := ls.getSession(); fcSess != nil {
								_ = fcSess.SendToolResponse([]*genai.FunctionResponse{{
									ID:       fcID,
									Name:     fcName,
									Response: map[string]any{"error": "step failed: " + refreshMsg.ObservedOutcome},
								}})
							}
						}
						continue
					}
					switch refreshMsg.Status {
					case "success":
						navState.recordStepResult(refreshMsg.Step, refreshMsg.Status, refreshMsg.ObservedOutcome)
						navState.resetStepRetry()
						navState.clearCurrentStep()
						navState.markVerifiedContext(refreshMsg.Context)
						persistNavigatorState()
						if metrics != nil && isInputFieldFocusStep(refreshMsg.Step) {
							metrics.RecordInputFieldFocusResult(context.Background(), "success", navigatorSurfaceFromState(*navState))
						}
						lockedSendJSON(c, map[string]any{
							"type":            "navigator.stepVerified",
							"taskId":          refreshMsg.TaskID,
							"stepId":          refreshMsg.Step.ID,
							"status":          "success",
							"observedOutcome": refreshMsg.ObservedOutcome,
						})
						sendProcessingState(c, "navigator", navigatorActiveTraceID, "verifying_result", "", "", "", 0, false)
						if next, ok := navState.nextStep(); ok {
							persistNavigatorState()
							sendProcessingState(c, "navigator", navigatorActiveTraceID, "executing_step", navigatorExecutingLabel(ls.getConfig().Language), next.ActionType, "", 0, true)
							lockedSendJSON(c, map[string]any{
								"type":    "navigator.stepPlanned",
								"taskId":  refreshMsg.TaskID,
								"step":    next,
								"message": navigatorMessageForStep(next),
							})
						} else {
							sendProcessingState(c, "navigator", navigatorActiveTraceID, "completing", navigatorCompletingLabel(ls.getConfig().Language), "", "", 0, true)
							lockedSendJSON(c, map[string]any{
								"type":    "navigator.completed",
								"taskId":  refreshMsg.TaskID,
								"summary": refreshMsg.ObservedOutcome,
							})
							sendProcessingState(c, "navigator", navigatorActiveTraceID, "completing", "", "", "", 0, false)
							finalizeNavigatorTask("completed", refreshMsg.ObservedOutcome, "")
						}
					case "guided_mode":
						navState.recordStepResult(refreshMsg.Step, refreshMsg.Status, refreshMsg.ObservedOutcome)
						navState.resetStepRetry()
						recordGuidedMetrics("verification_guided_mode", &refreshMsg.Step, refreshMsg.ObservedOutcome, navigatorSurfaceFromState(*navState))
						sendProcessingState(c, "navigator", navigatorActiveTraceID, "verifying_result", "", "", "", 0, false)
						lockedSendJSON(c, map[string]any{
							"type":        "navigator.guidedMode",
							"taskId":      refreshMsg.TaskID,
							"reason":      "verification_guided_mode",
							"instruction": refreshMsg.ObservedOutcome,
						})
						finalizeNavigatorTask("guided_mode", refreshMsg.ObservedOutcome, "")
					default:
						navState.recordStepResult(refreshMsg.Step, refreshMsg.Status, refreshMsg.ObservedOutcome)
						retryCount := navState.incrementStepRetry()
						if retryCount <= 2 {
							retryStep := refreshMsg.Step
							if retryCount == 2 && retryStep.FallbackActionType != "" {
								retryStep.ActionType = retryStep.FallbackActionType
								if len(retryStep.FallbackHotkey) > 0 {
									retryStep.Hotkey = retryStep.FallbackHotkey
								}
							}
							if retryStep.ActionType == "cdp_js" && retryStep.CDPScript != "" {
								if ctrl := ls.getCDPController(); ctrl != nil {
									jsResult, jsErr := ctrl.EvaluateJS(retryStep.CDPScript)
									if jsErr == nil && (jsResult == "focused" || jsResult == "clicked" || jsResult == "clicked_container") {
										slog.Info("navigator cdp_js search activation succeeded, retrying paste_text", "conn_id", c.ID, "result", jsResult)
										retryStep.ActionType = "paste_text"
										retryStep.FallbackActionType = ""
										retryStep.CDPScript = ""
									} else {
										slog.Info("navigator cdp_js fallback result", "conn_id", c.ID, "result", jsResult, "err", jsErr)
									}
								} else {
									slog.Info("navigator cdp_js: CDP controller unavailable, dispatching to client", "conn_id", c.ID)
								}
							}
							slog.Info("navigator self-healing retry", "conn_id", c.ID, "step_id", retryStep.ID, "retry", retryCount, "status", refreshMsg.Status)
							persistNavigatorState()
							sendProcessingState(c, "navigator", navigatorActiveTraceID, "verifying_result", "", "", "", 0, false)
							sendProcessingState(c, "navigator", navigatorActiveTraceID, "retrying_step", navigatorRetryingLabel(ls.getConfig().Language), "", "", 0, true)
							sendProcessingState(c, "navigator", navigatorActiveTraceID, "retrying_step", "", "", "", 0, false)
							sendProcessingState(c, "navigator", navigatorActiveTraceID, "executing_step", navigatorExecutingLabel(ls.getConfig().Language), retryStep.ActionType, "", 0, true)
							lockedSendJSON(c, map[string]any{
								"type":    "navigator.stepPlanned",
								"taskId":  refreshMsg.TaskID,
								"step":    retryStep,
								"message": navigatorMessageForStep(retryStep),
							})
							continue
						}
						navState.resetStepRetry()
						sendProcessingState(c, "navigator", navigatorActiveTraceID, "verifying_result", "", "", "", 0, false)
						recordFailureMetrics(refreshMsg.Step, refreshMsg.ObservedOutcome, navigatorSurfaceFromState(*navState))
						lockedSendJSON(c, map[string]any{
							"type":   "navigator.failed",
							"taskId": refreshMsg.TaskID,
							"reason": refreshMsg.ObservedOutcome,
						})
						finalizeNavigatorTask("failed", refreshMsg.ObservedOutcome, "")
					}

				case "screenCapture", "forceCapture":
					if adkClient == nil {
						continue
					}
					if ls.isModelSpeaking() && msg.Type == "screenCapture" {
						slog.Debug("skipping screen capture during model speech", "conn_id", c.ID)
						continue
					}
					var captureMsg struct {
						Type            string `json:"type"`
						Image           string `json:"image"`
						Context         string `json:"context"`
						SessionID       string `json:"sessionId"`
						UserID          string `json:"userId"`
						Character       string `json:"character"`
						Soul            string `json:"soul"`
						ActivityMinutes int    `json:"activityMinutes"`
						TraceID         string `json:"traceId"`
					}
					if parseErr := json.Unmarshal(data, &captureMsg); parseErr != nil {
						slog.Warn("parse capture message failed", "conn_id", c.ID, "error", parseErr)
						continue
					}
					runtime.setIdentity(captureMsg.UserID, captureMsg.SessionID)
					{
						capTraceID := strings.TrimSpace(captureMsg.TraceID)
						if capTraceID == "" {
							capTraceID = newTraceID("cap")
						}
						capPayload := visionCapturePayload{
							image:     captureMsg.Image,
							sessionID: captureMsg.SessionID,
							userID:    captureMsg.UserID,
							traceID:   capTraceID,
						}
						if ls.deliverVisionCapture(capPayload) {
							slog.Info("[HANDLER] screen capture delivered to vision verification", "conn_id", c.ID, "trace_id", capTraceID)
							continue
						}
						if ls.deliverCheckpointCapture(capPayload) {
							slog.Info("[HANDLER] screen capture delivered to mid-step checkpoint", "conn_id", c.ID, "trace_id", capTraceID)
							continue
						}
					}
					if !ls.beginProactiveAnalyze(captureMsg.Type) {
						traceID := strings.TrimSpace(captureMsg.TraceID)
						if traceID == "" {
							traceID = newTraceID("cap")
						}
						slog.Info("[HANDLER] skipping proactive analyze while previous screen capture analyze is still running", "conn_id", c.ID, "trace_id", traceID)
						sendTraceEvent(c, "proactive", traceID, "adk_analyze_skipped_inflight", time.Now(), captureMsg.Type)
						continue
					}
					go func() {
						defer ls.finishProactiveAnalyze(captureMsg.Type)
						traceID := strings.TrimSpace(captureMsg.TraceID)
						if traceID == "" {
							traceID = newTraceID("cap")
						}
						rootAt := time.Now()
						sendTraceEvent(c, "proactive", traceID, "gateway_capture_received", rootAt, fmt.Sprintf("type=%s context_len=%d", captureMsg.Type, len(captureMsg.Context)))
						slog.Info("[HANDLER] >>> ADK analyze request",
							"conn_id", c.ID,
							"trace_id", traceID,
							"image_bytes", len(captureMsg.Image),
							"context", captureMsg.Context,
							"character", captureMsg.Character,
							"has_soul", captureMsg.Soul != "",
						)
						cfg := ls.getConfig()
						sendTraceEvent(c, "proactive", traceID, "adk_analyze_start", rootAt, "")
						sendProcessingState(c, "proactive", traceID, "screen_analyzing", screenAnalyzingLabel(cfg.Language), screenAnalyzingDetail(cfg.Language), "", 0, true)
						stopContextHint := maybeStartProactiveContextHint(ctx, c, ls, ttsClient, metrics, captureMsg.Type, captureMsg.Context, traceID, rootAt)
						defer stopContextHint()
						analyzeTimeout := proactiveAnalyzeTimeout
						if captureMsg.Type == "forceCapture" {
							analyzeTimeout = forcedAnalyzeTimeout
						}
						analyzeCtx, analyzeCancel := context.WithTimeout(ctx, analyzeTimeout)
						defer analyzeCancel()
						tracer := otel.Tracer("vibecat/gateway")
						analyzeCtx, span := tracer.Start(analyzeCtx, "adk.analyze")
						defer span.End()
						span.SetAttributes(
							attribute.String("app.trace_id", traceID),
							attribute.String("capture.type", captureMsg.Type),
							attribute.Int("image.bytes", len(captureMsg.Image)),
							attribute.String("conn.id", c.ID),
						)
						startTime := time.Now()
						result, analyzeErr := adkClient.Analyze(analyzeCtx, adk.AnalysisRequest{
							Image:           captureMsg.Image,
							Context:         captureMsg.Context,
							SessionID:       captureMsg.SessionID,
							UserID:          captureMsg.UserID,
							Character:       captureMsg.Character,
							Soul:            captureMsg.Soul,
							ActivityMinutes: captureMsg.ActivityMinutes,
							TraceID:         traceID,
						})
						elapsed := time.Since(startTime)
						metrics.RecordADKAnalyzeDuration(analyzeCtx, captureMsg.Type, elapsed)
						if analyzeErr != nil {
							if errors.Is(analyzeErr, context.DeadlineExceeded) {
								slog.Info("[HANDLER] <<< ADK analyze timed out; silent fallback", "conn_id", c.ID, "trace_id", traceID, "elapsed", elapsed.String(), "capture_type", captureMsg.Type)
								sendTraceEvent(c, "proactive", traceID, "adk_analyze_timed_out", rootAt, captureMsg.Type)
								metrics.RecordADKAnalyzeError(analyzeCtx, captureMsg.Type, "timeout")
							} else {
								slog.Warn("[HANDLER] <<< ADK analyze FAILED", "conn_id", c.ID, "error", analyzeErr, "elapsed", elapsed.String())
								sendTraceEvent(c, "proactive", traceID, "adk_analyze_failed", rootAt, analyzeErr.Error())
								metrics.RecordADKAnalyzeError(analyzeCtx, captureMsg.Type, "error")
							}
							sendProcessingState(c, "proactive", traceID, "screen_analyzing", screenAnalyzingLabel(cfg.Language), screenAnalyzingDetail(cfg.Language), "", 0, false)
							return
						}

						shouldSpeak := result != nil && result.Decision != nil && result.Decision.ShouldSpeak
						reason := ""
						urgency := ""
						mood := ""
						if result.Decision != nil {
							reason = result.Decision.Reason
							urgency = result.Decision.Urgency
						}
						if result.Mood != nil {
							mood = result.Mood.Mood
						}
						significance := 0
						if result.Vision != nil {
							significance = result.Vision.Significance
						}

						slog.Info("[HANDLER] <<< ADK analyze result",
							"conn_id", c.ID,
							"trace_id", traceID,
							"elapsed", elapsed.String(),
							"should_speak", shouldSpeak,
							"reason", reason,
							"urgency", urgency,
							"mood", mood,
							"significance", significance,
							"speech_text_len", len(result.SpeechText),
							"speech_text_preview", func() string {
								if len(result.SpeechText) > 80 {
									return result.SpeechText[:80] + "..."
								}
								return result.SpeechText
							}(),
						)
						sendTraceEvent(c, "proactive", traceID, "adk_analyze_done", rootAt, fmt.Sprintf("shouldSpeak=%t urgency=%s significance=%d", shouldSpeak, urgency, significance))
						if result != nil && result.Vision != nil && result.Vision.Content != "" {
							runtime.appendExecution("screen: " + truncateText(result.Vision.Content, 240))
						}
						if result != nil && result.SpeechText != "" {
							runtime.appendConversation("assistant: " + truncateText(result.SpeechText, 240))
						}

						allowProactiveSpeech := captureMsg.Type == "forceCapture" || cfg.ProactiveAudio

						if shouldSpeak && result.SpeechText != "" && allowProactiveSpeech {
							sendProcessingState(c, "proactive", traceID, "response_preparing", responsePreparingLabel(cfg.Language), currentWindowPreparingDetail(cfg.Language), "", 0, true)
							sess := ls.getSession()
							switch {
							case sess == nil:
								if speakWithTTSFallback(ctx, c, ls, ttsClient, metrics, "proactive", traceID, rootAt, result.SpeechText, "proactive_no_live_session") {
									slog.Info("[HANDLER] proactive response recovered via TTS fallback", "conn_id", c.ID, "trace_id", traceID)
								} else {
									slog.Warn("[HANDLER] proactive prompt dropped: no live session", "conn_id", c.ID)
									sendTraceEvent(c, "proactive", traceID, "live_prompt_dropped", rootAt, "no_live_session")
								}
								sendProcessingState(c, "proactive", traceID, "response_preparing", responsePreparingLabel(cfg.Language), currentWindowPreparingDetail(cfg.Language), "", 0, false)
							case ls.isModelSpeaking():
								slog.Info("[HANDLER] proactive prompt dropped: model already speaking", "conn_id", c.ID)
								sendTraceEvent(c, "proactive", traceID, "live_prompt_dropped", rootAt, "model_already_speaking")
								sendProcessingState(c, "proactive", traceID, "response_preparing", responsePreparingLabel(cfg.Language), currentWindowPreparingDetail(cfg.Language), "", 0, false)
							default:
								prompt := buildProactivePrompt(cfg, result)
								ls.queueTurnTrace(traceID, "proactive", rootAt)
								if sendErr := sess.SendText(prompt); sendErr != nil {
									ls.clearPendingTurnTrace()
									slog.Warn("[HANDLER] proactive prompt injection failed", "conn_id", c.ID, "error", sendErr)
									sendTraceEvent(c, "proactive", traceID, "live_prompt_injection_failed", rootAt, sendErr.Error())
									if speakWithTTSFallback(ctx, c, ls, ttsClient, metrics, "proactive", traceID, rootAt, result.SpeechText, "proactive_live_prompt_failed") {
										slog.Info("[HANDLER] proactive response recovered via TTS fallback", "conn_id", c.ID, "trace_id", traceID)
									}
									sendProcessingState(c, "proactive", traceID, "response_preparing", responsePreparingLabel(cfg.Language), currentWindowPreparingDetail(cfg.Language), "", 0, false)
								} else {
									slog.Info("[HANDLER] proactive prompt injected into live session",
										"conn_id", c.ID,
										"urgency", urgency,
										"reason", reason,
										"text_len", len(result.SpeechText),
									)
									sendTraceEvent(c, "proactive", traceID, "live_prompt_injected", rootAt, fmt.Sprintf("urgency=%s reason=%s", urgency, reason))
								}
							}
						} else if result != nil && result.Vision != nil && result.Vision.Content != "" {
							if sess := ls.getSession(); sess != nil {
								contextMsg := fmt.Sprintf("[Screen Context] %s", result.Vision.Content)
								ls.queueTurnTrace(traceID, "context", rootAt)
								if sendErr := sess.SendText(contextMsg); sendErr != nil {
									ls.clearPendingTurnTrace()
									slog.Debug("inject screen context failed", "conn_id", c.ID, "error", sendErr)
									sendTraceEvent(c, "context", traceID, "context_injection_failed", rootAt, sendErr.Error())
									sendProcessingState(c, "context", traceID, "screen_analyzing", screenAnalyzingLabel(cfg.Language), screenAnalyzingDetail(cfg.Language), "", 0, false)
								} else {
									slog.Info("[HANDLER] injected screen context into live session (no ADK speech)", "conn_id", c.ID, "content_len", len(result.Vision.Content))
									sendTraceEvent(c, "context", traceID, "context_injected", rootAt, fmt.Sprintf("content_len=%d", len(result.Vision.Content)))
									sendProcessingState(c, "context", traceID, "screen_analyzing", screenAnalyzingLabel(cfg.Language), screenAnalyzingDetail(cfg.Language), "", 0, false)
								}
							}
						} else {
							sendProcessingState(c, "proactive", traceID, "screen_analyzing", screenAnalyzingLabel(cfg.Language), screenAnalyzingDetail(cfg.Language), "", 0, false)
						}
					}()

				case "clientContent":
					var ccMsg struct {
						ClientContent struct {
							TurnComplete bool `json:"turnComplete"`
							Turns        []struct {
								Role  string `json:"role"`
								Parts []struct {
									Text string `json:"text"`
								} `json:"parts"`
							} `json:"turns"`
						} `json:"clientContent"`
					}
					if parseErr := json.Unmarshal(data, &ccMsg); parseErr != nil {
						slog.Warn("[HANDLER] parse clientContent failed", "conn_id", c.ID, "error", parseErr)
						continue
					}
					for _, turn := range ccMsg.ClientContent.Turns {
						for _, part := range turn.Parts {
							if part.Text != "" {
								traceID := strings.TrimSpace(msg.TraceID)
								if traceID == "" {
									traceID = newTraceID("text")
								}
								handleUserTextQuery(ctx, c, ls, adkClient, ttsClient, metrics, runtime, part.Text, traceID, time.Now())
							}
						}
					}

				case "ping":
					lockedSendJSON(c, map[string]string{"type": "pong"})
				}
			}
		}
	}
}

func receiveFromGemini(ctx context.Context, c *Conn, sess *live.Session, ls *liveSessionState, adkClient adkService, ttsClient ttsSpeaker, metrics *Metrics, runtime *sessionRuntime) {
	turnHasAudio := false
	inputTranscript := ""
	outputTranscript := ""
	firstOutputEventSent := false
	flushInputTranscript := func(triggerTool bool) {
		query := strings.TrimSpace(inputTranscript)
		if query == "" {
			return
		}
		flow := "voice"
		traceID := newTraceID(flow)
		rootAt := time.Now()
		runtime.appendConversation("user: " + truncateText(query, 240))
		lockedSendJSON(c, map[string]any{
			"type":     "inputTranscription",
			"text":     query,
			"finished": true,
		})
		sendTraceEvent(c, flow, traceID, "input_transcription_finished", rootAt, fmt.Sprintf("text_len=%d", len(query)))
		if triggerTool {
			cfg := ls.getConfig()
			route := resolveQueryRoute(cfg, query)
			switch route.Kind {
			case queryRouteLiveSearch:
				sendTraceEvent(c, flow, traceID, "live_native_search_enabled", rootAt, "google_search")
				sendProcessingState(c, flow, traceID, "searching", searchingLabel(cfg.Language), searchDetail(cfg.Language), "google_search", 0, true)
				ls.queueTurnTrace(traceID, flow, rootAt)
			case queryRouteADKTool:
				if adkClient == nil {
					slog.Warn("[HANDLER] grounded tool lookup skipped: no adk client", "conn_id", c.ID)
					sendTraceEvent(c, flow, traceID, "tool_lookup_skipped", rootAt, "no_adk_client")
				} else {
					go maybeResolveTool(ctx, c, ls, adkClient, ttsClient, metrics, runtime, route.Tool, query, traceID, rootAt)
				}
			default:
				ls.queueTurnTrace(traceID, flow, rootAt)
			}
		} else {
			ls.queueTurnTrace(traceID, flow, rootAt)
		}
		inputTranscript = ""
	}
	flushOutputTranscript := func() {
		text := strings.TrimSpace(outputTranscript)
		if text == "" {
			outputTranscript = ""
			return
		}
		runtime.appendConversation("assistant: " + truncateText(text, 240))
		outputTranscript = ""
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		msg, err := sess.Receive()
		if err != nil {
			slog.Warn("gemini receive error", "conn_id", c.ID, "error", err)
			if turnHasAudio {
				ls.setModelSpeaking(false)
				if traceID, flow, rootAt, ok := ls.finishCurrentTurnTrace(); ok {
					sendTraceEvent(c, flow, traceID, "turn_failed", rootAt, err.Error())
					sendProcessingState(c, flow, traceID, "response_preparing", responsePreparingLabel(ls.getConfig().Language), "", "", 0, false)
				}
				sendTurnState(c, "idle", "live")
			}
			if ls.getSession() == sess {
				ls.setSession(nil)
			}
			select {
			case ls.errChan <- err:
			default:
			}
			return
		}
		if msg == nil {
			continue
		}

		if msg.SetupComplete != nil {
			slog.Info("gemini setup complete", "conn_id", c.ID)
			continue
		}

		if msg.ServerContent != nil {
			sc := msg.ServerContent

			hasAudio := false
			hasText := false
			if sc.ModelTurn != nil {
				for _, part := range sc.ModelTurn.Parts {
					if part.InlineData != nil && len(part.InlineData.Data) > 0 {
						hasAudio = true

						if !turnHasAudio {
							flushInputTranscript(false)
							traceID, flow, rootAt := ls.ensureCurrentTurnTrace("voice")
							turnHasAudio = true
							ls.setModelSpeaking(true)
							sendTraceEvent(c, flow, traceID, "turn_started", rootAt, "source=audio")
							sendTurnState(c, "speaking", "live")
							slog.Info("[GEMINI-RX] model audio started, suppressing client input", "conn_id", c.ID)
						}

						if ls.shouldDiscardModelAudio() {
							continue
						}

						c.mu.Lock()
						c.conn.SetWriteDeadline(time.Now().Add(writeWait))
						writeErr := c.conn.WriteMessage(websocket.BinaryMessage, part.InlineData.Data)
						c.mu.Unlock()
						if writeErr != nil {
							slog.Warn("write audio to client failed", "conn_id", c.ID, "error", writeErr)
							return
						}
					}
					if part.Text != "" {
						hasText = true
						slog.Info("[GEMINI-RX] text part", "conn_id", c.ID, "text", part.Text[:min(len(part.Text), 100)])
					}
				}
			}
			if hasAudio || hasText || sc.TurnComplete || sc.Interrupted {
				slog.Debug("[GEMINI-RX] serverContent", "conn_id", c.ID, "has_audio", hasAudio, "has_text", hasText, "turn_complete", sc.TurnComplete, "interrupted", sc.Interrupted)
			}

			if sc.InputTranscription != nil {
				if sc.InputTranscription.Text != "" {
					inputTranscript += sc.InputTranscription.Text
				}

				textForClient := sc.InputTranscription.Text
				if sc.InputTranscription.Finished && strings.TrimSpace(textForClient) == "" {
					textForClient = inputTranscript
				}

				if !sc.InputTranscription.Finished && textForClient != "" {
					lockedSendJSON(c, map[string]any{
						"type":     "inputTranscription",
						"text":     textForClient,
						"finished": sc.InputTranscription.Finished,
					})
				}

				if sc.InputTranscription.Finished {
					flushInputTranscript(true)
				}
			}

			if sc.OutputTranscription != nil && (sc.OutputTranscription.Text != "" || sc.OutputTranscription.Finished) {
				if ls.shouldDiscardModelAudio() {
					slog.Debug("[GEMINI-RX] dropping output transcription after barge-in", "conn_id", c.ID)
				} else {
					flushInputTranscript(false)
					traceID, flow, rootAt := ls.ensureCurrentTurnTrace("voice")
					if sc.OutputTranscription.Text != "" {
						outputTranscript += sc.OutputTranscription.Text
						if !firstOutputEventSent {
							firstOutputEventSent = true
							sendTraceEvent(c, flow, traceID, "first_output_text", rootAt, fmt.Sprintf("text_len=%d", len(sc.OutputTranscription.Text)))
						}
					}
					lockedSendJSON(c, map[string]any{
						"type":     "transcription",
						"text":     sc.OutputTranscription.Text,
						"finished": sc.OutputTranscription.Finished,
					})
					if sc.OutputTranscription.Finished {
						flushOutputTranscript()
					}
				}
			}

			if sc.GroundingMetadata != nil {
				traceID, flow, rootAt := ls.ensureCurrentTurnTrace("voice")
				detail := describeGroundingMetadata(sc.GroundingMetadata)
				sourceCount := len(extractGroundingSources(sc.GroundingMetadata))
				sendTraceEvent(c, flow, traceID, "grounding_metadata", rootAt, detail)
				sendProcessingState(c, flow, traceID, "grounding", groundingLabel(ls.getConfig().Language), groundingDetail(ls.getConfig().Language, sourceCount), "google_search", sourceCount, true)
				if detail != "" {
					runtime.appendObservability("grounding: " + truncateText(detail, 240))
				}
				slog.Info("[GEMINI-RX] grounding metadata",
					"conn_id", c.ID,
					"trace_id", traceID,
					"detail", detail,
				)
			}

			if sc.TurnComplete {
				flushOutputTranscript()
				if turnHasAudio {
					turnHasAudio = false
					ls.setModelSpeaking(false)
					sendTurnState(c, "idle", "live")
				} else {
					ls.clearDiscardModelAudio()
				}
				if traceID, flow, rootAt, ok := ls.finishCurrentTurnTrace(); ok {
					sendTraceEvent(c, flow, traceID, "turn_complete", rootAt, "")
					sendProcessingState(c, flow, traceID, "response_preparing", responsePreparingLabel(ls.getConfig().Language), "", "", 0, false)
				}
				lockedSendJSON(c, map[string]string{"type": "turnComplete"})
				firstOutputEventSent = false
			}

			if sc.Interrupted {
				outputTranscript = ""
				runtime.appendObservability("interrupt: model turn interrupted")
				if turnHasAudio {
					turnHasAudio = false
					ls.setModelSpeaking(false)
					sendTurnState(c, "idle", "live")
				} else {
					ls.clearDiscardModelAudio()
				}
				if traceID, flow, rootAt, ok := ls.finishCurrentTurnTrace(); ok {
					sendTraceEvent(c, flow, traceID, "turn_interrupted", rootAt, "")
					sendProcessingState(c, flow, traceID, "response_preparing", responsePreparingLabel(ls.getConfig().Language), "", "", 0, false)
				}
				lockedSendJSON(c, map[string]string{"type": "interrupted"})
				firstOutputEventSent = false
			}
		}

		if msg.SessionResumptionUpdate != nil {
			if strings.TrimSpace(msg.SessionResumptionUpdate.NewHandle) != "" {
				ls.setResumeHandle(msg.SessionResumptionUpdate.NewHandle)
			}
			lockedSendJSON(c, map[string]any{
				"type":          "sessionResumptionUpdate",
				"sessionHandle": msg.SessionResumptionUpdate.NewHandle,
			})
		}

		if msg.UsageMetadata != nil {
			slog.Info("[GEMINI-RX] usage metadata",
				"conn_id", c.ID,
				"prompt_tokens", msg.UsageMetadata.PromptTokenCount,
				"response_tokens", msg.UsageMetadata.ResponseTokenCount,
				"tool_prompt_tokens", msg.UsageMetadata.ToolUsePromptTokenCount,
				"total_tokens", msg.UsageMetadata.TotalTokenCount,
			)
		}

		if msg.ToolCall != nil && len(msg.ToolCall.FunctionCalls) > 0 {
			handleLiveToolCall(c, sess, ls, metrics, runtime, msg.ToolCall)
		}

		if msg.ToolCallCancellation != nil {
			slog.Info("[GEMINI-RX] tool call cancellation", "conn_id", c.ID, "ids", msg.ToolCallCancellation.IDs)
		}

		if msg.VoiceActivity != nil {
			slog.Debug("[GEMINI-RX] voice activity", "conn_id", c.ID, "type", msg.VoiceActivity.VoiceActivityType)
		}

		if msg.GoAway != nil {
			lockedSendJSON(c, map[string]any{
				"type":       "goAway",
				"reason":     "session_timeout",
				"timeLeftMs": msg.GoAway.TimeLeft.Milliseconds(),
			})
			select {
			case ls.errChan <- errLiveSessionGoAway:
			default:
			}
		}
	}
}

func lockedSendJSON(c *Conn, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		slog.Error("marshal json failed", "error", err)
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if writeErr := c.conn.WriteMessage(websocket.TextMessage, data); writeErr != nil {
		slog.Warn("write json to client failed", "conn_id", c.ID, "error", writeErr)
	}
}

func lockedSendBinary(c *Conn, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := c.conn.WriteMessage(websocket.BinaryMessage, payload); err != nil {
		slog.Warn("write binary to client failed", "conn_id", c.ID, "error", err)
		return err
	}
	return nil
}

const (
	actionStateLoadTimeout  = 350 * time.Millisecond
	actionStateWriteTimeout = 500 * time.Millisecond
)

func handleLiveToolCall(c *Conn, sess *live.Session, ls *liveSessionState, metrics *Metrics, runtime *sessionRuntime, toolCall *genai.LiveServerToolCall) {
	for _, fc := range toolCall.FunctionCalls {
		slog.Info("[GEMINI-RX] function call received", "conn_id", c.ID, "function", fc.Name, "call_id", fc.ID)
		switch fc.Name {
		case "navigate_text_entry":
			handleNavigateTextEntryToolCall(c, sess, ls, metrics, runtime, fc)
		case "navigate_hotkey":
			handleNavigateHotkeyToolCall(c, sess, ls, metrics, runtime, fc)
		case "navigate_focus_app":
			handleNavigateFocusAppToolCall(c, sess, ls, metrics, runtime, fc)
		case "navigate_open_url":
			handleNavigateOpenURLToolCall(c, sess, ls, metrics, runtime, fc)
		case "navigate_type_and_submit":
			handleNavigateTypeAndSubmitToolCall(c, sess, ls, metrics, runtime, fc)
		default:
			slog.Warn("[GEMINI-RX] unknown function call", "conn_id", c.ID, "function", fc.Name)
			sendToolErrorResponse(sess, fc, "unknown function: "+fc.Name)
		}
	}
}

func handleNavigateTextEntryToolCall(c *Conn, sess *live.Session, ls *liveSessionState, metrics *Metrics, runtime *sessionRuntime, fc *genai.FunctionCall) {
	text, _ := fc.Args["text"].(string)
	target, _ := fc.Args["target"].(string)
	text = strings.TrimSpace(text)
	target = strings.TrimSpace(target)

	// submit defaults to true when not explicitly set to false
	submit := true
	if raw, ok := fc.Args["submit"]; ok {
		if b, ok := raw.(bool); ok {
			submit = b
		}
	}

	if text == "" {
		slog.Warn("[GEMINI-RX] navigate_text_entry called with empty text", "conn_id", c.ID, "call_id", fc.ID)
		sendToolErrorResponse(sess, fc, "text parameter is required")
		return
	}

	traceID := newTraceID("fc_nav")
	rootAt := time.Now()
	runtime.appendExecution(fmt.Sprintf("tool_call[navigate_text_entry]: text=%q target=%q submit=%v", truncateText(text, 80), target, submit))

	taskID := "fc_" + newConnID()
	sendTraceEvent(c, "navigator", traceID, "function_call_text_entry", rootAt, fmt.Sprintf("text_len=%d target=%q submit=%v", len(text), target, submit))

	targetApp := resolveToolCallTargetApp(target)
	if targetApp == "" {
		if fallback := runtime.getLastFCTargetApp(); fallback != "" {
			slog.Info("[GEMINI-RX] navigate_text_entry: no target, using lastFCTargetApp fallback", "conn_id", c.ID, "fallback", fallback)
			targetApp = fallback
		}
	}
	runtime.setLastFCTargetApp(targetApp)
	targetLabel := resolveToolCallTargetLabel(target)

	steps := buildToolCallTextEntrySteps(text, targetApp, targetLabel, submit)
	lockedSendJSON(c, map[string]any{
		"type":             "navigator.commandAccepted",
		"taskId":           taskID,
		"command":          fmt.Sprintf("navigate_text_entry: %s", truncateText(text, 60)),
		"intentClass":      "execute_now",
		"intentConfidence": 0.95,
		"source":           "function_call",
	})

	if len(steps) == 0 {
		sendTraceEvent(c, "navigator", traceID, "function_call_steps_sent", rootAt, "step_count=0")
		if err := sess.SendToolResponse([]*genai.FunctionResponse{{
			ID:       fc.ID,
			Name:     fc.Name,
			Response: map[string]any{"status": "no_steps", "text": text, "target": target},
		}}); err != nil {
			slog.Warn("[GEMINI-RX] SendToolResponse failed", "conn_id", c.ID, "call_id", fc.ID, "error", err)
		}
		return
	}

	ls.setPendingFC(fc.ID, fc.Name, taskID, text, target, steps[0].ID, steps[1:])
	lockedSendJSON(c, map[string]any{
		"type":    "navigator.stepPlanned",
		"taskId":  taskID,
		"step":    steps[0],
		"message": navigatorMessageForStep(steps[0]),
	})

	if metrics != nil {
		metrics.RecordNavigatorTask(context.Background(), navigatorSurfaceFromNames(targetApp, ""), navigatorIntentExecuteNow)
	}

	sendTraceEvent(c, "navigator", traceID, "function_call_first_step_sent", rootAt, fmt.Sprintf("step_count=%d", len(steps)))
}

func sendToolErrorResponse(sess *live.Session, fc *genai.FunctionCall, errMsg string) {
	if err := sess.SendToolResponse([]*genai.FunctionResponse{
		{
			ID:       fc.ID,
			Name:     fc.Name,
			Response: map[string]any{"error": errMsg},
		},
	}); err != nil {
		slog.Warn("[GEMINI-RX] SendToolResponse (error) failed", "call_id", fc.ID, "error", err)
	}
}

func resolveToolCallTargetApp(target string) string {
	lowered := strings.ToLower(target)
	switch {
	case containsAny(lowered, []string{"chrome", "크롬", "browser", "브라우저", "youtube", "유튜브", "google", "구글"}):
		return "Chrome"
	case containsAny(lowered, []string{"terminal", "터미널", "iterm", "warp", "ghostty"}):
		return "Terminal"
	case containsAny(lowered, []string{"antigravity", "codex", "ide"}):
		return "Antigravity"
	default:
		return ""
	}
}

func resolveToolCallTargetLabel(target string) string {
	lowered := strings.ToLower(target)
	switch {
	case containsAny(lowered, []string{"search", "검색"}):
		return "search"
	case containsAny(lowered, []string{"address", "주소", "url"}):
		return "address"
	case containsAny(lowered, []string{"prompt", "프롬프트"}):
		return "prompt"
	case containsAny(lowered, []string{"message", "메시지", "chat", "채팅"}):
		return "message"
	default:
		return ""
	}
}

func buildToolCallTextEntrySteps(text, targetApp, targetLabel string, submit bool) []navigatorStep {
	descriptor := navigatorTargetDescriptor{
		Role:    "textfield",
		AppName: targetApp,
	}
	if targetLabel != "" {
		descriptor.Label = targetLabel
	}

	verifyHint := text
	if len(verifyHint) > 24 {
		verifyHint = verifyHint[:24]
	}

	var steps []navigatorStep
	if targetApp != "" {
		steps = append(steps, navigatorStep{
			ID:               "fc_focus_before_text_" + newConnID(),
			ActionType:       "focus_app",
			TargetApp:        targetApp,
			TargetDescriptor: navigatorTargetDescriptor{AppName: targetApp},
			ExpectedOutcome:  "Switched to " + targetApp,
			Confidence:       0.95,
			IntentConfidence: 0.95,
			RiskLevel:        "low",
			ExecutionPolicy:  navigatorExecutionPolicyLow,
			FallbackPolicy:   "guided_mode",
			Surface:          navigatorSurfaceValue(targetApp),
			MacroID:          "fc_focus_before_text",
			Narration:        "Switching to " + targetApp + " before typing.",
			VerifyContract: &navigatorVerifyContract{
				ExpectedBundleID:    navigatorBundleIDForSurface(targetApp),
				RequireFrontmostApp: true,
				ProofStrategy:       "frontmost_app",
			},
			TimeoutMs:  900,
			ProofLevel: "strong",
		})
	}

	isBrowser := navigatorSurfaceValue(targetApp) == "chrome"
	labelLower := strings.ToLower(targetLabel)
	wantsSearchActivation := isBrowser && (strings.Contains(labelLower, "search") ||
		strings.Contains(labelLower, "검색") ||
		strings.Contains(labelLower, "youtube"))
	if wantsSearchActivation {
		steps = append(steps, navigatorStep{
			ID:               "fc_search_activate_" + newConnID(),
			ActionType:       "hotkey",
			TargetApp:        targetApp,
			Hotkey:           []string{"/"},
			ExpectedOutcome:  "Search field activated via / shortcut",
			Confidence:       0.90,
			IntentConfidence: 0.95,
			RiskLevel:        "low",
			ExecutionPolicy:  navigatorExecutionPolicyLow,
			FallbackPolicy:   "guided_mode",
			Surface:          navigatorSurfaceValue(targetApp),
			MacroID:          "fc_search_activate",
			Narration:        "Activating search field.",
			TimeoutMs:        800,
			ProofLevel:       "basic",
		})
	}

	steps = append(steps, navigatorStep{
		ID:               "fc_paste_text_" + newConnID(),
		ActionType:       "paste_text",
		TargetApp:        targetApp,
		TargetDescriptor: descriptor,
		InputText:        text,
		ExpectedOutcome:  "Text entered into the target field",
		Confidence:       0.92,
		IntentConfidence: 0.95,
		RiskLevel:        "low",
		ExecutionPolicy:  navigatorExecutionPolicyLow,
		FallbackPolicy:   "guided_mode",
		VerifyHint:       strings.ToLower(strings.TrimSpace(verifyHint)),
		Surface:          navigatorSurfaceValue(targetApp),
		MacroID:          "fc_paste_text",
		Narration:        "Typing the requested text.",
		VerifyContract: &navigatorVerifyContract{
			ExpectedBundleID:          navigatorBundleIDForSurface(targetApp),
			RequireFrontmostApp:       targetApp != "",
			RequireWritableTarget:     true,
			MinCaptureConfidenceAfter: 0.5,
			ProofStrategy:             "text_entry",
		},
		FallbackActionType: func() string {
			if wantsSearchActivation {
				return "cdp_js"
			}
			if isBrowser {
				return "hotkey"
			}
			return ""
		}(),
		FallbackHotkey: func() []string {
			if wantsSearchActivation {
				return nil
			}
			if isBrowser {
				return []string{"/"}
			}
			return nil
		}(),
		CDPScript: func() string {
			if wantsSearchActivation {
				return `(function(){var sb=document.querySelector('ytmusic-search-box');if(!sb)return 'not_found';var sr=sb.shadowRoot;if(sr){var inp=sr.querySelector('input');if(inp){inp.focus();inp.click();return 'focused';}var btn=sr.querySelector('#search-button');if(btn){btn.click();return 'clicked';}}sb.click();return 'clicked_container';})()`
			}
			return ""
		}(),
		MaxLocalRetries: 2,
		TimeoutMs:       1200,
		ProofLevel:      "strong",
	})

	if submit {
		steps = append(steps, navigatorStep{
			ID:               "fc_submit_enter_" + newConnID(),
			ActionType:       "hotkey",
			TargetApp:        targetApp,
			TargetDescriptor: descriptor,
			Hotkey:           []string{"return"},
			ExpectedOutcome:  "Submitted the entered text",
			Confidence:       0.95,
			IntentConfidence: 0.95,
			RiskLevel:        "low",
			ExecutionPolicy:  navigatorExecutionPolicyLow,
			FallbackPolicy:   "guided_mode",
			Surface:          navigatorSurfaceValue(targetApp),
			MacroID:          "fc_submit_enter",
			Narration:        "Pressing Enter to submit.",
			TimeoutMs:        800,
			ProofLevel:       "basic",
		})
	}

	if submit && wantsSearchActivation {
		steps = append(steps, navigatorStep{
			ID:         "fc_play_result_" + newConnID(),
			ActionType: "press_ax",
			TargetApp:  targetApp,
			TargetDescriptor: navigatorTargetDescriptor{
				AppName:    targetApp,
				Role:       "link",
				Label:      "first music result",
				RegionHint: "search_results",
			},
			ExpectedOutcome:  "Click the first search result to start playback",
			Confidence:       0.60,
			IntentConfidence: 0.90,
			RiskLevel:        "low",
			ExecutionPolicy:  navigatorExecutionPolicyLow,
			FallbackPolicy:   "guided_mode",
			Surface:          navigatorSurfaceValue(targetApp),
			MacroID:          "fc_play_result",
			Narration:        "Clicking the first result to play music.",
			TimeoutMs:        2000,
			ProofLevel:       "basic",
		})
	}

	return steps
}

func extractFCKeys(raw any) []string {
	slice, ok := raw.([]any)
	if !ok {
		return nil
	}
	keys := make([]string, 0, len(slice))
	for _, k := range slice {
		if ks, ok := k.(string); ok && strings.TrimSpace(ks) != "" {
			keys = append(keys, strings.TrimSpace(ks))
		}
	}
	return keys
}

func buildToolCallHotkeySteps(keys []string, targetApp string) []navigatorStep {
	steps := make([]navigatorStep, 0, 2)
	if targetApp != "" {
		steps = append(steps, navigatorStep{
			ID:               "fc_focus_" + newConnID(),
			ActionType:       "focus_app",
			TargetApp:        targetApp,
			TargetDescriptor: navigatorTargetDescriptor{AppName: targetApp},
			ExpectedOutcome:  targetApp + " is frontmost",
			Confidence:       0.93,
			IntentConfidence: 0.95,
			RiskLevel:        "low",
			ExecutionPolicy:  navigatorExecutionPolicyLow,
			FallbackPolicy:   "guided_mode",
			Surface:          navigatorSurfaceValue(targetApp),
			MacroID:          "fc_focus_app",
			Narration:        "Switching to " + targetApp + ".",
			TimeoutMs:        900,
			ProofLevel:       "strong",
		})
	}
	steps = append(steps, navigatorStep{
		ID:               "fc_hotkey_" + newConnID(),
		ActionType:       "hotkey",
		TargetApp:        targetApp,
		TargetDescriptor: navigatorTargetDescriptor{AppName: targetApp},
		Hotkey:           keys,
		ExpectedOutcome:  "Hotkey sent: " + strings.Join(keys, "+"),
		Confidence:       0.95,
		IntentConfidence: 0.95,
		RiskLevel:        "low",
		ExecutionPolicy:  navigatorExecutionPolicyLow,
		FallbackPolicy:   "guided_mode",
		Surface:          navigatorSurfaceValue(targetApp),
		MacroID:          "fc_hotkey",
		Narration:        "Sending hotkey " + strings.Join(keys, "+") + ".",
		TimeoutMs:        800,
		ProofLevel:       "basic",
	})
	return steps
}

func buildToolCallFocusAppStep(appName string) navigatorStep {
	return navigatorStep{
		ID:               "fc_focus_" + newConnID(),
		ActionType:       "focus_app",
		TargetApp:        appName,
		TargetDescriptor: navigatorTargetDescriptor{AppName: appName},
		ExpectedOutcome:  appName + " is frontmost",
		Confidence:       0.93,
		IntentConfidence: 0.95,
		RiskLevel:        "low",
		ExecutionPolicy:  navigatorExecutionPolicyLow,
		FallbackPolicy:   "guided_mode",
		Surface:          navigatorSurfaceValue(appName),
		MacroID:          "fc_focus_app",
		Narration:        "Switching to " + appName + ".",
		VerifyContract: &navigatorVerifyContract{
			ExpectedBundleID:    navigatorBundleIDForSurface(appName),
			RequireFrontmostApp: true,
			ProofStrategy:       "frontmost_app",
		},
		TimeoutMs:  900,
		ProofLevel: "strong",
	}
}

func buildToolCallOpenURLStep(rawURL string) navigatorStep {
	return navigatorStep{
		ID:               "fc_open_url_" + newConnID(),
		ActionType:       "open_url",
		TargetApp:        "Chrome",
		TargetDescriptor: navigatorTargetDescriptor{AppName: "Chrome"},
		URL:              rawURL,
		ExpectedOutcome:  "URL opened: " + truncateText(rawURL, 60),
		Confidence:       0.93,
		IntentConfidence: 0.95,
		RiskLevel:        "low",
		ExecutionPolicy:  navigatorExecutionPolicyLow,
		FallbackPolicy:   "guided_mode",
		Surface:          "chrome",
		MacroID:          "fc_open_url",
		Narration:        "Opening URL in browser.",
		TimeoutMs:        1500,
		ProofLevel:       "strong",
	}
}

func sendFCToolResponse(sess *live.Session, fc *genai.FunctionCall, connID string, extra map[string]any) {
	resp := map[string]any{"status": "initiated"}
	for k, v := range extra {
		resp[k] = v
	}
	if err := sess.SendToolResponse([]*genai.FunctionResponse{{
		ID:       fc.ID,
		Name:     fc.Name,
		Response: resp,
	}}); err != nil {
		slog.Warn("[GEMINI-RX] SendToolResponse failed", "conn_id", connID, "call_id", fc.ID, "error", err)
	}
}

func dispatchFCSteps(c *Conn, sess *live.Session, ls *liveSessionState, metrics *Metrics, runtime *sessionRuntime, fc *genai.FunctionCall, taskID, command, logKey string, steps []navigatorStep, extraResp map[string]any) {
	traceID := newTraceID("fc_" + strings.ReplaceAll(fc.Name, "navigate_", ""))
	rootAt := time.Now()
	runtime.appendExecution(fmt.Sprintf("tool_call[%s]: %s", fc.Name, truncateText(command, 80)))
	sendTraceEvent(c, "navigator", traceID, logKey, rootAt, command)

	lockedSendJSON(c, map[string]any{
		"type":             "navigator.commandAccepted",
		"taskId":           taskID,
		"command":          fmt.Sprintf("%s: %s", fc.Name, truncateText(command, 60)),
		"intentClass":      "execute_now",
		"intentConfidence": 0.95,
		"source":           "function_call",
	})

	if len(steps) == 0 {
		sendFCToolResponse(sess, fc, c.ID, extraResp)
		return
	}

	ls.setPendingFC(fc.ID, fc.Name, taskID, command, "", steps[0].ID, steps[1:])
	lockedSendJSON(c, map[string]any{
		"type":    "navigator.stepPlanned",
		"taskId":  taskID,
		"step":    steps[0],
		"message": navigatorMessageForStep(steps[0]),
	})

	if metrics != nil {
		surface := ""
		if len(steps) > 0 {
			surface = navigatorSurfaceValue(steps[0].TargetApp)
		}
		metrics.RecordNavigatorTask(context.Background(), surface, navigatorIntentExecuteNow)
	}
	sendTraceEvent(c, "navigator", traceID, "function_call_first_step_sent", rootAt, fmt.Sprintf("step_count=%d", len(steps)))
}

func handleNavigateHotkeyToolCall(c *Conn, sess *live.Session, ls *liveSessionState, metrics *Metrics, runtime *sessionRuntime, fc *genai.FunctionCall) {
	keys := extractFCKeys(fc.Args["keys"])
	target, _ := fc.Args["target"].(string)
	target = strings.TrimSpace(target)

	if len(keys) == 0 {
		slog.Warn("[GEMINI-RX] navigate_hotkey called with empty keys", "conn_id", c.ID, "call_id", fc.ID)
		sendToolErrorResponse(sess, fc, "keys parameter is required")
		return
	}

	targetApp := resolveToolCallTargetApp(target)
	taskID := "fc_" + newConnID()
	steps := buildToolCallHotkeySteps(keys, targetApp)
	dispatchFCSteps(c, sess, ls, metrics, runtime, fc, taskID, strings.Join(keys, "+"), "function_call_hotkey", steps, map[string]any{"keys": keys, "target": target})
}

func handleNavigateFocusAppToolCall(c *Conn, sess *live.Session, ls *liveSessionState, metrics *Metrics, runtime *sessionRuntime, fc *genai.FunctionCall) {
	app, _ := fc.Args["app"].(string)
	app = strings.TrimSpace(app)
	if app == "" {
		slog.Warn("[GEMINI-RX] navigate_focus_app called with empty app", "conn_id", c.ID, "call_id", fc.ID)
		sendToolErrorResponse(sess, fc, "app parameter is required")
		return
	}

	runtime.setLastFCTargetApp(app)
	taskID := "fc_" + newConnID()
	step := buildToolCallFocusAppStep(app)
	dispatchFCSteps(c, sess, ls, metrics, runtime, fc, taskID, app, "function_call_focus_app", []navigatorStep{step}, map[string]any{"app": app})
}

func handleNavigateOpenURLToolCall(c *Conn, sess *live.Session, ls *liveSessionState, metrics *Metrics, runtime *sessionRuntime, fc *genai.FunctionCall) {
	rawURL, _ := fc.Args["url"].(string)
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		slog.Warn("[GEMINI-RX] navigate_open_url called with empty url", "conn_id", c.ID, "call_id", fc.ID)
		sendToolErrorResponse(sess, fc, "url parameter is required")
		return
	}

	if ctrl := ls.getCDPController(); ctrl != nil {
		if err := ctrl.Navigate(rawURL); err == nil {
			runtime.setLastFCTargetApp("Chrome")
			slog.Info("[CDP] navigate_open_url executed via CDP", "conn_id", c.ID, "url", truncateText(rawURL, 80))
			runtime.appendExecution(fmt.Sprintf("tool_call[navigate_open_url via CDP]: %s", truncateText(rawURL, 80)))
			taskID := "fc_" + newConnID()
			lockedSendJSON(c, map[string]any{
				"type":             "navigator.commandAccepted",
				"taskId":           taskID,
				"command":          fmt.Sprintf("navigate_open_url: %s", truncateText(rawURL, 60)),
				"intentClass":      "execute_now",
				"intentConfidence": 0.98,
				"source":           "function_call_cdp",
			})
			lockedSendJSON(c, map[string]any{
				"type":    "navigator.completed",
				"taskId":  taskID,
				"summary": "Navigated to " + truncateText(rawURL, 60) + " via CDP",
			})
			sendFCToolResponse(sess, fc, c.ID, map[string]any{"url": rawURL, "via": "cdp"})
			return
		} else {
			slog.Warn("[CDP] navigate failed, falling back to AX steps", "conn_id", c.ID, "url", truncateText(rawURL, 80), "error", err)
		}
	}

	runtime.setLastFCTargetApp("Chrome")
	taskID := "fc_" + newConnID()
	step := buildToolCallOpenURLStep(rawURL)
	dispatchFCSteps(c, sess, ls, metrics, runtime, fc, taskID, rawURL, "function_call_open_url", []navigatorStep{step}, map[string]any{"url": rawURL})
}

func handleNavigateTypeAndSubmitToolCall(c *Conn, sess *live.Session, ls *liveSessionState, metrics *Metrics, runtime *sessionRuntime, fc *genai.FunctionCall) {
	text, _ := fc.Args["text"].(string)
	text = strings.TrimSpace(text)
	if text == "" {
		slog.Warn("[GEMINI-RX] navigate_type_and_submit called with empty text", "conn_id", c.ID, "call_id", fc.ID)
		sendToolErrorResponse(sess, fc, "text parameter is required")
		return
	}

	target, _ := fc.Args["target"].(string)
	target = strings.TrimSpace(target)

	submit := true
	if raw, ok := fc.Args["submit"]; ok {
		if b, ok := raw.(bool); ok {
			submit = b
		}
	}

	traceID := newTraceID("fc_type_submit")
	rootAt := time.Now()
	runtime.appendExecution(fmt.Sprintf("tool_call[navigate_type_and_submit]: text=%q target=%q submit=%v", truncateText(text, 80), target, submit))
	taskID := "fc_" + newConnID()
	sendTraceEvent(c, "navigator", traceID, "function_call_type_and_submit", rootAt, fmt.Sprintf("text_len=%d target=%q submit=%v", len(text), target, submit))

	targetApp := resolveToolCallTargetApp(target)
	if targetApp == "" {
		if fallback := runtime.getLastFCTargetApp(); fallback != "" {
			slog.Info("[GEMINI-RX] navigate_type_and_submit: no target, using lastFCTargetApp fallback", "conn_id", c.ID, "fallback", fallback)
			targetApp = fallback
		}
	}
	runtime.setLastFCTargetApp(targetApp)
	targetLabel := resolveToolCallTargetLabel(target)
	steps := buildToolCallTextEntrySteps(text, targetApp, targetLabel, submit)
	lockedSendJSON(c, map[string]any{
		"type":             "navigator.commandAccepted",
		"taskId":           taskID,
		"command":          fmt.Sprintf("navigate_type_and_submit: %s", truncateText(text, 60)),
		"intentClass":      "execute_now",
		"intentConfidence": 0.95,
		"source":           "function_call",
	})

	if len(steps) == 0 {
		sendTraceEvent(c, "navigator", traceID, "function_call_first_step_sent", rootAt, "step_count=0")
		if err := sess.SendToolResponse([]*genai.FunctionResponse{{
			ID:       fc.ID,
			Name:     fc.Name,
			Response: map[string]any{"status": "no_steps", "text": text},
		}}); err != nil {
			slog.Warn("[GEMINI-RX] SendToolResponse failed", "conn_id", c.ID, "call_id", fc.ID, "error", err)
		}
		return
	}

	ls.setPendingFC(fc.ID, fc.Name, taskID, text, target, steps[0].ID, steps[1:])
	lockedSendJSON(c, map[string]any{
		"type":    "navigator.stepPlanned",
		"taskId":  taskID,
		"step":    steps[0],
		"message": navigatorMessageForStep(steps[0]),
	})

	if metrics != nil {
		metrics.RecordNavigatorTask(context.Background(), navigatorSurfaceFromNames("", ""), navigatorIntentExecuteNow)
	}
	sendTraceEvent(c, "navigator", traceID, "function_call_first_step_sent", rootAt, fmt.Sprintf("step_count=%d", len(steps)))
}

const minTranscriptionLen = 4

func couldBeQuestion(text string) bool {
	text = strings.TrimSpace(text)
	if len([]rune(text)) < minTranscriptionLen {
		return false
	}
	if strings.ContainsAny(text, "?？") {
		return true
	}
	lower := strings.ToLower(text)
	for _, q := range []string{
		"뭐", "뭔", "어떻", "어디", "언제", "왜", "누가", "누구", "얼마",
		"알려", "알아", "찾아", "검색", "서치",
		"what", "how", "where", "when", "why", "who", "search", "find",
	} {
		if strings.Contains(lower, q) {
			return true
		}
	}
	return false
}
