package search

import (
	"strings"
	"testing"

	"vibecat/adk-orchestrator/internal/models"
)

func TestShouldSearch(t *testing.T) {
	tests := []struct {
		name   string
		result models.AnalysisResult
		want   bool
	}{
		{
			name:   "stuck mood triggers search",
			result: models.AnalysisResult{Mood: &models.MoodState{Mood: models.MoodStuck}},
			want:   true,
		},
		{
			name: "frustrated plus error triggers search",
			result: models.AnalysisResult{
				Mood:   &models.MoodState{Mood: models.MoodFrustrated},
				Vision: &models.VisionAnalysis{ErrorDetected: true},
			},
			want: true,
		},
		{
			name: "error keyword triggers search",
			result: models.AnalysisResult{
				Vision: &models.VisionAnalysis{ErrorMessage: "Import error: module not found"},
			},
			want: true,
		},
		{
			name: "context keyword triggers search",
			result: models.AnalysisResult{
				Vision: &models.VisionAnalysis{Content: "how to fix auth in go"},
			},
			want: true,
		},
		{
			name: "no trigger returns false",
			result: models.AnalysisResult{
				Mood:   &models.MoodState{Mood: models.MoodFocused},
				Vision: &models.VisionAnalysis{Content: "editing a readme"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldSearch(tt.result)
			if got != tt.want {
				t.Fatalf("shouldSearch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildQuery(t *testing.T) {
	long := strings.Repeat("a", 300)
	tests := []struct {
		name   string
		result models.AnalysisResult
		want   string
	}{
		{
			name:   "prefers error message",
			result: models.AnalysisResult{Vision: &models.VisionAnalysis{ErrorMessage: "panic: nil pointer", Content: "fallback"}},
			want:   "panic: nil pointer",
		},
		{
			name:   "uses content and truncates to 200",
			result: models.AnalysisResult{Vision: &models.VisionAnalysis{Content: long}},
			want:   long[:200],
		},
		{
			name:   "default query when no vision fields",
			result: models.AnalysisResult{},
			want:   "developer error solution",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildQuery(tt.result)
			if got != tt.want {
				t.Fatalf("buildQuery() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSearchNilClientFallback(t *testing.T) {
	a := New(nil)
	got := a.search(nil, "go test failure", models.AnalysisResult{}, "English")
	if got == nil {
		t.Fatal("expected fallback search result")
	}
	if got.Query != "go test failure" {
		t.Fatalf("Query = %q", got.Query)
	}
	if !strings.Contains(got.Summary, "Search unavailable") {
		t.Fatalf("Summary = %q, want unavailable message", got.Summary)
	}
}
