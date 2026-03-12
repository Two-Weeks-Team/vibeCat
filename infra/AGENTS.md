# Infra Guide

## OVERVIEW

`infra/` holds the GCP bootstrap, deploy, teardown, observability sync, and dashboard assets for the Cloud Run backend stack.

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| First-time project bootstrap | `infra/setup.sh` | APIs, Artifact Registry, Firestore, secrets, IAM |
| Service deployment | `infra/deploy.sh` | build images, deploy orchestrator then gateway |
| Cleanup | `infra/teardown.sh` | project resource teardown |
| Observability setup | `infra/configure_observability.sh` | logging/monitoring wiring |
| Dashboard sync | `infra/observability/sync_dashboard.sh` | dashboard asset push |
| Observability dashboard assets | `infra/monitoring/` and `infra/observability/` | Cloud Monitoring YAML/JSON dashboard definitions |

## CONVENTIONS

- Project defaults are `vibecat-489105` and `asia-northeast3` unless explicitly overridden.
- Deploy orchestrator first, then gateway with `ADK_ORCHESTRATOR_URL` injected from Cloud Run.
- Secret names are fixed: `vibecat-gemini-api-key` and `vibecat-gateway-auth-secret`.
- Backend production port stays `8080`.

## ANTI-PATTERNS

- deploying gateway without resolving the current orchestrator URL
- casually renaming services, secret ids, region, or Artifact Registry paths
- moving credentials into repo files instead of Secret Manager
- editing dashboard assets without checking the matching sync/apply script

## COMMANDS

```bash
./infra/setup.sh
./infra/deploy.sh
./infra/configure_observability.sh
./infra/observability/sync_dashboard.sh
```
