package adk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"golang.org/x/oauth2"
	"google.golang.org/api/idtoken"
)

const defaultTimeout = 30 * time.Second

type AnalysisRequest struct {
	Image           string `json:"image"`
	Context         string `json:"context"`
	Language        string `json:"language,omitempty"`
	AppName         string `json:"appName,omitempty"`
	SessionID       string `json:"sessionId,omitempty"`
	UserID          string `json:"userId,omitempty"`
	Character       string `json:"character,omitempty"`
	Soul            string `json:"soul,omitempty"`
	ActivityMinutes int    `json:"activityMinutes,omitempty"`
	TraceID         string `json:"traceId,omitempty"`
}

type VisionAnalysis struct {
	Significance    int    `json:"significance"`
	Content         string `json:"content"`
	Emotion         string `json:"emotion"`
	ShouldSpeak     bool   `json:"shouldSpeak"`
	ErrorDetected   bool   `json:"errorDetected"`
	RepeatedError   bool   `json:"repeatedError"`
	SuccessDetected bool   `json:"successDetected"`
	ErrorMessage    string `json:"errorMessage,omitempty"`
}

type MediatorDecision struct {
	ShouldSpeak bool   `json:"shouldSpeak"`
	Reason      string `json:"reason"`
	Urgency     string `json:"urgency"`
}

type MoodState struct {
	Mood            string  `json:"mood"`
	Confidence      float64 `json:"confidence"`
	SuggestedAction string  `json:"suggestedAction"`
}

type CelebrationEvent struct {
	TriggerType string `json:"triggerType"`
	Emotion     string `json:"emotion"`
	Message     string `json:"message"`
}

type SearchResult struct {
	Query   string   `json:"query"`
	Summary string   `json:"summary"`
	Sources []string `json:"sources,omitempty"`
}

type AnalysisResult struct {
	Vision      *VisionAnalysis   `json:"vision,omitempty"`
	Decision    *MediatorDecision `json:"decision,omitempty"`
	Mood        *MoodState        `json:"mood,omitempty"`
	Celebration *CelebrationEvent `json:"celebration,omitempty"`
	Search      *SearchResult     `json:"search,omitempty"`
	SpeechText  string            `json:"speechText,omitempty"`
}

type SearchRequest struct {
	Query    string `json:"query"`
	Language string `json:"language,omitempty"`
	TraceID  string `json:"traceId,omitempty"`
}

type ToolKind string

const (
	ToolKindNone          ToolKind = "none"
	ToolKindSearch        ToolKind = "search"
	ToolKindMaps          ToolKind = "maps"
	ToolKindURLContext    ToolKind = "url_context"
	ToolKindCodeExecution ToolKind = "code_execution"
	ToolKindFileSearch    ToolKind = "file_search"
)

