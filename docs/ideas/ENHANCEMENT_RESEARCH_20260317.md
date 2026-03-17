# VibeCat Enhancement Research — 2026-03-17

> Source-verified implementation brief for post-submission development.
> Scope: `docs/ideas/enhancements/01-14`.
> Verified against current repo code, `google.golang.org/genai v1.49.0`, `google.golang.org/adk v0.6.0`, and current official Gemini / Google Cloud docs.

## Executive Summary

The original research set identified the right capability buckets, but several documents mixed together:

- Gemini Live features vs batch `GenerateContent` features
- SDK-supported fields vs notebook-only examples
- cloud-safe production patterns vs local-only experiments
- new work vs functionality that VibeCat already implements

This revision turns the set into an implementation brief the development team can actually execute.

Implementation ticket breakdown lives in `docs/ideas/IMPLEMENTATION_BACKLOG_20260317.md`.

## Source-Verified Corrections

1. The current Live model `gemini-2.5-flash-native-audio-preview-12-2025` does support thinking. The official Live guide explicitly says the latest native audio model supports thinking and enables dynamic thinking by default.
2. Context caching is still batch-only for VibeCat. `LiveConnectConfig` has no cached-content field. In Go SDK v1.49.0 the request field is `GenerateContentConfig.CachedContent`, not `CachedContentName`.
3. The explicit caching minimum for the models VibeCat uses is not 2,048 tokens across the board. Current docs show `Gemini 2.5 Flash` and `Gemini 3 Flash Preview` with a 1,024-token minimum.
4. `ToolConfig` exists on `GenerateContentConfig` only. Forced function-calling modes do not apply to `LiveConnectConfig`.
5. Parallel/multi-tool use exists in Live API, but VibeCat must not execute desktop UI actions in parallel yet. The current gateway stores only one pending FC at a time; multiple function calls in one response currently create a state overwrite risk.
6. Safety handling is not greenfield work. VibeCat already has navigator risk confirmation flow in the backend and client; the enhancement is to centralize and harden it.
7. Heartbeat is not greenfield work either. The gateway already sends silent PCM every 15 seconds. The enhancement is to refactor and tune it, not invent it.
8. Browser `computer_use` is still browser-only. It does not replace macOS-wide navigation.
9. Timer-based memory consolidation is not valid on Cloud Run because instances scale to zero. Use Cloud Scheduler + Cloud Tasks + HTTP job entrypoint.
10. Filesystem MCP and local-project RAG both have the same hard boundary: Cloud Run cannot read a user's Mac filesystem. Any design that assumes server-side `WalkDir("/Users/...")` for arbitrary local workspaces is invalid.

## Decision Matrix

| ID | Topic | Decision | Release Target | Why |
|---|---|---|---|---|
| 01 | ThinkingConfig + Thought Signatures | Implement in Live with budget + metrics; gate UI thought text behind feature flag | v0.2 | Supported on current Live model; useful for reasoning and observability |
| 02 | Context Caching | Implement for batch-only calls after token-count precheck | v0.3 | Real savings possible, but only on prompts that meet minimum token threshold |
| 03 | Forced Function Calling | Implement on direct batch `GenerateContent` routes only | v0.3 | Useful for deterministic tool routing; unavailable in Live config |
| 04 | Parallel Function Calling | Do not parallelize desktop navigator actions; first fix multi-call overwrite by serializing | v0.2 | Correctness and safety matter more than theoretical speedup |
| 05 | Safety Decision Handling | Implement as hardening of existing risk flow | v0.2 | Already partially present; needs central classifier and richer UX |
| 06 | Heartbeat Pattern | Implement as refactor/tuning of existing keepalive | v0.2 | Already partially present; should be session-owned and idle-aware |
| 07 | Controlled Generation | Implement with `application/json` + `responseJsonSchema` on direct calls | v0.2 | Immediate reliability gain; reduces parsing failure |
| 08 | Native `computer_use` | Defer | Deferred | Browser-only; does not solve VibeCat's desktop scope |
| 09 | Always-On Memory | Design now, implement after scheduler/task plumbing | Deferred | Cloud Run lifecycle and Firestore size constraints require real infra work |
| 10 | MCP Integration | Local-dev first, production only with remote/bundled transport | v0.3+ | Useful, but filesystem-on-cloud assumptions must be removed |
| 11 | Real-time RAG | Implement server-owned/uploaded docs first; local workspace sync later | v0.3 | Valuable, but needs ingestion architecture that matches desktop/cloud split |
| 12 | Expand Navigator Tools | Implement click first, then scroll/system/copy; fix parser gaps | v0.2 / v0.3 | Existing Swift runtime already supports much of this |
| 13 | Progress Communication | Implement protocol/UI refinements after 01, 05, 12 | v0.2 | Current UX is strong; remaining work is mostly protocol completeness |
| 14 | Errata | Keep as validation ledger only | Current | Corrections are now merged into all docs |

