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

type slowActionStateStore struct {
	loadDelay   time.Duration
	writeDelay  time.Duration
	deleteDelay time.Duration
}

func (s slowActionStateStore) Load(ctx context.Context, owner string) (navigatorSessionState, bool, error) {
	select {
	case <-time.After(s.loadDelay):
		return navigatorSessionState{}, false, nil
	case <-ctx.Done():
		return navigatorSessionState{}, false, ctx.Err()
	}
}

func (s slowActionStateStore) Save(ctx context.Context, owner string, state navigatorSessionState) error {
	select {
	case <-time.After(s.writeDelay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s slowActionStateStore) Delete(ctx context.Context, owner string) error {
	select {
	case <-time.After(s.deleteDelay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
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
	if last.Surface != "chrome" {
		t.Fatalf("surface = %q, want chrome", last.Surface)
	}
	if last.MacroID != "open_docs_search" {
		t.Fatalf("macroID = %q, want open_docs_search", last.MacroID)
	}
	if last.Narration == "" {
		t.Fatal("expected narration")
	}
	if last.VerifyContract == nil || last.VerifyContract.ExpectedBundleID != "com.google.Chrome" {
		t.Fatalf("verify contract = %#v, want Chrome bundle", last.VerifyContract)
	}
	if last.FallbackActionType != "hotkey" {
		t.Fatalf("fallback action = %q, want hotkey", last.FallbackActionType)
	}
	if len(last.FallbackHotkey) != 2 || last.FallbackHotkey[0] != "command" || last.FallbackHotkey[1] != "l" {
		t.Fatalf("fallback hotkey = %#v, want [command l]", last.FallbackHotkey)
	}
	if last.TimeoutMs != 1500 {
		t.Fatalf("timeout = %d, want 1500", last.TimeoutMs)
	}
	if last.ProofLevel != "strong" {
		t.Fatalf("proofLevel = %q, want strong", last.ProofLevel)
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

func TestPlanNavigatorCommandRequiresDerivedTextForScreenContentInsertion(t *testing.T) {
	plan := planNavigatorCommand("여기에 지금 최근 수정한 파일 세 개 이름을 입력해줘", navigatorContext{
		AppName:                "Codex",
		WindowTitle:            "Codex",
		FocusedRole:            "AXTextArea",
		FocusedLabel:           "후속 변경 사항을 부탁하세요",
		LastInputDescriptor:    "bundle=com.openai.codex|window=Codex|role=textarea|label=후속 변경 사항을 부탁하세요",
		VisibleInputCandidates: 1,
		CaptureConfidence:      0.84,
	}, false)

	if plan.IntentClass != navigatorIntentAmbiguous {
		t.Fatalf("intent = %q, want %q", plan.IntentClass, navigatorIntentAmbiguous)
	}
	if len(plan.Steps) != 0 {
		t.Fatalf("steps = %d, want 0 until screen-derived text is resolved", len(plan.Steps))
	}
}

func TestPlanNavigatorCommandRequiresDerivedTextForFindThenInsertFollowup(t *testing.T) {
	plan := planNavigatorCommand("그럼 8명 세 개만 찾아서 입력해 줄래?", navigatorContext{
		AppName:                "Codex",
		WindowTitle:            "Codex",
		FocusedRole:            "AXTextArea",
		FocusedLabel:           "후속 변경 사항을 부탁하세요",
		LastInputDescriptor:    "bundle=com.openai.codex|window=Codex|role=textarea|label=후속 변경 사항을 부탁하세요",
		VisibleInputCandidates: 1,
		CaptureConfidence:      0.86,
	}, false)

	if plan.IntentClass != navigatorIntentAmbiguous {
		t.Fatalf("intent = %q, want %q", plan.IntentClass, navigatorIntentAmbiguous)
	}
	if len(plan.Steps) != 0 {
		t.Fatalf("steps = %d, want 0 until visible text is resolved", len(plan.Steps))
	}
	if !strings.Contains(strings.ToLower(plan.ClarifyQuestion), "exact text to type") {
		t.Fatalf("clarify question = %q, want text-resolution guidance", plan.ClarifyQuestion)
	}
}

func TestPlanNavigatorCommandUsesIntrinsicAssistantNameForTextEntry(t *testing.T) {
	plan := planNavigatorCommand("네 이름을 입력해 보고", navigatorContext{
		AppName:                "Codex",
		WindowTitle:            "Codex",
		FocusedRole:            "AXTextArea",
		FocusedLabel:           "후속 변경 사항을 부탁하세요",
		LastInputDescriptor:    "bundle=com.openai.codex|window=Codex|role=textarea|label=후속 변경 사항을 부탁하세요",
		VisibleInputCandidates: 1,
		CaptureConfidence:      0.88,
	}, false)

	if plan.IntentClass != navigatorIntentExecuteNow {
		t.Fatalf("intent = %q, want %q", plan.IntentClass, navigatorIntentExecuteNow)
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(plan.Steps))
	}
	if got := plan.Steps[1].ActionType; got != "paste_text" {
		t.Fatalf("second action = %q, want paste_text", got)
	}
	if got := plan.Steps[1].InputText; got != "VibeCat" {
		t.Fatalf("inputText = %q, want VibeCat", got)
	}
}

func TestPlanNavigatorCommandTreatsAlphabetRangeAsLiteralTextPayload(t *testing.T) {
	plan := planNavigatorCommand("지금 입력 창에 A부터 Z까지 입력해 줘.", navigatorContext{
		AppName:                "Codex",
		WindowTitle:            "Codex",
		FocusedRole:            "AXTextArea",
		FocusedLabel:           "후속 변경 사항을 부탁하세요",
		LastInputDescriptor:    "bundle=com.openai.codex|window=Codex|role=textarea|label=후속 변경 사항을 부탁하세요",
		VisibleInputCandidates: 1,
		CaptureConfidence:      0.91,
	}, false)

	if plan.IntentClass != navigatorIntentExecuteNow {
		t.Fatalf("intent = %q, want %q", plan.IntentClass, navigatorIntentExecuteNow)
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(plan.Steps))
	}
	if got := plan.Steps[1].ActionType; got != "paste_text" {
		t.Fatalf("second action = %q, want paste_text", got)
	}
	if got := plan.Steps[1].InputText; got != "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
		t.Fatalf("inputText = %q, want alphabet payload", got)
	}
}

func TestPlanNavigatorCommandTreatsAlphabetRangeWithSuffixAsLiteralTextPayload(t *testing.T) {
	plan := planNavigatorCommand("지금 입력 창에 A부터 Z 뒤에 한 칸 띄고 하이라고 입력해 줘.", navigatorContext{
		AppName:                "Codex",
		WindowTitle:            "Codex",
		FocusedRole:            "AXTextArea",
		FocusedLabel:           "후속 변경 사항을 부탁하세요",
		LastInputDescriptor:    "bundle=com.openai.codex|window=Codex|role=textarea|label=후속 변경 사항을 부탁하세요",
		VisibleInputCandidates: 1,
		CaptureConfidence:      0.91,
	}, false)

	if plan.IntentClass != navigatorIntentExecuteNow {
		t.Fatalf("intent = %q, want %q", plan.IntentClass, navigatorIntentExecuteNow)
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(plan.Steps))
	}
	if got := plan.Steps[1].InputText; got != "ABCDEFGHIJKLMNOPQRSTUVWXYZ 하이" {
		t.Fatalf("inputText = %q, want compound alphabet payload", got)
	}
}

func TestPlanNavigatorCommandTreatsRelativeAppendAsPayload(t *testing.T) {
	plan := planNavigatorCommand("Z 뒤에 하이라고 입력해 줘.", navigatorContext{
		AppName:                "Codex",
		WindowTitle:            "Codex",
		FocusedRole:            "AXTextArea",
		FocusedLabel:           "후속 변경 사항을 부탁하세요",
		LastInputDescriptor:    "bundle=com.openai.codex|window=Codex|role=textarea|label=후속 변경 사항을 부탁하세요",
		VisibleInputCandidates: 1,
		CaptureConfidence:      0.94,
	}, false)

	if plan.IntentClass != navigatorIntentExecuteNow {
		t.Fatalf("intent = %q, want %q", plan.IntentClass, navigatorIntentExecuteNow)
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(plan.Steps))
	}
	if got := plan.Steps[1].InputText; got != "하이" {
		t.Fatalf("inputText = %q, want appended payload", got)
	}
}

func TestPlanNavigatorCommandTreatsBareDirectTextAsPayload(t *testing.T) {
	plan := planNavigatorCommand("하이라고 입력해 줘.", navigatorContext{
		AppName:                "Codex",
		WindowTitle:            "Codex",
		FocusedRole:            "AXTextArea",
		FocusedLabel:           "후속 변경 사항을 부탁하세요",
		LastInputDescriptor:    "bundle=com.openai.codex|window=Codex|role=textarea|label=후속 변경 사항을 부탁하세요",
		VisibleInputCandidates: 1,
		CaptureConfidence:      0.94,
	}, false)

	if plan.IntentClass != navigatorIntentExecuteNow {
		t.Fatalf("intent = %q, want %q", plan.IntentClass, navigatorIntentExecuteNow)
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(plan.Steps))
	}
	if got := plan.Steps[1].InputText; got != "하이" {
		t.Fatalf("inputText = %q, want bare direct payload", got)
	}
}

