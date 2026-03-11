package ws

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

type navigatorContext struct {
	AppName              string `json:"appName"`
	BundleID             string `json:"bundleId"`
	WindowTitle          string `json:"windowTitle"`
	FocusedRole          string `json:"focusedRole"`
	FocusedLabel         string `json:"focusedLabel"`
	SelectedText         string `json:"selectedText"`
	AXSnapshot           string `json:"axSnapshot"`
	AccessibilityTrusted bool   `json:"accessibilityTrusted"`
}

type navigatorTargetDescriptor struct {
	Role           string `json:"role,omitempty"`
	Label          string `json:"label,omitempty"`
	WindowTitle    string `json:"windowTitle,omitempty"`
	AppName        string `json:"appName,omitempty"`
	RelativeAnchor string `json:"relativeAnchor,omitempty"`
	RegionHint     string `json:"regionHint,omitempty"`
}

type navigatorStep struct {
	ID               string                    `json:"id"`
	ActionType       string                    `json:"actionType"`
	TargetApp        string                    `json:"targetApp"`
	TargetDescriptor navigatorTargetDescriptor `json:"targetDescriptor"`
	InputText        string                    `json:"inputText,omitempty"`
	ExpectedOutcome  string                    `json:"expectedOutcome"`
	Confidence       float64                   `json:"confidence"`
	IntentConfidence float64                   `json:"intentConfidence"`
	RiskLevel        string                    `json:"riskLevel"`
	ExecutionPolicy  string                    `json:"executionPolicy"`
	FallbackPolicy   string                    `json:"fallbackPolicy"`
	URL              string                    `json:"url,omitempty"`
	Hotkey           []string                  `json:"hotkey,omitempty"`
	VerifyHint       string                    `json:"verifyHint,omitempty"`
}

type navigatorIntentClass string

const (
	navigatorIntentExecuteNow    navigatorIntentClass = "execute_now"
	navigatorIntentOpenNavigate  navigatorIntentClass = "open_or_navigate"
	navigatorIntentFindLookup    navigatorIntentClass = "find_or_lookup"
	navigatorIntentAnalyzeOnly   navigatorIntentClass = "analyze_only"
	navigatorIntentAmbiguous     navigatorIntentClass = "ambiguous"
	navigatorExecutionPolicyLow                       = "safe_immediate"
)

type navigatorPlan struct {
	Command        string
	IntentClass    navigatorIntentClass
	IntentConfidence float64
	ClarifyQuestion string
	RiskQuestion    string
	RiskReason      string
	Steps           []navigatorStep
}

type navigatorSessionState struct {
	activeCommand               string
	pendingClarificationCommand string
	pendingRiskyCommand         string
	steps                       []navigatorStep
	nextStepIndex               int
}

func (s *navigatorSessionState) startPlan(command string, steps []navigatorStep) {
	s.activeCommand = command
	s.pendingClarificationCommand = ""
	s.pendingRiskyCommand = ""
	s.steps = steps
	s.nextStepIndex = 0
}

func (s *navigatorSessionState) clearPlan() {
	s.activeCommand = ""
	s.pendingClarificationCommand = ""
	s.pendingRiskyCommand = ""
	s.steps = nil
	s.nextStepIndex = 0
}

func (s *navigatorSessionState) nextStep() (navigatorStep, bool) {
	if s.nextStepIndex >= len(s.steps) {
		return navigatorStep{}, false
	}
	step := s.steps[s.nextStepIndex]
	s.nextStepIndex++
	return step, true
}

func (s *navigatorSessionState) hasRemainingSteps() bool {
	return s.nextStepIndex < len(s.steps)
}

func planNavigatorCommand(command string, ctx navigatorContext, allowRisky bool) navigatorPlan {
	intentClass, confidence, clarifyQuestion := classifyNavigatorIntent(command)
	plan := navigatorPlan{
		Command:          strings.TrimSpace(command),
		IntentClass:      intentClass,
		IntentConfidence: confidence,
		ClarifyQuestion:  clarifyQuestion,
	}

	if intentClass == navigatorIntentAmbiguous {
		return plan
	}

	if riskReason := navigatorRiskReason(command); riskReason != "" && !allowRisky {
		plan.RiskReason = riskReason
		plan.RiskQuestion = buildRiskQuestion(command)
		return plan
	}

	switch {
	case shouldUseDocsLookup(command):
		plan.Steps = buildDocsLookupSteps(command, ctx, confidence)
	case looksLikeDirectClick(command):
		if step, ok := buildAXPressStep(command, ctx, confidence); ok {
			plan.Steps = []navigatorStep{step}
		}
	case canUseTerminalCommand(command, ctx):
		plan.Steps = buildTerminalCommandSteps(command, ctx, confidence)
	case wantsAntigravityAction(command, ctx):
		plan.Steps = buildAntigravityInlineSteps(command, ctx, confidence)
	case intentClass == navigatorIntentAnalyzeOnly:
		plan.Steps = nil
	default:
		plan.ClarifyQuestion = defaultClarifyQuestion(command)
		plan.IntentClass = navigatorIntentAmbiguous
	}

	return plan
}

