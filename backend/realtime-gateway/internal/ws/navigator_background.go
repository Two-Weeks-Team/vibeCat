package ws

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"vibecat/realtime-gateway/internal/adk"
	"vibecat/realtime-gateway/internal/live"
)

func enqueueNavigatorBackground(ctx context.Context, adkClient adkService, runtime *sessionRuntime, cfg live.Config, snapshot *navigatorTaskSnapshot, outcome, outcomeDetail, traceID string) {
	if adkClient == nil || snapshot == nil {
		return
	}

	userID, sessionID, _ := runtime.snapshot()
	request := adk.NavigatorBackgroundRequest{
		UserID:                  userID,
		SessionID:               sessionID,
		TaskID:                  snapshot.TaskID,
		Command:                 snapshot.Command,
		Language:                cfg.Language,
		Outcome:                 outcome,
		OutcomeDetail:           outcomeDetail,
		Surface:                 snapshot.Surface,
		InitialAppName:          snapshot.InitialAppName,
		InitialWindowTitle:      snapshot.InitialWindowTitle,
		InitialContextHash:      snapshot.InitialContextHash,
		LastVerifiedContextHash: snapshot.LastVerifiedContextHash,
		StartedAt:               snapshot.StartedAt,
		CompletedAt:             snapshot.CompletedAt,
		Steps:                   backgroundSteps(snapshot.Steps),
		Attempts:                backgroundAttempts(snapshot.Attempts),
		TraceID:                 traceID,
	}

	go func() {
		backgroundCtx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		result, err := adkClient.NavigatorBackground(backgroundCtx, request)
		if err != nil {
			slog.Warn("navigator background failed", "task_id", snapshot.TaskID, "trace_id", traceID, "error", err)
			return
		}
		if result == nil {
			return
		}
		if runtime != nil && strings.TrimSpace(result.Summary) != "" {
			runtime.append(fmt.Sprintf("navigator[%s]: %s", snapshot.TaskID, truncateText(result.Summary, 240)))
		}
		if runtime != nil && strings.TrimSpace(result.ResearchSummary) != "" {
			runtime.append(fmt.Sprintf("navigator_research[%s]: %s", snapshot.TaskID, truncateText(result.ResearchSummary, 240)))
		}
		if runtime != nil && len(snapshot.Attempts) > 0 {
			last := snapshot.Attempts[len(snapshot.Attempts)-1]
			runtime.append(fmt.Sprintf(
				"navigator_attempt[%s]: route=%s outcome=%s detail=%s",
				last.ID,
				truncateText(last.Route, 40),
				truncateText(last.Outcome, 40),
				truncateText(last.OutcomeDetail, 180),
			))
		}
	}()
}

func backgroundSteps(steps []navigatorStepTrace) []adk.NavigatorBackgroundStep {
	if len(steps) == 0 {
		return nil
	}
	out := make([]adk.NavigatorBackgroundStep, 0, len(steps))
	for _, step := range steps {
		out = append(out, adk.NavigatorBackgroundStep{
			ID:         step.ID,
			ActionType: step.ActionType,
			TargetApp:  step.TargetApp,
			TargetDescriptor: adk.NavigatorTargetDescriptor{
				Role:           step.TargetDescriptor.Role,
				Label:          step.TargetDescriptor.Label,
				WindowTitle:    step.TargetDescriptor.WindowTitle,
				AppName:        step.TargetDescriptor.AppName,
				RelativeAnchor: step.TargetDescriptor.RelativeAnchor,
				RegionHint:     step.TargetDescriptor.RegionHint,
			},
			ResultStatus:    step.ResultStatus,
			ObservedOutcome: step.ObservedOutcome,
			PlannedAt:       step.PlannedAt,
			CompletedAt:     step.CompletedAt,
		})
	}
	return out
}

func backgroundAttempts(attempts []navigatorAttemptTrace) []adk.NavigatorBackgroundAttempt {
	if len(attempts) == 0 {
		return nil
	}
	out := make([]adk.NavigatorBackgroundAttempt, 0, len(attempts))
	for _, attempt := range attempts {
		out = append(out, adk.NavigatorBackgroundAttempt{
			ID:               attempt.ID,
			TaskID:           attempt.TaskID,
			Command:          attempt.Command,
			Surface:          attempt.Surface,
			Route:            attempt.Route,
			RouteReason:      attempt.RouteReason,
			ContextHash:      attempt.ContextHash,
			ScreenshotSource: attempt.ScreenshotSource,
			ScreenshotCached: attempt.ScreenshotCached,
			ScreenBasisID:    attempt.ScreenBasisID,
			ActiveDisplayID:  attempt.ActiveDisplayID,
			TargetDisplayID:  attempt.TargetDisplayID,
			Outcome:          attempt.Outcome,
			OutcomeDetail:    attempt.OutcomeDetail,
			StartedAt:        attempt.StartedAt,
			CompletedAt:      attempt.CompletedAt,
		})
	}
	return out
}

func isInputFieldFocusStep(step navigatorStep) bool {
	return step.ActionType == "press_ax" && looksLikeTextInputRole(step.TargetDescriptor.Role)
}

func shouldRecordWrongTarget(observedOutcome string) bool {
	lowered := strings.ToLower(strings.TrimSpace(observedOutcome))
	return strings.Contains(lowered, "wrong target") ||
		strings.Contains(lowered, "wrong field") ||
		strings.Contains(lowered, "not the intended target")
}
