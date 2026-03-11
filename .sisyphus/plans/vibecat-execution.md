# VibeCat Full Execution Plan

> HISTORICAL PLAN NOTE (2026-03-11): this is the original execution plan, not the current implementation ledger. Several completion gates and health endpoint examples are obsolete. Refer to `docs/CURRENT_STATUS_20260311.md` for the live baseline.

## TL;DR

> **Quick Summary**: Build VibeCat — a macOS desktop AI companion for the Gemini Live Agent Challenge 2026. 122 tasks organized into 4 macro-phases for a solo developer: Spike validation → Backend (Go) → Client (Swift) → Demo & Submission. Demo-driven development: features shown in the 4-minute demo get P0 priority.
> 
> **Deliverables**:
> - Working macOS desktop companion app (Swift 6 / SwiftUI)
> - Go backend: Realtime Gateway (GenAI SDK + Live API) + ADK Orchestrator (9-agent graph)
> - Deployed to GCP Cloud Run (asia-northeast3)
> - Demo video (≤4 min) on YouTube
> - Blog post with `#GeminiLiveAgentChallenge` tag (+0.6 bonus)
> - Devpost submission with all artifacts
> 
> **Estimated Effort**: XL (13 days, ~120 effective tasks)
> **Parallel Execution**: NO — solo developer, sequential waves
> **Critical Path**: Spike → Backend Bootstrap → Gateway → ADK Agents → Deploy → Client Core → Client UI → Integration → Demo

---

## Context

### Original Request
Build VibeCat for the Gemini Live Agent Challenge 2026. 122 implementation tasks (T-001~T-099 client, T-100~T-175 backend) organized into an execution plan for a solo developer. Mandatory stack: GenAI SDK + ADK + Gemini Live API + VAD.

### Interview Summary
**Key Discussions**:
- Backend: Go 1.24+ (GenAI SDK + ADK Go SDK) — non-negotiable
- Client: Swift 6 / SwiftUI / SPM (macOS)
- All 9 agents required in ADK Orchestrator
- TDD approach per TDD_VERIFICATION_PLAN.md, but demo priority overrides test purity
- Some scaffolding complete (Package.swift, Makefile, CI, infra scripts, 6 character personas)
- Sleeping sprite doesn't exist → user decided: "슬리핑 오버레이로 대체" (idle with dimmed overlay)
- Demo + blog post: ALWAYS REMIND at checkpoints

**Research Findings**:
- ADK Go SDK v0.5.0 confirmed available with agent graph support
- GenAI Go SDK supports Live API WebSocket streaming
- 13 days until deadline (March 16, 2026 5:00 PM PDT)
- ~9.2 tasks/day throughput needed — scope trimming critical

### Metis Review
**Identified Gaps** (addressed):
- **Existential risk**: GenAI Go SDK Live API + ADK Go SDK agent graph must be validated BEFORE any real development → Added Spike Phase (Wave 0)
- **Platform mismatch**: Package.swift says `.macOS(.v26)` but README says "macOS 15+" → [DECISION NEEDED: which target?]
- **13-day deadline**: 122 tasks in 13 days requires ruthless prioritization → Demo-scene-driven task priority applied
- **Screen recording entitlement**: May need additional entitlement for ScreenCaptureKit → Will verify during client implementation
- **English support**: Challenge requires English minimum → Will include in soul.md and prompt templates
- **No latency target**: Added `screen_capture → speech ≤ 3s` acceptance criterion
- **No submission assembly task**: Added as final wave
- **Context switching**: Solo dev → Group by technology (all Go, then all Swift, then integration)

---

## Work Objectives

### Core Objective
Deliver a working, deployed VibeCat application that demonstrates all 9 agents working together in a compelling 4-minute demo for the Gemini Live Agent Challenge 2026.

### Concrete Deliverables
- `backend/realtime-gateway/` — Go service, deployed to Cloud Run
- `backend/adk-orchestrator/` — Go service with 9-agent graph, deployed to Cloud Run
- `VibeCat/Sources/Core/` — Pure Swift core library
- `VibeCat/Sources/VibeCat/` — macOS SwiftUI app
- `VibeCat/Tests/VibeCatTests/` — Swift tests
- Demo video (≤4 min) uploaded to YouTube
- Blog post with `#GeminiLiveAgentChallenge`
- Devpost submission

### Definition of Done
- [ ] `swift build` succeeds in `VibeCat/`
- [ ] `go build ./...` succeeds in both `backend/realtime-gateway/` and `backend/adk-orchestrator/`
- [ ] Both Cloud Run services return 200 on `/healthz`
- [ ] Client connects to Gateway via WebSocket, sends audio, receives audio response
- [ ] Screen capture → VisionAgent → Mediator → speech decision works end-to-end
- [ ] Demo video recorded ≤4 min showing all 9 agents
- [ ] Blog post published
- [ ] Devpost submission completed

### Must Have
- GenAI SDK + ADK + Gemini Live API + VAD — all four stack components
- All 9 agents functional (VisionAgent, Mediator, AdaptiveScheduler, EngagementAgent, MemoryAgent, MoodDetector, CelebrationTrigger, SearchBuddy, VAD)
- Cloud Run deployment with observability
- English language support (minimum)
- Screen capture analysis → intelligent speech
- Voice conversation (bidirectional audio)
- Cross-session memory
- Demo video ≤4 min
- All code within contest period (Feb 16 - Mar 16, 2026)

### Must NOT Have (Guardrails)
- ❌ Client making direct Gemini API calls — ALL through backend
- ❌ API key stored on client — session tokens only, key in Secret Manager
- ❌ Elaborate error handling before happy path works
- ❌ Full TDD coverage blocking demo-critical features
- ❌ Features not shown in demo getting priority over demo features
- ❌ More than 2 hours on any single task without escalation
- ❌ Circle gesture detection (T-033) — use hotkey only for MVP
- ❌ Search caching in Firestore (T-168) — always search fresh
- ❌ Topic detection real-time NLP (T-151) — stub with keyword matching
- ❌ Elaborate fade transitions for background music (T-046) — binary on/off
- ❌ Launch-at-login feature — skip for MVP
- ❌ Over-abstraction or premature optimization
- ❌ Any mention of GeminiCat in committed files

---

## Verification Strategy (MANDATORY)

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed. No exceptions.

### Test Decision
- **Infrastructure exists**: NO (test framework not set up yet)
- **Automated tests**: YES (Tests-after) — TDD where practical, but demo priority wins
- **Swift framework**: XCTest (built-in with SPM)
- **Go framework**: `go test` (built-in)
- **Note**: Core library gets proper tests. Agent implementations get integration tests via curl/wscat. UI gets Playwright-equivalent manual QA scenarios.

### QA Policy
Every task MUST include agent-executed QA scenarios.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Go backend**: Use Bash (curl/wscat) — Send requests, assert status + response fields
- **Swift client**: Use Bash (swift build/test) + tmux for runtime verification
- **Deployment**: Use Bash (gcloud/curl) — Verify Cloud Run services
- **UI**: Use interactive_bash (tmux) — Run app, verify visual behavior
- **End-to-end latency**: `screen_capture → speech ≤ 3 seconds`

---

## Execution Strategy

### Solo Developer Sequential Waves (Demo-Driven)

> Solo developer = no true parallelism. Waves are sequential.
> Technology grouping minimizes context switching: Go first, then Swift, then integration.
> Demo scenes drive task priority: P0 = in demo, P1 = nice-to-have, P2 = skip for MVP.

```
Wave 0 — Spike Validation (Day 1):
├── Task 1: GenAI Go SDK Live API spike [deep]
├── Task 2: ADK Go SDK agent graph spike [deep]
└── Task 3: Fix scaffolding issues (Package.swift target, source dirs) [quick]

Wave 1 — Backend Bootstrap (Day 2):
├── Task 4: Backend repo structure + go.mod (T-100) [quick]
├── Task 5: Secret Manager setup (T-101) [quick]
├── Task 6: Firestore setup (T-102) [quick]
├── Task 7: Prompt management (T-137) [quick]
└── Task 8: Cloud Build pipeline (T-103) [quick]

Wave 2 — Realtime Gateway Core (Days 2-3):
├── Task 9: WebSocket server + health endpoints (T-110) [unspecified-high]
├── Task 10: Client authentication (T-111) [unspecified-high]
├── Task 11: GenAI SDK Live API session (T-112) [deep]
├── Task 12: Audio stream proxy (T-113) [deep]
├── Task 13: VAD configuration (T-114) [quick]
└── Task 14: Transcription forwarding (T-118) [quick]

Wave 3 — Gateway Advanced (Day 4):
├── Task 15: Session resumption proxy (T-115) [unspecified-high]
├── Task 16: Keepalive stack (T-116) [unspecified-high]
├── Task 17: Settings update handling (T-119) [quick]
├── Task 18: Affective dialog config (T-171) [quick]
├── Task 19: Proactive audio config (T-172) [quick]
└── Task 20: Graceful Gemini fallback (T-173) [deep]

Wave 4 — ADK Orchestrator Agents (Days 4-6):
├── Task 21: ADK project init + HTTP endpoint (T-130) [unspecified-high]
├── Task 22: VisionAgent (T-131) [deep]
├── Task 23: Frustration signal detection (T-154) [quick]
├── Task 24: Celebration detection patterns (T-159) [quick]
├── Task 25: Mediator (T-132) [deep]
├── Task 26: AdaptiveScheduler (T-133) [unspecified-high]
├── Task 27: EngagementAgent (T-134) [unspecified-high]
├── Task 28: Mood classification schema (T-153) [quick]
├── Task 29: MoodDetector (T-155) [deep]
├── Task 30: Mood-aware Mediator (T-156) [unspecified-high]
├── Task 31: Supportive message templates (T-158) [quick]
├── Task 32: CelebrationTrigger (T-160) [unspecified-high]
├── Task 33: Celebration in Mediator (T-162) [quick]
├── Task 34: Agent graph wiring (T-135) [deep]
├── Task 35: Google Search grounding (T-136, T-164) [unspecified-high]
├── Task 36: Search trigger detection (T-165) [quick]
├── Task 37: Auto-search from mood (T-166) [quick]
├── Task 38: Search result summarization (T-167) [unspecified-high]
└── Task 39: SearchBuddy → Gateway voice (T-169) [unspecified-high]

Wave 5 — Memory & Firestore (Day 6):
├── Task 40: Memory Firestore schema (T-147) [quick]
├── Task 41: End-of-session summary (T-148) [unspecified-high]
├── Task 42: Session-start memory retrieval (T-149) [unspecified-high]
├── Task 43: MemoryAgent in ADK graph (T-150) [deep]
├── Task 44: Memory context in Gateway session (T-152) [unspecified-high]
├── Task 45: Mood field in Firestore (T-157) [quick]
├── Task 46: Celebration cooldown (T-161) [quick]
└── Task 47: Gateway→ADK routing (T-117) [unspecified-high]

Wave 6 — Backend Deploy & Verify (Day 7):
├── Task 48: ADK fallback timeout (T-174) [quick]
├── Task 49: Containerize Gateway (T-140) [quick]
├── Task 50: Containerize ADK Orchestrator (T-141) [quick]
├── Task 51: Deploy to Cloud Run (T-142) [unspecified-high]
├── Task 52: Cloud Logging + Monitoring (T-143) [quick]
├── Task 53: Cloud Trace (T-144) [quick]
└── Task 54: Backend E2E test (T-145 partial — backend-only) [deep]

Wave 7 — Client Bootstrap (Day 7-8):
├── Task 55: Workspace + source directories (T-001 completion) [quick]
├── Task 56: App metadata + permissions (T-002) [quick]
├── Task 57: Runtime config model (T-003) [quick]
├── Task 58: Settings persistence (T-004) [quick]
└── Task 59: CI baseline fix (T-005 completion) [quick]

Wave 8 — Client Core Library (Day 8):
├── Task 60: Domain models (T-010) [quick]
├── Task 61: Prompt composition (T-011) [quick]
├── Task 62: Image processing (T-012) [quick]
├── Task 63: Image encoding (T-013) [quick]
├── Task 64: Visual change detection (T-014) [quick]
├── Task 65: Audio message parsing (T-015) [quick]
├── Task 66: PCM conversion (T-016) [quick]
├── Task 67: Keychain helper (T-017) [quick]
└── Task 68: Core test gate (T-018) [unspecified-high]

Wave 9 — Client Menu & Settings (Day 9):
├── Task 69: Status bar controller (T-020) [quick]
├── Task 70: Tray icon animation (T-021) [quick]
├── Task 71: Language/voice/chattiness menus (T-022) [quick]
├── Task 72: Model/reasoning controls (T-023) [quick]
├── Task 73: Capture/appearance controls (T-024) [quick]
├── Task 74: Advanced/music controls (T-025) [quick]
├── Task 75: API key onboarding (T-026) [unspecified-high]
└── Task 76: Reconnect/pause/mute/quit (T-027) [quick]

Wave 10 — Client Capture & Transport (Day 9-10):
├── Task 77: Capture around cursor (T-030) [unspecified-high]
├── Task 78: Changed-only capture path (T-031) [quick]
├── Task 79: Full window capture (T-032) [quick]
├── Task 80: WebSocket lifecycle — connect to Gateway (T-040) [deep]
├── Task 81: Setup payload policy (T-041) [quick]
├── Task 82: Server message handling (T-042) [unspecified-high]
├── Task 83: Heartbeat + zombie detection (T-043) [quick]
├── Task 84: Audio playback engine (T-044) [unspecified-high]
├── Task 85: Speech wrapper + TTS fallback (T-045) [unspecified-high]
└── Task 86: Background music (T-046) [quick]

Wave 11 — Client Agents & Orchestrator (Day 10):
├── Task 87: VisionAgent client-side (T-050) [unspecified-high]
├── Task 88: Mediator client-side (T-051) [unspecified-high]
├── Task 89: Adaptive scheduler client-side (T-052) [quick]
├── Task 90: Engagement agent client-side (T-053) [quick]
├── Task 91: Cat view model (T-060) [unspecified-high]
├── Task 92: Sprite animator (T-061) [unspecified-high]
├── Task 93: Bubble + emotion indicators (T-062) [quick]
├── Task 94: Screen analyzer loop (T-063) [deep]
├── Task 95: High-significance branch (T-064) [quick]
├── Task 96: Live transcription → bubble (T-065) [quick]
└── Task 97: REST fallback speech (T-066) [quick]

Wave 12 — Client Voice/Chat & Integration (Day 11):
├── Task 98: Global hotkey (T-070) [quick]
├── Task 99: Speech recognizer (T-071) [unspecified-high]
├── Task 100: Chat panel container (T-072) [quick]
├── Task 101: Chat view + messages (T-073) [quick]
├── Task 102: Interaction mode wiring (T-074) [quick]
├── Task 103: Main entrypoint + duplicate guard (T-080) [quick]
├── Task 104: App delegate wiring (T-081) [deep]
├── Task 105: Floating panel behavior (T-082) [quick]
├── Task 106: Status bar callback wiring (T-083) [quick]
├── Task 107: Startup mode (T-084) [quick]
└── Task 108: Companion message types (T-097) [unspecified-high]

Wave 13 — Grand Prize Features & Polish (Day 11-12):
├── Task 109: Decision Overlay HUD (T-094) [unspecified-high]
├── Task 110: Sprite screen pointing (T-095) [unspecified-high]
├── Task 111: Session farewell + sleep overlay (T-096) [quick]
├── Task 112: Graceful reconnection UX (T-098) [unspecified-high]
├── Task 113: Privacy controls UI (T-099) [quick]
├── Task 114: Celebration client protocol (T-163) [quick]
└── Task 115: Search result client protocol (T-169 client-side) [quick]

Wave 14 — End-to-End Validation (Day 12):
├── Task 116: Asset validation (T-090) [quick]
├── Task 117: Full operation scenarios (T-091) [deep]
├── Task 118: Backend E2E integration (T-145 full) [deep]
├── Task 119: Companion intelligence E2E (T-170) [deep]
└── Task 120: Topic detection stub (T-151) [quick]

Wave 15 — Submission Artifacts (Days 12-13):
├── Task 121: Deployment evidence pack (T-092, T-146) [unspecified-high]
├── Task 122: Demo video recording (≤4 min) [writing]
├── Task 123: Blog post writing + publish [writing]
├── Task 124: Devpost submission assembly [quick]
└── Task 125: Final handoff gate (T-093) [deep]

Wave FINAL — Independent Review (after ALL):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real manual QA (unspecified-high)
└── Task F4: Scope fidelity check (deep)

Critical Path: Task 1-2 (spike) → Task 11 (Live API) → Task 22 (VisionAgent) → Task 34 (agent graph) → Task 51 (deploy) → Task 80 (client WS) → Task 94 (screen analyzer) → Task 104 (app delegate) → Task 117 (full ops) → Task 122 (demo)
```

