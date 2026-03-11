package ws

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"vibecat/realtime-gateway/internal/adk"
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

func sendNavigatorSetup(t *testing.T, conn *websocket.Conn, deviceID string) {
	t.Helper()
	if err := conn.WriteJSON(map[string]any{
		"type": "setup",
		"config": map[string]any{
			"deviceId": deviceID,
			"voice":    "Zephyr",
			"language": "ko",
		},
	}); err != nil {
		t.Fatalf("send setup: %v", err)
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

func TestPlanNavigatorCommandBuildsTextEntryStepsForFocusedInputField(t *testing.T) {
	plan := planNavigatorCommand(`여기에 "gemini live api" 입력해줘`, navigatorContext{
		AppName:      "Google Chrome",
		WindowTitle:  "Google",
		FocusedRole:  "AXTextField",
		FocusedLabel: "Search",
		AXSnapshot:   "window:Google\nfocused:input:AXTextField:Search",
	}, false)

	if plan.IntentClass != navigatorIntentExecuteNow {
		t.Fatalf("intent = %q, want %q", plan.IntentClass, navigatorIntentExecuteNow)
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(plan.Steps))
	}
	if plan.Steps[0].ActionType != "press_ax" {
		t.Fatalf("first action = %q, want press_ax", plan.Steps[0].ActionType)
	}
	if plan.Steps[0].TargetDescriptor.Role != "textfield" {
		t.Fatalf("first role = %q, want textfield", plan.Steps[0].TargetDescriptor.Role)
	}
	if plan.Steps[1].ActionType != "paste_text" {
		t.Fatalf("second action = %q, want paste_text", plan.Steps[1].ActionType)
	}
	if plan.Steps[1].InputText != "gemini live api" {
		t.Fatalf("inputText = %q, want quoted payload", plan.Steps[1].InputText)
	}
}

func TestPlanNavigatorCommandBuildsInputFieldFocusStepWithoutPayload(t *testing.T) {
	plan := planNavigatorCommand("검색창 찾아줘", navigatorContext{
		AppName:     "Google Chrome",
		WindowTitle: "Google",
		AXSnapshot:  "window:Google\ninput:AXTextField:Search",
	}, false)

	if len(plan.Steps) != 1 {
		t.Fatalf("steps = %d, want 1", len(plan.Steps))
	}
	if plan.Steps[0].ActionType != "press_ax" {
		t.Fatalf("action = %q, want press_ax", plan.Steps[0].ActionType)
	}
	if plan.Steps[0].TargetDescriptor.Label != "search" {
		t.Fatalf("label = %q, want search", plan.Steps[0].TargetDescriptor.Label)
	}
}

func TestPlanNavigatorCommandUsesInputFieldHintForImplicitTextEntry(t *testing.T) {
	plan := planNavigatorCommand(`거기에 "gemini live api" 입력해줘`, navigatorContext{
		AppName:        "Google Chrome",
		WindowTitle:    "Google",
		FocusedRole:    "AXGroup",
		InputFieldHint: "Search",
		AXSnapshot:     "window:Google\nAXGroup:Search controls",
		FocusStableMs:  950,
	}, false)

	if plan.IntentClass != navigatorIntentExecuteNow {
		t.Fatalf("intent = %q, want %q", plan.IntentClass, navigatorIntentExecuteNow)
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(plan.Steps))
	}
	if got := plan.Steps[0].TargetDescriptor.Label; got != "Search" {
		t.Fatalf("label = %q, want Search", got)
	}
}

func TestPlanNavigatorCommandUsesLastInputFieldDescriptorWhenHintIsMissing(t *testing.T) {
	plan := planNavigatorCommand(`거기에 "gemini live api" 입력해줘`, navigatorContext{
		AppName:             "Google Chrome",
		WindowTitle:         "Google",
		FocusedRole:         "AXGroup",
		LastInputDescriptor: "bundle=com.google.Chrome|window=Google|role=textfield|label=Search",
		AXSnapshot:          "window:Google\nAXGroup:Search controls",
		FocusStableMs:       980,
		CaptureConfidence:   0.84,
	}, false)

	if plan.IntentClass != navigatorIntentExecuteNow {
		t.Fatalf("intent = %q, want %q", plan.IntentClass, navigatorIntentExecuteNow)
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(plan.Steps))
	}
	if got := plan.Steps[0].TargetDescriptor.Label; got != "Search" {
		t.Fatalf("label = %q, want Search", got)
	}
}

