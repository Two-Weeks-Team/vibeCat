# VibeCat Deployment Evidence

**Collected**: 2026-03-06
**Project**: vibecat-489105
**Region**: asia-northeast3 (Seoul)
**Last Deploy**: 2026-03-06

---

## 1. Cloud Run Services

### Realtime Gateway

| Field | Value |
|-------|-------|
| Service Name | `realtime-gateway` |
| URL | `https://realtime-gateway-163070481841.asia-northeast3.run.app` |
| Status | **Ready** (True) |
| Active Revision | `realtime-gateway-00006-mq9` |
| Traffic | 100% → latest revision |
| Created | 2026-03-04T00:28:46Z |
| Auth | Public (`allUsers` → `roles/run.invoker`) |

**Key Changes in 00006**:
- Live API reconnection race condition fix (`reconnecting` state flag)
- Conditional session nil with pointer comparison (prevents killing newer sessions)
- Audio frame drop during reconnect state

**Health Check**:
```
GET /health → 200 OK
{"connections":0,"service":"realtime-gateway","status":"ok"}

GET /readyz → 200 OK
{"service":"realtime-gateway","status":"ok"}
```

### ADK Orchestrator

| Field | Value |
|-------|-------|
| Service Name | `adk-orchestrator` |
| URL | `https://adk-orchestrator-163070481841.asia-northeast3.run.app` |
| Status | **Ready** (True) |
| Active Revision | `adk-orchestrator-00008-5t2` |
| Traffic | 100% → latest revision |
| Access | Internal only (invoked by Gateway via `POST /analyze`) |

**Key Changes in 00008**:
- LLM dynamic message generation for Mediator, Celebration, Engagement agents
- All hardcoded speech message pools replaced with `gemini-3.1-flash-lite-preview` generation
- `llmSearchAgent` wired as live SubAgent in wave3 (previously dead code)
- `genaiClient` passed to mediator, celebration, engagement constructors

**Health Check**:
```
GET /health → 200 OK
```

---

## 2. GCP Infrastructure

### Firestore

| Field | Value |
|-------|-------|
| Database | `(default)` |
| Type | FIRESTORE_NATIVE |
| Location | asia-northeast3 |
| Collections | sessions, metrics, memory |

### Secret Manager

| Secret | Created |
|--------|---------|
| `vibecat-gemini-api-key` | 2026-03-04T00:19:19 |
| `vibecat-gateway-auth-secret` | 2026-03-04T00:19:22 |

### Artifact Registry

| Repository | Format | Created |
|------------|--------|---------|
| `vibecat-images` | DOCKER | 2026-03-04T09:19:04 |

### Observability

| Service | Status | Details |
|---------|--------|---------|
| Cloud Trace | ✅ Active | OpenTelemetry → `opentelemetry-operations-go/exporter/trace` |
| Cloud Logging | ✅ Active | `cloud.google.com/go/logging` structured JSON |
| ADK Telemetry | ✅ Active | `google.golang.org/adk/telemetry` with GCP resource |

---

## 3. CI/CD Pipeline

### GitHub Actions CI — Run #22657462395

**Repository**: `Two-Weeks-Team/vibeCat`
**Branch**: `master`
**Status**: **SUCCESS** (all 4 jobs passed)
**URL**: `https://github.com/Two-Weeks-Team/vibeCat/actions/runs/22657462395`

| Job | Result | Started | Completed |
|-----|--------|---------|-----------|
| Client (Swift 6 / macOS) — Build + Test | **success** | 06:10:48 | 06:11:12 |
| Gateway (Go) — Build + Test + Vet | **success** | 06:10:47 | 06:12:01 |
| Orchestrator (Go) — Build + Test + Vet | **success** | 06:10:47 | 06:12:06 |
| Docker — Build images | **success** | 06:10:47 | 06:11:59 |

### CD Pipeline

| Field | Value |
|-------|-------|
| File | `.github/workflows/cd.yml` |
| Trigger | `workflow_dispatch` (manual) |
| Targets | Cloud Run: realtime-gateway + adk-orchestrator |
| Post-Deploy | E2E smoke test against deployed services |