func classifyNavigatorIntent(command string) (navigatorIntentClass, float64, string) {
	lowered := strings.ToLower(strings.TrimSpace(command))
	if lowered == "" {
		return navigatorIntentAmbiguous, 0, defaultClarifyQuestion(command)
	}

	executeScore := keywordScore(lowered, []string{
		"apply", "do it", "run it", "rerun", "retry", "fix", "execute", "take care of",
		"반영", "적용", "실행", "다시 돌려", "다시 실행", "수정", "해결", "처리해", "눌러",
	})
	openScore := keywordScore(lowered, []string{
		"open", "go to", "take me", "bring me", "jump", "navigate", "show me",
		"열어", "이동", "데려가", "가보자", "보여", "가자",
	})
	findScore := keywordScore(lowered, []string{
		"find", "look up", "search", "docs", "official", "where is", "locate",
		"찾아", "검색", "공식 문서", "위치", "어디",
	})
	analyzeScore := keywordScore(lowered, []string{
		"explain", "summarize", "what is", "why", "how", "tell me about",
		"설명", "요약", "왜", "어떻게", "뭐야", "알려줘",
	})

	if strings.Contains(lowered, "test") && (strings.Contains(lowered, "again") || strings.Contains(lowered, "retry") || strings.Contains(lowered, "check")) {
		executeScore = maxFloat(executeScore, 0.78)
	}
	if strings.Contains(lowered, "테스트") && (strings.Contains(lowered, "확인") || strings.Contains(lowered, "다시")) {
		executeScore = maxFloat(executeScore, 0.78)
	}
	if strings.Contains(lowered, "반영해") || strings.Contains(lowered, "적용해") || strings.Contains(lowered, "실행해") ||
		strings.Contains(lowered, "처리해") || strings.Contains(lowered, "수정해") || strings.Contains(lowered, "눌러줘") ||
		strings.Contains(lowered, "다시 돌려") {
		executeScore = maxFloat(executeScore, 0.76)
	}
	if strings.Contains(lowered, "열어줘") || strings.Contains(lowered, "데려가") || strings.Contains(lowered, "가보자") {
		openScore = maxFloat(openScore, 0.72)
	}
	if strings.Contains(lowered, "찾아줘") || strings.Contains(lowered, "찾아봐") {
		findScore = maxFloat(findScore, 0.72)
	}
	if strings.Contains(lowered, "official docs") || strings.Contains(lowered, "공식 문서") {
		findScore = maxFloat(findScore, 0.88)
	}

	topClass := navigatorIntentAmbiguous
	topScore := 0.0
	secondScore := 0.0
	for _, candidate := range []struct {
		class navigatorIntentClass
		score float64
	}{
		{navigatorIntentExecuteNow, executeScore},
		{navigatorIntentOpenNavigate, openScore},
		{navigatorIntentFindLookup, findScore},
		{navigatorIntentAnalyzeOnly, analyzeScore},
	} {
		if candidate.score > topScore {
			secondScore = topScore
			topScore = candidate.score
			topClass = candidate.class
		} else if candidate.score > secondScore {
			secondScore = candidate.score
		}
	}

	if topScore < 0.58 {
		return navigatorIntentAmbiguous, topScore, defaultClarifyQuestion(command)
	}
	if topScore-secondScore < 0.12 && topClass != navigatorIntentAnalyzeOnly {
		return navigatorIntentAmbiguous, topScore, defaultClarifyQuestion(command)
	}
	return topClass, topScore, ""
}

func keywordScore(text string, keywords []string) float64 {
	score := 0.0
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			score += 0.28
		}
	}
	if score > 0.96 {
		return 0.96
	}
	return score
}

