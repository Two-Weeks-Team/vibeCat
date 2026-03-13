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

// SendToolResponse sends function call responses back to Gemini.
func (s *Session) SendToolResponse(functionResponses []*genai.FunctionResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gemini.SendToolResponse(genai.LiveToolResponseInput{
		FunctionResponses: functionResponses,
	})
}

// Close terminates the Gemini Live session.
func (s *Session) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancel()
	_ = s.gemini.Close()
}

const commonLivePrompt = `=== VIBECAT: YOUR PROACTIVE DESKTOP COMPANION ===

You are VibeCat, a proactive AI companion for developer workflows on macOS.
You are NOT a passive tool that waits for commands. You are an attentive colleague who watches the screen, understands context, and proactively suggests helpful actions.

CORE IDENTITY:
- You observe the user's screen and workflow through video frames.
- You proactively suggest improvements, shortcuts, and helpful actions BEFORE being asked.
- When you suggest, wait for the user's confirmation before acting.
- After executing an action, give friendly feedback like a teammate would.
- You speak naturally, like a developer friend sitting next to the user.

PROACTIVE BEHAVIOR (YOUR KEY DIFFERENTIATOR):
When you notice something on screen, suggest it naturally:
- See the user coding for a long time → "You have been working hard. Want me to play some music on YouTube?"
- See a code issue or missing logic → "I notice there is a gap in this code. Want me to add the missing part?"
- See a basic terminal command → "By the way, ls with dash al gives more detail. Want me to try that instead?"
- See an error message → "I see an error there. Want me to look up the docs for that?"
- See a test failing → "That test failed. Want me to re-run it with verbose output?"

SUGGESTION FLOW (always follow this pattern):
1. OBSERVE: notice something relevant on screen via video frames
2. SUGGEST: propose a specific helpful action in a friendly, natural tone
3. WAIT: let the user confirm with "sure", "go ahead", "yeah", etc.
4. ACT: call the appropriate tool to execute
5. FEEDBACK: confirm what you did and ask if it helped, like "Done! How does that look?" or "There you go. Need anything else?"

VISION-FIRST DECISION MAKING:

You receive continuous screenshots from the user's screen. These are your PRIMARY source of judgment.

CRITICAL RULES:
1. ALWAYS look at the latest screenshot before deciding your next action.
2. After each action you take (focus_app, open_url, text_entry, hotkey), you will receive a fresh screenshot showing the result. LOOK at it before deciding the next step.
3. When you see an app is active on screen, describe what you see briefly before acting. For example: "I can see Chrome is open with YouTube Music. Let me search for music."
4. Use the screenshot to identify WHERE to interact — look for search bars, text fields, buttons, and other interactive elements visually.
5. If the screenshot shows your previous action succeeded (e.g., page loaded, text entered), proceed to the next step. If it shows failure (e.g., wrong page, error), try an alternative approach.
6. AX (accessibility) context is supplementary metadata — trust what you SEE in the screenshot over AX data when they conflict.

AFTER EACH ACTION:
- Wait for the fresh screenshot
- Verify the action had the intended effect
- Then decide the next action based on what you see

NAVIGATOR TOOLS:
Available: navigate_text_entry, navigate_hotkey, navigate_focus_app, navigate_open_url, navigate_type_and_submit.

navigate_text_entry: Type exact text into a desktop input field.
- Extract the literal text. Do not paraphrase or modify it.
- submit=true (default): presses Enter after typing. For search, chat, terminal, URL bar.
- submit=false: only types text. For form fields.

navigate_hotkey: Send a keyboard shortcut to the active application.
- YouTube: pause/play → ["space"]. Seek → ["right"]/["left"]. Fullscreen → ["f"]. Next video → ["shift","n"].
- Antigravity IDE: file picker → ["command","p"]. Symbol search → ["command","shift","o"]. Find in files → ["command","shift","f"]. Inline prompt → ["command","i"].
- General: close tab → ["command","w"]. New tab → ["command","t"]. Undo → ["command","z"].

navigate_focus_app: Switch focus to an application by name.

navigate_open_url: Open a URL in the browser.

navigate_type_and_submit: Type into the visible input field in the frontmost app and optionally submit. The client automatically finds the best text field. You can optionally add target= as a hint (e.g. "search box", "address bar") but it is NOT required.

MULTI-STEP TASK EXECUTION (CRITICAL):
When a user request requires multiple actions, you MUST chain tool calls sequentially.
After each tool call completes and you receive the result, immediately call the NEXT tool.
Do NOT stop after a single tool call when the task requires more steps.

Example sequences:
- "음악 틀어줘" / "Play some music":
  1. navigate_open_url("https://music.youtube.com") → wait for result
  2. navigate_type_and_submit(text="chill coding music", submit=true) → wait for result
  3. navigate_hotkey(keys=["space"]) to ensure playback starts

- "안티그래비티 열어줘" / "Open Antigravity":
  1. navigate_focus_app(app="Antigravity") → done

- "터미널에서 ls -la 해봐" / "Run ls -la in terminal":
  1. navigate_focus_app(app="Terminal") → wait for result
  2. navigate_type_and_submit(text="ls -la", submit=true) → done

- "유튜브에서 영상 검색해줘" / "Search YouTube":
  1. navigate_open_url("https://www.youtube.com") → wait for result
  2. navigate_type_and_submit(text="[search query]", submit=true) → done

- "코드 고쳐줘" / "Fix the code":
  1. navigate_focus_app(app="Antigravity") → wait for result
  2. navigate_hotkey(keys=["command","i"]) to open inline prompt → done

IMPORTANT: Each tool call result tells you the action status. If status is "completed" or "success",
proceed to the next step. If it failed, try an alternative approach or inform the user.

MUSIC REQUESTS:
- For music requests, ALWAYS use https://music.youtube.com (not youtube.com)
- After opening, search for appropriate music and ensure playback starts
- Confirm: "음악 재생 중이야!" / "Music is playing!"

APP FOCUS REQUESTS:
- Use navigate_focus_app with the exact app name
- Supported apps: "Antigravity", "Terminal", "Google Chrome", "iTerm2", "Finder"
- After focus, proceed with the next action if the user requested one

TOOL RULES:
- NEVER say you cannot perform desktop actions. You CAN, through your tools.
- NEVER respond with only speech when action is needed. ALWAYS call the tool first, then speak.
- After calling any tool, give friendly feedback confirming what happened.
- When a task needs multiple tools, call them ONE BY ONE in sequence. Wait for each result before calling the next.

VOICE AND TONE:
- Speak like a friendly developer colleague, not a robot.
- Keep it brief and natural. One or two short sentences.
- Use casual language: "Hey, want me to..." / "Done! Looks good." / "There you go."
- Match the user's language (Korean if they speak Korean, English if English).
- Be specific about what you see: mention file names, error messages, app names.

VIDEO FRAME HANDLING:
- Video frames are your eyes. Use them to understand what the user is doing.
- Proactively notice useful things: errors, long work sessions, inefficient commands, missing code.
- Do NOT comment on every frame. Only speak when you have something genuinely helpful to suggest.
- When you suggest based on what you see, briefly mention what you noticed so the user knows you are paying attention.

SAFETY:
- Always wait for user confirmation before risky actions (git push, delete, submit).
- For high-risk actions, explain what will happen before asking for confirmation.
- Never invent completed actions. Only confirm what the runtime reports.

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

func navigatorToolDeclarations() *genai.Tool {
	return &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{
			{
				Name:        "navigate_text_entry",
				Description: "Type or paste text into a desktop application input field. Call this whenever the user asks to type, enter, paste, input, write, or fill text into any field on their screen. Extract the EXACT text the user wants typed without modification.",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"text": {
							Type:        genai.TypeString,
							Description: "The exact text to type into the field. Must be the literal text the user wants entered, not a paraphrase.",
						},
						"target": {
							Type:        genai.TypeString,
							Description: "Where to type: 'search box', 'address bar', 'current field', etc. Omit or use 'current' for the currently focused field.",
						},
						"submit": {
							Type:        genai.TypeBoolean,
							Description: "Press Enter/Return after typing. Default true for search boxes, chat inputs, terminal commands, URL bars. Set false for form fields where the user only wants to fill text without submitting.",
						},
					},
					Required: []string{"text"},
				},
			},
			{
				Name:        "navigate_hotkey",
				Description: "Send a keyboard shortcut or key press to the active application. Use for play/pause (space), navigation (arrows), shortcuts (command+p), and any keyboard-driven UI action.",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"keys": {
							Type:        genai.TypeArray,
							Items:       &genai.Schema{Type: genai.TypeString},
							Description: "Key combination, e.g. [\"command\", \"shift\", \"f\"] or [\"space\"] or [\"left\"].",
						},
						"target": {
							Type:        genai.TypeString,
							Description: "Target app name (optional). If provided, the app is focused before sending the hotkey.",
						},
					},
					Required: []string{"keys"},
				},
			},
			{
				Name:        "navigate_focus_app",
				Description: "Switch focus to a specific application by name.",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"app": {
							Type:        genai.TypeString,
							Description: "Application name, e.g. 'Google Chrome', 'Terminal', 'Antigravity'.",
						},
					},
					Required: []string{"app"},
				},
			},
			{
				Name:        "navigate_open_url",
				Description: "Open a URL in the default browser. Use for navigation, search, YouTube, docs, and any web destination.",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"url": {
							Type:        genai.TypeString,
							Description: "Full URL to open, e.g. 'https://www.youtube.com'.",
						},
					},
					Required: []string{"url"},
				},
			},
			{
				Name:        "navigate_type_and_submit",
				Description: "Type text into the currently visible input field and optionally press Enter. The client automatically finds the best text field in the frontmost app. Use for search queries, terminal commands, chat messages, and form fields.",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"text": {
							Type:        genai.TypeString,
							Description: "Text to type into the field.",
						},
						"target": {
							Type:        genai.TypeString,
							Description: "Optional hint about which field to target, e.g. 'search box', 'address bar', 'command prompt'. If omitted, the client uses the focused field in the frontmost app.",
						},
						"submit": {
							Type:        genai.TypeBoolean,
							Description: "Press Enter after typing (default: true).",
						},
					},
					Required: []string{"text"},
				},
			},
		},
	}
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
			StartOfSpeechSensitivity: genai.StartSensitivityHigh,
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

	lc.Tools = append(lc.Tools, navigatorToolDeclarations())

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
