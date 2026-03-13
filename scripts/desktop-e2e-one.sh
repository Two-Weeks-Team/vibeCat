#!/bin/bash
# Usage: ./scripts/desktop-e2e-one.sh <scenario_name>
# Example: ./scripts/desktop-e2e-one.sh chrome_youtube_music
#
# Runs a single desktop E2E scenario by name.
# The scenario name is matched against TestDesktopLive_* test functions.
# Available scenarios: terminal_opencode, antigravity_inline, chrome_youtube_music

set -euo pipefail

SCENARIO=${1:-}
if [ -z "$SCENARIO" ]; then
  echo "Usage: $0 <scenario_name>"
  echo "Available: terminal_opencode, antigravity_inline, chrome_youtube_music"
  exit 1
fi

export VIBECAT_E2E_CONTROL=1
export DESKTOP_E2E=1

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/../tests/e2e"

echo "▶ Running desktop E2E scenario: $SCENARIO"
go test -v -count=1 -run "TestDesktopLive_.*${SCENARIO}" -timeout 120s ./...
