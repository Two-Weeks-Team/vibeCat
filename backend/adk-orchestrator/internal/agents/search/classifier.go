package search

import (
	"context"
	"log/slog"
	"strings"

	"google.golang.org/genai"
)

const classifierModel = "gemini-3.1-flash-lite-preview"

const classifyPrompt = `You are a search-intent classifier. Given user speech, decide if it requires a live web search.

Reply ONLY "YES" or "NO". Nothing else.

YES examples: news, weather, stock prices, current events, factual questions about the real world, "what is X", "search for Y"
NO examples: greetings, coding help, personal conversation, opinions, commands to the companion`

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

	resp, err := c.client.Models.GenerateContent(ctx, classifierModel, []*genai.Content{
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
