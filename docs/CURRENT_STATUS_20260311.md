# VibeCat Current Status (2026-03-11)

This document is the current-state snapshot for the repository, deployment, and remaining work. Use it as the ground truth ahead of older planning or handoff notes.

## Repository Snapshot

- branch: `master`
- HEAD: `7356f8c` (`Merge pull request #124 from Two-Weeks-Team/codex/issue-triage-20260311`)
- open PRs: none
- open GitHub issues: 7

## What Exists Today

### Client

The macOS client is implemented under `VibeCat/` and includes:

- core models, settings, keychain, image and audio helpers
- localization primitives
- status bar app, onboarding window, tray animation, chat bubble, chat panel
- screen capture and screen analysis loop
- Gateway WebSocket client with reconnect flow
- speech recognition and audio playback
- recent live-state improvements: fallback status labels, listening state handling, audio-device hot-plug recovery

### Backend

The backend is split into two deployed Go services:

- `backend/realtime-gateway/`: auth, Live API session handling, WebSocket transport, ADK routing, TTS client
- `backend/adk-orchestrator/`: 9-agent ADK graph, Firestore-backed memory/state, search grounding, mood/celebration/engagement/vision logic

### Tests and Verification Assets

- Swift unit tests under `VibeCat/Tests/VibeCatTests/`
- Go package tests in both backend services
- deployment-facing smoke and behavior tests in `tests/e2e/`
- CI workflow in `.github/workflows/ci.yml`
- manual deploy workflow in `.github/workflows/cd.yml`
- infra scripts in `infra/`

## Live Deployment Baseline

### Cloud Run

- `realtime-gateway`
  - ready revision: `realtime-gateway-00040-gcd`
  - traffic: `100%`
  - canonical URL: `https://realtime-gateway-a4akw2crra-du.a.run.app`
  - anonymous health check: `GET /health` returns `200`
- `adk-orchestrator`
  - ready revision: `adk-orchestrator-00038-t4c`
  - traffic: `100%`
  - canonical URL: `https://adk-orchestrator-a4akw2crra-du.a.run.app`
  - anonymous `GET /health` returns `403`
  - authenticated `GET /health` with identity token returns `{"service":"adk-orchestrator","status":"ok"}`

### GCP Resources

- project: `vibecat-489105`
- region: `asia-northeast3`
- Firestore: `(default)` native database in `asia-northeast3`
- Secret Manager:
  - `vibecat-gemini-api-key`
  - `vibecat-gateway-auth-secret`
- Artifact Registry:
  - `projects/vibecat-489105/locations/asia-northeast3/repositories/vibecat-images`

### Observability

- Cloud Logging: active, recent startup logs observed from both Cloud Run services
- Cloud Trace: active, recent trace IDs observed on 2026-03-11
- Cloud Monitoring exporter: enabled in code
- Cloud Monitoring dashboards: none configured yet

## CI/CD Status

- latest fully green CI run: `22932978716` on commit `ac5e4bf`
- latest `master` CI run: `22933714954` on commit `7356f8c`
- current `master` CI failure:
  - job: `Client (Swift 6 / macOS) — Build + Test`
  - failing step: `Select Xcode 16+ and accept license`
  - cause: self-hosted macOS runner has no licensed Xcode installation available
- current `master` CI successes:
  - Gateway Go build/test/vet
  - Orchestrator Go build/test/vet
  - Docker image builds
- CD:
  - GitHub manual deploy workflow exists
  - GCP Cloud Build YAML exists for both backend services
  - GCP Cloud Build triggers are not configured

## Issue Status

The open issue set has been reduced to the 7 items that still map cleanly to unfinished work:

- `#57` Complete deployment and operations evidence pack
- `#64` Add privacy controls UI
- `#90` Configure Cloud Logging and Monitoring
- `#91` Configure Cloud Trace
- `#117` End-to-end companion intelligence integration test
- `#120` Implement graceful fallback for Gemini unavailability
- `#121` Implement graceful fallback for ADK Orchestrator timeout

Issues `#55`, `#56`, `#58`, `#60`, `#61`, and `#68` were closed on 2026-03-11 during the cleanup pass because they were either already implemented, consolidated into `#57`, or no longer meaningful for the current shipped/deployed baseline.

## Current Gaps That Still Matter

- privacy-trust UI is incomplete
- deployment proof docs are stale and still need final submission-grade artifacts
- logs and traces exist, but monitoring dashboard and end-to-end trace acceptance are incomplete
- degraded-mode behavior for Gemini outage and ADK timeout is not complete
- the current E2E suite does not yet prove the full companion-intelligence session story

## Documents To Trust First

- `AGENTS.md`
- `docs/CURRENT_STATUS_20260311.md`
- `docs/evidence/DEPLOYMENT_EVIDENCE.md`
- `docs/deployment/PROOF_OF_GCP_DEPLOYMENT.md`

## Documents To Treat As Historical

- most files under `.sisyphus/`
- early execution plans under `docs/PRD/DETAILS/`
- older evidence files that still refer to pre-`00040`/`00038` revisions or placeholder screenshots
