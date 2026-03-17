# VibeCat Enhancement Research — 2026-03-17

> Post-submission (v0.1.0) research for future development.
> Source: GoogleCloudPlatform/generative-ai repo + Gemini Live Agent Challenge resources + competition analysis.

## Current State (v0.1.0)

| Component | Details |
|-----------|---------|
| **Category** | UI Navigator |
| **Client** | Swift 6.2, macOS 15+, ScreenCaptureKit, Accessibility API |
| **Backend** | Go 1.26.1, GenAI SDK v1.49.0, ADK v0.6.0 |
| **Cloud** | Cloud Run (asia-northeast3), Firestore, Secret Manager, Cloud Logging/Trace/Monitoring |
| **Models** | gemini-2.5-flash-native-audio-preview, gemini-2.5-flash-preview-tts, gemini-3.1-flash-lite-preview, gemini-2.5-flash |
| **Features** | Proactive Companion, 5 FC tools, Triple-source grounding (AX/CDP/Vision), Self-healing, 9-agent ADK graph |
| **Tests** | 131 tests (Swift 20 files + Go 29 files), all passing |

---

## Judging Criteria

| Criteria | Weight | Current Score | Gap |
|----------|:------:|:------------:|-----|
| Innovation & Multimodal UX | 40% | ★★★★☆ | Affective Dialog expansion, Thinking transparency |
| Technical Implementation | 30% | ★★★★★ | Model upgrade (3.1), Context Caching, Structured Output |
| Demo & Presentation | 30% | ★★★★☆ | Submitted (frozen) |

---

## Enhancement Roadmap (Priority Order)

### P0 — Direct UI Navigator Impact

#### 1. ThinkingConfig + Thought Signatures

**Source**: `gemini/thinking/intro_thought_signatures.ipynb`

**Current**: Model reasoning is opaque. Multi-step navigation relies on implicit context.

**Enhancement**: Enable thinking mode for transparent chain-of-thought reasoning.

```go
config := &genai.GenerateContentConfig{
    ThinkingConfig: &genai.ThinkingConfig{
        IncludeThoughts: true,
    },
}
```

**Thought Signatures** are critical for multi-step FC workflows:
- Model returns `thought_signature` (encrypted reasoning state) with each FC
- Must be preserved in conversation history for reasoning continuity
- **Rules**: Never merge signed Parts with unsigned Parts; never merge two signed Parts

**Impact**:
- Multi-step navigation accuracy improvement
- UI transparency ("thinking..." display)
- Debugging visibility for development

**Complexity**: Medium | **Files to modify**: `internal/live/session.go`, `internal/ws/handler.go`

---

#### 2. Context Caching

**Source**: `gemini/context-caching/intro_context_caching.ipynb`

**Current**: System prompt + persona context sent with every request.

**Enhancement**: Cache static context for ~75% input token cost reduction.

```go
cache, _ := client.Caches.Create(ctx, model, &genai.CreateCachedContentConfig{
    Contents:          systemPromptContents,
    SystemInstruction: vibeCatPersona,
    TTL:               time.Hour,
})

// Use in requests
resp, _ := client.Models.GenerateContent(ctx, model, userContents, &genai.GenerateContentConfig{
    CachedContent: cache.Name,
})
```

**Requirements**:
- Minimum 2,048 tokens (VibeCat system prompt exceeds this)
- Cache is model-specific
- Default TTL: 60 minutes (configurable)
- Usage metadata: `response.UsageMetadata.CachedContentTokenCount`

**Impact**: Cost reduction + faster TTFT (time to first token)

**Complexity**: Low | **Files to modify**: `internal/live/session.go`, `main.go` (cache lifecycle)

---

#### 3. Forced Function Calling (ANY Mode)

**Source**: `gemini/function-calling/forced_function_calling.ipynb`

**Current**: AUTO mode only — model decides whether to call a tool or respond with text.

**Enhancement**: Use `ANY` mode when user intent clearly requires tool execution.

