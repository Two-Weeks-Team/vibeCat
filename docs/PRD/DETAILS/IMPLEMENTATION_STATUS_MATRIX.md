# Implementation Status Matrix

This matrix defines what must be implemented in VibeCat and how completion is verified.

## Status Legend

- `Ready`: clear specification and test strategy are defined
- `In Progress`: partially specified and requires design completion
- `Planned`: listed, but implementation details need to be created

## Feature Matrix

| Area | Scope in VibeCat | Status | Verification Evidence in VibeCat | Acceptance Gate |
|---|---|---|---|---|
| Core Domain Types | message models, settings types, parser-safe models | Ready | `Sources/Core/*` unit tests in `Tests/Core/*` | all unit tests green |
| Image Pipeline | resize, jpeg encoding, image diff, base64 encoding | Ready | deterministic fixture tests | identical output for fixed fixtures |
| Audio Pipeline | PCM conversion and stream packet handling | Ready | frame-level conversion tests | no NaN, expected sample count |
| Prompt Assembly | system + task + fallback prompt builders | Ready | snapshot-like string tests | output contains required policy blocks |
| Realtime Transport | live websocket session, ping/pong, reconnect | In Progress | integration tests with mock server | reconnect and resume path passes |
| Vision Agent | structured multimodal analysis with schema output | In Progress | contract tests for JSON output schema | valid schema and retry on malformed output |
| Mediation Logic | speak/skip gating, cooldown, urgency handling | Ready | decision-table tests | expected decision for all rule branches |
| Adaptive Scheduler | interaction-rate based interval adaptation | In Progress | simulation tests over interaction timeline | bounded interval adjustment |
| Engagement Trigger | silence detection and proactive trigger policy | In Progress | clock-driven tests | trigger timing follows policy window |
| Orchestrator | capture -> analyze -> mediate -> speak flow | In Progress | end-to-end harness with stubs | no deadlock and bounded turn latency |
| UI Overlay | sprite state, bubble state, emotion state rendering | Planned | UI state reducer tests + smoke run | state transitions match events |
| Gesture Input | circle gesture detection and forced analysis | Ready | path sequence tests | 3-circle detection within time budget |
| Settings Persistence | durable save/load and runtime apply | Ready | persistence tests | persisted state round-trips losslessly |
| Voice/Chat Panel | hotkey open, STT flow, chat flow, history | Planned | integration tests with service stubs | open/send/receive cycle works |
| Error Reporting | typed errors and user-facing message mapping | Ready | mapping tests | all known error codes mapped |
| Cross-Session Memory | session summary, topic tracking, context bridge | Planned | memory round-trip tests | previous session context loads on new session |
| Mood Detection | frustration signals, mood classification, supportive response | Planned | mood scoring tests with simulated patterns | correct mood for repeated error + silence scenario |
| Celebration Detection | success pattern recognition, celebration cooldown | Planned | vision pattern tests for build/test success | celebration fires on test pass, respects cooldown |
| Search Assistance | search trigger, result summarization, caching | Planned | search integration tests with mock results | summarized answer returned within latency budget |
| Decision Overlay HUD | trigger display, analysis fields, confidence, cooldown | Planned | overlay toggle tests | overlay shows correct trigger and decision for each action |
| Sprite Screen Pointing | error region coordinates, sprite animation toward error | Planned | coordinate mapping tests | sprite moves toward detected error region and returns |
| Session Farewell | goodbye message, sleeping sprite, summary save | Planned | session end event tests | farewell plays on close, summary persists in Firestore |
| Graceful Fallback | TTS fallback on Gemini down, ADK timeout default | Planned | failure injection tests | agent speaks via TTS within 3s of Gemini failure |
| Privacy Controls | capture indicator, one-click pause, manual mode | Planned | capture state tests | indicator matches capture state, pause stops all captures |
| Automated Deployment | infra/ deploy and teardown scripts | Planned | script execution tests | deploy.sh creates both Cloud Run services |
| Deployment Artifacts | Cloud Run services, secrets, logging, tracing | Planned | CI build and deployment checklist | deployment proof artifacts complete |

## Completion Rule

Each area is complete only when all of the following are true:

1. Required code exists in VibeCat target structure.
2. Required tests exist and pass.
3. Acceptance gate in this matrix is satisfied.
4. Evidence link is recorded in implementation pull request notes.
