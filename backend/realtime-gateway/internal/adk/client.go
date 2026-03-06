package adk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

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
		"character", req.Character,
		"context", req.Context,
		"image_bytes", len(req.Image),
	)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/analyze", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("adk client: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	if c.tokenSource != nil {
		token, tokenErr := c.tokenSource.Token()
		if tokenErr == nil {
			httpReq.Header.Set("Authorization", "Bearer "+token.AccessToken)
		} else {
			slog.Warn("[ADK-CLIENT] failed to get ID token", "error", tokenErr)
		}
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

	slog.Info("[ADK-CLIENT] >>> POST /search", "query", req.Query, "language", req.Language)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/search", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("adk client: create search request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	if c.tokenSource != nil {
		token, tokenErr := c.tokenSource.Token()
		if tokenErr == nil {
			httpReq.Header.Set("Authorization", "Bearer "+token.AccessToken)
		}
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("adk client: search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		slog.Info("[ADK-CLIENT] <<< search: classifier rejected (not a search query)")
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("adk client: search unexpected status %d", resp.StatusCode)
	}

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("adk client: decode search response: %w", err)
	}

	slog.Info("[ADK-CLIENT] <<< search OK", "query", result.Query, "summary_len", len(result.Summary))
	return &result, nil
}
