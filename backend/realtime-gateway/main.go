package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"google.golang.org/genai"
	"vibecat/realtime-gateway/internal/adk"
	"vibecat/realtime-gateway/internal/auth"
	"vibecat/realtime-gateway/internal/live"
	"vibecat/realtime-gateway/internal/ws"
)

const serviceName = "realtime-gateway"

var registry = ws.NewRegistry()

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	authSecret := os.Getenv("GATEWAY_AUTH_SECRET")
	if authSecret == "" {
		authSecret = "dev-secret-change-in-production"
		slog.Warn("GATEWAY_AUTH_SECRET not set, using dev default")
	}
	jwtMgr := auth.NewManager(authSecret)

	var liveMgr *live.Manager
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey != "" {
		genaiClient, clientErr := genai.NewClient(context.Background(), &genai.ClientConfig{
			APIKey:  apiKey,
			Backend: genai.BackendGeminiAPI,
		})
		if clientErr != nil {
			slog.Error("failed to create genai client", "error", clientErr)
		} else {
			liveMgr = live.NewManager(genaiClient)
			slog.Info("gemini live manager initialized")
		}
	} else {
		slog.Warn("GEMINI_API_KEY not set, live sessions disabled (stub mode)")
	}

	adkURL := os.Getenv("ADK_ORCHESTRATOR_URL")
	var adkClient *adk.Client
	if adkURL != "" {
		adkClient = adk.NewClient(adkURL)
		slog.Info("adk orchestrator client initialized", "url", adkURL)
	} else {
		slog.Warn("ADK_ORCHESTRATOR_URL not set, screen capture analysis disabled")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/readyz", readyHandler)
	mux.HandleFunc("/api/v1/auth/register", auth.RegisterHandler(jwtMgr))
	mux.HandleFunc("/api/v1/auth/refresh", auth.RefreshHandler(jwtMgr))
	mux.Handle("/ws/live", auth.Middleware(jwtMgr, ws.Handler(registry, liveMgr, adkClient)))

	addr := ":8080"
	slog.Info("starting server", "service", serviceName, "addr", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]any{
		"status":      "ok",
		"service":     serviceName,
		"connections": registry.Count(),
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("failed to encode health response", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"status":  "ok",
		"service": serviceName,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("failed to encode ready response", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}
