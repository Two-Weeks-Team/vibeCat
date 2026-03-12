# ADK Orchestrator Guide

## OVERVIEW

`backend/adk-orchestrator/` hosts the ADK graph plus navigator escalation, background replay, memory, search, and tool endpoints.

## STRUCTURE

```text
backend/adk-orchestrator/
|-- main.go
|-- internal/agents/      # memory, search, tool-use, mood, etc.
|-- internal/navigator/   # escalation + background processors
|-- internal/store/       # Firestore-backed models and persistence
|-- internal/prompts/     # prompt text and system guidance
`-- internal/models/      # request/response payloads
```

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Process entrypoint | `backend/adk-orchestrator/main.go` | env boot, OTEL, HTTP handlers |
| Navigator escalator | `backend/adk-orchestrator/internal/navigator/processor.go` | screenshot+AX resolution, background summaries |
| Agent graph build | `backend/adk-orchestrator/internal/agents/graph/` | graph wiring; inspect from `main.go` call sites |
| Memory behavior | `backend/adk-orchestrator/internal/agents/memory/` | session/task summary persistence |
| Search enrichment | `backend/adk-orchestrator/internal/agents/search/` | docs/research path |
| Replay persistence | `backend/adk-orchestrator/internal/store/` | Firestore models and writes |
| Prompt constraints | `backend/adk-orchestrator/internal/prompts/prompts.go` | assistant behavior rules |

## CONVENTIONS

- Keep navigator escalation narrow: resolve targets or visible text, do not own desktop execution.
- Fall back to heuristics when screenshot decode, model calls, or JSON decode fail.
- Background processing may enrich research and memory, but should not block hot-path execution.
- Firestore replay writes are best-effort; warnings are acceptable, silent corruption is not.

## ANTI-PATTERNS

- inventing labels or targets unsupported by AX or screenshot evidence
- moving step-by-step UI planning out of the gateway into the orchestrator
- making background replay generation part of the synchronous hot path
- dropping heuristic fallback paths when GenAI is unavailable

## COMMANDS

```bash
cd backend/adk-orchestrator && go build ./...
cd backend/adk-orchestrator && go test ./...
cd backend/adk-orchestrator && go vet ./...
docker build -t vibecat-orchestrator backend/adk-orchestrator/
```