---

## 4. Test Coverage

### Backend — Realtime Gateway (Go)

| Package | Coverage |
|---------|----------|
| auth | 84.8% |
| adk | 69.0% |
| live | 53.7% |
| ws | 9.2% |

### Backend — ADK Orchestrator (Go)

| Package | Coverage |
|---------|----------|
| topic | 100.0% |
| prompts | 91.7% |
| engagement | 84.0% |
| celebration | 78.9% |
| graph | 67.9% |
| mood | 64.4% |
| mediator | 44.8% |
| search | 34.7% |
| vision | 33.3% |
| scheduler | 32.0% |
| memory | 30.2% |
| store | 13.8% |

### macOS Client (Swift)

| Test Suite | Tests | Status |
|------------|-------|--------|
| ImageDifferTests | 4 | **pass** |
| ImageProcessorTests | 4 | **pass** |
| PCMConverterTests | 4 | **pass** |
| ModelsTests | 4 | **pass** |
| AudioMessageParserTests | 4 | **pass** |
| SettingsTests | 4 | **pass** |
| **Total** | **24** | **all pass** |

### E2E Tests

| Test | Description | Status |
|------|-------------|--------|
| TestHealthCheck | Gateway `/readyz` returns 200 | **pass** |
| TestJWTRegistration | `POST /api/v1/register` returns JWT | **pass** |
| TestTokenRefresh | `POST /api/v1/refresh` with valid token | **pass** |
| TestWebSocketUpgrade | WS upgrade with auth header | **pass** |
| TestGeminiSetup | Gemini Live session initialization | **pass** |
| TestAuthRejection | No-auth request → 401 | **pass** |
| TestInvalidTokenRejection | Bad token → 401 | **pass** |

---

## 5. Architecture Verification

### Three-Layer Split (Challenge Requirement)

```
[macOS Client] ←WebSocket→ [Realtime Gateway] ←HTTP→ [ADK Orchestrator]
     Swift 6              Go + GenAI SDK          Go + ADK Go SDK
     Local                Cloud Run               Cloud Run
```

- Client NEVER calls Gemini API directly
- All model calls through backend Gateway
- API key stored in Secret Manager, never on client
- Gateway proxies to Gemini Live API via GenAI SDK
- Orchestrator runs 9-agent graph via ADK Go SDK

### Required Stack (All Present)

| Requirement | Implementation | Verified |
|-------------|---------------|----------|
| GenAI SDK | `google.golang.org/genai v1.48.0` in Gateway + Orchestrator | Yes |
| Google ADK | `google.golang.org/adk v0.5.0` in Orchestrator | Yes |
| Gemini Live API | Live session via GenAI SDK | Yes |
| VAD | `automaticActivityDetection: true` | Yes |

### 9 Agents (All Implemented)

| Agent | Package | Status |
|-------|---------|--------|
| VAD | Gemini Live API config | Implemented |
| VisionAgent | `agents/vision` | Implemented + tested |
| Mediator | `agents/mediator` | Implemented + tested + LLM dynamic messages |
| AdaptiveScheduler | `agents/scheduler` | Implemented + tested |
| EngagementAgent | `agents/engagement` | Implemented + tested + LLM dynamic messages |
| MemoryAgent | `agents/memory` | Implemented + tested |
| MoodDetector | `agents/mood` | Implemented + tested |
| CelebrationTrigger | `agents/celebration` | Implemented + tested + LLM dynamic messages |
| SearchBuddy | `agents/search` | Implemented + tested |
| LLMSearchBuddy | `agents/search` (llmagent) | Implemented (wave3 SubAgent) |

### Agent Graph (3-Wave Architecture)

```
runner.New(Config{
  Agent: sequentialagent → [
    parallelagent → [Vision, Memory]         ← Wave 1: Parallel perception
    parallelagent → [Mood, Celebration]       ← Wave 2: Parallel emotion
    sequentialagent → [Mediator, Scheduler, Engagement, SearchBuddy, LLMSearchBuddy]  ← Wave 3: Sequential decision
  ],
  SessionService: session.InMemoryService(),
  MemoryService: memory.InMemoryService(),
})
```