```go
// When user says "VS Code 열어줘" — force tool execution
config.ToolConfig = &genai.ToolConfig{
    FunctionCallingConfig: &genai.FunctionCallingConfig{
        Mode: genai.FunctionCallingConfigModeAny,
        AllowedFunctionNames: []string{"navigate_focus_app"},
    },
}

// For casual chat — disable tools
config.ToolConfig = &genai.ToolConfig{
    FunctionCallingConfig: &genai.FunctionCallingConfig{
        Mode: genai.FunctionCallingConfigModeNone,
    },
}
```

**Three modes**:
| Mode | When to Use |
|------|------------|
| `AUTO` | Default — model decides |
| `ANY` | Clear user intent for action (force FC) |
| `NONE` | Casual conversation (disable FC) |

**Impact**: Deterministic routing → fewer "I'll help you with that" responses when action is needed

**Complexity**: Low | **Files to modify**: `internal/live/session.go`, `internal/ws/handler.go` (intent classifier)

---

#### 4. Parallel Function Calling

**Source**: `gemini/function-calling/parallel_function_calling.ipynb`

**Current**: Sequential FC execution (`pendingFC sequential execution`).

**Enhancement**: Gemini can return multiple FCs in a single response → execute in parallel.

```go
calls := extractFunctionCalls(response) // Multiple FCs from single response

var wg sync.WaitGroup
results := make([]*genai.FunctionResponse, len(calls))
for i, call := range calls {
    wg.Add(1)
    go func(idx int, c *genai.FunctionCall) {
        defer wg.Done()
        results[idx] = execute(ctx, c)
    }(i, call)
}
wg.Wait()

session.SendToolResponse(results)
```

**Use case**: "터미널 열고 VS Code도 열어줘" → `navigate_focus_app("Terminal")` + `navigate_focus_app("Visual Studio Code")` simultaneously.

**Impact**: Multi-task scenarios ~2x faster

**Complexity**: Medium | **Files to modify**: `internal/ws/handler.go` (pendingFC logic)

---

#### 5. Safety Decision Handling

**Source**: `gemini/computer-use/web-agent/web_agent.py`

**Current**: No explicit safety confirmation for risky actions.

**Enhancement**: Model returns `safety_decision: "require_confirmation"` for risky operations → prompt user before execution.

```go
type SafetyDecision struct {
    Decision    string `json:"decision"`     // "require_confirmation" | "proceed"
    Explanation string `json:"explanation"`
}

func (h *Handler) handleFunctionCall(call FunctionCall) {
    if call.SafetyDecision != nil && call.SafetyDecision.Decision == "require_confirmation" {
        // Send to Swift UI for user approval
        approved := h.requestUserApproval(call.SafetyDecision.Explanation)
        if !approved {
            h.sendToolResponse(call.Name, map[string]any{"result": "user_denied"})
            return
        }
        // Include safety_acknowledgement: true in response
    }
    h.executeTool(call)
}
```

**Impact**: Production-level safety for file deletion, password entry, system commands

**Complexity**: Medium | **Files to modify**: `internal/ws/handler.go`, Swift UI (confirmation dialog)

---

#### 6. Heartbeat Pattern

**Source**: `gemini/sample-apps/gemini-live-telephony-app/utils/live_api.py`

**Current**: Session resumption on disconnect. No active keep-alive.

**Enhancement**: Send 320 bytes of silent PCM every 5 seconds to prevent timeout during idle periods.

```go
func (s *Session) startHeartbeat(ctx context.Context) {
    silence := make([]byte, 320) // 16-bit PCM silence
    ticker := time.NewTicker(5 * time.Second)
    go func() {
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                _ = s.SendAudio(silence)
            }
        }
    }()
}
```

**Impact**: Prevents connection drops during Proactive Companion idle observation periods.

**Complexity**: Low | **Files to modify**: `internal/live/session.go`

---

### P1 — Quality & Reliability

#### 7. Controlled Generation (JSON Schema)

**Source**: `gemini/controlled-generation/intro_controlled_generation.ipynb`

**Current**: Partially structured output in ADK tool classification.

