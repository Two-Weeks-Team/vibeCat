#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
VIDEO_DIR="$PROJECT_DIR/docs/video"
CLIPS="$VIDEO_DIR/clips"
TTS_DIR="$VIDEO_DIR/tts"
OUTPUT="$VIDEO_DIR/vibecat-final-dubbed.mp4"
SRT="$VIDEO_DIR/subtitles-final.srt"
BGM_SRC="/tmp/vc-bgm/peace-oliver-jensen.wav"
DUBBING_JSON="$SCRIPT_DIR/dubbing-script.json"

RES="1920:1248"
FPS=30
FREEZE_AT=1        # seconds into clip1 MOV where to freeze
FREEZE_DUR=5       # seconds of freeze
BGM_START_MS=22000 # 17s original + 5s freeze shift
TTS_VOL=1.5         # all voices boosted (TTS source is quiet)

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}=== VibeCat Final Video Composer (v3) ===${NC}"
echo "  Freeze: ${FREEZE_DUR}s at clip1 ${FREEZE_AT}s | BGM: @${BGM_START_MS}ms | TTS: ${TTS_VOL}"

echo -e "\n${YELLOW}[1/6] Building clip1 with freeze frame...${NC}"

ffmpeg -y -i "$CLIPS/01-music.mov" -t $FREEZE_AT \
    -vf "scale=$RES" -r $FPS \
    -c:v libx264 -preset medium -crf 18 -an \
    /tmp/vc-c1-before.mp4 2>/dev/null
echo "  clip1 part A: 0-${FREEZE_AT}s"

ffmpeg -y -ss $FREEZE_AT -i "$CLIPS/01-music.mov" -frames:v 1 \
    -vf "scale=$RES" /tmp/vc-freeze.jpg 2>/dev/null

ffmpeg -y -loop 1 -i /tmp/vc-freeze.jpg -t $FREEZE_DUR \
    -vf "scale=$RES" -r $FPS -pix_fmt yuv420p \
    -c:v libx264 -preset medium -crf 18 \
    /tmp/vc-c1-freeze.mp4 2>/dev/null
echo "  freeze: ${FREEZE_DUR}s still frame"

CUT_START=27
CUT_END=30
ffmpeg -y -ss $FREEZE_AT -i "$CLIPS/01-music.mov" -t $((CUT_START - FREEZE_AT)) \
    -vf "scale=$RES" -r $FPS \
    -c:v libx264 -preset medium -crf 18 -an \
    /tmp/vc-c1-b1.mp4 2>/dev/null
echo "  clip1 part B1: ${FREEZE_AT}-${CUT_START}s"

ffmpeg -y -ss $CUT_END -i "$CLIPS/01-music.mov" -t $((45 - CUT_END)) \
    -vf "scale=$RES" -r $FPS \
    -c:v libx264 -preset medium -crf 18 -an \
    /tmp/vc-c1-b2.mp4 2>/dev/null
echo "  clip1 part B2: ${CUT_END}-45s (skipped ${CUT_START}-${CUT_END}s)"

echo -e "\n${YELLOW}[2/6] Preparing other clips...${NC}"

GCP_TRIM=13
ORIG2="$VIDEO_DIR/archive/original-2-111s.mov"
C2_REPLACE_AT=47
C2_REPLACE_SRC_START=103
C2_REPLACE_SRC_END=109
C2_REJOIN=55

ffmpeg -y -ss 5 -i "$CLIPS/02-code-terminal.mov" -t $((C2_REPLACE_AT - 5)) \
    -vf "scale=$RES" -r $FPS \
    -c:v libx264 -preset medium -crf 18 -an \
    /tmp/vc-c2-a.mp4 2>/dev/null
echo "  clip2-a: MOV 5-${C2_REPLACE_AT}s ($(($C2_REPLACE_AT - 5))s)"

REPLACE_DUR=$((C2_REPLACE_SRC_END - C2_REPLACE_SRC_START))
ffmpeg -y -ss $C2_REPLACE_SRC_START -i "$ORIG2" -t $REPLACE_DUR \
    -vf "scale=$RES" -r $FPS \
    -c:v libx264 -preset medium -crf 18 -an \
    /tmp/vc-c2-replace.mp4 2>/dev/null
