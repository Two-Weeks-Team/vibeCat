package ws

import (
	"encoding/json"
	"strings"
	"testing"

	"google.golang.org/genai"
	"vibecat/realtime-gateway/internal/adk"
	"vibecat/realtime-gateway/internal/live"
)

func TestCouldBeQuestion(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{"오늘 뉴스 한번 검색해줄래?", true},
		{"뉴스 검색해줘", true},
		{"최신 뉴스 알려줘?", true},
		{"날씨 어때?", true},
		{"오늘 날씨 알아봐줘", true},
		{"환율 찾아봐", true},
		{"search for today's news", true},
		{"오늘 뉴스 뭐야?", true},
		{"삼성전자 시가총액이 얼마야?", true},
		{"what time is it?", true},
		{"how does this work?", true},

		{"네", false},
		{"응", false},
		{"ㅋㅋ", false},
		{"hi", false},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got := couldBeQuestion(tt.text)
			if got != tt.want {
				t.Errorf("couldBeQuestion(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestBuildProactivePromptIncludesGrounding(t *testing.T) {
	result := &adk.AnalysisResult{
		Vision: &adk.VisionAnalysis{
			Content:      "Xcode shows a failing unit test in AuthServiceTests",
			Emotion:      "concerned",
			ErrorMessage: "Expected 200 but got 401",
		},
		Decision: &adk.MediatorDecision{
			Reason:  "significant_event",
			Urgency: "high",
		},
		SpeechText: "인증 토큰이 만료됐는지 먼저 확인해보자.",
	}

	prompt := buildProactivePrompt(live.Config{Language: "ko"}, result)

	for _, want := range []string{
		"significant_event",
		"high",
		"concerned",
		"Expected 200 but got 401",
		"인증 토큰이 만료됐는지 먼저 확인해보자.",
		"Respond in ko",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q\n%s", want, prompt)
		}
	}
}

func TestBuildSearchPromptIncludesQueryAndSummary(t *testing.T) {
	prompt := buildSearchPrompt(
		live.Config{Language: "English"},
		"why does websocket close with 1006?",
		"Close code 1006 is abnormal closure and often means the connection dropped before a close frame was received.",
	)

	for _, want := range []string{
		"why does websocket close with 1006?",
		"abnormal closure",
		"Respond in English",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q\n%s", want, prompt)
		}
	}
}

func TestLegacyClientContentFallsBackToType(t *testing.T) {
	payload := []byte(`{"clientContent":{"turnComplete":true,"turns":[{"role":"user","parts":[{"text":"hello"}]}]}}`)

	var msg message
	if err := json.Unmarshal(payload, &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg.Type == "" && len(msg.ClientContent) > 0 {
		msg.Type = "clientContent"
	}

	if msg.Type != "clientContent" {
		t.Fatalf("expected clientContent fallback, got %q", msg.Type)
	}
}

func TestDescribeGroundingMetadata(t *testing.T) {
	meta := &genai.GroundingMetadata{
		WebSearchQueries: []string{"gemini live api best practices"},
		GroundingChunks: []*genai.GroundingChunk{
			{Web: &genai.GroundingChunkWeb{URI: "https://ai.google.dev/gemini-api/docs/live-api/best-practices"}},
			{Web: &genai.GroundingChunkWeb{URI: "https://ai.google.dev/gemini-api/docs/live-api/session-management"}},
			{Web: &genai.GroundingChunkWeb{URI: "https://ai.google.dev/gemini-api/docs/live-api/session-management"}},
		},
		RetrievalMetadata: &genai.RetrievalMetadata{
			GoogleSearchDynamicRetrievalScore: 0.82,
		},
	}

	got := describeGroundingMetadata(meta)
	for _, want := range []string{
		"queries=gemini live api best practices",
		"sources=2",
		"retrieval_score=0.82",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("describeGroundingMetadata() missing %q: %s", want, got)
		}
	}
}