### Dependency Matrix

| Task | Depends On | Blocks | Wave |
|------|-----------|--------|------|
| 1 (GenAI spike) | — | 11, 12 | 0 |
| 2 (ADK spike) | — | 21-39 | 0 |
| 3 (Fix scaffolding) | — | 55-68 | 0 |
| 4 (T-100) | — | 5-8, 9, 21 | 1 |
| 9 (T-110) | 4 | 10, 11, 47 | 2 |
| 11 (T-112) | 1, 9 | 12-20, 44 | 2 |
| 21 (T-130) | 2, 4 | 22-39, 47 | 4 |
| 22 (T-131) | 21 | 23, 24, 25 | 4 |
| 25 (T-132) | 22 | 26, 30, 33, 34 | 4 |
| 34 (T-135) | 22-27, 32 | 35, 48 | 4 |
| 47 (T-117) | 9, 21 | 48, 54 | 5 |
| 49 (T-140) | 17 | 51 | 6 |
| 50 (T-141) | 34 | 51 | 6 |
| 51 (T-142) | 49, 50 | 52, 53, 54 | 6 |
| 55 (T-001) | 3 | 56-68 | 7 |
| 80 (T-040) | 63, 57 | 81-86 | 10 |
| 94 (T-063) | 78, 87, 88 | 95-97 | 11 |
| 104 (T-081) | 94, 102 | 105-108 | 12 |
| 117 (T-091) | 107, 116 | 118, 121 | 14 |
| 122 (Demo) | 117, 118 | 124 | 15 |

### Agent Dispatch Summary

- **Wave 0**: 3 tasks — T1-T2 → `deep`, T3 → `quick`
- **Wave 1**: 5 tasks — all → `quick`
- **Wave 2**: 6 tasks — T9-T10 → `unspecified-high`, T11-T12 → `deep`, T13-T14 → `quick`
- **Wave 3**: 6 tasks — T15-T16 → `unspecified-high`, T17-T19 → `quick`, T20 → `deep`
- **Wave 4**: 19 tasks — mixed `deep`/`unspecified-high`/`quick`
- **Wave 5**: 8 tasks — T43 → `deep`, rest mixed
- **Wave 6**: 7 tasks — T54 → `deep`, rest `quick`/`unspecified-high`
- **Wave 7**: 5 tasks — all `quick`
- **Wave 8**: 9 tasks — T68 → `unspecified-high`, rest `quick`
- **Wave 9**: 8 tasks — T75 → `unspecified-high`, rest `quick`
- **Wave 10**: 10 tasks — T80 → `deep`, rest mixed
- **Wave 11**: 11 tasks — T94 → `deep`, rest mixed
- **Wave 12**: 11 tasks — T104 → `deep`, rest mixed
- **Wave 13**: 7 tasks — mixed
- **Wave 14**: 5 tasks — T117-T119 → `deep`, rest `quick`
- **Wave 15**: 5 tasks — T122-T123 → `writing`, rest mixed
- **Wave FINAL**: 4 tasks — T_F1 → `oracle`, rest mixed

---

## TODOs

> Implementation + QA = ONE Task. Never separate.
> EVERY task MUST have: Recommended Agent Profile + QA Scenarios.
> **A task WITHOUT QA Scenarios is INCOMPLETE. No exceptions.**
> **Maximum 2 hours per task. If exceeded → stub and move on.**

### Wave 0 — Spike Validation (Day 1)

- [x] 1. Validate GenAI Go SDK Live API WebSocket streaming

  **What to do**:
  - Create `spike/genai-live/main.go` with minimal Go program
  - Initialize GenAI client with API key from environment
  - Create a Live API session (`client.Live.Connect()`)
  - Send a text message, receive audio response
  - Verify WebSocket streaming works bidirectionally
  - Test `proactiveAudio`, `affectiveDialog`, `outputAudioTranscription` config options
  - Document which features are available/unavailable
  - Delete spike directory after validation (do NOT commit)

  **Must NOT do**:
  - Build production code — this is ONLY validation
  - Spend more than 2 hours — if blocked, document the blocker and proceed

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: [`playwright`]
    - `playwright`: For checking GenAI Go SDK docs if needed via browser
  - **Skills Evaluated but Omitted**:
    - `git-master`: Not needed — spike code is throwaway

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 2, 3)
  - **Parallel Group**: Wave 0
  - **Blocks**: Tasks 11, 12 (Gateway Live API implementation)
  - **Blocked By**: None

  **References**:
  - `docs/reference/gemini/live-api/01_get_started_live_api.md` — Live API setup and session creation
  - `docs/reference/gemini/live-api/02_live_capabilities_guide.md` — proactiveAudio, affectiveDialog, VAD config
  - `docs/reference/gemini/genai-sdk/01_sdk_overview.md` — Go SDK client initialization
  - `/tmp/go-genai/live.go` — Go SDK Live API source (already cached)
  - Context7: `google.golang.org/genai` — Query for Live API streaming examples

  **Acceptance Criteria**:
  - [ ] Go program compiles: `go build ./spike/genai-live/`
  - [ ] Live session connects successfully (no error on `client.Live.Connect()`)
  - [ ] Audio response received from Gemini
  - [ ] Feature availability documented in spike output

  **QA Scenarios**:
  ```
  Scenario: Live API connection + audio round-trip
    Tool: Bash
    Preconditions: GEMINI_API_KEY set in environment
    Steps:
      1. cd spike/genai-live && go run main.go
      2. Check stdout for "Session connected" or similar success message
      3. Check stdout for "Audio received: N bytes" indicating response
    Expected Result: Program exits 0, logs show successful connection and audio receipt
    Failure Indicators: Connection error, timeout, "unsupported" error for Live API
    Evidence: .sisyphus/evidence/task-1-genai-live-spike.txt

  Scenario: Feature availability check
    Tool: Bash
    Preconditions: Same as above
    Steps:
      1. Check spike output for proactiveAudio status
      2. Check spike output for affectiveDialog status
      3. Check spike output for outputAudioTranscription status
    Expected Result: Each feature reports AVAILABLE or UNAVAILABLE
    Evidence: .sisyphus/evidence/task-1-feature-availability.txt
  ```

  **Commit**: NO (spike — delete after validation)

- [x] 2. Validate ADK Go SDK agent graph with multiple agents

  **What to do**:
  - Create `spike/adk-graph/main.go` with minimal Go program
  - Import `google.golang.org/adk`
  - Define 2 simple agents (e.g., "analyzer" and "responder")
  - Wire them into an agent graph
  - Execute the graph with a test input
  - Verify agents can communicate and produce structured output
  - Document ADK Go SDK API surface and patterns
  - Delete spike directory after validation

  **Must NOT do**:
  - Build all 9 agents — just 2 to prove the pattern works
  - Spend more than 2 hours

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []
  - **Skills Evaluated but Omitted**:
    - `playwright`: ADK docs likely available via context7

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 1, 3)
  - **Parallel Group**: Wave 0
  - **Blocks**: Tasks 21-39 (all ADK Orchestrator tasks)
  - **Blocked By**: None

  **References**:
  - ADK Go SDK: `google.golang.org/adk` (v0.5.0)
  - ADK samples: `github.com/google/adk-samples/tree/main/go` — 63 Go files with real examples
  - `docs/reference/adk/components/01_agents.md` — Agent definition patterns
  - `docs/reference/adk/streaming/01_adk_streaming_overview.md` — Streaming/graph patterns
  - Context7: `google.golang.org/adk` — Query for agent graph examples

  **Acceptance Criteria**:
  - [ ] Go program compiles: `go build ./spike/adk-graph/`
  - [ ] Agent graph executes with 2 agents
  - [ ] Agents produce structured output
  - [ ] API patterns documented for production use

  **QA Scenarios**:
  ```
  Scenario: ADK agent graph execution
    Tool: Bash
    Preconditions: GEMINI_API_KEY or GOOGLE_API_KEY set
    Steps:
      1. cd spike/adk-graph && go run main.go
      2. Check stdout for agent execution logs
      3. Verify both agents produced output
    Expected Result: Program exits 0, both agents executed in graph order
    Failure Indicators: Import errors, nil pointer, "agent not found"
    Evidence: .sisyphus/evidence/task-2-adk-graph-spike.txt

  Scenario: Structured output from agents
    Tool: Bash
    Steps:
      1. Check spike output for JSON-structured agent responses
    Expected Result: Agents return typed/structured data, not raw strings
    Evidence: .sisyphus/evidence/task-2-adk-structured-output.txt
  ```

  **Commit**: NO (spike — delete after validation)

