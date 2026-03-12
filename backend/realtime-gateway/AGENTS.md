# Realtime Gateway Guide

## OVERVIEW

`backend/realtime-gateway/` is the Cloud Run WebSocket gateway that handles auth, Gemini Live session transport, intent/risk gating, step planning, and navigator task state.

## STRUCTURE

```text
backend/realtime-gateway/
|-- main.go
|-- internal/auth/      # JWT registration, refresh, middleware
|-- internal/live/      # Gemini Live session handling
|-- internal/tts/       # TTS bridge
|-- internal/ws/        # websocket handler, navigator planner, metrics, state
`-- cmd/videotest/      # manual Live API/video probe utility
```

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Process entrypoint | `backend/realtime-gateway/main.go` | env boot, OTEL, `/health`, `/readyz`, `/ws/live` |
| WebSocket runtime | `backend/realtime-gateway/internal/ws/handler.go` | session state, routing, processing states |
| Navigator planning | `backend/realtime-gateway/internal/ws/navigator.go` | intent classes, steps, task/session state |
| Low-confidence escalation | `backend/realtime-gateway/internal/ws/navigator_confidence.go` | ADK escalator bridge |
| ADK HTTP client | `backend/realtime-gateway/internal/adk/` | orchestrator client package used by websocket flows |
| Action state persistence | `backend/realtime-gateway/internal/ws/action_state_store.go` | in-memory + Firestore state |
| Metrics | `backend/realtime-gateway/internal/ws/metrics.go` | navigator and websocket metrics |
| Auth surface | `backend/realtime-gateway/internal/auth/` | register/refresh/middleware |

## CONVENTIONS

- `main.go` owns service bootstrapping, OTEL setup, and route registration.
- Navigator execution is single-active-task per session; replacement and clarification are first-class flows.
- UI actions stay narrow: classify intent, plan one step, refresh context, verify outcome.
- Keep explicit `/health` and `/readyz` handlers; websocket entrypoint stays `/ws/live`.
- Firestore is additive fallback state, not a reason to break in-memory fast paths.

## ANTI-PATTERNS

- letting insertion requests succeed as focus-only without resolved text
- blind action planning when intent is ambiguous or risk is high
- bypassing auth on websocket paths
- moving desktop execution details into the gateway
- removing replay fixtures or navigator metrics to hide regressions

## COMMANDS

```bash
cd backend/realtime-gateway && go build ./...
cd backend/realtime-gateway && go test ./...
cd backend/realtime-gateway && go vet ./...
docker build -t vibecat-gateway backend/realtime-gateway/
```
