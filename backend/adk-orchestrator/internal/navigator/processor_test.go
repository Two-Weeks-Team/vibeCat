package navigator

import (
	"testing"

	"vibecat/adk-orchestrator/internal/models"
)

func TestHeuristicEscalationUsesLastInputDescriptor(t *testing.T) {
	result := heuristicEscalation(models.NavigatorEscalationRequest{
		Command:                    `거기에 "gemini live api" 입력해줘`,
		AppName:                    "Google Chrome",
		WindowTitle:                "Google",
		LastInputFieldDescriptor:   "bundle=com.google.Chrome|window=Google|role=textfield|label=Search",
		VisibleInputCandidateCount: 2,
	})

	if result == nil || result.ResolvedDescriptor == nil {
		t.Fatal("expected resolved descriptor")
	}
	if result.ResolvedDescriptor.Label != "Search" {
		t.Fatalf("label = %q, want Search", result.ResolvedDescriptor.Label)
	}
	if result.Confidence < 0.72 {
		t.Fatalf("confidence = %v, want >= 0.72", result.Confidence)
	}
}

func TestBackgroundFallbackInfersReplayLabelAndSurface(t *testing.T) {
	result := backgroundFallback(models.NavigatorBackgroundRequest{
		Command:        "공식 문서 쪽으로 가보자",
		Outcome:        "completed",
		InitialAppName: "Antigravity IDE",
		Surface:        "",
	})

	if result.Surface != "Antigravity" {
		t.Fatalf("surface = %q, want Antigravity", result.Surface)
	}
	if result.ReplayLabel != "chrome_docs_lookup" {
		t.Fatalf("replay label = %q, want chrome_docs_lookup", result.ReplayLabel)
	}
	if len(result.Tags) == 0 {
		t.Fatal("expected tags")
	}
}

func TestValidEscalationResultAllowsResolvedTextWithoutDescriptor(t *testing.T) {
	result := models.NavigatorEscalationResult{
		ResolvedText:           "AppDelegate.swift\nAudioDeviceMonitor.swift",
		Confidence:             0.88,
		FallbackRecommendation: "safe_immediate",
		Reason:                 "visible_text_extracted",
	}

	if !validEscalationResult(result) {
		t.Fatal("expected resolved text without descriptor to be valid")
	}
}
