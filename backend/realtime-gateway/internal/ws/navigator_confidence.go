package ws

import (
	"context"
	"strings"
	"time"

	"vibecat/realtime-gateway/internal/adk"
)

const navigatorEscalationThreshold = 0.74

func maybeEscalateNavigatorPlan(ctx context.Context, adkClient adkService, metrics *Metrics, language, command string, navCtx navigatorContext, plan navigatorPlan, traceID string) navigatorPlan {
	if adkClient == nil || !shouldAttemptConfidenceEscalation(command, navCtx, plan) {
		return plan
	}

	escalationCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	result, err := adkClient.NavigatorEscalate(escalationCtx, adk.NavigatorEscalationRequest{
		Command:                    strings.TrimSpace(command),
		Language:                   strings.TrimSpace(language),
		AppName:                    strings.TrimSpace(navCtx.AppName),
		BundleID:                   strings.TrimSpace(navCtx.BundleID),
		FrontmostBundleID:          strings.TrimSpace(navCtx.FrontmostBundleID),
		WindowTitle:                strings.TrimSpace(navCtx.WindowTitle),
		FocusedRole:                strings.TrimSpace(navCtx.FocusedRole),
		FocusedLabel:               strings.TrimSpace(navCtx.FocusedLabel),
		SelectedText:               strings.TrimSpace(navCtx.SelectedText),
		AXSnapshot:                 strings.TrimSpace(navCtx.AXSnapshot),
		LastInputFieldDescriptor:   strings.TrimSpace(navCtx.LastInputDescriptor),
		Screenshot:                 strings.TrimSpace(navCtx.Screenshot),
		CaptureConfidence:          navCtx.CaptureConfidence,
		VisibleInputCandidateCount: navCtx.VisibleInputCandidates,
		TraceID:                    traceID,
	})
	if err != nil || result == nil {
		return plan
	}
	if result.Confidence < navigatorEscalationThreshold || strings.EqualFold(result.FallbackRecommendation, "guided_mode") {
		return plan
	}

	switch {
	case wantsTextEntry(command, navCtx):
		if steps := buildEscalatedTextEntrySteps(command, navCtx, plan.IntentConfidence, result); len(steps) > 0 {
			plan.IntentClass = navigatorIntentExecuteNow
			plan.ClarifyQuestion = ""
			plan.Steps = steps
			return plan
		}
	case looksLikeDirectClick(command):
		if step, ok := buildEscalatedAXPressStep(navCtx, plan.IntentConfidence, result); ok {
			plan.IntentClass = navigatorIntentExecuteNow
			plan.ClarifyQuestion = ""
			plan.Steps = []navigatorStep{step}
			return plan
		}
	}

	return plan
}

func shouldAttemptConfidenceEscalation(command string, navCtx navigatorContext, plan navigatorPlan) bool {
	if strings.TrimSpace(navCtx.Screenshot) == "" {
		return false
	}
	if !(wantsTextEntry(command, navCtx) || looksLikeDirectClick(command)) {
		return false
	}
	if wantsTextEntry(command, navCtx) && requiresDerivedTextPayload(command) {
		return true
	}
	if wantsTextEntry(command, navCtx) && requestsTextInsertion(command) && resolveTextEntryPayload(command, "") == "" {
		return false
	}
	if plan.IntentClass != navigatorIntentAmbiguous && len(plan.Steps) > 0 {
		return false
	}
	return navCtx.VisibleInputCandidates > 1 || navCtx.CaptureConfidence < 0.7 || strings.TrimSpace(navCtx.LastInputDescriptor) != ""
}

func buildEscalatedTextEntrySteps(command string, navCtx navigatorContext, intentConfidence float64, escalation *adk.NavigatorEscalationResult) []navigatorStep {
	if escalation == nil {
		return nil
	}
	descriptor := navigatorTargetDescriptor{}
	if escalation.ResolvedDescriptor != nil {
		descriptor = navigatorTargetDescriptor{
			Role:           cleanTopic(escalation.ResolvedDescriptor.Role),
			Label:          cleanTopic(escalation.ResolvedDescriptor.Label),
			WindowTitle:    cleanTopic(firstNonEmptyString(escalation.ResolvedDescriptor.WindowTitle, navCtx.WindowTitle)),
			AppName:        cleanTopic(firstNonEmptyString(escalation.ResolvedDescriptor.AppName, navCtx.AppName)),
			RelativeAnchor: cleanTopic(escalation.ResolvedDescriptor.RelativeAnchor),
			RegionHint:     cleanTopic(escalation.ResolvedDescriptor.RegionHint),
		}
	} else if baseline, ok := buildTextEntryDescriptor(command, navCtx); ok {
		descriptor = baseline
	}
	if descriptor.Role == "" && descriptor.Label == "" && descriptor.AppName == "" {
		return nil
	}
	return buildTextEntryStepsForDescriptor(command, navCtx, intentConfidence, descriptor, escalation.Confidence, escalation.ResolvedText)
}

