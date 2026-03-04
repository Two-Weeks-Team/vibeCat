package models

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestAnalysisResultJSONRoundTrip(t *testing.T) {
	now := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	input := AnalysisResult{
		Vision: &VisionAnalysis{
			Significance:    8,
			Content:         "go test failed",
			Emotion:         "concerned",
			ShouldSpeak:     true,
			ErrorDetected:   true,
			RepeatedError:   true,
			SuccessDetected: false,
			ErrorMessage:    "undefined: foo",
			ErrorRegion:     &Region{X: 1.2, Y: 3.4},
		},
		Decision: &MediatorDecision{ShouldSpeak: true, Reason: "significant_event", Urgency: "high"},
		Mood: &MoodState{
			Mood:            MoodFrustrated,
			Confidence:      0.9,
			Signals:         []string{"error_detected"},
			SuggestedAction: "offer_help",
			UpdatedAt:       now,
		},
		Celebration: &CelebrationEvent{TriggerType: "success_detected", Emotion: "happy", Message: "nice"},
		Search:      &SearchResult{Query: "foo", Summary: "bar", Sources: []string{"https://example.com"}},
		SpeechText:  "hello",
	}

	b, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var got AnalysisResult
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if !reflect.DeepEqual(got, input) {
		t.Fatalf("round-trip mismatch\n got: %#v\nwant: %#v", got, input)
	}
}

func TestOmitemptyFields(t *testing.T) {
	b, err := json.Marshal(VisionAnalysis{Significance: 1})
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	s := string(b)
	if strings.Contains(s, "errorMessage") {
		t.Fatalf("unexpected errorMessage field: %s", s)
	}
	if strings.Contains(s, "errorRegion") {
		t.Fatalf("unexpected errorRegion field: %s", s)
	}
}

func TestConstantsAndMessageCatalogs(t *testing.T) {
	if MoodFocused == "" || MoodFrustrated == "" || MoodStuck == "" || MoodIdle == "" {
		t.Fatal("expected mood constants to be non-empty")
	}
	if len(SupportiveMessages[MoodFrustrated]) == 0 {
		t.Fatal("expected supportive messages for frustrated mood")
	}
	if len(SupportiveMessages[MoodStuck]) == 0 {
		t.Fatal("expected supportive messages for stuck mood")
	}
	if len(CelebrationMessages) == 0 {
		t.Fatal("expected celebration messages")
	}
}