**Enhancement**: Full JSON Schema constraints with enum support.

```go
// For ADK orchestrator — structured action decisions
config := &genai.GenerateContentConfig{
    ResponseMIMEType: "application/json",
    ResponseSchema: &genai.Schema{
        Type: genai.TypeObject,
        Properties: map[string]*genai.Schema{
            "action":     {Type: genai.TypeString, Enum: []string{"observe", "suggest", "act", "feedback"}},
            "confidence": {Type: genai.TypeNumber},
            "target":     {Type: genai.TypeString},
            "reasoning":  {Type: genai.TypeString},
        },
        Required: []string{"action", "confidence"},
    },
}

// Enum mode for classification
config := &genai.GenerateContentConfig{
    ResponseMIMEType: "text/x.enum",
    ResponseSchema: &genai.Schema{
        Type: genai.TypeString,
        Enum: []string{"OBSERVE", "SUGGEST", "ACT", "FEEDBACK", "IDLE"},
    },
}
```

**Impact**: Eliminates JSON parse errors in agent communication

**Complexity**: Low | **Files to modify**: ADK agent configs

---

#### 8. Gemini Native `computer_use` Tool

**Source**: `gemini/computer-use/intro_computer_use.ipynb`, `web-agent/`

**Current**: 5 custom FC tools with absolute coordinates.

**Enhancement**: Gemini's built-in `computer_use` tool with normalized coordinates (0-1000).

```go
tools := []*genai.Tool{{
    ComputerUse: &genai.ComputerUse{
        Environment: genai.EnvironmentOS,
        ExcludedPredefinedFunctions: []string{"drag_and_drop"},
    },
}}
```

**Key differences from current approach**:
- **Normalized coordinates (0-1000)** — resolution-independent
- **FunctionResponse includes screenshot blob** — model verifies action results inline
- **Environment switching** — `ENVIRONMENT_BROWSER` vs `ENVIRONMENT_OS`

**⚠️ This is an architectural shift** — replaces the 5-tool approach with a unified tool.
Recommend: Experiment on a separate feature branch first.

**Impact**: Potentially better visual precision (key judging criterion)

**Complexity**: High | **Files to modify**: Major refactor of `internal/live/session.go`, `internal/ws/handler.go`, all navigator logic

---

### P2 — Agent Architecture

#### 9. Always-On Memory Agent (Sleep/Consolidation)

**Source**: `gemini/agents/always-on-memory-agent/`

**Current**: Firestore session memory (passive storage).

**Enhancement**: Human-like memory with active consolidation.

```
IngestAgent (multimodal) → SQLite → [30min timer] → ConsolidateAgent → QueryAgent
```

**Key features**:
- **Active consolidation**: Idle timer triggers cross-memory pattern discovery
- **Entity/Topic extraction**: Structured memory fields (not just raw text)
- **Importance scoring**: Priority-based memory retrieval
- **Connection tracking**: `from_id ↔ to_id` relationships between memories

**Use case**: VibeCat learns "user always opens Spotify after lunch" → proactive suggestion timing improves.

**Impact**: Better Proactive Companion suggestions over time

**Complexity**: High | **Files to modify**: New `internal/memory/` package in ADK orchestrator

---

#### 10. MCP (Model Context Protocol) Integration

**Source**: `gemini/mcp/adk_mcp_app/`, `adk_multiagent_mcp_app/`

**Current**: 5 internal FC tools, no external tool ecosystem.

**Enhancement**: Connect external services as MCP servers.

```go
// ADK + MCP toolset integration
rootAgent := adk.NewLLMAgent(adk.LLMAgentConfig{
    Tools: []adk.Tool{
        adk.NewMCPToolset(adk.StdioServerParams{
            Command: "npx", Args: []string{"@anthropic/filesystem-mcp"},
        }),
    },
})
```

**Potential MCP servers**:
- File System MCP → project file search/management
- GitHub MCP → issue/PR management
- Spotify MCP → music control
- Calendar MCP → schedule queries

**Impact**: Extensible capability without code changes

