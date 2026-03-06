package mediator

import (
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"
	"math/rand"
	"strings"
	"sync"
	"time"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"vibecat/adk-orchestrator/internal/models"
)

const (
	defaultCooldown  = 10 * time.Second
	moodCooldown     = 180 * time.Second
	highSignificance = 7
	lowSignificance  = 3
)

const (
	maxRecentSpeech = 5
	moodGenModel    = "gemini-3.1-flash-lite-preview"
)

type Agent struct {
	mu                sync.Mutex
	genaiClient       *genai.Client
	lastSpoke         time.Time
	lastMoodSpoke     time.Time
	cooldown          time.Duration
	recentSpeechTexts []string
}

func New(genaiClient *genai.Client) *Agent {
	return &Agent{genaiClient: genaiClient, cooldown: defaultCooldown}
}

func (a *Agent) Run(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
	return func(yield func(*session.Event, error) bool) {
		vision, mood, celebration := readInputsFromState(ctx)
		if vision == nil && mood == nil && celebration == nil {
			vision, mood, celebration = readInputsFromUserContent(ctx)
		}

		slog.Info("[MEDIATOR] inputs",
			"has_vision", vision != nil,
			"has_mood", mood != nil,
			"has_celebration", celebration != nil,
			"vision_significance", func() int {
				if vision != nil {
					return vision.Significance
				}
				return -1
			}(),
			"vision_shouldSpeak", func() bool {
				if vision != nil {
					return vision.ShouldSpeak
				}
				return false
			}(),
			"vision_content", func() string {
				if vision != nil {
					return truncate(vision.Content, 80)
				}
				return ""
			}(),
			"mood_mood", func() string {
				if mood != nil {
					return mood.Mood
				}
				return ""
			}(),
			"mood_confidence", func() float64 {
				if mood != nil {
					return mood.Confidence
				}
				return 0
			}(),
			"celebration_msg", func() string {
				if celebration != nil {
					return celebration.Message
				}
				return ""
			}(),
		)

		decision := a.decide(vision, mood, celebration)

		result := models.AnalysisResult{
			Vision:      vision,
			Decision:    decision,
			Mood:        mood,
			Celebration: celebration,
		}

		if decision.ShouldSpeak && vision != nil {
			if a.isSimilarToRecent(vision.Content) {
				decision.ShouldSpeak = false
				decision.Reason = "duplicate_content"
				slog.Info("[MEDIATOR] speech BLOCKED (similar to recent)", "text", truncate(vision.Content, 80))
			} else {
				result.SpeechText = vision.Content
				slog.Info("[MEDIATOR] speech from vision", "text", truncate(vision.Content, 80))
			}
		}
		if celebration != nil && celebration.Message != "" && (vision == nil || vision.Significance < 7) {
			result.SpeechText = celebration.Message
			slog.Info("[MEDIATOR] speech overridden by celebration", "text", celebration.Message)
		}

		if mood != nil && !decision.ShouldSpeak {
			language := readLanguageFromState(ctx)
			a.mu.Lock()
			sinceMood := time.Since(a.lastMoodSpoke)
			if sinceMood > moodCooldown {
				a.mu.Unlock()
				msg := a.generateMoodMessage(ctx, mood, vision, language)
				if msg != "" {
					a.mu.Lock()
					result.SpeechText = msg
					decision.ShouldSpeak = true
					decision.Reason = "mood_support"
					a.lastMoodSpoke = time.Now()
					a.lastSpoke = time.Now()
					a.mu.Unlock()
					slog.Info("[MEDIATOR] mood support triggered", "mood", mood.Mood, "text", msg, "since_last_mood", sinceMood.String())
				}
			} else {
				a.mu.Unlock()
				slog.Info("[MEDIATOR] mood support SKIPPED (mood cooldown)", "mood", mood.Mood, "since_last_mood", sinceMood.String(), "mood_cooldown", moodCooldown.String())
			}
		}

		if result.SpeechText != "" {
			a.trackSpeechText(result.SpeechText)
		}

		slog.Info("[MEDIATOR] final decision",
			"should_speak", decision.ShouldSpeak,
			"reason", decision.Reason,
			"urgency", decision.Urgency,
			"speech_text", truncate(result.SpeechText, 80),
		)

		data, err := json.Marshal(result)
		if err != nil {
			yield(nil, fmt.Errorf("mediator marshal: %w", err))
			return
		}

		yield(&session.Event{
			Actions: session.EventActions{
				StateDelta: map[string]any{"analysis_result": string(data)},
			},
			LLMResponse: model.LLMResponse{
				Content: &genai.Content{
					Parts: []*genai.Part{{Text: string(data)}},
				},
			},
		}, nil)
	}
}

