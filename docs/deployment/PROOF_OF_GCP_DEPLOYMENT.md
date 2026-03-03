# Proof of GCP Deployment

**Project ID**: `vibecat-489105`
**Region**: `asia-northeast3`
**Date**: [FILL ON DEPLOYMENT]

## Cloud Run Services

### Realtime Gateway
- Service URL: `https://realtime-gateway-XXXXX-an.a.run.app`
- Health: `GET /healthz` → 200
- Screenshot: [ATTACH]

### ADK Orchestrator
- Service URL: `https://adk-orchestrator-XXXXX-an.a.run.app`
- Health: `GET /healthz` → 200
- Screenshot: [ATTACH]

## Firestore
- Location: `asia-northeast3`
- Collections: `sessions`, `users`
- Screenshot: [ATTACH]

## Secret Manager
- `vibecat-gemini-api-key`: Active
- `vibecat-gateway-auth-secret`: Active
- Screenshot: [ATTACH]

## Observability
- Cloud Logging: [ATTACH structured log screenshot]
- Cloud Monitoring: [ATTACH dashboard screenshot]
- Cloud Trace: [ATTACH Client→Gateway→Orchestrator→Gemini span]
