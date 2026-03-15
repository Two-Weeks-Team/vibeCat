#!/bin/bash
set -euo pipefail

GATEWAY="https://realtime-gateway-163070481841.asia-northeast3.run.app"
VIBECAT="/Users/kimsejun/GitHub/vibeCat/VibeCat/.build/arm64-apple-macosx/debug/VibeCat"
LOG="/tmp/vibecat-rehearsal-$(date +%H%M%S).log"
PASS=0; FAIL=0; RUN=$1

inject() { curl -s "${GATEWAY}/debug/inject-text?text=$(python3 -c "import urllib.parse; print(urllib.parse.quote('$1'))")" > /dev/null; }

echo "===== 리허설 ${RUN}회차 시작 ====="

# 클린 스타트
pkill -f "VibeCat" 2>/dev/null || true
kill -9 $(pgrep -x "Google Chrome") 2>/dev/null || true
osascript -e 'tell application "Terminal" to close every window' 2>/dev/null || true
osascript -e 'tell application "System Events" to keystroke return' 2>/dev/null || true
sleep 3

echo "" > "$LOG"
open -a "Antigravity" /Users/kimsejun/GitHub/vibeCat/demo/UserService.swift
sleep 2; osascript -e 'tell application "Antigravity" to activate'

nohup "$VIBECAT" > "$LOG" 2>&1 &
echo "VibeCat PID: $!"
sleep 18

# 첫 발화 대기
for i in $(seq 1 15); do
    sleep 2
    S=$(strings "$LOG" 2>/dev/null | grep "transcription FINALIZED" | tail -1 | sed 's/.*): //')
    [ -n "$S" ] && echo "🐱 $S" && break
done

# ===== S1: 음악 =====
echo ""; echo "--- S1: 음악 ---"
inject "I want music playing. Call navigate_open_url with url https://music.youtube.com/search?q=relaxing+coding+music"
sleep 35

TITLE=$(osascript -e 'tell application "Google Chrome" to execute active tab of window 1 javascript "(function(){var b=document.querySelector(\"ytmusic-player-bar\");return b?b.querySelector(\".title\").textContent:\"no_bar\";})()"' 2>&1)
TABS=$(osascript -e 'tell application "Google Chrome"
    set tc to 0
    repeat with w in every window
        repeat with t in every tab of w
            if URL of t contains "youtube" then set tc to tc + 1
        end repeat
    end repeat
    return tc
end tell' 2>&1)
AUTOPLAY=$(strings "$LOG" | grep -c "NAV-YTMUSIC.*Auto-play.*clicked" || true)

if echo "$TITLE" | grep -qi "coding\|lofi\|chill\|session\|music" && [ "${TABS:-0}" -le 1 ]; then
    echo "✅ S1 PASS (tabs=${TABS}, title=${TITLE})"
    PASS=$((PASS+1))
else
    echo "❌ S1 FAIL (tabs=${TABS}, title=${TITLE}, autoplay=${AUTOPLAY})"
    FAIL=$((FAIL+1))
fi

# S1 정리
osascript -e 'tell application "Google Chrome"
    repeat with w in every window
        repeat with t in every tab of w
            if URL of t contains "youtube" then close t
        end repeat
    end repeat
end tell' 2>/dev/null || true
sleep 1; osascript -e 'tell application "Google Chrome" to activate' 2>/dev/null || true
sleep 0.5; osascript -e 'tell application "System Events" to keystroke return' 2>/dev/null || true
sleep 1; osascript -e 'tell application "Google Chrome" to quit' 2>/dev/null || true
sleep 2

# ===== S2: 코드 주석 =====
echo ""; echo "--- S2: 코드 주석 ---"
osascript -e 'tell application "Antigravity" to activate'; sleep 3
inject "Add a comment to the getUserData function"
sleep 25
inject "Yes add it now. Use navigate_text_entry to type the comment."
sleep 25

S2_FC=$(strings "$LOG" | grep -c "commandAccepted.*text_entry" || true)
if [ "${S2_FC:-0}" -ge 1 ]; then
    echo "✅ S2 PASS (text_entry=${S2_FC})"
    PASS=$((PASS+1))
else
    echo "❌ S2 FAIL (text_entry=${S2_FC})"
    FAIL=$((FAIL+1))
fi

# ===== S3: 터미널 lint =====
echo ""; echo "--- S3: 터미널 lint ---"
osascript -e 'tell application "Terminal" to do script "cd ~"' 2>/dev/null || true
sleep 2; osascript -e 'tell application "Terminal" to activate'; sleep 3
inject "Check the project lint for me"
sleep 20
inject "Yes run it. Use navigate_text_entry to run go vet in the Terminal."
sleep 25

S3_P=$(strings "$LOG" | grep -c "NAV-EXEC.*paste_text.*success" || true)
if [ "${S3_P:-0}" -ge 1 ]; then
    echo "✅ S3 PASS (paste=${S3_P})"
    PASS=$((PASS+1))
else
    echo "❌ S3 FAIL (paste=${S3_P})"
    FAIL=$((FAIL+1))
fi

# S3 정리
osascript -e 'tell application "Terminal" to close every window' 2>/dev/null || true
sleep 1; osascript -e 'tell application "System Events" to keystroke return' 2>/dev/null || true

# 결과
echo ""
echo "===== ${RUN}회차 결과: PASS=${PASS}/3 FAIL=${FAIL}/3 ====="
[ $PASS -eq 3 ] && echo "🎉 ${RUN}회차 전체 PASS" || echo "❌ ${RUN}회차 일부 FAIL"

# 정리
pkill -f "VibeCat" 2>/dev/null || true
kill -9 $(pgrep -x "Google Chrome") 2>/dev/null || true

exit $FAIL
