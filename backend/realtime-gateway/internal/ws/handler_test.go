package ws

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

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

func TestMemoryContextCacheRoundTrip(t *testing.T) {
	userID := "cache-user"
	language := "Korean"
	invalidateCachedMemoryContext(userID, language)

	if _, ok := getCachedMemoryContext(userID, language); ok {
		t.Fatal("expected empty cache")
	}

	putCachedMemoryContext(userID, language, "Recent developer context")

	got, ok := getCachedMemoryContext(userID, language)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got != "Recent developer context" {
		t.Fatalf("cache = %q", got)
	}

	invalidateCachedMemoryContext(userID, language)
	if _, ok := getCachedMemoryContext(userID, language); ok {
		t.Fatal("expected cache miss after invalidation")
	}
}

func TestMemoryContextCacheExpires(t *testing.T) {
	userID := "cache-expiry-user"
	language := "Korean"
	key := memoryContextCacheKey(userID, language)

	memoryContextCache.mu.Lock()
	memoryContextCache.entries[key] = memoryContextCacheEntry{
		context:   "stale",
		expiresAt: time.Now().Add(-time.Second),
	}
	memoryContextCache.mu.Unlock()

	if _, ok := getCachedMemoryContext(userID, language); ok {
		t.Fatal("expected expired cache miss")
	}
}

func TestResolveQueryRoute(t *testing.T) {
	tests := []struct {
		name  string
		cfg   live.Config
		query string
		want  queryRoute
	}{
		{
			name:  "plain live fallback",
			cfg:   live.Config{GoogleSearch: true},
			query: "이 함수 이름 더 좋게 바꿔줄래",
			want:  queryRoute{Kind: queryRoutePlainLive},
		},
		{
			name:  "live native search",
			cfg:   live.Config{GoogleSearch: true},
			query: "최신 Gemini Live API 문서 찾아줘",
			want:  queryRoute{Kind: queryRouteLiveSearch},
		},
		{
			name:  "url context goes to adk tool",
			cfg:   live.Config{GoogleSearch: true},
			query: "이 페이지 핵심만 요약해줘 https://ai.google.dev/gemini-api/docs/google-search",
			want:  queryRoute{Kind: queryRouteADKTool, Tool: adk.ToolKindURLContext},
		},
		{
			name:  "maps goes to adk tool",
			cfg:   live.Config{GoogleSearch: true},
			query: "강남역 근처 카페 추천해줘",
			want:  queryRoute{Kind: queryRouteADKTool, Tool: adk.ToolKindMaps},
		},
		{
			name:  "code execution goes to adk tool",
			cfg:   live.Config{GoogleSearch: true},
			query: "이 정규식 계산해서 검산해줘",
			want:  queryRoute{Kind: queryRouteADKTool, Tool: adk.ToolKindCodeExecution},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveQueryRoute(tt.cfg, tt.query)
			if got != tt.want {
				t.Fatalf("resolveQueryRoute(%q) = %#v, want %#v", tt.query, got, tt.want)
			}
		})
	}
}
