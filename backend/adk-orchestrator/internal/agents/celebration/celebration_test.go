package celebration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"vibecat/adk-orchestrator/internal/models"
)

type fakeInvocationContext struct {
	context.Context
	userContent *genai.Content
	ended       bool
}

func newFakeInvocationContext(text string) *fakeInvocationContext {
	return &fakeInvocationContext{
		Context: context.Background(),
		userContent: &genai.Content{
			Parts: []*genai.Part{{Text: text}},
		},
	}
}

func (f *fakeInvocationContext) Agent() agent.Agent          { return nil }
func (f *fakeInvocationContext) Artifacts() agent.Artifacts  { return nil }
func (f *fakeInvocationContext) Memory() agent.Memory        { return nil }
func (f *fakeInvocationContext) Session() session.Session    { return nil }
func (f *fakeInvocationContext) InvocationID() string        { return "test" }
func (f *fakeInvocationContext) Branch() string              { return "" }
func (f *fakeInvocationContext) UserContent() *genai.Content { return f.userContent }
func (f *fakeInvocationContext) RunConfig() *agent.RunConfig { return nil }
func (f *fakeInvocationContext) EndInvocation()              { f.ended = true }
func (f *fakeInvocationContext) Ended() bool                 { return f.ended }
func (f *fakeInvocationContext) WithContext(ctx context.Context) agent.InvocationContext {
	copy := *f
	copy.Context = ctx
	return &copy
}

func decodeSingleResult(t *testing.T, seq func(func(*session.Event, error) bool)) models.AnalysisResult {
	t.Helper()
	for event, err := range seq {
		if err != nil {
			t.Fatalf("agent run error: %v", err)
		}
		if event == nil || event.LLMResponse.Content == nil || len(event.LLMResponse.Content.Parts) == 0 {
			t.Fatal("missing event content")
		}
		var got models.AnalysisResult
		if err := json.Unmarshal([]byte(event.LLMResponse.Content.Parts[0].Text), &got); err != nil {
			t.Fatalf("unmarshal output: %v", err)
		}
		return got
	}
	t.Fatal("no event yielded")
	return models.AnalysisResult{}
}

func TestNew(t *testing.T) {
	a := New(nil)
	if !a.lastCelebration.IsZero() {
		t.Fatal("expected zero initial lastCelebration")
	}
}

func TestRunCelebrationTrigger(t *testing.T) {
	tests := []struct {
		name            string
		lastCelebration time.Time
		input           models.AnalysisResult
		wantTriggered   bool
	}{
		{
			name:            "triggers when success and outside cooldown",
			lastCelebration: time.Now().Add(-cooldown - time.Second),
			input:           models.AnalysisResult{Vision: &models.VisionAnalysis{SuccessDetected: true, Significance: 9}},
			wantTriggered:   true,
		},
		{
			name:            "suppressed during cooldown",
			lastCelebration: time.Now(),
			input:           models.AnalysisResult{Vision: &models.VisionAnalysis{SuccessDetected: true, Significance: 9}},
			wantTriggered:   false,
		},
		{
			name:            "no success no celebration",
			lastCelebration: time.Now().Add(-cooldown - time.Second),
			input:           models.AnalysisResult{Vision: &models.VisionAnalysis{SuccessDetected: false}},
			wantTriggered:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Agent{lastCelebration: tt.lastCelebration}
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("marshal input: %v", err)
			}

			ctx := newFakeInvocationContext(string(data))
			got := decodeSingleResult(t, a.Run(ctx))

			triggered := got.Celebration != nil
			if triggered != tt.wantTriggered {
				t.Fatalf("triggered = %v, want %v", triggered, tt.wantTriggered)
			}
			if tt.wantTriggered {
				if got.Celebration.TriggerType != "success_detected" {
					t.Fatalf("TriggerType = %q", got.Celebration.TriggerType)
				}
				if got.Celebration.Message == "" {
					t.Fatal("expected celebration message")
				}
			}
		})
	}
}
