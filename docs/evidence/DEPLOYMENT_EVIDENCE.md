# VibeCat Deployment Evidence

**Collected:** 2026-03-11
**Project:** `vibecat-489105`
**Region:** `asia-northeast3`

This document reflects the live deployment and CI state verified on 2026-03-11. It replaces older evidence that still referenced pre-`00040`/`00038` revisions or older CI runs.

Submission-oriented cross-links:

- evidence pack index: `docs/evidence/OPERATIONS_EVIDENCE_PACK_20260311.md`
- concise proof view: `docs/deployment/PROOF_OF_GCP_DEPLOYMENT.md`
- exported live artifacts: `docs/evidence/artifacts/20260311/`

## 1. Cloud Run Services

### Realtime Gateway

| Field | Value |
|-------|-------|
| Service | `realtime-gateway` |
| Ready Revision | `realtime-gateway-00040-gcd` |
| Traffic | `100%` to latest revision |
| Canonical URL | `https://realtime-gateway-a4akw2crra-du.a.run.app` |
| Legacy URL Alias | `https://realtime-gateway-163070481841.asia-northeast3.run.app` |
| Ingress | `all` |
| Access | public |
| Service Account | `163070481841-compute@developer.gserviceaccount.com` |

Verified responses:

```text
GET /health  -> 200 {"connections":0,"service":"realtime-gateway","status":"ok"}
GET /readyz  -> 200 {"service":"realtime-gateway","status":"ok"}
```

### ADK Orchestrator

| Field | Value |
|-------|-------|
| Service | `adk-orchestrator` |
| Ready Revision | `adk-orchestrator-00038-t4c` |
| Traffic | `100%` to latest revision |
| Canonical URL | `https://adk-orchestrator-a4akw2crra-du.a.run.app` |
| Legacy URL Alias | `https://adk-orchestrator-163070481841.asia-northeast3.run.app` |
| Ingress | `all` |
| Access | authenticated invocation required |
| Service Account | `163070481841-compute@developer.gserviceaccount.com` |

Verified responses:

```text
Anonymous GET /health         -> 403 Forbidden
Authenticated GET /health     -> {"service":"adk-orchestrator","status":"ok"}
```

The orchestrator is reachable from Cloud Run but not intended as a public health endpoint.

## 2. GCP Infrastructure

### Firestore

| Field | Value |
|-------|-------|
| Database | `(default)` |
| Type | `FIRESTORE_NATIVE` |
| Location | `asia-northeast3` |
| Concurrency | `PESSIMISTIC` |

### Secret Manager

| Secret | Replication |
|--------|-------------|
| `vibecat-gemini-api-key` | user-managed, `asia-northeast3` |
| `vibecat-gateway-auth-secret` | user-managed, `asia-northeast3` |

### Artifact Registry

| Field | Value |
|-------|-------|
| Repository | `vibecat-images` |
| Format | `DOCKER` |
| Location | `asia-northeast3` |
| Description | `VibeCat backend container images` |

## 3. Observability

### Cloud Logging

Observed on 2026-03-11:

```text
2026-03-11T02:36:18Z realtime-gateway      realtime-gateway-00040-gcd      INFO
2026-03-11T02:36:25Z adk-orchestrator      adk-orchestrator-00038-t4c      INFO
```

Both services are emitting Cloud Run logs.

### Cloud Trace

Recent trace IDs were returned by the Cloud Trace API on 2026-03-11:

```text
007dec0efd190738d0b27d8df33ca788
00c2bb5cb274501fbd3a0b8edb7de5e4
028b73af0956f4bd59081485878a5a33
```

Trace export is active. This branch also wires explicit Gateway-to-Orchestrator OpenTelemetry propagation in:

- `backend/realtime-gateway/internal/adk/client.go`
- `backend/realtime-gateway/main.go`
- `backend/adk-orchestrator/main.go`

That propagation becomes visible in Cloud Trace after the updated backend revisions are deployed.

### Cloud Monitoring

| Item | Status |
|------|--------|
| metric exporter in code | present |
| dashboard in project | `VibeCat Operations Overview` |
| dashboard resource | `projects/163070481841/dashboards/752c1986-5674-4963-a967-6f3595902be2` |
| dashboard definition in repo | `infra/monitoring/vibecat-operations-dashboard.yaml` |
| dashboard apply script | `infra/configure_observability.sh` |
| custom-metrics dashboard in project | `VibeCat Runtime Overview` |
| custom-metrics dashboard resource | `projects/163070481841/dashboards/0425463d-47b5-4235-a2e9-f7c9002ba2f6` |
| custom-metrics repo definition | `infra/observability/vibecat-runtime-dashboard.json` |
| custom-metrics apply script | `infra/observability/sync_dashboard.sh` |

Cloud Monitoring now has both a Cloud Run operations dashboard and a gateway custom-metrics dashboard. Live dashboard exports are stored in:

- `docs/evidence/artifacts/20260311/monitoring-dashboard-vibecat-operations-overview.json`
- `docs/evidence/artifacts/20260311/monitoring-dashboard-vibecat-runtime-overview.json`

## 4. CI/CD Baseline

### GitHub Actions CI

| Run | Commit | Result | Notes |
|-----|--------|--------|-------|
| `22932978716` | `ac5e4bf` | success | latest fully green run |
| `22933714954` | `7356f8c` | failure | only Swift self-hosted runner step failed |

Latest `master` CI job status on `22933714954`:

| Job | Result |
|-----|--------|
| Gateway (Go) — Build + Test + Vet | success |
| Orchestrator (Go) — Build + Test + Vet | success |
| Docker — Build images | success |
| Client (Swift 6 / macOS) — Build + Test | failure |

Swift failure details:

```text
Step: Select Xcode 16+ and accept license
Cause: No licensed Xcode installation is available on this runner.
```

### Deployment Workflow

| Item | State |
|------|-------|
| GitHub CD workflow | present (`.github/workflows/cd.yml`) |
| Deploy trigger | manual `workflow_dispatch` |
| Cloud Build YAML | present for both backend services |
| Cloud Build triggers in GCP | none configured |

## 5. Project Progress Summary

The deployed system is no longer scaffolding. The repository contains a working Swift client, a deployed Realtime Gateway, a deployed ADK Orchestrator, tests, and deployment automation.

What still remains is not foundational implementation but release hardening:

- ops evidence pack completion
- privacy controls UI
- trace verification after deploying the updated backend revisions
- full companion-intelligence E2E proof
- degraded-mode fallback behavior for Gemini and ADK failures

## 6. Meaningful Remaining Issues

The issues that still map cleanly to real unfinished work are:

- `#57` deployment and operations evidence pack
- `#64` privacy controls UI
- `#90` Cloud Logging and Monitoring completion
- `#91` Cloud Trace completion
- `#117` full companion intelligence integration test
- `#120` Gemini unavailable fallback
- `#121` ADK timeout fallback

The other still-open issues are either consolidatable into `#57`, optional UX, or already superseded by the current deployment/tooling reality.