func TestPlanNavigatorCommandClarifiesWhenExactTextIsMissing(t *testing.T) {
	plan := planNavigatorCommand("검색창에 입력해줘", navigatorContext{
		AppName:                "Google Chrome",
		WindowTitle:            "Google",
		FocusedRole:            "AXTextField",
		FocusedLabel:           "Search",
		VisibleInputCandidates: 1,
		CaptureConfidence:      0.9,
	}, false)

	if plan.IntentClass != navigatorIntentAmbiguous {
		t.Fatalf("intent = %q, want %q", plan.IntentClass, navigatorIntentAmbiguous)
	}
	if len(plan.Steps) != 0 {
		t.Fatalf("steps = %d, want 0 when exact text is missing", len(plan.Steps))
	}
	if !strings.Contains(strings.ToLower(plan.ClarifyQuestion), "exact text") {
		t.Fatalf("clarify question = %q, want exact-text clarification", plan.ClarifyQuestion)
	}
	if plan.ClarifyMode != navigatorClarificationProvideDetail {
		t.Fatalf("clarify mode = %q, want %q", plan.ClarifyMode, navigatorClarificationProvideDetail)
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

func TestPlanNavigatorCommandBuildsSystemActionStepForVolume(t *testing.T) {
	plan := planNavigatorCommand("볼륨 15 줄여줘", navigatorContext{
		AppName: "Codex",
	}, false)

	if plan.IntentClass != navigatorIntentExecuteNow {
		t.Fatalf("intent = %q, want %q", plan.IntentClass, navigatorIntentExecuteNow)
	}
	if len(plan.Steps) != 1 {
		t.Fatalf("steps = %d, want 1", len(plan.Steps))
	}
	if got := plan.Steps[0].ActionType; got != "system_action" {
		t.Fatalf("action = %q, want system_action", got)
	}
	if got := plan.Steps[0].SystemCommand; got != "volume" {
		t.Fatalf("system command = %q, want volume", got)
	}
	if got := plan.Steps[0].SystemValue; got != "down" {
		t.Fatalf("system value = %q, want down", got)
	}
	if got := plan.Steps[0].SystemAmount; got != 15 {
		t.Fatalf("system amount = %d, want 15", got)
	}
}

func TestBuildTerminalCommandStepsEmitExecutionContract(t *testing.T) {
	steps := buildTerminalCommandSteps("run `swift test`", navigatorContext{AppName: "Codex", WindowTitle: "Codex"}, 0.91)
	if len(steps) != 3 {
		t.Fatalf("steps = %d, want 3", len(steps))
	}
	paste := steps[1]
	if paste.Surface != "terminal" {
		t.Fatalf("surface = %q, want terminal", paste.Surface)
	}
	if paste.MacroID != "paste_terminal_command" {
		t.Fatalf("macroID = %q, want paste_terminal_command", paste.MacroID)
	}
	if paste.VerifyContract == nil || paste.VerifyContract.ExpectedBundleID != "com.apple.Terminal" {
		t.Fatalf("verify contract = %#v, want Terminal bundle", paste.VerifyContract)
	}
	if paste.MaxLocalRetries != 1 {
		t.Fatalf("maxLocalRetries = %d, want 1", paste.MaxLocalRetries)
	}
	if paste.ProofLevel != "strict" {
		t.Fatalf("proofLevel = %q, want strict", paste.ProofLevel)
	}

	submit := steps[2]
	if submit.MacroID != "submit_terminal_command" {
		t.Fatalf("submit macroID = %q, want submit_terminal_command", submit.MacroID)
	}
	if submit.TimeoutMs != 900 {
		t.Fatalf("submit timeout = %d, want 900", submit.TimeoutMs)
	}
}

func TestBuildAntigravityInlineStepsEmitExecutionContract(t *testing.T) {
	steps := buildAntigravityInlineSteps("이 에러를 고쳐줘", navigatorContext{AppName: "Chrome", WindowTitle: "Codex"}, 0.9)
	if len(steps) != 4 {
		t.Fatalf("steps = %d, want 4", len(steps))
	}
	paste := steps[2]
	if paste.Surface != "antigravity" {
		t.Fatalf("surface = %q, want antigravity", paste.Surface)
	}
	if paste.MacroID != "paste_antigravity_instruction" {
		t.Fatalf("macroID = %q, want paste_antigravity_instruction", paste.MacroID)
	}
	if paste.VerifyContract == nil || paste.VerifyContract.ExpectedBundleID != "com.openai.codex" {
		t.Fatalf("verify contract = %#v, want Antigravity bundle", paste.VerifyContract)
	}
	if paste.ProofLevel != "strict" {
		t.Fatalf("proofLevel = %q, want strict", paste.ProofLevel)
	}
	submit := steps[3]
	if submit.MacroID != "submit_antigravity_instruction" {
		t.Fatalf("submit macroID = %q, want submit_antigravity_instruction", submit.MacroID)
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

func TestPlanNavigatorCommandUsesLastInputFieldDescriptorRoleWhenFocusedRoleIsNotInput(t *testing.T) {
	plan := planNavigatorCommand(`거기에 "ABCDEFGHIJKLMNOPQRSTUVWXYZ" 입력해줘`, navigatorContext{
		AppName:             "Codex",
		WindowTitle:         "Codex",
		FocusedRole:         "AXGroup",
		LastInputDescriptor: "bundle=com.openai.codex|window=Codex|role=textarea|label=후속 변경 사항을 부탁하세요",
		CaptureConfidence:   0.86,
	}, false)

	if plan.IntentClass != navigatorIntentExecuteNow {
		t.Fatalf("intent = %q, want %q", plan.IntentClass, navigatorIntentExecuteNow)
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(plan.Steps))
	}
	if got := plan.Steps[0].TargetDescriptor.Role; got != "textarea" {
		t.Fatalf("role = %q, want textarea", got)
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

func TestMaybeEscalateNavigatorPlanUsesResolvedTextForScreenDerivedEntry(t *testing.T) {
	command := "그럼 8명 세 개만 찾아서 입력해 줄래?"
	ctx := navigatorContext{
		AppName:                "Codex",
		WindowTitle:            "Codex",
		FocusedRole:            "AXTextArea",
		FocusedLabel:           "후속 변경 사항을 부탁하세요",
		LastInputDescriptor:    "bundle=com.openai.codex|window=Codex|role=textarea|label=후속 변경 사항을 부탁하세요",
		Screenshot:             "dGVzdA==",
		VisibleInputCandidates: 1,
		CaptureConfidence:      0.83,
	}

	plan := planNavigatorCommand(command, ctx, false)
	if plan.IntentClass != navigatorIntentAmbiguous {
		t.Fatalf("baseline intent = %q, want ambiguous", plan.IntentClass)
	}

	escalated := maybeEscalateNavigatorPlan(context.Background(), &stubADK{
		navigatorEscalateFn: func(_ context.Context, req adk.NavigatorEscalationRequest) (*adk.NavigatorEscalationResult, error) {
			return &adk.NavigatorEscalationResult{
				ResolvedText:           "AppDelegate.swift\nAudioDeviceMonitor.swift\nNavigatorVoiceCommandDetector.swift",
				Confidence:             0.9,
				FallbackRecommendation: "safe_immediate",
				Reason:                 "visible_text_extracted",
			}, nil
		},
	}, nil, "ko", command, ctx, plan, "trace_nav_screen_text")

	if escalated.IntentClass != navigatorIntentExecuteNow {
		t.Fatalf("escalated intent = %q, want execute_now", escalated.IntentClass)
	}
	if len(escalated.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(escalated.Steps))
	}
	if got := escalated.Steps[1].ActionType; got != "paste_text" {
		t.Fatalf("last action = %q, want paste_text", got)
	}
	if got := escalated.Steps[1].InputText; !strings.Contains(got, "AppDelegate.swift") {
		t.Fatalf("inputText = %q, want extracted filenames", got)
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

func TestHandlerNavigatorCommandClarificationIncludesProvideDetailsModeForMissingText(t *testing.T) {
	reg := NewRegistry()
	server := httptest.NewServer(Handler(reg, nil, nil, nil, nil, NewInMemoryActionStateStore()))
	defer server.Close()

	conn := dialTestWebSocket(t, server.URL)
	defer conn.Close()

	if err := conn.WriteJSON(map[string]any{
		"type":    "navigator.command",
		"command": "검색창에 입력해줘",
		"context": map[string]any{
			"appName":                    "Codex",
			"windowTitle":                "Codex",
			"focusedRole":                "AXTextArea",
			"focusedLabel":               "후속 변경 사항을 부탁하세요",
			"lastInputFieldDescriptor":   "bundle=com.openai.codex|window=Codex|role=textarea|label=후속 변경 사항을 부탁하세요",
			"visibleInputCandidateCount": 1,
			"captureConfidence":          0.93,
			"accessibilityTrusted":       true,
		},
	}); err != nil {
		t.Fatalf("send navigator.command: %v", err)
	}

	msg := readJSONMessageOfType(t, conn, "navigator.intentClarificationNeeded")
	if got := msg["responseMode"]; got != string(navigatorClarificationProvideDetail) {
		t.Fatalf("responseMode = %v, want %q", got, navigatorClarificationProvideDetail)
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

func TestHandlerNavigatorCommandDoesNotBlockOnSlowActionStateStore(t *testing.T) {
	reg := NewRegistry()
	server := httptest.NewServer(Handler(reg, nil, nil, nil, nil, slowActionStateStore{
		loadDelay:   2 * time.Second,
		writeDelay:  2 * time.Second,
		deleteDelay: 2 * time.Second,
	}))
	defer server.Close()

	conn := dialTestWebSocket(t, server.URL)
	defer conn.Close()

	sendNavigatorSetup(t, conn, "device-slow-store")
	_ = readJSONMessageOfType(t, conn, "setupComplete")

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
	if accepted["taskId"] == nil || accepted["taskId"] == "" {
		t.Fatal("missing taskId")
	}
}

func TestHandlerNavigatorProvideDetailsClarificationReplansTextEntry(t *testing.T) {
	reg := NewRegistry()
	server := httptest.NewServer(Handler(reg, nil, nil, nil, nil, NewInMemoryActionStateStore()))
	defer server.Close()

	conn := dialTestWebSocket(t, server.URL)
	defer conn.Close()

	contextPayload := map[string]any{
		"appName":                    "Codex",
		"windowTitle":                "Codex",
		"focusedRole":                "AXTextArea",
		"focusedLabel":               "후속 변경 사항을 부탁하세요",
		"lastInputFieldDescriptor":   "bundle=com.openai.codex|window=Codex|role=textarea|label=후속 변경 사항을 부탁하세요",
		"visibleInputCandidateCount": 1,
		"captureConfidence":          0.93,
		"accessibilityTrusted":       true,
	}

	if err := conn.WriteJSON(map[string]any{
		"type":    "navigator.command",
		"command": "검색창에 입력해줘",
		"context": contextPayload,
	}); err != nil {
		t.Fatalf("send navigator.command: %v", err)
	}

	clarification := readJSONMessageOfType(t, conn, "navigator.intentClarificationNeeded")
	if got := clarification["responseMode"]; got != string(navigatorClarificationProvideDetail) {
		t.Fatalf("responseMode = %v, want %q", got, navigatorClarificationProvideDetail)
	}

	if err := conn.WriteJSON(map[string]any{
		"type":    "navigator.confirmAmbiguousIntent",
		"command": "검색창에 입력해줘",
		"answer":  "A부터 Z 뒤에 한 칸 띄고 하이라고 입력해 줘.",
		"context": contextPayload,
	}); err != nil {
		t.Fatalf("send navigator.confirmAmbiguousIntent: %v", err)
	}

	accepted := readJSONMessageOfType(t, conn, "navigator.commandAccepted")
	if got := accepted["command"]; got != "A부터 Z 뒤에 한 칸 띄고 하이라고 입력해 줘." {
		t.Fatalf("accepted command = %v, want clarified command", got)
	}
	taskID, _ := accepted["taskId"].(string)
	if taskID == "" {
		t.Fatal("missing taskId")
	}
	planned := readJSONMessageOfType(t, conn, "navigator.stepPlanned")
	step, _ := planned["step"].(map[string]any)
	if got := step["actionType"]; got != "press_ax" {
		t.Fatalf("first action = %v, want press_ax", got)
	}
	if err := conn.WriteJSON(map[string]any{
		"type":            "navigator.refreshContext",
		"command":         "A부터 Z 뒤에 한 칸 띄고 하이라고 입력해 줘.",
		"taskId":          taskID,
		"step":            step,
		"status":          "success",
		"observedOutcome": "Focused the target input field",
		"context":         contextPayload,
	}); err != nil {
		t.Fatalf("send navigator.refreshContext: %v", err)
	}
	_ = readJSONMessageOfType(t, conn, "navigator.stepVerified")
	next := readJSONMessageOfType(t, conn, "navigator.stepPlanned")
	nextStep, _ := next["step"].(map[string]any)
	if got := nextStep["actionType"]; got != "paste_text" {
		t.Fatalf("next action = %v, want paste_text", got)
	}
	if got := nextStep["inputText"]; got != "ABCDEFGHIJKLMNOPQRSTUVWXYZ 하이" {
		t.Fatalf("inputText = %v, want clarified compound payload", got)
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

func TestNavigatorSessionStateStepRetryCount(t *testing.T) {
	s := &navigatorSessionState{}
	steps := []navigatorStep{{ID: "s1", ActionType: "focus_app"}}
	s.startPlan("test command", steps)

	if s.stepRetryCount != 0 {
		t.Fatalf("stepRetryCount after startPlan = %d, want 0", s.stepRetryCount)
	}

	n := s.incrementStepRetry()
	if n != 1 || s.stepRetryCount != 1 {
		t.Fatalf("after first increment: n=%d retryCount=%d, want 1", n, s.stepRetryCount)
	}

	n = s.incrementStepRetry()
	if n != 2 || s.stepRetryCount != 2 {
		t.Fatalf("after second increment: n=%d retryCount=%d, want 2", n, s.stepRetryCount)
	}

	s.resetStepRetry()
	if s.stepRetryCount != 0 {
		t.Fatalf("after resetStepRetry: retryCount=%d, want 0", s.stepRetryCount)
	}

	s.clearPlan()
	if s.stepRetryCount != 0 {
		t.Fatalf("after clearPlan: retryCount=%d, want 0", s.stepRetryCount)
	}
}

func TestHandlerNavigatorSelfHealingRetryOnStepFailure(t *testing.T) {
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
			"selectedText":         "AuthServiceTests failing",
			"accessibilityTrusted": true,
		},
	}); err != nil {
		t.Fatalf("send navigator.command: %v", err)
	}

	accepted := readJSONMessageOfType(t, conn, "navigator.commandAccepted")
	taskID, _ := accepted["taskId"].(string)
	planned := readJSONMessageOfType(t, conn, "navigator.stepPlanned")
	step, _ := planned["step"].(map[string]any)

	for retryAttempt := 1; retryAttempt <= 2; retryAttempt++ {
		if err := conn.WriteJSON(map[string]any{
			"type":            "navigator.refreshContext",
			"command":         "공식 문서 쪽으로 가보자",
			"taskId":          taskID,
			"step":            step,
			"status":          "failed",
			"observedOutcome": "focus_failed: Chrome did not come to front",
			"context":         map[string]any{"appName": "Antigravity IDE", "accessibilityTrusted": true},
		}); err != nil {
			t.Fatalf("send refreshContext attempt %d: %v", retryAttempt, err)
		}

		retried := readJSONMessageOfType(t, conn, "navigator.stepPlanned")
		retriedTaskID, _ := retried["taskId"].(string)
		if retriedTaskID != taskID {
			t.Fatalf("retry %d: taskId = %q, want %q", retryAttempt, retriedTaskID, taskID)
		}
	}

	if err := conn.WriteJSON(map[string]any{
		"type":            "navigator.refreshContext",
		"command":         "공식 문서 쪽으로 가보자",
		"taskId":          taskID,
		"step":            step,
		"status":          "failed",
		"observedOutcome": "focus_failed: exhausted retries",
		"context":         map[string]any{"appName": "Antigravity IDE", "accessibilityTrusted": true},
	}); err != nil {
		t.Fatalf("send refreshContext exhausted: %v", err)
	}

	failed := readJSONMessageOfType(t, conn, "navigator.failed")
	failedTaskID, _ := failed["taskId"].(string)
	if failedTaskID != taskID {
		t.Fatalf("failed taskId = %q, want %q", failedTaskID, taskID)
	}
}
