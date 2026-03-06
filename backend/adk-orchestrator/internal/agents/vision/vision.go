package vision

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"
	"strings"
	"sync"
	"time"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"vibecat/adk-orchestrator/internal/models"
	"vibecat/adk-orchestrator/internal/prompts"
)

const visionModel = "gemini-3.1-flash-lite-preview"

func buildSystemPrompt(character, soul, language string) string {
	var sb strings.Builder
	language = normalizeLanguage(language)

	sb.WriteString(prompts.ProjectPurpose)
	sb.WriteString("\nYou are the screen analysis agent in this system.\n\n")

	sb.WriteString(prompts.CommonBehavior)
	sb.WriteString("\n\n")

	if soul != "" {
		sb.WriteString("=== CHARACTER PERSONA ===\n")
		sb.WriteString(soul)
		sb.WriteString("\n\n")
	}

	if character != "" {
		sb.WriteString(fmt.Sprintf("Character name: \"%s\". Keep this identity in tone, but do not sacrifice technical accuracy.\n\n", character))
	}

	sb.WriteString(`Analyze the provided screenshot and return JSON using this schema:

Required fields:
- significance (0-10): how important this screen event is
- content: what VibeCat should say to help the developer
- emotion: one of neutral, curious, surprised, happy, concerned
- shouldSpeak: true when VibeCat should speak now (error found, meaningful success, prolonged stuck state, etc.)
- errorDetected: true if an error/exception/failing test is visible
- repeatedError: true if it appears to be the same recurring error
- successDetected: true if signs like "tests passed", "build succeeded", "deployed", or "PR merged" are visible
- errorMessage: original error text when errorDetected is true

Significance scoring:
- 0-2: little or no change, idle screen, minor cursor movement
- 3-5: regular coding, small edits, normal navigation
- 6: meaningful code change, new file opened
- 7+: terminal output flood (build logs, test output, npm install, large compile output, rapid scrolling text)
- 8+: error visible (compile error, stack trace, red text, failing test)
- 9+: success event (tests passed, build succeeded, deployment complete)

IMPORTANT: Terminal output floods, build logs, large amounts of scrolling text, and rapid terminal changes are significant events (7+). Developers need to know when builds finish, tests complete, or large operations end. Do NOT score terminal activity as "regular coding" (3-5).

Rules:
- Read concrete details on screen (variable names, function names, stack traces, test output)
- Give actionable insight, not generic narration
- If there is an error, include likely cause and practical next step
- Use the character card only for style; keep technical guidance clear and correct
- Keep content concise for a speech bubble (around 100 characters is recommended, but complete sentences matter most)
- Respond in `)
	sb.WriteString(language)
	sb.WriteString(`.

Return JSON only. Do not use markdown code fences.`)

	return sb.String()
}

type Agent struct {
	mu             sync.Mutex
	genaiClient    *genai.Client
	errorHistory   []string
	successHistory []string
}

func New(client *genai.Client) *Agent {
	return &Agent{genaiClient: client}
}

func (a *Agent) Run(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
	return func(yield func(*session.Event, error) bool) {
		userContent := ctx.UserContent()
		if userContent == nil {
			yield(nil, fmt.Errorf("vision agent: no user content"))
			return
		}

		var req models.AnalysisRequest
		for _, part := range userContent.Parts {
			if part.Text != "" {
				if err := json.Unmarshal([]byte(part.Text), &req); err == nil && req.Image != "" {
					break
				}
			}
		}

		analysis := a.analyze(ctx, &req)

		data, err := json.Marshal(analysis)
		if err != nil {
			yield(nil, fmt.Errorf("vision agent marshal: %w", err))
			return
		}

		yield(&session.Event{
			Actions: session.EventActions{
				StateDelta: map[string]any{
					"vision_analysis": string(data),
					"language":        normalizeLanguage(req.Language),
				},
			},
			LLMResponse: model.LLMResponse{
				Content: &genai.Content{
					Parts: []*genai.Part{{Text: string(data)}},
				},
			},
		}, nil)
	}
}

