package navigator

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	memoryagent "vibecat/adk-orchestrator/internal/agents/memory"
	"vibecat/adk-orchestrator/internal/agents/search"
	"vibecat/adk-orchestrator/internal/geminiconfig"
	"vibecat/adk-orchestrator/internal/lang"
	"vibecat/adk-orchestrator/internal/models"
	"vibecat/adk-orchestrator/internal/store"

	"google.golang.org/genai"
)

type Processor struct {
	genaiClient *genai.Client
	searchAgent *search.Agent
	memoryAgent *memoryagent.Agent
	storeClient *store.Client
}

func NewProcessor(genaiClient *genai.Client, searchAgent *search.Agent, memoryAgent *memoryagent.Agent, storeClient *store.Client) *Processor {
	return &Processor{
		genaiClient: genaiClient,
		searchAgent: searchAgent,
		memoryAgent: memoryAgent,
		storeClient: storeClient,
	}
}

func (p *Processor) ResolveTarget(ctx context.Context, req models.NavigatorEscalationRequest) *models.NavigatorEscalationResult {
	fallback := heuristicEscalation(req)
	if p == nil || p.genaiClient == nil || strings.TrimSpace(req.Screenshot) == "" {
		return fallback
	}

	decoded, err := base64.StdEncoding.DecodeString(req.Screenshot)
	if err != nil {
		slog.Warn("navigator escalator: screenshot decode failed", "trace_id", req.TraceID, "error", err)
		return fallback
	}

	prompt := buildEscalationPrompt(req)
	resp, err := p.genaiClient.Models.GenerateContent(ctx, geminiconfig.VisionModel, []*genai.Content{
		{
			Role: genai.RoleUser,
			Parts: []*genai.Part{
				{Text: prompt},
				{InlineData: &genai.Blob{MIMEType: "image/jpeg", Data: decoded}},
			},
		},
	}, &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{
				Text: `You are a narrow target resolver for a macOS desktop UI navigator.

Resolve only one likely target for the user's command.

Rules:
- Use the screenshot as your PRIMARY source. AX evidence is secondary.
- When you can visually identify the target element in the screenshot, return clickX and clickY as normalized coordinates (0.0-1.0) relative to the full screenshot dimensions. clickX=0.0 is left edge, clickX=1.0 is right edge. clickY=0.0 is top edge, clickY=1.0 is bottom edge.
- For music/video player pages (YouTube Music, YouTube, Spotify), prioritize the first playable content item, shuffle button, or play button.
- Return guided_mode when the target is not clear enough.
- Never invent a label that is not supported by the screenshot or AX context.
- Keep confidence between 0.0 and 1.0.
- role must be one of textfield, textarea, button, link, tab, menuitem, or empty.
- Return JSON only.`,
			}},
		},
		MaxOutputTokens: 400,
	})
	if err != nil {
		slog.Warn("navigator escalator: model call failed", "trace_id", req.TraceID, "error", err)
		return fallback
	}

	text := extractResponseText(resp)
	if text == "" {
		return fallback
	}

	var result models.NavigatorEscalationResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		slog.Warn("navigator escalator: decode failed", "trace_id", req.TraceID, "error", err)
		return fallback
	}
	if !validEscalationResult(result) {
		return fallback
	}

	if result.Confidence < fallback.Confidence {
		return fallback
	}
	return &result
}

