# VibeCat — 4-Minute Demo Video Script

**Total Runtime:** 4:00
**Format:** Screen recording + voiceover + live cat overlay
**Tone:** Conversational, slightly witty — like a smart colleague, not a product pitch
**Key Identity:** OBSERVE → SUGGEST → WAIT → ACT → FEEDBACK

---

## PRE-ROLL (0:00–0:05)

> *Black screen. Soft lo-fi beat fades in.*

**TEXT ON SCREEN:**
```
Every developer knows this feeling...
```

---

## ACT 1: THE HOOK — LOSING FLOW (0:05–0:25)

**[SCREEN ACTION]**
Fast-cut montage, ~3 seconds each:
1. Code editor open — developer typing fast, in the zone
2. Terminal pops up — runs a command, waits
3. Browser opens — Googles an error message
4. Slack notification — developer glances, loses focus
5. Back to editor — cursor blinking, momentum gone

**[NARRATION — V.O.]**
> "Context switching. It's the silent killer of deep work. You're in the zone, then — terminal, browser, Slack, back to code. By the time you're back, the thought is gone."

**[TEXT ON SCREEN — brief flash]**
```
What if your computer just... helped?
```

---

## ACT 2: MAGIC MOMENT — "VIBECAT SPEAKS FIRST" (0:25–1:15) ⭐ PRIMARY WOW

### 0:25 — Setup

**[SCREEN ACTION]**
- Antigravity IDE fills the screen
- Developer typing a function — `getUserData()` — looks normal
- Cursor pauses on line 47

**[NARRATION — V.O.]**
> "Meet VibeCat. It's been watching."

---

### 0:33 — VibeCat Appears

**[SCREEN ACTION]**
- Small cat character slides in from bottom-right corner of screen
- Speech bubble appears with a soft *pop* sound
- Grounding overlay badge **"AX"** glows in top-right of IDE window

**[VIBECAT — spoken aloud, friendly voice]**
> *"Hey — I noticed a potential null check missing on line 47. Want me to add it?"*

**[NARRATION — V.O.]**
> "It didn't wait to be asked. It just... noticed."

---

### 0:42 — Developer Responds

**[DEVELOPER — casual]**
> *"Yeah, go ahead."*

---

### 0:45 — Transparent Execution

**[SCREEN ACTION]**
Floating overlay panel appears (semi-transparent, top of screen):
```
● Analyzing...        [AX]
● Planning fix...
● Executing...        Step 1/2
● Verifying...        ✓
```
- IDE cursor moves to line 47 — code is inserted:
  ```swift
  guard let user = user else { return nil }
  ```
- Overlay fades: **"✓ Done"**

**[VIBECAT — spoken]**
> *"Done! Null check added. Want me to run the tests?"*

**[NARRATION — V.O.]**
> "No silence. No mystery. Every step, narrated."

---

### 1:10 — Beat

**[TEXT ON SCREEN — 3 seconds]**
```
Proactive. Transparent. Yours.
```

---

## ACT 3: CROSS-APP FLOW — CHROME + YOUTUBE (1:15–2:15) ⭐ BREADTH WOW

### 1:15 — Developer Asks

**[SCREEN ACTION]**
- Developer still in IDE, cat still visible in corner

**[DEVELOPER — casual]**
> *"Hey, can you find some focus music?"*

---

### 1:20 — VibeCat Takes Over

**[SCREEN ACTION]**
Overlay panel appears:
```
● Focusing Chrome...   [CDP]
● Opening YouTube...
● Searching: "lo-fi focus music"
● Starting playback... [⌨ Space]
```

**[NARRATION — V.O.]**
> "Chrome. YouTube. Search. Play. Four apps, one sentence."

Step-by-step on screen:
1. Chrome comes to foreground — **CDP badge** glows
2. YouTube opens — search bar fills: *"lo-fi focus music"*
3. First result selected
4. Video starts — **Keyboard badge** flashes as Space is pressed

---

### 1:50 — VibeCat Reports Back

**[VIBECAT — spoken]**
> *"There you go — lo-fi beats to code by. I'll keep it going in the background."*

**[SCREEN ACTION]**
- Music playing in Chrome (minimized)
- Developer back in IDE, cat still in corner
- Subtle music waveform animation on cat character

**[NARRATION — V.O.]**
> "It didn't open a dialog. It didn't ask for a URL. It just handled it."

---

### 2:10 — Beat

**[TEXT ON SCREEN — 3 seconds]**
```
Across every app. Without breaking your flow.
```

---

## ACT 4: SELF-HEALING — TERMINAL SCENE (2:15–3:00) ⭐ TECHNICAL WOW

### 2:15 — Terminal Scene

**[SCREEN ACTION]**
- Terminal window in focus
- Developer has just run `ls` — basic output visible

**[VIBECAT — proactive, spoken]**
> *"ls -la would show hidden files and permissions too — want me to run it?"*

**[DEVELOPER]**
> *"Do it."*

---

### 2:25 — First Attempt Fails

**[SCREEN ACTION]**
Overlay panel:
```
● Focusing Terminal...  [AX]
● Typing command...
✗ Focus lost — wrong window
```
- Command appears in wrong app (e.g., IDE search bar)

