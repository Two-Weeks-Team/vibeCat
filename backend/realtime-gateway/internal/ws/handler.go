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

const (
	memoryContextCacheTTL   = 5 * time.Minute
	proactiveAnalyzeTimeout = 4 * time.Second
	forcedAnalyzeTimeout    = 8 * time.Second
)

var proactiveContextHintDelay = 1200 * time.Millisecond

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
	modelSpeaking     bool
	discardModelAudio bool
	pendingTurnTrace  string
	pendingTurnFlow   string
	pendingTurnRootAt time.Time
	currentTurnTrace  string
	currentTurnFlow   string
	currentTurnRootAt time.Time
}

type sessionRuntime struct {
	mu        sync.Mutex
	userID    string
	sessionID string
	history   []string
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

func (sr *sessionRuntime) append(event string) {
	event = strings.TrimSpace(event)
	if event == "" {
		return
	}

	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.history = append(sr.history, event)
	if len(sr.history) > 200 {
		sr.history = sr.history[len(sr.history)-200:]
	}
}

func (sr *sessionRuntime) snapshot() (string, string, []string) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	history := append([]string(nil), sr.history...)
	return sr.userID, sr.sessionID, history
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
	if !ls.modelSpeaking {
		return false
	}
	alreadyDiscarding := ls.discardModelAudio
	ls.discardModelAudio = true
	return !alreadyDiscarding
}

func (ls *liveSessionState) shouldDiscardModelAudio() bool {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	return ls.discardModelAudio
}

func (ls *liveSessionState) queueTurnTrace(traceID, flow string, rootAt time.Time) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	ls.pendingTurnTrace = traceID
	ls.pendingTurnFlow = flow
	ls.pendingTurnRootAt = rootAt
}