func (a *Agent) analyze(ctx context.Context, req *models.AnalysisRequest) *models.VisionAnalysis {
	if a.genaiClient == nil || req.Image == "" {
		return &models.VisionAnalysis{
			Significance: 3,
			Content:      req.Context,
			Emotion:      "neutral",
			ShouldSpeak:  false,
		}
	}

	parts := []*genai.Part{{Text: fmt.Sprintf("Current context: %s\n\nAnalyze this screenshot:", req.Context)}}

	decoded, decErr := base64.StdEncoding.DecodeString(req.Image)
	if decErr != nil {
		slog.Warn("vision agent base64 decode failed", "error", decErr)
		return &models.VisionAnalysis{Significance: 3, Content: "Image decode failed", Emotion: "neutral"}
	}
	parts = append(parts, &genai.Part{
		InlineData: &genai.Blob{
			MIMEType: "image/jpeg",
			Data:     decoded,
		},
	})

	language := normalizeLanguage(req.Language)
	sysPrompt := buildSystemPrompt(req.Character, req.Soul, language)

	slog.Info("[VISION] calling Gemini",
		"model", visionModel,
		"character", req.Character,
		"context", req.Context,
		"image_bytes", len(decoded),
		"prompt_len", len(sysPrompt),
	)

	startTime := time.Now()
	resp, err := a.genaiClient.Models.GenerateContent(ctx, visionModel, []*genai.Content{
		{Parts: parts, Role: genai.RoleUser},
	}, &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		MaxOutputTokens:  500,
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: sysPrompt}},
		},
	})
	elapsed := time.Since(startTime)
	if err != nil {
		slog.Warn("[VISION] Gemini call FAILED", "error", err, "model", visionModel, "elapsed", elapsed.String())
		return &models.VisionAnalysis{Significance: 3, Content: "Screen analysis unavailable", Emotion: "neutral"}
	}

	slog.Info("[VISION] Gemini responded", "elapsed", elapsed.String(), "candidates", len(resp.Candidates))

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		slog.Warn("[VISION] empty response from Gemini")
		return &models.VisionAnalysis{Significance: 0, Emotion: "neutral"}
	}

	text := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		text += part.Text
	}

	text = stripCodeFence(text)
	slog.Info("[VISION] raw response", "text", text[:min(200, len(text))])

	var analysis models.VisionAnalysis
	if err := json.Unmarshal([]byte(text), &analysis); err != nil {
		slog.Warn("[VISION] JSON parse FAILED", "error", err, "text", text[:min(100, len(text))])
		analysis = models.VisionAnalysis{
			Significance: 3,
			Content:      text,
			Emotion:      "neutral",
		}
	}

	if analysis.ErrorDetected && analysis.ErrorMessage != "" {
		analysis.RepeatedError = a.isRepeatedError(analysis.ErrorMessage)
		a.trackError(analysis.ErrorMessage)
	}
	if analysis.SuccessDetected && analysis.Content != "" {
		analysis.RepeatedSuccess = a.isRepeatedSuccess(analysis.Content)
		a.trackSuccess(analysis.Content)
		if analysis.RepeatedSuccess {
			analysis.ShouldSpeak = false
			slog.Info("[VISION] repeated success suppressed", "content", analysis.Content[:min(80, len(analysis.Content))])
		}
	}

	slog.Info("[VISION] analysis result",
		"significance", analysis.Significance,
		"emotion", analysis.Emotion,
		"shouldSpeak", analysis.ShouldSpeak,
		"errorDetected", analysis.ErrorDetected,
		"repeatedError", analysis.RepeatedError,
		"successDetected", analysis.SuccessDetected,
		"content", analysis.Content[:min(80, len(analysis.Content))],
	)

	return &analysis
}

func (a *Agent) isRepeatedError(msg string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	count := 0
	for _, h := range a.errorHistory {
		if strings.Contains(h, msg[:min(30, len(msg))]) {
			count++
		}
	}
	return count >= 2
}

func (a *Agent) trackError(msg string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.errorHistory) > 20 {
		a.errorHistory = a.errorHistory[1:]
	}
	a.errorHistory = append(a.errorHistory, msg)
}

func (a *Agent) isRepeatedSuccess(content string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	count := 0
	for _, h := range a.successHistory {
		if strings.Contains(h, content[:min(30, len(content))]) {
			count++
		}
	}
	return count >= 2
}

func (a *Agent) trackSuccess(content string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.successHistory) > 20 {
		a.successHistory = a.successHistory[1:]
	}
	a.successHistory = append(a.successHistory, content)
}

func stripCodeFence(s string) string {
	t := strings.TrimSpace(s)
	if strings.HasPrefix(t, "```") {
		if i := strings.Index(t, "\n"); i != -1 {
			t = t[i+1:]
		}
	}
	if strings.HasSuffix(t, "```") {
		t = t[:len(t)-3]
	}
	return strings.TrimSpace(t)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func normalizeLanguage(language string) string {
	trimmed := strings.TrimSpace(language)
	if trimmed == "" {
		return "Korean"
	}
	lower := strings.ToLower(trimmed)
	switch lower {
	case "ko", "kr", "korean", "korean language", "한국어":
		return "Korean"
	case "en", "eng", "english", "english language":
		return "English"
	default:
		return trimmed
	}
}
