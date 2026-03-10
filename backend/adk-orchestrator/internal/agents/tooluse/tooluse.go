package tooluse

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"google.golang.org/genai"

	"vibecat/adk-orchestrator/internal/geminiconfig"
	"vibecat/adk-orchestrator/internal/lang"
	"vibecat/adk-orchestrator/internal/models"
)

var urlPattern = regexp.MustCompile(`https?://[^\s]+`)

var mapsKeywords = []string{
	"nearby", "route", "directions", "distance", "travel time", "commute", "restaurant",
	"coffee", "cafe", "place", "places", "map", "maps", "where is", "near me",
	"근처", "길찾기", "거리", "소요시간", "카페", "맛집", "지도", "어디야",
}

var codeExecutionKeywords = []string{
	"calculate", "compute", "convert", "regex", "json", "csv", "sort", "transform",
	"simulate", "evaluate", "run this code", "check this math", "what is ",
	"계산", "변환", "정규식", "실행", "검산", "숫자", "코드로 확인",
}

var fileSearchKeywords = []string{
	"uploaded file", "uploaded files", "knowledge base", "knowledge-base", "attached file",
	"our docs", "our documentation", "company docs", "internal docs", "file store",
	"업로드한 파일", "첨부 파일", "지식베이스", "사내 문서", "내 문서",
}

var searchKeywords = []string{
	"search", "look up", "find", "browse", "check docs", "check the docs",
	"official docs", "latest", "release notes", "version", "error", "exception",
	"what does", "how do i", "how to", "why is", "docs", "documentation",
	"검색", "찾아", "알아봐", "공식 문서", "최신", "버전", "에러", "문서",
}

const classifyPrompt = `You are a tool router for a developer voice assistant.

Choose exactly one tool for the user's request and return ONLY valid JSON in this schema:
{"tool":"none|search|maps|url_context|code_execution|file_search","reason":"..."}

Tool policy:
- search: live web search, current facts, official docs, versions, releases, debugging help that needs external grounding
- maps: nearby places, addresses, routes, travel time, place recommendations, location grounding
- url_context: the user includes one or more URLs and wants those pages summarized, explained, compared, or answered from
- code_execution: calculations, conversions, data transforms, quick code validation, algorithm checks
- file_search: the user explicitly wants answers from uploaded files or a configured file-search knowledge base
- none: no external tool is clearly needed

Prefer url_context over search when a URL is present.
Prefer maps for places/routes/nearby questions.
Prefer code_execution for calculations or quick verifications.
Reply with one tool only.`

type Agent struct {
	genaiClient      *genai.Client
	fileSearchStores []string
}

func New(client *genai.Client, fileSearchStores []string) *Agent {
	var stores []string
	for _, storeName := range fileSearchStores {
		trimmed := strings.TrimSpace(storeName)
		if trimmed != "" {
			stores = append(stores, trimmed)
		}
	}
	return &Agent{
		genaiClient:      client,
		fileSearchStores: stores,
	}
}

func (a *Agent) Resolve(ctx context.Context, req models.ToolRequest) *models.ToolResult {
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return nil
	}

	plan := a.classify(ctx, query)
	if plan.Tool == models.ToolKindNone {
		return nil
	}
	if plan.Tool == models.ToolKindFileSearch && len(a.fileSearchStores) == 0 {
		slog.Info("[TOOLS] file search requested but no stores configured", "query", truncateText(query, 80))
		return nil
	}

	result := a.execute(ctx, req, plan)
	if result == nil {
		return nil
	}
	result.Query = query
	result.Reason = plan.Reason
	result.CreatedAt = time.Now()
	return result
}

type toolPlan struct {
	Tool   models.ToolKind `json:"tool"`
	Reason string          `json:"reason"`
}

