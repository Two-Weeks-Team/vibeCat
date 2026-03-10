package memory

import (
	"context"
	"strings"
	"testing"
	"time"

	"vibecat/adk-orchestrator/internal/store"
)

func TestNew(t *testing.T) {
	a := New(nil, nil)
	if a == nil {
		t.Fatal("expected non-nil agent")
	}
}

func TestGenerateSummaryFallback(t *testing.T) {
	a := New(nil, nil)
	tests := []struct {
		name    string
		history []string
		want    string
	}{
		{name: "no client returns default", history: []string{"did work"}, want: "Session completed."},
		{name: "empty history returns default", history: nil, want: "Session completed."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := a.generateSummary(context.Background(), tt.history, "English")
			if got != tt.want {
				t.Fatalf("generateSummary() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractUnresolvedIssues(t *testing.T) {
	history := []string{
		"cannot open file because permission denied " + strings.Repeat("x", 150),
		"Error: failed to compile package",
		"Exception: database timeout while querying",
		"error: undefined variable userID",
		"error: this one is fixed and resolved",
	}

	got := extractUnresolvedIssues(history)

	if len(got) != 3 {
		t.Fatalf("len(issues) = %d, want 3", len(got))
	}
	if strings.Contains(strings.ToLower(strings.Join(got, " ")), "resolved") {
		t.Fatalf("unexpected resolved issue included: %v", got)
	}
	if !strings.HasSuffix(got[0], "...") {
		t.Fatalf("expected truncated issue to end with ellipsis, got %q", got[0])
	}
}

func TestRetrieveMemoryNilStore(t *testing.T) {
	a := New(nil, nil)
	if got := a.retrieveMemory(context.Background(), "user-1", "English"); got != "" {
		t.Fatalf("retrieveMemory() = %q, want empty", got)
	}
}

func TestSaveSessionSummaryNilStore(t *testing.T) {
	a := New(nil, nil)
	if err := a.SaveSessionSummary(context.Background(), "user-1", []string{"some event"}, "English"); err != nil {
		t.Fatalf("SaveSessionSummary() error = %v", err)
	}
}

func TestMergeTopics(t *testing.T) {
	got := mergeTopics(nil, []string{
		"Swift build failed with auth error",
		"Need to debug websocket reconnect flow",
	})

	if len(got) == 0 {
		t.Fatal("mergeTopics() returned no topics")
	}
}

func TestFormatMemoryContext(t *testing.T) {
	entry := &store.MemoryEntry{
		RecentSummaries: []store.SessionSummary{
			{
				Date:    time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC),
				Summary: "Worked on websocket reconnect and search routing.",
			},
			{
				Date:             time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
				Summary:          "Stabilized barge-in and Live turn handling.",
				UnresolvedIssues: []string{"voice search follow-up still flaky"},
			},
		},
		KnownTopics: []store.Topic{
			{Name: "Gemini Live API", LastMentioned: time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC)},
			{Name: "Cloud Run", LastMentioned: time.Date(2026, 3, 9, 9, 0, 0, 0, time.UTC)},
			{Name: "resolved topic", LastMentioned: time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC), Resolved: true},
		},
	}

	got := formatMemoryContext(entry, "Korean")

	for _, want := range []string{
		"Recent developer context:",
		"Mar 8",
		"Mar 10",
		"voice search follow-up still flaky",
		"Active topics: Gemini Live API, Cloud Run",
		"Respond in Korean.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("formatMemoryContext() missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "resolved topic") {
		t.Fatalf("formatMemoryContext() should omit resolved topics:\n%s", got)
	}
}
