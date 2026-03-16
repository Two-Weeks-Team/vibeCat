# VibeCat — 4-Minute Demo Video Script (v4 — With Intro)

**Total Runtime:** 4:00
**Format:** Screen recording + VibeCat voice + user voice + TTS narration
**Language:** All English
**Key Rule:** VibeCat ALWAYS speaks first. User only approves.

> This script is based on VibeCat's **actual observed behavior** during testing.
> VibeCat proactively spoke first 5/5 times, in English, recognizing code, errors, and app state.

---

## CORE FLOW (every scenario)

```
VibeCat observes screen → VibeCat suggests action → User: "Yeah" → VibeCat executes
```

---

## INTRO: THE PROBLEM → THE FLIP (0:00–0:10)

Sets the stage — why VibeCat exists in one breath.

### INTRO-01: The Problem (~5s)

**Visual:** Dark background + `docs/Intro-01.jpg` (frustrated developer surrounded by browser tabs)
**Text overlay (typography):**
```
Every AI tool waits for your command.
But what if one didn't?
```
**Audio:** TTS narration (Kimsejun clone voice)

### INTRO-02: The Flip (~5s)

**Visual:** Transition to `docs/Intro-02.jpg` (VibeCat cat connecting code screens + music)
**Text overlay (typography):**
```
Meet VibeCat — a desktop companion
that suggests before you ask.
```
**Audio:** TTS narration (Kimsejun clone voice)

→ **Cut to live desktop recording — ACT 1 starts immediately**

### Why This Structure

| Element | Effect |
|---------|--------|
| "Every AI tool waits" | Judges immediately relate — all AI tools are reactive, this is known |
| "But what if one didn't?" | Curiosity hook — question form pulls attention forward |
| "suggests before you ask" | VibeCat's core differentiator in one sentence |
| 10 seconds total | 4-minute video cannot afford a long intro — get to the demo fast |

---

## ACT 1: GREETING + MUSIC (0:10–1:40) ⭐ First Impression

### Setup
- Antigravity IDE open with `demo/UserService.swift`
- VibeCat cat character visible in corner

### What Happens
1. **VibeCat speaks first** (proactive, ~10-30s after launch):
   - Expected: *"You've been working hard! Want me to play some chill music on YouTube?"*
   - Or similar: VibeCat will observe the screen and suggest something helpful

2. **User responds** (one word):
   - *"Yeah, play it."*

3. **VibeCat executes** (multi-step, no more user input):
   - Overlay shows: `Opening YouTube Music... [System]`
   - Overlay shows: `Searching: "chill coding music"... [CDP]`
   - Overlay shows: `Starting playback... [Hotkey: Space]`
   - YouTube Music opens → search → music plays

4. **VibeCat confirms**:
   - *"There you go! Music is playing."*

### What the Judges See
- AI that speaks FIRST (not waiting for commands)
- One approval → multi-step autonomous execution
- Real browser automation (not a mock)

---

## ACT 2: CODE FIX (1:40–2:55) ⭐ Technical Depth

### Setup
- Switch back to Antigravity (`Cmd+Tab`)
- `demo/UserService.swift` visible — has intentional null check issues

### What Happens
1. **VibeCat speaks first** (proactive):
   - Expected: *"I notice you're accessing properties on an optional without unwrapping. Want me to add a guard check?"*
   - Or: *"You're adding caching to UserService! The logic for updating the cache before checking nulls could crash."*

2. **User responds**:
   - *"Yeah, fix it."*

3. **VibeCat executes**:
   - Overlay shows: `Focusing Antigravity... [AX]`
   - Overlay shows: `Opening inline prompt... [Hotkey: Cmd+I]`
   - VibeCat uses Antigravity's inline AI to suggest the fix

4. **VibeCat confirms**:
   - *"Done! Added the null check."*

### What the Judges See
- VibeCat reads and understands code on screen
- Proactive bug detection (not just responding to errors)
- Real IDE integration

---

## ACT 3: TERMINAL COMMAND (2:55–3:35) ⭐ Self-Healing

### Setup
- Open Terminal
- Type `ls` and press Enter (basic output visible)

### What Happens
1. **VibeCat speaks first** (proactive):
   - Expected: *"By the way, ls -la would show hidden files and permissions too. Want me to run it?"*

2. **User responds**:
   - *"Do it."*

3. **VibeCat executes**:
   - Overlay shows: `Focusing Terminal... [AX]`
   - Overlay shows: `Typing: ls -la... [Text Entry]`
   - If first attempt fails → Overlay shows: `Retrying... [Attempt 2/3]`
   - Terminal runs `ls -la` → output appears

4. **VibeCat confirms**:
   - *"Got it! Notice the hidden files now showing?"*

### What the Judges See
- Proactive suggestion for better workflow
- Real terminal automation
- Self-healing if first attempt fails (retry with alternative grounding)

---

## ACT 4: ARCHITECTURE + CLOSE (3:35–4:00)

### What Happens
1. Show `architecture.png` in Preview (already open)
2. Flash GCP Console tab in Chrome (5 seconds — Cloud Run services green)
3. VibeCat waves goodbye:
   - *"I'll be here when you need me. Or before you need me."*

### Closing Text on Screen
```
VibeCat — Your Proactive Desktop Companion
github.com/Two-Weeks-Team/vibeCat
Built for the Gemini Live Agent Challenge 2026
```

---

## FALLBACK STRATEGIES

VibeCat is a real AI — it may not say exactly what's scripted. Here's how to handle:

| Situation | Response |
|-----------|----------|
| VibeCat doesn't suggest music | Say: *"Hey VibeCat, can you play some focus music?"* |
| VibeCat suggests something unexpected | Go with it! Approve whatever it suggests — shows authenticity |
| VibeCat's action fails | Great! Shows self-healing. Say: *"Try again?"* |
| VibeCat is quiet for >30s | Switch apps or scroll code to trigger screen analysis |
| VibeCat speaks Korean | This shouldn't happen (language=en), but if so, reply in English |

**The best demo is AUTHENTIC.** Don't fight VibeCat — follow its suggestions.

---

## OBSERVED VIBECAT BEHAVIORS (from testing)

These are things VibeCat **actually said** during our tests:

- *"You've been working hard! Want me to play some chill music on YouTube?"*
- *"You're adding caching to UserService! The logic for updating the cache..."*
- *"A Tavily MCP connection error appeared! Want me to look up the configuration?"*
- *"You're setting up a multi-output device! Should we test the sound?"*
- *"You're on the GDG profile page! Want to complete your developer profile?"*

Pattern: VibeCat always observes → describes what it sees → asks permission.

---

## SCORING TARGETS

| Criteria (Weight) | How This Demo Hits It |
|---|---|
| **Innovation & Multimodal UX (40%)** | Voice-first, proactive AI that speaks before asked. Triple-source grounding (AX+CDP+Vision). No text boxes. |
| **Technical Implementation (30%)** | GenAI SDK + ADK on Cloud Run. 5 FC tools. Self-healing with vision verification. |
| **Demo & Presentation (30%)** | Real working software. Architecture diagram. GCP Console proof. Clear problem→solution narrative. |
| **Blog Bonus (+0.6)** | 18 posts on dev.to/combba |
| **Auto Deploy (+0.2)** | infra/deploy.sh |
| **GDG (+0.2)** | gdg.community.dev/u/m5n58q/ |

---

*Script version: 4.0 — With 10s intro, 2026-03-16*
*Based on 5 observed proactive VibeCat utterances during live testing*
