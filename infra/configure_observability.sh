#!/usr/bin/env bash
set -euo pipefail

PROJECT_ID="${GCP_PROJECT:-vibecat-489105}"
DASHBOARD_NAME="VibeCat Operations Overview"
DASHBOARD_FILE="${DASHBOARD_FILE:-infra/monitoring/vibecat-operations-dashboard.yaml}"

if [[ ! -f "${DASHBOARD_FILE}" ]]; then
  echo "Dashboard file not found: ${DASHBOARD_FILE}" >&2
  exit 1
fi

echo "=== VibeCat Observability ==="
echo "Project:   ${PROJECT_ID}"
echo "Dashboard: ${DASHBOARD_NAME}"

existing_dashboard="$(
  gcloud monitoring dashboards list \
    --project "${PROJECT_ID}" \
    --format='value(name,displayName)' \
  | awk -F'\t' -v target="${DASHBOARD_NAME}" '$2 == target { print $1; exit }'
)"

if [[ -n "${existing_dashboard}" ]]; then
  etag="$(
    gcloud monitoring dashboards describe "${existing_dashboard}" \
      --project "${PROJECT_ID}" \
      --format='value(etag)'
  )"
  tmp_file="$(mktemp)"
  trap 'rm -f "${tmp_file}"' EXIT
  {
    echo "etag: ${etag}"
    cat "${DASHBOARD_FILE}"
  } > "${tmp_file}"

  gcloud monitoring dashboards update "${existing_dashboard}" \
    --project "${PROJECT_ID}" \
    --config-from-file "${tmp_file}"
  echo "Updated dashboard: ${existing_dashboard}"
else
  gcloud monitoring dashboards create \
    --project "${PROJECT_ID}" \
    --config-from-file "${DASHBOARD_FILE}"
  echo "Created dashboard: ${DASHBOARD_NAME}"
fi
