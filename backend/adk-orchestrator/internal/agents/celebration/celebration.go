package celebration

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

const cooldown = 5 * time.Minute

type Agent struct {
	lastCelebration time.Time
}

func New() *Agent {
	return &Agent{}
}

func (a *Agent) Run(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
	return func(yield func(*session.Event, error) bool) {
		userContent := ctx.UserContent()
		if userContent == nil {
			yield(nil, fmt.Errorf("celebration agent: no user content"))
			return
		}

		var result models.AnalysisResult
		for _, part := range userContent.Parts {
			if part.Text != "" {
				_ = json.Unmarshal([]byte(part.Text), &result)
			}
		}

		if result.Vision != nil && result.Vision.SuccessDetected {
			if time.Since(a.lastCelebration) > cooldown {
				a.lastCelebration = time.Now()
				result.Celebration = &models.CelebrationEvent{
					TriggerType: "success_detected",
					Emotion:     "happy",
					Message:     models.CelebrationMessages[0],
				}
			}
		}

		data, err := json.Marshal(result)
		if err != nil {
			yield(nil, fmt.Errorf("celebration marshal: %w", err))
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
