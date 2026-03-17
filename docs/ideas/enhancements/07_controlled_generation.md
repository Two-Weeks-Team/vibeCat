# P1-7: Controlled Generation (JSON Schema)

**SDK Verification (CONFIRMED via go doc v1.49.0)**:
- `GenerateContentConfig.ResponseMIMEType string` — EXISTS
- `GenerateContentConfig.ResponseSchema *genai.Schema` — EXISTS
- `genai.Schema{Type, Properties, Required, Enum, Items}` — EXISTS
- Supported MIME types: `application/json`, `text/x.enum`
- Live API compatible: NO (batch-only, not in LiveConnectConfig)

**Applicability**: ADK Orchestrator batch calls only.

**Current Code** (adk-orchestrator):
- `internal/agents/tooluse/tooluse.go:46-62` — classifyPrompt outputs unstructured text
- `internal/agents/tooluse/tooluse.go:181-268` — execute() parses response manually
- `internal/models/models.go:59-68` — ToolKind as plain strings
- Mood detector at mood.go — rule-based, no structured output
- Mediator at mediator.go — LLM classification without schema

**Implementation**:
1. Add response schemas to tool classification agent
2. Add response schemas to mediator decision agent
3. Use enum mode for mood classification
4. Add structured output for ADK orchestrator /analyze endpoint

**Go Code**:
```go
// Tool classification with schema:
classifyConfig := &genai.GenerateContentConfig{
    ResponseMIMEType: "application/json",
    ResponseSchema: &genai.Schema{
        Type: genai.TypeObject,
        Properties: map[string]*genai.Schema{
            "tool_kind": {
                Type: genai.TypeString,
                Enum: []string{"none", "search", "maps", "url_context", "code_execution", "file_search"},
            },
            "confidence": {Type: genai.TypeNumber},
            "reasoning":  {Type: genai.TypeString},
        },
        Required: []string{"tool_kind", "confidence"},
    },
}

// Mediator decision with enum:
mediatorConfig := &genai.GenerateContentConfig{
    ResponseMIMEType: "text/x.enum",
    ResponseSchema: &genai.Schema{
        Type: genai.TypeString,
        Enum: []string{"OBSERVE", "SUGGEST", "ACT", "FEEDBACK", "IDLE"},
    },
}

// Analysis result with full schema:
analysisConfig := &genai.GenerateContentConfig{
    ResponseMIMEType: "application/json",
    ResponseSchema: &genai.Schema{
        Type: genai.TypeObject,
        Properties: map[string]*genai.Schema{
            "action": {
                Type: genai.TypeString,
                Enum: []string{"observe", "suggest", "act", "feedback", "idle"},
            },
            "target":     {Type: genai.TypeString},
            "confidence": {Type: genai.TypeNumber},
            "suggestion": {Type: genai.TypeString},
            "mood":       {Type: genai.TypeString, Enum: []string{"focused", "frustrated", "curious", "idle", "celebrating"}},
        },
        Required: []string{"action", "confidence"},
    },
}
```

**Verification**:
- Send prompts that should trigger each tool kind
- Verify response is valid JSON matching schema
- Verify enum values are constrained (no hallucinated values)
- Measure parse error rate before/after (should drop to 0)

**Risks**:
- Structured output may increase latency slightly
- Not all models support all schema features equally
- enum mode is preview feature
