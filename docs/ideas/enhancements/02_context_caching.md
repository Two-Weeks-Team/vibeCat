# P0-2: Context Caching

## Status

Implement in the ADK orchestrator only, after token-count precheck. Do not attempt this in the Live gateway.

## Source-Verified Facts

- `LiveConnectConfig` has no cached-content field.
- In Go SDK v1.49.0 the batch request field is `GenerateContentConfig.CachedContent`.
- `CreateCachedContentConfig` supports `SystemInstruction`, `Contents`, `Tools`, and `ToolConfig`.
- Current explicit-caching docs show a 1,024-token minimum for `Gemini 2.5 Flash` and `Gemini 3 Flash Preview`.
- `Models.CountTokens()` exists and should be used instead of assuming the prompt is large enough.

## Current Repo State

- All candidate uses are in `backend/adk-orchestrator`
- Repeated system prompts exist in:
  - `internal/agents/vision/vision.go`
  - `internal/agents/tooluse/tooluse.go`
  - `internal/agents/memory/memory.go`
  - other direct `GenerateContent` call sites
- No cache manager exists

## Critical Corrections

- Use `CachedContent`, not `CachedContentName`
- Do not assume current prompts exceed the caching threshold
- Do not cache user-specific or rapidly changing prompt fragments unless they are actually reused

## Implementation Plan

### 1. Add a cache manager

Create `backend/adk-orchestrator/internal/cache/manager.go`:

- key by:
  - model
  - prompt version
  - normalized language
  - character/persona hash if applicable
  - tool declaration hash
- store:
  - cached content resource name
  - expire time
  - token count

### 2. Preflight with token counting

Before cache creation:

- call `Models.CountTokens()` with:
  - the exact `SystemInstruction`
  - any static `Contents`
  - tool declarations if they will be cached
- create a cache only if the result meets the model minimum

### 3. Pilot the feature on stable high-reuse prompts

Start with:

- `tooluse.classify()`
- `vision.analyze()` only if the system prompt alone meets threshold
- `memory.generateSummary()` only if token count and reuse justify it

Do not start with highly dynamic prompts that vary per request.

### 4. Inject cache references

For direct `GenerateContent` calls:

- set `GenerateContentConfig.CachedContent = cacheName`

For ADK `llmagent` use:

- pass a prepared `GenerateContentConfig` via `llmagent.Config.GenerateContentConfig`

## Concrete File Changes

- `backend/adk-orchestrator/main.go`
  - initialize shared cache manager when feature flag is on
- `backend/adk-orchestrator/internal/cache/manager.go`
  - create / refresh / invalidate caches
- `backend/adk-orchestrator/internal/agents/tooluse/tooluse.go`
  - use cache for classification prompt if threshold is met
- `backend/adk-orchestrator/internal/agents/vision/vision.go`
  - use cache only after count-tokens verification
- `backend/adk-orchestrator/internal/agents/memory/memory.go`
  - same rule

## Observability

Emit:

- cache created
- cache refreshed
- cache hit
- cache miss
- `CachedContentTokenCount`
- token-count precheck result

## Acceptance Criteria

1. Only batch orchestrator requests use caching.
2. Cache creation happens only for prompts that meet the token minimum.
3. Responses show non-zero `CachedContentTokenCount` on hits.
4. No correctness regressions when cache expires or is disabled.

## Risks

- Token minimum may exclude some prompts entirely
- Over-caching dynamic prompts can reduce correctness
- Cache-key mistakes can cross-contaminate languages/personas/toolsets

## Sources

- [Gemini caching guide](https://ai.google.dev/gemini-api/docs/caching)
