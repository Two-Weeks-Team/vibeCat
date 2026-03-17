# P2-9: Always-On Memory Agent

## Status

Design now. Defer implementation until scheduler/task infrastructure is in place.

## Source-Verified Facts

- Current repo memory is passive summary storage in Firestore.
- Cloud Run instances are ephemeral and scale to zero.
- A timer-running consolidator inside the service is not a reliable production design.

## Current Repo State

- `backend/adk-orchestrator/internal/agents/memory/memory.go`
  - retrieves memory at session start
  - writes summaries at session end / task end
- `backend/adk-orchestrator/internal/store/models.go`
  - `MemoryEntry` only stores:
    - recent summaries
    - known topics
- `backend/adk-orchestrator/internal/store/firestore.go`
  - stores everything in `users/{userId}/memory/data`

## Required Architecture Change

### Replace in-process timers with scheduled jobs

Use:

- Cloud Scheduler
- Cloud Tasks
- HTTP consolidation endpoint in `adk-orchestrator`

Flow:

```text
Cloud Scheduler
  -> Cloud Tasks
  -> POST /memory/consolidate
  -> orchestrator consolidates one batch of users
```

### Break memory into multiple documents

Do not keep the full memory graph inside one Firestore document forever.

Use:

- `users/{userId}/memory/data`
- `users/{userId}/memory/entities/{entityId}`
- `users/{userId}/memory/connections/{connectionId}`
- `users/{userId}/memory/insights/{insightId}`

## Implementation Scope

### Phase 1

- extend memory schema with entities and insights
- add consolidation endpoint
- batch over users with recent activity only
- use structured JSON output for extraction

### Phase 2

- relation graph
- importance scoring
- retrieval ranking by task type and recency

## Concrete File Changes

- `backend/adk-orchestrator/internal/store/models.go`
  - add entity / insight records
- `backend/adk-orchestrator/internal/store/firestore.go`
  - add CRUD helpers for subcollections
- `backend/adk-orchestrator/internal/agents/memory/`
  - add consolidator service or package
- `backend/adk-orchestrator/main.go`
  - add `/memory/consolidate` handler

## Retrieval Rule

Do not inject all memory into every request.

Retrieve only:

- top recent summaries
- top relevant entities
- unresolved issues with high relevance

## Acceptance Criteria

1. Consolidation runs from Scheduler/Tasks, not in-process timers.
2. Firestore document size remains bounded.
3. Retrieval payload is compact and relevant.

## Risks

- Background cost growth
- bad entity extraction can pollute memory
- stale insights if consolidation runs without good activity filtering

## Sources

- [Gemini thinking guide](https://ai.google.dev/gemini-api/docs/thinking)
