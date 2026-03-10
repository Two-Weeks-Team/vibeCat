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
						{"text": "Go websocket close code 1006 공식 문서 찾아서 요약해줘"},
					},
				},
			},
		},
	}); err != nil {
		t.Fatalf("send clientContent: %v", err)
	}

	deadline := time.Now().Add(35 * time.Second)
	sawTool := false
	sawAssistantResponse := false

	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(time.Now().Add(12 * time.Second))
		msgType, payload, err := conn.ReadMessage()
		if err != nil {
			if ne, ok := err.(interface{ Timeout() bool }); ok && ne.Timeout() {
				continue
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
		case "toolResult":
			if msg["tool"] == "search" {
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
	if !sawAssistantResponse {
		t.Fatal("toolResult arrived but assistant did not automatically respond")
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
	for _, want := range []string{"Previous session", "Respond in Korean"} {
		if !strings.Contains(payload.Context, want) {
			t.Fatalf("memory context missing %q: %s", want, payload.Context)
		}
	}
}
