package search

import (
	"context"
	"log/slog"
	"strings"

	"google.golang.org/genai"

	"vibecat/adk-orchestrator/internal/geminiconfig"
)

const classifyPrompt = `You are a search-intent classifier for a developer assistant. Given user speech, decide if it would benefit from a live web search.

Reply ONLY "YES" or "NO". Nothing else.

Return YES for:
- current information such as news, weather, prices, releases, versions, dates, schedules, regulations
- factual questions about the real world
- coding questions that likely need external facts or examples: library APIs, framework behavior, error messages, installation issues, version compatibility, docs lookup, "what does this error mean", "how do I fix X", "search for Y"
- explicit requests to search, look up, find, browse, check docs, or verify

Return NO for:
- greetings or small talk
- purely personal conversation or opinions
- commands to the companion that do not need external information
- questions that can be answered from immediate local context alone`

type Classifier struct {
	client *genai.Client
}

func NewClassifier(client *genai.Client) *Classifier {
	return &Classifier{client: client}
}

func (c *Classifier) NeedsSearch(ctx context.Context, text string) bool {
	if c.client == nil {
		return false
	}

	resp, err := c.client.Models.GenerateContent(ctx, geminiconfig.LiteTextModel, []*genai.Content{
		{Parts: []*genai.Part{{Text: text}}, Role: genai.RoleUser},
	}, &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: classifyPrompt}},
		},
		MaxOutputTokens: 4,
		Temperature:     ptrFloat32(0),
	})
	if err != nil {
		slog.Warn("[CLASSIFIER] failed", "error", err, "text", truncateText(text, 60))
		return false
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return false
	}

	answer := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		answer += part.Text
	}
	result := strings.TrimSpace(strings.ToUpper(answer)) == "YES"
	slog.Info("[CLASSIFIER] result", "needs_search", result, "answer", strings.TrimSpace(answer), "text", truncateText(text, 60))
	return result
}

func ptrFloat32(v float32) *float32 { return &v }

func truncateText(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "..."
}