echo "  clip2-replace: original-2 ${C2_REPLACE_SRC_START}-${C2_REPLACE_SRC_END}s (${REPLACE_DUR}s)"

ffmpeg -y -ss $C2_REJOIN -i "$CLIPS/02-code-terminal.mov" -t $((71 - C2_REJOIN)) \
    -vf "scale=$RES" -r $FPS \
    -c:v libx264 -preset medium -crf 18 -an \
    /tmp/vc-c2-b.mp4 2>/dev/null
echo "  clip2-b: MOV ${C2_REJOIN}-71s ($((71 - C2_REJOIN))s) — Terminal start"

for src in "00-title.mp4" "03-architecture.mp4"; do
    out="/tmp/vc-${src}"
    ffmpeg -y -i "$CLIPS/$src" -vf "scale=$RES" -r $FPS \
        -c:v libx264 -preset medium -crf 18 -an \
        "$out" 2>/dev/null
    echo "  $src: scaled"
done

ffmpeg -y -i "$CLIPS/04-gcp-proof.mp4" -t $GCP_TRIM \
    -vf "scale=$RES" -r $FPS \
    -c:v libx264 -preset medium -crf 18 -an \
    /tmp/vc-04-gcp-proof.mp4 2>/dev/null
echo "  04-gcp-proof.mp4: trimmed to ${GCP_TRIM}s"

ENDING_EXT=3
ffmpeg -y -i "$CLIPS/05-ending.mp4" -vf "scale=$RES" -r $FPS \
    -c:v libx264 -preset medium -crf 18 -an \
    /tmp/vc-05-ending-orig.mp4 2>/dev/null
ffmpeg -y -sseof -0.1 -i "$CLIPS/05-ending.mp4" -frames:v 1 \
    -vf "scale=$RES" /tmp/vc-end-frame.jpg 2>/dev/null
ffmpeg -y -loop 1 -i /tmp/vc-end-frame.jpg -t $ENDING_EXT \
    -vf "scale=$RES" -r $FPS -pix_fmt yuv420p \
    -c:v libx264 -preset medium -crf 18 \
    /tmp/vc-05-ending-ext.mp4 2>/dev/null
echo "  05-ending.mp4: 5s + ${ENDING_EXT}s extension = $((5 + ENDING_EXT))s"

echo -e "\n${YELLOW}[3/6] Concatenating video...${NC}"

cat > /tmp/vc-concat.txt << 'EOF'
file '/tmp/vc-00-title.mp4'
file '/tmp/vc-c1-before.mp4'
file '/tmp/vc-c1-freeze.mp4'
file '/tmp/vc-c1-b1.mp4'
file '/tmp/vc-c1-b2.mp4'
file '/tmp/vc-c2-a.mp4'
file '/tmp/vc-c2-replace.mp4'
file '/tmp/vc-c2-b.mp4'
file '/tmp/vc-03-architecture.mp4'
file '/tmp/vc-04-gcp-proof.mp4'
file '/tmp/vc-05-ending-orig.mp4'
file '/tmp/vc-05-ending-ext.mp4'
EOF

ffmpeg -y -f concat -safe 0 -i /tmp/vc-concat.txt \
    -c:v libx264 -preset medium -crf 18 \
    -movflags +faststart -an \
    /tmp/vc-video.mp4 2>/dev/null

TOTAL_DUR=$(ffprobe -v error -show_entries format=duration -of csv=p=0 /tmp/vc-video.mp4)
TOTAL_INT=$(printf "%.0f" "$TOTAL_DUR")
echo "  Total: ${TOTAL_DUR}s (title + freeze + clip1 + clip2 + arch + gcp + end)"

echo -e "\n${YELLOW}[4/6] Mixing TTS (all voices at ${TTS_VOL})...${NC}"

ffmpeg -y -f lavfi -i "anullsrc=r=24000:cl=mono" \
    -t "$TOTAL_INT" -c:a pcm_s16le /tmp/vc-silence.wav 2>/dev/null

