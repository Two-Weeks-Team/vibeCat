package celebration

import (
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"vibecat/adk-orchestrator/internal/models"
)

const (
	cooldown      = 10 * time.Minute
	celebGenModel = "gemini-3.1-flash-lite-preview"
)

type Agent struct {
	genaiClient     *genai.Client
	lastCelebration time.Time
	recentMessages  []string
}

func New(genaiClient *genai.Client) *Agent {
	return &Agent{genaiClient: genaiClient}
}

func (a *Agent) Run(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
	return func(yield func(*session.Event, error) bool) {
		vision := readVisionFromState(ctx)
		if vision == nil {
			vision = readVisionFromUserContent(ctx)
		}

		result := models.AnalysisResult{Vision: vision}

		// Recover cooldown from session state
		if sess := ctx.Session(); sess != nil && sess.State() != nil {
			if ts, err := sess.State().Get("celebration_last_time"); err == nil {
				if tsStr, ok := ts.(string); ok {
					if parsed, parseErr := time.Parse(time.RFC3339, tsStr); parseErr == nil {
						if parsed.After(a.lastCelebration) {
							a.lastCelebration = parsed
						}
					}
				}
			}
			if recent, err := sess.State().Get("celebration_recent_messages"); err == nil {
				if recentStr, ok := recent.(string); ok {
					var msgs []string
					if jsonErr := json.Unmarshal([]byte(recentStr), &msgs); jsonErr == nil {
						a.recentMessages = msgs
					}
				}
			}
		}

		slog.Info("[CELEBRATION] check",
			"has_vision", vision != nil,
			"success_detected", vision != nil && vision.SuccessDetected,
			"significance", func() int {
				if vision != nil {
					return vision.Significance
				}
				return -1
			}(),
			"since_last", time.Since(a.lastCelebration).String(),
			"cooldown", cooldown.String(),
		)

		if result.Vision != nil && result.Vision.SuccessDetected && result.Vision.Significance >= 9 {
			if time.Since(a.lastCelebration) > cooldown {
				a.lastCelebration = time.Now()
				language := readLanguageFromState(ctx)
				msg := a.generateCelebrationMessage(ctx, vision, language)
				result.Celebration = &models.CelebrationEvent{
					TriggerType: "success_detected",
					Emotion:     "happy",
					Message:     msg,
				}
				a.recentMessages = append(a.recentMessages, msg)
				if len(a.recentMessages) > 3 {
					a.recentMessages = a.recentMessages[len(a.recentMessages)-3:]
				}
				slog.Info("[CELEBRATION] TRIGGERED", "message", msg)
			} else {
				slog.Info("[CELEBRATION] skipped (cooldown)")
			}
		}

		data, err := json.Marshal(result)
		if err != nil {
			yield(nil, fmt.Errorf("celebration marshal: %w", err))
			return
		}

		stateDelta := map[string]any{}
		if result.Celebration != nil {
			celebrationJSON, marshalErr := json.Marshal(result.Celebration)
			if marshalErr != nil {
				yield(nil, fmt.Errorf("celebration state marshal: %w", marshalErr))
				return
			}
			stateDelta["celebration_event"] = string(celebrationJSON)
		} else {
			stateDelta["celebration_event"] = ""
		}
		stateDelta["celebration_last_time"] = a.lastCelebration.Format(time.RFC3339)
		recentJSON, _ := json.Marshal(a.recentMessages)
		stateDelta["celebration_recent_messages"] = string(recentJSON)

		yield(&session.Event{
			Actions: session.EventActions{StateDelta: stateDelta},
			LLMResponse: model.LLMResponse{
				Content: &genai.Content{
					Parts: []*genai.Part{{Text: string(data)}},
				},
			},
		}, nil)
	}
}

func readVisionFromState(ctx agent.InvocationContext) *models.VisionAnalysis {
	sess := ctx.Session()
	if sess == nil || sess.State() == nil {
		return nil
	}

	val, err := sess.State().Get("vision_analysis")
	if err != nil {
		return nil
	}

	var vision models.VisionAnalysis
	if !decodeStateJSON(val, &vision) {
		return nil
	}

	return &vision
}

func readVisionFromUserContent(ctx agent.InvocationContext) *models.VisionAnalysis {
	userContent := ctx.UserContent()
	if userContent == nil {
		return nil
	}

	var result models.AnalysisResult
	for _, part := range userContent.Parts {
		if part.Text != "" {
			if err := json.Unmarshal([]byte(part.Text), &result); err == nil && result.Vision != nil {
				return result.Vision
			}
		}
	}

	return nil
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

func (a *Agent) generateCelebrationMessage(ctx agent.InvocationContext, vision *models.VisionAnalysis, language string) string {
	if a.genaiClient != nil {
		msg := a.generateDynamic(ctx, vision, language)
		if msg != "" {
			return msg
		}
		slog.Warn("[CELEBRATION] dynamic generation failed, falling back to static pool")
	}
	return fallbackCelebrationMessage(language)
}

func (a *Agent) generateDynamic(ctx agent.InvocationContext, vision *models.VisionAnalysis, language string) string {
	lang := normalizeLanguage(language)
	screenContext := ""
	if vision != nil && vision.Content != "" {
		screenContext = fmt.Sprintf("\nScreen context: %s", truncate(vision.Content, 150))
	}

	prompt := fmt.Sprintf(`You are VibeCat, a coding companion celebrating a developer's success.
The developer just achieved something — a test passed, build succeeded, or deployment completed.
%s
Generate ONE short, enthusiastic celebration message (under 60 characters).

Rules:
- Be genuinely excited and specific to what they accomplished
- Reference what happened on screen when possible
- Keep it natural and brief — this goes in a speech bubble
- Vary your style: sometimes use exclamations, sometimes be understated-cool
- Respond in %s
- Do NOT use any of these recent phrases: %s
- Return ONLY the message text`, screenContext, lang, strings.Join(a.recentMessages, "; "))

	resp, err := a.genaiClient.Models.GenerateContent(ctx, celebGenModel, []*genai.Content{
		{Parts: []*genai.Part{{Text: prompt}}, Role: genai.RoleUser},
	}, &genai.GenerateContentConfig{
		MaxOutputTokens: 80,
	})
	if err != nil {
		slog.Warn("[CELEBRATION] LLM call failed", "error", err)
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
	if text == "" {
		return ""
	}
	return text
}

func fallbackCelebrationMessage(language string) string {
	msgs := models.CelebrationMessages[language]
	if len(msgs) == 0 {
		msgs = models.CelebrationMessages["Korean"]
	}
	if len(msgs) == 0 {
		return "Great job!"
	}
	return msgs[rand.Intn(len(msgs))]
}

func normalizeLanguage(language string) string {
	trimmed := strings.TrimSpace(language)
	if trimmed == "" {
		return "Korean"
	}
	lower := strings.ToLower(trimmed)
	switch lower {
	case "ko", "kr", "korean", "korean language":
		return "Korean"
	case "en", "eng", "english", "english language":
		return "English"
	default:
		return trimmed
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
