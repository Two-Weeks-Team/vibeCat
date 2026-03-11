package ws

import (
	"context"
	"testing"
)

func TestChainedActionStateStoreWarmsEarlierStoresOnLoad(t *testing.T) {
	ctx := context.Background()
	primary := NewInMemoryActionStateStore()
	secondary := NewInMemoryActionStateStore()
	chain := NewChainedActionStateStore(primary, secondary)

	var state navigatorSessionState
	taskID := state.startPlan("공식 문서 쪽으로 가보자", []navigatorStep{
		{ID: "step-1", ActionType: "open_url", TargetApp: "Chrome"},
	})
	state.bindLease("device-123", "conn-123")
	state.nextStep()

	if err := secondary.Save(ctx, "device-123", state); err != nil {
		t.Fatalf("seed secondary store: %v", err)
	}

	loaded, ok, err := chain.Load(ctx, "device-123")
	if err != nil {
		t.Fatalf("load chain: %v", err)
	}
	if !ok {
		t.Fatal("expected chained store hit")
	}
	if loaded.activeTaskID != taskID {
		t.Fatalf("taskID = %q, want %q", loaded.activeTaskID, taskID)
	}

	warmed, ok, err := primary.Load(ctx, "device-123")
	if err != nil {
		t.Fatalf("load warmed primary store: %v", err)
	}
	if !ok {
		t.Fatal("expected primary store to be warmed")
	}
	if warmed.currentStepID != "step-1" {
		t.Fatalf("currentStepID = %q, want step-1", warmed.currentStepID)
	}
}

func TestFirestoreActionStateRecordRoundTripPreservesNavigatorState(t *testing.T) {
	var state navigatorSessionState
	taskID := state.startPlan("여기에 입력해줘", []navigatorStep{
		{
			ID:         "focus_input_field",
			ActionType: "press_ax",
			TargetApp:  "Chrome",
			TargetDescriptor: navigatorTargetDescriptor{
				Role:        "textfield",
				Label:       "Search",
				WindowTitle: "Google",
				AppName:     "Chrome",
			},
		},
		{
			ID:         "paste_input_text",
			ActionType: "paste_text",
			TargetApp:  "Chrome",
			InputText:  "gemini live api",
		},
	})
	state.bindLease("device-42", "conn-42")
	state.pendingRiskyCommand = "git push 해줘"
	state.currentStepID = "focus_input_field"
	state.nextStepIndex = 1
	state.lastVerifiedContextHash = "hash123"

	record := newFirestoreActionStateRecord("device-42", state)
	if record.Status != "awaiting_risk_confirmation" {
		t.Fatalf("status = %q, want awaiting_risk_confirmation", record.Status)
	}
	if record.PromptState != "risky_action_confirmation" {
		t.Fatalf("promptState = %q, want risky_action_confirmation", record.PromptState)
	}

	restored := record.toNavigatorSessionState()
	if restored.activeTaskID != taskID {
		t.Fatalf("taskID = %q, want %q", restored.activeTaskID, taskID)
	}
	if restored.pendingRiskyCommand != "git push 해줘" {
		t.Fatalf("pendingRiskyCommand = %q, want original value", restored.pendingRiskyCommand)
	}
	if restored.deviceID != "device-42" {
		t.Fatalf("deviceID = %q, want device-42", restored.deviceID)
	}
	if restored.connectionID != "conn-42" {
		t.Fatalf("connectionID = %q, want conn-42", restored.connectionID)
	}
	if len(restored.steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(restored.steps))
	}
	if restored.steps[0].TargetDescriptor.Label != "Search" {
		t.Fatalf("label = %q, want Search", restored.steps[0].TargetDescriptor.Label)
	}
	if restored.lastVerifiedContextHash != "hash123" {
		t.Fatalf("lastVerifiedContextHash = %q, want hash123", restored.lastVerifiedContextHash)
	}
}
