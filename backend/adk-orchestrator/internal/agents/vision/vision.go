package vision

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"
	"strings"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"vibecat/adk-orchestrator/internal/models"
)

const systemPrompt = `You are a screen analysis agent for a developer companion app.
Analyze the provided screenshot and return a JSON object with these fields:
- significance (0-10): how important/interesting is what's on screen
- content: brief description of what you see
- emotion: one of: neutral, curious, surprised, happy, concerned
- shouldSpeak: true if the developer would benefit from a comment
- errorDetected: true if you see an error, exception, or failed test
- repeatedError: true if this looks like the same error seen before
- successDetected: true if you see "tests passed", "build succeeded", "deployed", "PR merged", or green CI
- errorMessage: the error message text if errorDetected is true

Return ONLY valid JSON, no markdown.`

type Agent struct {
	genaiClient  *genai.Client
	errorHistory []string
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

		var imageData string
		var contextText string
		for _, part := range userContent.Parts {
			if part.Text != "" {
				var req models.AnalysisRequest
				if err := json.Unmarshal([]byte(part.Text), &req); err == nil {
					imageData = req.Image
					contextText = req.Context
				}
			}
		}

		analysis := a.analyze(ctx, imageData, contextText)

		data, err := json.Marshal(analysis)
		if err != nil {
			yield(nil, fmt.Errorf("vision agent marshal: %w", err))
			return
		}

		yield(&session.Event{
			LLMResponse: model.LLMResponse{
				Content: &genai.Content{
					Parts: []*genai.Part{{Text: string(data)}},
				},
			},
		}, nil)
	}
}

func (a *Agent) analyze(ctx context.Context, imageData, contextText string) *models.VisionAnalysis {
	if a.genaiClient == nil || imageData == "" {
		return &models.VisionAnalysis{
			Significance: 3,
			Content:      contextText,
			Emotion:      "neutral",
			ShouldSpeak:  false,
		}
	}

	parts := []*genai.Part{
		{Text: fmt.Sprintf("Context: %s\n\nAnalyze this screenshot:", contextText)},
	}

	if imageData != "" {
		parts = append(parts, &genai.Part{
			InlineData: &genai.Blob{
				MIMEType: "image/jpeg",
				Data:     []byte(imageData),
			},
		})
	}

	resp, err := a.genaiClient.Models.GenerateContent(ctx, "gemini-2.0-flash", []*genai.Content{
		{Parts: parts, Role: genai.RoleUser},
	}, &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: systemPrompt}},
		},
	})
	if err != nil {
		slog.Warn("vision agent gemini call failed", "error", err)
		return &models.VisionAnalysis{
			Significance: 3,
			Content:      "Screen analysis unavailable",
			Emotion:      "neutral",
		}
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return &models.VisionAnalysis{Significance: 0, Emotion: "neutral"}
	}

	text := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		text += part.Text
	}

	var analysis models.VisionAnalysis
	if err := json.Unmarshal([]byte(text), &analysis); err != nil {
		slog.Warn("vision agent parse failed", "error", err, "text", text[:min(100, len(text))])
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

	analysis.SuccessDetected = analysis.SuccessDetected || detectSuccess(contextText)

	return &analysis
}

func (a *Agent) isRepeatedError(msg string) bool {
	count := 0
	for _, h := range a.errorHistory {
		if strings.Contains(h, msg[:min(30, len(msg))]) {
			count++
		}
	}
	return count >= 2
}

func (a *Agent) trackError(msg string) {
	if len(a.errorHistory) > 20 {
		a.errorHistory = a.errorHistory[1:]
	}
	a.errorHistory = append(a.errorHistory, msg)
}

func detectSuccess(text string) bool {
	lower := strings.ToLower(text)
	patterns := []string{"tests passed", "build succeeded", "deployed", "pr merged", "all tests"}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