## Cross-Cutting Rules

### 1. Separate Live and Batch Concerns

- `backend/realtime-gateway` is a Live API system. Only use features supported by `LiveConnectConfig`.
- `backend/adk-orchestrator` is a batch `GenerateContent` system. Use caching, tool configs, response schemas, embeddings, and MCP here.
- Do not copy examples between the two layers without checking actual SDK types first.

### 2. Respect Desktop/Cloud Boundaries

- Client-only: screen capture, AX inspection, keyboard/mouse control, local app focus, local workspace access.
- Server-only: Live session brokerage, batch model calls, search, memory persistence, vector search, remote integrations.
- Anything that needs the user's local files or local GUI must either run on the client or be explicitly uploaded/synced to the backend.

### 3. Safety Overrides Speed

- Never execute two desktop UI actions concurrently unless they are proven independent and the runtime has per-surface locking.
- Any action that can submit, delete, publish, deploy, overwrite, or reveal secrets must pass the shared risk classifier.
- Function-calling improvements are valid only if they preserve VibeCat's confirm-before-acting product behavior.

### 4. Ship Behind Flags

Add feature flags for every new capability:

- `ENABLE_LIVE_THINKING`
- `ENABLE_LIVE_THOUGHT_STREAM_UI`
- `ENABLE_BATCH_CONTEXT_CACHE`
- `ENABLE_FORCED_TOOL_ROUTING`
- `ENABLE_NAVIGATOR_FC_QUEUE`
- `ENABLE_NAVIGATOR_CLICK_TOOL`
- `ENABLE_NAVIGATOR_SCROLL_TOOL`
- `ENABLE_STRUCTURED_AGENT_OUTPUT`
- `ENABLE_RAG`
- `ENABLE_MCP`

### 5. Add Observability with Every Feature

Every enhancement must emit structured logs and metrics:

- feature enabled/disabled
- success/failure count
- latency impact
- token usage impact
- cache hit ratio where relevant
- user confirmation required / approved / denied where relevant

## Release Plan

### v0.2.0

- 01 ThinkingConfig Live enablement
- 04 Live multi-call serialization fix
- 05 Safety hardening
- 06 Heartbeat refactor
- 07 Structured outputs on direct batch calls
- 12 `navigate_click` and parser fixes
- 13 progress protocol additions needed by 01/05/12

### v0.3.0

- 02 Context caching
- 03 Forced FC on batch routes
- 12 `navigate_scroll`, `navigate_system`, `navigate_copy_selection`
- 10 MCP local-dev path and production design
- 11 RAG for uploaded/server-owned docs

### Deferred

- 08 browser `computer_use`
- 09 always-on memory consolidation
- 11 arbitrary local-workspace RAG without explicit sync/upload path
- true parallel desktop FC execution

## Required Test Gate

No enhancement is complete without:

1. Unit tests for new request/response helpers and classifiers
2. Integration tests for tool routing or queue behavior
3. Replay or fixture coverage for navigator changes
4. Manual acceptance run on macOS client for UI-affecting features
5. Log/metric verification in Cloud Run for backend features

## Primary Sources

- [Gemini Live API guide](https://ai.google.dev/gemini-api/docs/live-guide)
- [Gemini thinking guide](https://ai.google.dev/gemini-api/docs/thinking)
- [Gemini function calling guide](https://ai.google.dev/gemini-api/docs/function-calling)
- [Gemini caching guide](https://ai.google.dev/gemini-api/docs/caching)
- [Gemini structured output guide](https://ai.google.dev/gemini-api/docs/structured-output)
- [Gemini embeddings guide](https://ai.google.dev/gemini-api/docs/embeddings)
- [Gemini computer use guide](https://ai.google.dev/gemini-api/docs/computer-use)
- [Gemini models](https://ai.google.dev/gemini-api/docs/models)
- [Gemini deprecations](https://ai.google.dev/gemini-api/docs/deprecations)
- [Firestore vector search](https://cloud.google.com/firestore/native/docs/vector-search)

## Working Rule For The Team

If a feature description in an individual enhancement document conflicts with this file, follow this file and then update the feature document before implementation begins.
