# VibeCat Implementation Backlog — 2026-03-17

> Derived from `docs/ideas/ENHANCEMENT_RESEARCH_20260317.md` and enhancement docs `01-14`.
> This is the developer ticket backlog version: ready to copy into Linear, Jira, or GitHub Issues.

## Conventions

- Priority:
  - `P0`: must land before broader enhancement work
  - `P1`: next release after P0 stabilization
  - `P2`: design / deferred
- Estimate:
  - `S`: 0.5-1 day
  - `M`: 1-3 days
  - `L`: 3-5 days
  - `XL`: multi-ticket effort
- Owners:
  - `Gateway`: `backend/realtime-gateway`
  - `Orchestrator`: `backend/adk-orchestrator`
  - `Client`: `VibeCat`
  - `Infra`: Cloud Run / Firestore / Scheduler / Tasks / ops

## Release Plan

### v0.2

- VCAT-001 feature flags and config plumbing
- VCAT-002 Live thinking config and metrics
- VCAT-003 Live multi-call queue serialization
- VCAT-004 shared safety classifier
- VCAT-005 risky-action confirmation UX
- VCAT-006 heartbeat refactor
- VCAT-007 structured output for vision
- VCAT-008 structured output for tool router
- VCAT-009 structured output for memory summaries
- VCAT-010 navigator click tool backend
- VCAT-011 navigator parser contract completion
- VCAT-012 progress protocol step totals
- VCAT-013 replay and integration test pack

### v0.3

- VCAT-101 token-count precheck and cache manager
- VCAT-102 context caching rollout
- VCAT-103 forced function calling on batch routes
- VCAT-104 navigator scroll runtime
- VCAT-105 scroll/system/copy FC exposure
- VCAT-106 MCP local-dev integration
- VCAT-107 RAG ingestion and vector retrieval

### Deferred / design-only

- VCAT-201 always-on memory architecture
- VCAT-202 MCP production transport
- VCAT-203 local workspace sync for RAG
- VCAT-204 browser `computer_use` experiment

## Dependency Graph

```text
VCAT-001
  -> VCAT-002
  -> VCAT-006
  -> VCAT-010
  -> VCAT-012
  -> VCAT-101

VCAT-003
  -> VCAT-010
  -> VCAT-013

VCAT-004
  -> VCAT-005
  -> VCAT-010
  -> VCAT-105

VCAT-007
  -> VCAT-013

VCAT-008
  -> VCAT-103
  -> VCAT-102

VCAT-009
  -> VCAT-102

VCAT-010
  -> VCAT-011
  -> VCAT-013

VCAT-101
  -> VCAT-102

VCAT-104
  -> VCAT-105

VCAT-107
  -> VCAT-203

VCAT-201
  -> future implementation only
```

## Ticket List

## VCAT-001

### Title

Feature flag and config plumbing for enhancement rollout

### Meta

- Priority: `P0`
- Owner: `Gateway`, `Orchestrator`, `Client`
- Estimate: `M`
- Depends on: none

### Scope

Add explicit feature flags so all enhancement work can land safely behind toggles.

### Files

- `backend/realtime-gateway/main.go`
- `backend/realtime-gateway/internal/live/session.go`
- `backend/adk-orchestrator/main.go`
- `VibeCat/Sources/Core/Settings.swift`
- `VibeCat/Sources/VibeCat/AppDelegate.swift`

### Deliverables

- Environment/config support for:
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

### Acceptance

- Services boot with all flags off
- Flags can be enabled independently
- Flag values are logged once at startup

## VCAT-002

### Title

Enable Live thinking config and token observability

### Meta

- Priority: `P0`
- Owner: `Gateway`
- Estimate: `M`
- Depends on: `VCAT-001`

### Scope

Add `ThinkingConfig` to Live sessions, tune budget, and record thoughts token usage.

### Files

- `backend/realtime-gateway/internal/live/session.go`
- `backend/realtime-gateway/internal/ws/handler.go`

### Deliverables

- `live.Config` supports `EnableThinking` and `ThinkingBudget`
- `buildLiveConfig()` sets `ThinkingConfig` when enabled
- usage logs include `ThoughtsTokenCount`
- debug logs record presence of `Part.Thought` and `ThoughtSignature`

### Acceptance

- Live session still connects on current native audio model
- token usage logs include thoughts token field
- no regression in audio/tool-call flow

## VCAT-003

### Title

Serialize multiple Live function calls with a queue

### Meta

- Priority: `P0`
- Owner: `Gateway`
- Estimate: `L`
- Depends on: `VCAT-001`

### Scope

