package mediator

import (
	"sync"
	"testing"
	"time"

	"vibecat/adk-orchestrator/internal/models"
)

func TestNew(t *testing.T) {
	a := New(nil)
	if defaultCooldown != 10*time.Second {
		t.Fatalf("defaultCooldown = %v, want %v", defaultCooldown, 10*time.Second)
	}
	if a.cooldown != defaultCooldown {
		t.Fatalf("cooldown = %v, want %v", a.cooldown, defaultCooldown)
	}
}

func TestDecide(t *testing.T) {
	tests := []struct {
		name        string
		agent       *Agent
		vision      *models.VisionAnalysis
		mood        *models.MoodState
		celebration *models.CelebrationEvent
		wantSpeak   bool
		wantReason  string
		wantUrgency string
	}{
		{
			name:        "celebration bypasses cooldown",
			agent:       &Agent{lastSpoke: time.Now(), cooldown: defaultCooldown},
			celebration: &models.CelebrationEvent{Message: "yay"},
			wantSpeak:   true,
			wantReason:  "celebration",
			wantUrgency: "high",
		},
		{
			name:       "cooldown blocks speaking",
			agent:      &Agent{lastSpoke: time.Now(), cooldown: defaultCooldown},
			vision:     &models.VisionAnalysis{Significance: 10, Content: "important"},
			wantSpeak:  false,
			wantReason: "cooldown",
		},
		{
			name:       "no vision returns no_vision",
			agent:      &Agent{lastSpoke: time.Now().Add(-1 * time.Hour), cooldown: defaultCooldown},
			wantSpeak:  false,
			wantReason: "no_vision",
		},
		{
			name:       "focused mood raises threshold",
			agent:      &Agent{lastSpoke: time.Now().Add(-1 * time.Hour), cooldown: defaultCooldown},
			vision:     &models.VisionAnalysis{Significance: 8, Content: "some update"},
			mood:       &models.MoodState{Mood: models.MoodFocused},
			wantSpeak:  false,
			wantReason: "low_significance",
		},
		{
			name:       "frustrated mood lowers threshold",
			agent:      &Agent{lastSpoke: time.Now().Add(-1 * time.Hour), cooldown: defaultCooldown},
			vision:     &models.VisionAnalysis{Significance: 3, Content: "small but important"},
			mood:       &models.MoodState{Mood: models.MoodFrustrated},
			wantSpeak:  true,
			wantReason: "significant_event",
		},
		{
			name:        "error sets high urgency",
			agent:       &Agent{lastSpoke: time.Now().Add(-1 * time.Hour), cooldown: defaultCooldown},
			vision:      &models.VisionAnalysis{Significance: 10, Content: "error screen", ErrorDetected: true},
			wantSpeak:   true,
			wantReason:  "significant_event",
			wantUrgency: "high",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.agent.decide(tt.vision, tt.mood, tt.celebration)
			if got.ShouldSpeak != tt.wantSpeak {
				t.Fatalf("ShouldSpeak = %v, want %v", got.ShouldSpeak, tt.wantSpeak)
			}
			if got.Reason != tt.wantReason {
				t.Fatalf("Reason = %q, want %q", got.Reason, tt.wantReason)
			}
			if tt.wantUrgency != "" && got.Urgency != tt.wantUrgency {
				t.Fatalf("Urgency = %q, want %q", got.Urgency, tt.wantUrgency)
			}
		})
	}
}

func TestSupportiveMessage(t *testing.T) {
	tests := []struct {
		name string
		mood string
		want bool
	}{
		{name: "frustrated has supportive message", mood: models.MoodFrustrated, want: true},
		{name: "stuck has supportive message", mood: models.MoodStuck, want: true},
		{name: "focused has no supportive message", mood: models.MoodFocused, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := fallbackSupportiveMessage(tt.mood, "Korean")
			if (msg != "") != tt.want {
				t.Fatalf("fallbackSupportiveMessage(%q) = %q, want non-empty=%v", tt.mood, msg, tt.want)
			}
		})
	}
}

func TestConcurrentSpeechTracking(t *testing.T) {
	a := New(nil)
	a.lastSpoke = time.Now().Add(-1 * time.Hour)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.trackSpeechText("build failed at pkg/service")
			_ = a.isSimilarToRecent("build failed at pkg/service")
			_ = a.decide(&models.VisionAnalysis{Significance: 9, Content: "build failed at pkg/service"}, nil, nil)
		}()
	}
	wg.Wait()

	if len(a.recentSpeechTexts) > maxRecentSpeech {
		t.Fatalf("len(recentSpeechTexts) = %d, want <= %d", len(a.recentSpeechTexts), maxRecentSpeech)
	}
}