- [x] 3. Fix scaffolding issues and create source directories

  **What to do**:
  - Create missing directories: `VibeCat/Sources/Core/`, `VibeCat/Sources/VibeCat/`, `VibeCat/Tests/VibeCatTests/`
  - Add placeholder `.swift` files so `swift build` succeeds
  - Verify Package.swift platform target — resolve `.macOS(.v26)` vs `.macOS(.v15)` [DECISION NEEDED]
  - Create `backend/` directory structure stub
  - Verify `swift build` succeeds with placeholder targets
  - Verify Makefile `build` target works

  **Must NOT do**:
  - Write production code — just placeholders for compilation
  - Change CI workflow yet (that's T-005)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 1, 2)
  - **Parallel Group**: Wave 0
  - **Blocks**: Tasks 55-68 (Client waves)
  - **Blocked By**: None

  **References**:
  - `VibeCat/Package.swift` — Current SPM manifest (line 1-25)
  - `docs/PRD/DETAILS/IMPLEMENTATION_EXECUTION_PLAN.md:7-18` — Target repo skeleton
  - `Makefile` — Build targets that need to work

  **Acceptance Criteria**:
  - [ ] `VibeCat/Sources/Core/` exists with placeholder
  - [ ] `VibeCat/Sources/VibeCat/` exists with placeholder
  - [ ] `VibeCat/Tests/VibeCatTests/` exists with placeholder
  - [ ] `cd VibeCat && swift build` succeeds
  - [ ] `make build` succeeds

  **QA Scenarios**:
  ```
  Scenario: Swift build succeeds
    Tool: Bash
    Steps:
      1. cd VibeCat && swift build 2>&1
      2. Check exit code is 0
      3. Check output contains "Build complete!"
    Expected Result: Build succeeds with zero errors
    Failure Indicators: "error:", non-zero exit code
    Evidence: .sisyphus/evidence/task-3-swift-build.txt

  Scenario: Directory structure correct
    Tool: Bash
    Steps:
      1. ls -la VibeCat/Sources/Core/
      2. ls -la VibeCat/Sources/VibeCat/
      3. ls -la VibeCat/Tests/VibeCatTests/
    Expected Result: All directories exist with at least one .swift file
    Evidence: .sisyphus/evidence/task-3-directory-structure.txt
  ```

  **Commit**: YES
  - Message: `chore(client): create source directory skeleton and placeholders`
  - Files: `VibeCat/Sources/Core/`, `VibeCat/Sources/VibeCat/`, `VibeCat/Tests/VibeCatTests/`
  - Pre-commit: `cd VibeCat && swift build`

### Wave 1 — Backend Bootstrap (Day 2)

- [x] 4. Initialize backend repository structure (T-100)

  **What to do**:
  - Create `backend/realtime-gateway/` with `go.mod`, `main.go` (placeholder HTTP server on :8080)
  - Create `backend/adk-orchestrator/` with `go.mod`, `main.go` (placeholder HTTP server on :8080)
  - Add Go module dependencies per BACKEND_IMPLEMENTATION_TASKS.md
  - `go.mod` for gateway: `google.golang.org/genai`, `github.com/gorilla/websocket`, `cloud.google.com/go/secretmanager`, `cloud.google.com/go/firestore`
  - `go.mod` for orchestrator: `google.golang.org/adk`, `google.golang.org/genai`, `cloud.google.com/go/firestore`
  - Add `/healthz` and `/readyz` endpoints to both services
  - Verify both build and run

  **Must NOT do**:
  - Implement actual business logic
  - Add authentication yet

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO (first backend task)
  - **Blocks**: Tasks 5-8, 9, 21
  - **Blocked By**: None (can start immediately)

  **References**:
  - `docs/PRD/DETAILS/BACKEND_IMPLEMENTATION_TASKS.md:12-28` — Go module dependencies
  - `docs/PRD/DETAILS/BACKEND_ARCHITECTURE.md` — Service structure
  - `Makefile` — Backend build targets (backend-build, backend-test)

  **Acceptance Criteria**:
  - [ ] `cd backend/realtime-gateway && go build ./...` succeeds
  - [ ] `cd backend/adk-orchestrator && go build ./...` succeeds
  - [ ] `curl http://localhost:8080/healthz` returns `{"status":"ok"}`
  - [ ] Both go.mod files have correct dependencies

  **QA Scenarios**:
  ```
  Scenario: Gateway builds and serves health check
    Tool: Bash
    Steps:
      1. cd backend/realtime-gateway && go build -o gateway .
      2. ./gateway & sleep 2
      3. curl -s http://localhost:8080/healthz | jq '.status'
      4. kill %1
    Expected Result: Build succeeds, health check returns "ok"
    Evidence: .sisyphus/evidence/task-4-gateway-health.txt

  Scenario: Orchestrator builds and serves health check
    Tool: Bash
    Steps:
      1. cd backend/adk-orchestrator && go build -o orchestrator .
      2. ./orchestrator & sleep 2
      3. curl -s http://localhost:8080/healthz | jq '.status'
      4. kill %1
    Expected Result: Build succeeds, health check returns "ok"
    Evidence: .sisyphus/evidence/task-4-orchestrator-health.txt
  ```

  **Commit**: YES
  - Message: `feat(backend): initialize gateway and orchestrator Go modules with health endpoints`
  - Files: `backend/realtime-gateway/`, `backend/adk-orchestrator/`
  - Pre-commit: `cd backend/realtime-gateway && go build ./... && cd ../adk-orchestrator && go build ./...`

- [x] 5. Configure Secret Manager (T-101)

  **What to do**:
  - Use `gcloud` CLI to create secret `vibecat-gemini-api-key` (if not exists)
  - Create secret `vibecat-gateway-auth-secret` for client token signing
  - Set IAM bindings for Cloud Run service accounts
  - Write Go helper function in `backend/realtime-gateway/internal/secrets/secrets.go` to load secrets at runtime
  - Test secret access locally with `gcloud auth application-default login`

  **Must NOT do**:
  - Store actual API key in source code
  - Create Terraform configs

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **References**:
  - `docs/reference/gcp/secret-manager/01_secret_manager_overview.md` — Secret Manager API
  - `infra/setup.sh` — Existing GCP setup script (already creates secrets)
  - `docs/PRD/DETAILS/BACKEND_ARCHITECTURE.md` — Security architecture

  **Acceptance Criteria**:
  - [ ] `gcloud secrets describe vibecat-gemini-api-key --project=vibecat-489105` returns metadata
  - [ ] Go helper compiles: `cd backend/realtime-gateway && go build ./...`
  - [ ] Secret is accessible from Go code (unit test with mock or integration test)

  **QA Scenarios**:
  ```
  Scenario: Secret exists in GCP
    Tool: Bash
    Steps:
      1. gcloud secrets describe vibecat-gemini-api-key --project=vibecat-489105 --format=json | jq '.name'
    Expected Result: Secret name contains "vibecat-gemini-api-key"
    Evidence: .sisyphus/evidence/task-5-secret-exists.txt

  Scenario: Go helper compiles
    Tool: Bash
    Steps:
      1. cd backend/realtime-gateway && go build ./internal/secrets/...
    Expected Result: Build succeeds with zero errors
    Evidence: .sisyphus/evidence/task-5-secrets-build.txt
  ```

  **Commit**: YES
  - Message: `feat(backend/gateway): add Secret Manager integration for API key loading`
  - Files: `backend/realtime-gateway/internal/secrets/`
  - Pre-commit: `cd backend/realtime-gateway && go build ./...`

- [x] 6. Configure Firestore (T-102)

  **What to do**:
  - Verify Firestore DB exists in asia-northeast3 (created by infra/setup.sh)
  - Define collection structure: `sessions`, `metrics`, `history`, `users/{userId}/memory`
  - Write Go Firestore client helper in `backend/adk-orchestrator/internal/store/firestore.go`
  - Define Go structs for Session, Metric, MemoryEntry documents
  - Test read/write with emulator or live Firestore

  **Must NOT do**:
  - Set up Firestore emulator for development (defer — use live for now)
  - Create elaborate security rules (defer)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **References**:
  - `docs/reference/gcp/firestore/01_firestore_overview.md` — Firestore Go SDK
  - `docs/PRD/DETAILS/BACKEND_ARCHITECTURE.md:200-280` — Firestore schema definition
  - `infra/setup.sh` — Existing Firestore setup commands

  **Acceptance Criteria**:
  - [ ] Firestore client helper compiles
  - [ ] Go structs match the schema in BACKEND_ARCHITECTURE.md
  - [ ] Write/read cycle succeeds on at least one collection

  **QA Scenarios**:
  ```
  Scenario: Firestore Go client compiles
    Tool: Bash
    Steps:
      1. cd backend/adk-orchestrator && go build ./internal/store/...
    Expected Result: Build succeeds
    Evidence: .sisyphus/evidence/task-6-firestore-build.txt
  ```

  **Commit**: YES
  - Message: `feat(backend/orchestrator): add Firestore client and document schemas`
  - Files: `backend/adk-orchestrator/internal/store/`
  - Pre-commit: `cd backend/adk-orchestrator && go build ./...`

- [x] 7. Implement prompt management (T-137)

  **What to do**:
  - Create `backend/adk-orchestrator/internal/prompts/` package
  - Define prompt templates for: VisionAgent system prompt, cat personality prompt (load from soul.md), engagement directive, fallback personality
  - Support language-aware output (Korean primary, English fallback)
  - Load soul.md content from embedded filesystem or config
  - English support required for challenge compliance

  **Must NOT do**:
  - Spend time tuning prompts — use soul.md content directly
  - Over-engineer template system

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **References**:
  - `Assets/Sprites/cat/soul.md` — Cat personality prompt
  - `Assets/Sprites/*/preset.json` — Voice + persona config per character
  - `docs/PRD/DETAILS/BACKEND_ARCHITECTURE.md` — Agent prompt requirements

  **Acceptance Criteria**:
  - [ ] Prompt package compiles
  - [ ] VisionAgent prompt includes required analysis directives
  - [ ] Soul.md content loadable at runtime

  **QA Scenarios**:
  ```
  Scenario: Prompt package builds
    Tool: Bash
    Steps:
      1. cd backend/adk-orchestrator && go build ./internal/prompts/...
    Expected Result: Build succeeds
    Evidence: .sisyphus/evidence/task-7-prompts-build.txt
  ```

  **Commit**: YES
  - Message: `feat(backend/orchestrator): add prompt templates with character persona support`
  - Files: `backend/adk-orchestrator/internal/prompts/`
  - Pre-commit: `cd backend/adk-orchestrator && go build ./...`

- [x] 8. Set up Cloud Build pipeline (T-103)

  **What to do**:
  - Create `backend/realtime-gateway/cloudbuild.yaml` — build Go binary, create Docker image, push to Artifact Registry
  - Create `backend/adk-orchestrator/cloudbuild.yaml` — same pattern
  - Verify Artifact Registry repo `vibecat-images` exists (from infra/setup.sh)
  - Test Cloud Build manually: `gcloud builds submit`

  **Must NOT do**:
  - Set up automatic triggers yet (defer — manual builds for now)
  - Create elaborate multi-env pipelines

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **References**:
  - `docs/reference/gcp/cloud-build/01_cloud_build_overview.md` — Cloud Build config
  - `docs/PRD/DETAILS/CLOUDBUILD_SPEC.md` — Cloud Build YAML specs
  - `infra/deploy.sh` — Existing deploy script (references Cloud Build)

  **Acceptance Criteria**:
  - [ ] `cloudbuild.yaml` exists for both services
  - [ ] YAML is valid: `python -c "import yaml; yaml.safe_load(open('cloudbuild.yaml'))"`

  **QA Scenarios**:
  ```
  Scenario: Cloud Build YAML is valid
    Tool: Bash
    Steps:
      1. python3 -c "import yaml; yaml.safe_load(open('backend/realtime-gateway/cloudbuild.yaml'))"
      2. python3 -c "import yaml; yaml.safe_load(open('backend/adk-orchestrator/cloudbuild.yaml'))"
    Expected Result: Both commands exit 0 (valid YAML)
    Evidence: .sisyphus/evidence/task-8-cloudbuild-yaml.txt
  ```

  **Commit**: YES
  - Message: `feat(infra): add Cloud Build configs for gateway and orchestrator`
  - Files: `backend/realtime-gateway/cloudbuild.yaml`, `backend/adk-orchestrator/cloudbuild.yaml`

### Wave 2 — Realtime Gateway Core (Days 2-3)

- [x] 9. Implement WebSocket server + health endpoints (T-110)

  **What to do**:
  - Implement WebSocket upgrade handler at `/ws/live` using `gorilla/websocket`
  - Handle open/message/close/error events
  - Maintain client connection registry
  - `/healthz` and `/readyz` already exist from Task 4 — enhance with connection count
  - Accept PCM audio frames from client, echo for now

  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocks**: Tasks 10, 11, 47 | **Blocked By**: Task 4

  **References**:
  - `docs/PRD/DETAILS/CLIENT_BACKEND_PROTOCOL.md` — WebSocket message types and format
  - `docs/PRD/DETAILS/BACKEND_ARCHITECTURE.md:100-150` — Gateway connection handling

  **QA Scenarios**:
  ```
  Scenario: WebSocket connection + echo
    Tool: Bash
    Steps:
      1. Start gateway server
      2. wscat -c ws://localhost:8080/ws/live
      3. Send JSON message: {"type":"ping"}
      4. Verify response received
    Expected Result: WebSocket connects, messages flow bidirectionally
    Evidence: .sisyphus/evidence/task-9-ws-echo.txt
  ```
  **Commit**: YES — `feat(backend/gateway): implement WebSocket server with connection management`

- [x] 10. Implement client authentication (T-111)

  **What to do**:
  - `POST /api/v1/auth/register` — validate API key against Secret Manager, return JWT token
  - `POST /api/v1/auth/refresh` — refresh expired tokens
  - Token validation middleware on WebSocket upgrade
  - For MVP: simple JWT with HMAC signing using gateway auth secret

  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocks**: None directly | **Blocked By**: Tasks 5, 9

  **References**:
  - `docs/PRD/DETAILS/CLIENT_BACKEND_PROTOCOL.md:50-80` — Auth endpoints spec
  - `docs/PRD/DETAILS/BACKEND_ARCHITECTURE.md` — Security model

  **QA Scenarios**:
  ```
  Scenario: Auth register + token validation
    Tool: Bash
    Steps:
      1. curl -X POST http://localhost:8080/api/v1/auth/register -d '{"apiKey":"test-key"}' -H 'Content-Type: application/json'
      2. Extract token from response
      3. wscat -c "ws://localhost:8080/ws/live" -H "Authorization: Bearer {token}"
    Expected Result: Valid key → token returned. Token → WebSocket connects. Invalid key → 401.
    Evidence: .sisyphus/evidence/task-10-auth.txt
  ```
  **Commit**: YES — `feat(backend/gateway): add client authentication with JWT tokens`

- [x] 11. Implement GenAI SDK Live API session (T-112)

  **What to do**:
  - Initialize GenAI Go client with API key from Secret Manager
  - On WebSocket connect: create Gemini Live session via `client.Live.Connect()`
  - Configure session: voice, language, tools, VAD (`automaticActivityDetection`), resumption, compression, proactiveAudio, affectiveDialog, outputAudioTranscription
  - Use spike findings from Task 1 for available features
  - Handle setup-complete event from Gemini

  **Recommended Agent Profile**: `deep` | Skills: []
  **Blocks**: Tasks 12-20, 44 | **Blocked By**: Tasks 1 (spike), 9

  **References**:
  - `docs/reference/gemini/live-api/01_get_started_live_api.md` — Session creation
  - `docs/reference/gemini/live-api/02_live_capabilities_guide.md` — All config options
  - `/tmp/go-genai/live.go` — Go SDK Live API source
  - Task 1 spike output — Feature availability results

  **QA Scenarios**:
  ```
  Scenario: Live API session creation
    Tool: Bash
    Steps:
      1. Start gateway with GEMINI_API_KEY
      2. Connect WebSocket client
      3. Check gateway logs for "Gemini session established" or "setup-complete"
    Expected Result: Gemini Live session created per client connection
    Evidence: .sisyphus/evidence/task-11-live-session.txt
  ```
  **Commit**: YES — `feat(backend/gateway): integrate GenAI SDK Live API session management`

- [x] 12. Implement audio stream proxy (T-113)

  **What to do**:
  - Forward client PCM audio (16kHz 16-bit mono) → Gemini via `session.SendRealtimeInput()`
  - Forward Gemini audio response (24kHz) → client WebSocket
  - Handle interruptions (barge-in)
  - Handle turn completion events
  - Maintain audio buffer state

  **Recommended Agent Profile**: `deep` | Skills: []
  **Blocks**: Tasks 18, 41, 48 | **Blocked By**: Task 11

  **References**:
  - `docs/reference/gemini/live-api/02_live_capabilities_guide.md` — Audio streaming
  - `docs/PRD/DETAILS/CLIENT_BACKEND_PROTOCOL.md` — Audio message format
  - `/tmp/go-genai/live.go` — `SendRealtimeInput`, `Receive` methods

  **QA Scenarios**:
  ```
  Scenario: Audio round-trip through Gateway
    Tool: Bash
    Steps:
      1. Start gateway with GEMINI_API_KEY
      2. Connect WebSocket, send auth
      3. Send PCM audio frame (16kHz 16-bit, "Hello" spoken audio file)
      4. Wait for audio response bytes
    Expected Result: Gemini responds with audio. Latency < 3 seconds from send to first audio byte.
    Evidence: .sisyphus/evidence/task-12-audio-proxy.txt
  ```
  **Commit**: YES — `feat(backend/gateway): implement bidirectional audio stream proxy`

- [x] 13. Configure VAD (T-114)

  **What to do**:
  - Set `automaticActivityDetection` config: disabled=false, startSensitivity=LOW, endSensitivity=LOW, prefixPadding=20ms, silenceDuration=100ms
  - Apply in Live API session setup
  - Verify speech start/end events are received

  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocks**: None | **Blocked By**: Task 11

  **References**:
  - `docs/PRD/DETAILS/IMPLEMENTATION_REQUIREMENTS.md` — VAD settings
  - `docs/reference/gemini/live-api/02_live_capabilities_guide.md` — VAD configuration

  **QA Scenarios**:
  ```
  Scenario: VAD config applied
    Tool: Bash
    Steps:
      1. Check gateway logs for VAD configuration in session setup
    Expected Result: Logs show VAD config with specified sensitivity values
    Evidence: .sisyphus/evidence/task-13-vad-config.txt
  ```
  **Commit**: YES (groups with Task 11) — `feat(backend/gateway): configure VAD for speech detection`

- [x] 14. Implement transcription forwarding (T-118)

  **What to do**:
  - Parse transcription events from Gemini Live response
  - Forward to client with finished flag and turn completion
  - Sentence-level granularity for UI display

  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocks**: None | **Blocked By**: Task 12

  **References**:
  - `docs/reference/gemini/live-api/02_live_capabilities_guide.md` — Transcription events
  - `docs/PRD/DETAILS/CLIENT_BACKEND_PROTOCOL.md` — transcription message type

  **QA Scenarios**:
  ```
  Scenario: Transcription forwarded to client
    Tool: Bash
    Steps:
      1. Send audio to gateway, monitor WebSocket for transcription messages
    Expected Result: Client receives transcription with text and finished flag
    Evidence: .sisyphus/evidence/task-14-transcription.txt
  ```
  **Commit**: YES — `feat(backend/gateway): forward Gemini transcription to client WebSocket`

### Wave 3 — Gateway Advanced (Day 4)

- [x] 15. Implement session resumption proxy (T-115)

  **What to do**:
  - Store resumption handles from Gemini session
  - On client reconnect, use resumption handle to resume conversation
  - Handle GoAway messages gracefully

  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocks**: None | **Blocked By**: Task 11

  **References**:
  - `docs/reference/gemini/live-api/04_live_session_management.md` — Session resumption, GoAway

  **QA Scenarios**:
  ```
  Scenario: Session resume after disconnect
    Tool: Bash
    Steps:
      1. Connect, have conversation, disconnect
      2. Reconnect with resumption handle
      3. Verify conversation context maintained
    Expected Result: Resumed session remembers prior conversation
    Evidence: .sisyphus/evidence/task-15-session-resume.txt
  ```
  **Commit**: YES — `feat(backend/gateway): implement session resumption with handle storage`

- [x] 16. Implement keepalive stack (T-116)

  **What to do**:
  - Protocol ping/pong (15s interval)
  - Heartbeat to Gemini (30s)
  - Respond to client pings
  - Zombie detection (45s timeout → force disconnect)

  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocks**: None | **Blocked By**: Task 11

  **References**:
  - `docs/PRD/DETAILS/BACKEND_ARCHITECTURE.md` — Keepalive stack spec

  **QA Scenarios**:
  ```
  Scenario: Zombie connection cleanup
    Tool: Bash
    Steps:
      1. Connect WebSocket, stop sending any messages
      2. Wait 50 seconds
      3. Check connection is closed by server
    Expected Result: Server closes stale connection after ~45s timeout
    Evidence: .sisyphus/evidence/task-16-zombie-detection.txt
  ```
  **Commit**: YES — `feat(backend/gateway): add keepalive, ping/pong, and zombie detection`

- [x] 17. Implement settings update handling (T-119)

  **What to do**:
  - Handle `settingsUpdate` WebSocket message from client
  - Voice/model/language changes → reconnect Gemini session with new config
  - Chattiness/capture changes → pass-through to ADK or apply locally

  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocks**: Task 49 | **Blocked By**: Task 11

  **References**:
  - `docs/PRD/DETAILS/CLIENT_BACKEND_PROTOCOL.md` — settingsUpdate message spec

  **Commit**: YES — `feat(backend/gateway): handle runtime settings updates from client`

- [x] 18. Enable affective dialog (T-171)

  **What to do**:
  - Add `affectiveDialog: {enabled: true}` to Gemini Live API setup config
  - Conditional on spike findings from Task 1

  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 11, Task 1 (availability)

  **Commit**: YES (groups with Task 13) — `feat(backend/gateway): enable affective dialog for emotional voice`

- [x] 19. Enable proactive audio (T-172)

  **What to do**:
  - Add `proactiveAudio: {enabled: true}` to Gemini Live API setup config
  - Enables agent to speak first without user input

  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 11, Task 1 (availability)

  **Commit**: YES (groups with Task 18) — `feat(backend/gateway): enable proactive audio for agent-initiated speech`

- [x] 20. Implement graceful Gemini fallback (T-173)

  **What to do**:
  - Detect Gemini Live API connection failure or timeout
  - Switch to REST TTS endpoint (gemini-2.5-flash-preview-tts) for voice output
  - Queue pending messages and deliver via TTS fallback
  - Send reconnecting indicator to client
  - Auto-resume Live session when Gemini becomes available

  **Recommended Agent Profile**: `deep` | Skills: []
  **Blocks**: None | **Blocked By**: Task 12

  **References**:
  - `docs/PRD/DETAILS/BACKEND_ARCHITECTURE.md` — Fallback architecture
  - `docs/reference/gemini/genai-sdk/01_sdk_overview.md` — REST TTS endpoint

  **QA Scenarios**:
  ```
  Scenario: TTS fallback on Gemini failure
    Tool: Bash
    Steps:
      1. Start gateway without Gemini connectivity (invalid API key)
      2. Send text message via WebSocket
      3. Verify TTS fallback produces audio response
    Expected Result: Fallback audio delivered within 3 seconds. Client notified of fallback mode.
    Evidence: .sisyphus/evidence/task-20-tts-fallback.txt
  ```
  **Commit**: YES — `feat(backend/gateway): add TTS fallback for Gemini Live API unavailability`

### Wave 4 — ADK Orchestrator Agents (Days 4-6)

- [x] 21. Initialize ADK project + HTTP endpoint (T-130)

  **What to do**:
  - Add `google.golang.org/adk` to orchestrator go.mod
  - Create HTTP `POST /analyze` endpoint accepting screen capture + context
  - Define internal agent module structure following ADK Go patterns from spike
  - Return structured JSON response (AnalysisResult)

  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocks**: Tasks 22-39, 47 | **Blocked By**: Task 2 (spike), 4

  **References**:
  - Task 2 spike output — ADK Go SDK API patterns
  - `docs/reference/adk/components/01_agents.md` — Agent definitions
  - `docs/PRD/DETAILS/BACKEND_ARCHITECTURE.md:80-200` — Agent interaction matrix

  **QA Scenarios**:
  ```
  Scenario: /analyze endpoint responds
    Tool: Bash
    Steps:
      1. Start orchestrator
      2. curl -X POST http://localhost:8080/analyze -d '{"image":"base64...","context":"test"}' -H 'Content-Type: application/json'
    Expected Result: Returns JSON with analysisResult structure
    Evidence: .sisyphus/evidence/task-21-analyze-endpoint.txt
  ```
  **Commit**: YES — `feat(backend/orchestrator): initialize ADK project with /analyze endpoint`

- [x] 22. Implement VisionAgent (T-131)

  **What to do**:
  - ADK agent: takes image+context, calls Gemini REST (gemini-2.0-flash) with responseSchema
  - Returns VisionAnalysis: `{significance, content, emotion, shouldSpeak, errorDetected, repeatedError, successDetected, errorMessage}`
  - Include frustration signals (T-154) and celebration patterns (T-159) in the same agent

  **Recommended Agent Profile**: `deep` | Skills: []
  **Blocks**: Tasks 23, 24, 25, 34 | **Blocked By**: Task 21

  **References**:
  - `docs/PRD/DETAILS/BACKEND_ARCHITECTURE.md` — VisionAgent spec
  - `docs/PRD/DETAILS/BACKEND_IMPLEMENTATION_TASKS.md:141-150` — VisionAgent steps

  **QA Scenarios**:
  ```
  Scenario: VisionAgent analyzes screenshot
    Tool: Bash
    Steps:
      1. Send test screenshot (error screen) to /analyze
      2. Parse VisionAnalysis from response
      3. Verify significance > 0, content non-empty, emotion defined
    Expected Result: Structured analysis with all required fields
    Evidence: .sisyphus/evidence/task-22-vision-agent.txt
  ```
  **Commit**: YES — `feat(backend/orchestrator): implement VisionAgent with structured screen analysis`

- [x] 23. Implement frustration signal detection in VisionAgent (T-154)

  **What to do**:
  - Extend VisionAgent response schema: `errorDetected`, `repeatedError` (same error 3+ times), `successDetected`, `errorMessage`
  - Track error history in agent state

  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 22

  **Commit**: YES (groups with 22) — `feat(backend/orchestrator): add frustration signal detection to VisionAgent`

- [x] 24. Implement celebration detection patterns in VisionAgent (T-159)

  **What to do**:
  - Add detection for: "tests passed", "Build succeeded", "Deployed", "PR merged", green CI badges
  - Return `successDetected=true` for matched patterns
  - Limit to 3 core patterns for MVP (tests passed, build succeeded, deployed)

  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 22

  **Commit**: YES (groups with 22)

- [x] 25. Implement Mediator (T-132)

  **What to do**:
  - ADK agent: receives VisionAnalysis, applies cooldown, significance threshold, duplicate detection
  - Returns MediatorDecision: `{shouldSpeak, reason, urgency}`
  - Decision tree: high significance → speak, low → skip, duplicate → skip, cooldown active → skip

  **Recommended Agent Profile**: `deep` | Skills: []
  **Blocks**: Tasks 26, 30, 33, 34 | **Blocked By**: Task 22

  **References**:
  - `docs/PRD/DETAILS/BACKEND_ARCHITECTURE.md` — Mediator decision matrix

  **QA Scenarios**:
  ```
  Scenario: Mediator gates speech correctly
    Tool: Bash (go test)
    Steps:
      1. cd backend/adk-orchestrator && go test ./internal/agents/mediator/... -v
    Expected Result: All decision branches tested: high sig → speak, low sig → skip, cooldown → skip
    Evidence: .sisyphus/evidence/task-25-mediator.txt
  ```
  **Commit**: YES — `feat(backend/orchestrator): implement Mediator with speech gating logic`

- [x] 26. Implement AdaptiveScheduler (T-133)

  **What to do**:
  - ADK agent: reads/writes Firestore metrics
  - Tracks utterances, responses, interruptions
  - Adjusts silenceThreshold (10-45s) and cooldownSeconds (5-20s)

  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocked By**: Tasks 6, 25

  **Commit**: YES — `feat(backend/orchestrator): implement AdaptiveScheduler with metric-based tuning`

- [x] 27. Implement EngagementAgent (T-134)

  **What to do**:
  - ADK agent: monitors silence duration from Firestore
  - Composes proactive prompt with context after silence threshold
  - Suppresses during active turn

  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocked By**: Task 26

  **Commit**: YES — `feat(backend/orchestrator): implement EngagementAgent for proactive triggers`

- [x] 28. Define mood classification schema (T-153)

  **What to do**:
  - Define 4 mood states: focused, frustrated, stuck, idle
  - Signal weights: repeated error (0.3), same file revisited (0.2), long silence after error (0.25), interaction rate drop (0.25)
  - Confidence threshold: 0.7

  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 21

  **Commit**: YES — `feat(backend/orchestrator): define mood classification schema and signal weights`

- [x] 29. Implement MoodDetector (T-155)

  **What to do**:
  - ADK agent: combines VisionAgent signals + silence duration + interaction rate
  - Output: MoodState `{mood, confidence, signals, suggestedAction}`
  - Update `sessions/{sessionId}/metrics.currentMood` in Firestore

  **Recommended Agent Profile**: `deep` | Skills: []
  **Blocked By**: Tasks 28, 23, 27

  **Commit**: YES — `feat(backend/orchestrator): implement MoodDetector with multi-signal fusion`

- [x] 30. Integrate mood into Mediator (T-156)

  **What to do**:
  - Mediator reads MoodState from MoodDetector
  - frustrated → lower speak threshold (more supportive messages)
  - focused → raise threshold (disturb less)
  - stuck → trigger SearchBuddy

  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocked By**: Tasks 29, 25

  **Commit**: YES — `feat(backend/orchestrator): add mood-aware speech gating to Mediator`

- [x] 31. Create supportive message templates (T-158)

  **What to do**:
  - Define 3 message variants per mood state (frustrated, stuck)
  - Korean primary + English variants
  - Hardcoded strings, not generated

  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 28

  **Commit**: YES — `feat(backend/orchestrator): add supportive message templates per mood state`

- [x] 32. Implement CelebrationTrigger (T-160)

  **What to do**:
  - ADK agent: receives VisionAgent.successDetected
  - Check cooldown (last celebration > 5 min ago)
  - Output: celebration event with trigger type, emotion=happy, celebratory message

  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocked By**: Task 24

  **Commit**: YES — `feat(backend/orchestrator): implement CelebrationTrigger agent`

- [x] 33. Integrate celebration into Mediator (T-162)

  **What to do**:
  - Celebration events bypass normal significance gating
  - Route directly to Gateway
  - Trigger happy sprite state

  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Tasks 32, 25

  **Commit**: YES (groups with 32)

- [x] 34. Wire ADK agent graph (T-135)

  **What to do**:
  - Connect all agents: screenCapture → VisionAgent → Mediator → routing (speak/silent)
  - EngagementAgent runs parallel timer-based
  - MoodDetector feeds into Mediator
  - CelebrationTrigger bypasses Mediator
  - Full graph executes E2E

  **Recommended Agent Profile**: `deep` | Skills: []
  **Blocks**: Tasks 35, 48 | **Blocked By**: Tasks 22, 25, 26, 27, 29, 32

  **References**:
  - `docs/reference/adk/streaming/01_adk_streaming_overview.md` — Graph wiring
  - Task 2 spike — ADK graph patterns

  **QA Scenarios**:
  ```
  Scenario: Full agent graph execution
    Tool: Bash
    Steps:
      1. Send test capture to /analyze
      2. Verify all agents executed in correct order (check logs)
      3. Verify structured result returned
    Expected Result: VisionAgent → Mediator → decision returned. Logs show agent chain.
    Evidence: .sisyphus/evidence/task-34-agent-graph.txt
  ```
  **Commit**: YES — `feat(backend/orchestrator): wire complete 9-agent graph`

- [x] 35. Implement Google Search grounding (T-136, T-164)

  **What to do**:
  - Register Google Search as ADK tool
  - Configure in agent graph for SearchBuddy
  - Test search returns results

  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocked By**: Task 34

  **Commit**: YES — `feat(backend/orchestrator): add Google Search grounding tool`

- [x] 36. Implement search trigger detection (T-165)

  **What to do**:
  - Monitor conversation for trigger phrases: "이거 뭐야", "왜 안돼", "어떻게 해", explicit search requests
  - Accept searchRequest WebSocket messages

  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 35

  **Commit**: YES (groups with 35)

- [x] 37. Implement auto-search from MoodDetector (T-166)

  **What to do**:
  - Track time on same error (VisionAgent.errorMessage)
  - After 3 min stuck → auto-trigger SearchBuddy
  - Only once per unique error

  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Tasks 29, 35

  **Commit**: YES — `feat(backend/orchestrator): add auto-search trigger from mood detection`

- [x] 38. Implement search result summarization (T-167)

  **What to do**:
  - Google Search results → Gemini summarization → 2-3 sentence voice-friendly response
  - Format: "찾아봤는데, [source]에서 [summary]"
  - Include source attribution

  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocked By**: Task 35

  **Commit**: YES — `feat(backend/orchestrator): add search result summarization for voice output`

- [x] 39. Integrate SearchBuddy response into Gateway voice (T-169)

  **What to do**:
  - SearchBuddy sends searchResult to Gateway
  - Gateway injects into Gemini Live API as tool response
  - Gemini speaks summarized result in current voice/persona

  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocked By**: Tasks 38, 12

  **Commit**: YES — `feat(backend): integrate search results into Gateway voice output`

### Wave 5 — Memory & Firestore Integration (Day 6)

- [x] 40. Define memory Firestore schema (T-147)

  **What to do**:
  - Create `users/{userId}/memory` collection with recentSummaries, knownTopics, codingPatterns
  - Set Firestore security rules for service access
  - Create indexes for topic queries

  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 5

  **Commit**: YES — `feat(backend/orchestrator): define cross-session memory Firestore schema`

- [x] 41. Implement end-of-session summary (T-148)

  **What to do**:
  - On session end, collect conversation highlights + VisionAgent findings
  - Call Gemini to generate summary with unresolved issues
  - Write to `users/{userId}/memory.recentSummaries` (cap at 10)

  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocked By**: Tasks 40, 12

  **Commit**: YES — `feat(backend): generate and store end-of-session summary`

- [x] 42. Implement session-start memory retrieval (T-149)

  **What to do**:
  - On session open, read `users/{userId}/memory`
  - Select most recent summary + unresolved topics
  - Format as context bridge text

  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocked By**: Task 40

  **Commit**: YES — `feat(backend/orchestrator): retrieve memory context at session start`

- [x] 43. Implement MemoryAgent in ADK graph (T-150)

  **What to do**:
  - ADK agent: on session start inject context bridge ("어제 그 API 이슈, 오늘은 괜찮아?")
  - Register memory tool for topic lookup during conversation
  - Wire into agent graph

  **Recommended Agent Profile**: `deep` | Skills: []
  **Blocked By**: Tasks 42, 21

  **Commit**: YES — `feat(backend/orchestrator): implement MemoryAgent with context bridge injection`

- [x] 44. Add memory context to Gateway session init (T-152)

  **What to do**:
  - Gateway requests memory from ADK Orchestrator at session open
  - Injects memory summary into Gemini systemInstruction
  - Handle MEMORY_UNAVAILABLE gracefully

  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocked By**: Tasks 42, 11

  **Commit**: YES — `feat(backend/gateway): inject memory context into Gemini session instruction`

- [x] 45. Add mood field to Firestore metrics (T-157)

  **What to do**:
  - Add currentMood, moodConfidence to session metrics
  - Log mood changes to session history

  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Tasks 29, 6

  **Commit**: YES — `feat(backend/orchestrator): track mood changes in Firestore metrics`

- [x] 46. Implement celebration cooldown (T-161)

  **What to do**:
  - Track celebrationCount and lastCelebrationAt in session metrics
  - Enforce 5-min gap between celebrations
  - Reset on session end

  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Tasks 32, 6

  **Commit**: YES — `feat(backend/orchestrator): add celebration cooldown tracking`

- [x] 47. Implement Gateway→ADK routing (T-117)

  **What to do**:
  - On screenCapture/forceCapture from client, POST to ADK Orchestrator `/analyze`
  - Receive analysis result, use to determine speech behavior
  - Service-to-service auth (IAM identity tokens for Cloud Run)

  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocks**: Tasks 48, 54 | **Blocked By**: Tasks 9, 21

  **QA Scenarios**:
  ```
  Scenario: Gateway routes to ADK and gets analysis
    Tool: Bash
    Steps:
      1. Start both services locally
      2. Send screenCapture via WebSocket to Gateway
      3. Verify Gateway calls ADK /analyze and returns result
    Expected Result: Analysis result flows Gateway → ADK → Gateway → Client
    Evidence: .sisyphus/evidence/task-47-gateway-adk-routing.txt
  ```
  **Commit**: YES — `feat(backend/gateway): route screen captures to ADK Orchestrator for analysis`

### Wave 6 — Backend Deploy & Verify (Day 7)

- [x] 48. Implement ADK fallback timeout (T-174)

  **What to do**:
  - Set 5s timeout on Gateway→ADK HTTP call
  - On timeout: return default decision (shouldSpeak: false)
  - Log timeout event, don't block audio pipeline

  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 47

  **Commit**: YES — `feat(backend/gateway): add 5s timeout and safe fallback for ADK calls`

- [x] 49. Containerize Realtime Gateway (T-140)

  **What to do**:
  - Multi-stage Dockerfile: Go build → minimal runtime image
  - Port 8080, health check, WebSocket support
  - Test: `docker build -t vibecat-gateway .`

  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 17

  **QA Scenarios**:
  ```
  Scenario: Docker image builds and runs
    Tool: Bash
    Steps:
      1. cd backend/realtime-gateway && docker build -t vibecat-gateway .
      2. docker run -p 8080:8080 -d vibecat-gateway
      3. curl -s http://localhost:8080/healthz
    Expected Result: Image builds, container starts, health check returns 200
    Evidence: .sisyphus/evidence/task-49-gateway-docker.txt
  ```
  **Commit**: YES — `feat(backend/gateway): add multi-stage Dockerfile`

- [x] 50. Containerize ADK Orchestrator (T-141)

  **What to do**:
  - Multi-stage Dockerfile: Go build → minimal runtime image
  - Port 8080, health check

  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 34

  **Commit**: YES — `feat(backend/orchestrator): add multi-stage Dockerfile`

- [ ] 51. Deploy to Cloud Run (T-142)

  **What to do**:
  - Use `infra/deploy.sh` to deploy both services
  - Configure: min 0, max 10 instances
  - Bind Secret Manager secrets
  - Set env vars (ORCHESTRATOR_URL, PROJECT_ID, REGION)
  - Service-to-service IAM auth

  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocks**: Tasks 52, 53, 54 | **Blocked By**: Tasks 49, 50

  **QA Scenarios**:
  ```
  Scenario: Both services deployed and healthy
    Tool: Bash
    Steps:
      1. ./infra/deploy.sh
      2. curl -s https://realtime-gateway-*.asia-northeast3.run.app/healthz
      3. curl -s https://adk-orchestrator-*.asia-northeast3.run.app/healthz
    Expected Result: Both return {"status":"ok"}
    Evidence: .sisyphus/evidence/task-51-cloud-run-deploy.txt
  ```
  **Commit**: YES — `feat(infra): deploy gateway and orchestrator to Cloud Run`

- [ ] 52. Configure Cloud Logging + Monitoring (T-143)

  **What to do**:
  - Structured JSON logging in both services
  - Create monitoring dashboard (connections, latency, errors)

  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 51

  **Commit**: YES — `feat(infra): configure Cloud Logging and Monitoring dashboard`

- [ ] 53. Configure Cloud Trace (T-144)

  **What to do**:
  - Trace context propagation Gateway↔Orchestrator
  - Instrument key operations

  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 51

  **Commit**: YES — `feat(infra): configure Cloud Trace for distributed tracing`

- [ ] 54. Backend E2E test — backend only (T-145 partial)

  **What to do**:
  - Test via wscat/curl: connect → send capture → receive analysis → audio plays
  - Test voice chat, interruption, reconnection (backend only)
  - Full E2E with client deferred to Wave 14

  **Recommended Agent Profile**: `deep` | Skills: []
  **Blocked By**: Tasks 51, 47

  **QA Scenarios**:
  ```
  Scenario: Full backend E2E via wscat
    Tool: Bash
    Steps:
      1. Connect to deployed Gateway via wscat
      2. Authenticate
      3. Send screen capture (base64 test image)
      4. Verify analysis result received
      5. Send audio, verify audio response
    Expected Result: Full capture→analysis→speech pipeline works on Cloud Run
    Evidence: .sisyphus/evidence/task-54-backend-e2e.txt
  ```
  **Commit**: NO (test only)

### Wave 7 — Client Bootstrap (Days 7-8)

- [x] 55. Complete workspace + source directories (T-001)

  **What to do**:
  - Verify directories from Task 3 exist. Create `Sources/Core/`, `Sources/VibeCat/`, `Tests/VibeCatTests/` with proper Swift placeholders
  - Ensure `swift build` produces both `Core` and `VibeCat` targets
  - Follow structure in `docs/PRD/DETAILS/IMPLEMENTATION_EXECUTION_PLAN.md:35-57`

  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocks**: Tasks 56-68 | **Blocked By**: Task 3

  **References**:
  - `docs/PRD/DETAILS/IMPLEMENTATION_EXECUTION_PLAN.md:35-57` — Module inventory
  - `VibeCat/Package.swift` — SPM manifest
  - `docs/PRD/DETAILS/END_TO_END_IMPLEMENTATION_TASKS.md:14-19` — T-001 spec

  **QA Scenarios**:
  ```
  Scenario: Both targets build
    Tool: Bash
    Steps:
      1. cd VibeCat && swift build 2>&1 | tail -3
    Expected Result: "Build complete!" with zero errors
    Evidence: .sisyphus/evidence/task-55-swift-build.txt
  ```
  **Commit**: YES — `feat(client): complete workspace skeleton with Core and VibeCat targets`

- [x] 56. App metadata + permissions (T-002)

  **What to do**: Verify/update bundle ID, app name, screen recording + microphone permission descriptions in Info.plist. Validate entitlements for ScreenCaptureKit.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 55
  **References**: `docs/PRD/DETAILS/END_TO_END_IMPLEMENTATION_TASKS.md:22-26` — T-002 spec, `VibeCat/Info.plist`, `VibeCat/VibeCat.entitlements`
  **Commit**: YES — `feat(client): configure app metadata and runtime permissions`

- [x] 57. Runtime config model (T-003)

  **What to do**: Define API key loading: session token from backend → keychain storage. Config validation (empty/malformed/valid/timeout/error).
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 55
  **References**: `docs/PRD/DETAILS/END_TO_END_IMPLEMENTATION_TASKS.md:28-33` — T-003 spec
  **Commit**: YES — `feat(client/core): implement runtime configuration model with backend auth`

- [x] 58. Settings persistence (T-004)

  **What to do**: Define UserDefaults-backed settings: language, voice, capture, models, interaction, music. Round-trip tests.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 55
  **References**: `docs/PRD/DETAILS/END_TO_END_IMPLEMENTATION_TASKS.md:35-40` — T-004 spec
  **Commit**: YES — `feat(client/core): implement settings persistence with UserDefaults`

- [x] 59. Fix CI baseline (T-005)

  **What to do**: Update `.github/workflows/ci.yml` to match actual source structure. Ensure Swift build + test + Go build + test all pass.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Tasks 55, 4
  **References**: `.github/workflows/ci.yml`
  **Commit**: YES — `fix(ci): update CI workflow for actual source structure`

### Wave 8 — Client Core Library (Day 8)

- [x] 60. Domain models (T-010)

  **What to do**: ChatMessage, SettingsTypes, CharacterPresetConfig — stable shared types. Serialization tests.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 58
  **References**: `docs/PRD/DETAILS/END_TO_END_IMPLEMENTATION_TASKS.md:52-56` — T-010 spec
  **Commit**: YES — `feat(client/core): implement domain models and settings types`

- [x] 61. Prompt composition (T-011)

  **What to do**: PromptBuilder — deterministic prompt composition for live, fallback, engagement. Language-aware. NOTE: In client-backend split, this is now a thin client-side helper that prepares context for backend prompts.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 60
  **Commit**: YES — `feat(client/core): implement prompt composition primitives`

- [x] 62. Image processing (T-012)

  **What to do**: ImageProcessor — resize-if-needed, JPEG conversion with quality bounds. CoreGraphics-based.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 60
  **References**: `docs/PRD/DETAILS/END_TO_END_IMPLEMENTATION_TASKS.md:66-70` — T-012 spec
  **Commit**: YES — `feat(client/core): implement image processing with resize and JPEG conversion`

- [x] 63. Image encoding (T-013)

  **What to do**: ImageEncoder — text-only, text+image, realtime image, realtime audio payload builders for WebSocket transport.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 62
  **Commit**: YES — `feat(client/core): implement image encoding and realtime payload helpers`

- [x] 64. Visual change detection (T-014)

  **What to do**: ImageDiffer — thumbnail generation, threshold-based pixel diff to skip unchanged screens.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 62
  **Commit**: YES — `feat(client/core): implement visual change detection`

- [x] 65. Audio message parsing (T-015)

  **What to do**: AudioMessageParser — decode server audio chunks, transcription events, interruptions, turn-complete from WebSocket messages.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 63
  **Commit**: YES — `feat(client/core): implement audio message parser`

- [x] 66. PCM conversion (T-016)

  **What to do**: PCMConverter — Int16↔Float32 conversion with deterministic normalization. Edge-value tests (min/max/zero/odd).
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 60
  **Commit**: YES — `feat(client/core): implement PCM conversion primitives`

- [x] 67. Keychain helper (T-017)

  **What to do**: KeychainHelper — save/load/delete with stable service/account names. Stores backend session token.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 57
  **Commit**: YES — `feat(client/core): implement secure keychain storage wrapper`

- [x] 68. Core test gate (T-018)

  **What to do**: Run all Core module tests. Verify zero failures before proceeding to app layer.
  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocked By**: Tasks 61-67
  **QA Scenarios**:
  ```
  Scenario: All core tests pass
    Tool: Bash
    Steps:
      1. cd VibeCat && swift test --filter VibeCatTests 2>&1
    Expected Result: All tests pass with zero failures
    Evidence: .sisyphus/evidence/task-68-core-tests.txt
  ```
  **Commit**: NO (gate check only)

### Wave 9 — Client Menu & Settings (Day 9)

- [ ] 69. Status bar controller (T-020)

  **What to do**: StatusBarController — menu bar entry point, root menu, status row, callback registration.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 58
  **References**: `docs/PRD/DETAILS/MENU_AND_RUNTIME_OPERATIONS_SPEC.md`
  **Commit**: YES — `feat(client/app): implement status bar controller skeleton`

- [ ] 70. Tray icon animation (T-021)

  **What to do**: SpriteAnimator for tray — load TrayIcons_Clean frames, frame timer, emotion-state switching.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 69
  **References**: `Assets/TrayIcons_Clean/` — 24 PNGs (8 frames × 3 scales)
  **Commit**: YES — `feat(client/app): implement tray icon animation pipeline`

- [ ] 71. Language/voice/chattiness menus (T-022)

  **What to do**: Checkable submenus bound to persisted settings. Selection survives restart.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 69
  **Commit**: YES — `feat(client/app): add language, voice, and chattiness submenus`

- [ ] 72. Model/reasoning controls (T-023)

  **What to do**: Nested model menus, reconnect triggers for live model changes.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 69
  **Commit**: YES — `feat(client/app): add model and reasoning controls`

- [ ] 73. Capture/appearance controls (T-024)

  **What to do**: Capture interval/sensitivity/quality, visual presentation controls. Runtime-apply without relaunch.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 69
  **Commit**: YES — `feat(client/app): add capture and appearance controls`

- [ ] 74. Advanced/music controls (T-025)

  **What to do**: Google Search toggle, proactive audio, affective dialog, background music track + volume. Skip launch-at-login for MVP.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 69
  **Commit**: YES — `feat(client/app): add advanced and background music controls`

- [ ] 75. API key onboarding (T-026)

  **What to do**: First-launch key entry → backend validation → token receipt → keychain storage. Error feedback.
  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocked By**: Tasks 57, 69
  **Commit**: YES — `feat(client/app): implement API key onboarding and validation flow`

- [ ] 76. Reconnect/pause/mute/quit (T-027)

  **What to do**: Action handlers tied to orchestrator/live/audio components. Pause stops loops, mute clears speech, reconnect restarts live client, quit exits cleanly.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 69
  **Commit**: YES — `feat(client/app): implement reconnect, pause, mute, and quit actions`

### Wave 10 — Client Capture & Transport (Days 9-10)

- [ ] 77. Capture around cursor (T-030)

  **What to do**: ScreenCaptureService — region/window capture with ScreenCaptureKit, panel exclusion, JPEG output.
  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocked By**: Tasks 62, 64
  **References**: `docs/PRD/DETAILS/END_TO_END_IMPLEMENTATION_TASKS.md:174-179` — T-030 spec
  **Commit**: YES — `feat(client/app): implement screen capture around cursor`

- [ ] 78. Changed-only capture path (T-031)

  **What to do**: Compare thumbnails, emit unchanged/unavailable/captured outcomes. Unchanged screens skip analysis.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 77
  **Commit**: YES — `feat(client/app): implement changed-only capture filtering`

- [ ] 79. Full window capture (T-032)

  **What to do**: High-significance deep context: capture frontmost app window with stricter max dimensions.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 77
  **Commit**: YES — `feat(client/app): implement full window capture for high-significance analysis`

- [ ] 80. WebSocket lifecycle — connect to Gateway (T-040)

  **What to do**: GeminiLiveClient → connect to deployed Gateway WebSocket (`wss://{host}/ws/live`). Send setup payload, receive loop, disconnect, manual reconnect. Use token from Task 75's onboarding flow.
  **Recommended Agent Profile**: `deep` | Skills: []
  **Blocks**: Tasks 81-86 | **Blocked By**: Tasks 63, 57

  **References**:
  - `docs/PRD/DETAILS/CLIENT_BACKEND_PROTOCOL.md` — Full WebSocket message spec
  - `docs/PRD/DETAILS/END_TO_END_IMPLEMENTATION_TASKS.md:206-209` — T-040 spec

  **QA Scenarios**:
  ```
  Scenario: Client connects to deployed Gateway
    Tool: Bash (tmux)
    Steps:
      1. Run VibeCat app
      2. Enter API key in onboarding
      3. Check logs for "WebSocket connected to Gateway"
    Expected Result: Stable WebSocket connection established
    Evidence: .sisyphus/evidence/task-80-ws-connect.txt
  ```
  **Commit**: YES — `feat(client/app): implement WebSocket lifecycle connecting to Gateway`

- [ ] 81. Setup payload policy (T-041)

  **What to do**: Map settings → setup payload fields. Language, voice, tools, proactivity, session resumption.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 80
  **Commit**: YES — `feat(client/app): implement setup payload policy from settings`

- [ ] 82. Server message handling (T-042)

  **What to do**: Route live audio, transcription, interruption, turn-complete, analysisResult, celebration, mood events to callbacks.
  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocked By**: Tasks 65, 80
  **Commit**: YES — `feat(client/app): implement server message routing and callbacks`

- [ ] 83. Heartbeat + zombie detection (T-043)

  **What to do**: Client-side ping/pong, heartbeat timer, forced reconnect on stale connection.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 80
  **Commit**: YES — `feat(client/app): add heartbeat and zombie detection`

- [ ] 84. Audio playback engine (T-044)

  **What to do**: AudioPlayer — enqueue decoded PCM buffers, AVAudioEngine playback, state tracking (isPlaying).
  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocked By**: Task 66
  **Commit**: YES — `feat(client/app): implement streaming audio playback engine`

- [ ] 85. Speech wrapper + TTS fallback (T-045)

  **What to do**: CatVoice — unified speech output. Delegates to AudioPlayer for live, falls back to local TTS when disconnected.
  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocked By**: Task 84
  **Commit**: YES — `feat(client/app): implement speech wrapper with TTS fallback`

- [ ] 86. Background music (T-046)

  **What to do**: BackgroundMusicPlayer — preload track from `Assets/Music/`, loop playback. Binary on/off for MVP (no fade transitions).
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 84
  **Commit**: YES — `feat(client/app): implement background music player`

### Wave 11 — Client Agents & Orchestrator (Day 10)

- [ ] 87. VisionAgent client-side (T-050)

  **What to do**: Client-side VisionAgent — sends image+prompt to backend via WebSocket screenCapture message, parses structured AnalysisResult response.
  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocked By**: Tasks 63, 77
  **Commit**: YES — `feat(client/app): implement client-side VisionAgent`

- [ ] 88. Mediator client-side (T-051)

  **What to do**: Client receives MediatorDecision from backend. Routes speak/skip decisions to speech output.
  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocked By**: Task 87
  **Commit**: YES — `feat(client/app): implement client-side Mediator decision handling`

- [ ] 89. Adaptive scheduler client-side (T-052)

  **What to do**: Client applies scheduling params from backend (silenceThreshold, cooldownSeconds).
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 88
  **Commit**: YES — `feat(client/app): implement adaptive scheduler client sync`

- [ ] 90. Engagement agent client-side (T-053)

  **What to do**: Client receives proactive prompts from backend, plays them through speech system.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 89
  **Commit**: YES — `feat(client/app): handle engagement agent proactive triggers`

- [ ] 91. Cat view model (T-060)

  **What to do**: CatViewModel — cursor tracking, lerp movement, jump threshold, panel migration across screens. Boundary-safe.
  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocked By**: Task 73
  **Commit**: YES — `feat(client/app): implement cat view model with cursor tracking`

- [ ] 92. Sprite animator (T-061)

  **What to do**: SpriteAnimator — resolve sprite root per character, load per-state frames (idle/thinking/happy/surprised), frame timer, character switching.
  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocked By**: Task 70
  **References**: `Assets/Sprites/{character}/` — 6 characters, ~16 frames each
  **Commit**: YES — `feat(client/app): implement sprite animator with character switching`

- [ ] 93. Bubble + emotion indicators (T-062)

  **What to do**: ChatBubbleView — speech text display, visibility animation, edge-safe placement near cat.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Tasks 91, 92
  **Commit**: YES — `feat(client/app): implement chat bubble and emotion indicators`

- [ ] 94. Screen analyzer loop (T-063)

  **What to do**: ScreenAnalyzer — periodic capture cycle → send to backend → receive decision → route speech/skip. No concurrent overlap, no deadlocks.
  **Recommended Agent Profile**: `deep` | Skills: []
  **Blocks**: Tasks 95-97 | **Blocked By**: Tasks 78, 87, 88

  **QA Scenarios**:
  ```
  Scenario: Analysis loop runs without deadlock
    Tool: Bash (tmux)
    Steps:
      1. Run app for 60 seconds with active screen
      2. Check logs for periodic analysis cycles
      3. Verify no duplicate concurrent cycles
    Expected Result: Regular capture→analysis→decision cycles in logs
    Evidence: .sisyphus/evidence/task-94-analyzer-loop.txt
  ```
  **Commit**: YES — `feat(client/app): implement screen analyzer orchestration loop`

- [ ] 95. High-significance branch (T-064)

  **What to do**: On high significance score, capture full active window and include in analysis.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Tasks 79, 94
  **Commit**: YES — `feat(client/app): add high-significance full-window capture branch`

- [ ] 96. Live transcription → bubble (T-065)

  **What to do**: Buffer partial text, handle sentence finished, flush on turn-complete.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Tasks 82, 94
  **Commit**: YES — `feat(client/app): map live transcription to bubble updates`

- [ ] 97. REST fallback speech (T-066)

  **What to do**: When live socket unavailable, generate fallback text → TTS → play audio.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Tasks 85, 94
  **Commit**: YES — `feat(client/app): implement REST fallback speech path`

### Wave 12 — Client Voice/Chat & App Integration (Day 11)

- [ ] 98. Global hotkey (T-070)

  **What to do**: GlobalHotkeyManager — monitor global/local key events, trigger chat/voice interaction.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 73
  **Commit**: YES — `feat(client/app): implement global hotkey manager`

- [ ] 99. Speech recognizer (T-071)

  **What to do**: SpeechRecognizer — request permissions, start recognition, detect wake words, extract trailing query.
  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocked By**: Tasks 56, 98
  **Commit**: YES — `feat(client/app): implement speech recognizer with wake-word detection`

- [ ] 100. Chat panel container (T-072)

  **What to do**: GeminiChatPanel — borderless panel near cat, show/dismiss lifecycle, screen-bound positioning.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 91
  **Commit**: YES — `feat(client/app): implement chat panel container`

- [ ] 101. Chat view + messages (T-073)

  **What to do**: GeminiChatView — role-based message rows, input field, auto-scroll, submit flow.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 100
  **Commit**: YES — `feat(client/app): implement chat view with message list`

- [ ] 102. Interaction mode wiring (T-074)

  **What to do**: Wire hotkey → speech recognizer → chat panel → send path.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Tasks 98, 99, 100, 101
  **Commit**: YES — `feat(client/app): wire interaction modes (hotkey, voice, chat)`

- [ ] 103. Main entrypoint + duplicate guard (T-080)

  **What to do**: main.swift — duplicate detection by bundle ID, accessory activation policy.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 55
  **Commit**: YES — `feat(client/app): implement main entrypoint with duplicate-instance guard`

- [ ] 104. App delegate wiring (T-081)

  **What to do**: AppDelegate — instantiate all components, inject dependencies, attach callbacks. Correct initialization order.
  **Recommended Agent Profile**: `deep` | Skills: []
  **Blocks**: Tasks 105-108 | **Blocked By**: Tasks 94, 102

  **QA Scenarios**:
  ```
  Scenario: App starts without nil-access errors
    Tool: Bash (tmux)
    Steps:
      1. make run
      2. Check console for crash or nil-access errors
      3. Verify menu bar icon appears
    Expected Result: App launches cleanly, menu bar icon visible
    Evidence: .sisyphus/evidence/task-104-app-launch.txt
  ```
  **Commit**: YES — `feat(client/app): implement app delegate with full dependency wiring`

- [ ] 105. Floating panel behavior (T-082)

  **What to do**: Panel level, transparency, collection behavior, mouse-event policy. Visible across spaces, click-through.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 104
  **Commit**: YES — `feat(client/app): configure floating panel behavior`

- [ ] 106. Status bar callback wiring (T-083)

  **What to do**: Wire all menu actions → runtime changes (pause/mute/reconnect/model/capture/music/reset).
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Tasks 76, 104
  **Commit**: YES — `feat(client/app): wire status bar menu callbacks`

- [ ] 107. Startup mode behavior (T-084)

  **What to do**: Onboarding (no key) vs. direct analysis start (key exists + not paused).
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Tasks 75, 104
  **Commit**: YES — `feat(client/app): implement startup mode with and without API key`

- [ ] 108. Companion message types (T-097)

  **What to do**: Handle new WebSocket message types: memoryContext (display on session start), moodUpdate (adjust sprite), celebration (happy sprite + animation), searchResult (display sources in chat), searchRequest, memoryFeedback.
  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocked By**: Tasks 80, 97
  **Commit**: YES — `feat(client/app): handle companion intelligence message types`

### Wave 13 — Grand Prize Features & Polish (Days 11-12)

- [ ] 109. Decision Overlay HUD (T-094)

  **What to do**: Translucent overlay: trigger source, VisionAgent detection, Mediator decision, MoodDetector state, cooldown timer. Toggle with hotkey. MAX 5 fields, no charts/graphs.
  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocked By**: Task 97
  **Commit**: YES — `feat(client/app): implement Decision Overlay HUD`

- [ ] 110. Sprite screen pointing (T-095)

  **What to do**: Animate cat toward error region coordinates from VisionAgent. Linear lerp only (no bezier/physics). Return to home after 5s.
  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocked By**: Tasks 92, 97
  **Commit**: YES — `feat(client/app): implement sprite screen pointing toward errors`

- [ ] 111. Session farewell + sleep overlay (T-096)

  **What to do**: On session end: farewell message ("오늘 고생했어, 내일 보자"), idle sprite with dimmed overlay (no sleeping sprite — per user decision), save session summary via backend.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Tasks 97, 41 (backend T-148)
  **Commit**: YES — `feat(client/app): implement session farewell with sleep overlay`

- [ ] 112. Graceful reconnection UX (T-098)

  **What to do**: On disconnect: thinking sprite + reconnecting indicator. 1s delay reconnect. On success: send resumptionHandle + "돌아왔어!" message. 3 consecutive failures → error state.
  **Recommended Agent Profile**: `unspecified-high` | Skills: []
  **Blocked By**: Tasks 80, 20 (backend T-173)
  **Commit**: YES — `feat(client/app): implement graceful reconnection UX`

- [ ] 113. Privacy controls UI (T-099)

  **What to do**: Capture indicator (menu bar dot), one-click pause, manual-only mode, "no screenshots stored" statement.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Tasks 72, 97
  **Commit**: YES — `feat(client/app): add privacy controls UI`

- [ ] 114. Celebration client protocol (T-163)

  **What to do**: Handle celebration WebSocket message → happy sprite → celebration voice → return to idle.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 108
  **Commit**: YES — `feat(client/app): handle celebration events from backend`

- [ ] 115. Search result client display

  **What to do**: Display searchResult in chat panel with clickable source links.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Task 108
  **Commit**: YES — `feat(client/app): display search results with source links in chat`

### Wave 14 — End-to-End Validation (Day 12)

- [ ] 116. Asset validation (T-090)

  **What to do**: Validate sprite/tray/music/voice-sample counts. Runtime path resolution for all characters.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Tasks 92, 86

  **QA Scenarios**:
  ```
  Scenario: All assets discoverable at runtime
    Tool: Bash
    Steps:
      1. Count files in Assets/Sprites/cat/ — expect 16+
      2. Count files in Assets/TrayIcons_Clean/ — expect 24
      3. Count files in Assets/Music/ — expect 2
    Expected Result: All asset counts match expectations
    Evidence: .sisyphus/evidence/task-116-asset-validation.txt
  ```
  **Commit**: NO (validation only)

- [ ] 117. Full operation scenarios (T-091)

  **What to do**: Execute end-to-end: first-launch onboarding → live connection → capture cycle → proactive trigger → fallback mode → pause/resume → mute/unmute → reconnect.
  **Recommended Agent Profile**: `deep` | Skills: [`playwright`]
  **Blocks**: Tasks 118, 121 | **Blocked By**: Tasks 107, 116

  **QA Scenarios**:
  ```
  Scenario: Full E2E operation
    Tool: Bash (tmux)
    Steps:
      1. make run (fresh state)
      2. Enter API key → connection established
      3. Wait for capture cycle → agent speaks
      4. Pause → capture stops
      5. Resume → capture restarts
      6. Mute → speech stops
      7. Reconnect → new session
    Expected Result: All scenarios produce expected state transitions
    Evidence: .sisyphus/evidence/task-117-full-e2e.txt
  ```
  **Commit**: NO (E2E validation)

- [ ] 118. Full E2E integration test (T-145 complete)

  **What to do**: Client + deployed backend: capture → analysis → speech. Voice chat, gesture, interruption, reconnection.
  **Recommended Agent Profile**: `deep` | Skills: []
  **Blocked By**: Tasks 51, 117

  **QA Scenarios**:
  ```
  Scenario: Client↔CloudRun full flow
    Tool: Bash (tmux)
    Steps:
      1. Run app connected to Cloud Run backend
      2. Send screen capture → receive analysis → hear speech
      3. Test voice conversation
      4. Test reconnection
    Expected Result: Latency < 3 seconds capture-to-speech
    Evidence: .sisyphus/evidence/task-118-full-integration.txt
  ```
  **Commit**: NO (E2E test)

- [ ] 119. Companion intelligence E2E (T-170)

  **What to do**: Full session flow: session start → memory loads → coding → error → mood shifts → 3 min stuck → search → fix → tests pass → celebration → session end → summary saved.
  **Recommended Agent Profile**: `deep` | Skills: []
  **Blocked By**: Tasks 44, 30, 33, 39
  **Commit**: NO (E2E test)

- [ ] 120. Topic detection stub (T-151)

  **What to do**: Keyword matching only (not NLP). Detect recurring themes from conversation. Update knownTopics in Firestore.
  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Tasks 43, 40
  **Commit**: YES — `feat(backend/orchestrator): add keyword-based topic detection stub`

### Wave 15 — Submission Artifacts (Days 12-13)

> 🔴 **DEMO VIDEO + BLOG POST** — This is the submission. Everything before this is just preparation.

- [ ] 121. Deployment evidence pack (T-092, T-146)

  **What to do**: Screenshots of Cloud Run, Firestore, Logging, Monitoring, Trace dashboards. Fill `docs/deployment/PROOF_OF_GCP_DEPLOYMENT.md`.
  **Recommended Agent Profile**: `unspecified-high` | Skills: [`playwright`]
  **Blocked By**: Tasks 52, 53, 117

  **QA Scenarios**:
  ```
  Scenario: All evidence artifacts exist
    Tool: Bash
    Steps:
      1. ls docs/deployment/PROOF_OF_GCP_DEPLOYMENT.md
      2. Verify it contains Cloud Run, Firestore, Logging, Monitoring screenshots
    Expected Result: All evidence files present and linked
    Evidence: .sisyphus/evidence/task-121-deployment-evidence.txt
  ```
  **Commit**: YES — `docs(deployment): complete GCP deployment evidence pack`

- [ ] 122. Demo video recording (≤4 min)

  **What to do**:
  - Record 4-minute demo following scenes in `docs/PRD/DETAILS/SUBMISSION_AND_DEMO_PLAN.md`
  - 13 scenes: session start, conversation, screen analysis, decision HUD, search, mood, celebration, quiet moment, resilience, architecture, cloud proof, farewell
  - Upload to YouTube (public)
  - English subtitles (challenge requirement)
  - Pre-grant all permissions before recording, enable DND

  **Recommended Agent Profile**: `writing` | Skills: []
  **Blocked By**: Tasks 117, 118

  **References**:
  - `docs/PRD/DETAILS/SUBMISSION_AND_DEMO_PLAN.md` — Demo script and scenes

  **QA Scenarios**:
  ```
  Scenario: Demo video meets requirements
    Tool: Bash
    Steps:
      1. ffprobe demo.mp4 -show_entries format=duration -v quiet -of csv="p=0"
      2. Verify duration ≤ 240 seconds
    Expected Result: Video under 4 minutes, all scenes present
    Evidence: .sisyphus/evidence/task-122-demo-video.txt
  ```
  **Commit**: YES — `docs(submission): add demo video`

- [ ] 123. Blog post writing + publish

  **What to do**:
  - Write blog post about VibeCat development journey
  - Include `#GeminiLiveAgentChallenge` tag (REQUIRED for +0.6 bonus)
  - Cover: problem, approach, architecture, agents, challenges, learnings
  - Publish on dev.to or Medium

  **Recommended Agent Profile**: `writing` | Skills: [`devto-daily-devlog-uploader`]
  **Blocked By**: Task 122

  **Commit**: NO (external publication)

- [ ] 124. Devpost submission assembly

  **What to do**:
  - Text description (features, stack, learnings)
  - Video URL (YouTube)
  - Public repo URL (https://github.com/Two-Weeks-Team/vibeCat)
  - Blog post URL
  - Architecture diagram
  - Deployment evidence

  **Recommended Agent Profile**: `quick` | Skills: []
  **Blocked By**: Tasks 121, 122, 123

  **Commit**: NO (external submission)

- [ ] 125. Final handoff gate (T-093)

  **What to do**: Confirm all artifacts complete. Doc-only dry-run walkthrough. No open blockers. Link validation clean.
  **Recommended Agent Profile**: `deep` | Skills: []
  **Blocked By**: Task 124
  **Commit**: YES — `docs: complete final handoff gate verification`

---

## Final Verification Wave (MANDATORY — after ALL implementation tasks)

> 4 review agents run in PARALLEL. ALL must APPROVE. Rejection → fix → re-run.

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, curl endpoint, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan. Verify all 9 agents are functional and demonstrated.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `swift build` + `go build ./...` + `go test ./...`. Review all changed files for: `as any`/`@ts-ignore`, empty catches, print/println in prod, commented-out code, unused imports. Check AI slop: excessive comments, over-abstraction, generic names. Verify no API keys in source code.
  Output: `Build [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [ ] F3. **Real Manual QA** — `unspecified-high`
  Start from clean state. Test: app launches → API key entry → screen capture starts → agent speaks → voice conversation → mood detection → celebration → session end. Save screenshots to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff. Verify 1:1. Check "Must NOT do" compliance. Detect scope creep. Flag unaccounted changes. Verify demo video matches actual app behavior.
  Output: `Tasks [N/N compliant] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

- Atomic commits after each passing task
- Format: `type(scope): description`
- Scopes: `backend/gateway`, `backend/orchestrator`, `client/core`, `client/app`, `infra`, `docs`
- All commits within contest period (Feb 16 - Mar 16, 2026)
- Never commit secrets, API keys, or `.env` files
- Pre-commit: relevant build/test command must pass

---

## Success Criteria

### Verification Commands
```bash
# Swift build
cd VibeCat && swift build 2>&1 | tail -1
# Expected: "Build complete!"

# Go Gateway build
cd backend/realtime-gateway && go build ./...
# Expected: no output (success)

# Go Orchestrator build  
cd backend/adk-orchestrator && go build ./...
# Expected: no output (success)

# Go tests
cd backend/realtime-gateway && go test ./...
cd backend/adk-orchestrator && go test ./...
# Expected: PASS

# Cloud Run health
curl -s https://realtime-gateway-XXXXX.asia-northeast3.run.app/healthz
# Expected: {"status":"ok"}

curl -s https://adk-orchestrator-XXXXX.asia-northeast3.run.app/healthz
# Expected: {"status":"ok"}

# Demo video duration
ffprobe demo.mp4 -show_entries format=duration -v quiet -of csv="p=0"
# Expected: ≤ 240 (seconds)
```

### Final Checklist
- [ ] All 9 agents functional
- [ ] GenAI SDK + ADK + Gemini Live API + VAD — all four present
- [ ] Client never makes direct Gemini API calls
- [ ] No API keys in source code
- [ ] Cloud Run deployment active
- [ ] Demo video ≤4 min uploaded
- [ ] Blog post published with #GeminiLiveAgentChallenge
- [ ] Devpost submission complete
- [ ] English language support functional

---

## 🔴 REMINDER: Demo Video + Blog Post
> **Demo video (4 min max)** and **blog post** (`#GeminiLiveAgentChallenge`, +0.6 bonus) are REQUIRED for submission.
> Reserve final 2 days (March 14-15) for recording, editing, writing, and submitting.
> Do NOT start demo recording until Wave 14 (E2E validation) is complete.
