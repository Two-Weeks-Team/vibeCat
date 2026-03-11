# Proof of GCP Deployment

**Project ID**: `vibecat-489105`
**Region**: `asia-northeast3`
**Verification Date**: 2026-03-11

This file is the concise deployment-proof view for final handoff and submission packaging.

## Cloud Run Services

### Realtime Gateway

- service: `realtime-gateway`
- ready revision: `realtime-gateway-00040-gcd`
- URL: `https://realtime-gateway-a4akw2crra-du.a.run.app`
- legacy alias: `https://realtime-gateway-163070481841.asia-northeast3.run.app`
- traffic: `100%`
- access: public
- health:
  - `GET /health` -> `200 {"connections":0,"service":"realtime-gateway","status":"ok"}`
  - `GET /readyz` -> `200 {"service":"realtime-gateway","status":"ok"}`

### ADK Orchestrator

- service: `adk-orchestrator`
- ready revision: `adk-orchestrator-00038-t4c`
- URL: `https://adk-orchestrator-a4akw2crra-du.a.run.app`
- legacy alias: `https://adk-orchestrator-163070481841.asia-northeast3.run.app`
- traffic: `100%`
- access: authenticated invocation required
- health:
  - anonymous `GET /health` -> `403 Forbidden`
  - authenticated `GET /health` with identity token -> `{"service":"adk-orchestrator","status":"ok"}`

## Supporting GCP Resources

### Firestore

- database: `(default)`
- type: `FIRESTORE_NATIVE`
- location: `asia-northeast3`

### Secret Manager

- `vibecat-gemini-api-key`
- `vibecat-gateway-auth-secret`

Both are user-managed and replicated in `asia-northeast3`.

### Artifact Registry

- repository: `vibecat-images`
- path: `asia-northeast3-docker.pkg.dev/vibecat-489105/vibecat-images`
- format: `DOCKER`

## Observability Proof

### Cloud Logging

Recent Cloud Run logs were observed from both services on 2026-03-11.

### Cloud Trace

Recent trace IDs observed on 2026-03-11:

```text
007dec0efd190738d0b27d8df33ca788
00c2bb5cb274501fbd3a0b8edb7de5e4
028b73af0956f4bd59081485878a5a33
```

### Cloud Monitoring

- OpenTelemetry metric exporter is wired in code
- no Monitoring dashboard is configured yet

This means observability is partially proven, but dashboard-level completion still belongs to the remaining ops work.

## CI/CD Proof

### CI

- latest fully green CI run: `22932978716` on commit `ac5e4bf`
- latest `master` CI run: `22933714954` on commit `7356f8c`

Current `master` CI results:

- Gateway Go job: pass
- Orchestrator Go job: pass
- Docker build job: pass
- Swift macOS job: fail due self-hosted runner Xcode license state

### CD

- GitHub manual deployment workflow exists at `.github/workflows/cd.yml`
- Cloud Build YAML exists for both backend services
- Cloud Build triggers are not configured in GCP

## Submission Notes

For final submission packaging, the code and live deployment are already in place. What still needs to be attached manually, if required by the submission surface, is screenshot-grade evidence from:

- Cloud Run service detail pages
- Firestore database page
- Cloud Logging viewer
- Cloud Trace explorer

Do not rely on older screenshot placeholders; verify them against the current `00040` and `00038` revisions.