func buildEscalatedAXPressStep(navCtx navigatorContext, intentConfidence float64, escalation *adk.NavigatorEscalationResult) (navigatorStep, bool) {
	if escalation == nil || escalation.ResolvedDescriptor == nil {
		return navigatorStep{}, false
	}
	descriptor := navigatorTargetDescriptor{
		Role:           cleanTopic(escalation.ResolvedDescriptor.Role),
		Label:          cleanTopic(escalation.ResolvedDescriptor.Label),
		WindowTitle:    cleanTopic(firstNonEmptyString(escalation.ResolvedDescriptor.WindowTitle, navCtx.WindowTitle)),
		AppName:        cleanTopic(firstNonEmptyString(escalation.ResolvedDescriptor.AppName, navCtx.AppName)),
		RelativeAnchor: cleanTopic(escalation.ResolvedDescriptor.RelativeAnchor),
		RegionHint:     cleanTopic(escalation.ResolvedDescriptor.RegionHint),
	}
	if descriptor.Role == "" && descriptor.Label == "" {
		return navigatorStep{}, false
	}
	label := descriptor.Label
	if label == "" {
		label = "target control"
	}
	return navigatorStep{
		ID:               "press_ax_target",
		ActionType:       "press_ax",
		TargetApp:        firstNonEmptyString(descriptor.AppName, navCtx.AppName),
		TargetDescriptor: descriptor,
		ExpectedOutcome:  "Press the " + label,
		Confidence:       maxFloat(escalation.Confidence, 0.76),
		IntentConfidence: intentConfidence,
		RiskLevel:        "low",
		ExecutionPolicy:  navigatorExecutionPolicyLow,
		FallbackPolicy:   "guided_mode",
		VerifyHint:       strings.ToLower(cleanTopic(label)),
	}, true
}

func buildTextEntryStepsForDescriptor(command string, navCtx navigatorContext, intentConfidence float64, descriptor navigatorTargetDescriptor, confidence float64, resolvedText string) []navigatorStep {
	targetApp := cleanTopic(firstNonEmptyString(descriptor.AppName, navCtx.AppName))
	if targetApp == "" {
		return nil
	}
	if descriptor.Role == "" {
		descriptor.Role = "textfield"
	}
	if descriptor.WindowTitle == "" {
		descriptor.WindowTitle = cleanTopic(navCtx.WindowTitle)
	}
	fieldSummary := descriptor.Label
	if fieldSummary == "" {
		fieldSummary = fallbackFieldSummary(descriptor.Role, navCtx)
	}
	if fieldSummary == "" {
		fieldSummary = "input field"
	}

	steps := []navigatorStep{{
		ID:               "focus_input_field",
		ActionType:       "press_ax",
		TargetApp:        targetApp,
		TargetDescriptor: descriptor,
		ExpectedOutcome:  "Focus the " + fieldSummary,
		Confidence:       maxFloat(confidence, 0.82),
		IntentConfidence: intentConfidence,
		RiskLevel:        "low",
		ExecutionPolicy:  navigatorExecutionPolicyLow,
		FallbackPolicy:   "guided_mode",
		VerifyHint:       strings.ToLower(cleanTopic(fieldSummary)),
	}}

	if payload := resolveTextEntryPayload(command, resolvedText); payload != "" {
		verifyHint := payload
		if len(verifyHint) > 24 {
			verifyHint = verifyHint[:24]
		}
		steps = append(steps, navigatorStep{
			ID:               "paste_input_text",
			ActionType:       "paste_text",
			TargetApp:        targetApp,
			TargetDescriptor: descriptor,
			InputText:        payload,
			ExpectedOutcome:  "Insert text into the " + fieldSummary,
			Confidence:       maxFloat(confidence-0.02, 0.8),
			IntentConfidence: intentConfidence,
			RiskLevel:        "low",
			ExecutionPolicy:  navigatorExecutionPolicyLow,
			FallbackPolicy:   "guided_mode",
			VerifyHint:       strings.ToLower(cleanTopic(verifyHint)),
		})
	} else if requestsTextInsertion(command) {
		return nil
	}
	return steps
}

func navigatorSurfaceFromState(state navigatorSessionState) string {
	return navigatorSurfaceFromNames(state.initialAppName, state.activeCommand)
}

func navigatorSurfaceFromContext(navCtx navigatorContext) string {
	return navigatorSurfaceFromNames(navCtx.AppName, "")
}

func navigatorSurfaceFromNames(appName, command string) string {
	lowered := strings.ToLower(strings.TrimSpace(appName) + "\n" + strings.TrimSpace(command))
	switch {
	case strings.Contains(lowered, "chrome"):
		return "Chrome"
	case strings.Contains(lowered, "terminal"):
		return "Terminal"
	case strings.Contains(lowered, "antigravity"):
		return "Antigravity"
	default:
		return "Other"
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
