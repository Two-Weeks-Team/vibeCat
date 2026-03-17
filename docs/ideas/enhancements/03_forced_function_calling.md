# P0-3: Forced Function Calling

## Status

Implement for direct batch `GenerateContent` routes only. Do not plan this for the Live gateway.

## Source-Verified Facts

- `GenerateContentConfig.ToolConfig` exists.
- `FunctionCallingConfig` supports `AUTO`, `ANY`, `NONE`, and preview `VALIDATED`.
- `LiveConnectConfig` does not have `ToolConfig`, so this does not apply to Live API configuration.

## Current Repo State

- `backend/adk-orchestrator/internal/agents/tooluse/tooluse.go`
  - uses direct `Models.GenerateContent()`
  - already has a fast-path classifier before model routing
- `backend/adk-orchestrator/internal/agents/search/llmsearch.go`
  - uses ADK `llmagent` with tools
  - should not be the first target for forced FC work

## Implementation Decision

### Use cases that should use forced FC

- The router already knows the exact tool family to call
- The request must return a tool invocation rather than free-form text
- Examples:
  - URL present and `url_context` is already selected
  - explicit calculation request and `code_execution` is already selected
  - explicit maps/place request and `maps` is already selected

### Use cases that should not

- Live gateway tool routing
- ADK `llmagent` tool-use paths in v0.2
- cases where natural-language fallback is desirable

## Recommended Mode Usage

- `ANY`
  - direct routed tool invocation where a tool call is required
- `NONE`
  - classification or text-only paths where tools must not fire
- `VALIDATED`
  - experiment later if we want tool-or-text with schema adherence

## Concrete File Changes

### `backend/adk-orchestrator/internal/agents/tooluse/tooluse.go`

- Add helper:
  - `toolConfigForPlan(plan toolPlan) *genai.ToolConfig`
- When fast-path or LLM classifier already selected one tool kind:
  - restrict tools to that kind
  - set `Mode: ANY`
- When running text-only classifier passes:
  - set `Mode: NONE`

### Do not change

- `backend/realtime-gateway/internal/live/session.go`
- Live tool declarations

## Acceptance Criteria

1. Direct batch routes with known tool kind always produce function calls.
2. Text-only classifier paths never emit function calls.
3. No change in Live API behavior.

## Risks

- `ANY` can force low-quality tool calls if routing confidence is weak
- Tool constraints must stay aligned with the actual tool list

## Sources

- [Gemini function calling guide](https://ai.google.dev/gemini-api/docs/function-calling)
