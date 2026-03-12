package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"vibecat/adk-orchestrator/internal/models"
)

func TestOTelHTTPHandlerPreservesRemoteTrace(t *testing.T) {
	previousPropagator := otel.GetTextMapPropagator()
	previousProvider := otel.GetTracerProvider()
	t.Cleanup(func() {
		otel.SetTextMapPropagator(previousPropagator)
		otel.SetTracerProvider(previousProvider)
	})

	tp := sdktrace.NewTracerProvider()
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
	})

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	parentCtx, span := tp.Tracer("test").Start(context.Background(), "parent")
	defer span.End()
	expectedTraceID := span.SpanContext().TraceID().String()

	req := httptest.NewRequest(http.MethodPost, "/analyze", nil)
	otel.GetTextMapPropagator().Inject(parentCtx, propagation.HeaderCarrier(req.Header))

	var gotTraceID string
	handler := otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceID = trace.SpanFromContext(r.Context()).SpanContext().TraceID().String()
		w.WriteHeader(http.StatusNoContent)
	}), serviceName)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if gotTraceID == "" {
		t.Fatal("expected trace context to be extracted")
	}
	if gotTraceID != expectedTraceID {
		t.Fatalf("trace ID = %s, want %s", gotTraceID, expectedTraceID)
	}
}

func TestNeedsAnalyzeSearchBackfill(t *testing.T) {
	t.Run("true for frustrated mood without search", func(t *testing.T) {
		result := models.AnalysisResult{Mood: &models.MoodState{Mood: models.MoodFrustrated}}
		if !needsAnalyzeSearchBackfill(result, "") {
			t.Fatal("expected frustrated mood to trigger search backfill")
		}
	})

	t.Run("true for failing context without search", func(t *testing.T) {
		result := models.AnalysisResult{}
		if !needsAnalyzeSearchBackfill(result, "AuthServiceTests failed with 401 and the developer is stuck") {
			t.Fatal("expected failing context cues to trigger search backfill")
		}
	})

	t.Run("false when search already exists", func(t *testing.T) {
		result := models.AnalysisResult{Search: &models.SearchResult{Summary: "already grounded"}}
		if needsAnalyzeSearchBackfill(result, "build failed") {
			t.Fatal("expected existing search result to suppress backfill")
		}
	})
}

func TestBackfillAnalyzeMoodFromContext(t *testing.T) {
	t.Run("promotes focused mood to frustrated on strong failure context", func(t *testing.T) {
		result := models.AnalysisResult{Mood: &models.MoodState{Mood: models.MoodFocused, Confidence: 0.5}}
		backfillAnalyzeMoodFromContext(&result, "The same test failed with 401 for several minutes and the developer is stuck and sighing")
		if result.Mood == nil || result.Mood.Mood != models.MoodFrustrated {
			t.Fatalf("expected frustrated mood, got %+v", result.Mood)
		}
	})

	t.Run("keeps non-focused mood unchanged", func(t *testing.T) {
		result := models.AnalysisResult{Mood: &models.MoodState{Mood: models.MoodStuck, Confidence: 0.8}}
		backfillAnalyzeMoodFromContext(&result, "failed with 401 and stuck")
		if result.Mood == nil || result.Mood.Mood != models.MoodStuck {
			t.Fatalf("expected stuck mood to stay unchanged, got %+v", result.Mood)
		}
	})
}

func TestBackfillAnalyzeSuccessFromContext(t *testing.T) {
	t.Run("marks success when strong success cues exist", func(t *testing.T) {
		result := models.AnalysisResult{Vision: &models.VisionAnalysis{Significance: 3}}
		backfillAnalyzeSuccessFromContext(&result, "The developer fixed the auth bug, all tests passing, BUILD SUCCEEDED, and shouted yes after the green test run")
		if result.Vision == nil || !result.Vision.SuccessDetected {
			t.Fatalf("expected success detection, got %+v", result.Vision)
		}
		if result.Vision.Significance < 9 {
			t.Fatalf("expected significance to be raised, got %+v", result.Vision)
		}
	})

	t.Run("does not mark success on weak context", func(t *testing.T) {
		result := models.AnalysisResult{Vision: &models.VisionAnalysis{Significance: 3}}
		backfillAnalyzeSuccessFromContext(&result, "developer is reading docs")
		if result.Vision == nil || result.Vision.SuccessDetected {
			t.Fatalf("expected successDetected to stay false, got %+v", result.Vision)
		}
	})
}
