#!/bin/bash
# Run all 3 desktop E2E scenarios with per-scenario consecutive-pass requirements.
#
# Requirements:
#   terminal_opencode:    3 consecutive passes
#   antigravity_inline:   3 consecutive passes
#   chrome_youtube_music: 5 consecutive passes
#
# Usage: ./scripts/desktop-e2e-all.sh
#
# Requires VibeCat E2E control bridge running on localhost:9876.
# Set VIBECAT_E2E_BRIDGE_URL to override the bridge URL.
# Set ORCHESTRATOR_URL to enable Gemini Vision verification.

set -uo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

export VIBECAT_E2E_CONTROL=1
export DESKTOP_E2E=1

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo -e "${CYAN}Desktop E2E Full Suite${NC}"
echo "  Bridge: ${VIBECAT_E2E_BRIDGE_URL:-http://localhost:9876}"
echo "  Orchestrator: ${ORCHESTRATOR_URL:-<not set>}"
echo ""

declare -a SCENARIOS=("terminal_opencode" "antigravity_inline" "chrome_youtube_music")
declare -a PASSES=(3 3 5)
declare -a RESULTS=()
total=0
passed=0
failed=0

for idx in "${!SCENARIOS[@]}"; do
  scenario="${SCENARIOS[$idx]}"
  required="${PASSES[$idx]}"
  total=$((total + 1))

  echo -e "${YELLOW}=== [$scenario] ${required}x consecutive ===${NC}"
  if "$SCRIPT_DIR/desktop-e2e-loop.sh" "$scenario" "$required"; then
    echo -e "${GREEN}[$scenario] PASSED${NC}"
    RESULTS+=("PASS")
    passed=$((passed + 1))
  else
    echo -e "${RED}[$scenario] FAILED${NC}"
    RESULTS+=("FAIL")
    failed=$((failed + 1))
  fi
  echo ""
done

echo -e "${CYAN}========== SUMMARY ==========${NC}"
for idx in "${!SCENARIOS[@]}"; do
  scenario="${SCENARIOS[$idx]}"
  result="${RESULTS[$idx]}"
  required="${PASSES[$idx]}"
  if [ "$result" = "PASS" ]; then
    echo -e "  ${GREEN}$scenario (${required}x): $result${NC}"
  else
    echo -e "  ${RED}$scenario (${required}x): $result${NC}"
  fi
done
echo ""
echo -e "  Total: $total | ${GREEN}Passed: $passed${NC} | ${RED}Failed: $failed${NC}"
echo -e "${CYAN}=============================${NC}"

if [ $failed -gt 0 ]; then
  exit 1
fi
echo -e "${GREEN}All scenarios passed!${NC}"
exit 0
