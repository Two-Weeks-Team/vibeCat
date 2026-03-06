# Proof of GCP Deployment

**Project ID**: `vibecat-489105`
**Region**: `asia-northeast3`
**Date**: 2026-03-06

## Cloud Run Services

### Realtime Gateway
- Service URL: `https://realtime-gateway-163070481841.asia-northeast3.run.app`
- Health: `GET /health` → 200 `{"connections":0,"service":"realtime-gateway","status":"ok"}`
- Ready: `GET /readyz` → 200 `{"service":"realtime-gateway","status":"ok"}`
- Revision: `realtime-gateway-00006-mq9`
- Auth: Public (`allUsers` → `roles/run.invoker`)
- Screenshot: [ATTACH]

### ADK Orchestrator
- Service URL: `https://adk-orchestrator-163070481841.asia-northeast3.run.app`
- Health: `GET /health` → 200
- Revision: `adk-orchestrator-00008-5t2`
- Auth: Internal only (service account → `roles/run.invoker`)
- Screenshot: [ATTACH]

## Cloud Build
- Gateway build: `eb802b72` → SUCCESS (1m46s)
- Orchestrator build: `6e335282` → SUCCESS (2m22s)
- Artifact Registry: `asia-northeast3-docker.pkg.dev/vibecat-489105/vibecat-images/`

## Firestore
- Location: `asia-northeast3`
- Collections: `sessions`, `users`
- Screenshot: [ATTACH]

## Secret Manager
- `vibecat-gemini-api-key`: Active (1 version)
- `vibecat-gateway-auth-secret`: Active (1 version)
- Region: `asia-northeast3` (user-managed)
- Screenshot: [ATTACH]

## Observability
- Cloud Trace: OpenTelemetry spans via `opentelemetry-operations-go/exporter/trace`
  - Gateway: `gateway.ws.handle`, `adk.analyze` spans
  - Orchestrator: `orchestrator.analyze` span
- Cloud Logging: Structured JSON via `cloud.google.com/go/logging`
- ADK Telemetry: `google.golang.org/adk/telemetry` with GCP resource project
- Screenshot: [ATTACH Cloud Trace Explorer showing spans]

## IAM Roles (Compute Service Account)
- `roles/editor` (broad)
- `roles/cloudtrace.agent` (trace write)
- `roles/datastore.user` (Firestore access)

## Agent Architecture (Current)

### ADK Features Used (11)
`agent.New()`, `sequentialagent.New()`, `parallelagent.New()`, `llmagent.New()`, `session.InMemoryService()`, `memory.InMemoryService()`, `runner.New()`, `telemetry.New()`, `session.State/Event`, `functiontool.New()`, `geminitool.GoogleSearch{}`

### Agent Graph (3-Wave)
```
wave1 (parallel): [VisionAgent, MemoryAgent]
wave2 (parallel): [MoodDetector, CelebrationTrigger]
wave3 (sequential): [Mediator, Scheduler, Engagement, SearchBuddy, LLMSearchBuddy]
```

### Dynamic Message Generation
All speech agents (Mediator, Celebration, Engagement) use `gemini-3.1-flash-lite-preview` for contextual LLM-generated messages. Hardcoded message pools serve as fallback only.

### Live API Stability
Gateway implements reconnection race condition protection:
- `reconnecting` state flag prevents duplicate reconnect attempts
- Conditional session nil with pointer comparison prevents killing newer sessions
- Audio frames dropped during reconnect to prevent stale data

## Known Issues (Resolved)
- `/healthz` path is intercepted by Cloud Run infrastructure → renamed to `/health`
- Cold start with min-instances=0 adds ~2-3s on first request → acceptable for demo
- Live API reconnection race condition → fixed with `reconnecting` state flag + pointer comparison
- Repetitive speech messages → fixed with LLM dynamic generation (no more hardcoded pools)
