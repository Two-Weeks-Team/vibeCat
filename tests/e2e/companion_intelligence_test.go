package e2e

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestCompanionIntelligenceFlowRealAPI(t *testing.T) {
	base := orchestratorURL(t)
	token := orchestratorIdentityToken(t)

	userID := fmt.Sprintf("e2e-companion-%d", time.Now().UnixNano())
	sessionID := fmt.Sprintf("session-%d", time.Now().UnixNano())

	resp, body := postJSONWithBearer(t, base+"/memory/context", token, map[string]any{
		"userId":   userID,
		"language": "Korean",
	})
	if resp.StatusCode != 200 {
		t.Fatalf("/memory/context initial expected 200, got %d: %s", resp.StatusCode, body)
	}

	var initialMemory struct {
		Context string `json:"context"`
	}
	if err := json.Unmarshal(body, &initialMemory); err != nil {
		t.Fatalf("decode initial memory context: %v (%s)", err, body)
	}

	resp, body = postJSONWithBearer(t, base+"/analyze", token, map[string]any{
		"userId":          userID,
		"sessionId":       sessionID,
		"language":        "Korean",
		"traceId":         "e2e_companion_frustrated",
		"character":       "cat",
		"soul":            "You are a pragmatic coding companion cat.",
		"activityMinutes": 12,
		"context": "Xcode has been showing the same failing AuthServiceTests error for several minutes. " +
			"The terminal keeps printing Expected status 200 but got 401 and the developer is stuck and sighing.",
		"image": "ZmFrZV9pbWFnZQ==",
	})
	if resp.StatusCode != 200 {
		t.Fatalf("/analyze frustrated expected 200, got %d: %s", resp.StatusCode, body)
	}

	var frustrated struct {
		Mood struct {
			Mood string `json:"mood"`
		} `json:"mood"`
		Decision struct {
			ShouldSpeak bool `json:"shouldSpeak"`
			Reason      string `json:"reason"`
		} `json:"decision"`
		Search struct {
			Query   string   `json:"query"`
			Summary string   `json:"summary"`
			Sources []string `json:"sources"`
		} `json:"search"`
		SpeechText string `json:"speechText"`
	}
	if err := json.Unmarshal(body, &frustrated); err != nil {
		t.Fatalf("decode frustrated analyze response: %v (%s)", err, body)
	}
	if strings.TrimSpace(frustrated.Mood.Mood) == "" {
		t.Fatalf("expected mood signal from frustrated analyze: %s", body)
	}
	if strings.TrimSpace(frustrated.SpeechText) == "" && !frustrated.Decision.ShouldSpeak {
		t.Fatalf("expected frustrated analyze to produce speech or a speak decision: %s", body)
	}
	if strings.TrimSpace(frustrated.Search.Summary) == "" && frustrated.Decision.Reason != "search_result" {
		t.Fatalf("expected frustrated analyze to include a grounded search follow-up: %s", body)
	}

	resp, body = postJSONWithBearer(t, base+"/search", token, map[string]any{
		"query":     "Go websocket close code 1006 공식 문서 찾아서 해결 요약해줘",
		"language":  "Korean",
		"traceId":   "e2e_companion_search",
		"userId":    userID,
		"sessionId": sessionID,
	})
	if resp.StatusCode != 200 {
		t.Fatalf("/search expected 200, got %d: %s", resp.StatusCode, body)
	}

	var searchResult struct {
		Query   string   `json:"query"`
		Summary string   `json:"summary"`
		Sources []string `json:"sources"`
	}
	if err := json.Unmarshal(body, &searchResult); err != nil {
		t.Fatalf("decode search response: %v (%s)", err, body)
	}
	if strings.TrimSpace(searchResult.Summary) == "" || len(searchResult.Sources) == 0 {
		t.Fatalf("expected grounded search result with sources: %s", body)
	}

	resp, body = postJSONWithBearer(t, base+"/analyze", token, map[string]any{
		"userId":          userID,
		"sessionId":       sessionID,
		"language":        "Korean",
		"traceId":         "e2e_companion_success",
		"character":       "cat",
		"soul":            "You are a pragmatic coding companion cat.",
		"activityMinutes": 18,
		"context": "The developer fixed the auth bug. Xcode now shows all tests passing, terminal says BUILD SUCCEEDED, " +
			"and the developer just shouted yes after the green test run.",
		"image": "ZmFrZV9pbWFnZQ==",
	})
	if resp.StatusCode != 200 {
		t.Fatalf("/analyze success expected 200, got %d: %s", resp.StatusCode, body)
	}

	var success struct {
		Mood struct {
			Mood string `json:"mood"`
		} `json:"mood"`
		Vision struct {
			SuccessDetected bool `json:"successDetected"`
		} `json:"vision"`
		Celebration struct {
			Message string `json:"message"`
		} `json:"celebration"`
		SpeechText string `json:"speechText"`
	}
	if err := json.Unmarshal(body, &success); err != nil {
		t.Fatalf("decode success analyze response: %v (%s)", err, body)
	}
	if strings.TrimSpace(success.Mood.Mood) == "" {
		t.Fatalf("expected success analyze to return mood state: %s", body)
	}
	if success.Mood.Mood == frustrated.Mood.Mood {
		t.Fatalf("expected mood transition across the same session: frustrated=%q success=%q", frustrated.Mood.Mood, success.Mood.Mood)
	}
	if !success.Vision.SuccessDetected && strings.TrimSpace(success.Celebration.Message) == "" {
		t.Fatalf("expected success analyze to detect celebration: %s", body)
	}

	resp, body = postJSONWithBearer(t, base+"/memory/session-summary", token, map[string]any{
		"userId":    userID,
		"sessionId": sessionID,
		"language":  "Korean",
		"history": []string{
			"user: 인증 테스트가 계속 실패해",
			"assistant: 401 응답 경로를 먼저 확인해보자",
			"tool[search]: " + searchResult.Summary,
			"user: 수정했고 이제 테스트가 전부 통과했어",
			"assistant: 좋았어, 인증 흐름이 정상으로 돌아왔어",
		},
	})
	if resp.StatusCode != 200 {
		t.Fatalf("/memory/session-summary expected 200, got %d: %s", resp.StatusCode, body)
	}

	resp, body = postJSONWithBearer(t, base+"/memory/context", token, map[string]any{
		"userId":   userID,
		"language": "Korean",
	})
	if resp.StatusCode != 200 {
		t.Fatalf("/memory/context final expected 200, got %d: %s", resp.StatusCode, body)
	}

	var finalMemory struct {
		Context string `json:"context"`
	}
	if err := json.Unmarshal(body, &finalMemory); err != nil {
		t.Fatalf("decode final memory context: %v (%s)", err, body)
	}
	if strings.TrimSpace(finalMemory.Context) == "" {
		t.Fatalf("expected saved memory context to be non-empty after summary: %s", body)
	}
	if finalMemory.Context == initialMemory.Context {
		t.Fatalf("expected memory context to change after session summary save: initial=%q final=%q", initialMemory.Context, finalMemory.Context)
	}
}
