# VibeCat Current Status (2026-03-11)

This is the current repository and submission snapshot for the **UI Navigator** pivot.

Cross-check deployment and submission proof with:

- `docs/evidence/DEPLOYMENT_EVIDENCE.md`
- `docs/deployment/PROOF_OF_GCP_DEPLOYMENT.md`
- `docs/FINAL_ARCHITECTURE.md`

## Submission Truth

- category: **UI Navigator**
- product framing: **desktop UI navigator for developer workflows on macOS**
- contract: **acts when intent is clear, asks when it is not**
- hero surfaces: **Antigravity IDE, Terminal, Chrome**

Historical companion framing should be treated as archival unless a document was explicitly rewritten for the navigator pivot.

## What Is Implemented

### Client

The macOS client now provides:

- Gemini Live + VAD PM session
- chat input and clarification flow
- AX-first local action worker
- before-action context with screenshot + AX metadata
- input-field-aware `focus -> paste` execution
- screenshot-backed text extraction for screen-derived typing requests
- deterministic macOS system actions for basic interactions like volume control
- wrong-target-aware verification

### Gateway

The Realtime Gateway now provides:

- one-active-task runtime with replacement handoff
- Firestore-backed `ActionStateStore` plus in-memory cache
- reconnect-safe lease enforcement and stale-connection rejection
- risk gating and clarification prompts
- narrow confidence escalator invocation for low-confidence targets
- separate planning lanes for UI target steps, screen-derived text entry, and macOS system actions
- text-entry payload resolution for explicit text, assistant/self references, and visible screen-derived text
- guardrail that prevents `focus-only` completion when an insertion request still lacks text to type
- step history persistence and per-step outcome tracking
- navigator metrics and replay fixtures
- async background lane dispatch after task completion

### Orchestrator

The ADK Orchestrator now provides:

- multimodal confidence escalator at `/navigator/escalate`
- visible-text extraction support for screenshot-derived typing commands
- async task summary / replay labeling / memory write path at `/navigator/background`
- retained search, tool, analyze, and session-memory endpoints
- Firestore-backed replay persistence

## Implementation Plan Status

`docs/PRD/DETAILS/UI_NAVIGATOR_IMPLEMENTATION_PLAN_20260311.md` is now reflected in code:

1. `Live PM + single-task worker` boundary lock
2. action state externalization
3. before-action context expansion
4. confidence escalator
5. background intelligence lane
6. continuous evaluation metrics + replay fixtures
7. submission-pack alignment docs

## Evaluation Coverage

Planner and worker regression coverage now includes:

- ambiguous command
- risky command
- low-confidence target
- active-task replacement
- Antigravity failure-state replay fixture
- Chrome docs lookup replay fixture
- Terminal command re-run replay fixture
- input field focus and text insertion replay fixture

Fixture location:

- `backend/realtime-gateway/internal/ws/testdata/navigator_replays/`

## Deployment Baseline

- project: `vibecat-489105`
- region: `asia-northeast3`
- gateway URL: `https://realtime-gateway-a4akw2crra-du.a.run.app`
- orchestrator URL: `https://adk-orchestrator-a4akw2crra-du.a.run.app`
- Firestore: `(default)` native database
- required secrets: present

## Remaining Non-Plan Work

The implementation plan itself is now represented in code, but production hardening items outside that plan still remain in the issue tracker, especially:

- deployment/operations evidence polish
- privacy controls UI
- Cloud Monitoring / Trace dashboard completion
- end-to-end fallback polish for Gemini / ADK failure modes