func TestPlanNavigatorCommandClarifiesWhenCurrentTargetIsUnstable(t *testing.T) {
	plan := planNavigatorCommand(`여기에 "gemini live api" 입력해줘`, navigatorContext{
		AppName:       "Google Chrome",
		WindowTitle:   "Google",
		FocusedRole:   "AXGroup",
		FocusedLabel:  "Search controls",
		AXSnapshot:    "window:Google\nAXGroup:Search controls",
		FocusStableMs: 120,
	}, false)

	if plan.IntentClass != navigatorIntentAmbiguous {
		t.Fatalf("intent = %q, want %q", plan.IntentClass, navigatorIntentAmbiguous)
	}
	if !strings.Contains(strings.ToLower(plan.ClarifyQuestion), "input field") {
		t.Fatalf("clarify question = %q, want input-field guidance", plan.ClarifyQuestion)
	}
}

func TestPlanNavigatorCommandClarifiesWhenMultipleInputCandidatesAreVisible(t *testing.T) {
	plan := planNavigatorCommand(`여기에 "gemini live api" 입력해줘`, navigatorContext{
		AppName:                "Google Chrome",
		WindowTitle:            "Google",
		FocusedRole:            "AXGroup",
		FocusedLabel:           "Search controls",
		AXSnapshot:             "window:Google\ninput:AXTextField:Search\ninput:AXTextField:Address",
		VisibleInputCandidates: 2,
		FocusStableMs:          640,
		CaptureConfidence:      0.41,
	}, false)

	if plan.IntentClass != navigatorIntentAmbiguous {
		t.Fatalf("intent = %q, want %q", plan.IntentClass, navigatorIntentAmbiguous)
	}
	if !strings.Contains(strings.ToLower(plan.ClarifyQuestion), "possible input fields") {
		t.Fatalf("clarify question = %q, want multi-input guidance", plan.ClarifyQuestion)
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
	taskID := state.startPlan("run it", []navigatorStep{
		{ID: "step-1"},
		{ID: "step-2"},
	})

	step, ok := state.nextStep()
	if !ok || step.ID != "step-1" {
		t.Fatalf("next step = %#v, %v", step, ok)
	}
	if !state.acceptsRefresh("run it", taskID, "step-1") {
		t.Fatal("expected current step refresh to be accepted")
	}
	if state.acceptsRefresh("run it", taskID, "step-2") {
		t.Fatal("stale or future step refresh should be rejected")
	}
	if state.acceptsRefresh("other command", taskID, "step-1") {
		t.Fatal("mismatched command should be rejected")
	}
	state.clearCurrentStep()
	if state.acceptsRefresh("run it", taskID, "step-1") {
		t.Fatal("cleared current step should reject refresh")
	}
}

func TestNavigatorSessionStateRejectsWrongTaskID(t *testing.T) {
	var state navigatorSessionState
	state.startPlan("run it", []navigatorStep{{ID: "step-1"}})

	step, ok := state.nextStep()
	if !ok {
		t.Fatal("expected current step")
	}
	if state.acceptsRefresh("run it", "task_other", step.ID) {
		t.Fatal("refresh with the wrong task id should be rejected")
	}
}

func TestHandlerNavigatorCommandClarifiesAmbiguousRequest(t *testing.T) {
	reg := NewRegistry()
	server := httptest.NewServer(Handler(reg, nil, nil, nil, nil, NewInMemoryActionStateStore()))
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
	server := httptest.NewServer(Handler(reg, nil, nil, nil, nil, NewInMemoryActionStateStore()))
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
	if accepted["taskId"] == nil || accepted["taskId"] == "" {
		t.Fatal("missing taskId")
	}
	if planned["taskId"] == nil || planned["taskId"] == "" {
		t.Fatal("missing taskId on planned step")
	}
}

func TestHandlerNavigatorCompletionEnqueuesBackgroundReplay(t *testing.T) {
	reg := NewRegistry()
	backgroundCalls := make(chan adk.NavigatorBackgroundRequest, 1)
	server := httptest.NewServer(Handler(reg, nil, &stubADK{
		navigatorBackgroundFn: func(_ context.Context, req adk.NavigatorBackgroundRequest) (*adk.NavigatorBackgroundResult, error) {
			backgroundCalls <- req
			return &adk.NavigatorBackgroundResult{Summary: "Chrome search opened", Surface: "Chrome"}, nil
		},
	}, nil, nil, NewInMemoryActionStateStore()))
	defer server.Close()

	conn := dialTestWebSocket(t, server.URL)
	defer conn.Close()

	if err := conn.WriteJSON(map[string]any{
		"type":    "navigator.command",
		"command": "검색창 찾아줘",
		"context": map[string]any{
			"appName":                    "Google Chrome",
			"bundleId":                   "com.google.Chrome",
			"frontmostBundleId":          "com.google.Chrome",
			"windowTitle":                "Google",
			"axSnapshot":                 "window:Google\ninput:AXTextField:Search",
			"visibleInputCandidateCount": 1,
			"captureConfidence":          0.88,
			"accessibilityTrusted":       true,
		},
	}); err != nil {
		t.Fatalf("send navigator.command: %v", err)
	}

	accepted := readJSONMessageOfType(t, conn, "navigator.commandAccepted")
	taskID, _ := accepted["taskId"].(string)
	planned := readJSONMessageOfType(t, conn, "navigator.stepPlanned")
	step, _ := planned["step"].(map[string]any)

	if err := conn.WriteJSON(map[string]any{
		"type":            "navigator.refreshContext",
		"command":         "검색창 찾아줘",
		"taskId":          taskID,
		"step":            step,
		"status":          "success",
		"observedOutcome": "Focused the target input field",
		"context": map[string]any{
			"appName":                  "Google Chrome",
			"bundleId":                 "com.google.Chrome",
			"frontmostBundleId":        "com.google.Chrome",
			"windowTitle":              "Google",
			"focusedRole":              "AXTextField",
			"focusedLabel":             "Search",
			"axSnapshot":               "window:Google\nfocused:input:AXTextField:Search",
			"inputFieldHint":           "Search",
			"lastInputFieldDescriptor": "bundle=com.google.Chrome|window=Google|role=textfield|label=Search",
			"captureConfidence":        0.91,
			"accessibilityTrusted":     true,
		},
	}); err != nil {
		t.Fatalf("send navigator.refreshContext: %v", err)
	}

	_ = readJSONMessageOfType(t, conn, "navigator.stepVerified")
	_ = readJSONMessageOfType(t, conn, "navigator.completed")

	select {
	case req := <-backgroundCalls:
		if req.TaskID != taskID {
			t.Fatalf("background taskId = %q, want %q", req.TaskID, taskID)
		}
		if req.Outcome != "completed" {
			t.Fatalf("background outcome = %q, want completed", req.Outcome)
		}
		if len(req.Steps) != 1 {
			t.Fatalf("background steps = %d, want 1", len(req.Steps))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected navigator background request")
	}
}

func TestHandlerNavigatorCommandClarifiesBeforeReplacingActiveTask(t *testing.T) {
	reg := NewRegistry()
	server := httptest.NewServer(Handler(reg, nil, nil, nil, nil, NewInMemoryActionStateStore()))
	defer server.Close()

	conn := dialTestWebSocket(t, server.URL)
	defer conn.Close()

	first := map[string]any{
		"type":    "navigator.command",
		"command": "공식 문서 쪽으로 가보자",
		"context": map[string]any{
			"appName":              "Antigravity IDE",
			"windowTitle":          "AuthServiceTests.swift",
			"selectedText":         "AuthServiceTests missing token",
			"accessibilityTrusted": true,
		},
	}
	if err := conn.WriteJSON(first); err != nil {
		t.Fatalf("send first navigator.command: %v", err)
	}
	_ = readJSONMessageOfType(t, conn, "navigator.commandAccepted")
	_ = readJSONMessageOfType(t, conn, "navigator.stepPlanned")

	second := map[string]any{
		"type":    "navigator.command",
		"command": "검색창 찾아줘",
		"context": map[string]any{
			"appName":              "Google Chrome",
			"windowTitle":          "Google",
			"axSnapshot":           "window:Google\ninput:AXTextField:Search",
			"accessibilityTrusted": true,
		},
	}
	if err := conn.WriteJSON(second); err != nil {
		t.Fatalf("send second navigator.command: %v", err)
	}

	msg := readJSONMessageOfType(t, conn, "navigator.intentClarificationNeeded")
	question, _ := msg["question"].(string)
	if !strings.Contains(strings.ToLower(question), "already working on") {
		t.Fatalf("question = %q, want active-task clarification", question)
	}
}

func TestHandlerRestoresNavigatorTaskAcrossReconnect(t *testing.T) {
	reg := NewRegistry()
	store := NewInMemoryActionStateStore()
	server := httptest.NewServer(Handler(reg, nil, nil, nil, nil, store))
	defer server.Close()

	conn1 := dialTestWebSocket(t, server.URL)
	defer conn1.Close()

	sendNavigatorSetup(t, conn1, "device_restore")
	_ = readJSONMessageOfType(t, conn1, "setupComplete")

	if err := conn1.WriteJSON(map[string]any{
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

	accepted := readJSONMessageOfType(t, conn1, "navigator.commandAccepted")
	taskID, _ := accepted["taskId"].(string)
	if taskID == "" {
		t.Fatal("missing taskId")
	}
	_ = readJSONMessageOfType(t, conn1, "navigator.stepPlanned")
	if err := conn1.Close(); err != nil {
		t.Fatalf("close first connection: %v", err)
	}

	conn2 := dialTestWebSocket(t, server.URL)
	defer conn2.Close()

	sendNavigatorSetup(t, conn2, "device_restore")
	restored := readJSONMessageOfType(t, conn2, "navigator.guidedMode")
	if restored["reason"] != "restored_task_state" {
		t.Fatalf("reason = %v, want restored_task_state", restored["reason"])
	}
	if restored["taskId"] != taskID {
		t.Fatalf("taskId = %v, want %s", restored["taskId"], taskID)
	}
	_ = readJSONMessageOfType(t, conn2, "setupComplete")
}

func TestHandlerRejectsStaleConnectionAfterReconnect(t *testing.T) {
	reg := NewRegistry()
	store := NewInMemoryActionStateStore()
	server := httptest.NewServer(Handler(reg, nil, nil, nil, nil, store))
	defer server.Close()

	conn1 := dialTestWebSocket(t, server.URL)
	defer conn1.Close()

	sendNavigatorSetup(t, conn1, "device_stale")
	_ = readJSONMessageOfType(t, conn1, "setupComplete")

	if err := conn1.WriteJSON(map[string]any{
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

	accepted := readJSONMessageOfType(t, conn1, "navigator.commandAccepted")
	taskID, _ := accepted["taskId"].(string)
	planned := readJSONMessageOfType(t, conn1, "navigator.stepPlanned")
	step, _ := planned["step"].(map[string]any)

	conn2 := dialTestWebSocket(t, server.URL)
	defer conn2.Close()

	sendNavigatorSetup(t, conn2, "device_stale")
	_ = readJSONMessageOfType(t, conn2, "navigator.guidedMode")
	_ = readJSONMessageOfType(t, conn2, "setupComplete")

	if err := conn1.WriteJSON(map[string]any{
		"type":            "navigator.refreshContext",
		"command":         "공식 문서 쪽으로 가보자",
		"taskId":          taskID,
		"step":            step,
		"status":          "success",
		"observedOutcome": "Chrome opened the official docs search.",
		"context": map[string]any{
			"appName":              "Google Chrome",
			"windowTitle":          "Google Search",
			"focusedRole":          "AXWebArea",
			"focusedLabel":         "Search results",
			"accessibilityTrusted": true,
		},
	}); err != nil {
		t.Fatalf("send stale navigator.refreshContext: %v", err)
	}

	failed := readJSONMessageOfType(t, conn1, "navigator.failed")
	reason, _ := failed["reason"].(string)
	if !strings.Contains(strings.ToLower(reason), "stale") {
		t.Fatalf("reason = %q, want stale connection rejection", reason)
	}
}
