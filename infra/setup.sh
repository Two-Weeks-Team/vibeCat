#!/usr/bin/env bash
set -euo pipefail

# VibeCat GCP Project Setup (One-Time)
# Bootstraps all GCP services needed for VibeCat backend.
# Run once per project. Idempotent — safe to re-run.
#
# Prerequisites:
#   - gcloud CLI installed and authenticated (gcloud auth login)
#   - Billing enabled on the GCP project
#
# Usage:
#   ./infra/setup.sh                         # Uses defaults
#   GCP_PROJECT=my-project ./infra/setup.sh  # Override project
#   GEMINI_API_KEY=abc123 ./infra/setup.sh   # Auto-store API key

PROJECT_ID="${GCP_PROJECT:-vibecat-489105}"
REGION="${GCP_REGION:-asia-northeast3}"
ACCOUNT="${GCP_ACCOUNT:-centisgood@gmail.com}"

echo "=== VibeCat GCP Setup ==="
echo "Project:  ${PROJECT_ID}"
echo "Region:   ${REGION}"
echo "Account:  ${ACCOUNT}"
echo ""

# ─── 1. Set project and account ───────────────────────────────────────────────

echo "[1/8] Setting active project and account..."
gcloud config set project "${PROJECT_ID}"
gcloud config set account "${ACCOUNT}"
gcloud config set run/region "${REGION}"

# ─── 2. Enable required APIs ──────────────────────────────────────────────────

echo "[2/8] Enabling APIs..."
gcloud services enable \
  run.googleapis.com \
  firestore.googleapis.com \
  secretmanager.googleapis.com \
  cloudbuild.googleapis.com \
  artifactregistry.googleapis.com \
  logging.googleapis.com \
  monitoring.googleapis.com \
  cloudtrace.googleapis.com \
  generativelanguage.googleapis.com \
  --project "${PROJECT_ID}"

echo "  APIs enabled."

# ─── 3. Artifact Registry ─────────────────────────────────────────────────────

echo "[3/8] Creating Artifact Registry repository..."
if gcloud artifacts repositories describe vibecat-images \
    --location="${REGION}" \
    --project="${PROJECT_ID}" &>/dev/null; then
  echo "  Repository 'vibecat-images' already exists."
else
  gcloud artifacts repositories create vibecat-images \
    --repository-format=docker \
    --location="${REGION}" \
    --project="${PROJECT_ID}" \
    --description="VibeCat backend container images"
  echo "  Repository 'vibecat-images' created."
fi

# ─── 4. Firestore ─────────────────────────────────────────────────────────────

echo "[4/8] Setting up Firestore..."
# Check if Firestore database already exists
if gcloud firestore databases describe \
    --project="${PROJECT_ID}" &>/dev/null; then
  echo "  Firestore (default) database already exists."
else
  gcloud firestore databases create \
    --location="${REGION}" \
    --project="${PROJECT_ID}" \
    --type=firestore-native
  echo "  Firestore (default) database created in ${REGION}."
fi

# ─── 5. Secret Manager ────────────────────────────────────────────────────────

echo "[5/8] Creating secrets in Secret Manager..."

create_secret() {
  local name="$1"
  local description="$2"
  if gcloud secrets describe "${name}" --project="${PROJECT_ID}" &>/dev/null; then
    echo "  Secret '${name}' already exists."
  else
    gcloud secrets create "${name}" \
      --replication-policy="user-managed" \
      --locations="${REGION}" \
      --project="${PROJECT_ID}" \
      --labels="app=vibecat"
    echo "  Secret '${name}' created. (${description})"
  fi
}

create_secret "vibecat-gemini-api-key" "Gemini API key for GenAI SDK"
create_secret "vibecat-gateway-auth-secret" "Client session token signing key"

# Auto-store Gemini API key if provided via env
if [[ -n "${GEMINI_API_KEY:-}" ]]; then
  echo -n "${GEMINI_API_KEY}" | gcloud secrets versions add vibecat-gemini-api-key \
    --data-file=- \
    --project="${PROJECT_ID}"
  echo "  Gemini API key stored in Secret Manager."