### ADK Features Used (11)

| # | Feature | Import |
|---|---------|--------|
| 1 | `agent.New()` | `google.golang.org/adk/agent` |
| 2 | `sequentialagent.New()` | `.../workflowagents/sequentialagent` |
| 3 | `parallelagent.New()` | `.../workflowagents/parallelagent` |
| 4 | `llmagent.New()` | `google.golang.org/adk/agent/llmagent` |
| 5 | `session.InMemoryService()` | `google.golang.org/adk/session` |
| 6 | `memory.InMemoryService()` | `google.golang.org/adk/memory` |
| 7 | `runner.New()` | `google.golang.org/adk/runner` |
| 8 | `telemetry.New()` | `google.golang.org/adk/telemetry` |
| 9 | `session.State/Event` | `google.golang.org/adk/session` |
| 10 | `functiontool.New()` | `google.golang.org/adk/tool/functiontool` |
| 11 | `geminitool.GoogleSearch{}` | `google.golang.org/adk/tool/geminitool` |

### AI Models Used

| Model | Purpose | Location |
|-------|---------|----------|
| `gemini-2.5-flash-native-audio-latest` | Live API (voice conversation) | Gateway |
| `gemini-2.5-flash-preview-tts` | Text-to-speech | Gateway |
| `gemini-3.1-flash-lite-preview` | Vision analysis | Orchestrator |
| `gemini-3.1-flash-lite-preview` | LLM Search agent | Orchestrator |
| `gemini-3.1-flash-lite-preview` | Dynamic message generation (Mediator, Celebration, Engagement) | Orchestrator |

---

## 6. Blog Posts (dev.to)

| # | Title | Status |
|---|-------|--------|
| 1 | "Why I Killed My App Two Weeks Before the Deadline" | **Published** |
| 2 | "The Empty Chair Problem" | **Published** |
| 3 | "Teaching Nine Agents to Think Like a Colleague" | **Published** |
| 4 | "The WebSocket Proxy Nobody Asked For" | **Published** |
| 5 | "Six Characters, One Soul Format" | **Published** |
| P2 | Additional posts | Draft (user-managed) |

All posts include `#GeminiLiveAgentChallenge` tag for +0.6 bonus points.
Blog publishing is user-managed.

---

## 7. Client Configuration

| Setting | Default Value |
|---------|--------------|
| Gateway URL | `wss://realtime-gateway-163070481841.asia-northeast3.run.app` |
| Language | `ko` |
| Voice | `Zephyr` |
| Character | `cat` |
| Chattiness | `normal` |
| Capture Interval | 5.0s |
| Live Model | `gemini-2.5-flash-native-audio-latest` |

---

## 8. Live API Features

| Feature | Status |
|---------|--------|
| VAD (Voice Activity Detection) | ✅ Active |
| Barge-in (StartOfActivityInterrupts) | ✅ Active |
| ProactiveAudio | ✅ Active |
| OutputAudioTranscription | ✅ Active |
| InputAudioTranscription | ✅ Active |
| ContextWindowCompression | ✅ Active (trigger 4096, target 2048) |
| SessionResumption | ✅ Active (handle-based) |
| AffectiveDialog | ✅ Active |

---

## Verification Commands

```bash
# Health check
curl https://realtime-gateway-163070481841.asia-northeast3.run.app/health
curl https://realtime-gateway-163070481841.asia-northeast3.run.app/readyz

# Cloud Run status
gcloud run services list --project=vibecat-489105 --region=asia-northeast3

# Run all tests
make test                    # Swift (24 tests)
make backend-test            # Go (Gateway + Orchestrator)

# Build & run client
make run

# Local backend development
source .env && cd backend/adk-orchestrator && PORT=9091 GEMINI_API_KEY=$GEMINI_API_KEY go run .
source .env && cd backend/realtime-gateway && PORT=9090 GEMINI_API_KEY=$GEMINI_API_KEY ADK_ORCHESTRATOR_URL=http://localhost:9091 go run .
```
