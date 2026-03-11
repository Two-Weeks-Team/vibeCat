# UI Navigator Implementation Plan (2026-03-11)

## Goal

Ship VibeCat as a production-grade **desktop UI navigator for developer workflows on macOS** with:

- one user-facing `Live PM` session powered by Gemini Live + VAD
- one `single-task action worker` for executable UI work
- safe-immediate execution for low-risk actions
- explicit clarification and replacement handoff for ambiguity or concurrent requests
- verifiable cross-app behavior on Antigravity IDE, Terminal, and Chrome

This document is the current plan of record for implementation sequencing.

## Constraints

- Gemini Live + VAD remains the primary user interaction path.
- The system must never run multiple action tasks in parallel for one user session.
- All model access remains server-side.
- The macOS client remains responsible for local UI execution, capture, and playback.
- Low-confidence targeting must degrade to guided mode rather than guessing.

## Current Baseline

Already implemented or in progress:

- `Live PM` kept separate from screen-triggered proactive commentary
- `single-task action worker` runtime split between gateway and macOS client
- input-field-aware `focus -> paste` planning for text entry
- task-aware navigator protocol using `taskId`
- stale refresh rejection and active-task replacement clarification
- audio input device recovery and system-default input following

## Phase 1: Runtime Boundary Lock

Objective: make the `PM plane` and `action plane` unambiguous.

### Deliverables

- keep Gemini Live session responsible only for:
  - user speech
  - clarification
  - summaries
  - guided explanations
- keep navigator worker responsible only for:
  - task creation
  - step planning
  - risk decisions
  - step verification
  - completion/failure state
- ensure every executable path carries:
  - `taskId`
  - `command`
  - `stepId`
  - `status`

### Code Targets

- `backend/realtime-gateway/internal/live/`
- `backend/realtime-gateway/internal/ws/handler.go`
- `backend/realtime-gateway/internal/ws/navigator.go`
- `VibeCat/Sources/VibeCat/GatewayClient.swift`
- `VibeCat/Sources/VibeCat/NavigatorActionWorker.swift`

### Acceptance

- one active task max per session
- new action command during active task triggers replacement clarification
- explanation-only requests never enter the local executor

## Phase 2: Action State Externalization

Objective: move task state out of connection-local memory.

### Deliverables

- define `ActionStateStore` abstraction
- persist the following by `sessionId + deviceId + taskId`:
  - active task metadata
  - pending prompt kind
  - current step id
  - last verified context hash
  - task timestamps
- support reconnect-safe task continuation
- support stale connection rejection after reconnect

### Recommended Storage

- primary: Firestore
- optional cache: in-memory per instance for hot reads

### Data Model

- `ActionTask`
  - `taskId`
  - `command`
  - `status`
  - `riskState`
  - `promptState`
  - `currentStepId`
  - `stepIndex`
  - `createdAt`
  - `updatedAt`
- `ActionContextSnapshot`
  - `appName`
  - `bundleId`
  - `windowTitle`
  - `focusedRole`
  - `focusedLabel`
  - `selectedTextHash`
  - `axSnapshotHash`

### Acceptance

- task survives gateway reconnect
- refresh from stale task or stale connection is rejected
- completion clears persisted active task state

## Phase 3: Before-Action Context Schema

Objective: improve deterministic targeting before any action is planned.

### Deliverables

- expand navigator context to include:
  - `frontmostBundleId`
  - `accessibilityPermission`
  - `focusStableMs`
  - `captureConfidence`
  - `lastInputFieldDescriptor`
  - `visibleInputCandidateCount`
- normalize AX summaries for:
  - input fields
  - buttons
  - links
  - tabs
  - menus
- compute context hash for verification and dedupe

### Acceptance

- input field detection works on:
  - Chrome search box
  - Chrome address bar
  - Antigravity inline prompt
  - Terminal prompt
- verification can compare `before` and `after` with stable hashes

## Phase 4: Confidence Escalator

Objective: add a narrow multimodal escalation path only when AX confidence is low.

### Deliverables

- keep `AX-first` as default
- add escalation only when:
  - multiple plausible targets exist
  - no target label is strong enough
  - the target surface is composite or virtualized
- escalation inputs:
  - AX snapshot
  - current screenshot
  - selected text
  - focused element metadata
- return:
  - resolved descriptor
  - confidence
  - fallback recommendation

### Pattern

- this is not a general multi-agent system
- this is a narrow confidence escalator behind one action worker

### Acceptance

- blind click rate remains zero
- guided mode rate drops on ambiguous browser and IDE surfaces

## Phase 5: Background Intelligence Lane

Objective: move expensive reasoning off the action hot path.

### Deliverables

- keep action hot path limited to:
  - PM
  - action controller
  - executor
  - verifier
- use ADK/orchestrator asynchronously for:
  - post-task summaries
  - memory writes
  - docs research enrichment
  - replay labeling
- never block step execution on mood or celebration logic

### Acceptance

- action latency remains stable even when research/memory work is active
- post-task summaries still appear after completion

## Phase 6: Continuous Evaluation

Objective: measure action quality, not only end results.

### Deliverables

- emit metrics for:
  - `time_to_first_action_ms`
  - `clarification_rate`
  - `task_replacement_rate`
  - `guided_mode_rate`
  - `step_verification_fail_rate`
  - `input_field_focus_success_rate`
  - `wrong_target_rate`
- store replay fixtures for:
  - Antigravity failure state
  - Chrome docs lookup
  - Terminal command re-run
  - input field focus and text insertion
- add eval cases for:
  - ambiguous command
  - risky command
  - low-confidence target
  - active-task replacement

### Acceptance

- every hero workflow run emits complete per-step traces
- replay fixtures can be used to compare regressions

## Phase 7: Submission Pack Lock

Objective: align product, demo, and written story.

### Deliverables

- hero demo locked to:
  - Antigravity
  - Terminal
  - Chrome
- architecture diagram updated to:
  - Live PM
  - single-task worker
  - AX executor
  - background intelligence lane
  - observability
- Devpost copy aligned to:
  - natural intent
  - asks when unclear
  - acts one step at a time
- Dev.to drafts aligned to the pivot

### Acceptance

- demo matches runtime truth
- docs and blog drafts stop describing VibeCat as primarily a proactive companion

## Immediate Next Implementation Slice

The next code slice after the current worker split should be:

1. add `ActionStateStore` abstraction in the gateway
2. persist active task state across reconnects
3. extend navigator context with stable pre-action fields
4. add step-level metrics and replay-friendly structured logs

This is the highest-leverage path because it improves reliability, reconnect behavior, and submission evidence at the same time.
