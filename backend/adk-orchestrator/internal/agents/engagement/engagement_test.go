package engagement

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

func TestRunEngagementAfterSilence(t *testing.T) {
	a := New(nil)
	a.lastActivity = time.Now().Add(-silenceThreshold - time.Second)

	data, err := json.Marshal(models.AnalysisResult{})
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}

	got := decodeSingleResult(t, a.Run(newFakeInvocationContext(string(data))))

	if got.Decision == nil || !got.Decision.ShouldSpeak {
		t.Fatal("expected engagement to trigger speech")
	}
	if got.Decision.Reason != "silence_engagement" {
		t.Fatalf("Reason = %q, want silence_engagement", got.Decision.Reason)
	}
	if got.SpeechText == "" {
		t.Fatal("expected proactive speech text")
	}
}

func TestRunPreservesExistingSpeakDecision(t *testing.T) {
	a := New(nil)
	a.lastActivity = time.Now().Add(-silenceThreshold - time.Second)

	input := models.AnalysisResult{Decision: &models.MediatorDecision{ShouldSpeak: true, Reason: "existing"}}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}

	got := decodeSingleResult(t, a.Run(newFakeInvocationContext(string(data))))

	if got.Decision == nil || !got.Decision.ShouldSpeak {
		t.Fatal("expected ShouldSpeak to remain true")
	}
	if got.Decision.Reason != "existing" {
		t.Fatalf("Reason = %q, want existing", got.Decision.Reason)
	}
}
