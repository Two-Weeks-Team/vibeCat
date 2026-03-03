#!/usr/bin/env bash
set -euo pipefail

# VibeCat Cloud Run Teardown
# Removes both backend services from GCP Cloud Run

PROJECT_ID="${GCP_PROJECT:-vibecat-489105}"
REGION="${GCP_REGION:-asia-northeast3}"

echo "=== VibeCat Teardown ==="
echo "Project:  ${PROJECT_ID}"
echo "Region:   ${REGION}"
echo ""

read -rp "This will delete both Cloud Run services. Continue? [y/N] " confirm
if [[ "${confirm}" != "y" && "${confirm}" != "Y" ]]; then
  echo "Aborted."
  exit 0
fi

echo "[1/2] Deleting realtime-gateway..."
gcloud run services delete realtime-gateway \
  --region "${REGION}" \
  --project "${PROJECT_ID}" \
  --quiet 2>/dev/null || echo "  (not found or already deleted)"

echo "[2/2] Deleting adk-orchestrator..."
gcloud run services delete adk-orchestrator \
  --region "${REGION}" \
  --project "${PROJECT_ID}" \
  --quiet 2>/dev/null || echo "  (not found or already deleted)"

echo ""
echo "=== Teardown Complete ==="
echo "Note: Container images in gcr.io/${PROJECT_ID}/ are NOT deleted."
echo "      Delete manually if needed: gcloud container images list --repository=gcr.io/${PROJECT_ID}"
