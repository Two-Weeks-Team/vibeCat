package search

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/adk/tool/geminitool"
	"google.golang.org/genai"

	"vibecat/adk-orchestrator/internal/lang"
)

// FormatSearchResultInput is the typed input for the format_search_result function tool.
type FormatSearchResultInput struct {
	// Query is the original search query.
	Query string `json:"query" jsonschema:"description=The original search query"`
	// Summary is a concise summary of the search results (2-3 sentences).
	Summary string `json:"summary" jsonschema:"description=Concise actionable summary of search results in 2-3 sentences"`
	// Sources are URLs of the sources used.
	Sources []string `json:"sources" jsonschema:"description=URLs of the sources used (up to 3)"`
}

// FormatSearchResultOutput is the typed output for the format_search_result function tool.
type FormatSearchResultOutput struct {
	// Formatted is the formatted search result ready for speech.
	Formatted string `json:"formatted"`
	// Query is the original query echoed back.
	Query string `json:"query"`
	// SourceCount is the number of sources found.
	SourceCount int `json:"source_count"`
}

// formatSearchResult is a typed function tool that formats search results for speech output.
func formatSearchResult(ctx tool.Context, input FormatSearchResultInput) (FormatSearchResultOutput, error) {
	slog.Info("[SEARCH-TOOL] formatting result",
		"query", truncateText(input.Query, 60),
		"summary_len", len(input.Summary),
		"sources", len(input.Sources),
	)

	formatted := input.Summary
	if len(formatted) > 300 {
		formatted = formatted[:300] + "..."
	}

	return FormatSearchResultOutput{
		Formatted:   formatted,
		Query:       input.Query,
		SourceCount: len(input.Sources),
	}, nil
}

// NewLLMSearchAgent creates an llmagent-based search agent that uses
// geminitool.GoogleSearch for native Google Search grounding and
// functiontool for typed result formatting.
//
// This demonstrates ADK's llmagent, geminitool, and functiontool packages.
func NewLLMSearchAgent(apiKey string, language string) (agent.Agent, error) {
	// Create typed function tool for formatting search results
	formatTool, err := functiontool.New(
		functiontool.Config{
			Name:        "format_search_result",
			Description: "Formats search results into a concise summary suitable for speech output. Call this after finding search results to format them for the user.",
		},
		formatSearchResult,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create format tool: %w", err)
	}

	// Create Gemini model via ADK model/gemini package
	llm, err := gemini.NewModel(context.Background(), searchModel, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini model for search: %w", err)
	}

	lang := lang.NormalizeLanguage(language)

	searchAgent, err := llmagent.New(llmagent.Config{
		Name:        "llm_search_buddy",
		Description: "LLM-based search agent with Google Search grounding and typed function tools",
		Model:       llm,
		Instruction: fmt.Sprintf(`You are SearchBuddy in VibeCat, an AI coding companion for solo developers.

When given a developer's problem or error message:
1. Use Google Search to find relevant solutions
2. Call format_search_result with the query, a concise summary (2-3 sentences), and source URLs
3. Respond with the formatted result

Be concise and actionable. Focus on practical solutions.
Respond in %s.`, lang),
		Tools: []tool.Tool{
			geminitool.GoogleSearch{},
			formatTool,
		},
		OutputKey: "search_result",
		BeforeModelCallbacks: []llmagent.BeforeModelCallback{
			func(ctx agent.CallbackContext, req *model.LLMRequest) (*model.LLMResponse, error) {
				slog.Info("[ADK] LLM call started", "agent", ctx.AgentName(), "contents", len(req.Contents))
				return nil, nil
			},
		},
		AfterModelCallbacks: []llmagent.AfterModelCallback{
			func(ctx agent.CallbackContext, resp *model.LLMResponse, respErr error) (*model.LLMResponse, error) {
				if respErr != nil {
					slog.Warn("[ADK] LLM call failed", "agent", ctx.AgentName(), "error", respErr)
				} else {
					slog.Info("[ADK] LLM call completed", "agent", ctx.AgentName())
				}
				return nil, nil
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create llm search agent: %w", err)
	}

	return searchAgent, nil
}
