package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	cloudlogging "cloud.google.com/go/logging"
	mexporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric"
	texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"google.golang.org/adk/agent"
	adkmemory "google.golang.org/adk/memory"
	"google.golang.org/adk/plugin"
	"google.golang.org/adk/plugin/retryandreflect"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/adk/telemetry"
	"google.golang.org/genai"

	"vibecat/adk-orchestrator/internal/agents/graph"
	memoryagent "vibecat/adk-orchestrator/internal/agents/memory"
	"vibecat/adk-orchestrator/internal/agents/search"
	"vibecat/adk-orchestrator/internal/agents/tooluse"
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
	toolAgent      *tooluse.Agent
	memoryAgent    *memoryagent.Agent
	storeClient    *store.Client
	appName        string
	analyzeCounter metric.Int64Counter
	analyzeDurHist metric.Float64Histogram
	errorCounter   metric.Int64Counter
}

func shortHash(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(trimmed))
	return hex.EncodeToString(sum[:])[:12]
}

func appendIdentityLogFields(fields []any, userID, sessionID string) []any {
	userID = strings.TrimSpace(userID)
	sessionID = strings.TrimSpace(sessionID)
	fields = append(fields,
		"user_present", userID != "",
		"session_present", sessionID != "",
	)
	if userID != "" {
		fields = append(fields, "user_ref", shortHash(userID))
	}
	if sessionID != "" {
		fields = append(fields, "session_ref", shortHash(sessionID))
	}
	return fields
}

func appendContextLogFields(fields []any, contextText string) []any {
	contextText = strings.TrimSpace(contextText)
	return append(fields,
		"context_present", contextText != "",
		"context_len", len(contextText),
	)
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
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(serviceName),
	)

	// Cloud Trace
	traceExporter, traceErr := texporter.New(texporter.WithProjectID(projectID))
	if traceErr != nil {
		slog.Warn("cloud trace init failed — tracing disabled", "error", traceErr)
	} else {
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(traceExporter),
			sdktrace.WithResource(res),
		)
		otel.SetTracerProvider(tp)
		defer tp.Shutdown(ctx)
		slog.Info("cloud trace initialized", "project", projectID)
	}

	// Cloud Monitoring (custom metrics)
	metricExporter, metricErr := mexporter.New(mexporter.WithProjectID(projectID))
	if metricErr != nil {
		slog.Warn("cloud monitoring init failed — metrics disabled", "error", metricErr)
	} else {
		mp := sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
			sdkmetric.WithResource(res),
		)
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
	memService := adkmemory.InMemoryService()
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
		toolAgent:      tooluse.New(genaiClient, parseCSVEnv("GEMINI_FILE_SEARCH_STORES")),
		memoryAgent:    memoryagent.New(genaiClient, storeClient),
		storeClient:    storeClient,
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
	mux.HandleFunc("/tool", orch.toolHandler)
	mux.HandleFunc("/memory/session-summary", orch.sessionSummaryHandler)
	mux.HandleFunc("/memory/context", orch.memoryContextHandler)

	addr := ":" + portOrDefault("8080")
	slog.Info("starting server", "service", serviceName, "addr", addr)

	if err := http.ListenAndServe(addr, otelhttp.NewHandler(mux, serviceName)); err != nil {
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
	requestFields := []any{
		"trace_id", req.TraceID,
		"image_bytes", imageLen,
	}
	requestFields = appendContextLogFields(requestFields, req.Context)
	requestFields = appendIdentityLogFields(requestFields, req.UserID, req.SessionID)
	slog.Info("analyze request parsed", requestFields...)
	attrs := []attribute.KeyValue{
		attribute.Int("image.bytes", imageLen),
		attribute.Bool("user.present", strings.TrimSpace(req.UserID) != ""),
		attribute.Bool("session.present", strings.TrimSpace(req.SessionID) != ""),
	}
	if req.TraceID != "" {
		attrs = append(attrs, attribute.String("app.trace_id", req.TraceID))
	}
	span.SetAttributes(attrs...)

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
			sessionFields := appendIdentityLogFields([]any{"error", createErr}, userID, sessionID)
			slog.Warn("failed to create session", sessionFields...)
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
		sessionFields := appendIdentityLogFields([]any{"error", getErr}, userID, sessionID)
		slog.Warn("failed to load session state after run", sessionFields...)
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
		resultFields := []any{
			"trace_id", req.TraceID,
			"had_error", hadError,
			"elapsed", time.Since(analyzeStart).String(),
		}
		resultFields = appendIdentityLogFields(resultFields, userID, sessionID)
		slog.Warn("agent graph produced no usable result", resultFields...)
		http.Error(w, "agent graph failed to produce result", http.StatusServiceUnavailable)
		return
	}

	completeFields := []any{
		"trace_id", req.TraceID,
		"elapsed", time.Since(analyzeStart).String(),
		"should_speak", lastResult.Decision != nil && lastResult.Decision.ShouldSpeak,
		"speech_text_len", len(lastResult.SpeechText),
	}
	completeFields = appendIdentityLogFields(completeFields, userID, sessionID)
	slog.Info("[TIMING] analyze complete", completeFields...)

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

	searchStart := time.Now()
	slog.Info("[SEARCH] voice search request", "trace_id", req.TraceID, "query", req.Query, "language", req.Language)

	result := o.searchAgent.DirectSearch(r.Context(), req.Query, req.Language)

	if result == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	slog.Info("[SEARCH] voice search result", "trace_id", req.TraceID, "elapsed", time.Since(searchStart).String(), "query", result.Query, "summary_len", len(result.Summary))

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		slog.Error("failed to encode search response", "error", err)
	}
}

