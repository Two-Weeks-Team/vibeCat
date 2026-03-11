#!/usr/bin/env bash
set -euo pipefail

PROJECT_ID="${GCP_PROJECT:-${GCP_PROJECT_ID:-vibecat-489105}}"
CONFIG_PATH="${1:-infra/observability/vibecat-runtime-dashboard.json}"

if ! command -v gcloud >/dev/null 2>&1; then
  echo "gcloud is required" >&2
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required" >&2
  exit 1
fi

DISPLAY_NAME="$(jq -r '.displayName' "$CONFIG_PATH")"
if [[ -z "$DISPLAY_NAME" || "$DISPLAY_NAME" == "null" ]]; then
  echo "dashboard displayName is missing in $CONFIG_PATH" >&2
  exit 1
fi

existing_name="$(
  gcloud monitoring dashboards list \
    --project "$PROJECT_ID" \
    --format=json | jq -r --arg name "$DISPLAY_NAME" '.[] | select(.displayName == $name) | .name' | head -n 1
)"

if [[ -n "$existing_name" ]]; then
  tmp_config="$(mktemp)"
  jq --arg dashboard_name "$existing_name" '.name = $dashboard_name' "$CONFIG_PATH" > "$tmp_config"
  gcloud monitoring dashboards update --project "$PROJECT_ID" --config-from-file "$tmp_config"
  rm -f "$tmp_config"
else
  gcloud monitoring dashboards create --project "$PROJECT_ID" --config-from-file "$CONFIG_PATH"
fi