func navigatorRiskReason(command string) string {
	lowered := strings.ToLower(command)
	for _, keyword := range []string{"password", "token", "secret", "deploy", "production", "prod", "delete", "remove", "git push", "publish", "submit", "send", "비밀번호", "토큰", "배포", "삭제", "전송", "제출"} {
		if strings.Contains(lowered, keyword) {
			return fmt.Sprintf("blocked as a risky action because it mentions %q", keyword)
		}
	}
	return ""
}

func buildRiskQuestion(command string) string {
	return "This could change something important. Do you want me to proceed, or should I only explain the next step?"
}

func defaultClarifyQuestion(command string) string {
	return "Do you want me to act on this now, or just explain the next step?"
}

func shouldUseDocsLookup(command string) bool {
	lowered := strings.ToLower(command)
	return strings.Contains(lowered, "official") || strings.Contains(lowered, "docs") || strings.Contains(lowered, "documentation") || strings.Contains(lowered, "공식 문서")
}

func wantsAntigravityAction(command string, ctx navigatorContext) bool {
	lowered := strings.ToLower(command)
	app := strings.ToLower(ctx.AppName)
	return strings.Contains(app, "antigravity") ||
		strings.Contains(lowered, "antigravity") ||
		strings.Contains(lowered, "apply") ||
		strings.Contains(lowered, "반영") ||
		strings.Contains(lowered, "fix") ||
		strings.Contains(lowered, "수정") ||
		strings.Contains(lowered, "move me") ||
		strings.Contains(lowered, "데려가")
}

func canUseTerminalCommand(command string, ctx navigatorContext) bool {
	if extractCommandPayload(command) != "" {
		return true
	}
	return looksLikeShellCommand(ctx.SelectedText)
}

func looksLikeDirectClick(command string) bool {
	lowered := strings.ToLower(command)
	return strings.Contains(lowered, "click ") || strings.Contains(lowered, "press ") || strings.Contains(lowered, "눌러")
}

func buildDocsLookupSteps(command string, ctx navigatorContext, intentConfidence float64) []navigatorStep {
	steps := make([]navigatorStep, 0, 2)
	if !strings.Contains(strings.ToLower(ctx.AppName), "chrome") {
		steps = append(steps, navigatorStep{
			ID:               "focus_chrome",
			ActionType:       "focus_app",
			TargetApp:        "Chrome",
			TargetDescriptor: navigatorTargetDescriptor{AppName: "Chrome"},
			ExpectedOutcome:  "Chrome is ready for the docs lookup",
			Confidence:       0.96,
			IntentConfidence: intentConfidence,
			RiskLevel:        "low",
			ExecutionPolicy:  navigatorExecutionPolicyLow,
			FallbackPolicy:   "guided_mode",
			VerifyHint:       "chrome",
		})
	}

	query := buildDocsSearchQuery(command, ctx)
	steps = append(steps, navigatorStep{
		ID:         "open_docs_search",
		ActionType: "open_url",
		TargetApp:  "Chrome",
		TargetDescriptor: navigatorTargetDescriptor{
			AppName: "Chrome",
		},
		ExpectedOutcome:  "Chrome opens the relevant official docs search",
		Confidence:       0.93,
		IntentConfidence: intentConfidence,
		RiskLevel:        "low",
		ExecutionPolicy:  navigatorExecutionPolicyLow,
		FallbackPolicy:   "guided_mode",
		URL:              "https://www.google.com/search?q=" + url.QueryEscape(query),
		VerifyHint:       "google",
	})
	return steps
}

func buildDocsSearchQuery(command string, ctx navigatorContext) string {
	topic := cleanTopic(ctx.SelectedText)
	if topic == "" {
		topic = cleanTopic(ctx.WindowTitle)
	}
	if topic == "" {
		topic = cleanTopic(command)
	}
	if topic == "" {
		topic = "Antigravity IDE documentation"
	}
	return "site:developers.google.com OR site:ai.google.dev " + topic
}

