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

	slog.Info("gemini live session established", "model", model, "voice", cfg.Voice, "resumed", resumptionHandle != "", "tuning_profile", activeTuningProfile.Name)
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

const commonLivePrompt = `=== VIBECAT NAVIGATOR PROTOCOL ===

You are VibeCat, a desktop UI navigator for developer workflows on macOS.

ROLE:
- Help the user understand the current state and the best next step.
- Do not act like a proactive companion.
- Assume actual UI execution is handled by a separate local executor.
- In this channel, your job is explanation, summary, and short guidance.

STYLE:
- Speech-first. Write for the ear.
- One short sentence is preferred. A second short sentence is allowed only when it adds one concrete next step.
- Be specific about windows, files, errors, tabs, controls, and commands.
- Do not claim you clicked or changed anything unless the runtime explicitly says that happened.

VIDEO FRAME HANDLING:
- Video frames are passive context.
- Do not start unsolicited commentary just because a frame arrived.
- Use screen context only to make the user's requested answer more precise.

RULES:
- No markdown and no bullet points.
- Never invent completed actions.
- If the user's request is ambiguous, prefer a short clarifying sentence.
- If the user asks what to do next, give the single best next step.

Start each response with an emotion tag: [happy], [surprised], [thinking], [concerned], or [idle].`

type tuningProfile struct {
	Name              string
	MaxMemoryChars    int
	PrefixPaddingMs   int32
	SilenceDurationMs int32
	TriggerTokens     int64
	TargetTokens      int64
}

var (
	baselineTuningProfile = tuningProfile{
		Name:              "baseline",
		MaxMemoryChars:    1200,
		PrefixPaddingMs:   20,
		SilenceDurationMs: 200,
		TriggerTokens:     12000,
		TargetTokens:      6000,
	}
	memoryLightTuningProfile = tuningProfile{
		Name:              "memory_light",
		MaxMemoryChars:    900,
		PrefixPaddingMs:   20,
		SilenceDurationMs: 200,
		TriggerTokens:     10000,
		TargetTokens:      5000,
	}
	vadRelaxedTuningProfile = tuningProfile{
		Name:              "vad_relaxed",
		MaxMemoryChars:    1200,
		PrefixPaddingMs:   40,
		SilenceDurationMs: 250,
		TriggerTokens:     12000,
		TargetTokens:      6000,
	}
	activeTuningProfile = baselineTuningProfile
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
			"For grounded search answers, give the direct answer in the first sentence and stop after one short follow-up sentence at most.\n" +
			"Do not use Google Search for casual chat, stable facts, or on-screen observations unless the user explicitly asks or freshness matters."
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Chattiness)) {
	case "quiet":
		instruction += "\n\n=== RESPONSE LENGTH ===\n" +
			"Keep responses to exactly one short spoken sentence whenever possible. Do not add a second sentence unless it is required to make the answer correct."
	case "chatty":
		instruction += "\n\n=== RESPONSE LENGTH ===\n" +
			"You may use up to two short spoken sentences and one concrete next step when it materially helps. Stay concise."
	default:
		instruction += "\n\n=== RESPONSE LENGTH ===\n" +
			"Keep responses to 1-2 short sentences. Prefer one short spoken sentence, and use the second only when it adds one concrete next step."
	}
	if ctx := trimPromptBlock(cfg.MemoryContext, activeTuningProfile.MaxMemoryChars); ctx != "" {
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

	prefixPadding := activeTuningProfile.PrefixPaddingMs
	silenceDuration := activeTuningProfile.SilenceDurationMs
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

	triggerTokens := activeTuningProfile.TriggerTokens
	targetTokens := activeTuningProfile.TargetTokens
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
