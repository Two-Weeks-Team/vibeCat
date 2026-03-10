package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func orchestratorURL(t *testing.T) string {
	t.Helper()
	u := os.Getenv("ORCHESTRATOR_URL")
	if u == "" {
		t.Skip("ORCHESTRATOR_URL not set — skipping orchestrator real-API tests")
	}
	return strings.TrimRight(u, "/")
}

func gcloudOutput(t *testing.T, args ...string) string {
	t.Helper()
	if _, err := exec.LookPath("gcloud"); err != nil {
		t.Skip("gcloud not installed")
	}
	cmd := exec.Command("gcloud", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gcloud %v failed: %v (%s)", args, err, output)
	}
	return strings.TrimSpace(string(output))
}

func orchestratorIdentityToken(t *testing.T) string {
	t.Helper()
	token := gcloudOutput(t, "auth", "print-identity-token")
	if token == "" {
		t.Fatal("empty gcloud identity token")
	}
	return token
}

func postJSONWithBearer(t *testing.T, url string, bearer string, payload any) (*http.Response, []byte) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s failed: %v", url, err)
	}
	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return resp, respBody
}

func TestSearchEndpointRealAPI(t *testing.T) {
	base := orchestratorURL(t)
	token := orchestratorIdentityToken(t)

	resp, body := postJSONWithBearer(t, base+"/search", token, map[string]any{
		"query":     "Go websocket close code 1006 공식 문서 찾아서 요약해줘",
		"language":  "Korean",
		"userId":    "e2e-search-user",
		"sessionId": "e2e-search-session",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/search expected 200, got %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Query   string   `json:"query"`
		Summary string   `json:"summary"`
		Sources []string `json:"sources"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("decode search response: %v (%s)", err, body)
	}
	if strings.TrimSpace(result.Summary) == "" {
		t.Fatalf("search summary empty: %s", body)
	}
	if len([]rune(result.Summary)) < 20 {
		t.Fatalf("search summary too short: %q", result.Summary)
	}
}

func TestToolRoutingRealAPI(t *testing.T) {
	base := orchestratorURL(t)
	token := orchestratorIdentityToken(t)

	tests := []struct {
		name         string
		query        string
		wantTool     string
		wantURLMeta  bool
		wantCodeMeta bool
	}{
		{
			name:     "search",
			query:    "Go websocket close code 1006 공식 문서 찾아서 요약해줘",
			wantTool: "search",
		},
		{
			name:     "maps",
			query:    "강남역 근처 카페 2곳 추천해줘",
			wantTool: "maps",
		},
		{
			name:        "url context",
			query:       "이 페이지 핵심만 요약해줘 https://ai.google.dev/gemini-api/docs/google-search",
			wantTool:    "url_context",
			wantURLMeta: true,
		},
		{
			name:         "code execution",
			query:        "섭씨 37도를 화씨로 계산해줘",
			wantTool:     "code_execution",
			wantCodeMeta: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, body := postJSONWithBearer(t, base+"/tool", token, map[string]any{
				"query":     tt.query,
				"language":  "Korean",
				"userId":    "e2e-tool-user",
				"sessionId": "e2e-tool-session",
			})
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("/tool expected 200, got %d: %s", resp.StatusCode, body)
			}

			var result struct {
				Tool          string   `json:"tool"`
				Summary       string   `json:"summary"`
				Sources       []string `json:"sources"`
				RetrievedURLs []string `json:"retrievedUrls"`
				GeneratedCode string   `json:"generatedCode"`
				CodeOutput    string   `json:"codeOutput"`
			}
			if err := json.Unmarshal(body, &result); err != nil {
				t.Fatalf("decode /tool response: %v (%s)", err, body)
			}
			if result.Tool != tt.wantTool {
				t.Fatalf("tool = %q, want %q (%s)", result.Tool, tt.wantTool, body)
			}
			if strings.TrimSpace(result.Summary) == "" {
				t.Fatalf("tool summary empty: %s", body)
			}
			if tt.wantURLMeta && len(result.RetrievedURLs) == 0 {
				t.Fatalf("expected retrievedUrls for URL context: %s", body)
			}
			if tt.wantCodeMeta && (strings.TrimSpace(result.GeneratedCode) == "" || strings.TrimSpace(result.CodeOutput) == "") {
				t.Fatalf("expected generatedCode and codeOutput for code execution: %s", body)
			}
		})
	}
}

func TestGatewayToolAutoResponseRealAPI(t *testing.T) {
	base := gatewayURL(t)
	token := registerToken(t, base)

	conn := dialWS(t, base, token)
	defer conn.Close()

	if err := conn.WriteJSON(map[string]any{
		"type": "setup",
		"config": map[string]any{
			"voice":         "Zephyr",
			"language":      "ko",
			"searchEnabled": true,
		},
	}); err != nil {
		t.Fatalf("send setup: %v", err)
	}
	waitForSetupComplete(t, conn)

	if err := conn.WriteJSON(map[string]any{
		"type": "clientContent",
		"clientContent": map[string]any{
			"turnComplete": true,
			"turns": []map[string]any{
				{
					"role": "user",
					"parts": []map[string]string{
						{"text": "이 페이지 핵심만 요약해줘 https://ai.google.dev/gemini-api/docs/google-search"},
					},
				},
			},
		},
	}); err != nil {
		t.Fatalf("send clientContent: %v", err)
	}

	deadline := time.Now().Add(35 * time.Second)
	sawTool := false
	sawProcessingState := false
	sawResponsePreparing := false
	sawAssistantResponse := false

	_ = conn.SetReadDeadline(deadline)
	for time.Now().Before(deadline) {
		msgType, payload, err := conn.ReadMessage()
		if err != nil {
			if ne, ok := err.(interface{ Timeout() bool }); ok && ne.Timeout() {
				break
			}
			t.Fatalf("read websocket message: %v", err)
		}

		if msgType == websocket.BinaryMessage && len(payload) > 0 && sawTool {
			sawAssistantResponse = true
			break
		}
		if msgType != websocket.TextMessage {
			continue
		}

		var msg map[string]any
		if err := json.Unmarshal(payload, &msg); err != nil {
			continue
		}

		switch msg["type"] {
		case "processingState":
			stage, _ := msg["stage"].(string)
			active, _ := msg["active"].(bool)
			if active && stage == "tool_running" {
				sawProcessingState = true
			}
			if active && stage == "response_preparing" {
				sawResponsePreparing = true
			}
		case "toolResult":
			if msg["tool"] == "url_context" {
				sawTool = true
			}
		case "turnState":
			if sawTool && msg["state"] == "speaking" {
				sawAssistantResponse = true
				break
			}
		case "transcription":
			if sawTool {
				if text, _ := msg["text"].(string); strings.TrimSpace(text) != "" {
					sawAssistantResponse = true
					break
				}
			}
		}
	}

	if !sawTool {
		t.Fatal("did not observe toolResult for URL context query")
	}
	if !sawProcessingState {
		t.Fatal("did not observe processingState tool_running before tool result")
	}
	if !sawResponsePreparing {
		t.Fatal("did not observe processingState response_preparing before assistant response")
	}
	if !sawAssistantResponse {
		t.Fatal("toolResult arrived but assistant did not automatically respond")
	}
}

func TestGatewayLiveSearchProcessingStateRealAPI(t *testing.T) {
	base := gatewayURL(t)
	token := registerToken(t, base)

	conn := dialWS(t, base, token)
	defer conn.Close()

	if err := conn.WriteJSON(map[string]any{
		"type": "setup",
		"config": map[string]any{
			"voice":         "Zephyr",
			"language":      "ko",
			"searchEnabled": true,
		},
	}); err != nil {
		t.Fatalf("send setup: %v", err)
	}
	waitForSetupComplete(t, conn)

	if err := conn.WriteJSON(map[string]any{
		"type": "clientContent",
		"clientContent": map[string]any{
			"turnComplete": true,
			"turns": []map[string]any{
				{
					"role": "user",
					"parts": []map[string]string{
						{"text": "오늘 서울 날씨 검색해서 알려줘"},
					},
				},
			},
		},
	}); err != nil {
		t.Fatalf("send search clientContent: %v", err)
	}

	deadline := time.Now().Add(35 * time.Second)
	sawSearching := false
	sawGrounding := false
	sawAssistantResponse := false
	sawTurnComplete := false

	_ = conn.SetReadDeadline(deadline)
	for time.Now().Before(deadline) {
		msgType, payload, err := conn.ReadMessage()
		if err != nil {
			if ne, ok := err.(interface{ Timeout() bool }); ok && ne.Timeout() {
				break
			}
			t.Fatalf("read websocket message: %v", err)
		}

		if msgType == websocket.BinaryMessage && len(payload) > 0 && sawSearching {
			sawAssistantResponse = true
			continue
		}
		if msgType != websocket.TextMessage {
			continue
		}

		var msg map[string]any
		if err := json.Unmarshal(payload, &msg); err != nil {
			continue
		}

		switch msg["type"] {
		case "processingState":
			stage, _ := msg["stage"].(string)
			active, _ := msg["active"].(bool)
			if active && stage == "searching" {
				sawSearching = true
			}
			if active && stage == "grounding" {
				sawGrounding = true
			}
		case "turnState":
			if msg["state"] == "speaking" && sawSearching {
				sawAssistantResponse = true
			}
		case "transcription":
			if sawSearching {
				if text, _ := msg["text"].(string); strings.TrimSpace(text) != "" {
					sawAssistantResponse = true
				}
			}
		case "turnComplete":
			sawTurnComplete = true
			if sawSearching && sawGrounding && sawAssistantResponse {
				break
			}
		}
		if sawSearching && sawGrounding && sawAssistantResponse && sawTurnComplete {
			break
		}
	}

	if !sawSearching {
		t.Fatal("did not observe processingState searching for live native search")
	}
	if !sawGrounding {
		t.Fatal("did not observe grounding processingState for live native search")
	}
	if !sawAssistantResponse {
		t.Fatal("search processing states arrived but assistant did not respond")
	}
}

func TestGatewaySessionResumptionRealAPI(t *testing.T) {
	base := gatewayURL(t)
	token := registerToken(t, base)

	conn := dialWS(t, base, token)
	if err := conn.WriteJSON(map[string]any{
		"type": "setup",
		"config": map[string]any{
			"voice":    "Zephyr",
			"language": "ko",
		},
	}); err != nil {
		t.Fatalf("send initial setup: %v", err)
	}
	waitForSetupComplete(t, conn)

	sendClientTextTurn(t, conn, "안녕, 세션 재개 테스트야.")
	handle := waitForResumptionHandle(t, conn)
	if err := conn.Close(); err != nil {
		t.Fatalf("close initial websocket: %v", err)
	}

	resumed := dialWS(t, base, token)
	defer resumed.Close()

	if err := resumed.WriteJSON(map[string]any{
		"type": "setup",
		"config": map[string]any{
			"voice":            "Zephyr",
			"language":         "ko",
			"resumptionHandle": handle,
		},
	}); err != nil {
		t.Fatalf("send resumed setup: %v", err)
	}
	waitForSetupComplete(t, resumed)

	sendClientTextTurn(t, resumed, "이전 대화가 이어지는지 확인해줘.")
	nextHandle := waitForResumptionHandle(t, resumed)
	if strings.TrimSpace(nextHandle) == "" {
		t.Fatal("session resumed but did not receive a fresh resumption handle")
	}
}

func waitForResumptionHandle(t *testing.T, conn *websocket.Conn) string {
	t.Helper()

	deadline := time.Now().Add(35 * time.Second)
	_ = conn.SetReadDeadline(deadline)
	for time.Now().Before(deadline) {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			if ne, ok := err.(interface{ Timeout() bool }); ok && ne.Timeout() {
				break
			}
			t.Fatalf("read resumption handle: %v", err)
		}

		var msg map[string]any
		if err := json.Unmarshal(payload, &msg); err != nil {
			continue
		}
		if msg["type"] != "sessionResumptionUpdate" {
			continue
		}
		handle, _ := msg["sessionHandle"].(string)
		if strings.TrimSpace(handle) != "" {
			return handle
		}
	}

	t.Fatal("did not receive non-empty sessionResumptionUpdate")
	return ""
}

func sendClientTextTurn(t *testing.T, conn *websocket.Conn, text string) {
	t.Helper()

	if err := conn.WriteJSON(map[string]any{
		"type": "clientContent",
		"clientContent": map[string]any{
			"turnComplete": true,
			"turns": []map[string]any{
				{
					"role": "user",
					"parts": []map[string]string{
						{"text": text},
					},
				},
			},
		},
	}); err != nil {
		t.Fatalf("send clientContent: %v", err)
	}
}

func TestMemorySessionSummaryRoundTripRealAPI(t *testing.T) {
	base := orchestratorURL(t)
	token := orchestratorIdentityToken(t)
	userID := fmt.Sprintf("e2e-memory-%d", time.Now().UnixNano())

	saveResp, saveBody := postJSONWithBearer(t, base+"/memory/session-summary", token, map[string]any{
		"userId":    userID,
		"sessionId": "e2e-memory-session",
		"language":  "Korean",
		"history": []string{
			"user: swift auth error 때문에 로그인 테스트가 깨졌어",
			"tool[search]: auth token 갱신 경로와 401 처리 확인",
			"assistant: auth middleware와 token refresh 순서를 먼저 점검하자",
			"error: auth token refresh failed with 401",
		},
	})
	if saveResp.StatusCode != http.StatusOK {
		t.Fatalf("save session summary expected 200, got %d: %s", saveResp.StatusCode, saveBody)
	}

	ctxResp, ctxBody := postJSONWithBearer(t, base+"/memory/context", token, map[string]any{
		"userId":   userID,
		"language": "Korean",
	})
	if ctxResp.StatusCode != http.StatusOK {
		t.Fatalf("memory context expected 200, got %d: %s", ctxResp.StatusCode, ctxBody)
	}

	var payload struct {
		Context string `json:"context"`
	}
	if err := json.Unmarshal(ctxBody, &payload); err != nil {
		t.Fatalf("decode memory context: %v (%s)", err, ctxBody)
	}
	if strings.TrimSpace(payload.Context) == "" {
		t.Fatalf("memory context empty after save: %s", ctxBody)
	}
	for _, want := range []string{"Recent developer context", "Respond in Korean"} {
		if !strings.Contains(payload.Context, want) {
			t.Fatalf("memory context missing %q: %s", want, payload.Context)
		}
	}
}