Replace single pending FC state with queued serial execution for multiple function calls returned in one Live response.

### Files

- `backend/realtime-gateway/internal/ws/handler.go`
- `backend/realtime-gateway/internal/ws/navigator.go`

### Deliverables

- pending FC queue data structure
- one active FC chain at a time
- ordered dispatch of multiple function calls from one response
- explicit clear/cancel behavior for queue

### Acceptance

- two-call and three-call fixtures run without overwriting prior state
- desktop actions remain serialized

## VCAT-004

### Title

Extract shared safety classifier

### Meta

- Priority: `P0`
- Owner: `Gateway`
- Estimate: `M`
- Depends on: none

### Scope

Move risk detection into a shared package used by both navigator planning and direct FC handlers.

### Files

- `backend/realtime-gateway/internal/safety/classifier.go`
- `backend/realtime-gateway/internal/ws/navigator.go`
- `backend/realtime-gateway/internal/ws/handler.go`

### Deliverables

- `Assessment` model with risk level and reason
- coverage for shell, secret, submit, deploy, delete, suspicious URL patterns
- unified blocked/denied/timeout response mapping

### Acceptance

- planner and direct FC handlers return the same risk classification for the same action
- unit tests cover at least 15 representative patterns

## VCAT-005

### Title

Upgrade risky-action confirmation UX

### Meta

- Priority: `P0`
- Owner: `Client`
- Estimate: `M`
- Depends on: `VCAT-004`

### Scope

Replace light-weight chat-only risky-action prompting with explicit allow/block UI.

### Files

- `VibeCat/Sources/VibeCat/AppDelegate.swift`
- `VibeCat/Sources/VibeCat/CompanionChatPanel.swift`
- optionally `VibeCat/Sources/VibeCat/CatPanel.swift`

### Deliverables

- action summary
- risk reason
- approve button
- deny button
- timeout fallback UI state

### Acceptance

- risky action shows explicit controls
- approve and deny both propagate correct backend responses

## VCAT-006

### Title

Move heartbeat into Live session lifecycle

### Meta

- Priority: `P0`
- Owner: `Gateway`
- Estimate: `M`
- Depends on: `VCAT-001`

### Scope

Refactor current silent-audio keepalive into `live.Session`.

### Files

- `backend/realtime-gateway/internal/live/session.go`
- `backend/realtime-gateway/internal/ws/handler.go`

### Deliverables

- session-owned heartbeat
- idle-aware send suppression
- logs/counters for sent/skipped/failed heartbeat events

### Acceptance

- idle connection survives staging soak test
- duplicate handler-owned keepalive loop removed

## VCAT-007

### Title

Structured output for vision agent

### Meta

- Priority: `P0`
- Owner: `Orchestrator`
- Estimate: `M`
- Depends on: `VCAT-001`

### Scope

Enforce typed JSON output for screen analysis.

### Files

- `backend/adk-orchestrator/internal/agents/vision/vision.go`
- `backend/adk-orchestrator/internal/models/models.go`

### Deliverables

- response schema for `VisionAnalysis`
- deterministic fallback on invalid JSON
- updated tests

### Acceptance

- invalid free-form model output no longer silently becomes primary success path

## VCAT-008

### Title

Structured output for tool router

### Meta

- Priority: `P0`
- Owner: `Orchestrator`
- Estimate: `M`
- Depends on: `VCAT-001`

### Scope

Make tool routing return strongly typed JSON with confidence and reason.

### Files

- `backend/adk-orchestrator/internal/agents/tooluse/tooluse.go`

### Deliverables

- JSON schema for:
  - `tool_kind`
  - `confidence`
  - `reason`
- stricter parsing
- test coverage for all tool classes

### Acceptance

- parse failure rate drops to zero in unit fixtures

## VCAT-009

### Title

Structured output for memory summaries

### Meta

- Priority: `P0`
- Owner: `Orchestrator`
- Estimate: `S`
- Depends on: `VCAT-001`

### Scope

Normalize summary generation output so memory writes are typed and stable.

### Files

- `backend/adk-orchestrator/internal/agents/memory/memory.go`

### Deliverables

- JSON schema for summary payload
- optional unresolved issue list

### Acceptance

- memory summary generation no longer relies on permissive text parsing

## VCAT-010

### Title

Add `navigate_click` function-calling support

### Meta

- Priority: `P0`
- Owner: `Gateway`
- Estimate: `L`
- Depends on: `VCAT-001`, `VCAT-003`, `VCAT-004`

### Scope

Expose semantic click and coordinate click to Gemini through a new Live FC tool.

### Files

