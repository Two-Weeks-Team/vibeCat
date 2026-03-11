# PROJECT KNOWLEDGE BASE

**Generated:** 2026-03-11
**Branch:** `master`
**HEAD:** `7356f8c` (`Merge pull request #124 from Two-Weeks-Team/codex/issue-triage-20260311`)
**Repo State:** `master`, no open PRs

## OVERVIEW

VibeCat is now an implemented and deployed macOS desktop companion for solo developers. The repository contains:

- a Swift 6 macOS client under `VibeCat/`
- a Go Realtime Gateway under `backend/realtime-gateway/`
- a Go ADK Orchestrator under `backend/adk-orchestrator/`
- deployed Cloud Run services in `asia-northeast3`
- deployment scripts, CI/CD workflows, and E2E smoke tests

The original project-planning documents remain useful for architecture intent, but they no longer describe the actual implementation state by themselves. For the current ground truth, start with `docs/CURRENT_STATUS_20260311.md` and the deployment evidence docs.

## CURRENT SNAPSHOT

### Implemented

- Swift client foundations: settings, keychain, image/audio primitives, localization
- macOS app shell: status bar, onboarding, tray animation, chat panel, overlay HUD
- live transport: Gateway WebSocket client, reconnect handling, audio playback
- capture/audio: ScreenCaptureKit flow, speech recognition, audio-device hot-plug recovery
- backend gateway: auth, Live API session handling, ADK routing, TTS client, structured logs
- backend orchestrator: 9-agent graph, Firestore store, search grounding, mood/memory/vision logic
- deployment: Cloud Run, Artifact Registry, Secret Manager, Firestore, GitHub Actions CI/CD

### Live Deployment Baseline

- GCP project: `vibecat-489105`
- region: `asia-northeast3`
- gateway: `realtime-gateway-00040-gcd`, URL `https://realtime-gateway-a4akw2crra-du.a.run.app`
- orchestrator: `adk-orchestrator-00038-t4c`, URL `https://adk-orchestrator-a4akw2crra-du.a.run.app`
- Firestore: `(default)` native database in `asia-northeast3`
- secrets: `vibecat-gemini-api-key`, `vibecat-gateway-auth-secret`
- Artifact Registry: `asia-northeast3-docker.pkg.dev/vibecat-489105/vibecat-images`

### CI Reality

- latest fully green CI run: GitHub Actions run `22932978716` on commit `ac5e4bf`
- latest `master` CI run: `22933714954` on merge commit `7356f8c`
- current `master` CI failure is environmental, not code-level: the self-hosted macOS runner failed at `Select Xcode 16+ and accept license`
- current Go and Docker jobs on `master` still pass

### Meaningful Remaining Work

After the 2026-03-11 issue audit, the truly meaningful open work is:

- `#57` deployment and operations evidence pack
- `#64` privacy controls UI
- `#90` Cloud Logging and Monitoring completion
- `#91` Cloud Trace completion
- `#117` companion intelligence end-to-end integration test
- `#120` graceful Gemini-unavailable fallback
- `#121` graceful ADK-timeout fallback

The rest of the open issue list is mostly consolidation, optional UX, or duplicate ops tracking.

## STRUCTURE

```text
vibeCat/
├── VibeCat/
│   ├── Package.swift
│   ├── Sources/Core/
│   ├── Sources/VibeCat/
│   └── Tests/VibeCatTests/
├── backend/
│   ├── realtime-gateway/
│   │   ├── internal/
│   │   ├── cmd/videotest/
│   │   └── cloudbuild.yaml
│   └── adk-orchestrator/
│       ├── internal/agents/
│       ├── internal/store/
│       └── cloudbuild.yaml
├── tests/e2e/
├── infra/
├── Assets/
├── docs/
└── voice_samples/
```

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Current repo/deploy status | `docs/CURRENT_STATUS_20260311.md` | Most accurate high-level snapshot |
| Current deployment evidence | `docs/evidence/DEPLOYMENT_EVIDENCE.md` | Cloud Run, CI, observability, remaining gaps |
| Submission proof template | `docs/deployment/PROOF_OF_GCP_DEPLOYMENT.md` | Final proof-oriented deployment checklist |
| Historical issue audit | `docs/ISSUE_TRIAGE_20260311.md` | PR #124 issue cleanup result |
| Final architecture | `docs/FINAL_ARCHITECTURE.md` | Current layered architecture and flows |
| PRD and task map | `docs/PRD/` | Original implementation plan and specs |
| Client core primitives | `VibeCat/Sources/Core/` | Models, localization, image/audio helpers |
| Client app runtime | `VibeCat/Sources/VibeCat/` | UI, capture, transport, playback, app wiring |
| Gateway implementation | `backend/realtime-gateway/` | Live API proxy, auth, ADK routing |
| Orchestrator implementation | `backend/adk-orchestrator/` | 9-agent graph and Firestore-backed state |
| Deployment scripts | `infra/` | Setup, deploy, teardown |

