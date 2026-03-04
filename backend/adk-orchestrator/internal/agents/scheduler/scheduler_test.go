package scheduler

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	a := New()
	if a.cooldown != 10*time.Second {
		t.Fatalf("cooldown = %v, want %v", a.cooldown, 10*time.Second)
	}
	if a.silence != 30*time.Second {
		t.Fatalf("silence = %v, want %v", a.silence, 30*time.Second)
	}
}

func TestAdjust(t *testing.T) {
	tests := []struct {
		name         string
		utterances   int
		wantCooldown time.Duration
		wantSilence  time.Duration
	}{
		{
			name:         "high interaction rate increases thresholds",
			utterances:   3,
			wantCooldown: 20 * time.Second,
			wantSilence:  45 * time.Second,
		},
		{
			name:         "low interaction rate decreases thresholds",
			utterances:   0,
			wantCooldown: 5 * time.Second,
			wantSilence:  10 * time.Second,
		},
		{
			name:         "mid interaction rate keeps defaults",
			utterances:   1,
			wantCooldown: 10 * time.Second,
			wantSilence:  30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Agent{
				utterances: tt.utterances,
				lastUpdate: time.Now().Add(-1 * time.Minute),
				cooldown:   10 * time.Second,
				silence:    30 * time.Second,
			}
			a.adjust()
			if a.cooldown != tt.wantCooldown {
				t.Fatalf("cooldown = %v, want %v", a.cooldown, tt.wantCooldown)
			}
			if a.silence != tt.wantSilence {
				t.Fatalf("silence = %v, want %v", a.silence, tt.wantSilence)
			}
		})
	}
}
