package search

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
	"vibecat/adk-orchestrator/internal/prompts"
)

// searchKeywords are terms that trigger automatic search.
var searchKeywords = []string{
	"how to", "how do", "error:", "exception:", "undefined", "not found",
	"cannot", "can't", "failed", "failure", "why is", "what is",
	"npm install", "go get", "pip install", "import error",
}

const searchModel = "gemini-3.1-flash-lite-preview"

type Agent struct {
	genaiClient *genai.Client
	classifier  *Classifier
}

func New(client *genai.Client) *Agent {
	return &Agent{
		genaiClient: client,
		classifier:  NewClassifier(client),
	}
}

func (a *Agent) Run(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
	return func(yield func(*session.Event, error) bool) {
		userContent := ctx.UserContent()
		if userContent == nil {
			yield(nil, fmt.Errorf("search agent: no user content"))
			return
		}

		var result models.AnalysisResult
		for _, part := range userContent.Parts {
			if part.Text != "" {
				_ = json.Unmarshal([]byte(part.Text), &result)
			}
		}

		// Determine if search should be triggered
		if !shouldSearch(result) {
			// Pass through without searching
			data, err := json.Marshal(result)
			if err != nil {
				yield(nil, fmt.Errorf("search marshal: %w", err))
				return
			}
			yield(&session.Event{
				LLMResponse: model.LLMResponse{
					Content: &genai.Content{
						Parts: []*genai.Part{{Text: string(data)}},
					},
				},
			}, nil)
			return
		}

		query := buildQuery(result)
		searchResult := a.search(ctx, query, result, readLanguageFromState(ctx))
		result.Search = searchResult

		// If search found something useful, update speech text
		if searchResult != nil && searchResult.Summary != "" {
			result.SpeechText = searchResult.Summary
			if result.Decision == nil {
				result.Decision = &models.MediatorDecision{}
			}
			result.Decision.ShouldSpeak = true
			result.Decision.Reason = "search_result"
		}

		data, err := json.Marshal(result)
		if err != nil {
			yield(nil, fmt.Errorf("search marshal: %w", err))
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

// shouldSearch returns true if the current analysis warrants a Google Search.
func shouldSearch(result models.AnalysisResult) bool {
	// Auto-search when mood is stuck
	if result.Mood != nil && result.Mood.Mood == models.MoodStuck {
		return true
	}
	// Auto-search when mood is frustrated and error detected
	if result.Mood != nil && result.Mood.Mood == models.MoodFrustrated &&
		result.Vision != nil && result.Vision.ErrorDetected {
		return true
	}
	// Search when error message contains search keywords
	if result.Vision != nil && result.Vision.ErrorMessage != "" {
		lower := strings.ToLower(result.Vision.ErrorMessage)
		for _, kw := range searchKeywords {
			if strings.Contains(lower, kw) {
				return true
			}
		}
	}
	// Search when context contains search keywords
	if result.Vision != nil && result.Vision.Content != "" {
		lower := strings.ToLower(result.Vision.Content)
		for _, kw := range searchKeywords {
			if strings.Contains(lower, kw) {
				return true
			}
		}
	}
	return false
}

// buildQuery constructs a search query from the analysis result.
func buildQuery(result models.AnalysisResult) string {
	if result.Vision != nil && result.Vision.ErrorMessage != "" {
		return result.Vision.ErrorMessage
	}
	if result.Vision != nil && result.Vision.Content != "" {
		// Truncate to reasonable query length
		content := result.Vision.Content
		if len(content) > 200 {
			content = content[:200]
		}
		return content
	}
	return "developer error solution"
}

// search performs a Google Search grounded query via Gemini.
func (a *Agent) search(ctx agent.InvocationContext, query string, result models.AnalysisResult, language string) *models.SearchResult {
	if a.genaiClient == nil {
		return &models.SearchResult{
			Query:   query,
			Summary: "Search unavailable (no client configured)",
		}
	}

	prompt := fmt.Sprintf("Search for solutions to this developer problem: %s\n\nProvide a concise, actionable answer. Respond in %s.", query, normalizeLanguage(language))

	resp, err := a.genaiClient.Models.GenerateContent(ctx, "gemini-3.1-pro-preview", []*genai.Content{
		{Parts: []*genai.Part{{Text: prompt}}, Role: genai.RoleUser},
	}, &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: buildSystemPrompt(language)}},
		},
		Tools: []*genai.Tool{
			{GoogleSearch: &genai.GoogleSearch{}},
		},
	})
	if err != nil {
		slog.Warn("search agent gemini call failed", "error", err, "query", query)
		return &models.SearchResult{
			Query:   query,
			Summary: "Search failed: " + err.Error(),
		}
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return &models.SearchResult{Query: query, Summary: "No results found"}
	}

	text := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		text += part.Text
	}

	// Try to parse as JSON first
	var sr models.SearchResult
	if err := json.Unmarshal([]byte(text), &sr); err == nil {
		if sr.Query == "" {
			sr.Query = query
		}
		return &sr
	}

	// Fallback: use raw text as summary
	summary := text
	if len(summary) > 300 {
		summary = summary[:300] + "..."
	}
	return &models.SearchResult{
		Query:   query,
		Summary: summary,
	}
}