- `backend/realtime-gateway/internal/live/session.go`
- `backend/realtime-gateway/internal/ws/handler.go`
- `backend/realtime-gateway/internal/ws/navigator.go`

### Deliverables

- new `navigate_click` function declaration
- handler that prefers `press_ax`
- coordinate fallback with `screenBasisId` propagation
- risk-classifier integration

### Acceptance

- model can click visible semantic controls
- stale-screen coordinate click is rejected safely

## VCAT-011

### Title

Complete click-related parser and data contract on the client

### Meta

- Priority: `P0`
- Owner: `Client`
- Estimate: `M`
- Depends on: `VCAT-010`

### Scope

Fix missing fields in the Swift message parser and model decoding.

### Files

- `VibeCat/Sources/Core/AudioMessageParser.swift`
- `VibeCat/Sources/Core/NavigatorModels.swift`

### Deliverables

- parse `clickX`
- parse `clickY`
- parse `screenBasisId`
- parse `verificationCue`

### Acceptance

- coordinate-click steps generated by backend round-trip intact to Swift runtime

## VCAT-012

### Title

Add total-step metadata to navigator progress protocol

### Meta

- Priority: `P0`
- Owner: `Gateway`, `Client`
- Estimate: `M`
- Depends on: `VCAT-001`

### Scope

Make multi-step progress explicit in the protocol so the overlay can show real totals.

### Files

- `backend/realtime-gateway/internal/ws/handler.go`
- `VibeCat/Sources/Core/AudioMessageParser.swift`
- `VibeCat/Sources/VibeCat/AppDelegate.swift`
- `VibeCat/Sources/VibeCat/NavigatorOverlayPanel.swift`

### Deliverables

- add `stepNumber` and `totalSteps`, or new `navigator.planStarted`
- client passes real totals to overlay

### Acceptance

- 3-step task shows `1/3`, `2/3`, `3/3`

## VCAT-013

### Title

Build v0.2 replay and integration verification pack

### Meta

- Priority: `P0`
- Owner: `Gateway`, `Client`
- Estimate: `M`
- Depends on: `VCAT-003`, `VCAT-007`, `VCAT-010`, `VCAT-011`, `VCAT-012`

### Scope

Create the minimum regression suite for the v0.2 changes.

### Files

- `backend/realtime-gateway/internal/ws/*_test.go`
- `backend/realtime-gateway/internal/ws/testdata/navigator_replays/*`
- `VibeCat/Tests/VibeCatTests/*`

### Deliverables

- multi-call queue test
- risky-action flow test
- click parser test
- step-total progress test
- manual acceptance checklist

### Acceptance

- CI covers all new core flows

## VCAT-101

### Title

Add token-count precheck and shared cache manager

### Meta

- Priority: `P1`
- Owner: `Orchestrator`
- Estimate: `M`
- Depends on: `VCAT-001`

### Scope

Build shared cache manager with token-count gate before cache creation.

### Files

- `backend/adk-orchestrator/internal/cache/manager.go`
- `backend/adk-orchestrator/main.go`

### Deliverables

- count-tokens preflight
- keyed cache registry
- refresh-before-expiry logic
- cache hit/miss metrics

### Acceptance

- cache is created only when prompt/tool payload meets threshold

## VCAT-102

### Title

Roll out batch context caching to selected orchestrator calls

### Meta

- Priority: `P1`
- Owner: `Orchestrator`
- Estimate: `M`
- Depends on: `VCAT-101`

### Scope

Pilot caching on the highest-reuse direct batch calls.

### Files

- `backend/adk-orchestrator/internal/agents/tooluse/tooluse.go`
- `backend/adk-orchestrator/internal/agents/vision/vision.go`
- `backend/adk-orchestrator/internal/agents/memory/memory.go`

### Deliverables

- cache use where token minimum is satisfied
- metrics on `CachedContentTokenCount`

### Acceptance

- at least one target path shows real cache hits in staging

## VCAT-103

### Title

Apply forced function-calling to direct batch routes

### Meta

- Priority: `P1`
- Owner: `Orchestrator`
- Estimate: `M`
- Depends on: `VCAT-008`

### Scope

Use `ToolConfig` with `ANY` / `NONE` on direct `GenerateContent` routes where routing confidence is already known.

### Files

- `backend/adk-orchestrator/internal/agents/tooluse/tooluse.go`

### Deliverables

- `Mode: ANY` for known tool kind
- `Mode: NONE` for text-only classifier cases
- logs on chosen tool mode

### Acceptance

- routed cases always return function calls when expected

## VCAT-104

### Title

Implement scroll runtime in the macOS client

### Meta

- Priority: `P1`
- Owner: `Client`
- Estimate: `M`
- Depends on: `VCAT-001`

