package mood

import (
	"testing"
	"time"

	"vibecat/adk-orchestrator/internal/models"
)

func TestNew(t *testing.T) {
	a := New()
	if a.lastInteraction.IsZero() || a.silenceStart.IsZero() {
		t.Fatal("expected initialized timestamps")
	}
}

func TestClassify(t *testing.T) {
	tests := []struct {
		name        string
		agent       *Agent
		vision      *models.VisionAnalysis
		wantMood    string
		wantAction  string
		wantSignals []string
	}{
		{
			name: "frustrated when errors accumulate with high confidence",
			agent: &Agent{
				errorCount:   0,
				silenceStart: time.Now().Add(-6 * time.Minute),
			},
			vision:      &models.VisionAnalysis{ErrorDetected: true, RepeatedError: true},
			wantMood:    models.MoodFrustrated,
			wantAction:  "offer_help",
			wantSignals: []string{"error_detected", "repeated_error", "long_silence"},
		},
		{
			name: "stuck when many prior errors and moderate confidence",
			agent: &Agent{
				errorCount:   3,
				silenceStart: time.Now().Add(-6 * time.Minute),
			},
			vision:      &models.VisionAnalysis{RepeatedError: true},
			wantMood:    models.MoodStuck,
			wantAction:  "search",
			wantSignals: []string{"repeated_error", "long_silence"},
		},
		{
			name: "idle after very long silence",
			agent: &Agent{
				errorCount:   0,
				silenceStart: time.Now().Add(-11 * time.Minute),
			},
			vision:      nil,
			wantMood:    models.MoodIdle,
			wantAction:  "engage",
			wantSignals: []string{"long_silence"},
		},
		{
			name: "focused default when no strong signal",
			agent: &Agent{
				errorCount:   0,
				silenceStart: time.Now(),
			},
			vision:      nil,
			wantMood:    models.MoodFocused,
			wantAction:  "continue",
			wantSignals: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.agent.classify(tt.vision)
			if got.Mood != tt.wantMood {
				t.Fatalf("Mood = %q, want %q", got.Mood, tt.wantMood)
			}
			if got.SuggestedAction != tt.wantAction {
				t.Fatalf("SuggestedAction = %q, want %q", got.SuggestedAction, tt.wantAction)
			}
			if len(got.Signals) != len(tt.wantSignals) {
				t.Fatalf("Signals len = %d, want %d", len(got.Signals), len(tt.wantSignals))
			}
			for i := range tt.wantSignals {
				if got.Signals[i] != tt.wantSignals[i] {
					t.Fatalf("Signals[%d] = %q, want %q", i, got.Signals[i], tt.wantSignals[i])
				}
			}
		})
	}
}

func TestClassifyIncrementsErrorCount(t *testing.T) {
	a := &Agent{silenceStart: time.Now()}
	_ = a.classify(&models.VisionAnalysis{ErrorDetected: true})
	if a.errorCount != 1 {
		t.Fatalf("errorCount = %d, want 1", a.errorCount)
	}
}
