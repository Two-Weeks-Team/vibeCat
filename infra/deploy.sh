#!/usr/bin/env bash
set -euo pipefail

# VibeCat Cloud Run Deployment
# Deploys both backend services to GCP Cloud Run (asia-northeast3)
#
# Prerequisites:
#   - Run ./infra/setup.sh first (one-time GCP project bootstrap)
#   - gcloud CLI authenticated (gcloud auth login)
#   - Gemini API key stored in Secret Manager

PROJECT_ID="${GCP_PROJECT:-vibecat-489105}"
REGION="${GCP_REGION:-asia-northeast3}"
REGISTRY="${REGION}-docker.pkg.dev/${PROJECT_ID}/vibecat-images"
GATEWAY_IMAGE="${REGISTRY}/realtime-gateway"
ORCHESTRATOR_IMAGE="${REGISTRY}/adk-orchestrator"

echo "=== VibeCat Deployment ==="
echo "Project:  ${PROJECT_ID}"
echo "Region:   ${REGION}"
echo ""

# --- Build ---
echo "[1/4] Building realtime-gateway..."
gcloud builds submit backend/realtime-gateway/ \
  --tag "${GATEWAY_IMAGE}" \
  --project "${PROJECT_ID}" \
  --quiet

echo "[2/4] Building adk-orchestrator..."
gcloud builds submit backend/adk-orchestrator/ \
  --tag "${ORCHESTRATOR_IMAGE}" \
  --project "${PROJECT_ID}" \
  --quiet

# --- Deploy ---
echo "[3/5] Deploying adk-orchestrator..."
gcloud run deploy adk-orchestrator \
  --image "${ORCHESTRATOR_IMAGE}" \
  --region "${REGION}" \
  --project "${PROJECT_ID}" \
  --port 8080 \
  --memory 1Gi \
  --cpu 1 \
  --min-instances 0 \
  --max-instances 3 \
  --no-allow-unauthenticated \
  --set-secrets "GEMINI_API_KEY=vibecat-gemini-api-key:latest" \
  --quiet

ORCHESTRATOR_URL=$(gcloud run services describe adk-orchestrator \
  --region "${REGION}" \
  --project "${PROJECT_ID}" \
  --format "value(status.url)")

echo "[4/5] Deploying realtime-gateway..."
gcloud run deploy realtime-gateway \
  --image "${GATEWAY_IMAGE}" \
  --region "${REGION}" \
  --project "${PROJECT_ID}" \
  --port 8080 \
  --memory 512Mi \
  --cpu 1 \
  --min-instances 0 \
  --max-instances 3 \
  --allow-unauthenticated \
  --set-secrets "GEMINI_API_KEY=vibecat-gemini-api-key:latest,AUTH_SECRET=vibecat-gateway-auth-secret:latest" \
  --set-env-vars "ORCHESTRATOR_URL=${ORCHESTRATOR_URL}" \
  --quiet

GATEWAY_URL=$(gcloud run services describe realtime-gateway \
  --region "${REGION}" \
  --project "${PROJECT_ID}" \
  --format "value(status.url)")

echo "[5/5] Granting gateway → orchestrator invocation..."
COMPUTE_SA=$(gcloud iam service-accounts list \
  --project="${PROJECT_ID}" \
  --filter="email:compute@developer.gserviceaccount.com" \
  --format="value(email)")
gcloud run services add-iam-policy-binding adk-orchestrator \
  --member="serviceAccount:${COMPUTE_SA}" \
  --role="roles/run.invoker" \
  --region="${REGION}" \
  --project="${PROJECT_ID}" \
  --quiet

echo ""
echo "=== Deployment Complete ==="
echo "Gateway:      ${GATEWAY_URL}"
echo "Orchestrator: ${ORCHESTRATOR_URL} (internal only)"