### Scope

Add `scroll` action to `NavigatorActionType` and execute it with CGEvent.

### Files

- `VibeCat/Sources/Core/NavigatorModels.swift`
- `VibeCat/Sources/VibeCat/AccessibilityNavigator.swift`

### Deliverables

- vertical scroll runtime
- basic verification path

### Acceptance

- manual test passes on Chrome and code editor surfaces

## VCAT-105

### Title

Expose scroll, system, and copy actions as FC tools

### Meta

- Priority: `P1`
- Owner: `Gateway`, `Client`
- Estimate: `L`
- Depends on: `VCAT-004`, `VCAT-104`

### Scope

Expose currently available runtime actions through explicit function declarations.

### Files

- `backend/realtime-gateway/internal/live/session.go`
- `backend/realtime-gateway/internal/ws/handler.go`
- `backend/realtime-gateway/internal/ws/navigator.go`
- parser/model files in `VibeCat`

### Deliverables

- `navigate_scroll`
- `navigate_copy_selection`
- `navigate_system_action`

### Acceptance

- all three tool types can be generated and executed end-to-end

## VCAT-106

### Title

Integrate MCP for local development

### Meta

- Priority: `P1`
- Owner: `Orchestrator`
- Estimate: `M`
- Depends on: `VCAT-001`

### Scope

Add feature-flagged MCP integration with a local-dev transport only.

### Files

- `backend/adk-orchestrator/internal/mcp/*`
- `backend/adk-orchestrator/internal/agents/graph/graph.go`

### Deliverables

- MCP transport factory
- local-dev only enablement
- confirmation required on write tools

### Acceptance

- service still boots when MCP is unavailable

## VCAT-107

### Title

Implement uploaded-doc RAG with Firestore vector search

### Meta

- Priority: `P1`
- Owner: `Orchestrator`, `Infra`
- Estimate: `XL`
- Depends on: `VCAT-001`

### Scope

Build production-valid RAG around uploaded or server-owned docs only.

### Files

- `backend/adk-orchestrator/internal/rag/*`
- `backend/adk-orchestrator/main.go`
- Firestore index/config provisioning

### Deliverables

- chunking
- embeddings with `RETRIEVAL_DOCUMENT` / `RETRIEVAL_QUERY`
- Firestore vector storage
- nearest-neighbor retrieval
- context injection into orchestrator requests

### Acceptance

- top-K retrieval works against uploaded/server-side docs in staging

## VCAT-201

### Title

Design Cloud Run-safe always-on memory architecture

### Meta

- Priority: `P2`
- Owner: `Orchestrator`, `Infra`
- Estimate: `M`
- Depends on: none

### Scope

Produce an implementation design for scheduled consolidation jobs and subcollection-based memory storage.

### Deliverables

- architecture doc
- endpoint contract
- Firestore schema change plan
- cost estimate

## VCAT-202

### Title

Design MCP production transport and deployment model

### Meta

- Priority: `P2`
- Owner: `Orchestrator`, `Infra`
- Estimate: `S`
- Depends on: `VCAT-106`

### Scope

Decide between:

- bundled binary
- remote MCP service

### Deliverables

- production transport decision
- security model
- rollout checklist

## VCAT-203

### Title

Design local workspace sync pipeline for RAG

### Meta

- Priority: `P2`
- Owner: `Client`, `Orchestrator`
- Estimate: `M`
- Depends on: `VCAT-107`

### Scope

Define how local docs move from desktop to cloud safely.

### Deliverables

- allowed file types
- upload/sync trigger model
- delete/update semantics
- privacy and scoping rules

## VCAT-204

### Title

Run browser-only `computer_use` comparison spike

### Meta

- Priority: `P2`
- Owner: `Gateway`
- Estimate: `S`
- Depends on: none

### Scope

Experimental branch only. Compare browser-only computer-use against current CDP flow.

### Deliverables

- spike notes
- measured pros/cons
- keep / drop recommendation

## Ready-First Order

If the team wants the safest immediate sequence, start in this exact order:

1. VCAT-001
2. VCAT-003
3. VCAT-004
4. VCAT-002
5. VCAT-006
6. VCAT-007
7. VCAT-008
8. VCAT-009
9. VCAT-010
10. VCAT-011
11. VCAT-012
12. VCAT-005
13. VCAT-013

## Definition of Done

A ticket is not done until all of the following are true:

- code merged behind the intended flag
- unit/integration tests added or updated
- staging/manual verification completed if user-visible
- logs/metrics added if backend behavior changed
- related enhancement doc updated if scope changed during implementation
