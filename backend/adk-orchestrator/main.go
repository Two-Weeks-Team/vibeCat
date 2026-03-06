package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	cloudlogging "cloud.google.com/go/logging"
	mexporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric"
	texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/memory"
	"google.golang.org/adk/plugin"
	"google.golang.org/adk/plugin/retryandreflect"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/adk/telemetry"
	"google.golang.org/genai"

	"vibecat/adk-orchestrator/internal/agents/graph"
	"vibecat/adk-orchestrator/internal/agents/search"
	"vibecat/adk-orchestrator/internal/models"
	"vibecat/adk-orchestrator/internal/store"
)

const serviceName = "adk-orchestrator"

// orchestrator holds the ADK runner and genai client for the lifetime of the server.
type orchestrator struct {
	runner         *runner.Runner
	genaiClient    *genai.Client
	sessionService session.Service
	searchAgent    *search.Agent
	appName        string
	analyzeCounter metric.Int64Counter
	analyzeDurHist metric.Float64Histogram
	errorCounter   metric.Int64Counter
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

	// Cloud Trace
	traceExporter, traceErr := texporter.New(texporter.WithProjectID(projectID))
	if traceErr != nil {
		slog.Warn("cloud trace init failed — tracing disabled", "error", traceErr)
	} else {
		tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(traceExporter))
		otel.SetTracerProvider(tp)
		defer tp.Shutdown(ctx)
		slog.Info("cloud trace initialized", "project", projectID)
	}

	// Cloud Monitoring (custom metrics)
	metricExporter, metricErr := mexporter.New(mexporter.WithProjectID(projectID))
	if metricErr != nil {
		slog.Warn("cloud monitoring init failed — metrics disabled", "error", metricErr)
	} else {
		mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)))
		otel.SetMeterProvider(mp)
		defer mp.Shutdown(ctx)
		slog.Info("cloud monitoring initialized", "project", projectID)
	}

	// Cloud Logging
	cloudLogger, logErr := cloudlogging.NewClient(ctx, projectID)
	if logErr != nil {
		slog.Warn("cloud logging init failed — using stdout only", "error", logErr)
	} else {
		defer cloudLogger.Close()
		slog.Info("cloud logging initialized", "project", projectID)
	}
	_ = cloudLogger

	adkTelemetry, telErr := telemetry.New(ctx,
		telemetry.WithGcpResourceProject(projectID),
	)
	if telErr != nil {
		slog.Warn("adk telemetry init failed", "error", telErr)
	} else {
		adkTelemetry.SetGlobalOtelProviders()
		defer adkTelemetry.Shutdown(ctx)
		slog.Info("adk telemetry initialized", "project", projectID)
	}

	var storeErr error
	storeClient, storeErr = store.NewClient(ctx, projectID)
	if storeErr != nil {
		slog.Warn("failed to create firestore client — memory disabled", "error", storeErr)
		storeClient = nil
	}

	// Build the 9-agent graph
	agentGraph, err := graph.New(genaiClient, storeClient, apiKey)
	if err != nil {
		slog.Error("failed to build agent graph", "error", err)
		os.Exit(1)
	}

	// Create ADK runner with in-memory session service + retryandreflect plugin
	sessService := session.InMemoryService()
	memService := memory.InMemoryService()
	retryPlugin := retryandreflect.MustNew(
		retryandreflect.WithMaxRetries(3),
		retryandreflect.WithTrackingScope(retryandreflect.Invocation),
	)
	r, err := runner.New(runner.Config{
		AppName:        "vibecat",
		Agent:          agentGraph,
		SessionService: sessService,
		MemoryService:  memService,
		PluginConfig: runner.PluginConfig{
			Plugins: []*plugin.Plugin{retryPlugin},
		},
	})
	if err != nil {
		slog.Error("failed to create runner", "error", err)
		os.Exit(1)
	}

	meter := otel.Meter("vibecat/orchestrator")
	analyzeCounter, _ := meter.Int64Counter("vibecat.analyze.requests",
		metric.WithDescription("Total analyze requests"),
	)
	analyzeDurHist, _ := meter.Float64Histogram("vibecat.analyze.duration_ms",
		metric.WithDescription("Analyze request duration in milliseconds"),
	)
	errorCounter, _ := meter.Int64Counter("vibecat.analyze.errors",
		metric.WithDescription("Total analyze errors"),
	)

	orch := &orchestrator{
		runner:         r,
		genaiClient:    genaiClient,
		sessionService: sessService,
		searchAgent:    search.New(genaiClient),
		appName:        "vibecat",
		analyzeCounter: analyzeCounter,
		analyzeDurHist: analyzeDurHist,
		errorCounter:   errorCounter,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/readyz", readyHandler)
	mux.HandleFunc("/analyze", orch.analyzeHandler)
	mux.HandleFunc("/search", orch.searchHandler)

	addr := ":" + portOrDefault("8080")
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

	analyzeStart := time.Now()
	o.analyzeCounter.Add(r.Context(), 1)

	slog.Info("analyze request received", "content_length", r.ContentLength, "remote", r.RemoteAddr)

	tracer := otel.Tracer("vibecat/orchestrator")
	_, span := tracer.Start(r.Context(), "orchestrator.analyze")
	defer span.End()
	defer func() {
		o.analyzeDurHist.Record(r.Context(), float64(time.Since(analyzeStart).Milliseconds()))
	}()

	var req models.AnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("analyze request decode failed", "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	imageLen := len(req.Image)
	slog.Info("analyze request parsed", "image_bytes", imageLen, "context", req.Context, "user_id", req.UserID, "session_id", req.SessionID)

	// Use session/user IDs from request, or defaults
	userID := req.UserID
	if userID == "" {
		userID = "default"
	}
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = "default"
	}

	getResp, getErr := o.sessionService.Get(r.Context(), &session.GetRequest{
		AppName:   o.appName,
		UserID:    userID,
		SessionID: sessionID,
	})
	if getErr != nil {
		createResp, createErr := o.sessionService.Create(r.Context(), &session.CreateRequest{
			AppName:   o.appName,
			UserID:    userID,
			SessionID: sessionID,
		})
		if createErr != nil {
			slog.Warn("failed to create session", "error", createErr, "session_id", sessionID)
		} else if createResp != nil && createResp.Session != nil {
			getResp = &session.GetResponse{Session: createResp.Session}
		}
	}
	if getResp != nil && getResp.Session != nil && getResp.Session.State() != nil {
		_ = getResp.Session.State().Set("activity_minutes", req.ActivityMinutes)
		if req.Language != "" {
			_ = getResp.Session.State().Set("language", req.Language)
		}
	}

	reqJSON, err := json.Marshal(req)
	if err != nil {
		http.Error(w, "failed to serialize request", http.StatusInternalServerError)
		return
	}

	msg := genai.NewContentFromText(string(reqJSON), genai.RoleUser)

	// Run the agent graph
	var lastResult models.AnalysisResult
	var hadError bool
	for event, err := range o.runner.Run(r.Context(), userID, sessionID, msg, agent.RunConfig{}) {
		if err != nil {
			slog.Warn("agent graph error", "error", err)
			hadError = true
			continue
		}
		if event == nil {
			continue
		}
		agentName := ""
		if event.Author != "" {
			agentName = event.Author
		}
		slog.Info("[GRAPH] agent event", "agent", agentName, "has_content", event.LLMResponse.Content != nil)
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

	getResp, getErr = o.sessionService.Get(r.Context(), &session.GetRequest{
		AppName:   o.appName,
		UserID:    userID,
		SessionID: sessionID,
	})
	if getErr != nil {
		slog.Warn("failed to load session state after run", "error", getErr, "user_id", userID, "session_id", sessionID)
	} else if getResp != nil && getResp.Session != nil && getResp.Session.State() != nil {
		if val, err := getResp.Session.State().Get("analysis_result"); err == nil {
			if str, ok := val.(string); ok {
				var stateResult models.AnalysisResult
				if unmarshalErr := json.Unmarshal([]byte(str), &stateResult); unmarshalErr == nil {
					lastResult = stateResult
				} else {
					slog.Warn("failed to parse analysis_result from session state", "error", unmarshalErr)
				}
			}
		}
	}

	if lastResult.Decision == nil && lastResult.Vision == nil && lastResult.SpeechText == "" {
		o.errorCounter.Add(r.Context(), 1)
		slog.Warn("agent graph produced no usable result", "user_id", userID, "session_id", sessionID, "had_error", hadError)
		http.Error(w, "agent graph failed to produce result", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(lastResult); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func (o *orchestrator) searchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "query is required", http.StatusBadRequest)
		return
	}

	slog.Info("[SEARCH] voice search request", "query", req.Query, "language", req.Language)

	result := o.searchAgent.DirectSearch(r.Context(), req.Query, req.Language)

	if result == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	slog.Info("[SEARCH] voice search result", "query", result.Query, "summary_len", len(result.Summary))

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		slog.Error("failed to encode search response", "error", err)
	}
}

func portOrDefault(fallback string) string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return fallback
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
