package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"vibecat/adk-orchestrator/internal/agents/graph"
	"vibecat/adk-orchestrator/internal/models"
	"vibecat/adk-orchestrator/internal/store"
)

const serviceName = "adk-orchestrator"

// orchestrator holds the ADK runner and genai client for the lifetime of the server.
type orchestrator struct {
	runner      *runner.Runner
	genaiClient *genai.Client
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx := context.Background()

	// Initialize GenAI client (API key from env or Secret Manager)
	apiKey := os.Getenv("GEMINI_API_KEY")
	var genaiClient *genai.Client
	if apiKey != "" {
		var err error
		genaiClient, err = genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  apiKey,
			Backend: genai.BackendGeminiAPI,
		})
		if err != nil {
			slog.Warn("failed to create genai client", "error", err)
		}
	} else {
		slog.Warn("GEMINI_API_KEY not set — agents will run in stub mode")
	}

	var storeClient *store.Client
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		projectID = "vibecat-489105"
	}
	var storeErr error
	storeClient, storeErr = store.NewClient(ctx, projectID)
	if storeErr != nil {
		slog.Warn("failed to create firestore client — memory disabled", "error", storeErr)
		storeClient = nil
	}

	// Build the 9-agent graph
	agentGraph, err := graph.New(genaiClient, storeClient)
	if err != nil {
		slog.Error("failed to build agent graph", "error", err)
		os.Exit(1)
	}

	// Create ADK runner with in-memory session service
	r, err := runner.New(runner.Config{
		AppName:        "vibecat",
		Agent:          agentGraph,
		SessionService: session.InMemoryService(),
	})
	if err != nil {
		slog.Error("failed to create runner", "error", err)
		os.Exit(1)
	}

	orch := &orchestrator{
		runner:      r,
		genaiClient: genaiClient,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/readyz", readyHandler)
	mux.HandleFunc("/analyze", orch.analyzeHandler)

	addr := ":8080"
	slog.Info("starting server", "service", serviceName, "addr", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

// analyzeHandler handles POST /analyze requests from the Realtime Gateway.
// It runs the full 9-agent graph and returns the analysis result.
func (o *orchestrator) analyzeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.AnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Use session/user IDs from request, or defaults
	userID := req.UserID
	if userID == "" {
		userID = "default"
	}
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = "default"
	}

	// Serialize request as JSON for the agent graph input
	reqJSON, err := json.Marshal(req)
	if err != nil {
		http.Error(w, "failed to serialize request", http.StatusInternalServerError)
		return
	}

	msg := genai.NewContentFromText(string(reqJSON), genai.RoleUser)

	// Run the agent graph
	var lastResult models.AnalysisResult
	for event, err := range o.runner.Run(r.Context(), userID, sessionID, msg, agent.RunConfig{}) {
		if err != nil {
			slog.Warn("agent graph error", "error", err)
			continue
		}
		if event == nil {
			continue
		}
		// Extract the last result from the final agent's output
		if event.LLMResponse.Content != nil {
			for _, part := range event.LLMResponse.Content.Parts {
				if part.Text != "" {
					var result models.AnalysisResult
					if jsonErr := json.Unmarshal([]byte(part.Text), &result); jsonErr == nil {
						lastResult = result
					}
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(lastResult); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"status":  "ok",
		"service": serviceName,
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
