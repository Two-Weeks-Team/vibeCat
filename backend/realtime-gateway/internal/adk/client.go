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
	Image     string `json:"image"`
	Context   string `json:"context"`
	AppName   string `json:"appName,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	UserID    string `json:"userId,omitempty"`
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

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/analyze", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("adk client: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Attach Cloud Run ID token if available
	if c.tokenSource != nil {
		token, tokenErr := c.tokenSource.Token()
		if tokenErr == nil {
			httpReq.Header.Set("Authorization", "Bearer "+token.AccessToken)
		} else {
			slog.Warn("adk client: failed to get ID token", "error", tokenErr)
		}
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("adk client: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("adk client: unexpected status %d", resp.StatusCode)
	}

	var result AnalysisResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("adk client: decode response: %w", err)
	}

	return &result, nil
}
