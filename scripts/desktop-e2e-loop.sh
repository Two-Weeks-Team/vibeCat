#!/bin/bash
# Usage: ./scripts/desktop-e2e-loop.sh <scenario_name> [N] [--until-pass]
#
# Modes:
#   Default: Runs scenario N consecutive times, exits on first failure.
#   --until-pass: Retries until N consecutive passes (max 50 total attempts).
#
# Examples:
#   ./scripts/desktop-e2e-loop.sh chrome_youtube_music 5
#   ./scripts/desktop-e2e-loop.sh terminal_opencode 3 --until-pass

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

SCENARIO=${1:-}
CONSECUTIVE=${2:-3}
MODE="strict"
MAX_TOTAL=50

if [ -z "$SCENARIO" ]; then
  echo -e "${RED}Usage: $0 <scenario_name> [consecutive_passes] [--until-pass]${NC}"
  echo "Available: terminal_opencode, antigravity_inline, chrome_youtube_music"
  exit 1
fi

for arg in "$@"; do
  if [ "$arg" = "--until-pass" ]; then
    MODE="until_pass"
  fi
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo -e "${YELLOW}Loop runner: scenario=$SCENARIO consecutive=$CONSECUTIVE mode=$MODE${NC}"

if [ "$MODE" = "until_pass" ]; then
  pass_streak=0
  attempt=0
  while [ $pass_streak -lt "$CONSECUTIVE" ] && [ $attempt -lt $MAX_TOTAL ]; do
    attempt=$((attempt + 1))
    echo ""
    echo -e "${YELLOW}=== Attempt $attempt (streak: $pass_streak/$CONSECUTIVE) ===${NC}"
    if "$SCRIPT_DIR/desktop-e2e-one.sh" "$SCENARIO"; then
      pass_streak=$((pass_streak + 1))
      echo -e "${GREEN}PASS (streak: $pass_streak/$CONSECUTIVE)${NC}"
    else
      pass_streak=0
      echo -e "${RED}FAIL (streak reset)${NC}"
    fi
    [ $pass_streak -lt "$CONSECUTIVE" ] && sleep 3
  done

  echo ""
  if [ $pass_streak -ge "$CONSECUTIVE" ]; then
    echo -e "${GREEN}$CONSECUTIVE consecutive passes achieved in $attempt attempts${NC}"
    exit 0
  else
    echo -e "${RED}Failed to achieve $CONSECUTIVE consecutive passes in $MAX_TOTAL attempts${NC}"
    exit 1
  fi
else
  for i in $(seq 1 "$CONSECUTIVE"); do
    echo ""
    echo -e "${YELLOW}=== Run $i/$CONSECUTIVE ===${NC}"
    if ! "$SCRIPT_DIR/desktop-e2e-one.sh" "$SCENARIO"; then
      echo -e "${RED}FAILED on run $i/$CONSECUTIVE${NC}"
      exit 1
    fi
    echo -e "${GREEN}PASS ($i/$CONSECUTIVE)${NC}"
    [ "$i" -lt "$CONSECUTIVE" ] && sleep 3
  done

  echo ""
  echo -e "${GREEN}All $CONSECUTIVE consecutive runs passed${NC}"
  exit 0
fi
