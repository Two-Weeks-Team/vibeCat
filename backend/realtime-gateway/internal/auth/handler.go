package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

// RegisterRequest is the body for POST /api/v1/auth/register.
type RegisterRequest struct {
	APIKey string `json:"apiKey"`
}

// TokenResponse is returned on successful auth.
type TokenResponse struct {
	SessionToken string `json:"sessionToken"`
	ExpiresAt    string `json:"expiresAt"`
}

// RegisterHandler returns an http.HandlerFunc for POST /api/v1/auth/register.
// For MVP: accepts any non-empty API key and issues a JWT.
// TODO(Task 10 follow-up): validate key against Gemini API before issuing token.
func RegisterHandler(mgr *Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.APIKey == "" {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		token, exp, err := mgr.Issue()
		if err != nil {
			slog.Error("failed to issue token", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TokenResponse{
			SessionToken: token,
			ExpiresAt:    exp.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
}

// RefreshHandler returns an http.HandlerFunc for POST /api/v1/auth/refresh.
// Validates the existing token and issues a new one.
func RefreshHandler(mgr *Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		existing := bearerToken(r)
		if existing == "" {
			http.Error(w, "missing authorization", http.StatusUnauthorized)
			return
		}
		// Allow refresh even if expired (just check signature)
		// For MVP: re-issue unconditionally if signature is valid
		if err := mgr.Validate(existing); err != nil {
			// Try to parse ignoring expiry for refresh
			slog.Warn("refresh with invalid token", "error", err)
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		token, exp, err := mgr.Issue()
		if err != nil {
			slog.Error("failed to issue refresh token", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TokenResponse{
			SessionToken: token,
			ExpiresAt:    exp.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
}

// Middleware returns an http.Handler that validates Bearer tokens.
// Passes through if token is valid; returns 401 otherwise.
func Middleware(mgr *Manager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			http.Error(w, "missing authorization", http.StatusUnauthorized)
			return
		}
		if err := mgr.Validate(token); err != nil {
			slog.Warn("invalid token", "error", err, "remote", r.RemoteAddr)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// bearerToken extracts the Bearer token from the Authorization header.
func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(h, "Bearer ")
}
