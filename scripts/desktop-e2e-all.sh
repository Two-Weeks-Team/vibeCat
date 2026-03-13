#!/bin/bash
# Run all 3 desktop E2E scenarios.
#
# Usage: ./scripts/desktop-e2e-all.sh
#
# Requires VibeCat E2E control bridge running on localhost:9876.
# Set VIBECAT_E2E_BRIDGE_URL to override the bridge URL.
# Set ORCHESTRATOR_URL to enable Gemini Vision verification.

set -euo pipefail

export VIBECAT_E2E_CONTROL=1
export DESKTOP_E2E=1

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/../tests/e2e"

echo "▶ Running all desktop E2E scenarios"
echo "  Bridge: ${VIBECAT_E2E_BRIDGE_URL:-http://localhost:9876}"
echo "  Orchestrator: ${ORCHESTRATOR_URL:-<not set — vision verify disabled>}"
echo ""

go test -v -count=1 -run "TestDesktopLive" -timeout 300s ./...
