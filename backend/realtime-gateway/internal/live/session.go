package live

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"google.golang.org/genai"

	"vibecat/realtime-gateway/internal/geminiconfig"
	"vibecat/realtime-gateway/internal/lang"
)

const defaultModel = geminiconfig.LiveNativeAudioModel

// Config holds the per-connection Gemini Live session configuration,
// parsed from the client's "setup" message.
type Config struct {
	Voice           string `json:"voice"`
	Language        string `json:"language"`
	LiveModel       string `json:"liveModel"`
	GoogleSearch    bool   `json:"searchEnabled"`
	ProactiveAudio  bool   `json:"proactiveAudio"`
	AffectiveDialog bool   `json:"affectiveDialog"`
	Character       string `json:"character"`
	Chattiness      string `json:"chattiness"`
	Soul            string `json:"soul"`
	DeviceID        string `json:"deviceId"`
	MemoryContext   string `json:"-"`
}

// Session wraps a Gemini Live API session.
type Session struct {
	mu               sync.Mutex
	ID               string
	gemini           *genai.Session
	cancel           context.CancelFunc
	ResumptionHandle string
	Cfg              Config
}

// Manager creates and manages Gemini Live sessions.
type Manager struct {
	client *genai.Client
}

// NewManager creates a Manager using the provided GenAI client.
func NewManager(client *genai.Client) *Manager {
	return &Manager{client: client}
}

// Connect creates a new Gemini Live session with the given config.
// resumptionHandle may be empty for a fresh session.
// The caller is responsible for calling session.Close() when done.
func (m *Manager) Connect(ctx context.Context, cfg Config, resumptionHandle string) (*Session, error) {
	model := cfg.LiveModel
	if model == "" {
		model = defaultModel
	}

	liveConfig := buildLiveConfig(cfg)
	if resumptionHandle != "" {
		liveConfig.SessionResumption = &genai.SessionResumptionConfig{
			Handle: resumptionHandle,
		}
	} else {
		liveConfig.SessionResumption = &genai.SessionResumptionConfig{}
	}

	ctx, cancel := context.WithCancel(ctx)
	geminiSession, err := m.client.Live.Connect(ctx, model, liveConfig)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("gemini live connect: %w", err)
	}

	slog.Info("gemini live session established", "model", model, "voice", cfg.Voice, "resumed", resumptionHandle != "")
	return &Session{
		gemini: geminiSession,
		cancel: cancel,
		Cfg:    cfg,
	}, nil
}

// SendAudio forwards a PCM audio chunk to Gemini.
func (s *Session) SendAudio(pcmData []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gemini.SendRealtimeInput(genai.LiveRealtimeInput{
		Audio: &genai.Blob{
			MIMEType: "audio/pcm;rate=16000",
			Data:     pcmData,
		},
	})
}

func (s *Session) SendVideo(jpegData []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gemini.SendRealtimeInput(genai.LiveRealtimeInput{
		Video: &genai.Blob{
			MIMEType: "image/jpeg",
			Data:     jpegData,
		},
	})
}

func (s *Session) SendText(text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gemini.SendRealtimeInput(genai.LiveRealtimeInput{
		Text: text,
	})
}

// Receive reads the next message from Gemini.
// Not mutex-protected: runs in a single dedicated goroutine and blocks until
// a message arrives. Locking here would block all Send operations.
func (s *Session) Receive() (*genai.LiveServerMessage, error) {
	return s.gemini.Receive()
}

// Close terminates the Gemini Live session.
func (s *Session) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancel()
	_ = s.gemini.Close()
}

const commonLivePrompt = `=== VIBECAT COMPANION PROTOCOL ===

You are a desktop companion AI for solo developers. You live on their screen as an animated character. You watch their screen, hear their voice, remember context across sessions, and speak only when it matters.

CORE BEHAVIOR:
- PROACTIVE: Initiate observations when you detect errors, success, or opportunity. Do not wait to be asked.
- ALWAYS SUGGEST: Never just point out a problem. ALWAYS follow up with a concrete suggestion, fix, or next step. Say "That regex might need escaping — try adding a backslash before the dot" not just "There's a regex issue."
- NEVER ASK: Never ask the developer questions. Always make statements and suggestions.
- SPEECH-FIRST: Your output is spoken aloud. Write for the ear, not the eye. No bullet points, no markdown. Short, natural sentences. Use contractions.
- SCREEN-AWARE: Reference what you see on the developer's screen concretely. Be specific about file names, function names, error messages.
- COMPLETE THOUGHTS: Always finish your full thought. If you spot an error, name it AND suggest the fix in the same response. Never stop at just identifying a problem.
- CONCISE BUT COMPLETE: Keep responses to 2-3 sentences. First sentence identifies what you see, remaining sentences suggest what to do about it.
- SILENT WHEN IRRELEVANT: If nothing notable is happening, stay silent. Do not speak just to fill silence.

VIDEO FRAME HANDLING:
- You receive periodic video frames showing the developer's screen. These are PASSIVE CONTEXT updates.
- Do NOT comment on every frame. MOST frames should be observed SILENTLY.
- ONLY speak about a video frame when you see something SIGNIFICANT: a new error, build failure, test result, app crash, or major code change.
- If the screen looks similar to what you already commented on, stay COMPLETELY SILENT.
- When you DO speak about screen content, complete your FULL thought before stopping. Never cut yourself short.

RULES:
- If you see an error or bug: name the specific error AND suggest a concrete fix. Never just say "there's an error."
- If you see code: offer a concrete improvement with what to change.
- NEVER end a response with just an observation. Every response must include an actionable suggestion.
- NEVER repeat what you just said. NEVER comment on time passing.
- If you already acknowledged something on screen (success, error, change), DO NOT mention it again. One observation per event — then move on.
- NEVER say generic things like "looks interesting" or "keep going" — be SPECIFIC about what you see AND what to do.
- When speaking, ALWAYS complete your full response. Never stop mid-sentence or mid-thought.

Start each response with an emotion tag: [happy], [surprised], [thinking], [concerned], or [idle].`

