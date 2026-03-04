package ws

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"vibecat/realtime-gateway/internal/adk"
	"vibecat/realtime-gateway/internal/live"
)

const (
	pingInterval  = 15 * time.Second
	zombieTimeout = 45 * time.Second
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Conn represents a single WebSocket client connection.
type Conn struct {
	ID   string
	conn *websocket.Conn
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

func newConnID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
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

		defer func() {
			reg.Remove(c.ID)
			rawConn.Close()
			slog.Info("websocket disconnected", "conn_id", c.ID)
		}()

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		rawConn.SetReadDeadline(time.Now().Add(zombieTimeout))
		rawConn.SetPongHandler(func(string) error {
			rawConn.SetReadDeadline(time.Now().Add(zombieTimeout))
			return nil
		})

		go func() {
			ticker := time.NewTicker(pingInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := rawConn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
						slog.Warn("ping failed", "conn_id", c.ID, "error", err)
						cancel()
						return
					}
				}
			}
		}()

		var liveSession *live.Session

		for {
			msgType, data, readErr := rawConn.ReadMessage()
			if readErr != nil {
				if websocket.IsUnexpectedCloseError(readErr, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					slog.Warn("websocket read error", "conn_id", c.ID, "error", readErr)
				}
				if liveSession != nil {
					liveSession.Close()
				}
				return
			}

			switch msgType {
			case websocket.BinaryMessage:
				if liveSession != nil {
					if sendErr := liveSession.SendAudio(data); sendErr != nil {
						slog.Warn("send audio to gemini failed", "conn_id", c.ID, "error", sendErr)
					}
				} else {
					_ = rawConn.WriteMessage(websocket.BinaryMessage, data)
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
						sendJSON(rawConn, setupCompleteMsg{Type: "setupComplete", SessionID: c.ID})
						continue
					}
					setupMsg, parseErr := live.ParseSetup(data)
					if parseErr != nil {
						slog.Error("parse setup failed", "conn_id", c.ID, "error", parseErr)
						sendJSON(rawConn, errorMsg{Type: "error", Code: "SETUP_FAILED", Message: parseErr.Error()})
						continue
					}
					sess, connectErr := liveMgr.Connect(ctx, setupMsg.Config, setupMsg.ResumptionHandle)
					if connectErr != nil {
						slog.Error("gemini connect failed", "conn_id", c.ID, "error", connectErr)
						sendJSON(rawConn, errorMsg{Type: "error", Code: "GEMINI_CONNECT_FAILED", Message: connectErr.Error()})
						continue
					}
					liveSession = sess
					sendJSON(rawConn, setupCompleteMsg{Type: "setupComplete", SessionID: c.ID})
					go receiveFromGemini(ctx, rawConn, liveSession, c.ID)

				case "settingsUpdate":
					slog.Info("settings update received", "conn_id", c.ID)
					if liveSession != nil {
						liveSession.Close()
						liveSession = nil
					}

				case "screenCapture", "forceCapture":
					if adkClient == nil {
						continue
					}
					var captureMsg struct {
						Type      string `json:"type"`
						Image     string `json:"image"`
						Context   string `json:"context"`
						SessionID string `json:"sessionId"`
						UserID    string `json:"userId"`
					}
					if parseErr := json.Unmarshal(data, &captureMsg); parseErr != nil {
						slog.Warn("parse capture message failed", "conn_id", c.ID, "error", parseErr)
						continue
					}
					go func() {
						analyzeCtx, analyzeCancel := context.WithTimeout(ctx, 5*time.Second)
						defer analyzeCancel()
						result, analyzeErr := adkClient.Analyze(analyzeCtx, adk.AnalysisRequest{
							Image:     captureMsg.Image,
							Context:   captureMsg.Context,
							SessionID: captureMsg.SessionID,
							UserID:    captureMsg.UserID,
						})
						if analyzeErr != nil {
							slog.Warn("adk analyze failed", "conn_id", c.ID, "error", analyzeErr)
							return
						}
						if result != nil && result.Decision != nil && result.Decision.ShouldSpeak && result.SpeechText != "" {
							sendJSON(rawConn, map[string]any{
								"type": "companionSpeech",
								"text": result.SpeechText,
								"emotion": func() string {
									if result.Vision != nil {
										return result.Vision.Emotion
									}
									return "neutral"
								}(),
								"urgency": result.Decision.Urgency,
							})
						}
					}()

				case "ping":
					sendJSON(rawConn, map[string]string{"type": "pong"})
				}
			}
		}
	}
}

// receiveFromGemini reads messages from Gemini and forwards them to the client.
func receiveFromGemini(ctx context.Context, conn *websocket.Conn, sess *live.Session, connID string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		msg, err := sess.Receive()
		if err != nil {
			slog.Warn("gemini receive error", "conn_id", connID, "error", err)
			return
		}
		if msg == nil {
			continue
		}

		if msg.SetupComplete != nil {
			slog.Info("gemini setup complete", "conn_id", connID)
			continue
		}

		if msg.ServerContent != nil {
			sc := msg.ServerContent

			if sc.ModelTurn != nil {
				for _, part := range sc.ModelTurn.Parts {
					if part.InlineData != nil && len(part.InlineData.Data) > 0 {
						if writeErr := conn.WriteMessage(websocket.BinaryMessage, part.InlineData.Data); writeErr != nil {
							slog.Warn("write audio to client failed", "conn_id", connID, "error", writeErr)
							return
						}
					}
				}
			}

			if sc.OutputTranscription != nil && sc.OutputTranscription.Text != "" {
				sendJSON(conn, map[string]any{
					"type":     "transcription",
					"text":     sc.OutputTranscription.Text,
					"finished": sc.TurnComplete,
				})
			}

			if sc.TurnComplete {
				sendJSON(conn, map[string]string{"type": "turnComplete"})
			}

			if sc.Interrupted {
				sendJSON(conn, map[string]string{"type": "interrupted"})
			}
		}

		if msg.SessionResumptionUpdate != nil && msg.SessionResumptionUpdate.NewHandle != "" {
			sess.ResumptionHandle = msg.SessionResumptionUpdate.NewHandle
			sendJSON(conn, map[string]any{
				"type":             "setupComplete",
				"sessionId":        connID,
				"resumptionHandle": sess.ResumptionHandle,
			})
		}

		if msg.GoAway != nil {
			sendJSON(conn, map[string]any{
				"type":       "goAway",
				"reason":     "session_timeout",
				"timeLeftMs": msg.GoAway.TimeLeft.Milliseconds(),
			})
		}
	}
}

func sendJSON(conn *websocket.Conn, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		slog.Error("marshal json failed", "error", err)
		return
	}
	if writeErr := conn.WriteMessage(websocket.TextMessage, data); writeErr != nil {
		slog.Warn("write json to client failed", "error", writeErr)
	}
}