func (p *Processor) ProcessBackground(ctx context.Context, req models.NavigatorBackgroundRequest) *models.NavigatorBackgroundResult {
	result := backgroundFallback(req)
	if p == nil {
		return result
	}

	if p.genaiClient != nil {
		if generated := p.generateBackgroundSummary(ctx, req); generated != nil {
			result = generated
		}
	}

	if shouldEnrichResearch(req.Command) && p.searchAgent != nil {
		if enrichment := p.searchAgent.DirectSearch(ctx, buildResearchQuery(req), req.Language); enrichment != nil {
			result.ResearchSummary = strings.TrimSpace(enrichment.Summary)
			result.ResearchSources = append([]string(nil), enrichment.Sources...)
			if result.ResearchSummary != "" && !containsString(result.Tags, "research_enriched") {
				result.Tags = append(result.Tags, "research_enriched")
			}
		}
	}

	history := backgroundHistory(req, result)
	if p.memoryAgent != nil && strings.TrimSpace(req.UserID) != "" {
		if err := p.memoryAgent.SaveTaskSummary(ctx, req.UserID, result.Summary, history, req.Language); err != nil {
			slog.Warn("navigator background: memory write failed", "trace_id", req.TraceID, "error", err)
		}
	}

	if p.storeClient != nil {
		replay := &store.NavigatorReplay{
			TaskID:                  req.TaskID,
			UserID:                  req.UserID,
			SessionID:               req.SessionID,
			Command:                 req.Command,
			Outcome:                 req.Outcome,
			OutcomeDetail:           req.OutcomeDetail,
			Surface:                 firstNonEmpty(result.Surface, req.Surface),
			Summary:                 result.Summary,
			ReplayLabel:             result.ReplayLabel,
			ResearchSummary:         result.ResearchSummary,
			ResearchSources:         append([]string(nil), result.ResearchSources...),
			Tags:                    append([]string(nil), result.Tags...),
			InitialAppName:          req.InitialAppName,
			InitialWindowTitle:      req.InitialWindowTitle,
			InitialContextHash:      req.InitialContextHash,
			LastVerifiedContextHash: req.LastVerifiedContextHash,
			StartedAt:               req.StartedAt,
			CompletedAt:             req.CompletedAt,
			UpdatedAt:               time.Now().UTC(),
			Attempts:                replayAttempts(req.Attempts),
		}
		if err := p.storeClient.StoreNavigatorReplay(ctx, replay); err != nil {
			slog.Warn("navigator background: replay store failed", "trace_id", req.TraceID, "error", err)
		}
	}

	return result
}

func buildEscalationPrompt(req models.NavigatorEscalationRequest) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Command: %s\n", strings.TrimSpace(req.Command))
	fmt.Fprintf(&b, "Language: %s\n", lang.NormalizeLanguage(req.Language))
	fmt.Fprintf(&b, "App: %s\n", strings.TrimSpace(req.AppName))
	fmt.Fprintf(&b, "Bundle: %s\n", strings.TrimSpace(req.BundleID))
	fmt.Fprintf(&b, "Frontmost bundle: %s\n", strings.TrimSpace(req.FrontmostBundleID))
	fmt.Fprintf(&b, "Window: %s\n", strings.TrimSpace(req.WindowTitle))
	fmt.Fprintf(&b, "Focused role: %s\n", strings.TrimSpace(req.FocusedRole))
	fmt.Fprintf(&b, "Focused label: %s\n", strings.TrimSpace(req.FocusedLabel))
	fmt.Fprintf(&b, "Selected text: %s\n", truncate(req.SelectedText, 200))
	fmt.Fprintf(&b, "AX snapshot:\n%s\n", truncate(req.AXSnapshot, 1600))
	fmt.Fprintf(&b, "Last input descriptor: %s\n", strings.TrimSpace(req.LastInputFieldDescriptor))
	fmt.Fprintf(&b, "Capture confidence: %.2f\n", req.CaptureConfidence)
	fmt.Fprintf(&b, "Visible input candidates: %d\n", req.VisibleInputCandidateCount)
	b.WriteString("If the user wants you to type text that must be copied from visible UI content, extract only the exact visible text to place into resolvedText.\n")
	b.WriteString("If the requested text is not clearly visible, leave resolvedText empty.\n")
	b.WriteString(`Return JSON using this schema:
{"resolvedDescriptor":{"role":"","label":"","windowTitle":"","appName":"","relativeAnchor":"","regionHint":"","clickX":0.0,"clickY":0.0},"resolvedText":"","confidence":0.0,"fallbackRecommendation":"guided_mode|ask_clarify|safe_immediate","reason":""}`)
	return b.String()
}