func buildAntigravityInlineSteps(command string, ctx navigatorContext, intentConfidence float64) []navigatorStep {
	steps := make([]navigatorStep, 0, 3)
	if !strings.Contains(strings.ToLower(ctx.AppName), "antigravity") {
		steps = append(steps, navigatorStep{
			ID:               "focus_antigravity",
			ActionType:       "focus_app",
			TargetApp:        "Antigravity",
			TargetDescriptor: navigatorTargetDescriptor{AppName: "Antigravity"},
			ExpectedOutcome:  "Antigravity is frontmost",
			Confidence:       0.9,
			IntentConfidence: intentConfidence,
			RiskLevel:        "low",
			ExecutionPolicy:  navigatorExecutionPolicyLow,
			FallbackPolicy:   "guided_mode",
			VerifyHint:       "antigravity",
		})
	}

	prompt := buildAntigravityPrompt(command, ctx)
	steps = append(steps,
		navigatorStep{
			ID:               "open_antigravity_inline_prompt",
			ActionType:       "hotkey",
			TargetApp:        "Antigravity",
			TargetDescriptor: navigatorTargetDescriptor{AppName: "Antigravity", WindowTitle: ctx.WindowTitle},
			ExpectedOutcome:  "Antigravity is ready to receive an inline instruction",
			Confidence:       0.82,
			IntentConfidence: intentConfidence,
			RiskLevel:        "low",
			ExecutionPolicy:  navigatorExecutionPolicyLow,
			FallbackPolicy:   "guided_mode",
			Hotkey:           []string{"command", "i"},
		},
		navigatorStep{
			ID:               "paste_antigravity_instruction",
			ActionType:       "paste_text",
			TargetApp:        "Antigravity",
			TargetDescriptor: navigatorTargetDescriptor{AppName: "Antigravity", WindowTitle: ctx.WindowTitle},
			InputText:        prompt,
			ExpectedOutcome:  "Antigravity receives the requested navigation or apply instruction",
			Confidence:       0.8,
			IntentConfidence: intentConfidence,
			RiskLevel:        "low",
			ExecutionPolicy:  navigatorExecutionPolicyLow,
			FallbackPolicy:   "guided_mode",
		},
		navigatorStep{
			ID:               "submit_antigravity_instruction",
			ActionType:       "hotkey",
			TargetApp:        "Antigravity",
			TargetDescriptor: navigatorTargetDescriptor{AppName: "Antigravity", WindowTitle: ctx.WindowTitle},
			ExpectedOutcome:  "Antigravity starts the requested action",
			Confidence:       0.76,
			IntentConfidence: intentConfidence,
			RiskLevel:        "low",
			ExecutionPolicy:  navigatorExecutionPolicyLow,
			FallbackPolicy:   "guided_mode",
			Hotkey:           []string{"return"},
		},
	)
	return steps
}

func buildAntigravityPrompt(command string, ctx navigatorContext) string {
	contextBits := make([]string, 0, 3)
	if selected := cleanTopic(ctx.SelectedText); selected != "" {
		contextBits = append(contextBits, "Context: "+selected)
	}
	if title := cleanTopic(ctx.WindowTitle); title != "" {
		contextBits = append(contextBits, "Window: "+title)
	}
	base := fmt.Sprintf("Please help with this request inside the current project: %s", strings.TrimSpace(command))
	if len(contextBits) == 0 {
		return base
	}
	return base + "\n" + strings.Join(contextBits, "\n")
}

func buildTerminalCommandSteps(command string, ctx navigatorContext, intentConfidence float64) []navigatorStep {
	payload := extractCommandPayload(command)
	if payload == "" {
		payload = cleanTopic(ctx.SelectedText)
	}
	if payload == "" {
		return nil
	}

	steps := make([]navigatorStep, 0, 3)
	if !strings.Contains(strings.ToLower(ctx.AppName), "terminal") {
		steps = append(steps, navigatorStep{
			ID:               "focus_terminal",
			ActionType:       "focus_app",
			TargetApp:        "Terminal",
			TargetDescriptor: navigatorTargetDescriptor{AppName: "Terminal"},
			ExpectedOutcome:  "Terminal is frontmost",
			Confidence:       0.93,
			IntentConfidence: intentConfidence,
			RiskLevel:        "low",
			ExecutionPolicy:  navigatorExecutionPolicyLow,
			FallbackPolicy:   "guided_mode",
			VerifyHint:       "terminal",
		})
	}
	steps = append(steps,
		navigatorStep{
			ID:               "paste_terminal_command",
			ActionType:       "paste_text",
			TargetApp:        "Terminal",
			TargetDescriptor: navigatorTargetDescriptor{AppName: "Terminal", WindowTitle: ctx.WindowTitle},
			InputText:        payload,
			ExpectedOutcome:  "The command is ready in Terminal",
			Confidence:       0.88,
			IntentConfidence: intentConfidence,
			RiskLevel:        "low",
			ExecutionPolicy:  navigatorExecutionPolicyLow,
			FallbackPolicy:   "guided_mode",
		},
		navigatorStep{
			ID:               "submit_terminal_command",
			ActionType:       "hotkey",
			TargetApp:        "Terminal",
			TargetDescriptor: navigatorTargetDescriptor{AppName: "Terminal", WindowTitle: ctx.WindowTitle},
			ExpectedOutcome:  "Terminal runs the requested command",
			Confidence:       0.88,
			IntentConfidence: intentConfidence,
			RiskLevel:        "low",
			ExecutionPolicy:  navigatorExecutionPolicyLow,
			FallbackPolicy:   "guided_mode",
			Hotkey:           []string{"return"},
			VerifyHint:       "terminal",
		},
	)
	return steps
}

