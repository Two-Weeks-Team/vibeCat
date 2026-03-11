package ws

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func readJSONMessageOfType(t *testing.T, conn *websocket.Conn, wantType string) map[string]any {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read %s: %v", wantType, err)
		}

		var msg map[string]any
		if err := json.Unmarshal(payload, &msg); err != nil {
			t.Fatalf("unmarshal %s: %v", wantType, err)
		}
		if msg["type"] == "traceEvent" {
			continue
		}
		if msg["type"] != wantType {
			t.Fatalf("type = %v, want %s", msg["type"], wantType)
		}
		return msg
	}
}

func TestClassifyNavigatorIntentTreatsImplicitApplyAsExecute(t *testing.T) {
	intent, confidence, question := classifyNavigatorIntent("이거 반영해")
	if intent != navigatorIntentExecuteNow {
		t.Fatalf("intent = %q, want %q", intent, navigatorIntentExecuteNow)
	}
	if confidence < 0.58 {
		t.Fatalf("confidence = %v, want >= 0.58", confidence)
	}
	if question != "" {
		t.Fatalf("question = %q, want empty", question)
	}
}

func TestPlanNavigatorCommandBuildsDocsLookupSteps(t *testing.T) {
	plan := planNavigatorCommand("공식 문서 쪽으로 가보자", navigatorContext{
		AppName:      "Antigravity IDE",
		WindowTitle:  "AuthServiceTests.swift",
		SelectedText: "AuthServiceTests failing with missing token",
	}, false)

	if plan.IntentClass != navigatorIntentFindLookup && plan.IntentClass != navigatorIntentOpenNavigate {
		t.Fatalf("intent = %q, want docs-related navigator intent", plan.IntentClass)
	}
	if len(plan.Steps) == 0 {
		t.Fatal("expected planned steps")
	}
	last := plan.Steps[len(plan.Steps)-1]
	if last.ActionType != "open_url" {
		t.Fatalf("last action = %q, want open_url", last.ActionType)
	}
	if !strings.Contains(last.URL, "google.com/search") {
		t.Fatalf("url = %q, want google search", last.URL)
	}
}

func TestPlanNavigatorCommandBypassesRiskBlockForAnalyzeOnlySensitiveRequest(t *testing.T) {
	plan := planNavigatorCommand("배포 토큰이 왜 필요한지 설명해줘", navigatorContext{
		AppName: "Antigravity IDE",
	}, false)

	if plan.IntentClass != navigatorIntentAnalyzeOnly {
		t.Fatalf("intent = %q, want %q", plan.IntentClass, navigatorIntentAnalyzeOnly)
	}
	if plan.RiskQuestion != "" {
		t.Fatalf("risk question = %q, want empty", plan.RiskQuestion)
	}
	if len(plan.Steps) != 0 {
		t.Fatalf("steps = %d, want 0", len(plan.Steps))
	}
}

func TestPlanNavigatorCommandAllowsSafeDocsLookupForSensitiveTopic(t *testing.T) {
	plan := planNavigatorCommand("공식 토큰 문서 열어줘", navigatorContext{
		AppName: "Antigravity IDE",
	}, false)

	if plan.RiskQuestion != "" {
		t.Fatalf("risk question = %q, want empty", plan.RiskQuestion)
	}
	if len(plan.Steps) == 0 {
		t.Fatal("expected docs lookup steps")
	}
}

func TestPlanNavigatorCommandUsesSelectedTextForRiskDecision(t *testing.T) {
	plan := planNavigatorCommand("이거 실행해줘", navigatorContext{
		AppName:      "Terminal",
		SelectedText: "rm -rf ~/danger-zone",
	}, false)

	if plan.RiskQuestion == "" {
		t.Fatal("expected risk confirmation for dangerous selected text")
	}
	if plan.RiskReason == "" {
		t.Fatal("expected risk reason")
	}
	if len(plan.Steps) != 0 {
		t.Fatalf("steps = %d, want 0 after risk block", len(plan.Steps))
	}
}

func TestAffirmativeAnswerRequiresClearApproval(t *testing.T) {
	if affirmativeAnswer("maybe") {
		t.Fatal("maybe should not count as affirmative")
	}
	if affirmativeAnswer("why") {
		t.Fatal("why should not count as affirmative")
	}
	if !affirmativeAnswer("y") {
		t.Fatal("single-letter y should count as affirmative")
	}
	if !affirmativeAnswer("yes, do it") {
		t.Fatal("clear confirmation should count as affirmative")
	}
}

func TestNavigatorSessionStateRejectsStaleRefresh(t *testing.T) {
	var state navigatorSessionState
	state.startPlan("run it", []navigatorStep{
		{ID: "step-1"},
		{ID: "step-2"},
	})

	step, ok := state.nextStep()
	if !ok || step.ID != "step-1" {
		t.Fatalf("next step = %#v, %v", step, ok)
	}
	if !state.acceptsRefresh("run it", "step-1") {
		t.Fatal("expected current step refresh to be accepted")
	}
	if state.acceptsRefresh("run it", "step-2") {
		t.Fatal("stale or future step refresh should be rejected")
	}
	if state.acceptsRefresh("other command", "step-1") {
		t.Fatal("mismatched command should be rejected")
	}
	state.clearCurrentStep()
	if state.acceptsRefresh("run it", "step-1") {
		t.Fatal("cleared current step should reject refresh")
	}
}

func TestHandlerNavigatorCommandClarifiesAmbiguousRequest(t *testing.T) {
	reg := NewRegistry()
	server := httptest.NewServer(Handler(reg, nil, nil, nil, nil))
	defer server.Close()

	conn := dialTestWebSocket(t, server.URL)
	defer conn.Close()

	if err := conn.WriteJSON(map[string]any{
		"type":    "navigator.command",
		"command": "이거 한번 봐",
		"context": map[string]any{
			"appName":              "Antigravity IDE",
			"windowTitle":          "AuthServiceTests.swift",
			"selectedText":         "",
			"accessibilityTrusted": true,
		},
	}); err != nil {
		t.Fatalf("send navigator.command: %v", err)
	}

	msg := readJSONMessageOfType(t, conn, "navigator.intentClarificationNeeded")
	if msg["question"] == "" {
		t.Fatal("expected clarification question")
	}
}

func TestHandlerNavigatorCommandPlansStep(t *testing.T) {
	reg := NewRegistry()
	server := httptest.NewServer(Handler(reg, nil, nil, nil, nil))
	defer server.Close()

	conn := dialTestWebSocket(t, server.URL)
	defer conn.Close()

	if err := conn.WriteJSON(map[string]any{
		"type":    "navigator.command",
		"command": "공식 문서 쪽으로 가보자",
		"context": map[string]any{
			"appName":              "Antigravity IDE",
			"windowTitle":          "AuthServiceTests.swift",
			"selectedText":         "AuthServiceTests missing token",
			"accessibilityTrusted": true,
		},
	}); err != nil {
		t.Fatalf("send navigator.command: %v", err)
	}

	accepted := readJSONMessageOfType(t, conn, "navigator.commandAccepted")
	planned := readJSONMessageOfType(t, conn, "navigator.stepPlanned")
	step, _ := planned["step"].(map[string]any)
	if step["actionType"] == nil {
		t.Fatal("missing actionType")
	}
	if accepted["intentClass"] == nil {
		t.Fatal("missing intentClass")
	}
}
