#!/bin/bash
# VibeCat Demo Video Post-Processing Script
# Usage: ./scripts/demo-post-process.sh [input.mov]
#
# This script:
# 1. Trims to 4 minutes max
# 2. Burns in English subtitles
# 3. Compresses for YouTube (H.264 + AAC)
# 4. Optionally creates a version without subtitles

set -euo pipefail

# --- Configuration ---
INPUT="${1:-$HOME/Desktop/Screen Recording*.mov}"
SRT_FILE="$(dirname "$0")/../docs/demo_subtitles.srt"
OUTPUT_DIR="$(dirname "$0")/../docs/video"
MAX_DURATION=240  # 4 minutes

# --- Colors ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}=== VibeCat Demo Video Post-Processor ===${NC}"
echo ""

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Find input file (handle glob)
if [[ "$INPUT" == *"*"* ]]; then
    INPUT_FILE=$(ls -t $INPUT 2>/dev/null | head -1)
    if [ -z "$INPUT_FILE" ]; then
        echo -e "${RED}Error: No screen recording found on Desktop${NC}"
        echo "Usage: $0 /path/to/recording.mov"
        exit 1
    fi
else
    INPUT_FILE="$INPUT"
fi

if [ ! -f "$INPUT_FILE" ]; then
    echo -e "${RED}Error: Input file not found: $INPUT_FILE${NC}"
    exit 1
fi

echo -e "${YELLOW}Input:${NC} $INPUT_FILE"
echo -e "${YELLOW}SRT:${NC}   $SRT_FILE"
echo ""

# Get input duration
DURATION=$(ffprobe -v error -show_entries format=duration -of csv=p=0 "$INPUT_FILE" | cut -d. -f1)
echo -e "${YELLOW}Input duration:${NC} ${DURATION}s"

# --- Step 1: Create clean trimmed version (no subtitles) ---
echo ""
echo -e "${GREEN}[1/3] Creating clean version (no subtitles)...${NC}"
ffmpeg -y -i "$INPUT_FILE" \
    -t $MAX_DURATION \
    -c:v libx264 -preset medium -crf 20 \
    -c:a aac -b:a 192k \
    -movflags +faststart \
    -vf "scale=trunc(iw/2)*2:trunc(ih/2)*2" \
    "$OUTPUT_DIR/vibecat-demo-clean.mp4" 2>/dev/null

echo -e "${GREEN}  -> ${OUTPUT_DIR}/vibecat-demo-clean.mp4${NC}"

# --- Step 2: Create version with burned-in subtitles ---
echo ""
echo -e "${GREEN}[2/3] Creating version with subtitles...${NC}"
if [ -f "$SRT_FILE" ]; then
    # Need to escape path for ffmpeg subtitles filter
    SRT_ESCAPED=$(echo "$SRT_FILE" | sed "s/'/\\\\'/g" | sed 's/:/\\:/g')
    
    ffmpeg -y -i "$INPUT_FILE" \
        -t $MAX_DURATION \
        -c:v libx264 -preset medium -crf 20 \
        -c:a aac -b:a 192k \
        -movflags +faststart \
        -vf "scale=trunc(iw/2)*2:trunc(ih/2)*2,subtitles=${SRT_ESCAPED}:force_style='FontSize=22,FontName=Arial,PrimaryColour=&H00FFFFFF,OutlineColour=&H00000000,Outline=2,Shadow=1,MarginV=40'" \
        "$OUTPUT_DIR/vibecat-demo-subtitled.mp4" 2>/dev/null
    
    echo -e "${GREEN}  -> ${OUTPUT_DIR}/vibecat-demo-subtitled.mp4${NC}"
else
    echo -e "${YELLOW}  Warning: SRT file not found at $SRT_FILE${NC}"
    echo -e "${YELLOW}  Skipping subtitle version${NC}"
fi

# --- Step 3: Create YouTube-optimized version ---
echo ""
echo -e "${GREEN}[3/3] Creating YouTube-optimized version...${NC}"
ffmpeg -y -i "$INPUT_FILE" \
    -t $MAX_DURATION \
    -c:v libx264 -preset slow -crf 18 \
    -c:a aac -b:a 256k -ar 48000 \
    -movflags +faststart \
    -pix_fmt yuv420p \
    -vf "scale=trunc(iw/2)*2:trunc(ih/2)*2" \
    "$OUTPUT_DIR/vibecat-demo-youtube.mp4" 2>/dev/null

echo -e "${GREEN}  -> ${OUTPUT_DIR}/vibecat-demo-youtube.mp4${NC}"

# --- Summary ---
echo ""
echo -e "${GREEN}=== Post-Processing Complete ===${NC}"
echo ""
echo "Output files:"
ls -lh "$OUTPUT_DIR"/vibecat-demo-*.mp4 2>/dev/null
echo ""
echo -e "${YELLOW}Recommended for YouTube upload:${NC}"
echo "  1. vibecat-demo-youtube.mp4 (high quality, no burned subs)"
echo "     + Upload docs/demo_subtitles.srt as separate subtitle track"
echo ""
echo "  OR"
echo ""
echo "  2. vibecat-demo-subtitled.mp4 (subtitles burned in)"
echo "     No separate SRT needed, but subs can't be toggled off"
echo ""
echo -e "${GREEN}YouTube recommends uploading separate SRT for accessibility.${NC}"