func (a *Agent) DirectSearch(ctx context.Context, query, language string) *models.SearchResult {
	if a.genaiClient == nil {
		return nil
	}
	lang := normalizeLanguage(language)

	if !a.classifier.NeedsSearch(ctx, query) {
		slog.Info("[SEARCH] classifier rejected", "query", truncateText(query, 60))
		return nil
	}

	slog.Info("[SEARCH] executing search", "query", truncateText(query, 60), "model", searchModel)
	return a.executeSearch(ctx, query, lang)
}

func (a *Agent) executeSearch(ctx context.Context, query, language string) *models.SearchResult {
	prompt := fmt.Sprintf("Search and answer: %s\nRespond in %s. Be concise (2-3 sentences).", query, language)
	resp, err := a.genaiClient.Models.GenerateContent(ctx, searchModel, []*genai.Content{
		{Parts: []*genai.Part{{Text: prompt}}, Role: genai.RoleUser},
	}, &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: buildSearchPrompt(language)}},
		},
		MaxOutputTokens: 300,
		Tools:           []*genai.Tool{{GoogleSearch: &genai.GoogleSearch{}}},
	})
	if err != nil {
		slog.Warn("[SEARCH] execution failed", "error", err, "query", query)
		return &models.SearchResult{Query: query, Summary: "Search failed"}
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return &models.SearchResult{Query: query, Summary: "No results found"}
	}

	text := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		text += part.Text
	}

	var sr models.SearchResult
	if err := json.Unmarshal([]byte(text), &sr); err == nil {
		if sr.Query == "" {
			sr.Query = query
		}
		slog.Info("[SEARCH] result parsed", "query", sr.Query, "summary_len", len(sr.Summary))
		return &sr
	}

	if len(text) > 500 {
		text = text[:500] + "..."
	}
	return &models.SearchResult{Query: query, Summary: text}
}

func buildSearchPrompt(language string) string {
	return fmt.Sprintf(`You are a fast search assistant. Search the web and provide a concise answer.
Return JSON: {"query":"...","summary":"...","sources":["url1"]}
Summary must be 2-3 natural sentences suitable for speech. Respond in %s. Return only valid JSON.`, language)
}

func buildSystemPrompt(language string) string {
	return fmt.Sprintf(`You are SearchBuddy in VibeCat.
%s

When given a developer's problem or error message, search for relevant solutions.
Return JSON with:
- query: the query you used
- summary: concise, actionable findings (2-3 sentences)
- sources: relevant URLs (up to 3)

Respond in %s.
Return only valid JSON without markdown.`, prompts.ProjectPurpose, normalizeLanguage(language))
}

func readLanguageFromState(ctx agent.InvocationContext) string {
	sess := ctx.Session()
	if sess == nil || sess.State() == nil {
		return "Korean"
	}
	val, err := sess.State().Get("language")
	if err != nil {
		return "Korean"
	}
	if s, ok := val.(string); ok {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			return trimmed
		}
	}
	return "Korean"
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
