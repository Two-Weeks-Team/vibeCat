#!/bin/bash
# Usage: ./scripts/desktop-e2e-loop.sh <scenario_name> [max_iterations]
# Example: ./scripts/desktop-e2e-loop.sh chrome_youtube_music 5
#
# Runs a scenario repeatedly until it passes or max iterations reached.
# Default max iterations: 10

set -euo pipefail

SCENARIO=${1:-}
MAX_ITER=${2:-10}

if [ -z "$SCENARIO" ]; then
  echo "Usage: $0 <scenario_name> [max_iterations]"
  echo "Available: terminal_opencode, antigravity_inline, chrome_youtube_music"
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "▶ Loop runner: scenario=$SCENARIO max_iterations=$MAX_ITER"

for i in $(seq 1 $MAX_ITER); do
  echo ""
  echo "=== Iteration $i/$MAX_ITER ==="
  if "$SCRIPT_DIR/desktop-e2e-one.sh" "$SCENARIO"; then
    echo "✅ PASSED on iteration $i"
    exit 0
  fi
  echo "❌ Failed iteration $i — will retry"
  sleep 5
done

echo ""
echo "❌ MAX ITERATIONS REACHED ($MAX_ITER) — scenario $SCENARIO did not pass"
exit 1
