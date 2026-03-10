package engagement

import (
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"
	"strings"
	"sync"
	"time"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"vibecat/adk-orchestrator/internal/lang"
	"vibecat/adk-orchestrator/internal/models"
)

const (
	silenceThreshold     = 180 * time.Second
	restReminderInterval = 50 * time.Minute
	restReminderCooldown = 30 * time.Minute
	engageGenModel       = "gemini-3.1-flash-lite-preview"
)

type Agent struct {
	mu               sync.Mutex
	genaiClient      *genai.Client
	lastActivity     time.Time
	lastRestReminder time.Time
}

func New(genaiClient *genai.Client) *Agent {
	return &Agent{genaiClient: genaiClient, lastActivity: time.Now()}
}

func (a *Agent) Run(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
	return func(yield func(*session.Event, error) bool) {
		a.mu.Lock()
		defer a.mu.Unlock()

		userContent := ctx.UserContent()
		if userContent == nil {
			yield(nil, fmt.Errorf("engagement: no user content"))
			return
		}

		var result models.AnalysisResult
		for _, part := range userContent.Parts {
			if part.Text != "" {
				_ = json.Unmarshal([]byte(part.Text), &result)
			}
		}

		if result.Decision != nil && result.Decision.ShouldSpeak {
			a.lastActivity = time.Now()
		}

		sinceLast := time.Since(a.lastActivity)
		slog.Info("[ENGAGEMENT] check",
			"since_last_activity", sinceLast.String(),
			"threshold", silenceThreshold.String(),
			"already_speaking", result.Decision != nil && result.Decision.ShouldSpeak,
		)

		if sinceLast > silenceThreshold {
			if result.Decision == nil {
				result.Decision = &models.MediatorDecision{}
			}
			if !result.Decision.ShouldSpeak {
				result.Decision.ShouldSpeak = true
				result.Decision.Reason = "silence_engagement"
				result.SpeechText = a.generateSilenceMessage(ctx, readLanguageFromState(ctx))
				a.lastActivity = time.Now()
				slog.Info("[ENGAGEMENT] silence engagement TRIGGERED", "silence_duration", sinceLast.String())
			}
		}

		activityMin := readActivityMinutes(ctx)
		sinceLastReminder := time.Since(a.lastRestReminder)
		if activityMin >= int(restReminderInterval.Minutes()) && sinceLastReminder > restReminderCooldown {
			if result.Decision == nil {
				result.Decision = &models.MediatorDecision{}
			}
			if !result.Decision.ShouldSpeak {
				lang := readLanguageFromState(ctx)
				result.Decision.ShouldSpeak = true
				result.Decision.Reason = "rest_reminder"
				result.SpeechText = a.generateRestMessage(ctx, lang, activityMin)
				a.lastRestReminder = time.Now()
				slog.Info("[ENGAGEMENT] rest reminder TRIGGERED", "activity_minutes", activityMin)
			}
		}

		data, err := json.Marshal(result)
		if err != nil {
			yield(nil, fmt.Errorf("engagement marshal: %w", err))
			return
		}

		yield(&session.Event{
			LLMResponse: model.LLMResponse{
				Content: &genai.Content{
					Parts: []*genai.Part{{Text: string(data)}},
				},
			},
		}, nil)
	}
}

func readActivityMinutes(ctx agent.InvocationContext) int {
	sess := ctx.Session()
	if sess == nil || sess.State() == nil {
		return 0
	}
	val, err := sess.State().Get("activity_minutes")
	if err != nil {
		return 0
	}
	switch v := val.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	}
	return 0
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

func (a *Agent) generateSilenceMessage(ctx agent.InvocationContext, language string) string {
	if a.genaiClient != nil {
		msg := a.generateDynamic(ctx, language)
		if msg != "" {
			return msg
		}
		slog.Warn("[ENGAGEMENT] dynamic generation failed, using fallback")
	}
	if language == "English" {
		return "Things have been quiet. Need a hand with anything?"
	}
	return "좀 조용하네요. 혹시 도움이 필요한 부분이 있나요?"
}

func (a *Agent) generateRestMessage(ctx agent.InvocationContext, language string, minutes int) string {
	if a.genaiClient != nil {
		lang := lang.NormalizeLanguage(language)
		prompt := fmt.Sprintf(`You are VibeCat, a caring coding companion.
The developer has been working for %d minutes straight without a break.
Generate ONE short rest reminder (under 80 characters).

Rules:
- Be warm and caring, not nagging
- Mention a specific suggestion: stretch, water, walk, look away from screen
- Respond in %s
- Return ONLY the message text`, minutes, lang)

		resp, err := a.genaiClient.Models.GenerateContent(ctx, engageGenModel, []*genai.Content{
			{Parts: []*genai.Part{{Text: prompt}}, Role: genai.RoleUser},
		}, &genai.GenerateContentConfig{
			MaxOutputTokens: 80,
		})
		if err == nil && len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
			text := ""
			for _, part := range resp.Candidates[0].Content.Parts {
				text += part.Text
			}
			text = strings.TrimSpace(text)
			if text != "" {
				return text
			}
		}
	}
	if language == "English" {
		return fmt.Sprintf("You've been coding for %d minutes. Time to stretch and grab some water!", minutes)
	}
	return fmt.Sprintf("%d분째 코딩 중이에요. 잠깐 스트레칭하고 물 한 잔 어때요?", minutes)
}

func (a *Agent) generateDynamic(ctx agent.InvocationContext, language string) string {
	lang := lang.NormalizeLanguage(language)

	prompt := fmt.Sprintf(`You are VibeCat, a coding companion for solo developers.
The developer has been quiet for a while. Generate ONE short check-in message (under 80 characters).

Rules:
- Be casual and warm, not pushy
- Suggest something concrete: take a break, try a different approach, review recent changes, etc.
- Vary your tone: sometimes concerned, sometimes playful, sometimes practical
- Do NOT say "한동안 조용하네요" or start with observations about silence
- Respond in %s
- Return ONLY the message text`, lang)

	resp, err := a.genaiClient.Models.GenerateContent(ctx, engageGenModel, []*genai.Content{
		{Parts: []*genai.Part{{Text: prompt}}, Role: genai.RoleUser},
	}, &genai.GenerateContentConfig{
		MaxOutputTokens: 80,
	})
	if err != nil {
		slog.Warn("[ENGAGEMENT] LLM call failed", "error", err)
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
