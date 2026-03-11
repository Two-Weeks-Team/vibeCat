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
	navigatorTasks    metric.Int64Counter
	timeToFirstAction metric.Float64Histogram
	clarifications    metric.Int64Counter
	taskReplacements  metric.Int64Counter
	guidedModes       metric.Int64Counter
	verificationFails metric.Int64Counter
	inputFocusResults metric.Int64Counter
	wrongTargets      metric.Int64Counter
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

	navigatorTasks, err := meter.Int64Counter(
		"vibecat.gateway.navigator.tasks",
		metric.WithDescription("Accepted navigator tasks"),
	)
	if err != nil {
		return nil, err
	}

	timeToFirstAction, err := meter.Float64Histogram(
		"vibecat.gateway.navigator.time_to_first_action_ms",
		metric.WithDescription("Time from task acceptance to the first planned action in milliseconds"),
	)
	if err != nil {
		return nil, err
	}

	clarifications, err := meter.Int64Counter(
		"vibecat.gateway.navigator.clarifications",
		metric.WithDescription("Navigator clarification prompts by kind"),
	)
	if err != nil {
		return nil, err
	}

	taskReplacements, err := meter.Int64Counter(
		"vibecat.gateway.navigator.task_replacements",
		metric.WithDescription("Navigator task replacement prompts"),
	)
	if err != nil {
		return nil, err
	}

	guidedModes, err := meter.Int64Counter(
		"vibecat.gateway.navigator.guided_modes",
		metric.WithDescription("Navigator guided mode outcomes"),
	)
	if err != nil {
		return nil, err
	}

	verificationFails, err := meter.Int64Counter(
		"vibecat.gateway.navigator.step_verification_failures",
		metric.WithDescription("Navigator step verification failures"),
	)
	if err != nil {
		return nil, err
	}

	inputFocusResults, err := meter.Int64Counter(
		"vibecat.gateway.navigator.input_field_focus_results",
		metric.WithDescription("Navigator input field focus verification results"),
	)
	if err != nil {
		return nil, err
	}

	wrongTargets, err := meter.Int64Counter(
		"vibecat.gateway.navigator.wrong_targets",
		metric.WithDescription("Navigator wrong-target detections"),
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
		navigatorTasks:    navigatorTasks,
		timeToFirstAction: timeToFirstAction,
		clarifications:    clarifications,
		taskReplacements:  taskReplacements,
		guidedModes:       guidedModes,
		verificationFails: verificationFails,
		inputFocusResults: inputFocusResults,
		wrongTargets:      wrongTargets,
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

func (m *Metrics) RecordNavigatorTask(ctx context.Context, surface string, intentClass navigatorIntentClass) {
	if m == nil || m.navigatorTasks == nil {
		return
	}
	m.navigatorTasks.Add(ctx, 1, metric.WithAttributes(
		attribute.String("surface", surface),
		attribute.String("intent_class", string(intentClass)),
	))
}

func (m *Metrics) RecordTimeToFirstAction(ctx context.Context, surface string, elapsed time.Duration) {
	if m == nil || m.timeToFirstAction == nil {
		return
	}
	m.timeToFirstAction.Record(ctx, float64(elapsed.Milliseconds()), metric.WithAttributes(
		attribute.String("surface", surface),
	))
}

func (m *Metrics) RecordClarification(ctx context.Context, kind string, surface string) {
	if m == nil || m.clarifications == nil {
		return
	}
	m.clarifications.Add(ctx, 1, metric.WithAttributes(
		attribute.String("kind", kind),
		attribute.String("surface", surface),
	))
}

func (m *Metrics) RecordTaskReplacement(ctx context.Context, surface string) {
	if m == nil || m.taskReplacements == nil {
		return
	}
	m.taskReplacements.Add(ctx, 1, metric.WithAttributes(
		attribute.String("surface", surface),
	))
}

func (m *Metrics) RecordGuidedMode(ctx context.Context, reason string, surface string) {
	if m == nil || m.guidedModes == nil {
		return
	}
	m.guidedModes.Add(ctx, 1, metric.WithAttributes(
		attribute.String("reason", reason),
		attribute.String("surface", surface),
	))
}

func (m *Metrics) RecordVerificationFailure(ctx context.Context, actionType string, surface string) {
	if m == nil || m.verificationFails == nil {
		return
	}
	m.verificationFails.Add(ctx, 1, metric.WithAttributes(
		attribute.String("action_type", actionType),
		attribute.String("surface", surface),
	))
}

func (m *Metrics) RecordInputFieldFocusResult(ctx context.Context, result string, surface string) {
	if m == nil || m.inputFocusResults == nil {
		return
	}
	m.inputFocusResults.Add(ctx, 1, metric.WithAttributes(
		attribute.String("result", result),
		attribute.String("surface", surface),
	))
}

func (m *Metrics) RecordWrongTarget(ctx context.Context, actionType string, surface string) {
	if m == nil || m.wrongTargets == nil {
		return
	}
	m.wrongTargets.Add(ctx, 1, metric.WithAttributes(
		attribute.String("action_type", actionType),
		attribute.String("surface", surface),
	))
}