func (a *Agent) classify(ctx context.Context, query string) toolPlan {
	if tool := detectFastPath(query, len(a.fileSearchStores) > 0); tool != models.ToolKindNone {
		return toolPlan{Tool: tool, Reason: "fast_path"}
	}
	if a.genaiClient == nil {
		return toolPlan{Tool: models.ToolKindNone}
	}

	prompt := classifyPrompt
	if len(a.fileSearchStores) == 0 {
		prompt += "\n\nFile search is NOT available in this environment, so never choose file_search."
	}

	resp, err := a.genaiClient.Models.GenerateContent(ctx, geminiconfig.LiteTextModel, []*genai.Content{
		{Parts: []*genai.Part{{Text: query}}, Role: genai.RoleUser},
	}, &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: prompt}},
		},
		ResponseMIMEType: "application/json",
		MaxOutputTokens:  80,
		Temperature:      ptrFloat32(0),
	})
	if err != nil {
		slog.Warn("[TOOLS] classifier failed", "error", err, "query", truncateText(query, 80))
		return toolPlan{Tool: models.ToolKindNone}
	}

	text := extractText(resp)
	if text == "" {
		return toolPlan{Tool: models.ToolKindNone}
	}

	var plan toolPlan
	if err := json.Unmarshal([]byte(text), &plan); err != nil {
		slog.Warn("[TOOLS] classifier parse failed", "error", err, "raw", text)
		return toolPlan{Tool: models.ToolKindNone}
	}

	switch plan.Tool {
	case models.ToolKindSearch, models.ToolKindMaps, models.ToolKindURLContext, models.ToolKindCodeExecution, models.ToolKindFileSearch:
	default:
		plan.Tool = models.ToolKindNone
	}
	if plan.Tool == models.ToolKindFileSearch && len(a.fileSearchStores) == 0 {
		plan.Tool = models.ToolKindNone
	}
	return plan
}

func detectFastPath(query string, fileSearchEnabled bool) models.ToolKind {
	lower := strings.ToLower(query)
	switch {
	case urlPattern.MatchString(query):
		return models.ToolKindURLContext
	case containsAny(lower, mapsKeywords):
		return models.ToolKindMaps
	case containsAny(lower, codeExecutionKeywords):
		return models.ToolKindCodeExecution
	case fileSearchEnabled && containsAny(lower, fileSearchKeywords):
		return models.ToolKindFileSearch
	case containsAny(lower, searchKeywords):
		return models.ToolKindSearch
	default:
		return models.ToolKindNone
	}
}

func (a *Agent) execute(ctx context.Context, req models.ToolRequest, plan toolPlan) *models.ToolResult {
	if a.genaiClient == nil {
		return nil
	}

	language := lang.NormalizeLanguage(req.Language)
	query := strings.TrimSpace(req.Query)

	var (
		tool         *genai.Tool
		systemPrompt string
		userPrompt   = query
	)

	switch plan.Tool {
	case models.ToolKindSearch:
		tool = &genai.Tool{GoogleSearch: &genai.GoogleSearch{}}
		systemPrompt = fmt.Sprintf(`You are SearchBuddy for VibeCat.
Use Google Search grounding when needed and answer the user's request naturally in %s.
Be concise, concrete, and useful for a solo developer. Prefer official docs or primary sources when possible.`, language)
		userPrompt = fmt.Sprintf("Search the web and answer this in %s using 2-3 complete sentences. Prefer official docs or primary sources when possible.\nUser request: %s\nFinish the final sentence completely.", language, query)
	case models.ToolKindMaps:
		tool = &genai.Tool{GoogleMaps: &genai.GoogleMaps{}}
		systemPrompt = fmt.Sprintf(`You are MapsBuddy for VibeCat.
Use Google Maps grounding to answer place, nearby, route, or travel-time questions.
Respond in %s. Give concise, practical recommendations or directions.`, language)
	case models.ToolKindURLContext:
		tool = &genai.Tool{URLContext: &genai.URLContext{}}
		systemPrompt = fmt.Sprintf(`You are URLBuddy for VibeCat.
Use URL context to read the referenced page and answer from that page.
Respond in %s. Summarize clearly and mention important caveats from the page if relevant.`, language)
	case models.ToolKindCodeExecution:
		tool = &genai.Tool{CodeExecution: &genai.ToolCodeExecution{}}
		systemPrompt = fmt.Sprintf(`You are CodeExecBuddy for VibeCat.
Use code execution whenever calculation or verification would help.
Respond in %s. Give the final answer briefly, then mention the checked result when useful.`, language)
	case models.ToolKindFileSearch:
		topK := int32(6)
		tool = &genai.Tool{
			FileSearch: &genai.FileSearch{
				FileSearchStoreNames: a.fileSearchStores,
				TopK:                 &topK,
			},
		}
		systemPrompt = fmt.Sprintf(`You are FileSearchBuddy for VibeCat.
Use file search only against the configured stores and answer in %s.
Be concise and grounded in the retrieved files.`, language)
	default:
		return nil
	}

	resp, err := a.genaiClient.Models.GenerateContent(ctx, geminiconfig.ToolModel, []*genai.Content{
		{Parts: []*genai.Part{{Text: userPrompt}}, Role: genai.RoleUser},
	}, &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: systemPrompt}},
		},
		MaxOutputTokens: 600,
		Tools:           []*genai.Tool{tool},
	})
	if err != nil {
		slog.Warn("[TOOLS] tool execution failed", "tool", plan.Tool, "error", err, "query", truncateText(query, 80))
		return &models.ToolResult{
			Tool:    plan.Tool,
			Summary: "Tool execution failed.",
		}
	}

	candidate := firstCandidate(resp)
	if candidate == nil || candidate.Content == nil {
		return &models.ToolResult{
			Tool:    plan.Tool,
			Summary: "No grounded result was returned.",
		}
	}

	result := &models.ToolResult{
		Tool:    plan.Tool,
		Summary: dedupeParagraphs(strings.TrimSpace(extractText(resp))),
		Sources: extractSources(candidate),
	}
	result.RetrievedURLs = extractRetrievedURLs(candidate)
	result.GeneratedCode, result.CodeOutput = extractCodeArtifacts(candidate)

	if result.Summary == "" {
		result.Summary = "No grounded result was returned."
	}
	return result
}