func buildAXPressStep(command string, ctx navigatorContext, intentConfidence float64) (navigatorStep, bool) {
	label := extractClickLabel(command)
	if label == "" {
		return navigatorStep{}, false
	}
	return navigatorStep{
		ID:         "press_ax_target",
		ActionType: "press_ax",
		TargetApp:  ctx.AppName,
		TargetDescriptor: navigatorTargetDescriptor{
			AppName:        ctx.AppName,
			WindowTitle:    ctx.WindowTitle,
			Label:          label,
			RelativeAnchor: cleanTopic(ctx.FocusedLabel),
		},
		ExpectedOutcome:  fmt.Sprintf("Press the %s control", label),
		Confidence:       0.7,
		IntentConfidence: intentConfidence,
		RiskLevel:        "low",
		ExecutionPolicy:  navigatorExecutionPolicyLow,
		FallbackPolicy:   "guided_mode",
		VerifyHint:       strings.ToLower(label),
	}, true
}

func cleanTopic(raw string) string {
	cleaned := strings.TrimSpace(raw)
	if cleaned == "" {
		return ""
	}
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	cleaned = strings.ReplaceAll(cleaned, "\t", " ")
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	if len(cleaned) > 120 {
		cleaned = cleaned[:120]
	}
	return strings.TrimSpace(cleaned)
}

var commandLiteralPattern = regexp.MustCompile("`([^`]+)`")

func extractCommandPayload(command string) string {
	if matches := commandLiteralPattern.FindStringSubmatch(command); len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}
	lowered := strings.ToLower(command)
	if idx := strings.Index(lowered, "run "); idx >= 0 {
		rest := strings.TrimSpace(command[idx+4:])
		if strings.Contains(rest, " ") {
			return rest
		}
	}
	return ""
}

func looksLikeShellCommand(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	for _, prefix := range []string{"go ", "npm ", "pnpm ", "yarn ", "swift ", "xcodebuild ", "python ", "pytest ", "make ", "cargo ", "git "} {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return strings.Contains(trimmed, "./") || strings.Contains(trimmed, " --")
}

func extractClickLabel(command string) string {
	command = strings.TrimSpace(command)
	if strings.Contains(command, "\"") {
		parts := strings.Split(command, "\"")
		if len(parts) >= 3 {
			return cleanTopic(parts[1])
		}
	}
	lowered := strings.ToLower(command)
	for _, marker := range []string{"click ", "press ", "눌러 "} {
		if idx := strings.Index(lowered, marker); idx >= 0 {
			return cleanTopic(command[idx+len(marker):])
		}
	}
	return ""
}

func navigatorMessageForStep(step navigatorStep) string {
	switch step.ActionType {
	case "focus_app":
		return fmt.Sprintf("I can switch to %s now.", step.TargetApp)
	case "open_url":
		return "I can open the relevant docs now."
	case "press_ax":
		return "I found a likely control and can act on it now."
	case "paste_text":
		return "I can insert the next instruction now."
	case "hotkey":
		return "I can trigger the next UI step now."
	default:
		return "I have the next step."
	}
}

func affirmativeAnswer(answer string) bool {
	lowered := strings.ToLower(strings.TrimSpace(answer))
	for _, keyword := range []string{"yes", "y", "do it", "go ahead", "proceed", "apply", "run it", "맞아", "응", "그래", "진행", "해줘", "실행"} {
		if strings.Contains(lowered, keyword) {
			return true
		}
	}
	return false
}

func explanationAnswer(answer string) bool {
	lowered := strings.ToLower(strings.TrimSpace(answer))
	for _, keyword := range []string{"explain", "just explain", "tell me", "설명", "말만", "방법만", "안내만"} {
		if strings.Contains(lowered, keyword) {
			return true
		}
	}
	return false
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
