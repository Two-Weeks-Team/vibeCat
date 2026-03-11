// Package memory implements the MemoryAgent for cross-session context.
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"
	"sort"
	"strings"
	"time"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"vibecat/adk-orchestrator/internal/agents/topic"
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

func (a *Agent) RetrieveContext(ctx context.Context, userID, language string) string {
	return a.retrieveMemory(ctx, userID, language)
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

	return formatMemoryContext(entry, language)
}

// SaveSessionSummary generates and stores an end-of-session summary.
// Called by the Gateway when a session ends.
func (a *Agent) SaveSessionSummary(ctx context.Context, userID string, history []string, language string) error {
	if a.store == nil {
		return nil
	}

	language = lang.NormalizeLanguage(language)
	summary := a.generateSummary(ctx, history, language)
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
	entry.KnownTopics = mergeTopics(entry.KnownTopics, history)

	return a.store.UpdateMemory(ctx, userID, entry)
}

// SaveTaskSummary appends a navigator-task-sized summary into cross-session memory.
// This runs on the background lane and must stay lightweight on the caller path.
func (a *Agent) SaveTaskSummary(ctx context.Context, userID, summary string, history []string, language string) error {
	if a.store == nil || strings.TrimSpace(userID) == "" {
		return nil
	}

	language = lang.NormalizeLanguage(language)
	summary = strings.TrimSpace(summary)
	if summary == "" {
		summary = a.generateSummary(ctx, history, language)
	}
	if summary == "" {
		return nil
	}

	entry, err := a.store.GetMemory(ctx, userID)
	if err != nil {
		slog.Warn("memory agent: failed to get memory for task update", "error", err)
		entry = nil
	}
	if entry == nil {
		entry = &store.MemoryEntry{
			UserID:    userID,
			UpdatedAt: time.Now(),
		}
	}

	entry.RecentSummaries = append(entry.RecentSummaries, store.SessionSummary{
		Date:             time.Now(),
		Summary:          summary,
		UnresolvedIssues: extractUnresolvedIssues(history),
	})
	if len(entry.RecentSummaries) > 10 {
		entry.RecentSummaries = entry.RecentSummaries[len(entry.RecentSummaries)-10:]
	}
	entry.UpdatedAt = time.Now()
	entry.KnownTopics = mergeTopics(entry.KnownTopics, history)

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

func mergeTopics(existing []store.Topic, history []string) []store.Topic {
	if len(history) == 0 {
		return existing
	}

	index := make(map[string]int, len(existing))
	for i, topicEntry := range existing {
		index[topicEntry.Name] = i
	}

	now := time.Now()
	for _, event := range history {
		for _, name := range topic.Detect(event) {
			if pos, ok := index[name]; ok {
				existing[pos].LastMentioned = now
				continue
			}
			index[name] = len(existing)
			existing = append(existing, store.Topic{
				Name:          name,
				LastMentioned: now,
			})
		}
	}

	if len(existing) > 20 {
		existing = existing[len(existing)-20:]
	}
	return existing
}

func formatMemoryContext(entry *store.MemoryEntry, language string) string {
	if entry == nil || len(entry.RecentSummaries) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Recent developer context:\n")

	summaryStart := len(entry.RecentSummaries) - 2
	if summaryStart < 0 {
		summaryStart = 0
	}
	for _, summary := range entry.RecentSummaries[summaryStart:] {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", summary.Date.Format("Jan 2"), summary.Summary))
		if len(summary.UnresolvedIssues) > 0 {
			sb.WriteString("  Unresolved: ")
			sb.WriteString(strings.Join(summary.UnresolvedIssues, ", "))
			sb.WriteString("\n")
		}
	}

	topics := append([]store.Topic(nil), entry.KnownTopics...)
	sort.Slice(topics, func(i, j int) bool {
		return topics[i].LastMentioned.After(topics[j].LastMentioned)
	})
	activeTopics := make([]string, 0, 3)
	for _, topicEntry := range topics {
		if topicEntry.Resolved || strings.TrimSpace(topicEntry.Name) == "" {
			continue
		}
		activeTopics = append(activeTopics, topicEntry.Name)
		if len(activeTopics) == 3 {
			break
		}
	}
	if len(activeTopics) > 0 {
		sb.WriteString("Active topics: ")
		sb.WriteString(strings.Join(activeTopics, ", "))
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("Respond in %s.", language))
	return strings.TrimSpace(sb.String())
}
