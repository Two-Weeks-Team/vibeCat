package scheduler

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

type Agent struct {
	utterances    int
	interruptions int
	lastUpdate    time.Time
	cooldown      time.Duration
	silence       time.Duration
}

func New() *Agent {
	return &Agent{
		cooldown:   10 * time.Second,
		silence:    30 * time.Second,
		lastUpdate: time.Now(),
	}
}

func (a *Agent) Run(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
	return func(yield func(*session.Event, error) bool) {
		userContent := ctx.UserContent()
		if userContent == nil {
			yield(nil, fmt.Errorf("scheduler: no user content"))
			return
		}

		var result models.AnalysisResult
		for _, part := range userContent.Parts {
			if part.Text != "" {
				_ = json.Unmarshal([]byte(part.Text), &result)
			}
		}

		if result.Decision != nil && result.Decision.ShouldSpeak {
			a.utterances++
		}

		a.adjust()

		data, err := json.Marshal(result)
		if err != nil {
			yield(nil, fmt.Errorf("scheduler marshal: %w", err))
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

func (a *Agent) adjust() {
	rate := float64(a.utterances) / time.Since(a.lastUpdate).Minutes()
	if rate > 2 {
		a.cooldown = 20 * time.Second
		a.silence = 45 * time.Second
	} else if rate < 0.5 {
		a.cooldown = 5 * time.Second
		a.silence = 10 * time.Second
	}
}
