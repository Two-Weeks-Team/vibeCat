package adk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	baseURL := "http://example.test"
	c := NewClient(baseURL)

	if c == nil {
		t.Fatal("expected client, got nil")
	}
	if c.baseURL != baseURL {
		t.Fatalf("baseURL = %q, want %q", c.baseURL, baseURL)
	}
	if c.httpClient == nil {
		t.Fatal("expected httpClient to be initialized")
	}
	if c.httpClient.Timeout != defaultTimeout {
		t.Fatalf("http timeout = %v, want %v", c.httpClient.Timeout, defaultTimeout)
	}
}

func TestClientAnalyze(t *testing.T) {
	tests := []struct {
		name        string
		handler     http.HandlerFunc
		wantErr     bool
		errContains string
		checkResult func(t *testing.T, got *AnalysisResult)
	}{
		{
			name: "success response",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					http.Error(w, "bad method", http.StatusMethodNotAllowed)
					return
				}
				if r.URL.Path != "/analyze" {
					http.Error(w, "bad path", http.StatusNotFound)
					return
				}
				if got := r.Header.Get("Content-Type"); !strings.Contains(got, "application/json") {
					http.Error(w, "bad content type", http.StatusBadRequest)
					return
				}

				var req AnalysisRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, "bad body", http.StatusBadRequest)
					return
				}
				if req.Image == "" || req.Context == "" {
					http.Error(w, "missing fields", http.StatusBadRequest)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"speechText":"hello","decision":{"shouldSpeak":true,"reason":"needed","urgency":"high"}}`))
			},
			wantErr: false,
			checkResult: func(t *testing.T, got *AnalysisResult) {
				t.Helper()
				if got == nil {
					t.Fatal("expected result, got nil")
				}
				if got.SpeechText != "hello" {
					t.Fatalf("SpeechText = %q, want hello", got.SpeechText)
				}
				if got.Decision == nil || !got.Decision.ShouldSpeak {
					t.Fatalf("Decision = %#v, want shouldSpeak=true", got.Decision)
				}
			},
		},
		{
			name: "error status",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "server error", http.StatusInternalServerError)
			},
			wantErr:     true,
			errContains: "unexpected status 500",
		},
		{
			name: "malformed json response",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte("{"))
			},
			wantErr:     true,
			errContains: "decode response",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(tc.handler)
			defer ts.Close()

			c := &Client{
				baseURL:    ts.URL,
				httpClient: &http.Client{Timeout: 5 * time.Second},
			}

			got, err := c.Analyze(context.Background(), AnalysisRequest{
				Image:     "base64data",
				Context:   "editor",
				AppName:   "VSCode",
				SessionID: "session-1",
				UserID:    "user-1",
			})

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Fatalf("error = %v, want containing %q", err, tc.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("Analyze() error = %v", err)
			}
			if tc.checkResult != nil {
				tc.checkResult(t, got)
			}
		})
	}
}
