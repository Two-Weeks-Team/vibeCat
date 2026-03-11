# VibeCat Current Status (2026-03-11)

This document is the current-state snapshot for the repository, deployment, and submission direction.

Cross-check deployment and submission evidence with `docs/evidence/DEPLOYMENT_EVIDENCE.md` and `docs/deployment/PROOF_OF_GCP_DEPLOYMENT.md`.

## Submission Direction

- challenge category: **UI Navigator**
- product framing: **desktop UI navigator for developer workflows on macOS**
- interaction contract: **acts when intent is clear, asks when it is not**
- gold-tier workflow surfaces: **Antigravity IDE, Terminal, Chrome**

Historical Live Agent planning still exists in the repo, but it is no longer the submission truth.

## Repository Snapshot

- branch: `codex/ui-navigator-pivot`
- working tree: active implementation branch with navigator pivot in progress
- backend services: deployed in `asia-northeast3`
- client: Swift 6 macOS app under `VibeCat/`

## What Exists Today

### Client

The macOS client already includes:

- status bar shell, onboarding, tray animation, overlay cat, chat panel
- ScreenCaptureKit capture loop
- speech recognition and audio playback
- gateway websocket client with reconnect handling
- processing state bubbles and status UI

Navigator-direction additions now become the primary path:

- command-driven chat input
- desktop context capture for app/window/focused element state
- accessibility-backed action execution
- clarification prompts for ambiguous requests
- visible step planning and verification UI

### Backend

The backend remains split into two Go services:

- `backend/realtime-gateway/`: websocket transport, auth, Gemini Live integration, navigator intent/risk/step handling
- `backend/adk-orchestrator/`: ADK graph for contextual analysis, search, and retained supporting intelligence

### Deployed Baseline

- project: `vibecat-489105`
- region: `asia-northeast3`
- gateway URL: `https://realtime-gateway-a4akw2crra-du.a.run.app`
- orchestrator URL: `https://adk-orchestrator-a4akw2crra-du.a.run.app`
- Firestore: `(default)` native database
- Secret Manager: Gemini API key and gateway auth secret present

## Submission-Critical Acceptance

The current submission-critical goals are:

1. natural-language intent inference without exact trigger phrases
2. ambiguity gate with a single clarification prompt
3. safe-immediate execution for low-risk actions
4. no blind clicks on low-confidence targets
5. one-step plan -> execute -> verify -> replan loop
6. hero workflow across Antigravity IDE, Terminal, and Chrome
7. Cloud Run, trace, logging, monitoring, and proof assets aligned with the UI Navigator story

## Documents To Trust First

- `README.md`
- `docs/CURRENT_STATUS_20260311.md`
- `docs/FINAL_ARCHITECTURE.md`
- `docs/analysis/DEMO_STORYBOARD.md`
- `docs/evidence/DEPLOYMENT_EVIDENCE.md`
- `docs/deployment/PROOF_OF_GCP_DEPLOYMENT.md`

These are the only documents that should be used to support current UI Navigator submission claims without additional validation.

## Documents To Treat As Historical

- `docs/PRD/LIVE_AGENTS_PRD.md`
- `docs/SPEECH_BUBBLE_ARCHITECTURE.md`
- `docs/SCREEN_CAPTURE_REDESIGN.md`
- most older `docs/analysis/` files that still describe proactive companion behavior as the main submission path

## Current Work Focus

- document-first UI Navigator conversion
- gateway intent classification, ambiguity handling, and risk gating
- client-side accessibility execution and verification loop
- Antigravity IDE + Terminal + Chrome golden workflow hardening
- final demo/proof asset alignment