**Complexity**: Medium | **Files to modify**: ADK orchestrator agent graph

---

#### 11. Real-time RAG

**Source**: `gemini/multimodal-live-api/real_time_rag_bank_loans_gemini_2_0.ipynb`

**Current**: Google Search grounding only (no custom document search).

**Enhancement**: Embed user's project docs → vector similarity search during live conversation.

```
User voice → intent extraction → text-embedding-005 → cosine similarity → top-K chunks → context injection
```

**Use case**: User asks about project API → VibeCat searches project README/docs instead of web.

**Impact**: Project-specific knowledge grounding

**Complexity**: High | **New services**: Embedding pipeline, vector store (or Firestore with embedding field)

---

## Competition Insights

### What Judges Value Most

| Judge (Organization) | Key Insight |
|---|---|
| Richard Moot (Square) | "Projects that consider ALL judging criteria stand out" |
| Warren Marusiak (Atlassian) | "Would I actually use this? Something unique that differentiates" |
| Kelvin Boateng (Google) | "Visual appeal first, then functionality. Depart from templates" |

### Winning Patterns (from past Google-sponsored competitions)

| Winner | Differentiator | VibeCat Parallel |
|--------|---------------|-----------------|
| Jayu (2024 Best Overall) | Screen-aware personal assistant | VibeCat's proactive approach is more advanced |
| Prospera (Most Useful) | Real-time sales coaching during live conversations | Similar real-time feedback loop |
| NurAI (2026 submission) | Seamless see-hear-speak loop with interruptible conversations | VibeCat has this |

### VibeCat's Unique Advantage

Most submissions are **reactive** chatbots. VibeCat's **proactive companion model** (OBSERVE → SUGGEST → WAIT → ACT → FEEDBACK) is a genuine paradigm shift that few competitors match.

---

## Technical Reference

### Models Available (March 2026)

| Model | Purpose | Status |
|-------|---------|--------|
| `gemini-2.5-flash-native-audio-preview-12-2025` | Live API (native audio) | ✅ Using |
| `gemini-2.5-flash-preview-tts` | Text-to-speech | ✅ Using |
| `gemini-3.1-flash-lite-preview` | Vision (cost-effective) | ✅ Using |
| `gemini-2.5-flash` | Search + tool use | ✅ Using |
| `gemini-3.1-pro-preview` | Premium reasoning | ❌ Not using |
| `gemini-2.5-flash-tts` | Expressive TTS | ❌ Not using |
| `gemini-embedding-2-preview` | Multimodal embeddings | ❌ Not using |

**⚠️ Deprecation**: `gemini-2.5-flash` and `gemini-2.5-pro` deprecated June 17, 2026.

### GCP Services Currently Used (7)

Cloud Run, Firestore, Secret Manager, Cloud Logging, Cloud Trace, Cloud Monitoring, Artifact Registry

### GCP Services Available to Add

| Service | Use Case | Priority |
|---------|----------|:--------:|
| Cloud Storage | Generated artifact storage | Low |
| Cloud Tasks | Background job scheduling | Medium |
| Cloud Pub/Sub | Service-to-service async messaging | Medium |
| Error Reporting | Centralized error tracking | Low |

---

## Research Sources

| Source | What Was Analyzed |
|--------|------------------|
| [GoogleCloudPlatform/generative-ai](https://github.com/GoogleCloudPlatform/generative-ai) | Cloned and analyzed: computer-use, Live API, context-caching, thinking, controlled-generation, MCP, function-calling, agents, url-context, grounding |
| [Gemini Live Agent Challenge Resources](https://geminiliveagentchallenge.devpost.com/resources) | ADK bidi-streaming, GenMedia, multimodality agents |
| [Challenge Rules](https://geminiliveagentchallenge.devpost.com/rules) | Post-submission development policy |
| Competition Analysis | Past Devpost winners, judge interview insights, submission patterns |
| Gemini API Docs | Latest capabilities, model changelog, deprecation notices |

---

*Document generated: 2026-03-17 | For VibeCat v0.2.0+ development planning*
