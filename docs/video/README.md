# VibeCat Demo Video — Production Guide

Final output: **`vibecat-final-dubbed.mp4`** (1920x1248, ~144s, H.264 + AAC)

---

## Quick Rebuild

```bash
# 1. Download BGM (one-time)
mkdir -p /tmp/vc-bgm
yt-dlp -x --audio-format wav -o "/tmp/vc-bgm/peace-oliver-jensen.%(ext)s" \
  "https://www.youtube.com/watch?v=uELLrTTnXjM"

# 2. Generate TTS voices (needs GEMINI_API_KEY in .env.test)
python3 scripts/generate-tts.py

# 3. Compose final video
bash scripts/compose-final-video.sh
```

---

## Source Clips

All in `docs/video/clips/`. Numbered in playback order.

| File | Duration | Content | Source |
|------|----------|---------|--------|
| `00-title.mp4` | 4s | Title card (black + text) | Original edit |
| `01-music.mov` | 80s | Music scenario screen recording | Raw capture |
| `02-code-terminal.mov` | 90s | Code + Terminal screen recording | Raw capture |
| `03-architecture.mp4` | 8s | Architecture diagram | Original edit |
| `04-gcp-proof.mp4` | 19s (trimmed to 13s) | GCP Console pages | Original edit |
| `05-ending.mp4` | 5s (extended to 8s) | Ending card | Original edit |

**Archive** (`clips/archive/`): Old edits with burned-in oversized text overlays. Not used in final.

**External source**: `docs/video/archive/original-2-111s.mov` at 1:43-1:49 replaces the code enhancement result scene.

---

## Video Timeline (144s)

```
 0:00  ┌─────────────┐
       │  00-title    │  4s   Title card
 0:04  ├─────────────┤
       │  01-music    │  1s   Architecture + cat greeting bubble
 0:05  ├─ FREEZE ─────┤  5s   Still frame (cat "Hey!" bubble stays visible)
 0:10  ├─────────────┤
       │  01-music    │  26s  YouTube Music opens → search → click
 0:36  ├─ CUT ────────┤       (3s static YT page removed: MOV 27-30s)
       │  01-music    │  15s  YouTube Music playing
 0:51  ├─────────────┤
       │  02-code     │  42s  IDE: code reading → Gemini Chat typing
 1:33  ├─ REPLACE ────┤       (original-2 MOV 1:43-1:49 = enhanced code result)
       │  original-2  │  6s   Gemini analysis panel + Apple-style docs
 1:39  ├─────────────┤
       │  02-code     │  16s  Terminal: go vet, lint check
 1:55  ├─────────────┤
       │  03-arch     │  8s   Architecture diagram
 2:03  ├─────────────┤
       │  04-gcp      │  13s  Cloud Run, Cloud Logging, Secret Manager
 2:16  ├─────────────┤
       │  05-ending   │  5s   Ending card (LOOK→DECIDE→MOVE→CLICK→VERIFY)
       │  + extension │  3s   Last frame held for closing narration
 2:24  └─────────────┘
```

### Key Edits

| Edit | Location | Reason |
|------|----------|--------|
| **Freeze frame** | 0:05-0:10 | Cat greeting TTS (6.2s) needs static visual |
| **Cut 3s** | MOV 27-30s | Static YouTube Music page, no action |
| **Replace 6s** | 1:33-1:39 | Swap unenhanced code with Gemini result from `original-2` |
| **Trim GCP** | 19s → 13s | Remove trailing black frames |
| **Extend ending** | 5s → 8s | Prevent closing narration from being cut off |

---

## Audio Layers

### 1. TTS Voices (21 lines)

Generated via `scripts/generate-tts.py` using Gemini TTS API (`gemini-2.5-flash-preview-tts`).

| Voice | Gemini Name | Role | Volume |
|-------|-------------|------|--------|
| Cat | **Zephyr** | VibeCat character dialogue | x1.5 boost |
| Narrator | **Charon** | Scene descriptions, technical narration | x1.5 boost |
| User | **Puck** | User approval responses | x1.5 boost |

All voices are boosted x1.5 in the final mix because Gemini TTS output levels are low.

#### Full Dubbing Script

