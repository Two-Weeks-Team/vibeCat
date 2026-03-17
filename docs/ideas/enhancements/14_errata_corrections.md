# 14: Errata and Corrections for Documents 01-11

This document corrects errors, fills gaps, and updates findings in the original 11 enhancement specifications based on deeper codebase analysis.

## Critical Corrections

### Doc 01 (ThinkingConfig): Model Compatibility Warning

**Original claim**: "Live API compatible: YES"
**Correction**: Partially true. `LiveConnectConfig.ThinkingConfig` field EXISTS (confirmed via `go doc`), BUT the current Live model `gemini-2.5-flash-native-audio-preview-12-2025` may NOT support thinking. The SDK docs state: "An error will be returned if this field is set for models that don't support thinking."

**Required**: Test with current model first. If error, need to:
1. Upgrade to a thinking-capable model (Gemini 3.x series)
2. Or use ThinkingConfig only in ADK orchestrator batch calls

**Add to Risks section**:
```
- Current Live model (gemini-2.5-flash-native-audio-preview) may not support thinking
- Test with ThinkingConfig.IncludeThoughts=true before full implementation
- Fallback: Use ThinkingConfig only in ADK batch calls via GenerateContentConfig
```

### Doc 01 (ThinkingConfig): Missing ThoughtSignature Handling

**Original**: References `Part.ThoughtSignature` but no code for preserving signatures across turns.

**Correction**: In Live API, conversation history is managed by the server (not client). ThoughtSignature preservation is automatic within a session. However, if implementing ThinkingConfig in ADK orchestrator batch calls, signatures MUST be manually preserved:

```go
// For ADK batch calls (not Live API):
// When receiving response with thought parts, preserve them in next request:
for _, part := range response.Candidates[0].Content.Parts {
    if part.Thought && len(part.ThoughtSignature) > 0 {
        // Include this part AS-IS in the next request's conversation history
        // NEVER merge signed parts with unsigned parts
        // NEVER merge two signed parts
        historyParts = append(historyParts, part)
    }
}
```

### Doc 02 (Context Caching): ADK Integration Path Corrected

**Original**: Shows standalone `cache.Manager` but doesn't show how to use with ADK agents.

**Correction**: ADK `llmagent.Config` has `GenerateContentConfig *genai.GenerateContentConfig` field (confirmed at llmagent.go:153). Correct injection:

```go
// Method 1: Via llmagent.Config (for ADK-managed agents)
cacheName, _ := cacheManager.GetOrCreate(ctx, model, systemPrompt, tools)
agent, _ := llmagent.New(llmagent.Config{
    Name:  "vision_agent",
    Model: genaimodel.GoogleAI(geminiconfig.VisionModel),
    GenerateContentConfig: &genai.GenerateContentConfig{
        CachedContentName: cacheName,
    },
})

// Method 2: Via direct GenerateContent (for non-ADK calls in tooluse.go, search.go, etc.)
resp, _ := client.Models.GenerateContent(ctx, model, contents, &genai.GenerateContentConfig{
    CachedContentName: cacheName,
    // other config...
})
```

**Additional note**: The orchestrator makes 9+ direct `client.Models.GenerateContent()` calls across files:
- tooluse.go:126, search.go:163, classifier.go:42, memory.go:216, mediator.go:353, engagement.go:179, celebration.go:238, vision.go:170, processor.go:61
All of these can use `CachedContentName` for their respective system prompts.

### Doc 03 (Forced FC): ADK ToolConfig Path Corrected

**Original**: Shows modifying `tooluse.go execute()` which makes direct GenerateContent calls.

**Correction**: This is actually CORRECT for VibeCat because the orchestrator agents use direct `client.Models.GenerateContent()` calls, NOT ADK's built-in model calling. So `ToolConfig` can be added directly:

```go
// In tooluse.go execute() — this IS a direct GenerateContent call, so ToolConfig works:
config := &genai.GenerateContentConfig{
    Tools: a.toolsForKind(kind),
    ToolConfig: &genai.ToolConfig{
        FunctionCallingConfig: &genai.FunctionCallingConfig{
            Mode: genai.FunctionCallingConfigModeAny,
        },
    },
}
resp, err := a.client.Models.GenerateContent(ctx, model, contents, config)
```

**However**: For ADK-managed agents (like `llmsearch.go:90-124`), `ToolConfig` is NOT directly exposed via `llmagent.Config`. Tools are passed via `Tools []tool.Tool` field and processed internally. Forced FC mode is NOT available for ADK-managed agents.

**Clarify in doc**: "Forced FC works for direct GenerateContent calls (most orchestrator agents). Does NOT work for ADK-managed agents via llmagent.Config."

### Doc 03 (Forced FC): Fix Broken Code Example

**Original**: `shouldSkipTools()` creates config but never uses it.
**Correction**: The function should use the config in an actual GenerateContent call:

```go
// CORRECTED: Use NONE mode when mediator determines casual conversation
func (a *MediatorAgent) classifyCasual(ctx context.Context, input string) (bool, error) {
    config := &genai.GenerateContentConfig{
        ToolConfig: &genai.ToolConfig{
            FunctionCallingConfig: &genai.FunctionCallingConfig{
                Mode: genai.FunctionCallingConfigModeNone,
            },
        },
        ResponseMIMEType: "text/x.enum",
        ResponseSchema: &genai.Schema{
            Type: genai.TypeString,
            Enum: []string{"CASUAL", "ACTION_REQUIRED"},
        },
    }
    resp, err := a.client.Models.GenerateContent(ctx, model, contents, config)
    // Parse enum response...
    return resp == "CASUAL", err
}
```

### Doc 05 (Safety Decision): Already Partially Implemented

**Original**: Describes safety as entirely new feature.
**Correction**: VibeCat ALREADY has safety handling:
- `navigatorRiskyActionBlocked` ServerMessage type (AudioMessageParser.swift:27)
- `sendNavigatorRiskConfirmation()` client method (GatewayClient.swift:221-232)
- Risk classification exists in backend navigator.go
- UI handling in AppDelegate.swift:1122-1132

**What's actually new in Doc 05**: The pattern-based `safety.Classifier` in Go is an ENHANCEMENT to existing risk detection, not a replacement. The doc should be reframed as "Enhance existing safety system with pattern-based classification" rather than "Add safety from scratch."

### Doc 07 (Controlled Generation): ADK OutputSchema Shortcut

**Original**: Shows manual `ResponseSchema` and `ResponseMIMEType` injection.
**Correction**: ADK v0.6.0 has `OutputSchema` shortcut in `llmagent.Config` that auto-injects both:

```go
// ADK auto-injection path (for ADK-managed agents):
agent, _ := llmagent.New(llmagent.Config{
    Name:         "tool_classifier",
    OutputSchema: &genai.Schema{
        Type: genai.TypeObject,
        Properties: map[string]*genai.Schema{
            "tool_kind": {Type: genai.TypeString, Enum: []string{...}},
        },
    },
    // ADK automatically sets ResponseSchema and ResponseMIMEType="application/json"
})

// Direct GenerateContent path (for non-ADK agents):
config := &genai.GenerateContentConfig{
    ResponseMIMEType: "application/json",
    ResponseSchema:   mySchema,
}
```

The ADK auto-injection is at: `basic_processor.go:46-51`:
```go
if state.OutputSchema != nil && !needOutputSchemaProcessor(state) {
    req.Config.ResponseSchema = state.OutputSchema
    req.Config.ResponseMIMEType = "application/json"
}
```

### Doc 08 (Computer Use): Analysis Confirmed Correct

No corrections needed. The DEFERRED recommendation is correct. `EnvironmentOS` does not exist in Go SDK v1.49.0.

### Doc 09 (Always-On Memory): Cloud Run Limitations

