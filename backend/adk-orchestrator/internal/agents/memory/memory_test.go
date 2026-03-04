package memory

import (
	"context"
	"strings"
	"testing"
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
			got := a.generateSummary(context.Background(), tt.history)
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
	if got := a.retrieveMemory(context.Background(), "user-1"); got != "" {
		t.Fatalf("retrieveMemory() = %q, want empty", got)
	}
}

func TestSaveSessionSummaryNilStore(t *testing.T) {
	a := New(nil, nil)
	if err := a.SaveSessionSummary(context.Background(), "user-1", []string{"some event"}); err != nil {
		t.Fatalf("SaveSessionSummary() error = %v", err)
	}
}