**[NARRATION — V.O.]**
> "It missed. That happens. Here's what's different."

---

### 2:35 — Self-Healing Kicks In

**[SCREEN ACTION]**
Overlay panel updates:
```
● Retrying...           [Attempt 2/3]
● Re-focusing Terminal  [CDP fallback]
● Retyping: ls -la
● Executing...
● Verifying result...   [Vision ✓]
```
- Terminal correctly focused
- `ls -la` typed and executed
- Output appears — hidden files visible

**[VIBECAT — spoken]**
> *"Got it. Notice the dot-files now showing? There's a .env in there you might want to check."*

**[NARRATION — V.O.]**
> "It retried. It verified. It explained. No crash report, no silent failure."

---

### 2:58 — Beat

**[TEXT ON SCREEN — 2 seconds]**
```
It fails gracefully. Then it fixes itself.
```

---

## ACT 5: ARCHITECTURE + CLOSE (3:00–4:00)

### 3:00 — Architecture Diagram (50 seconds max)

**[SCREEN ACTION]**
Clean dark-background diagram animates in, component by component:

```
[macOS Swift Client]
        ↓  voice + screen
[WebSocket · Cloud Run]
        ↓
[Gemini Live API]
        ↓  function calls
[5 Navigator Tools]
        ↓
[Your Desktop]
```

**[NARRATION — V.O., brisk]**
> "Under the hood: a native Swift client on macOS captures your screen and voice. It streams to a Go backend on Cloud Run. Gemini Live API handles the conversation and decides when to act. When it does, it calls one of five precise navigator tools — text entry, hotkeys, app focus, URL open, or type-and-submit."

As each is mentioned, highlight on diagram:
- **"Native Swift"** — Swift badge glows
- **"Self-healing"** — retry arrows animate
- **"Triple-source grounding"** — AX + CDP + Vision badges appear together
- **"Vision verification"** — screenshot thumbnail with checkmark

**[NARRATION — V.O.]**
> "Triple-source grounding means it checks the accessibility tree, Chrome DevTools, and a live screenshot before every action. It doesn't guess. It confirms."

---

### 3:40 — GCP Console Flash

**[SCREEN ACTION]**
Brief 5-second shot: GCP Cloud Run console showing two services live:
- `realtime-gateway` — green status
- `adk-orchestrator` — green status
- Cloud Logging trace visible in background

**[NARRATION — V.O.]**
> "Deployed on Google Cloud Run. Always on. Zero cold-start drama."

---

### 3:48 — Cat Waves Goodbye

**[SCREEN ACTION]**
- Full screen fades to soft gradient background
- VibeCat cat character centered, does a little wave animation
- Speech bubble appears

**[VIBECAT — spoken, warm]**
> *"I'll be here when you need me. Or before you need me."*

**[TEXT ON SCREEN — fades in]**
```
VibeCat
Your Proactive Desktop Companion
```

**[NARRATION — V.O.]**
> "VibeCat. It watches, it suggests, it acts — and it always waits for you to say yes."

---

### 3:55 — GitHub URL

**[TEXT ON SCREEN]**
```
github.com/[your-handle]/vibeCat

Built for the Gemini Live Agent Challenge 2026
```

> *Music fades out softly at 4:00.*

---

## PRODUCTION NOTES

### Voice Direction
- **VibeCat voice:** Warm, slightly playful. Think "smart intern who's really good." Not robotic, not over-eager.
- **Narration V.O.:** Calm, confident. Conversational pace — not a product ad read.
- **Developer voice:** Casual, minimal. One or two words per response.

### Visual Style
- Dark IDE theme throughout (matches Antigravity IDE aesthetic)
- Cat character: small, non-intrusive, bottom-right corner
- Overlay panel: semi-transparent dark glass, monospace font, subtle glow on status changes
- Grounding badges: small pill labels — `AX` (blue), `CDP` (orange), `⌨` (green), `Vision` (purple)

### WOW Markers Summary

| Timecode | Moment | Why It Lands |
|----------|--------|--------------|
| **0:33** | VibeCat speaks first, unprompted | Breaks expectation — AI that initiates |
| **0:45** | Transparent overlay during execution | Shows the "thinking" — no black box |
| **1:20** | Cross-app flow in one sentence | Demonstrates breadth without a tutorial |
| **2:25** | First attempt fails visibly | Honesty builds trust |
| **2:35** | Self-healing retry + vision verify | Technical depth without jargon |
| **3:48** | Cat waves, speaks last line | Personality close — memorable |

### Key Differentiators (Show, Don't Tell)

| # | Differentiator | Where It Appears |
|---|---------------|-----------------|
| 1 | **Proactive** — VibeCat speaks first | Acts 2, 3, 4 |
| 2 | **Voice-native** — all interactions spoken | Throughout |
| 3 | **Transparent** — overlay shows every step | Acts 2, 3, 4 |
| 4 | **Self-healing** — failure shown and recovered | Act 4 |
| 5 | **Native macOS Swift** — AX tree, real app focus | Act 5 diagram |

---

*Script version: 2.0 — 2026-03-12*
