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
	"vibecat/realtime-gateway/internal/live"
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

var errLiveSessionGoAway = errors.New("gemini live goAway")

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

func fetchMemoryContext(ctx context.Context, adkClient *adk.Client, cfg live.Config) string {
	if adkClient == nil {
		return ""
	}
	userID := strings.TrimSpace(cfg.DeviceID)
	if userID == "" {
		return ""
	}

	memoryCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()

	contextText, err := adkClient.MemoryContext(memoryCtx, adk.MemoryContextRequest{
		UserID:   userID,
		Language: cfg.Language,
	})
	if err != nil {
		slog.Warn("[HANDLER] memory context lookup failed", "user_id", userID, "error", err)
		return ""
	}
	return strings.TrimSpace(contextText)
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

func maybeResolveTool(ctx context.Context, c *Conn, ls *liveSessionState, adkClient *adk.Client, runtime *sessionRuntime, query string, traceID string, rootAt time.Time) bool {
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
	toolCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	result, err := adkClient.Tool(toolCtx, adk.ToolRequest{
		Query:     query,
		Language:  ls.getConfig().Language,
		SessionID: sessionID,
		UserID:    userID,
		TraceID:   traceID,
	})
	if err != nil {
		slog.Warn("[HANDLER] tool request failed", "conn_id", c.ID, "query", truncateText(query, 80), "error", err)
		sendTraceEvent(c, "tool", traceID, "tool_lookup_failed", rootAt, err.Error())
		return false
	}
	if result == nil || strings.TrimSpace(result.Summary) == "" {
		sendTraceEvent(c, "tool", traceID, "tool_lookup_empty", rootAt, "")
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
		slog.Warn("[HANDLER] grounded tool prompt dropped: no live session", "conn_id", c.ID)
		sendTraceEvent(c, "tool", traceID, "tool_prompt_dropped", rootAt, "no_live_session")
		return true
	}
	ls.queueTurnTrace(traceID, "tool", rootAt)
	if err := liveSess.SendText(buildToolPrompt(ls.getConfig(), result)); err != nil {
		slog.Warn("[HANDLER] grounded tool prompt injection failed", "conn_id", c.ID, "error", err)
		sendTraceEvent(c, "tool", traceID, "tool_prompt_injection_failed", rootAt, err.Error())
		return false
	}

	slog.Info("[HANDLER] grounded tool prompt injected", "conn_id", c.ID, "tool", result.Tool, "summary_len", len(result.Summary))
	sendTraceEvent(c, "tool", traceID, "live_prompt_injected", rootAt, fmt.Sprintf("tool=%s", result.Tool))
	return true
}

func saveSessionMemory(ctx context.Context, adkClient *adk.Client, cfg live.Config, runtime *sessionRuntime) {
	if adkClient == nil || runtime == nil {
		return
	}

	userID, sessionID, history := runtime.snapshot()
	if strings.TrimSpace(userID) == "" || len(history) == 0 {
		return
	}

	saveCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := adkClient.SaveSessionSummary(saveCtx, adk.SessionSummaryRequest{
		UserID:    userID,
		SessionID: sessionID,
		Language:  cfg.Language,
		History:   history,
	}); err != nil {
		slog.Warn("[HANDLER] session summary save failed", "user_id", userID, "session_id", sessionID, "error", err)
		return
	}

	slog.Info("[HANDLER] session summary saved", "user_id", userID, "session_id", sessionID, "history_len", len(history))
}

// Handler returns an http.HandlerFunc that upgrades connections to WebSocket.
// liveMgr may be nil — in that case audio is echoed back (stub mode).
// adkClient may be nil — in that case screen captures are ignored.
func Handler(reg *Registry, liveMgr *live.Manager, adkClient *adk.Client) http.HandlerFunc {
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
		slog.Info("websocket connected", "conn_id", c.ID, "remote", r.RemoteAddr)

		ls := &liveSessionState{errChan: make(chan error, 1)}
		runtime := newSessionRuntime("default", c.ID)

		defer func() {
			saveSessionMemory(context.Background(), adkClient, ls.getConfig(), runtime)
			if sess := ls.getSession(); sess != nil {
				sess.Close()
			}
			reg.Remove(c.ID)
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
					} else if !ls.isReconnecting() {
						_ = rawConn.WriteMessage(websocket.BinaryMessage, data)
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
					if liveMgr == nil {
						lockedSendJSON(c, setupCompleteMsg{Type: "setupComplete", SessionID: c.ID})
						continue
					}
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
					go receiveFromGemini(ctx, c, sess, ls, adkClient, runtime)

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
										go receiveFromGemini(ctx, c, newSess, ls, adkClient, runtime)
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
						sendTraceEvent(c, "proactive", traceID, "adk_analyze_start", rootAt, "")
						analyzeCtx, analyzeCancel := context.WithTimeout(ctx, 30*time.Second)
						defer analyzeCancel()
						tracer := otel.Tracer("vibecat/gateway")
						analyzeCtx, span := tracer.Start(analyzeCtx, "adk.analyze")
						defer span.End()
						span.SetAttributes(
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
						if analyzeErr != nil {
							slog.Warn("[HANDLER] <<< ADK analyze FAILED", "conn_id", c.ID, "error", analyzeErr, "elapsed", elapsed.String())
							sendTraceEvent(c, "proactive", traceID, "adk_analyze_failed", rootAt, analyzeErr.Error())
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

						allowProactiveSpeech := captureMsg.Type == "forceCapture" || ls.getConfig().ProactiveAudio

						if shouldSpeak && result.SpeechText != "" && allowProactiveSpeech {
							sess := ls.getSession()
							switch {
							case sess == nil:
								slog.Warn("[HANDLER] proactive prompt dropped: no live session", "conn_id", c.ID)
								sendTraceEvent(c, "proactive", traceID, "live_prompt_dropped", rootAt, "no_live_session")
							case ls.isModelSpeaking():
								slog.Info("[HANDLER] proactive prompt dropped: model already speaking", "conn_id", c.ID)
								sendTraceEvent(c, "proactive", traceID, "live_prompt_dropped", rootAt, "model_already_speaking")
							default:
								prompt := buildProactivePrompt(ls.getConfig(), result)
								ls.queueTurnTrace(traceID, "proactive", rootAt)
								if sendErr := sess.SendText(prompt); sendErr != nil {
									slog.Warn("[HANDLER] proactive prompt injection failed", "conn_id", c.ID, "error", sendErr)
									sendTraceEvent(c, "proactive", traceID, "live_prompt_injection_failed", rootAt, sendErr.Error())
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
									slog.Debug("inject screen context failed", "conn_id", c.ID, "error", sendErr)
									sendTraceEvent(c, "context", traceID, "context_injection_failed", rootAt, sendErr.Error())
								} else {
									slog.Info("[HANDLER] injected screen context into live session (no ADK speech)", "conn_id", c.ID, "content_len", len(result.Vision.Content))
									sendTraceEvent(c, "context", traceID, "context_injected", rootAt, fmt.Sprintf("content_len=%d", len(result.Vision.Content)))
								}
							}
						}
					}()

				case "clientContent":
					sess := ls.getSession()
					if sess == nil {
						slog.Warn("[HANDLER] clientContent received but no live session", "conn_id", c.ID)
						continue
					}
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
								rootAt := time.Now()
								runtime.append("user: " + truncateText(part.Text, 240))
								sendTraceEvent(c, "text", traceID, "text_received", rootAt, fmt.Sprintf("text_len=%d", len(part.Text)))
								if ls.getConfig().GoogleSearch && maybeResolveTool(ctx, c, ls, adkClient, runtime, part.Text, traceID, rootAt) {
									slog.Info("[HANDLER] clientContent handled via grounded tool", "conn_id", c.ID)
									continue
								}
								slog.Info("[HANDLER] >>> forwarding clientContent to Gemini",
									"conn_id", c.ID,
									"trace_id", traceID,
									"text_len", len(part.Text),
								)
								ls.queueTurnTrace(traceID, "text", rootAt)
								if sendErr := sess.SendText(part.Text); sendErr != nil {
									slog.Warn("[HANDLER] Gemini SendText failed", "conn_id", c.ID, "error", sendErr)
									sendTraceEvent(c, "text", traceID, "live_text_forward_failed", rootAt, sendErr.Error())
								} else {
									sendTraceEvent(c, "text", traceID, "live_text_forwarded", rootAt, "")
								}
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

func receiveFromGemini(ctx context.Context, c *Conn, sess *live.Session, ls *liveSessionState, adkClient *adk.Client, runtime *sessionRuntime) {
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
		ls.queueTurnTrace(traceID, flow, rootAt)
		if triggerTool {
			cfg := ls.getConfig()
			if useNativeLiveSearch(cfg) {
				sendTraceEvent(c, flow, traceID, "live_native_search_enabled", rootAt, "google_search")
			} else if !cfg.GoogleSearch {
				slog.Info("[HANDLER] grounded tool lookup skipped: disabled in session config", "conn_id", c.ID)
				sendTraceEvent(c, flow, traceID, "tool_lookup_skipped", rootAt, "disabled_in_session_config")
			} else if adkClient == nil {
				slog.Warn("[HANDLER] grounded tool lookup skipped: no adk client", "conn_id", c.ID)
				sendTraceEvent(c, flow, traceID, "tool_lookup_skipped", rootAt, "no_adk_client")
			} else {
				go maybeResolveTool(ctx, c, ls, adkClient, runtime, query, traceID, rootAt)
			}
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
				sendTraceEvent(c, flow, traceID, "grounding_metadata", rootAt, detail)
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
				}
				lockedSendJSON(c, map[string]string{"type": "interrupted"})
				firstOutputEventSent = false
			}
		}

		if msg.SessionResumptionUpdate != nil {
			ls.setResumeHandle(msg.SessionResumptionUpdate.NewHandle)
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