fi

# ─── 6. Service Accounts & IAM ────────────────────────────────────────────────

echo "[6/8] Configuring service accounts and IAM..."

# Get the project number for default compute SA
PROJECT_NUMBER=$(gcloud projects describe "${PROJECT_ID}" --format="value(projectNumber)")
COMPUTE_SA="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"

# Grant Cloud Run default SA access to secrets
for secret in vibecat-gemini-api-key vibecat-gateway-auth-secret; do
  gcloud secrets add-iam-policy-binding "${secret}" \
    --member="serviceAccount:${COMPUTE_SA}" \
    --role="roles/secretmanager.secretAccessor" \
    --project="${PROJECT_ID}" \
    --quiet
done
echo "  Compute SA can access secrets."

# Grant Cloud Run invoker role for gateway → orchestrator calls
# (Gateway's SA needs to invoke orchestrator)
gcloud run services add-iam-policy-binding adk-orchestrator \
  --member="serviceAccount:${COMPUTE_SA}" \
  --role="roles/run.invoker" \
  --region="${REGION}" \
  --project="${PROJECT_ID}" \
  --quiet 2>/dev/null || echo "  (orchestrator not deployed yet — IAM binding will apply on first deploy)"

# Grant Firestore access
gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
  --member="serviceAccount:${COMPUTE_SA}" \
  --role="roles/datastore.user" \
  --quiet
echo "  Compute SA has Firestore access."

# Grant Cloud Trace writer
gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
  --member="serviceAccount:${COMPUTE_SA}" \
  --role="roles/cloudtrace.agent" \
  --quiet
echo "  Compute SA has Cloud Trace access."

# ─── 7. Cloud Build permissions ───────────────────────────────────────────────

echo "[7/8] Configuring Cloud Build permissions..."
CLOUDBUILD_SA="${PROJECT_NUMBER}@cloudbuild.gserviceaccount.com"

# Cloud Build needs to push to Artifact Registry and deploy to Cloud Run
gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
  --member="serviceAccount:${CLOUDBUILD_SA}" \
  --role="roles/run.admin" \
  --quiet
gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
  --member="serviceAccount:${CLOUDBUILD_SA}" \
  --role="roles/artifactregistry.writer" \
  --quiet
gcloud iam service-accounts add-iam-policy-binding "${COMPUTE_SA}" \
  --member="serviceAccount:${CLOUDBUILD_SA}" \
  --role="roles/iam.serviceAccountUser" \
  --project="${PROJECT_ID}" \
  --quiet
echo "  Cloud Build can deploy to Cloud Run."

# ─── 8. Verify ────────────────────────────────────────────────────────────────

echo "[8/8] Verification..."
echo ""
echo "  Project:           ${PROJECT_ID}"
echo "  Region:            ${REGION}"
echo "  Artifact Registry: ${REGION}-docker.pkg.dev/${PROJECT_ID}/vibecat-images"
echo "  Firestore:         (default) in ${REGION}"
echo "  Secrets:           vibecat-gemini-api-key, vibecat-gateway-auth-secret"
echo "  Compute SA:        ${COMPUTE_SA}"
echo "  Cloud Build SA:    ${CLOUDBUILD_SA}"
echo ""

# Check if Gemini API key has a version
if gcloud secrets versions list vibecat-gemini-api-key \
    --project="${PROJECT_ID}" --format="value(name)" --limit=1 2>/dev/null | grep -q .; then
  echo "  Gemini API key: STORED"
else
  echo "  Gemini API key: NOT YET STORED"
  echo "    Store it with:"
  echo "    echo -n 'YOUR_KEY' | gcloud secrets versions add vibecat-gemini-api-key --data-file=-"
fi

echo ""
echo "=== Setup Complete ==="
echo ""
echo "Next steps:"
echo "  1. Store Gemini API key (if not done): GEMINI_API_KEY=xxx ./infra/setup.sh"
echo "  2. Deploy services: ./infra/deploy.sh"