func (o *orchestrator) toolHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.ToolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Query) == "" {
		http.Error(w, "query is required", http.StatusBadRequest)
		return
	}

	if o.toolAgent == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	toolStart := time.Now()
	slog.Info("[TOOL] request", "trace_id", req.TraceID, "query", req.Query, "language", req.Language)
	result := o.toolAgent.Resolve(r.Context(), req)
	if result == nil {
		slog.Info("[TOOL] empty result", "trace_id", req.TraceID, "elapsed", time.Since(toolStart).String())
		w.WriteHeader(http.StatusNoContent)
		return
	}

	slog.Info("[TOOL] result", "trace_id", req.TraceID, "elapsed", time.Since(toolStart).String(), "tool", result.Tool, "summary_len", len(result.Summary))

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		slog.Error("failed to encode tool response", "error", err)
	}
}

func (o *orchestrator) sessionSummaryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.SessionSummaryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.UserID) == "" {
		http.Error(w, "userId is required", http.StatusBadRequest)
		return
	}
	if len(req.History) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if o.memoryAgent != nil {
		if err := o.memoryAgent.SaveSessionSummary(r.Context(), req.UserID, req.History, req.Language); err != nil {
			summaryFields := appendIdentityLogFields([]any{"error", err}, req.UserID, req.SessionID)
			slog.Warn("failed to save session summary", summaryFields...)
			http.Error(w, "failed to save summary", http.StatusBadGateway)
			return
		}
	}

	if o.storeClient != nil && req.SessionID != "" {
		for _, event := range req.History {
			if strings.TrimSpace(event) == "" {
				continue
			}
			_ = o.storeClient.AddHistoryEntry(r.Context(), req.SessionID, &store.HistoryEntry{
				Timestamp:    time.Now(),
				Type:         inferHistoryType(event),
				Content:      event,
				Significance: 1,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"saved":      true,
		"historyLen": len(req.History),
	})
}

func (o *orchestrator) memoryContextHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.MemoryContextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.UserID) == "" {
		http.Error(w, "userId is required", http.StatusBadRequest)
		return
	}

	contextText := ""
	if o.memoryAgent != nil {
		contextText = o.memoryAgent.RetrieveContext(r.Context(), req.UserID, req.Language)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(models.MemoryContextResponse{
		Context: contextText,
	})
}

func inferHistoryType(event string) string {
	lower := strings.ToLower(event)
	switch {
	case strings.HasPrefix(lower, "user:"):
		return "speech"
	case strings.HasPrefix(lower, "assistant:"):
		return "speech"
	case strings.HasPrefix(lower, "tool["):
		return "search"
	case strings.Contains(lower, "interrupt"):
		return "interruption"
	case strings.HasPrefix(lower, "screen:"):
		return "vision_analysis"
	default:
		return "session_event"
	}
}

func parseCSVEnv(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
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
