#!/bin/bash

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'
PASS=0
FAIL=0
WARN=0

check() {
    if eval "$2" > /dev/null 2>&1; then
        echo -e "  ${GREEN}PASS${NC} $1"
        PASS=$((PASS + 1))
    else
        echo -e "  ${RED}FAIL${NC} $1"
        FAIL=$((FAIL + 1))
    fi
}

warn() {
    echo -e "  ${YELLOW}WARN${NC} $1"
    WARN=$((WARN + 1))
}

echo "=== VibeCat Pre-Submission Verification ==="
echo ""

echo "[Build]"
check "Swift build" "cd /Users/kimsejun/GitHub/vibeCat/VibeCat && swift build 2>&1 | grep -q 'Build complete'"
check "Go gateway build" "cd /Users/kimsejun/GitHub/vibeCat/backend/realtime-gateway && go build ./... 2>&1"
check "Go orchestrator build" "cd /Users/kimsejun/GitHub/vibeCat/backend/adk-orchestrator && go build ./... 2>&1"
echo ""

echo "[Production Services]"
check "Gateway health" "curl -sf https://realtime-gateway-163070481841.asia-northeast3.run.app/health | grep -q ok"
check "Gateway readiness" "curl -sf https://realtime-gateway-163070481841.asia-northeast3.run.app/readyz | grep -q ok"
echo ""

echo "[Submission Assets]"
check "Architecture diagram" "test -f /Users/kimsejun/GitHub/vibeCat/docs/architecture.png"
check "Demo video script" "test -f /Users/kimsejun/GitHub/vibeCat/docs/DEMO_VIDEO_SCRIPT.md"
check "Devpost submission text" "test -f /Users/kimsejun/GitHub/vibeCat/docs/DEVPOST_SUBMISSION.md"
check "SRT subtitles" "test -f /Users/kimsejun/GitHub/vibeCat/docs/demo_subtitles.srt"
check "Deployment evidence" "test -f /Users/kimsejun/GitHub/vibeCat/docs/evidence/DEPLOYMENT_EVIDENCE.md"
check "Deploy script (bonus)" "test -x /Users/kimsejun/GitHub/vibeCat/infra/deploy.sh"
echo ""

echo "[Code Quality]"
check "Go vet gateway" "cd /Users/kimsejun/GitHub/vibeCat/backend/realtime-gateway && go vet ./... 2>&1"
check "Go vet orchestrator" "cd /Users/kimsejun/GitHub/vibeCat/backend/adk-orchestrator && go vet ./... 2>&1"
echo ""

echo "[Key Files for Judges]"
check "README.md exists" "test -f /Users/kimsejun/GitHub/vibeCat/README.md"
check "README has spin-up instructions" "grep -q 'swift build' /Users/kimsejun/GitHub/vibeCat/README.md"
check "README has deploy instructions" "grep -q 'deploy.sh' /Users/kimsejun/GitHub/vibeCat/README.md"
check "Dockerfiles exist" "test -f /Users/kimsejun/GitHub/vibeCat/backend/realtime-gateway/Dockerfile && test -f /Users/kimsejun/GitHub/vibeCat/backend/adk-orchestrator/Dockerfile"
echo ""

echo "[Third-Party Disclosures]"
check "GenAI SDK mentioned" "grep -q 'GenAI SDK\|google.golang.org/genai' /Users/kimsejun/GitHub/vibeCat/docs/DEVPOST_SUBMISSION.md"
check "ADK mentioned" "grep -q 'ADK\|Agent Development Kit' /Users/kimsejun/GitHub/vibeCat/docs/DEVPOST_SUBMISSION.md"
check "chromedp mentioned" "grep -q 'chromedp' /Users/kimsejun/GitHub/vibeCat/docs/DEVPOST_SUBMISSION.md"
echo ""

echo "=== Results ==="
echo -e "  ${GREEN}PASS: ${PASS}${NC}  ${RED}FAIL: ${FAIL}${NC}  ${YELLOW}WARN: ${WARN}${NC}"
if [ $FAIL -eq 0 ]; then
    echo -e "  ${GREEN}Ready for submission!${NC}"
else
    echo -e "  ${RED}Fix ${FAIL} issue(s) before submitting.${NC}"
fi
