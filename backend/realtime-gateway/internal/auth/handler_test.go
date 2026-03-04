package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegisterHandler(t *testing.T) {
	mgr := NewManager("register-secret")
	h := RegisterHandler(mgr)

	tests := []struct {
		name       string
		method     string
		body       string
		wantStatus int
	}{
		{name: "post success", method: http.MethodPost, body: `{"apiKey":"abc"}`, wantStatus: http.StatusOK},
		{name: "wrong method", method: http.MethodGet, body: `{"apiKey":"abc"}`, wantStatus: http.StatusMethodNotAllowed},
		{name: "empty body", method: http.MethodPost, body: ``, wantStatus: http.StatusBadRequest},
		{name: "empty api key", method: http.MethodPost, body: `{"apiKey":""}`, wantStatus: http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/api/v1/auth/register", strings.NewReader(tc.body))
			res := httptest.NewRecorder()

			h.ServeHTTP(res, req)

			if res.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", res.Code, tc.wantStatus)
			}

			if tc.wantStatus == http.StatusOK {
				if got := res.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
					t.Fatalf("content-type = %q, want application/json", got)
				}

				var resp TokenResponse
				if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if resp.SessionToken == "" {
					t.Fatal("expected session token")
				}
				if resp.ExpiresAt == "" {
					t.Fatal("expected expiresAt")
				}
				if err := mgr.Validate(resp.SessionToken); err != nil {
					t.Fatalf("issued token should validate, got %v", err)
				}
			}
		})
	}
}

func TestRefreshHandler(t *testing.T) {
	mgr := NewManager("refresh-secret")
	h := RefreshHandler(mgr)

	validToken, _, err := mgr.Issue()
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{name: "success", authHeader: "Bearer " + validToken, wantStatus: http.StatusOK},
		{name: "missing auth", authHeader: "", wantStatus: http.StatusUnauthorized},
		{name: "invalid token", authHeader: "Bearer invalid.token.value", wantStatus: http.StatusUnauthorized},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			res := httptest.NewRecorder()

			h.ServeHTTP(res, req)

			if res.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", res.Code, tc.wantStatus)
			}

			if tc.wantStatus == http.StatusOK {
				var resp TokenResponse
				if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if resp.SessionToken == "" {
					t.Fatal("expected session token")
				}
			}
		})
	}
}

func TestMiddleware(t *testing.T) {
	mgr := NewManager("middleware-secret")
	validToken, _, err := mgr.Issue()
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	h := Middleware(mgr, next)

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
		wantNext   bool
	}{
		{name: "valid", authHeader: "Bearer " + validToken, wantStatus: http.StatusNoContent, wantNext: true},
		{name: "missing", authHeader: "", wantStatus: http.StatusUnauthorized, wantNext: false},
		{name: "invalid", authHeader: "Bearer bad.token", wantStatus: http.StatusUnauthorized, wantNext: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			nextCalled = false
			req := httptest.NewRequest(http.MethodGet, "/secured", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			res := httptest.NewRecorder()

			h.ServeHTTP(res, req)

			if res.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", res.Code, tc.wantStatus)
			}
			if nextCalled != tc.wantNext {
				t.Fatalf("nextCalled = %v, want %v", nextCalled, tc.wantNext)
			}
		})
	}
}

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		wantToken string
	}{
		{name: "valid bearer", header: "Bearer token-123", wantToken: "token-123"},
		{name: "missing header", header: "", wantToken: ""},
		{name: "wrong scheme", header: "Basic abc", wantToken: ""},
		{name: "bearer no value", header: "Bearer ", wantToken: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			got := bearerToken(req)
			if got != tc.wantToken {
				t.Fatalf("bearerToken() = %q, want %q", got, tc.wantToken)
			}
		})
	}
}
