package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	cloudlogging "cloud.google.com/go/logging"
	mexporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric"
	texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"google.golang.org/genai"
	"vibecat/realtime-gateway/internal/adk"
	"vibecat/realtime-gateway/internal/auth"
	"vibecat/realtime-gateway/internal/live"
	"vibecat/realtime-gateway/internal/tts"
	"vibecat/realtime-gateway/internal/ws"
)

const serviceName = "realtime-gateway"

var registry = ws.NewRegistry()

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

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
		defer tp.Shutdown(context.Background())
		slog.Info("cloud trace initialized", "project", projectID)
	}

	// Cloud Monitoring
	metricExporter, metricErr := mexporter.New(mexporter.WithProjectID(projectID))
	if metricErr != nil {
		slog.Warn("cloud monitoring init failed — metrics disabled", "error", metricErr)
	} else {
		mp := sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
			sdkmetric.WithResource(res),
		)
		otel.SetMeterProvider(mp)
		defer mp.Shutdown(context.Background())
		slog.Info("cloud monitoring initialized", "project", projectID)
	}

	// Cloud Logging
	cloudLogger, logErr := cloudlogging.NewClient(context.Background(), projectID)
	if logErr != nil {
		slog.Warn("cloud logging init failed — using stdout only", "error", logErr)
	} else {
		defer cloudLogger.Close()
		slog.Info("cloud logging initialized", "project", projectID)
	}
	_ = cloudLogger

	authSecret := os.Getenv("GATEWAY_AUTH_SECRET")
	if authSecret == "" {
		authSecret = "dev-secret-change-in-production"
		slog.Warn("GATEWAY_AUTH_SECRET not set, using dev default")
	}
	jwtMgr := auth.NewManager(authSecret)

	var liveMgr *live.Manager
	var ttsClient *tts.Client
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey != "" {
		genaiClient, clientErr := genai.NewClient(context.Background(), &genai.ClientConfig{
			APIKey:  apiKey,
			Backend: genai.BackendGeminiAPI,
			HTTPOptions: genai.HTTPOptions{
				APIVersion: "v1alpha",
			},
		})
		if clientErr != nil {
			slog.Error("failed to create genai client", "error", clientErr)
		} else {
			liveMgr = live.NewManager(genaiClient)
			ttsClient = tts.NewClient(genaiClient)
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

	var wsMetrics *ws.Metrics
	if createdMetrics, err := ws.NewMetrics(otel.Meter("vibecat/gateway")); err != nil {
		slog.Warn("gateway websocket metrics init failed", "error", err)
	} else {
		wsMetrics = createdMetrics
	}

	memoryActionStateStore := ws.NewInMemoryActionStateStore()
	var actionStateStore ws.ActionStateStore = memoryActionStateStore
	if firestoreActionStateStore, err := ws.NewFirestoreActionStateStore(context.Background(), projectID, os.Getenv("ACTION_STATE_FIRESTORE_COLLECTION")); err != nil {
		slog.Warn("action state firestore init failed — using in-memory only", "error", err)
	} else {
		defer firestoreActionStateStore.Close()
		actionStateStore = ws.NewChainedActionStateStore(memoryActionStateStore, firestoreActionStateStore)
		slog.Info("action state firestore initialized", "project", projectID)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/readyz", readyHandler)
	mux.HandleFunc("/api/v1/auth/register", auth.RegisterHandler(jwtMgr))
	mux.HandleFunc("/api/v1/auth/refresh", auth.RefreshHandler(jwtMgr))
	mux.Handle("/ws/live", auth.Middleware(jwtMgr, ws.Handler(registry, liveMgr, adkClient, ttsClient, wsMetrics, actionStateStore)))
	mux.HandleFunc("/debug/inject-text", func(w http.ResponseWriter, r *http.Request) {
		text := r.URL.Query().Get("text")
		if text == "" {
			http.Error(w, "text required", http.StatusBadRequest)
			return
		}
		if err := registry.InjectText(text); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/debug/execute", func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.Query().Get("url")
		text := r.URL.Query().Get("text")
		target := r.URL.Query().Get("target")
		if url == "" && text == "" {
			http.Error(w, "url or text required", http.StatusBadRequest)
			return
		}
		step := ws.BuildDebugStep(url, text, target)
		if err := registry.DispatchStep(step); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write([]byte("ok"))
	})

	addr := ":" + portOrDefault("8080")
	slog.Info("starting server", "service", serviceName, "addr", addr)

	if err := http.ListenAndServe(addr, otelhttp.NewHandler(mux, serviceName)); err != nil {
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

func portOrDefault(fallback string) string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return fallback
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