**Original**: 30-minute timer for consolidation.
**Correction**: Cloud Run scales to zero. A timer-based consolidation agent will NOT work because:
1. Cloud Run instances are ephemeral
2. No guaranteed instance lifetime
3. Timer would reset on every cold start

**Fix**: Use Cloud Scheduler + Cloud Tasks instead:
```
Cloud Scheduler (every 30 min)
    -> Cloud Tasks HTTP trigger
    -> POST /memory/consolidate
    -> Orchestrator runs consolidation for all active users
```

**Also**: Firestore document size limit is 1MB. Large entity graphs may exceed this. Consider subcollections:
```
users/{userId}/memory/data          -> core MemoryEntry (summaries, topics)
users/{userId}/memory/entities      -> separate collection for entities
users/{userId}/memory/connections   -> separate collection for connections
users/{userId}/memory/insights      -> separate collection for insights
```

### Doc 10 (MCP Integration): Cloud Run Deployment Issue

**Original**: Uses `npx` for MCP server startup.
**Correction**: Cloud Run containers do NOT have `npx` or `node` pre-installed. Options:

1. **Sidecar container**: Run MCP server as separate Cloud Run service
2. **Pre-built binary**: Include MCP server binary in Docker image
3. **Local development only**: Use npx for local dev, HTTP-based MCP for production

```go
// Production-compatible MCP transport:
func NewFileSystemMCPToolset(production bool) (tool.Toolset, error) {
    var transport mcptoolset.Transport
    if production {
        // HTTP transport to sidecar service
        transport = mcptoolset.HTTPTransport{
            URL: os.Getenv("MCP_FILESYSTEM_URL"), // e.g., http://mcp-fs:8080
        }
    } else {
        // Stdio transport for local development
        transport = mcptoolset.StdioTransport{
            Command: "npx",
            Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/Users"},
        }
    }
    return mcptoolset.New(mcptoolset.Config{
        Transport:           transport,
        RequireConfirmation: true,
    })
}
```

**Also**: Verify exact `mcptoolset.StdioTransport` type name — may be `mcptoolset.StdioServerParams` or different in ADK v0.6.0.

### Doc 11 (Real-time RAG): API Corrections

**Original**: Uses `client.Models.EmbedContent()` with `EmbedContentConfig`.
**Correction**: Verify exact Go SDK API. The method signature may differ:

```go
// Verify this compiles — exact types may differ:
resp, err := client.Models.EmbedContent(ctx, "text-embedding-005", &genai.EmbedContentConfig{
    Contents: []*genai.Content{{Parts: []*genai.Part{{Text: chunk}}}},
})
```

**Also**: `filepath.Glob()` in Go stdlib does NOT support `**` pattern. Use `filepath.WalkDir()` instead:
```go
func indexProjectDocs(ctx context.Context, projectPath string) error {
    return filepath.WalkDir(projectPath, func(path string, d fs.DirEntry, err error) error {
        if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
            return nil
        }
        content, _ := os.ReadFile(path)
        return ragStore.Embed(ctx, genaiClient, string(content), path)
    })
}
```

## Gap Analysis Correction

**Original gap analysis claimed**: "Mouse click, scroll, drag are NOT implemented."
**Corrected finding**: Swift client AccessibilityNavigator.swift already implements:
- `clickCoordinates` (line 553-596) using CGEvent mouse events
- `pressAX` (line 469-551) using AXUIElementPerformAction
- `copySelection` (line 456-468)
- `pasteText` (line 389-455)
- `systemAction` (line 598-600)

**Actual gap**: These capabilities are NOT exposed as Gemini FC tools. Only 5 of 8+ action types have corresponding FC declarations in session.go.

**True missing capabilities** (not in Swift at all):
- Scroll (CGEvent scroll wheel)
- Drag (CGEvent drag sequence)
- Right-click (CGEvent rightMouseDown/Up)

See Doc 12 (Expand Navigator Tools) for the complete solution.
