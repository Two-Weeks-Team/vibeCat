package ws

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type Metrics struct {
	activeConnections metric.Int64UpDownCounter
	reconnectAttempts metric.Int64Counter
	fallbackRequests  metric.Int64Counter
	adkAnalyzeLatency metric.Float64Histogram
	adkAnalyzeErrors  metric.Int64Counter
}

func NewMetrics(meter metric.Meter) (*Metrics, error) {
	activeConnections, err := meter.Int64UpDownCounter(
		"vibecat.gateway.connections.active",
		metric.WithDescription("Active websocket connections"),
	)
	if err != nil {
		return nil, err
	}

	reconnectAttempts, err := meter.Int64Counter(
		"vibecat.gateway.reconnect.attempts",
		metric.WithDescription("Gemini Live reconnect attempts"),
	)
	if err != nil {
		return nil, err
	}

	fallbackRequests, err := meter.Int64Counter(
		"vibecat.gateway.fallback.requests",
		metric.WithDescription("Fallback requests by kind"),
	)
	if err != nil {
		return nil, err
	}

	adkAnalyzeLatency, err := meter.Float64Histogram(
		"vibecat.gateway.adk.analyze.duration_ms",
		metric.WithDescription("Gateway to ADK analyze duration in milliseconds"),
	)
	if err != nil {
		return nil, err
	}

	adkAnalyzeErrors, err := meter.Int64Counter(
		"vibecat.gateway.adk.analyze.errors",
		metric.WithDescription("Gateway to ADK analyze errors"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		activeConnections: activeConnections,
		reconnectAttempts: reconnectAttempts,
		fallbackRequests:  fallbackRequests,
		adkAnalyzeLatency: adkAnalyzeLatency,
		adkAnalyzeErrors:  adkAnalyzeErrors,
	}, nil
}

func (m *Metrics) ConnectionOpened(ctx context.Context) {
	if m == nil || m.activeConnections == nil {
		return
	}
	m.activeConnections.Add(ctx, 1)
}

func (m *Metrics) ConnectionClosed(ctx context.Context) {
	if m == nil || m.activeConnections == nil {
		return
	}
	m.activeConnections.Add(ctx, -1)
}

func (m *Metrics) ReconnectAttempt(ctx context.Context, trigger string) {
	if m == nil || m.reconnectAttempts == nil {
		return
	}
	m.reconnectAttempts.Add(ctx, 1, metric.WithAttributes(
		attribute.String("trigger", trigger),
	))
}

func (m *Metrics) RecordFallback(ctx context.Context, kind, flow, reason string) {
	if m == nil || m.fallbackRequests == nil {
		return
	}
	m.fallbackRequests.Add(ctx, 1, metric.WithAttributes(
		attribute.String("kind", kind),
		attribute.String("flow", flow),
		attribute.String("reason", reason),
	))
}

func (m *Metrics) RecordADKAnalyzeDuration(ctx context.Context, captureType string, elapsed time.Duration) {
	if m == nil || m.adkAnalyzeLatency == nil {
		return
	}
	m.adkAnalyzeLatency.Record(ctx, float64(elapsed.Milliseconds()), metric.WithAttributes(
		attribute.String("capture_type", captureType),
	))
}

func (m *Metrics) RecordADKAnalyzeError(ctx context.Context, captureType, reason string) {
	if m == nil || m.adkAnalyzeErrors == nil {
		return
	}
	m.adkAnalyzeErrors.Add(ctx, 1, metric.WithAttributes(
		attribute.String("capture_type", captureType),
		attribute.String("reason", reason),
	))
}
