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
	"vibecat/realtime-gateway/internal/adk"
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

type Conn struct {
	ID   string
	conn *websocket.Conn
	mu   sync.Mutex
}

type message struct {
	Type string `json:"type"`
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

var errLiveSessionGoAway = errors.New("gemini live goAway")

type liveSessionState struct {
	mu           sync.RWMutex
	session      *live.Session
	config       live.Config
	resumeHandle string
	reconnecting bool
	errChan      chan error
	ttsMu        sync.Mutex
	ttsCancel    context.CancelFunc

	// modelSpeaking is true while any assistant audio is actively streaming to the client.
	// While true, screen captures are deferred and incoming user speech is treated as barge-in.
	modelSpeaking     bool
	discardModelAudio bool
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

func (ls *liveSessionState) cancelTTS() {
	ls.ttsMu.Lock()
	if ls.ttsCancel != nil {
		ls.ttsCancel()
		ls.ttsCancel = nil
	}
	ls.ttsMu.Unlock()
}

func (ls *liveSessionState) setTTSCancel(cancel context.CancelFunc) {
	ls.ttsMu.Lock()
	ls.ttsCancel = cancel
	ls.ttsMu.Unlock()
}

func (ls *liveSessionState) hasActiveTTS() bool {
	ls.ttsMu.Lock()
	defer ls.ttsMu.Unlock()
	return ls.ttsCancel != nil
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

func isJPEG(data []byte) bool {
	return len(data) > 2 && data[0] == 0xFF && data[1] == 0xD8
}

func newConnID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func handleBargeIn(c *Conn, ls *liveSessionState) {
	shouldInterrupt := ls.markBargeInPending()
	if ls.hasActiveTTS() {
		ls.cancelTTS()
		shouldInterrupt = true
	}
	if shouldInterrupt {
		lockedSendJSON(c, map[string]string{"type": "interrupted"})
	}
}

// Handler returns an http.HandlerFunc that upgrades connections to WebSocket.
// liveMgr may be nil — in that case audio is echoed back (stub mode).
// adkClient may be nil — in that case screen captures are ignored.
func Handler(reg *Registry, liveMgr *live.Manager, adkClient *adk.Client, ttsClient *tts.Client) http.HandlerFunc {
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

		defer func() {
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
					if ls.isModelSpeaking() || ls.hasActiveTTS() {
						handleBargeIn(c, ls)
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
					ls.setConfig(setupMsg.Config)
					ls.setResumeHandle(setupMsg.ResumptionHandle)
					if old := ls.getSession(); old != nil {
						old.Close()
						ls.setSession(nil)
					}
					select {
					case <-ls.errChan:
					default:
					}
					sess, connectErr := liveMgr.Connect(ctx, setupMsg.Config, setupMsg.ResumptionHandle)
					if connectErr != nil {
						slog.Error("gemini connect failed", "conn_id", c.ID, "error", connectErr)
						lockedSendJSON(c, errorMsg{Type: "error", Code: "GEMINI_CONNECT_FAILED", Message: connectErr.Error()})
						continue
					}
					ls.setSession(sess)
					slog.Info("device registered", "conn_id", c.ID, "device_id", setupMsg.Config.DeviceID)
					lockedSendJSON(c, setupCompleteMsg{Type: "setupComplete", SessionID: c.ID})
					go receiveFromGemini(ctx, c, sess, ls, adkClient, ttsClient)

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
										go receiveFromGemini(ctx, c, newSess, ls, adkClient, ttsClient)
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
					}
					if parseErr := json.Unmarshal(data, &captureMsg); parseErr != nil {
						slog.Warn("parse capture message failed", "conn_id", c.ID, "error", parseErr)
						continue
					}
					go func() {
						slog.Info("[HANDLER] >>> ADK analyze request",
							"conn_id", c.ID,
							"image_bytes", len(captureMsg.Image),
							"context", captureMsg.Context,
							"character", captureMsg.Character,
							"has_soul", captureMsg.Soul != "",
						)
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
						})
						elapsed := time.Since(startTime)
						if analyzeErr != nil {
							slog.Warn("[HANDLER] <<< ADK analyze FAILED", "conn_id", c.ID, "error", analyzeErr, "elapsed", elapsed.String())
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

						if shouldSpeak && result.SpeechText != "" {
							emotion := "neutral"
							if result.Vision != nil {
								emotion = result.Vision.Emotion
							}
							slog.Info("[HANDLER] >>> sending companionSpeech to client",
								"conn_id", c.ID,
								"emotion", emotion,
								"text_len", len(result.SpeechText),
							)
							lockedSendJSON(c, map[string]any{
								"type":    "companionSpeech",
								"text":    result.SpeechText,
								"emotion": emotion,
								"urgency": urgency,
							})

							if urgency == "high" || urgency == "critical" {
								if ttsClient != nil {
									ttsText := fmt.Sprintf("[%s] %s", emotion, result.SpeechText)
									startTTSStream(ctx, c, ls, ttsClient, ttsText, result.SpeechText)
								} else {
									slog.Warn("[HANDLER] no TTS client — audio will not play", "conn_id", c.ID)
								}
							} else {
								slog.Info("[HANDLER] bubble-only mode (low urgency)", "conn_id", c.ID, "urgency", urgency, "text_len", len(result.SpeechText))
							}
						} else if result != nil && result.Vision != nil && result.Vision.Content != "" {
							if sess := ls.getSession(); sess != nil {
								contextMsg := fmt.Sprintf("[Screen Context] %s", result.Vision.Content)
								if sendErr := sess.SendText(contextMsg); sendErr != nil {
									slog.Debug("inject screen context failed", "conn_id", c.ID, "error", sendErr)
								} else {
									slog.Info("[HANDLER] injected screen context into live session (no ADK speech)", "conn_id", c.ID, "content_len", len(result.Vision.Content))
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
								slog.Info("[HANDLER] >>> forwarding clientContent to Gemini",
									"conn_id", c.ID,
									"text_len", len(part.Text),
								)
								if sendErr := sess.SendText(part.Text); sendErr != nil {
									slog.Warn("[HANDLER] Gemini SendText failed", "conn_id", c.ID, "error", sendErr)
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

func receiveFromGemini(ctx context.Context, c *Conn, sess *live.Session, ls *liveSessionState, adkClient *adk.Client, ttsClient *tts.Client) {
	turnHasAudio := false

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
				lockedSendJSON(c, map[string]string{"type": "ttsEnd"})
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
							turnHasAudio = true
							ls.setModelSpeaking(true)
							lockedSendJSON(c, map[string]any{"type": "ttsStart", "text": ""})
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

			if sc.OutputTranscription != nil && sc.OutputTranscription.Text != "" {
				lockedSendJSON(c, map[string]any{
					"type":     "transcription",
					"text":     sc.OutputTranscription.Text,
					"finished": sc.OutputTranscription.Finished,
				})
			}

			if sc.InputTranscription != nil && sc.InputTranscription.Text != "" {
				lockedSendJSON(c, map[string]any{
					"type":     "inputTranscription",
					"text":     sc.InputTranscription.Text,
					"finished": sc.InputTranscription.Finished,
				})

				if sc.InputTranscription.Finished && adkClient != nil && couldBeQuestion(sc.InputTranscription.Text) {
					query := sc.InputTranscription.Text
					slog.Info("[HANDLER] potential search query", "conn_id", c.ID, "query", query)
					go func() {
						searchCtx, searchCancel := context.WithTimeout(ctx, 10*time.Second)
						defer searchCancel()
						result, searchErr := adkClient.Search(searchCtx, adk.SearchRequest{
							Query:    query,
							Language: ls.getConfig().Language,
						})
						if searchErr != nil {
							slog.Warn("[HANDLER] voice search failed", "conn_id", c.ID, "error", searchErr)
							return
						}
						if result == nil || result.Summary == "" {
							return
						}
						if ttsClient != nil {
							startTTSStream(ctx, c, ls, ttsClient, result.Summary, result.Summary)
						}
					}()
				}
			}

			if sc.TurnComplete {
				if turnHasAudio {
					turnHasAudio = false
					ls.setModelSpeaking(false)
					lockedSendJSON(c, map[string]string{"type": "ttsEnd"})
				}
				lockedSendJSON(c, map[string]string{"type": "turnComplete"})
			}

			if sc.Interrupted {
				if turnHasAudio {
					turnHasAudio = false
					ls.setModelSpeaking(false)
					lockedSendJSON(c, map[string]string{"type": "ttsEnd"})
				}
				lockedSendJSON(c, map[string]string{"type": "interrupted"})
			}
		}

		if msg.SessionResumptionUpdate != nil {
			ls.setResumeHandle(msg.SessionResumptionUpdate.NewHandle)
			lockedSendJSON(c, map[string]any{
				"type":          "sessionResumptionUpdate",
				"sessionHandle": msg.SessionResumptionUpdate.NewHandle,
			})
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

func startTTSStream(ctx context.Context, c *Conn, ls *liveSessionState, ttsClient *tts.Client, text string, displayText string) {
	ttsCtx, ttsCancel := context.WithCancel(ctx)
	ls.setTTSCancel(ttsCancel)
	ls.setModelSpeaking(true)
	lockedSendJSON(c, map[string]any{"type": "ttsStart", "text": displayText})

	go func() {
		defer func() {
			ls.setTTSCancel(nil)
			ls.setModelSpeaking(false)
			lockedSendJSON(c, map[string]string{"type": "ttsEnd"})
		}()
		cfg := ls.getConfig()
		voice := cfg.Voice
		if voice == "" {
			voice = "Zephyr"
		}
		ttsErr := ttsClient.StreamSpeak(ttsCtx, tts.Config{
			Voice:    voice,
			Language: cfg.Language,
			Text:     text,
		}, func(chunk []byte) error {
			c.mu.Lock()
			defer c.mu.Unlock()
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			return c.conn.WriteMessage(websocket.BinaryMessage, chunk)
		})
		if ttsErr != nil && ttsCtx.Err() == nil {
			slog.Warn("[HANDLER] TTS stream failed", "conn_id", c.ID, "error", ttsErr)
		}
	}()
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