func firstCandidate(resp *genai.GenerateContentResponse) *genai.Candidate {
	if resp == nil || len(resp.Candidates) == 0 {
		return nil
	}
	return resp.Candidates[0]
}

func extractText(resp *genai.GenerateContentResponse) string {
	candidate := firstCandidate(resp)
	if candidate == nil || candidate.Content == nil {
		return ""
	}

	var parts []string
	for _, part := range candidate.Content.Parts {
		if strings.TrimSpace(part.Text) != "" {
			parts = append(parts, strings.TrimSpace(part.Text))
		}
	}
	return strings.Join(parts, "\n")
}

func dedupeParagraphs(text string) string {
	if text == "" {
		return text
	}

	segments := strings.Split(text, "\n")
	seen := map[string]struct{}{}
	var deduped []string
	for _, segment := range segments {
		trimmed := strings.TrimSpace(segment)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, trimmed)
	}
	return strings.Join(deduped, "\n")
}

func extractSources(candidate *genai.Candidate) []string {
	if candidate == nil || candidate.GroundingMetadata == nil {
		return nil
	}

	seen := map[string]struct{}{}
	var sources []string
	for _, chunk := range candidate.GroundingMetadata.GroundingChunks {
		if chunk == nil || chunk.Web == nil || chunk.Web.URI == "" {
			continue
		}
		if _, ok := seen[chunk.Web.URI]; ok {
			continue
		}
		seen[chunk.Web.URI] = struct{}{}
		sources = append(sources, chunk.Web.URI)
	}
	return sources
}

func extractRetrievedURLs(candidate *genai.Candidate) []string {
	if candidate == nil || candidate.URLContextMetadata == nil {
		return nil
	}

	seen := map[string]struct{}{}
	var urls []string
	for _, metadata := range candidate.URLContextMetadata.URLMetadata {
		if metadata == nil || metadata.RetrievedURL == "" {
			continue
		}
		if _, ok := seen[metadata.RetrievedURL]; ok {
			continue
		}
		seen[metadata.RetrievedURL] = struct{}{}
		urls = append(urls, metadata.RetrievedURL)
	}
	return urls
}

func extractCodeArtifacts(candidate *genai.Candidate) (string, string) {
	if candidate == nil || candidate.Content == nil {
		return "", ""
	}

	var codeParts []string
	var outputs []string
	for _, part := range candidate.Content.Parts {
		if part.ExecutableCode != nil && strings.TrimSpace(part.ExecutableCode.Code) != "" {
			codeParts = append(codeParts, part.ExecutableCode.Code)
		}
		if part.CodeExecutionResult != nil && strings.TrimSpace(part.CodeExecutionResult.Output) != "" {
			outputs = append(outputs, strings.TrimSpace(part.CodeExecutionResult.Output))
		}
	}
	return strings.Join(codeParts, "\n\n"), strings.Join(outputs, "\n")
}

func containsAny(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func ptrFloat32(v float32) *float32 { return &v }

func truncateText(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}
