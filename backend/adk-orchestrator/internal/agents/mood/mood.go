package mood

import (
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"
	"time"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"vibecat/adk-orchestrator/internal/models"
)

type Agent struct {
	errorCount      int
	lastErrorTime   time.Time
	lastInteraction time.Time
	silenceStart    time.Time
}

func New() *Agent {
	return &Agent{lastInteraction: time.Now(), silenceStart: time.Now()}
}

func (a *Agent) Run(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
	return func(yield func(*session.Event, error) bool) {
		vision := readVisionFromState(ctx)
		if vision == nil {
			vision = readVisionFromUserContent(ctx)
		}

		slog.Info("[MOOD] input",
			"has_vision", vision != nil,
			"error_detected", vision != nil && vision.ErrorDetected,
			"repeated_error", vision != nil && vision.RepeatedError,
			"error_count_before", a.errorCount,
		)

		voiceTone, voiceConfidence := readVoiceToneFromState(ctx)
		mood := a.classify(vision, voiceTone, voiceConfidence)
		result := models.AnalysisResult{Vision: vision}
		result.Mood = mood

		slog.Info("[MOOD] result",
			"mood", mood.Mood,
			"confidence", mood.Confidence,
			"signals", fmt.Sprintf("%v", mood.Signals),
			"action", mood.SuggestedAction,
			"error_count_after", a.errorCount,
		)

		data, err := json.Marshal(result)
		if err != nil {
			yield(nil, fmt.Errorf("mood marshal: %w", err))
			return
		}

		moodJSON, err := json.Marshal(mood)
		if err != nil {
			yield(nil, fmt.Errorf("mood state marshal: %w", err))
			return
		}

		yield(&session.Event{
			Actions: session.EventActions{
				StateDelta: map[string]any{"mood_state": string(moodJSON)},
			},
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

func readVoiceToneFromState(ctx agent.InvocationContext) (string, float64) {
	sess := ctx.Session()
	if sess == nil || sess.State() == nil {
		return "", 0
	}

	var tone string
	if val, err := sess.State().Get("voice_tone"); err == nil {
		if s, ok := val.(string); ok {
			tone = s
		}
	}

	var conf float64
	if val, err := sess.State().Get("voice_confidence"); err == nil {
		switch v := val.(type) {
		case float64:
			conf = v
		case float32:
			conf = float64(v)
		case int:
			conf = float64(v)
		}
	}

	return tone, conf
}

func (a *Agent) classify(vision *models.VisionAnalysis, voiceTone string, voiceConfidence float64) *models.MoodState {
	now := time.Now()
	silence := now.Sub(a.silenceStart)

	if vision != nil && vision.ErrorDetected {
		a.errorCount++
		a.lastErrorTime = now
	} else if a.errorCount > 0 && now.Sub(a.lastErrorTime) > 2*time.Minute {
		a.errorCount = max(0, a.errorCount-1)
		slog.Info("[MOOD] errorCount decayed", "new_count", a.errorCount, "since_last_error", now.Sub(a.lastErrorTime).String())
	}

	var signals []string
	confidence := 0.0

	if vision != nil && vision.ErrorDetected {
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

	// Voice-based signals (multimodal emotion fusion)
	if voiceTone == "stressed" || voiceTone == "frustrated" {
		signals = append(signals, "voice_stressed")
		confidence += 0.25
	} else if voiceTone == "positive" || voiceTone == "excited" {
		signals = append(signals, "voice_positive")
		confidence -= 0.15
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
		VoiceTone:       voiceTone,
		VoiceConfidence: voiceConfidence,
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
