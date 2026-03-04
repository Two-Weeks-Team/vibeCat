package mood

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
	errorCount      int
	lastInteraction time.Time
	silenceStart    time.Time
}

func New() *Agent {
	return &Agent{lastInteraction: time.Now(), silenceStart: time.Now()}
}

func (a *Agent) Run(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
	return func(yield func(*session.Event, error) bool) {
		userContent := ctx.UserContent()
		if userContent == nil {
			yield(nil, fmt.Errorf("mood agent: no user content"))
			return
		}

		var result models.AnalysisResult
		for _, part := range userContent.Parts {
			if part.Text != "" {
				_ = json.Unmarshal([]byte(part.Text), &result)
			}
		}

		mood := a.classify(result.Vision)
		result.Mood = mood

		data, err := json.Marshal(result)
		if err != nil {
			yield(nil, fmt.Errorf("mood marshal: %w", err))
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

func (a *Agent) classify(vision *models.VisionAnalysis) *models.MoodState {
	now := time.Now()
	silence := now.Sub(a.silenceStart)

	var signals []string
	confidence := 0.0

	if vision != nil && vision.ErrorDetected {
		a.errorCount++
		signals = append(signals, "error_detected")
		confidence += 0.3
	}
	if vision != nil && vision.RepeatedError {
		signals = append(signals, "repeated_error")
		confidence += 0.3
	}
	if silence > 5*time.Minute {
		signals = append(signals, "long_silence")
		confidence += 0.25
	}

	mood := models.MoodFocused
	action := "continue"

	if confidence >= 0.6 && vision != nil && vision.ErrorDetected {
		mood = models.MoodFrustrated
		action = "offer_help"
	} else if confidence >= 0.5 && a.errorCount >= 3 {
		mood = models.MoodStuck
		action = "search"
	} else if silence > 10*time.Minute {
		mood = models.MoodIdle
		action = "engage"
	}

	if confidence < 0.7 {
		confidence = 0.5
	}

	return &models.MoodState{
		Mood:            mood,
		Confidence:      confidence,
		Signals:         signals,
		SuggestedAction: action,
		UpdatedAt:       now,
	}
}
