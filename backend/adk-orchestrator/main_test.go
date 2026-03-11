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
