# Deployment Evidence

**Last verified:** 2026-03-14T17:08 KST
**Submission category:** UI Navigator

## Deployed Services (Live)

| Service | URL | Status |
|---------|-----|--------|
| **Realtime Gateway** | `https://realtime-gateway-163070481841.asia-northeast3.run.app` | ✅ 200 OK |
| **ADK Orchestrator** | `https://adk-orchestrator-163070481841.asia-northeast3.run.app` | ✅ Deployed (auth-only) |

- **GCP Project:** `vibecat-489105`
- **Region:** `asia-northeast3`
- **Firestore:** `(default)` native database

### Live Health Check (2026-03-14T08:08:36Z)

```json
{
    "connections": 1,
    "service": "realtime-gateway",
    "status": "ok"
}
```

## Proof of Google Cloud Deployment

### Option 1: Code Files Demonstrating GCP Usage

| File | Google Cloud Service | Purpose |
|------|---------------------|---------|
| [`infra/deploy.sh`](../../infra/deploy.sh) | Cloud Run, Cloud Build, IAM, Secret Manager | Automated deployment script |
| [`backend/realtime-gateway/Dockerfile`](../../backend/realtime-gateway/Dockerfile) | Cloud Run (distroless container) | Gateway container image |
| [`backend/adk-orchestrator/Dockerfile`](../../backend/adk-orchestrator/Dockerfile) | Cloud Run (distroless container) | Orchestrator container image |
| [`backend/realtime-gateway/internal/ws/action_state_store.go`](../../backend/realtime-gateway/internal/ws/action_state_store.go) | Firestore | Session state persistence |
| [`backend/realtime-gateway/internal/live/session.go`](../../backend/realtime-gateway/internal/live/session.go) | Gemini Live API (GenAI SDK) | Real-time multimodal AI conversation |
| [`backend/adk-orchestrator/internal/navigator/processor.go`](../../backend/adk-orchestrator/internal/navigator/processor.go) | ADK + GenAI SDK | Vision analysis, confidence escalation |
| [`backend/realtime-gateway/main.go`](../../backend/realtime-gateway/main.go) | Cloud Logging, Cloud Trace (OTEL) | Observability |
| [`infra/setup.sh`](../../infra/setup.sh) | APIs, Artifact Registry, Firestore, Secret Manager, IAM | One-time GCP project bootstrap |

### Google Cloud Services Used

| Service | Purpose | Evidence |
|---------|---------|----------|
| **Cloud Run** | Serverless hosting for gateway + orchestrator | `infra/deploy.sh` lines 38-71, Dockerfile |
| **Gemini Live API** | Real-time voice + vision AI conversation | `internal/live/session.go`, GenAI SDK v1.49.0 |
| **ADK (Agent Development Kit)** | Screenshot analysis, confidence escalation | `backend/adk-orchestrator/`, ADK v0.6.0 |
| **Firestore** | Action state persistence, session memory, replay | `action_state_store.go`, orchestrator replay |
| **Secret Manager** | API keys, auth secrets | `deploy.sh` --set-secrets flags |
| **Cloud Build** | Container image builds | `deploy.sh` gcloud builds submit |
| **Cloud Logging** | Structured logging, navigator metrics | `main.go` OTEL setup |
| **Cloud Trace** | Distributed tracing | `main.go` OTEL setup |
| **Artifact Registry** | Container image storage | `deploy.sh` registry config |

### Deployment Architecture

```
infra/deploy.sh execution flow:
  1. gcloud builds submit → Cloud Build → Artifact Registry
  2. gcloud run deploy adk-orchestrator → Cloud Run (internal, auth-only)
  3. gcloud run deploy realtime-gateway → Cloud Run (public, WebSocket)
  4. IAM binding: gateway service account → orchestrator invoker role
```

### Automated Deployment Script

`infra/deploy.sh` automates the full deployment pipeline:
- Builds Docker images via Cloud Build
- Deploys orchestrator first (internal, no unauthenticated access)
- Deploys gateway second (public, with orchestrator URL injected)
- Configures IAM for gateway-to-orchestrator invocation

This qualifies for the **Automated Cloud Deployment** bonus (+0.2 points).