## IMPLEMENTATION STATUS

### Swift Client

- `Sources/Core/`: implemented
- `Sources/VibeCat/`: implemented app runtime with 22 top-level source files
- tests: 9 test files under `VibeCat/Tests/VibeCatTests/`
- notable recent additions: localization, assistant transcription assembly, live status fallback labels, audio-device monitoring

### Realtime Gateway

- Go module implemented under `backend/realtime-gateway/`
- includes auth, Live session management, ADK client, TTS client, language helpers, WebSocket registry/handler
- smoke and package tests exist

### ADK Orchestrator

- Go module implemented under `backend/adk-orchestrator/`
- includes vision, mediator, scheduler, engagement, memory, mood, celebration, search, tool-use, topic, prompts, Firestore store
- agent graph and package tests exist

### End-to-End

- `tests/e2e/` covers deployed health/auth/session/search-memory/barge-in paths
- full companion-intelligence session coverage is still incomplete and tracked by `#117`

## DEPLOYMENT CRITERIA

The practical deployment baseline for this repository is:

1. `realtime-gateway` and `adk-orchestrator` both deployed to Cloud Run in `asia-northeast3`
2. Gateway health endpoints reachable and returning `200`
3. Orchestrator authenticated health endpoint reachable with identity token
4. Firestore and both required secrets present in `vibecat-489105`
5. Artifact Registry repository present for backend images
6. GitHub CI green for Go and Docker; Swift CI depends on a licensed self-hosted Xcode runner
7. Logs and traces visible in GCP
8. Remaining production-readiness gaps tracked by `#57`, `#64`, `#90`, `#91`, `#117`, `#120`, `#121`

## CONVENTIONS

- Mandatory stack remains unchanged: GenAI SDK + ADK + Gemini Live API + VAD
- All model access stays server-side
- Client responsibilities are UI, capture, transport, and playback
- Core Swift module must remain UI-free
- Preserve exact asset directories and file names
- Production backend port remains `8080`
- GCP region remains `asia-northeast3`

## ANTI-PATTERNS

- direct client-to-Gemini calls
- client-side storage of Gemini API credentials
- moving prompt logic back into the client
- reintroducing client-side agent logic that now belongs in the orchestrator
- treating historical planning docs as proof of current implementation

## CHARACTERS

6 sprite characters are present under `Assets/Sprites/`:

- `cat`
- `derpy`
- `jinwoo`
- `kimjongun`
- `saja`
- `trump`

Each character includes `preset.json` and `soul.md`. Current verified counts:

- `preset.json`: 6
- `soul.md`: 6
- `TrayIcons_Clean` assets: 24
- `Music` assets: 2
- `voice_samples`: 13

## COMMANDS

```bash
# Repo state
git status --short --branch
git log -1 --oneline

# Swift client
cd VibeCat && swift build
cd VibeCat && swift test

# Go backends
cd backend/realtime-gateway && go test ./...
cd backend/adk-orchestrator && go test ./...

# E2E
cd tests/e2e && go test ./...

# Deploy / verify
./infra/deploy.sh
gcloud run services describe realtime-gateway --region asia-northeast3 --project vibecat-489105
gcloud run services describe adk-orchestrator --region asia-northeast3 --project vibecat-489105
```

## NOTES

- `docs/analysis/` contains point-in-time design and hardening notes; not all files there are current truth
- `.sisyphus/` contains historical plans and handoff notes; verify dates before trusting them
- Cloud Monitoring exporter is enabled in code, but no Monitoring dashboard is currently configured
- Cloud Build YAML exists for both backends, but no GCP Cloud Build trigger is currently configured