func (ls *liveSessionState) clearPendingTurnTrace() {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	ls.pendingTurnTrace = ""
	ls.pendingTurnFlow = ""
	ls.pendingTurnRootAt = time.Time{}
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
	targetLower := strings.ToLower(ctx.TargetKind)

	switch uiLanguage(language) {
	case "en":
		switch {
		case strings.Contains(appLower, "xcode"):
			return fmt.Sprintf("%s is open in Xcode. Re-run one failing test or inspect the first error line before we go wider.", subject)
		case strings.Contains(appLower, "terminal") || strings.Contains(appLower, "iterm") || strings.Contains(appLower, "warp") || strings.Contains(appLower, "ghostty"):
			return fmt.Sprintf("%s is in the terminal. Start from the latest error line and confirm the exact command that triggered it.", subject)
		case strings.Contains(appLower, "cursor") || strings.Contains(appLower, "visual studio code") || strings.Contains(appLower, "code"):
			return fmt.Sprintf("%s is in the editor. Narrow it to one changed function or one failing file first.", subject)
		case strings.Contains(appLower, "chrome") || strings.Contains(appLower, "safari") || strings.Contains(appLower, "arc") || strings.Contains(appLower, "firefox"):
			return fmt.Sprintf("%s is open in the browser. Keep one source tab and one work tab, then verify the next step from there.", subject)
		case strings.Contains(targetLower, "display"):
			return "The whole display changed. Lock onto the one window that actually matters, then I can follow with a deeper read."
		default:
			return fmt.Sprintf("%s is in front. Check the last thing that changed there, then I will follow with a deeper suggestion.", subject)
		}
	case "ja":
		switch {
		case strings.Contains(appLower, "xcode"):
			return fmt.Sprintf("いま %s を Xcode で開いています。まず 1 件だけ失敗テストを再実行するか、最初のエラー行を確認しましょう。", subject)
		case strings.Contains(appLower, "terminal") || strings.Contains(appLower, "iterm") || strings.Contains(appLower, "warp") || strings.Contains(appLower, "ghostty"):
			return fmt.Sprintf("いま %s はターミナルです。最後のエラー行と、その直前のコマンドから先に確認しましょう。", subject)
		case strings.Contains(appLower, "cursor") || strings.Contains(appLower, "visual studio code") || strings.Contains(appLower, "code"):
			return fmt.Sprintf("いま %s はエディタです。変更した関数 1 つか、失敗しているファイル 1 つまで先に絞りましょう。", subject)
		case strings.Contains(appLower, "chrome") || strings.Contains(appLower, "safari") || strings.Contains(appLower, "arc") || strings.Contains(appLower, "firefox"):
			return fmt.Sprintf("いま %s はブラウザです。参照タブを 1 つに絞って、次の確認点だけ先に押さえましょう。", subject)
		case strings.Contains(targetLower, "display"):
			return "画面全体が変わっています。まず今重要なウィンドウ 1 つに絞ると、そのあと深く追えます。"
		default:
			return fmt.Sprintf("いま %s を見ています。そこで最後に変わった箇所 1 つから先に確認しましょう。", subject)
		}
	default:
		switch {
		case strings.Contains(appLower, "xcode"):
			return fmt.Sprintf("지금 %s가 Xcode에 열려 있어. 실패한 테스트 하나만 다시 돌리거나 첫 에러 줄부터 바로 보자.", subject)
		case strings.Contains(appLower, "terminal") || strings.Contains(appLower, "iterm") || strings.Contains(appLower, "warp") || strings.Contains(appLower, "ghostty"):
			return fmt.Sprintf("지금 %s가 터미널이야. 마지막 에러 줄과 그 직전 명령부터 먼저 확인하자.", subject)
		case strings.Contains(appLower, "cursor") || strings.Contains(appLower, "visual studio code") || strings.Contains(appLower, "code"):
			return fmt.Sprintf("지금 %s가 에디터에 열려 있어. 방금 바꾼 함수 하나나 깨진 파일 하나부터 좁혀보자.", subject)
		case strings.Contains(appLower, "chrome") || strings.Contains(appLower, "safari") || strings.Contains(appLower, "arc") || strings.Contains(appLower, "firefox"):
			return fmt.Sprintf("지금 %s가 브라우저에 열려 있어. 참고 탭 하나만 남기고 다음 확인 포인트부터 잡자.", subject)
		case strings.Contains(targetLower, "display"):
			return "지금은 화면 전체가 바뀌었어. 먼저 중요한 창 하나만 딱 고르면 내가 그다음 흐름을 더 깊게 이어갈게."
		default:
			return fmt.Sprintf("지금 %s 쪽이야. 거기서 마지막으로 바뀐 지점 하나부터 먼저 확인하자.", subject)
		}
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

	runtime.append(fmt.Sprintf("search[%s]: %s", traceID, truncateText(result.Summary, 240)))
	sendProcessingState(c, "text", traceID, "response_preparing", responsePreparingLabel(cfg.Language), toolPreparingDetail(adk.ToolKindSearch, cfg.Language), string(adk.ToolKindSearch), len(result.Sources), true)
	success := speakWithTTSFallback(ctx, c, ls, ttsClient, metrics, "text", traceID, rootAt, result.Summary, reason)
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

	runtime.append("user: " + truncateText(query, 240))
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

	runtime.append(fmt.Sprintf("tool[%s]: %s => %s", result.Tool, query, truncateText(result.Summary, 240)))
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
		actionStateOwner := strings.TrimSpace(c.ID)

		persistNavigatorState := func() {
			navState.bindLease(strings.TrimSpace(ls.getConfig().DeviceID), c.ID)
			if actionStore == nil {
				return
			}
			if !navState.hasPersistableState() {
				if err := actionStore.Delete(context.Background(), actionStateOwner); err != nil {
					slog.Warn("action state delete failed", "conn_id", c.ID, "owner", actionStateOwner, "error", err)
				}
				return
			}
			if err := actionStore.Save(context.Background(), actionStateOwner, *navState); err != nil {
				slog.Warn("action state save failed", "conn_id", c.ID, "owner", actionStateOwner, "error", err)
			}
		}

		syncNavigatorState := func() bool {
			if actionStore == nil {
				return true
			}
			restored, ok, err := actionStore.Load(context.Background(), actionStateOwner)
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

		finalizeNavigatorTask := func(outcome, outcomeDetail, traceID string) {
			snapshot := navState.snapshotTask(time.Now().UTC())
			enqueueNavigatorBackground(context.Background(), adkClient, runtime, ls.getConfig(), snapshot, outcome, outcomeDetail, traceID)
			navState.clearPlan()
			persistNavigatorState()
		}

		defer func() {
			saveSessionMemory(context.Background(), adkClient, ls.getConfig(), runtime)
			if sess := ls.getSession(); sess != nil {
				sess.Close()
			}
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
						cfg.MemoryContext = fetchMemoryContext(ctx, adkClient, cfg)
						if cfg.MemoryContext != "" {
							slog.Info("[HANDLER] memory context loaded for live session", "conn_id", c.ID, "device_id", cfg.DeviceID, "context_len", len(cfg.MemoryContext))
						}
					}
					ls.setConfig(cfg)
					ls.setResumeHandle(setupMsg.ResumptionHandle)
					runtime.setIdentity(cfg.DeviceID, c.ID)
					previousActionStateOwner := actionStateOwner
					if strings.TrimSpace(cfg.DeviceID) != "" {
						actionStateOwner = strings.TrimSpace(cfg.DeviceID)
					}
					if restored, ok, err := actionStore.Load(ctx, actionStateOwner); err != nil {
						slog.Warn("action state restore failed", "conn_id", c.ID, "owner", actionStateOwner, "error", err)
					} else if ok {
						*navState = restored
						navState.bindLease(strings.TrimSpace(cfg.DeviceID), c.ID)
						persistNavigatorState()
						if previousActionStateOwner != actionStateOwner {
							if err := actionStore.Delete(context.Background(), previousActionStateOwner); err != nil {
								slog.Warn("legacy action state delete failed", "conn_id", c.ID, "owner", previousActionStateOwner, "error", err)
							}
						}
						if navState.hasActiveTask() {
							lockedSendJSON(c, map[string]any{
								"type":        "navigator.guidedMode",
								"taskId":      navState.activeTaskID,
								"reason":      "restored_task_state",
								"instruction": fmt.Sprintf("I restored the previous action state for %q. Ask me to resume it or give me a new command.", navState.activeCommand),
							})
						}
					} else {
						navState.bindLease(strings.TrimSpace(cfg.DeviceID), c.ID)
						if previousActionStateOwner != actionStateOwner && navState.hasPersistableState() {
							persistNavigatorState()
							if err := actionStore.Delete(context.Background(), previousActionStateOwner); err != nil {
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
					runtime.append("interrupt: user barge-in")
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
					rootAt := time.Now()
					sendTraceEvent(c, "navigator", traceID, "command_received", rootAt, truncateText(navMsg.Command, 120))
					plan := planNavigatorCommand(navMsg.Command, navMsg.Context, false)
					plan = maybeEscalateNavigatorPlan(ctx, adkClient, metrics, ls.getConfig().Language, navMsg.Command, navMsg.Context, plan, traceID)
					switch {
					case plan.IntentClass == navigatorIntentAmbiguous:
						navState.stageClarification(navigatorPromptIntent, navMsg.Command)
						persistNavigatorState()
						if metrics != nil {
							metrics.RecordClarification(context.Background(), string(navigatorPromptIntent), navigatorSurfaceFromContext(navMsg.Context))
						}
						lockedSendJSON(c, map[string]any{
							"type":     "navigator.intentClarificationNeeded",
							"command":  navMsg.Command,
							"question": plan.ClarifyQuestion,
						})
					case plan.IntentClass == navigatorIntentAnalyzeOnly:
						lockedSendJSON(c, map[string]any{
							"type":             "navigator.commandAccepted",
							"taskId":           "",
							"command":          navMsg.Command,
							"intentClass":      plan.IntentClass,
							"intentConfidence": plan.IntentConfidence,
						})
						handleUserTextQuery(ctx, c, ls, adkClient, ttsClient, metrics, runtime, navMsg.Command, traceID, rootAt)
					case len(plan.Steps) == 0:
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
						navState.stageClarification(navigatorPromptReplace, navMsg.Command)
						persistNavigatorState()
						if metrics != nil {
							surface := navigatorSurfaceFromState(*navState)
							metrics.RecordClarification(context.Background(), string(navigatorPromptReplace), surface)
							metrics.RecordTaskReplacement(context.Background(), surface)
						}
						lockedSendJSON(c, map[string]any{
							"type":     "navigator.intentClarificationNeeded",
							"command":  navMsg.Command,
							"question": buildTaskReplacementQuestion(activeCommand, navMsg.Command),
						})
					case plan.RiskQuestion != "":
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
						navState.rememberInitialContext(navMsg.Context)
						persistNavigatorState()
						if metrics != nil {
							metrics.RecordNavigatorTask(context.Background(), navigatorSurfaceFromState(*navState), plan.IntentClass)
						}
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
					plan := planNavigatorCommand(command+" "+confirmMsg.Answer, confirmMsg.Context, false)
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
					if !navState.acceptsRefresh(refreshMsg.Command, refreshMsg.TaskID, refreshMsg.Step.ID) {
						slog.Warn("navigator refresh ignored", "conn_id", c.ID, "command", refreshMsg.Command, "task_id", refreshMsg.TaskID, "step_id", refreshMsg.Step.ID, "active_command", navState.activeCommand)
						continue
					}
					switch refreshMsg.Status {
					case "success":
						navState.recordStepResult(refreshMsg.Step, refreshMsg.Status, refreshMsg.ObservedOutcome)
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
						if next, ok := navState.nextStep(); ok {
							persistNavigatorState()
							lockedSendJSON(c, map[string]any{
								"type":    "navigator.stepPlanned",
								"taskId":  refreshMsg.TaskID,
								"step":    next,
								"message": navigatorMessageForStep(next),
							})
						} else {
							lockedSendJSON(c, map[string]any{
								"type":    "navigator.completed",
								"taskId":  refreshMsg.TaskID,
								"summary": refreshMsg.ObservedOutcome,
							})
							finalizeNavigatorTask("completed", refreshMsg.ObservedOutcome, "")
						}
					case "guided_mode":
						navState.recordStepResult(refreshMsg.Step, refreshMsg.Status, refreshMsg.ObservedOutcome)
						recordGuidedMetrics("verification_guided_mode", &refreshMsg.Step, refreshMsg.ObservedOutcome, navigatorSurfaceFromState(*navState))
						lockedSendJSON(c, map[string]any{
							"type":        "navigator.guidedMode",
							"taskId":      refreshMsg.TaskID,
							"reason":      "verification_guided_mode",
							"instruction": refreshMsg.ObservedOutcome,
						})
						finalizeNavigatorTask("guided_mode", refreshMsg.ObservedOutcome, "")
					default:
						navState.recordStepResult(refreshMsg.Step, refreshMsg.Status, refreshMsg.ObservedOutcome)
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
					go func() {
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
							runtime.append("screen: " + truncateText(result.Vision.Content, 240))
						}
						if result != nil && result.SpeechText != "" {
							runtime.append("assistant: " + truncateText(result.SpeechText, 240))
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
		runtime.append("user: " + truncateText(query, 240))
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
		runtime.append("assistant: " + truncateText(text, 240))
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
					runtime.append("grounding: " + truncateText(detail, 240))
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
				runtime.append("interrupt: model turn interrupted")
				if turnHasAudio {
					turnHasAudio = false
					ls.setModelSpeaking(false)
					sendTurnState(c, "idle", "live")
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