FILTER=""
IDX=1
INPUTS="-i /tmp/vc-silence.wav"

while IFS= read -r line; do
    id=$(echo "$line" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
    ms=$(echo "$line" | python3 -c "import sys,json; print(json.load(sys.stdin)['start_ms'])")
    wav="$TTS_DIR/${id}.wav"
    [ ! -f "$wav" ] && echo "  SKIP $id" && continue

    INPUTS="$INPUTS -i $wav"
    FILTER="${FILTER}[${IDX}]adelay=${ms}|${ms}[d${IDX}];"
    IDX=$((IDX + 1))
    echo "  + $id @ ${ms}ms"
done < <(python3 -c "
import json
with open('$DUBBING_JSON') as f:
    for l in json.load(f)['lines']: print(json.dumps(l))
")

MIX="[0]"
for i in $(seq 1 $((IDX - 1))); do MIX="${MIX}[d${i}]"; done
FILTER="${FILTER}${MIX}amix=inputs=${IDX}:duration=first:dropout_transition=0,volume=${TTS_VOL}[tts]"

eval ffmpeg -y $INPUTS \
    -filter_complex "\"$FILTER\"" \
    -map '"[tts]"' -c:a pcm_s16le -ar 24000 \
    /tmp/vc-tts.wav 2>/dev/null
echo "  TTS mixed: $IDX tracks at ${TTS_VOL} volume"

echo -e "\n${YELLOW}[5/6] Preparing BGM (40% start → 30% fade)...${NC}"

if [ -f "$BGM_SRC" ]; then
    BGM_START_SEC=$(echo "scale=1; $BGM_START_MS/1000" | bc)
    ffmpeg -y -i "$BGM_SRC" \
        -af "afade=t=in:d=2,volume='if(lt(t,5),0.20,if(lt(t,8),0.20-0.0467*(t-5),0.06))':eval=frame,afade=t=out:st=107:d=5,adelay=${BGM_START_MS}|${BGM_START_MS},apad=whole_dur=${TOTAL_INT}" \
        -ar 24000 -ac 1 -c:a pcm_s16le \
        /tmp/vc-bgm.wav 2>/dev/null
    echo "  BGM: starts@${BGM_START_SEC}s, 20%→6% over 3s"

    ffmpeg -y -i /tmp/vc-tts.wav -i /tmp/vc-bgm.wav \
        -filter_complex "[0][1]amix=inputs=2:duration=first:normalize=0[out]" \
        -map "[out]" -c:a aac -b:a 192k -ar 48000 \
        /tmp/vc-audio.m4a 2>/dev/null
    echo "  Final audio: TTS + BGM merged"
else
    ffmpeg -y -i /tmp/vc-tts.wav -c:a aac -b:a 192k -ar 48000 \
        /tmp/vc-audio.m4a 2>/dev/null
    echo "  No BGM found, TTS only"
fi

echo -e "\n${YELLOW}[6/6] Final composition...${NC}"

SRT_ESC=$(echo "$SRT" | sed "s/'/\\\\'/g" | sed 's/:/\\:/g')
ffmpeg -y -i /tmp/vc-video.mp4 -i /tmp/vc-audio.m4a \
    -c:v libx264 -preset slow -crf 18 -c:a copy \
    -vf "subtitles=${SRT_ESC}:force_style='FontSize=13,FontName=Arial,PrimaryColour=&H00FFFFFF,OutlineColour=&H00000000,Outline=2,Shadow=1,MarginV=30,Alignment=2'" \
    -movflags +faststart -pix_fmt yuv420p \
    -shortest "$OUTPUT" 2>/dev/null

echo -e "\n${GREEN}=== Verification ===${NC}"
ls -lh "$OUTPUT"
V=$(ffprobe -v error -show_entries stream=width,height,duration -select_streams v:0 -of csv=p=0 "$OUTPUT")
A=$(ffprobe -v error -show_entries stream=codec_name,sample_rate -select_streams a:0 -of csv=p=0 "$OUTPUT")
echo "  Video: $V | Audio: $A"
echo -e "\n${GREEN}=== Done: $OUTPUT ===${NC}"
