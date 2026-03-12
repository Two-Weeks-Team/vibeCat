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
|-- internal/cdp/       # Chrome DevTools Protocol controller
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
| Proactive Companion prompt | `backend/realtime-gateway/internal/live/session.go` | `commonLivePrompt, FC tool declarations` |
| FC tool handlers | `backend/realtime-gateway/internal/ws/handler.go` | `handleNavigate{Hotkey,FocusApp,OpenURL,TypeAndSubmit}ToolCall` |
| Self-healing retry | `backend/realtime-gateway/internal/ws/handler.go` | `pendingFC* fields, stepRetryCount, max 2 retries` |
| Vision verification | `backend/realtime-gateway/internal/ws/handler.go` | `pendingVisionVerification, requestScreenCapture` |
| Chrome controller | `backend/realtime-gateway/internal/cdp/chrome.go` | `chromedp CDP: Click, Type, Navigate, Scroll, Screenshot` |

## CONVENTIONS

- `main.go` owns service bootstrapping, OTEL setup, and route registration.
- Navigator execution is single-active-task per session; replacement and clarification are first-class flows.
- UI actions stay narrow: classify intent, plan one step, refresh context, verify outcome.
- Keep explicit `/health` and `/readyz` handlers; websocket entrypoint stays `/ws/live`.
- Firestore is additive fallback state, not a reason to break in-memory fast paths.
- Navigator uses 5 FC tools: text_entry, hotkey, focus_app, open_url, type_and_submit.
- Self-healing retries max 2 times with alternative grounding sources.
- Vision verification via ADK screenshot analysis after risky actions.
- Proactive Companion identity: OBSERVE → SUGGEST → WAIT → ACT → FEEDBACK.

## ANTI-PATTERNS

- letting insertion requests succeed as focus-only without resolved text
- blind action planning when intent is ambiguous or risk is high
- bypassing auth on websocket paths
- moving desktop execution details into the gateway
- removing replay fixtures or navigator metrics to hide regressions
- sending multiple FC steps simultaneously (use pendingFC sequential mechanism)
- skipping vision verification for risky or complex actions

## COMMANDS

```bash
cd backend/realtime-gateway && go build ./...
cd backend/realtime-gateway && go test ./...
cd backend/realtime-gateway && go vet ./...
docker build -t vibecat-gateway backend/realtime-gateway/
```
