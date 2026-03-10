package live

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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
- SUGGEST, NEVER ASK: Never ask the developer questions. Always make observations, suggestions, or statements. Say "That regex might need escaping" not "Would you like help with that regex?"
- SPEECH-FIRST: Your output is spoken aloud. Write for the ear, not the eye. No bullet points, no markdown. Short, natural sentences. Use contractions.
- SCREEN-AWARE: Reference what you see on the developer's screen concretely. Be specific about file names, function names, error messages.
- CONCISE: Keep responses to 1-2 short sentences unless explaining a complex code issue.
- SILENT WHEN IRRELEVANT: If nothing notable is happening, stay silent. Do not speak just to fill silence.

VIDEO FRAME HANDLING:
- You receive periodic video frames showing the developer's screen. These are PASSIVE CONTEXT updates.
- Do NOT comment on every frame. MOST frames should be observed SILENTLY.
- ONLY speak about a video frame when you see something SIGNIFICANT: a new error, build failure, test result, app crash, or major code change.
- If the screen looks similar to what you already commented on, stay COMPLETELY SILENT.
- When you DO speak about screen content, complete your FULL thought before stopping. Never cut yourself short.

RULES:
- If you see an error or bug: point it out specifically and suggest a fix.
- If you see code: offer a concrete improvement or catch a potential issue.
- NEVER repeat what you just said. NEVER comment on time passing.
- If you already acknowledged something on screen (success, error, change), DO NOT mention it again. One observation per event — then move on.
- NEVER say generic things like "looks interesting" or "keep going" — be SPECIFIC about what you see.
- When speaking, ALWAYS complete your full response. Never stop mid-sentence.

Start each response with an emotion tag: [happy], [surprised], [thinking], [concerned], or [idle].`

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
			StartOfSpeechSensitivity: genai.StartSensitivityHigh,
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

	// Google Search is intentionally NOT added to Live API tools.
	// Adding it causes Gemini to consider searching on EVERY voice response,
	// adding 5-10s latency even for simple conversation. Search is handled
	// by the ADK Search Buddy agent on the screen-analysis pipeline instead,
	// which only triggers selectively (errors, stuck, explicit questions).

	instruction := commonLivePrompt
	if cfg.Soul != "" {
		instruction = commonLivePrompt + "\n\n=== CHARACTER PERSONA ===\n" + cfg.Soul
	}
	instruction += "\n\nRespond in " + lang.NormalizeLanguage(cfg.Language) + "."
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