type ToolRequest struct {
	Query     string `json:"query"`
	Language  string `json:"language,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	UserID    string `json:"userId,omitempty"`
	TraceID   string `json:"traceId,omitempty"`
}

type ToolResult struct {
	Tool          ToolKind `json:"tool"`
	Query         string   `json:"query"`
	Summary       string   `json:"summary"`
	Sources       []string `json:"sources,omitempty"`
	RetrievedURLs []string `json:"retrievedUrls,omitempty"`
	GeneratedCode string   `json:"generatedCode,omitempty"`
	CodeOutput    string   `json:"codeOutput,omitempty"`
	Reason        string   `json:"reason,omitempty"`
}

type NavigatorTargetDescriptor struct {
	Role           string `json:"role,omitempty"`
	Label          string `json:"label,omitempty"`
	WindowTitle    string `json:"windowTitle,omitempty"`
	AppName        string `json:"appName,omitempty"`
	RelativeAnchor string `json:"relativeAnchor,omitempty"`
	RegionHint     string `json:"regionHint,omitempty"`
}

type NavigatorEscalationRequest struct {
	Command                    string  `json:"command"`
	Language                   string  `json:"language,omitempty"`
	AppName                    string  `json:"appName,omitempty"`
	BundleID                   string  `json:"bundleId,omitempty"`
	FrontmostBundleID          string  `json:"frontmostBundleId,omitempty"`
	WindowTitle                string  `json:"windowTitle,omitempty"`
	FocusedRole                string  `json:"focusedRole,omitempty"`
	FocusedLabel               string  `json:"focusedLabel,omitempty"`
	SelectedText               string  `json:"selectedText,omitempty"`
	AXSnapshot                 string  `json:"axSnapshot,omitempty"`
	LastInputFieldDescriptor   string  `json:"lastInputFieldDescriptor,omitempty"`
	Screenshot                 string  `json:"screenshot,omitempty"`
	CaptureConfidence          float64 `json:"captureConfidence,omitempty"`
	VisibleInputCandidateCount int     `json:"visibleInputCandidateCount,omitempty"`
	TraceID                    string  `json:"traceId,omitempty"`
}

type NavigatorEscalationResult struct {
	ResolvedDescriptor     *NavigatorTargetDescriptor `json:"resolvedDescriptor,omitempty"`
	ResolvedText           string                     `json:"resolvedText,omitempty"`
	Confidence             float64                    `json:"confidence"`
	FallbackRecommendation string                     `json:"fallbackRecommendation,omitempty"`
	Reason                 string                     `json:"reason,omitempty"`
}

type NavigatorBackgroundStep struct {
	ID               string                    `json:"id"`
	ActionType       string                    `json:"actionType"`
	TargetApp        string                    `json:"targetApp,omitempty"`
	TargetDescriptor NavigatorTargetDescriptor `json:"targetDescriptor,omitempty"`
	ResultStatus     string                    `json:"resultStatus,omitempty"`
	ObservedOutcome  string                    `json:"observedOutcome,omitempty"`
	PlannedAt        time.Time                 `json:"plannedAt,omitempty"`
	CompletedAt      time.Time                 `json:"completedAt,omitempty"`
}

type NavigatorBackgroundAttempt struct {
	ID               string    `json:"id"`
	TaskID           string    `json:"taskId,omitempty"`
	Command          string    `json:"command"`
	Surface          string    `json:"surface,omitempty"`
	Route            string    `json:"route"`
	RouteReason      string    `json:"routeReason,omitempty"`
	ContextHash      string    `json:"contextHash,omitempty"`
	ScreenshotSource string    `json:"screenshotSource,omitempty"`
	ScreenshotCached bool      `json:"screenshotCached,omitempty"`
	ScreenBasisID    string    `json:"screenBasisId,omitempty"`
	ActiveDisplayID  string    `json:"activeDisplayId,omitempty"`
	TargetDisplayID  string    `json:"targetDisplayId,omitempty"`
	Outcome          string    `json:"outcome,omitempty"`
	OutcomeDetail    string    `json:"outcomeDetail,omitempty"`
	StartedAt        time.Time `json:"startedAt,omitzero"`
	CompletedAt      time.Time `json:"completedAt,omitzero"`
}

type NavigatorBackgroundRequest struct {
	UserID                  string                       `json:"userId,omitempty"`
	SessionID               string                       `json:"sessionId,omitempty"`
	TaskID                  string                       `json:"taskId"`
	Command                 string                       `json:"command"`
	Language                string                       `json:"language,omitempty"`
	Outcome                 string                       `json:"outcome"`
	OutcomeDetail           string                       `json:"outcomeDetail,omitempty"`
	Surface                 string                       `json:"surface,omitempty"`
	InitialAppName          string                       `json:"initialAppName,omitempty"`
	InitialWindowTitle      string                       `json:"initialWindowTitle,omitempty"`
	InitialContextHash      string                       `json:"initialContextHash,omitempty"`
	LastVerifiedContextHash string                       `json:"lastVerifiedContextHash,omitempty"`
	StartedAt               time.Time                    `json:"startedAt,omitzero"`
	CompletedAt             time.Time                    `json:"completedAt,omitzero"`
	Steps                   []NavigatorBackgroundStep    `json:"steps,omitempty"`
	Attempts                []NavigatorBackgroundAttempt `json:"attempts,omitempty"`
	TraceID                 string                       `json:"traceId,omitempty"`
}

type NavigatorBackgroundResult struct {
	Summary         string   `json:"summary"`
	ReplayLabel     string   `json:"replayLabel,omitempty"`
	Surface         string   `json:"surface,omitempty"`
	ResearchSummary string   `json:"researchSummary,omitempty"`
	ResearchSources []string `json:"researchSources,omitempty"`
	Tags            []string `json:"tags,omitempty"`
}

type SessionSummaryRequest struct {
	UserID    string   `json:"userId"`
	SessionID string   `json:"sessionId,omitempty"`
	Language  string   `json:"language,omitempty"`
	History   []string `json:"history"`
}

type MemoryContextRequest struct {
	UserID   string `json:"userId"`
	Language string `json:"language,omitempty"`
}

type MemoryContextResponse struct {
	Context string `json:"context"`
}

type Client struct {
	baseURL     string
	httpClient  *http.Client
	tokenSource oauth2.TokenSource
}

func NewClient(baseURL string) *Client {
	c := &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}

	// Try to create ID token source for Cloud Run service-to-service auth.
	// Succeeds on GCP (metadata server available), fails gracefully on local dev.
	ts, err := idtoken.NewTokenSource(context.Background(), baseURL)
	if err != nil {
		slog.Warn("adk client: no ID token source — local dev mode", "error", err)
	} else {
		c.tokenSource = ts
		slog.Info("adk client: Cloud Run ID token auth enabled")
	}

	return c
}

func (c *Client) Analyze(ctx context.Context, req AnalysisRequest) (*AnalysisResult, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("adk client: marshal request: %w", err)
	}

	slog.Info("[ADK-CLIENT] >>> POST /analyze",
		"url", c.baseURL+"/analyze",
		"body_bytes", len(body),
		"trace_id", req.TraceID,
		"character", req.Character,
		"context", req.Context,
		"image_bytes", len(req.Image),
	)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/analyze", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("adk client: create request: %w", err)
	}
	if err := c.prepareJSONRequest(httpReq); err != nil {
		slog.Warn("[ADK-CLIENT] failed to prepare analyze request", "error", err)
	}

	startTime := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	elapsed := time.Since(startTime)
	if err != nil {
		slog.Warn("[ADK-CLIENT] <<< request FAILED", "elapsed", elapsed.String(), "error", err)
		return nil, fmt.Errorf("adk client: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("[ADK-CLIENT] <<< unexpected status", "status", resp.StatusCode, "elapsed", elapsed.String())
		return nil, fmt.Errorf("adk client: unexpected status %d", resp.StatusCode)
	}

	var result AnalysisResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		slog.Warn("[ADK-CLIENT] <<< decode FAILED", "elapsed", elapsed.String(), "error", err)
		return nil, fmt.Errorf("adk client: decode response: %w", err)
	}

	slog.Info("[ADK-CLIENT] <<< response OK",
		"trace_id", req.TraceID,
		"elapsed", elapsed.String(),
		"has_vision", result.Vision != nil,
		"has_decision", result.Decision != nil,
		"has_mood", result.Mood != nil,
		"has_celebration", result.Celebration != nil,
		"speech_text_len", len(result.SpeechText),
	)

	return &result, nil
}

func (c *Client) Search(ctx context.Context, req SearchRequest) (*SearchResult, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("adk client: marshal search request: %w", err)
	}

	slog.Info("[ADK-CLIENT] >>> POST /search", "trace_id", req.TraceID, "query", req.Query, "language", req.Language)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/search", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("adk client: create search request: %w", err)
	}
	if err := c.prepareJSONRequest(httpReq); err != nil {
		slog.Warn("[ADK-CLIENT] failed to prepare search request", "error", err)
	}

	startTime := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("adk client: search request failed: %w", err)
	}
	defer resp.Body.Close()
	elapsed := time.Since(startTime)

	if resp.StatusCode == http.StatusNoContent {
		slog.Info("[ADK-CLIENT] <<< search: classifier rejected (not a search query)", "trace_id", req.TraceID, "elapsed", elapsed.String())
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("adk client: search unexpected status %d", resp.StatusCode)
	}

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("adk client: decode search response: %w", err)
	}

	slog.Info("[ADK-CLIENT] <<< search OK", "trace_id", req.TraceID, "elapsed", elapsed.String(), "query", result.Query, "summary_len", len(result.Summary))
	return &result, nil
}

func (c *Client) Tool(ctx context.Context, req ToolRequest) (*ToolResult, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("adk client: marshal tool request: %w", err)
	}

	slog.Info("[ADK-CLIENT] >>> POST /tool", "trace_id", req.TraceID, "query", req.Query, "language", req.Language)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/tool", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("adk client: create tool request: %w", err)
	}
	if err := c.prepareJSONRequest(httpReq); err != nil {
		slog.Warn("[ADK-CLIENT] failed to prepare tool request", "error", err)
	}

	startTime := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("adk client: tool request failed: %w", err)
	}
	defer resp.Body.Close()
	elapsed := time.Since(startTime)

	if resp.StatusCode == http.StatusNoContent {
		slog.Info("[ADK-CLIENT] <<< tool: no result", "trace_id", req.TraceID, "elapsed", elapsed.String())
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("adk client: tool unexpected status %d", resp.StatusCode)
	}

	var result ToolResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("adk client: decode tool response: %w", err)
	}
	slog.Info("[ADK-CLIENT] <<< tool OK", "trace_id", req.TraceID, "elapsed", elapsed.String(), "tool", result.Tool, "summary_len", len(result.Summary))
	return &result, nil
}

func (c *Client) SaveSessionSummary(ctx context.Context, req SessionSummaryRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("adk client: marshal session summary request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/memory/session-summary", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("adk client: create session summary request: %w", err)
	}
	if err := c.prepareJSONRequest(httpReq); err != nil {
		slog.Warn("[ADK-CLIENT] failed to prepare session summary request", "error", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("adk client: session summary request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("adk client: session summary unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) NavigatorEscalate(ctx context.Context, req NavigatorEscalationRequest) (*NavigatorEscalationResult, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("adk client: marshal navigator escalation request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/navigator/escalate", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("adk client: create navigator escalation request: %w", err)
	}
	if err := c.prepareJSONRequest(httpReq); err != nil {
		slog.Warn("[ADK-CLIENT] failed to prepare navigator escalation request", "error", err)
	}

	startTime := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("adk client: navigator escalation request failed: %w", err)
	}
	defer resp.Body.Close()
	elapsed := time.Since(startTime)

	if resp.StatusCode == http.StatusNoContent {
		slog.Info("[ADK-CLIENT] <<< navigator escalation: no result", "trace_id", req.TraceID, "elapsed", elapsed.String())
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("adk client: navigator escalation unexpected status %d", resp.StatusCode)
	}

	var result NavigatorEscalationResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("adk client: decode navigator escalation response: %w", err)
	}
	slog.Info("[ADK-CLIENT] <<< navigator escalation OK", "trace_id", req.TraceID, "elapsed", elapsed.String(), "confidence", result.Confidence)
	return &result, nil
}

func (c *Client) NavigatorBackground(ctx context.Context, req NavigatorBackgroundRequest) (*NavigatorBackgroundResult, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("adk client: marshal navigator background request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/navigator/background", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("adk client: create navigator background request: %w", err)
	}
	if err := c.prepareJSONRequest(httpReq); err != nil {
		slog.Warn("[ADK-CLIENT] failed to prepare navigator background request", "error", err)
	}

	startTime := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("adk client: navigator background request failed: %w", err)
	}
	defer resp.Body.Close()
	elapsed := time.Since(startTime)

	if resp.StatusCode == http.StatusNoContent {
		slog.Info("[ADK-CLIENT] <<< navigator background: no result", "trace_id", req.TraceID, "elapsed", elapsed.String())
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("adk client: navigator background unexpected status %d", resp.StatusCode)
	}

	var result NavigatorBackgroundResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("adk client: decode navigator background response: %w", err)
	}
	slog.Info("[ADK-CLIENT] <<< navigator background OK", "trace_id", req.TraceID, "elapsed", elapsed.String(), "summary_len", len(result.Summary))
	return &result, nil
}

func (c *Client) MemoryContext(ctx context.Context, req MemoryContextRequest) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("adk client: marshal memory context request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/memory/context", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("adk client: create memory context request: %w", err)
	}
	if err := c.prepareJSONRequest(httpReq); err != nil {
		slog.Warn("[ADK-CLIENT] failed to prepare memory context request", "error", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("adk client: memory context request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return "", nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("adk client: memory context unexpected status %d", resp.StatusCode)
	}

	var result MemoryContextResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("adk client: decode memory context response: %w", err)
	}
	return result.Context, nil
}

func (c *Client) prepareJSONRequest(req *http.Request) error {
	req.Header.Set("Content-Type", "application/json")
	otel.GetTextMapPropagator().Inject(req.Context(), propagation.HeaderCarrier(req.Header))

	if c.tokenSource == nil {
		return nil
	}

	token, err := c.tokenSource.Token()
	if err != nil {
		return fmt.Errorf("adk client: get ID token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	return nil
}