| Time | Voice | Text |
|------|-------|------|
| 0:00.5 | Narrator | "VibeCat. Your Proactive Desktop Companion." |
| 0:05.0 | Cat | "Hey! Good to see you. How about we set the vibe with some relaxing background music while you work?" |
| 0:11.5 | User | "Yes, play it." |
| 0:16.0 | Narrator | "Using vision-based control. Look, decide, move, click, verify." |
| 0:23.0 | Cat | "Meow! Music is playing! Enjoy your coding session." |
| 0:37.0 | Narrator | "Music continues in the background throughout the entire demo." |
| 0:51.5 | Narrator | "Act two. Proactive Code Enhancement." |
| 0:55.5 | Narrator | "VibeCat reads your code in Antigravity IDE." |
| 1:00.0 | Narrator | "Clicking Gemini Chat and typing: Enhance the comments for this code." |
| 1:04.0 | Cat | "Done! I've typed 'Enhance the comments for this code' into Antigravity." |
| 1:10.0 | Cat | "Nya! You're working on getUserData. Don't forget to add logic to save the data to sessionCache afterwards!" |
| 1:21.0 | Narrator | "Gemini AI analyzes and improves code documentation." |
| 1:27.0 | Narrator | "Gemini completes enhancement with detailed documentation." |
| 1:38.5 | Narrator | "Act three. Terminal Automation." |
| 1:41.0 | Narrator | "VibeCat switches to Terminal and reads the screen." |
| 1:46.0 | Narrator | "Typing go vet, verifying the result. Lint check passed." |
| 1:50.0 | Cat | "Running go vet! Checking for lint issues for you." |
| 1:55.5 | Narrator | "Architecture overview. Swift client connects to Cloud Run gateway, powered by Gemini Live API." |
| 2:04.0 | Narrator | "Running live on Google Cloud Platform. Cloud Run, Firestore, and Cloud Logging." |
| 2:12.0 | Narrator | "VibeCat by Two Weeks Team. Observe, suggest, wait, act, verify. Built for the Gemini Live Agent Challenge 2026." |

### 2. Background Music

| Property | Value |
|----------|-------|
| Song | "Peace" by Oliver Jensen (Lofi Type Beats) |
| Source | `https://www.youtube.com/watch?v=uELLrTTnXjM` |
| Duration | 112s |
| Start | 0:22 (play button click in video) |
| Volume | 20% initial → fade to 6% (30% of initial) over 3s |
| Fade-in | 2s at start |
| Fade-out | 5s at song end (~2:09) |

### Audio Mix

```
TTS (x1.5 boost) ─┐
                   ├─ amix (normalize=0) ─→ AAC 192kbps 48kHz
BGM (20%→6%)   ───┘
```

---

## Subtitles

`subtitles-final.srt` — burned into video via ffmpeg `subtitles` filter.

| Property | Value |
|----------|-------|
| Font | Arial, size 13 |
| Color | White with black outline (2px) + shadow |
| Position | Bottom center, 30px margin |
| Entries | 21 |

---

## Scripts

| Script | Purpose |
|--------|---------|
| `scripts/generate-tts.py` | Reads `dubbing-script.json`, calls Gemini TTS API, outputs WAV files to `docs/video/tts/` |
| `scripts/compose-final-video.sh` | Full pipeline: clip prep → freeze → concat → TTS mix → BGM → subtitles → final MP4 |
| `scripts/dubbing-script.json` | Voice assignments, timestamps (ms), and text for all 21 TTS lines |

### Dependencies

- `ffmpeg` with libx264 and subtitles filter
- `python3` with `google-genai` package
- `yt-dlp` (BGM download only)
- `git-lfs` (for video/audio files in repo)
- `GEMINI_API_KEY` in `.env.test`

---

## File Inventory

```
docs/video/
  vibecat-final-dubbed.mp4    Final output (LFS)
  subtitles-final.srt         Subtitle source
  tts/                        Generated TTS WAV files (LFS)
    01-title.wav              ... through 22-closing-motto.wav
    manifest.json             TTS metadata
  clips/
    00-title.mp4              Title card (LFS)
    01-music.mov              Music scenario raw (LFS)
    02-code-terminal.mov      Code+Terminal raw (LFS)
    03-architecture.mp4       Architecture diagram (LFS)
    04-gcp-proof.mp4          GCP Console (LFS)
    05-ending.mp4             Ending card (LFS)
    archive/                  Old edits (not used)
  archive/
    original-1-125s.mov       Full take 1 (LFS)
    original-2-111s.mov       Full take 2 — used for 1:33-1:39 replacement (LFS)
    original-v2-gcp.mov       GCP recording (LFS)
```
