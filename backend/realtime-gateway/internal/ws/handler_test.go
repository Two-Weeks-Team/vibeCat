package ws

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/genai"
	"vibecat/realtime-gateway/internal/adk"
	"vibecat/realtime-gateway/internal/live"
	"vibecat/realtime-gateway/internal/tts"
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

func TestProactiveContextHintTextUsesWindowAndAppGuidance(t *testing.T) {
	got := proactiveContextHintText("ko", "[app=Xcode target=frontmost_window bundle=com.apple.dt.Xcode window=AuthServiceTests.swift]")

	for _, want := range []string{"Xcode", "테스트"} {
		if !strings.Contains(got, want) {
			t.Fatalf("proactiveContextHintText() missing %q: %s", want, got)
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

func TestHandlerScreenCaptureTimeoutReturnsSilentFallbackQuickly(t *testing.T) {
	reg := NewRegistry()
	fakeADK := &stubADK{
		analyzeFn: func(ctx context.Context, req adk.AnalysisRequest) (*adk.AnalysisResult, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	server := httptest.NewServer(Handler(reg, nil, fakeADK, nil, nil, NewInMemoryActionStateStore()))
	defer server.Close()

	conn := dialTestWebSocket(t, server.URL)
	defer conn.Close()

	start := time.Now()
	if err := conn.WriteJSON(map[string]any{
		"type":            "screenCapture",
		"image":           "ZmFrZV9pbWFnZQ==",
		"context":         "Xcode failing test",
		"sessionId":       "session-timeout",
		"userId":          "user-timeout",
		"character":       "cat",
		"activityMinutes": 3,
	}); err != nil {
		t.Fatalf("send screenCapture: %v", err)
	}

	deadline := time.Now().Add(6 * time.Second)
	if err := conn.SetReadDeadline(deadline); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	var inactiveAt time.Time
	for time.Now().Before(deadline) {
		msgType, payload, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read websocket message: %v", err)
		}
		if msgType != websocket.TextMessage {
			continue
		}
		var msg map[string]any
		if err := json.Unmarshal(payload, &msg); err != nil {
			continue
		}
		if msg["type"] != "processingState" || msg["stage"] != "screen_analyzing" {
			continue
		}
		active, _ := msg["active"].(bool)
		if !active {
			inactiveAt = time.Now()
			break
		}
	}

	if inactiveAt.IsZero() {
		t.Fatal("did not observe screen_analyzing inactive message before deadline")
	}
	if elapsed := inactiveAt.Sub(start); elapsed >= 5*time.Second {
		t.Fatalf("silent fallback completed in %v, want < 5s", elapsed)
	}
}

func TestHandlerForceCaptureSpeaksContextHintBeforeAnalyzeCompletes(t *testing.T) {
	originalDelay := proactiveContextHintDelay
	proactiveContextHintDelay = 20 * time.Millisecond
	t.Cleanup(func() {
		proactiveContextHintDelay = originalDelay
	})

	reg := NewRegistry()
	analyzeStarted := make(chan struct{}, 1)
	analyzeDone := make(chan struct{})
	hintTextCh := make(chan string, 1)
	audioCh := make(chan struct{}, 1)
	var releaseAnalyze sync.Once
	t.Cleanup(func() {
		releaseAnalyze.Do(func() {
			close(analyzeDone)
		})
	})
	fakeADK := &stubADK{
		analyzeFn: func(ctx context.Context, req adk.AnalysisRequest) (*adk.AnalysisResult, error) {
			analyzeStarted <- struct{}{}
			select {
			case <-analyzeDone:
				return &adk.AnalysisResult{
					Vision: &adk.VisionAnalysis{Content: "AuthServiceTests.swift in Xcode"},
				}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	}
	fakeTTS := &stubTTS{
		streamSpeakFn: func(ctx context.Context, cfg tts.Config, sink tts.AudioSink) error {
			select {
			case hintTextCh <- cfg.Text:
			default:
			}
			if err := sink([]byte{0x01, 0x02, 0x03}); err != nil {
				return err
			}
			select {
			case audioCh <- struct{}{}:
			default:
			}
			return nil
		},
	}

	server := httptest.NewServer(Handler(reg, nil, fakeADK, fakeTTS, nil, NewInMemoryActionStateStore()))
	defer server.Close()

	conn := dialTestWebSocket(t, server.URL)
	defer conn.Close()

	if err := conn.WriteJSON(map[string]any{
		"type": "setup",
		"config": map[string]any{
			"voice":          "Zephyr",
			"language":       "ko",
			"proactiveAudio": true,
		},
	}); err != nil {
		t.Fatalf("send setup: %v", err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set setup read deadline: %v", err)
	}
	if _, payload, err := conn.ReadMessage(); err != nil {
		t.Fatalf("read setupComplete: %v", err)
	} else {
		var msg map[string]any
		if err := json.Unmarshal(payload, &msg); err != nil || msg["type"] != "setupComplete" {
			t.Fatalf("expected setupComplete, got %s", payload)
		}
	}

	if err := conn.WriteJSON(map[string]any{
		"type":            "forceCapture",
		"image":           "ZmFrZV9pbWFnZQ==",
		"context":         "[app=Xcode target=frontmost_window bundle=com.apple.dt.Xcode window=AuthServiceTests.swift]",
		"sessionId":       "session-force",
		"userId":          "user-force",
		"character":       "cat",
		"activityMinutes": 7,
	}); err != nil {
		t.Fatalf("send forceCapture: %v", err)
	}

	select {
	case <-analyzeStarted:
	case <-time.After(time.Second):
		t.Fatal("analyze did not start")
	}

	drainDone := make(chan struct{})
	go func() {
		defer close(drainDone)
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	var hintText string
	select {
	case hintText = <-hintTextCh:
		select {
		case <-analyzeDone:
			t.Fatal("analyze completed before proactive context hint fired")
		default:
		}
	case <-time.After(2 * time.Second):
		t.Fatal("did not observe proactive context hint before analyze completion")
	}
	if !strings.Contains(hintText, "Xcode") {
		t.Fatalf("hint text = %q, want Xcode-specific guidance", hintText)
	}
	select {
	case <-audioCh:
		select {
		case <-analyzeDone:
			t.Fatal("analyze completed before proactive context hint audio arrived")
		default:
		}
	case <-time.After(2 * time.Second):
		t.Fatal("did not observe context hint audio chunk")
	}
	releaseAnalyze.Do(func() {
		close(analyzeDone)
	})
	<-drainDone
}

func TestHandlerLiveSearchFallbackSpeaksViaTTSWithoutLiveSession(t *testing.T) {
	installTestTracerProvider(t)

	reg := NewRegistry()
	searchSpanCh := make(chan bool, 1)
	fakeADK := &stubADK{
		searchFn: func(ctx context.Context, req adk.SearchRequest) (*adk.SearchResult, error) {
			searchSpanCh <- trace.SpanFromContext(ctx).SpanContext().IsValid()
			return &adk.SearchResult{
				Query:   req.Query,
				Summary: "공식 문서 기준으로 1006은 close frame 없이 연결이 끊긴 경우를 뜻해.",
				Sources: []string{"https://example.com/close-1006"},
			}, nil
		},
	}
	fakeTTS := &stubTTS{
		streamSpeakFn: func(ctx context.Context, cfg tts.Config, sink tts.AudioSink) error {
			if err := sink([]byte{0x01, 0x02, 0x03}); err != nil {
				return err
			}
			return nil
		},
	}

	server := httptest.NewServer(Handler(reg, nil, fakeADK, fakeTTS, nil, NewInMemoryActionStateStore()))
	defer server.Close()

	conn := dialTestWebSocket(t, server.URL)
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
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set setup read deadline: %v", err)
	}
	if _, payload, err := conn.ReadMessage(); err != nil {
		t.Fatalf("read setupComplete: %v", err)
	} else {
		var msg map[string]any
		if err := json.Unmarshal(payload, &msg); err != nil || msg["type"] != "setupComplete" {
			t.Fatalf("expected setupComplete, got %s", payload)
		}
	}

	if err := conn.WriteJSON(map[string]any{
		"type": "clientContent",
		"clientContent": map[string]any{
			"turnComplete": true,
			"turns": []map[string]any{
				{
					"role": "user",
					"parts": []map[string]string{
						{"text": "웹소켓 1006 오류 공식 문서 찾아서 해결 요약해줘"},
					},
				},
			},
		},
	}); err != nil {
		t.Fatalf("send clientContent: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	if err := conn.SetReadDeadline(deadline); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	sawSearching := false
	sawToolResult := false
	sawTTSStart := false
	sawAudioChunk := false
	sawTTSEnd := false

	for time.Now().Before(deadline) {
		msgType, payload, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read websocket message: %v", err)
		}

		if msgType == websocket.BinaryMessage && len(payload) > 0 {
			sawAudioChunk = true
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
			if stage == "searching" && active {
				sawSearching = true
			}
		case "toolResult":
			if msg["tool"] == string(adk.ToolKindSearch) {
				sawToolResult = true
			}
		case "ttsStart":
			text, _ := msg["text"].(string)
			if strings.TrimSpace(text) != "" {
				sawTTSStart = true
			}
		case "ttsEnd":
			sawTTSEnd = true
		}
		if sawSearching && sawToolResult && sawTTSStart && sawAudioChunk && sawTTSEnd {
			break
		}
	}

	if !sawSearching {
		t.Fatal("did not observe searching processing state")
	}
	if !sawToolResult {
		t.Fatal("did not observe fallback search tool result")
	}
	if !sawTTSStart {
		t.Fatal("did not observe ttsStart for fallback speech")
	}
	if !sawAudioChunk {
		t.Fatal("did not observe fallback audio chunk")
	}
	if !sawTTSEnd {
		t.Fatal("did not observe ttsEnd for fallback speech")
	}
	select {
	case sawSpan := <-searchSpanCh:
		if !sawSpan {
			t.Fatal("expected search fallback to run under an active trace span")
		}
	default:
		t.Fatal("expected search fallback to invoke ADK search")
	}
}

func TestFetchMemoryContextStartsTraceSpan(t *testing.T) {
	installTestTracerProvider(t)

	userID := "trace-memory-user"
	language := "Korean"
	invalidateCachedMemoryContext(userID, language)
	defer invalidateCachedMemoryContext(userID, language)

	sawSpan := false
	fakeADK := &stubADK{
		memoryContextFn: func(ctx context.Context, req adk.MemoryContextRequest) (string, error) {
			sawSpan = trace.SpanFromContext(ctx).SpanContext().IsValid()
			return "Recent coding context", nil
		},
	}

	got := fetchMemoryContext(context.Background(), fakeADK, live.Config{
		DeviceID: userID,
		Language: language,
	})

	if got != "Recent coding context" {
		t.Fatalf("fetchMemoryContext() = %q", got)
	}
	if !sawSpan {
		t.Fatal("expected memory context lookup to run under an active trace span")
	}
}

func TestHandlerToolRoutingStartsTraceSpan(t *testing.T) {
	installTestTracerProvider(t)

	reg := NewRegistry()
	toolSpanCh := make(chan bool, 1)
	fakeADK := &stubADK{
		toolFn: func(ctx context.Context, req adk.ToolRequest) (*adk.ToolResult, error) {
			toolSpanCh <- trace.SpanFromContext(ctx).SpanContext().IsValid()
			return &adk.ToolResult{
				Tool:    adk.ToolKindMaps,
				Query:   req.Query,
				Summary: "강남역 근처 카페 두 곳을 찾았어.",
				Sources: []string{"https://example.com/maps"},
			}, nil
		},
	}

	server := httptest.NewServer(Handler(reg, nil, fakeADK, nil, nil, NewInMemoryActionStateStore()))
	defer server.Close()

	conn := dialTestWebSocket(t, server.URL)
	defer conn.Close()

	if err := conn.WriteJSON(map[string]any{
		"type": "clientContent",
		"clientContent": map[string]any{
			"turnComplete": true,
			"turns": []map[string]any{
				{
					"role": "user",
					"parts": []map[string]string{
						{"text": "강남역 근처 카페 추천해줘"},
					},
				},
			},
		},
	}); err != nil {
		t.Fatalf("send clientContent: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	if err := conn.SetReadDeadline(deadline); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}

	sawToolResult := false
	for time.Now().Before(deadline) {
		msgType, payload, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read websocket message: %v", err)
		}
		if msgType != websocket.TextMessage {
			continue
		}

		var msg map[string]any
		if err := json.Unmarshal(payload, &msg); err != nil {
			continue
		}
		if msg["type"] == "toolResult" {
			sawToolResult = true
			break
		}
	}

	if !sawToolResult {
		t.Fatal("did not observe toolResult for routed tool request")
	}
	select {
	case sawSpan := <-toolSpanCh:
		if !sawSpan {
			t.Fatal("expected tool routing to run under an active trace span")
		}
	default:
		t.Fatal("expected tool routing to invoke ADK tool")
	}
}

func TestSaveSessionMemoryStartsTraceSpan(t *testing.T) {
	installTestTracerProvider(t)

	sawSpan := false
	fakeADK := &stubADK{
		saveSessionSummaryFn: func(ctx context.Context, req adk.SessionSummaryRequest) error {
			sawSpan = trace.SpanFromContext(ctx).SpanContext().IsValid()
			return nil
		},
	}
	runtime := newSessionRuntime("trace-save-user", "trace-save-session")
	runtime.append("user: 인증 테스트가 계속 실패해")
	runtime.append("assistant: 401 응답 경로를 먼저 확인해보자")

	saveSessionMemory(context.Background(), fakeADK, live.Config{Language: "Korean"}, runtime)

	if !sawSpan {
		t.Fatal("expected session summary save to run under an active trace span")
	}
}

func installTestTracerProvider(t *testing.T) {
	t.Helper()

	previousProvider := otel.GetTracerProvider()
	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
	})

	tp := sdktrace.NewTracerProvider()
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
	})
	otel.SetTracerProvider(tp)
}

type stubADK struct {
	analyzeFn            func(context.Context, adk.AnalysisRequest) (*adk.AnalysisResult, error)
	searchFn             func(context.Context, adk.SearchRequest) (*adk.SearchResult, error)
	toolFn               func(context.Context, adk.ToolRequest) (*adk.ToolResult, error)
	saveSessionSummaryFn func(context.Context, adk.SessionSummaryRequest) error
	navigatorEscalateFn  func(context.Context, adk.NavigatorEscalationRequest) (*adk.NavigatorEscalationResult, error)
	navigatorBackgroundFn func(context.Context, adk.NavigatorBackgroundRequest) (*adk.NavigatorBackgroundResult, error)
	memoryContextFn      func(context.Context, adk.MemoryContextRequest) (string, error)
}

func (s *stubADK) Analyze(ctx context.Context, req adk.AnalysisRequest) (*adk.AnalysisResult, error) {
	if s.analyzeFn != nil {
		return s.analyzeFn(ctx, req)
	}
	return nil, errors.New("Analyze not implemented")
}

func (s *stubADK) Search(ctx context.Context, req adk.SearchRequest) (*adk.SearchResult, error) {
	if s.searchFn != nil {
		return s.searchFn(ctx, req)
	}
	return nil, errors.New("Search not implemented")
}

func (s *stubADK) Tool(ctx context.Context, req adk.ToolRequest) (*adk.ToolResult, error) {
	if s.toolFn != nil {
		return s.toolFn(ctx, req)
	}
	return nil, errors.New("Tool not implemented")
}

func (s *stubADK) SaveSessionSummary(ctx context.Context, req adk.SessionSummaryRequest) error {
	if s.saveSessionSummaryFn != nil {
		return s.saveSessionSummaryFn(ctx, req)
	}
	return nil
}

func (s *stubADK) NavigatorEscalate(ctx context.Context, req adk.NavigatorEscalationRequest) (*adk.NavigatorEscalationResult, error) {
	if s.navigatorEscalateFn != nil {
		return s.navigatorEscalateFn(ctx, req)
	}
	return nil, nil
}

func (s *stubADK) NavigatorBackground(ctx context.Context, req adk.NavigatorBackgroundRequest) (*adk.NavigatorBackgroundResult, error) {
	if s.navigatorBackgroundFn != nil {
		return s.navigatorBackgroundFn(ctx, req)
	}
	return nil, nil
}

func (s *stubADK) MemoryContext(ctx context.Context, req adk.MemoryContextRequest) (string, error) {
	if s.memoryContextFn != nil {
		return s.memoryContextFn(ctx, req)
	}
	return "", nil
}

type stubTTS struct {
	streamSpeakFn func(context.Context, tts.Config, tts.AudioSink) error
}

func (s *stubTTS) StreamSpeak(ctx context.Context, cfg tts.Config, sink tts.AudioSink) error {
	if s.streamSpeakFn != nil {
		return s.streamSpeakFn(ctx, cfg, sink)
	}
	return nil
}

func dialTestWebSocket(t *testing.T, baseURL string) *websocket.Conn {
	t.Helper()
	parsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("parse base url: %v", err)
	}
	switch parsed.Scheme {
	case "https":
		parsed.Scheme = "wss"
	default:
		parsed.Scheme = "ws"
	}

	conn, _, err := websocket.DefaultDialer.Dial(parsed.String(), nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	return conn
}
