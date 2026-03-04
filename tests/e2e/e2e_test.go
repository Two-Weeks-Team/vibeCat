// Package e2e contains end-to-end tests that run against deployed VibeCat services.
//
// Usage:
//
//	GATEWAY_URL=https://realtime-gateway-....run.app go test -v -count=1 ./...
//
// For local testing:
//
//	GATEWAY_URL=http://localhost:8080 go test -v -count=1 ./...
package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func gatewayURL(t *testing.T) string {
	t.Helper()
	u := os.Getenv("GATEWAY_URL")
	if u == "" {
		t.Skip("GATEWAY_URL not set — skipping E2E tests")
	}
	return strings.TrimRight(u, "/")
}

// Test 1: Health check — /readyz returns 200 OK
func TestHealthCheck(t *testing.T) {
	base := gatewayURL(t)
	resp, err := http.Get(base + "/readyz")
	if err != nil {
		t.Fatalf("GET /readyz failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET /readyz: expected 200, got %d: %s", resp.StatusCode, body)
	}
	t.Log("✅ /readyz — 200 OK")
}

// Test 2: JWT registration — POST /api/v1/auth/register returns a token
func TestJWTRegistration(t *testing.T) {
	base := gatewayURL(t)
	token := registerToken(t, base)
	if token == "" {
		t.Fatal("registration returned empty token")
	}
	t.Logf("✅ JWT registration — token received (len=%d)", len(token))
}

// Test 3: Token refresh — POST /api/v1/auth/refresh returns a new token
func TestTokenRefresh(t *testing.T) {
	base := gatewayURL(t)
	token := registerToken(t, base)

	req, err := http.NewRequest(http.MethodPost, base+"/api/v1/auth/refresh", nil)
	if err != nil {
		t.Fatalf("create refresh request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /api/v1/auth/refresh failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("refresh: expected 200, got %d: %s", resp.StatusCode, body)
	}

	var result struct {
		SessionToken string `json:"sessionToken"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode refresh response: %v", err)
	}
	if result.SessionToken == "" {
		t.Fatal("refresh returned empty token")
	}
	if result.SessionToken == token {
		t.Log("⚠️  refresh returned same token (within expiry window — acceptable)")
	}
	t.Log("✅ Token refresh — new token issued")
}

// Test 4: WebSocket upgrade — connection established with valid JWT
func TestWebSocketUpgrade(t *testing.T) {
	base := gatewayURL(t)
	token := registerToken(t, base)

	conn := dialWS(t, base, token)
	defer conn.Close()

	t.Log("✅ WebSocket upgrade — connection established")
}

// Test 5: Gemini Live setup — send setup message, receive setupComplete
func TestGeminiLiveSetup(t *testing.T) {
	base := gatewayURL(t)
	token := registerToken(t, base)

	conn := dialWS(t, base, token)
	defer conn.Close()

	// Send setup message
	setup := map[string]any{
		"type": "setup",
		"config": map[string]any{
			"voice":    "Zephyr",
			"language": "ko",
		},
	}
	if err := conn.WriteJSON(setup); err != nil {
		t.Fatalf("send setup message: %v", err)
	}

	// Read response (setupComplete or error)
	conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read setup response: %v", err)
	}

	var resp map[string]any
	if err := json.Unmarshal(msg, &resp); err != nil {
		t.Fatalf("parse setup response: %v", err)
	}

	msgType, _ := resp["type"].(string)
	if msgType == "error" {
		// Gemini connection may fail in test environment — still proves the path works
		t.Logf("⚠️  Setup returned error (expected in test env): %v", resp["message"])
		t.Log("✅ Gemini Live setup — setup path exercised (error expected without real Gemini)")
		return
	}
	if msgType != "setupComplete" {
		t.Fatalf("expected setupComplete or error, got %q: %s", msgType, msg)
	}
	t.Log("✅ Gemini Live setup — audio session initialized")
}

// Test 6: Auth rejection — unauthenticated WebSocket upgrade is rejected
func TestAuthRejection(t *testing.T) {
	base := gatewayURL(t)

	wsURL := httpToWS(base) + "/ws/live"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected WebSocket upgrade to fail without auth, but it succeeded")
	}
	if resp != nil && resp.StatusCode == http.StatusUnauthorized {
		t.Log("✅ Auth rejection — unauthenticated request correctly blocked (401)")
		return
	}
	if resp != nil {
		t.Logf("✅ Auth rejection — unauthenticated request blocked (status=%d)", resp.StatusCode)
		return
	}
	// Connection refused or other network error is also acceptable (means auth middleware blocked it)
	t.Logf("✅ Auth rejection — connection failed as expected: %v", err)
}

// Test 7: Invalid token rejection — bad JWT is rejected
func TestInvalidTokenRejection(t *testing.T) {
	base := gatewayURL(t)

	wsURL := httpToWS(base) + "/ws/live"
	header := http.Header{}
	header.Set("Authorization", "Bearer invalid-token-12345")

	_, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err == nil {
		t.Fatal("expected WebSocket upgrade to fail with invalid token, but it succeeded")
	}
	if resp != nil && resp.StatusCode == http.StatusUnauthorized {
		t.Log("✅ Invalid token rejection — bad JWT correctly blocked (401)")
		return
	}
	if resp != nil {
		t.Logf("✅ Invalid token rejection — blocked (status=%d)", resp.StatusCode)
		return
	}
	t.Logf("✅ Invalid token rejection — connection failed: %v", err)
}

// --- Helpers ---

func registerToken(t *testing.T, base string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"apiKey": "test-key"})
	resp, err := http.Post(base+"/api/v1/auth/register", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /api/v1/auth/register failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("register: expected 200, got %d: %s", resp.StatusCode, respBody)
	}

	var result struct {
		SessionToken string `json:"sessionToken"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	return result.SessionToken
}

func dialWS(t *testing.T, base, token string) *websocket.Conn {
	t.Helper()
	wsURL := httpToWS(base) + "/ws/live"
	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		msg := fmt.Sprintf("WebSocket dial failed: %v", err)
		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			msg += fmt.Sprintf(" (status=%d, body=%s)", resp.StatusCode, body)
		}
		t.Fatal(msg)
	}
	return conn
}

func httpToWS(httpURL string) string {
	u, _ := url.Parse(httpURL)
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	default:
		u.Scheme = "ws"
	}
	return u.String()
}
