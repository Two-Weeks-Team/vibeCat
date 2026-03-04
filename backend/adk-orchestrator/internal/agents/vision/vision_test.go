package vision

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	a := New(nil)
	if a == nil {
		t.Fatal("expected non-nil agent")
	}
}

func TestDetectSuccess(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{name: "detects tests passed", text: "All tests passed in CI", want: true},
		{name: "detects deployed", text: "service deployed to prod", want: true},
		{name: "detects pr merged", text: "PR merged successfully", want: true},
		{name: "no success pattern", text: "still debugging issue", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectSuccess(tt.text); got != tt.want {
				t.Fatalf("detectSuccess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMin(t *testing.T) {
	if got := min(1, 3); got != 1 {
		t.Fatalf("min(1,3) = %d", got)
	}
	if got := min(5, 2); got != 2 {
		t.Fatalf("min(5,2) = %d", got)
	}
}

func TestTrackErrorAndRepeatedError(t *testing.T) {
	a := New(nil)
	msg := "TypeError: cannot read property length of undefined"

	a.trackError(msg)
	if a.isRepeatedError(msg) {
		t.Fatal("expected not repeated with one history entry")
	}

	a.trackError(msg)
	if !a.isRepeatedError(msg) {
		t.Fatal("expected repeated after two matching history entries")
	}
}

func TestTrackErrorHistoryCap(t *testing.T) {
	a := New(nil)
	for i := 0; i < 30; i++ {
		a.trackError(fmt.Sprintf("err-%d", i))
	}
	if len(a.errorHistory) != 21 {
		t.Fatalf("len(errorHistory) = %d, want 21", len(a.errorHistory))
	}
}

func TestAnalyzeFallbackWithoutClientOrImage(t *testing.T) {
	a := New(nil)
	got := a.analyze(context.Background(), "", "terminal shows build output")

	if got.Significance != 3 {
		t.Fatalf("Significance = %d, want 3", got.Significance)
	}
	if got.Content != "terminal shows build output" {
		t.Fatalf("Content = %q", got.Content)
	}
	if got.Emotion != "neutral" {
		t.Fatalf("Emotion = %q", got.Emotion)
	}
	if got.ShouldSpeak {
		t.Fatal("ShouldSpeak should be false in fallback")
	}
	if strings.Contains(got.Content, "unavailable") {
		t.Fatal("empty image fallback should preserve provided context text")
	}
}
