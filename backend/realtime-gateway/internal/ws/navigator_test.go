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
		AppName:     "Antigravity IDE",
		WindowTitle: "AuthServiceTests.swift",
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

func TestHandlerNavigatorCommandClarifiesAmbiguousRequest(t *testing.T) {
	reg := NewRegistry()
	server := httptest.NewServer(Handler(reg, nil, nil, nil, nil))
	defer server.Close()

	conn := dialTestWebSocket(t, server.URL)
	defer conn.Close()

	if err := conn.WriteJSON(map[string]any{
		"type": "navigator.command",
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
		"type": "navigator.command",
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
