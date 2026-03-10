// Package memory implements the MemoryAgent for cross-session context.
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"vibecat/adk-orchestrator/internal/geminiconfig"
	"vibecat/adk-orchestrator/internal/lang"
	"vibecat/adk-orchestrator/internal/models"
	"vibecat/adk-orchestrator/internal/store"
)

// Agent implements cross-session memory: retrieves context at session start
// and writes summaries at session end.
type Agent struct {
	genaiClient *genai.Client
	store       *store.Client
}

// New creates a MemoryAgent. Both genaiClient and store may be nil (stub mode).
func New(genaiClient *genai.Client, storeClient *store.Client) *Agent {
	return &Agent{
		genaiClient: genaiClient,
		store:       storeClient,
	}
}

func (a *Agent) Run(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
	return func(yield func(*session.Event, error) bool) {
		userContent := ctx.UserContent()
		if userContent == nil {
			yield(nil, fmt.Errorf("memory agent: no user content"))
			return
		}

		var req models.AnalysisRequest
		var result models.AnalysisResult
		for _, part := range userContent.Parts {
			if part.Text != "" {
				// Try to parse as AnalysisRequest (first agent in chain)
				if err := json.Unmarshal([]byte(part.Text), &req); err == nil && req.UserID != "" {
					break
				}
				// Or as AnalysisResult (later in chain)
				_ = json.Unmarshal([]byte(part.Text), &result)
			}
		}

		userID := req.UserID
		if userID == "" {
			userID = "default"
		}
		language := lang.NormalizeLanguage(req.Language)

		// Retrieve memory context for this user
		memCtx := a.retrieveMemory(ctx, userID, language)
		if memCtx != "" {
			// Inject memory context into the result's speech text as a context bridge
			// This will be picked up by the Gateway to inject into Gemini's system instruction
			if result.SpeechText == "" {
				result.SpeechText = memCtx
			}
		}

		data, err := json.Marshal(result)
		if err != nil {
			yield(nil, fmt.Errorf("memory marshal: %w", err))
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

// retrieveMemory fetches the user's cross-session memory and formats it as context.
func (a *Agent) retrieveMemory(ctx context.Context, userID, language string) string {
	if a.store == nil {
		return ""
	}

	entry, err := a.store.GetMemory(ctx, userID)
	if err != nil {
		slog.Warn("memory agent: failed to retrieve memory", "user_id", userID, "error", err)
		return ""
	}
	if entry == nil || len(entry.RecentSummaries) == 0 {
		return ""
	}

	// Build context bridge from most recent summary
	latest := entry.RecentSummaries[len(entry.RecentSummaries)-1]
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Previous session (%s): %s", latest.Date.Format("Jan 2"), latest.Summary))
	if len(latest.UnresolvedIssues) > 0 {
		sb.WriteString("\nUnresolved issues: ")
		sb.WriteString(strings.Join(latest.UnresolvedIssues, ", "))
	}
	sb.WriteString(fmt.Sprintf("\nRespond in %s.", language))
	return sb.String()
}

// SaveSessionSummary generates and stores an end-of-session summary.
// Called by the Gateway when a session ends.
func (a *Agent) SaveSessionSummary(ctx context.Context, userID string, history []string) error {
	if a.store == nil {
		return nil
	}

	summary := a.generateSummary(ctx, history, "English")
	unresolvedIssues := extractUnresolvedIssues(history)

	entry, err := a.store.GetMemory(ctx, userID)
	if err != nil {
		slog.Warn("memory agent: failed to get memory for update", "error", err)
		entry = nil
	}
	if entry == nil {
		entry = &store.MemoryEntry{
			UserID:    userID,
			UpdatedAt: time.Now(),
		}
	}

	// Append new summary, cap at 10
	entry.RecentSummaries = append(entry.RecentSummaries, store.SessionSummary{
		Date:             time.Now(),
		Summary:          summary,
		UnresolvedIssues: unresolvedIssues,
	})
	if len(entry.RecentSummaries) > 10 {
		entry.RecentSummaries = entry.RecentSummaries[len(entry.RecentSummaries)-10:]
	}
	entry.UpdatedAt = time.Now()

	return a.store.UpdateMemory(ctx, userID, entry)
}

// generateSummary uses Gemini to summarize the session history.
func (a *Agent) generateSummary(ctx context.Context, history []string, language string) string {
	if a.genaiClient == nil || len(history) == 0 {
		return "Session completed."
	}

	combined := strings.Join(history, "\n")
	if len(combined) > 4000 {
		combined = combined[len(combined)-4000:]
	}

	prompt := fmt.Sprintf(`VibeCat is a macOS desktop AI companion for solo developers.
Summarize this developer session in 1-2 sentences. Focus on what was worked on and any unresolved issues.

Session events:
%s

Return ONLY valid JSON in this schema: {"summary":"..."}.
Respond in %s.`, combined, lang.NormalizeLanguage(language))

	resp, err := a.genaiClient.Models.GenerateContent(ctx, geminiconfig.LiteTextModel, []*genai.Content{
		{Parts: []*genai.Part{{Text: prompt}}, Role: genai.RoleUser},
	}, &genai.GenerateContentConfig{ResponseMIMEType: "application/json"})
	if err != nil {
		slog.Warn("memory agent: summary generation failed", "error", err)
		return "Session completed."
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return "Session completed."
	}

	text := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		text += part.Text
	}
	if text == "" {
		return "Session completed."
	}

	var payload struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(text), &payload); err == nil && payload.Summary != "" {
		return payload.Summary
	}

	return text
}

// extractUnresolvedIssues looks for error patterns in history that were never resolved.
func extractUnresolvedIssues(history []string) []string {
	var issues []string
	errorKeywords := []string{"error:", "exception:", "failed", "undefined", "cannot"}
	successKeywords := []string{"fixed", "resolved", "passed", "succeeded", "deployed"}

	for _, h := range history {
		lower := strings.ToLower(h)
		hasError := false
		for _, kw := range errorKeywords {
			if strings.Contains(lower, kw) {
				hasError = true
				break
			}
		}
		if !hasError {
			continue
		}
		hasSuccess := false
		for _, kw := range successKeywords {
			if strings.Contains(lower, kw) {
				hasSuccess = true
				break
			}
		}
		if !hasSuccess && len(issues) < 3 {
			// Truncate to reasonable length
			issue := h
			if len(issue) > 100 {
				issue = issue[:100] + "..."
			}
			issues = append(issues, issue)
		}
	}
	return issues
}
