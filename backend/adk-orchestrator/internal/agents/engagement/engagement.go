package engagement

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

const silenceThreshold = 30 * time.Second

type Agent struct {
	lastActivity time.Time
}

func New() *Agent {
	return &Agent{lastActivity: time.Now()}
}

func (a *Agent) Run(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
	return func(yield func(*session.Event, error) bool) {
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

		if time.Since(a.lastActivity) > silenceThreshold {
			if result.Decision == nil {
				result.Decision = &models.MediatorDecision{}
			}
			if !result.Decision.ShouldSpeak {
				result.Decision.ShouldSpeak = true
				result.Decision.Reason = "silence_engagement"
				result.SpeechText = "잘 되고 있어요? 뭔가 도움이 필요하면 말해줘요!"
				a.lastActivity = time.Now()
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