func readInputsFromState(ctx agent.InvocationContext) (*models.VisionAnalysis, *models.MoodState, *models.CelebrationEvent) {
	sess := ctx.Session()
	if sess == nil || sess.State() == nil {
		return nil, nil, nil
	}

	var vision *models.VisionAnalysis
	if val, err := sess.State().Get("vision_analysis"); err == nil {
		var v models.VisionAnalysis
		if decodeStateJSON(val, &v) {
			vision = &v
		}
	}

	var mood *models.MoodState
	if val, err := sess.State().Get("mood_state"); err == nil {
		var m models.MoodState
		if decodeStateJSON(val, &m) {
			mood = &m
		}
	}

	var celebration *models.CelebrationEvent
	if val, err := sess.State().Get("celebration_event"); err == nil {
		var c models.CelebrationEvent
		if decodeStateJSON(val, &c) && c.Message != "" {
			celebration = &c
		}
	}

	return vision, mood, celebration
}

func readInputsFromUserContent(ctx agent.InvocationContext) (*models.VisionAnalysis, *models.MoodState, *models.CelebrationEvent) {
	userContent := ctx.UserContent()
	if userContent == nil {
		return nil, nil, nil
	}

	var vision *models.VisionAnalysis
	var mood *models.MoodState
	var celebration *models.CelebrationEvent

	for _, part := range userContent.Parts {
		if part.Text == "" {
			continue
		}
		var result models.AnalysisResult
		if err := json.Unmarshal([]byte(part.Text), &result); err == nil {
			vision = result.Vision
			mood = result.Mood
			celebration = result.Celebration
		}
	}

	return vision, mood, celebration
}

func decodeStateJSON(v any, out any) bool {
	switch data := v.(type) {
	case string:
		return json.Unmarshal([]byte(data), out) == nil
	case []byte:
		return json.Unmarshal(data, out) == nil
	default:
		b, err := json.Marshal(data)
		if err != nil {
			return false
		}
		return json.Unmarshal(b, out) == nil
	}
}

func (a *Agent) decide(vision *models.VisionAnalysis, mood *models.MoodState, celebration *models.CelebrationEvent) *models.MediatorDecision {
	a.mu.Lock()
	defer a.mu.Unlock()

	if celebration != nil {
		slog.Info("[MEDIATOR] decide: celebration bypass")
		return &models.MediatorDecision{ShouldSpeak: true, Reason: "celebration", Urgency: "high"}
	}

	if mood != nil && mood.Mood == models.MoodFocused && mood.Confidence >= 0.5 {
		a.cooldown = 30 * time.Second
	} else {
		a.cooldown = defaultCooldown
	}

	sinceLast := time.Since(a.lastSpoke)
	if sinceLast < a.cooldown {
		slog.Info("[MEDIATOR] decide: cooldown active", "since_last", sinceLast.String(), "cooldown", a.cooldown.String(), "flow_extended", a.cooldown > defaultCooldown)
		return &models.MediatorDecision{ShouldSpeak: false, Reason: "cooldown", Urgency: "low"}
	}

	if vision == nil {
		slog.Info("[MEDIATOR] decide: no vision data")
		return &models.MediatorDecision{ShouldSpeak: false, Reason: "no_vision", Urgency: "low"}
	}

	if !vision.ShouldSpeak {
		threshold := highSignificance
		if mood != nil && mood.Mood == models.MoodFrustrated {
			threshold = lowSignificance
		}
		if mood != nil && mood.Mood == models.MoodFocused {
			threshold = 9
		}
		if vision.Significance < threshold {
			slog.Info("[MEDIATOR] decide: below threshold", "significance", vision.Significance, "threshold", threshold, "mood", func() string {
				if mood != nil {
					return mood.Mood
				}
				return "none"
			}())
			return &models.MediatorDecision{ShouldSpeak: false, Reason: "low_significance", Urgency: "low"}
		}
	}

	a.lastSpoke = time.Now()

	urgency := "normal"
	if vision.ErrorDetected || vision.RepeatedError {
		urgency = "high"
	}

	slog.Info("[MEDIATOR] decide: SPEAK", "reason", "significant_event", "urgency", urgency, "significance", vision.Significance)
	return &models.MediatorDecision{ShouldSpeak: true, Reason: "significant_event", Urgency: urgency}
}

func (a *Agent) generateMoodMessage(ctx agent.InvocationContext, mood *models.MoodState, vision *models.VisionAnalysis, language string) string {
	if a.genaiClient != nil {
		msg := a.generateDynamic(ctx, mood, vision, language)
		if msg != "" {
			return msg
		}
		slog.Warn("[MEDIATOR] dynamic mood message failed, falling back to static pool")
	}
	return fallbackSupportiveMessage(mood.Mood, language)
}

