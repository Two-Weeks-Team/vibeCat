package ws

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"vibecat/realtime-gateway/internal/adk"
)

type navigatorReplayFixture struct {
	Name    string           `json:"name"`
	Command string           `json:"command"`
	Context navigatorContext `json:"context"`
	Expect  struct {
		IntentClass string `json:"intentClass"`
		StepCount   int    `json:"stepCount"`
		FirstAction string `json:"firstAction"`
		LastAction  string `json:"lastAction"`
	} `json:"expect"`
}

func TestMaybeEscalateNavigatorPlanUsesResolvedDescriptor(t *testing.T) {
	command := `거기에 "gemini live api" 입력해줘`
	ctx := navigatorContext{
		AppName:                "Google Chrome",
		WindowTitle:            "Google",
		FocusedRole:            "AXGroup",
		FocusedLabel:           "Search controls",
		AXSnapshot:             "window:Google\nAXGroup:Search controls",
		LastInputDescriptor:    "bundle=com.google.Chrome|window=Google|role=textfield|label=Search",
		Screenshot:             "dGVzdA==",
		VisibleInputCandidates: 2,
		CaptureConfidence:      0.32,
	}

	plan := planNavigatorCommand(command, ctx, false)
	if plan.IntentClass != navigatorIntentAmbiguous {
		t.Fatalf("baseline intent = %q, want ambiguous", plan.IntentClass)
	}

	escalated := maybeEscalateNavigatorPlan(context.Background(), &stubADK{
		navigatorEscalateFn: func(_ context.Context, req adk.NavigatorEscalationRequest) (*adk.NavigatorEscalationResult, error) {
			return &adk.NavigatorEscalationResult{
				ResolvedDescriptor: &adk.NavigatorTargetDescriptor{
					Role:        "textfield",
					Label:       "Search",
					WindowTitle: "Google",
					AppName:     "Google Chrome",
				},
				Confidence:             0.91,
				FallbackRecommendation: "safe_immediate",
				Reason:                 "visual_resolution",
			}, nil
		},
	}, nil, "ko", command, ctx, plan, "trace_nav")

	if escalated.IntentClass != navigatorIntentExecuteNow {
		t.Fatalf("escalated intent = %q, want execute_now", escalated.IntentClass)
	}
	if len(escalated.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(escalated.Steps))
	}
	if got := escalated.Steps[0].TargetDescriptor.Label; got != "Search" {
		t.Fatalf("label = %q, want Search", got)
	}
}

func TestNavigatorHeroReplayFixtures(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "navigator_replays", "*.json"))
	if err != nil {
		t.Fatalf("glob fixtures: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected replay fixtures")
	}

	for _, path := range files {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			var fixture navigatorReplayFixture
			if err := json.Unmarshal(raw, &fixture); err != nil {
				t.Fatalf("decode fixture: %v", err)
			}

			plan := planNavigatorCommand(fixture.Command, fixture.Context, false)
			if got := string(plan.IntentClass); got != fixture.Expect.IntentClass {
				t.Fatalf("intent = %q, want %q", got, fixture.Expect.IntentClass)
			}
			if len(plan.Steps) != fixture.Expect.StepCount {
				t.Fatalf("step count = %d, want %d", len(plan.Steps), fixture.Expect.StepCount)
			}
			if fixture.Expect.StepCount == 0 {
				return
			}
			if got := plan.Steps[0].ActionType; got != fixture.Expect.FirstAction {
				t.Fatalf("first action = %q, want %q", got, fixture.Expect.FirstAction)
			}
			if got := plan.Steps[len(plan.Steps)-1].ActionType; got != fixture.Expect.LastAction {
				t.Fatalf("last action = %q, want %q", got, fixture.Expect.LastAction)
			}
		})
	}
}
