package mediator

import (
	"encoding/json"
	"fmt"
	"iter"
	"time"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"vibecat/adk-orchestrator/internal/models"
)

const (
	defaultCooldown  = 10 * time.Second
	highSignificance = 7
	lowSignificance  = 3
)

type Agent struct {
	lastSpoke   time.Time
	lastContent string
	cooldown    time.Duration
}

func New() *Agent {
	return &Agent{cooldown: defaultCooldown}
}

func (a *Agent) Run(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
	return func(yield func(*session.Event, error) bool) {
		userContent := ctx.UserContent()
		if userContent == nil {
			yield(nil, fmt.Errorf("mediator: no user content"))
			return
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

		decision := a.decide(vision, mood, celebration)

		result := models.AnalysisResult{
			Vision:      vision,
			Decision:    decision,
			Mood:        mood,
			Celebration: celebration,
		}

		if decision.ShouldSpeak && vision != nil {
			result.SpeechText = vision.Content
		}
		if celebration != nil && celebration.Message != "" {
			result.SpeechText = celebration.Message
		}
		if mood != nil && !decision.ShouldSpeak {
			if msg := supportiveMessage(mood.Mood); msg != "" {
				result.SpeechText = msg
				decision.ShouldSpeak = true
				decision.Reason = "mood_support"
			}
		}

		data, err := json.Marshal(result)
		if err != nil {
			yield(nil, fmt.Errorf("mediator marshal: %w", err))
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

func (a *Agent) decide(vision *models.VisionAnalysis, mood *models.MoodState, celebration *models.CelebrationEvent) *models.MediatorDecision {
	if celebration != nil {
		return &models.MediatorDecision{ShouldSpeak: true, Reason: "celebration", Urgency: "high"}
	}

	if time.Since(a.lastSpoke) < a.cooldown {
		return &models.MediatorDecision{ShouldSpeak: false, Reason: "cooldown", Urgency: "low"}
	}

	if vision == nil {
		return &models.MediatorDecision{ShouldSpeak: false, Reason: "no_vision", Urgency: "low"}
	}

	threshold := highSignificance
	if mood != nil && mood.Mood == models.MoodFrustrated {
		threshold = lowSignificance
	}
	if mood != nil && mood.Mood == models.MoodFocused {
		threshold = 9
	}

	if vision.Significance < threshold {
		return &models.MediatorDecision{ShouldSpeak: false, Reason: "low_significance", Urgency: "low"}
	}

	if vision.Content == a.lastContent {
		return &models.MediatorDecision{ShouldSpeak: false, Reason: "duplicate", Urgency: "low"}
	}

	a.lastSpoke = time.Now()
	a.lastContent = vision.Content

	urgency := "normal"
	if vision.ErrorDetected || vision.RepeatedError {
		urgency = "high"
	}

	return &models.MediatorDecision{ShouldSpeak: true, Reason: "significant_event", Urgency: urgency}
}

func supportiveMessage(mood string) string {
	msgs, ok := models.SupportiveMessages[mood]
	if !ok || len(msgs) == 0 {
		return ""
	}
	return msgs[0]
}