func (a *Agent) generateDynamic(ctx agent.InvocationContext, mood *models.MoodState, vision *models.VisionAnalysis, language string) string {
	lang := normalizeLanguage(language)

	var context strings.Builder
	context.WriteString(fmt.Sprintf("Developer mood: %s\n", mood.Mood))
	if len(mood.Signals) > 0 {
		context.WriteString(fmt.Sprintf("Signals: %s\n", strings.Join(mood.Signals, ", ")))
	}
	if vision != nil {
		if vision.ErrorMessage != "" {
			context.WriteString(fmt.Sprintf("Current error: %s\n", truncate(vision.ErrorMessage, 150)))
		}
		if vision.Content != "" {
			context.WriteString(fmt.Sprintf("Screen context: %s\n", truncate(vision.Content, 150)))
		}
	}

	a.mu.Lock()
	recentCopy := make([]string, len(a.recentSpeechTexts))
	copy(recentCopy, a.recentSpeechTexts)
	a.mu.Unlock()

	avoidList := ""
	if len(recentCopy) > 0 {
		avoidList = fmt.Sprintf("\nDo NOT repeat or paraphrase these recent messages:\n- %s", strings.Join(recentCopy, "\n- "))
	}

	prompt := fmt.Sprintf(`You are VibeCat, a coding companion for solo developers.
The developer seems %s right now. Generate ONE short supportive message (under 80 characters).

%s%s

Rules:
- Be specific to the situation when possible (reference the error or screen context)
- Be warm and practical, not generic
- Suggest a concrete next step when appropriate
- Do NOT start with "비슷한 문제" or any repeated pattern
- Respond in %s
- Return ONLY the message text, nothing else`, mood.Mood, context.String(), avoidList, lang)

	resp, err := a.genaiClient.Models.GenerateContent(ctx, moodGenModel, []*genai.Content{
		{Parts: []*genai.Part{{Text: prompt}}, Role: genai.RoleUser},
	}, &genai.GenerateContentConfig{
		MaxOutputTokens: 100,
	})
	if err != nil {
		slog.Warn("[MEDIATOR] mood LLM call failed", "error", err)
		return ""
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return ""
	}

	text := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		text += part.Text
	}
	text = strings.TrimSpace(text)
	if len(text) > 200 {
		text = text[:200]
	}
	if text == "" {
		return ""
	}
	return text
}

func normalizeLanguage(language string) string {
	trimmed := strings.TrimSpace(language)
	if trimmed == "" {
		return "Korean"
	}
	lower := strings.ToLower(trimmed)
	switch lower {
	case "ko", "kr", "korean", "korean language", "한국어":
		return "Korean"
	case "en", "eng", "english", "english language":
		return "English"
	default:
		return trimmed
	}
}

func fallbackSupportiveMessage(mood, language string) string {
	langMsgs, ok := models.SupportiveMessages[mood]
	if !ok {
		return ""
	}
	msgs := langMsgs[language]
	if len(msgs) == 0 {
		msgs = langMsgs["Korean"]
	}
	if len(msgs) == 0 {
		return ""
	}
	return msgs[rand.Intn(len(msgs))]
}

func readLanguageFromState(ctx agent.InvocationContext) string {
	sess := ctx.Session()
	if sess == nil || sess.State() == nil {
		return "Korean"
	}
	val, err := sess.State().Get("language")
	if err != nil {
		return "Korean"
	}
	if s, ok := val.(string); ok {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			return trimmed
		}
	}
	return "Korean"
}

func (a *Agent) isSimilarToRecent(text string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(text) == 0 {
		return false
	}
	textRunes := []rune(text)
	for _, recent := range a.recentSpeechTexts {
		recentRunes := []rune(recent)
		shorter := len(textRunes)
		if len(recentRunes) < shorter {
			shorter = len(recentRunes)
		}
		if shorter == 0 {
			continue
		}
		match := 0
		for i := 0; i < shorter; i++ {
			if textRunes[i] == recentRunes[i] {
				match++
			}
		}
		ratio := float64(match) / float64(shorter)
		if ratio > 0.6 {
			slog.Info("[MEDIATOR] duplicate detected", "ratio", fmt.Sprintf("%.2f", ratio), "new", truncate(text, 40), "recent", truncate(recent, 40))
			return true
		}
	}
	return false
}

func (a *Agent) trackSpeechText(text string) {
	if text == "" {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.recentSpeechTexts = append(a.recentSpeechTexts, text)
	if len(a.recentSpeechTexts) > maxRecentSpeech {
		a.recentSpeechTexts = a.recentSpeechTexts[1:]
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
