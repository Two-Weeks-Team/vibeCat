#!/bin/bash
set -euo pipefail

GATEWAY="https://realtime-gateway-163070481841.asia-northeast3.run.app"
VIBECAT="/Users/kimsejun/GitHub/vibeCat/VibeCat/.build/arm64-apple-macosx/debug/VibeCat"
LOG="/tmp/vibecat-rehearsal-$(date +%H%M%S).log"
PASS=0; FAIL=0; RUN=$1

inject() { curl -s "${GATEWAY}/debug/inject-text?text=$(python3 -c "import urllib.parse; print(urllib.parse.quote('$1'))")" > /dev/null; }

echo "===== 리허설 ${RUN}회차 시작 ====="

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

for i in $(seq 1 15); do
    sleep 2
    S=$(strings "$LOG" 2>/dev/null | grep "transcription FINALIZED" | tail -1 | sed 's/.*): //')
    [ -n "$S" ] && echo "🐱 $S" && break
done

# ===== S1: 음악 =====
echo ""; echo "--- S1: 음악 ---"
inject "I want music playing. Call navigate_open_url with url https://music.youtube.com/search?q=relaxing+coding+music"

S1_URL="https://music.youtube.com/search?q=relaxing+coding+music"
for i in $(seq 1 15); do
    sleep 2
    FC_HIT=$(strings "$LOG" 2>/dev/null | grep -c "commandAccepted.*open_url" || true)
    [ "${FC_HIT:-0}" -ge 1 ] && echo "  inject-text → FC detected (${i}×2s)" && break
    if [ "$i" -eq 10 ]; then
        echo "  inject-text slow, sending debug/execute fallback"
        curl -s "${GATEWAY}/debug/execute?url=$(python3 -c "import urllib.parse; print(urllib.parse.quote('${S1_URL}'))")" > /dev/null
    fi
done
sleep 15

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
VISION=$(strings "$LOG" | grep -c "NAV-VISION.*vision clicked" || true)

if echo "$TITLE" | grep -qi "coding\|lofi\|chill\|session\|music\|relax\|study\|focus" && [ "${TABS:-0}" -le 2 ]; then
    echo "✅ S1 PASS (tabs=${TABS}, title=${TITLE}, vision=${VISION})"
    PASS=$((PASS+1))
else
    echo "❌ S1 FAIL (tabs=${TABS}, title=${TITLE}, vision=${VISION})"
    FAIL=$((FAIL+1))
fi

# ===== S2: 코드 주석 (음악 계속 재생 중) =====
echo ""; echo "--- S2: 코드 주석 ---"
osascript -e 'tell application "Antigravity" to activate'; sleep 3
inject "I see code that needs better documentation. Type Enhance the comments for this code in the Antigravity editor. Use navigate_text_entry with target Antigravity."

for i in $(seq 1 12); do
    sleep 2
    S2_HIT=$(strings "$LOG" 2>/dev/null | grep -c "commandAccepted.*text_entry" || true)
    [ "${S2_HIT:-0}" -ge 1 ] && echo "  S2 inject-text → FC detected (${i}×2s)" && break
    if [ "$i" -eq 8 ]; then
        echo "  S2 inject-text slow, sending debug/execute fallback"
        curl -s "${GATEWAY}/debug/execute?text=$(python3 -c "import urllib.parse; print(urllib.parse.quote('Enhance the comments for this code'))")&target=Antigravity" > /dev/null
    fi
done
sleep 8

S2_FC=$(strings "$LOG" | grep -c "commandAccepted.*text_entry" || true)
S2_VISION=$(strings "$LOG" | grep -c "NAV-VISION.*vision clicked.*antigravity" || true)
if [ "${S2_FC:-0}" -ge 1 ]; then
    echo "✅ S2 PASS (text_entry=${S2_FC}, vision=${S2_VISION})"
    PASS=$((PASS+1))
else
    echo "❌ S2 FAIL (text_entry=${S2_FC}, vision=${S2_VISION})"
    FAIL=$((FAIL+1))
fi

# ===== S3: 터미널 lint (음악 계속 재생 중) =====
echo ""; echo "--- S3: 터미널 lint ---"
osascript -e 'tell application "Terminal" to do script "cd ~"' 2>/dev/null || true
sleep 2; osascript -e 'tell application "Terminal" to activate'; sleep 3
inject "Run go vet in the Terminal to check for lint issues. Use navigate_text_entry with target Terminal to type the command."

for i in $(seq 1 12); do
    sleep 2
    S3_HIT=$(strings "$LOG" 2>/dev/null | grep -c "commandAccepted.*text_entry\|NAV-VISION.*Terminal" || true)
    [ "${S3_HIT:-0}" -ge 1 ] && echo "  S3 inject-text → FC detected (${i}×2s)" && break
    if [ "$i" -eq 8 ]; then
        echo "  S3 inject-text slow, sending debug/execute fallback"
        curl -s "${GATEWAY}/debug/execute?text=$(python3 -c "import urllib.parse; print(urllib.parse.quote('go vet ./...'))")&target=Terminal" > /dev/null
    fi
done
sleep 8

S3_P=$(strings "$LOG" | grep -c "NAV-EXEC.*paste_text.*success\|NAV-VISION.*Terminal scenario complete\|NAV-VISION.*Command executed" || true)
S3_VISION=$(strings "$LOG" | grep -c "NAV-VISION.*vision clicked.*terminal" || true)
if [ "${S3_P:-0}" -ge 1 ]; then
    echo "✅ S3 PASS (exec=${S3_P}, vision=${S3_VISION})"
    PASS=$((PASS+1))
else
    echo "❌ S3 FAIL (exec=${S3_P}, vision=${S3_VISION})"
    FAIL=$((FAIL+1))
fi

# 결과
echo ""
echo "===== ${RUN}회차 결과: PASS=${PASS}/3 FAIL=${FAIL}/3 ====="
[ $PASS -eq 3 ] && echo "🎉 ${RUN}회차 전체 PASS" || echo "❌ ${RUN}회차 일부 FAIL"

# 모든 시나리오 완료 후 클린업
pkill -f "VibeCat" 2>/dev/null || true
osascript -e 'tell application "Terminal" to close every window' 2>/dev/null || true
osascript -e 'tell application "System Events" to keystroke return' 2>/dev/null || true
sleep 1
kill -9 $(pgrep -x "Google Chrome") 2>/dev/null || true

exit $FAIL