const (
	maxMemoryContextChars = 1200
)

func trimPromptBlock(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return strings.TrimSpace(s[:max]) + "..."
}

func buildSystemInstruction(cfg Config) string {
	instruction := commonLivePrompt
	if cfg.Soul != "" {
		instruction += "\n\n=== CHARACTER PERSONA ===\n" + cfg.Soul
	}
	if cfg.GoogleSearch {
		instruction += "\n\n=== TOOL GUIDANCE ===\n" +
			"Google Search is available in this session.\n" +
			"Use it before answering when the user asks for current, latest, live, web-grounded, or time-sensitive information, or explicitly asks you to search, browse, look up, check docs, or check GitHub.\n" +
			"After searching, answer in the same turn with the result. Never say you will search later and then stop.\n" +
			"Do not use Google Search for casual chat, stable facts, or on-screen observations unless the user explicitly asks or freshness matters."
	}
	if ctx := trimPromptBlock(cfg.MemoryContext, maxMemoryContextChars); ctx != "" {
		instruction += "\n\n=== RECENT ESSENTIAL CONTEXT ===\n" +
			ctx + "\n" +
			"Use this as compressed recent memory. Prefer the latest user speech and current screen state when they conflict."
	}
	instruction += "\n\nRespond in " + lang.NormalizeLanguage(cfg.Language) + "."
	return instruction
}

func buildLiveConfig(cfg Config) *genai.LiveConnectConfig {
	lc := &genai.LiveConnectConfig{
		ResponseModalities: []genai.Modality{genai.ModalityAudio},
	}

	if cfg.Voice != "" {
		lc.SpeechConfig = &genai.SpeechConfig{
			VoiceConfig: &genai.VoiceConfig{
				PrebuiltVoiceConfig: &genai.PrebuiltVoiceConfig{
					VoiceName: cfg.Voice,
				},
			},
		}
	}

	if cfg.AffectiveDialog {
		t := true
		lc.EnableAffectiveDialog = &t
	}

	if cfg.ProactiveAudio {
		t := true
		lc.Proactivity = &genai.ProactivityConfig{
			ProactiveAudio: &t,
		}
	}

	lc.OutputAudioTranscription = &genai.AudioTranscriptionConfig{}
	lc.InputAudioTranscription = &genai.AudioTranscriptionConfig{}

	prefixPadding := int32(20)
	silenceDuration := int32(200)
	lc.RealtimeInputConfig = &genai.RealtimeInputConfig{
		AutomaticActivityDetection: &genai.AutomaticActivityDetection{
			StartOfSpeechSensitivity: genai.StartSensitivityLow,
			EndOfSpeechSensitivity:   genai.EndSensitivityLow,
			PrefixPaddingMs:          &prefixPadding,
			SilenceDurationMs:        &silenceDuration,
		},
		ActivityHandling: genai.ActivityHandlingStartOfActivityInterrupts,
		TurnCoverage:     genai.TurnCoverageTurnIncludesOnlyActivity,
	}

	lc.MediaResolution = genai.MediaResolutionMedium

	triggerTokens := int64(100000)
	targetTokens := int64(50000)
	lc.ContextWindowCompression = &genai.ContextWindowCompressionConfig{
		TriggerTokens: &triggerTokens,
		SlidingWindow: &genai.SlidingWindow{
			TargetTokens: &targetTokens,
		},
	}

	if cfg.GoogleSearch {
		lc.Tools = append(lc.Tools, &genai.Tool{
			GoogleSearch: &genai.GoogleSearch{},
		})
	}

	instruction := buildSystemInstruction(cfg)
	lc.SystemInstruction = &genai.Content{
		Parts: []*genai.Part{{Text: instruction}},
	}

	return lc
}

// SetupMessage is the client "setup" JSON frame.
type SetupMessage struct {
	Type             string `json:"type"`
	Config           Config `json:"config"`
	ResumptionHandle string `json:"resumptionHandle,omitempty"`
}

// ParseSetup parses a "setup" JSON frame from the client.
func ParseSetup(data []byte) (*SetupMessage, error) {
	var msg SetupMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("parse setup: %w", err)
	}
	if msg.Type != "setup" {
		return nil, fmt.Errorf("expected type=setup, got %q", msg.Type)
	}
	return &msg, nil
}