func replayAttempts(attempts []models.NavigatorBackgroundAttempt) []store.NavigatorAttempt {
	if len(attempts) == 0 {
		return nil
	}
	out := make([]store.NavigatorAttempt, 0, len(attempts))
	for _, attempt := range attempts {
		out = append(out, store.NavigatorAttempt{
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

func heuristicEscalation(req models.NavigatorEscalationRequest) *models.NavigatorEscalationResult {
	label := descriptorLabel(req.LastInputFieldDescriptor)
	role := "textfield"
	if label == "" && looksLikeTextInputRole(req.FocusedRole) {
		label = clean(req.FocusedLabel)
	}
	if label == "" {
		switch {
		case containsAny(req.Command, "search", "검색"):
			label = "Search"
		case containsAny(req.Command, "address bar", "주소창", "url bar"):
			label = "Address"
		case containsAny(req.Command, "prompt", "프롬프트"):
			label = "Prompt"
		}
	}
	confidence := 0.46
	if label != "" {
		confidence = 0.72
	}
	if req.VisibleInputCandidateCount == 1 && label != "" {
		confidence = 0.78
	}
	result := &models.NavigatorEscalationResult{
		Confidence:             confidence,
		FallbackRecommendation: "guided_mode",
		Reason:                 "no_strong_visual_resolution",
	}
	if label != "" {
		result.ResolvedDescriptor = &models.NavigatorTargetDescriptor{
			Role:        role,
			Label:       label,
			WindowTitle: clean(req.WindowTitle),
			AppName:     clean(req.AppName),
		}
		result.FallbackRecommendation = "safe_immediate"
		result.Reason = "heuristic_textfield_resolution"
	}
	return result
}

func validEscalationResult(result models.NavigatorEscalationResult) bool {
	if result.Confidence <= 0 {
		return false
	}
	if result.ResolvedDescriptor == nil {
		if strings.TrimSpace(result.ResolvedText) != "" {
			return true
		}
		return strings.TrimSpace(result.FallbackRecommendation) != ""
	}
	return strings.TrimSpace(result.ResolvedDescriptor.Role) != "" ||
		strings.TrimSpace(result.ResolvedDescriptor.Label) != "" ||
		strings.TrimSpace(result.ResolvedText) != ""
}

func (p *Processor) generateBackgroundSummary(ctx context.Context, req models.NavigatorBackgroundRequest) *models.NavigatorBackgroundResult {
	prompt := buildBackgroundPrompt(req)
	resp, err := p.genaiClient.Models.GenerateContent(ctx, geminiconfig.LiteTextModel, []*genai.Content{
		{Role: genai.RoleUser, Parts: []*genai.Part{{Text: prompt}}},
	}, &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		MaxOutputTokens:  300,
	})
	if err != nil {
		slog.Warn("navigator background: summary generation failed", "trace_id", req.TraceID, "error", err)
		return nil
	}

	text := extractResponseText(resp)
	if text == "" {
		return nil
	}

	var result models.NavigatorBackgroundResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		slog.Warn("navigator background: decode failed", "trace_id", req.TraceID, "error", err)
		return nil
	}
	result.Summary = strings.TrimSpace(result.Summary)
	if result.Summary == "" {
		return nil
	}
	if result.Surface == "" {
		result.Surface = inferSurface(req)
	}
	return &result
}

func buildBackgroundPrompt(req models.NavigatorBackgroundRequest) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Language: %s\n", lang.NormalizeLanguage(req.Language))
	fmt.Fprintf(&b, "Command: %s\n", strings.TrimSpace(req.Command))
	fmt.Fprintf(&b, "Outcome: %s\n", strings.TrimSpace(req.Outcome))
	fmt.Fprintf(&b, "Outcome detail: %s\n", truncate(req.OutcomeDetail, 240))
	fmt.Fprintf(&b, "Surface: %s\n", inferSurface(req))
	fmt.Fprintf(&b, "Initial app: %s\n", strings.TrimSpace(req.InitialAppName))
	fmt.Fprintf(&b, "Initial window: %s\n", strings.TrimSpace(req.InitialWindowTitle))
	for idx, step := range req.Steps {
		fmt.Fprintf(&b, "Step %d: %s %s -> %s (%s)\n", idx+1, step.ActionType, truncate(step.TargetApp, 60), truncate(step.ObservedOutcome, 120), step.ResultStatus)
	}
	b.WriteString(`Return JSON only in this schema:
{"summary":"1-2 sentence post-task summary","replayLabel":"short regression label","surface":"Antigravity|Terminal|Chrome|Other","tags":["tag1","tag2"]}`)
	return b.String()
}

func backgroundFallback(req models.NavigatorBackgroundRequest) *models.NavigatorBackgroundResult {
	surface := inferSurface(req)
	label := inferReplayLabel(req)
	summary := fmt.Sprintf("Navigator %s on %s for %q.", strings.TrimSpace(req.Outcome), surface, truncate(req.Command, 80))
	if detail := clean(req.OutcomeDetail); detail != "" {
		summary = fmt.Sprintf("%s %s", summary, detail)
	}
	return &models.NavigatorBackgroundResult{
		Summary:     summary,
		ReplayLabel: label,
		Surface:     surface,
		Tags:        backgroundTags(req, surface),
	}
}

func inferSurface(req models.NavigatorBackgroundRequest) string {
	surface := strings.TrimSpace(req.Surface)
	if surface != "" {
		return surface
	}
	for _, candidate := range []string{req.InitialAppName, req.Command} {
		lowered := strings.ToLower(candidate)
		switch {
		case strings.Contains(lowered, "chrome"):
			return "Chrome"
		case strings.Contains(lowered, "terminal"):
			return "Terminal"
		case strings.Contains(lowered, "antigravity"):
			return "Antigravity"
		}
	}
	return "Other"
}

func inferReplayLabel(req models.NavigatorBackgroundRequest) string {
	commandLower := strings.ToLower(req.Command)
	switch {
	case containsAny(commandLower, "docs", "documentation", "공식 문서"):
		return "chrome_docs_lookup"
	case containsAny(commandLower, "rerun", "run again", "다시 실행", "retry"):
		return "terminal_command_rerun"
	case containsAny(commandLower, "input", "search box", "검색창", "paste", "입력"):
		return "input_field_focus_and_text_insertion"
	case inferSurface(req) == "Antigravity":
		return "antigravity_failure_state"
	default:
		return "navigator_task_replay"
	}
}

func backgroundTags(req models.NavigatorBackgroundRequest, surface string) []string {
	tags := []string{
		"navigator",
		"outcome_" + cleanLower(req.Outcome),
		"surface_" + cleanLower(surface),
	}
	if containsAny(req.Command, "docs", "documentation", "공식 문서") {
		tags = append(tags, "docs_lookup")
	}
	if containsAny(req.Command, "input", "paste", "검색창", "입력") {
		tags = append(tags, "text_entry")
	}
	return unique(tags)
}

func backgroundHistory(req models.NavigatorBackgroundRequest, result *models.NavigatorBackgroundResult) []string {
	history := []string{
		"user: " + strings.TrimSpace(req.Command),
		"navigator_outcome: " + strings.TrimSpace(req.Outcome),
	}
	if result != nil && strings.TrimSpace(result.Summary) != "" {
		history = append(history, "assistant: "+strings.TrimSpace(result.Summary))
	}
	for _, step := range req.Steps {
		line := fmt.Sprintf("navigator_step: %s %s %s", step.ActionType, step.ResultStatus, strings.TrimSpace(step.ObservedOutcome))
		history = append(history, strings.TrimSpace(line))
	}
	return history
}

func buildResearchQuery(req models.NavigatorBackgroundRequest) string {
	switch inferReplayLabel(req) {
	case "chrome_docs_lookup":
		return strings.TrimSpace(req.Command)
	default:
		return strings.TrimSpace(req.Command) + " official docs"
	}
}

func shouldEnrichResearch(command string) bool {
	return containsAny(strings.ToLower(command), "docs", "documentation", "공식 문서", "reference", "latest")
}

func extractResponseText(resp *genai.GenerateContentResponse) string {
	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return ""
	}
	var b strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		b.WriteString(part.Text)
	}
	return strings.TrimSpace(strings.Trim(strings.TrimSpace(b.String()), "`"))
}

func descriptorLabel(raw string) string {
	for _, part := range strings.Split(raw, "|") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToLower(part), "label=") {
			return strings.TrimSpace(part[len("label="):])
		}
	}
	return ""
}

func looksLikeTextInputRole(role string) bool {
	lowered := strings.ToLower(strings.TrimSpace(role))
	return containsAny(lowered, "textfield", "textarea", "searchfield")
}

func truncate(raw string, limit int) string {
	raw = strings.TrimSpace(raw)
	if limit <= 0 || len(raw) <= limit {
		return raw
	}
	return strings.TrimSpace(raw[:limit]) + "..."
}

func clean(raw string) string {
	return strings.TrimSpace(raw)
}

func cleanLower(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	raw = strings.ReplaceAll(raw, " ", "_")
	return raw
}

func containsAny(text string, keywords ...string) bool {
	lowered := strings.ToLower(text)
	for _, keyword := range keywords {
		if strings.Contains(lowered, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func unique(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
