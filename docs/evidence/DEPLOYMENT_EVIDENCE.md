# VibeCat Deployment Evidence

**Collected**: 2026-03-04
**Project**: vibecat-489105
**Region**: asia-northeast3 (Seoul)

---

## 1. Cloud Run Services

### Realtime Gateway

| Field | Value |
|-------|-------|
| Service Name | `realtime-gateway` |
| URL | `https://realtime-gateway-a4akw2crra-du.a.run.app` |
| Alt URL | `https://realtime-gateway-163070481841.asia-northeast3.run.app` |
| Status | **Ready** (True) |
| Active Revision | `realtime-gateway-00002-4sw` |
| Traffic | 100% → latest revision |
| Created | 2026-03-04T00:28:46Z |
| Last Deploy | 2026-03-04T00:32:43Z |

**Health Check**:
```
GET /readyz → 200 OK
{"service":"realtime-gateway","status":"ok"}
```

### ADK Orchestrator

| Field | Value |
|-------|-------|
| Service Name | `adk-orchestrator` |
| URL | `https://adk-orchestrator-a4akw2crra-du.a.run.app` |
| Status | **Ready** (True) |
| Active Revision | `adk-orchestrator-00001-mbt` |
| Traffic | 100% → latest revision |
| Created | 2026-03-04T00:28:33Z |
| Access | Internal only (invoked by Gateway via `POST /analyze`) |

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
| GenAI SDK | `google.golang.org/genai` in Gateway | Yes |
| Google ADK | `google.golang.org/adk` in Orchestrator | Yes |
| Gemini Live API | Live session via GenAI SDK | Yes |
| VAD | `automaticActivityDetection: true` | Yes |

### 9 Agents (All Implemented)

| Agent | Package | Status |
|-------|---------|--------|
| VAD | Gemini Live API config | Implemented |
| VisionAgent | `agents/vision` | Implemented + tested |
| Mediator | `agents/mediator` | Implemented + tested |
| AdaptiveScheduler | `agents/scheduler` | Implemented + tested |
| EngagementAgent | `agents/engagement` | Implemented + tested |
| MemoryAgent | `agents/memory` | Implemented + tested |
| MoodDetector | `agents/mood` | Implemented + tested |
| CelebrationTrigger | `agents/celebration` | Implemented + tested |
| SearchBuddy | `agents/search` | Implemented + tested |

---

## 6. Blog Posts (dev.to)

| # | Title | dev.to ID | Status | Score |
|---|-------|-----------|--------|-------|
| Pivot | "Why I Killed My App Two Weeks Before the Deadline" | — | **Published** | — |
| P1 | "The Empty Chair Problem" | 3307451 | Draft | 98/100 |
| P2 | "Teaching Nine Agents to Think Like a Colleague" | 3307456 | Draft | 97/100 |
| P3 | "The WebSocket Proxy Nobody Asked For" | 3307463 | Draft | 95/100 |
| P4 | "Six Characters, One Soul Format" | 3307467 | Draft | 99/100 |
| P5 | "Making Swift 6 Talk to Go" | 3307469 | Draft | 96/100 |
| P6 | "The Cloud Run 404 That Almost Ended Everything" | 3307471 | Draft | 95/100 |
| P7 | "Retrospective: What VibeCat Taught Me" | 3307474 | Draft | 96/100 |

All posts include `#GeminiLiveAgentChallenge` tag for +0.6 bonus points.

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
| Live Model | `gemini-2.0-flash-live-001` |

---

## Verification Commands

```bash
# Health check
curl https://realtime-gateway-a4akw2crra-du.a.run.app/readyz

# Cloud Run status
gcloud run services list --project=vibecat-489105 --region=asia-northeast3

# Run all tests
make test                    # Swift (24 tests)
make backend-test            # Go (Gateway + Orchestrator)
GATEWAY_URL=https://realtime-gateway-a4akw2crra-du.a.run.app go test ./...  # E2E (in tests/e2e/)

# CI status
gh run list --repo Two-Weeks-Team/vibeCat --limit 3

# Build & run client
make run
```
