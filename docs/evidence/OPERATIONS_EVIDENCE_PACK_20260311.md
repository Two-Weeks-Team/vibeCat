# VibeCat Operations Evidence Pack (2026-03-11)

This document is the submission-oriented index for deployment, observability, and runtime evidence gathered on 2026-03-11.

## Primary Documents

- current snapshot: `docs/CURRENT_STATUS_20260311.md`
- deployment evidence: `docs/evidence/DEPLOYMENT_EVIDENCE.md`
- concise proof view: `docs/deployment/PROOF_OF_GCP_DEPLOYMENT.md`
- issue baseline: `docs/ISSUE_TRIAGE_20260311.md`

## Collected Live Artifacts

All CLI-exported live artifacts are stored under `docs/evidence/artifacts/20260311/`.

| Artifact | Purpose |
|----------|---------|
| `realtime-gateway-service.json` | Cloud Run service configuration and latest ready revision for the public gateway |
| `adk-orchestrator-service.json` | Cloud Run service configuration and latest ready revision for the authenticated orchestrator |
| `realtime-gateway-health.json` | Public `/health` response from the deployed gateway |
| `adk-orchestrator-health.json` | Authenticated `/health` response from the deployed orchestrator |
| `monitoring-dashboard-vibecat-operations-overview.json` | Live Cloud Monitoring dashboard export for `VibeCat Operations Overview` |
| `monitoring-dashboard-vibecat-runtime-overview.json` | Live Cloud Monitoring dashboard export for `VibeCat Runtime Overview` |

## Observability Assets

- dashboard definition in repo: `infra/monitoring/vibecat-operations-dashboard.yaml`
- dashboard apply/update script: `infra/configure_observability.sh`
- live dashboard display name: `VibeCat Operations Overview`
- live dashboard resource: `projects/163070481841/dashboards/752c1986-5674-4963-a967-6f3595902be2`
- custom-metrics definition in repo: `infra/observability/vibecat-runtime-dashboard.json`
- custom-metrics apply/update script: `infra/observability/sync_dashboard.sh`
- custom-metrics dashboard display name: `VibeCat Runtime Overview`
- custom-metrics dashboard resource: `projects/163070481841/dashboards/0425463d-47b5-4235-a2e9-f7c9002ba2f6`

## Trace Verification Path

End-to-end Gateway-to-Orchestrator trace propagation is now wired in code via:

- `backend/realtime-gateway/internal/adk/client.go`
- `backend/realtime-gateway/internal/ws/handler.go`
- `backend/realtime-gateway/main.go`
- `backend/adk-orchestrator/main.go`

Verification coverage added in this branch:

- request-header injection test: `backend/realtime-gateway/internal/adk/client_test.go`
- companion-intelligence integration path: `tests/e2e/companion_intelligence_test.go`

The dashboard is live now. The new trace propagation and fallback behavior become live after the updated backend revisions are deployed from this branch.

## Screenshot Checklist

If screenshot-grade console evidence is required by the submission surface, capture these pages after backend deploy:

1. Cloud Run service details for `realtime-gateway`
2. Cloud Run service details for `adk-orchestrator`
3. Cloud Monitoring dashboard `VibeCat Operations Overview`
4. Cloud Monitoring dashboard `VibeCat Runtime Overview`
5. Cloud Logging query showing both services
6. Cloud Trace explorer showing a trace with both Gateway and Orchestrator spans

Recommended filenames:

- `cloud-run-realtime-gateway.png`
- `cloud-run-adk-orchestrator.png`
- `cloud-monitoring-vibecat-operations-overview.png`
- `cloud-monitoring-vibecat-runtime-overview.png`
- `cloud-logging-services.png`
- `cloud-trace-gateway-orchestrator.png`

## Re-run Commands

```bash
./infra/configure_observability.sh
./infra/observability/sync_dashboard.sh
gcloud monitoring dashboards describe projects/163070481841/dashboards/752c1986-5674-4963-a967-6f3595902be2 --project vibecat-489105
gcloud monitoring dashboards describe projects/163070481841/dashboards/0425463d-47b5-4235-a2e9-f7c9002ba2f6 --project vibecat-489105
gcloud run services describe realtime-gateway --region asia-northeast3 --project vibecat-489105
gcloud run services describe adk-orchestrator --region asia-northeast3 --project vibecat-489105
```
