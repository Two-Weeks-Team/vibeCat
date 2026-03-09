# VibeCat Demo Storyboard — Gemini Live Agent Challenge 2026

**Total Duration:** 4:00  
**Language:** English narration + English subtitles (Korean UI visible)  
**Format:** Split-screen recommended — App on left, GCP Console on right (technical sections)  
**Recording Mode:** Hybrid — pre-recorded B-roll spliced with live demo segments  
**Aspect Ratio:** 16:9, 1920×1080 minimum  

---

## PRE-DEMO SETUP CHECKLIST

> Complete before hitting record. These must be ready before the demo starts.

### GCP Console Tabs (open in advance, logged in as `centisgood@gmail.com`, project `vibecat-489105`)
| Tab | URL Path | Purpose |
|-----|----------|---------|
| Tab 1 | Cloud Run → Services | Show both `realtime-gateway` and `adk-orchestrator` running |
| Tab 2 | Cloud Trace → Trace List | Filter: last 1 hour, service=adk-orchestrator |
| Tab 3 | Cloud Logging → Log Explorer | Filter: `resource.type="cloud_run_revision"` |
| Tab 4 | Firestore → Data → sessions collection | Show memory documents |
| Tab 5 | Secret Manager → `gemini-api-key` | Show key exists, value hidden |
| Tab 6 | Cloud Monitoring → Metrics Explorer | Custom metrics: `vibecat/agent_duration_ms`, `vibecat/active_sessions`, `vibecat/analysis_total` |

### Terminal Windows (pre-staged, not yet running)
```bash
# Terminal A — Gateway live log tail (run at 0:30)
gcloud run services logs tail realtime-gateway --region=asia-northeast3 --project=vibecat-489105

# Terminal B — Orchestrator live log tail (run at 1:00)
gcloud run services logs tail adk-orchestrator --region=asia-northeast3 --project=vibecat-489105

# Terminal C — Local test code with intentional error (pre-written, ready to open)
# File: /tmp/demo_code.py  (see content in [1:00] section)
```

### VibeCat App State
- App installed, character set to `cat`
- App is **closed** before demo starts (will launch live at 0:30)
- System audio output routed to screen recording software
- Microphone active, no noise cancellation (VAD handles it)

---

## STORYBOARD

---

### [0:00 – 0:30] BEFORE — The Empty Chair
**Duration:** 30 seconds  
**Recording Mode:** Pre-recorded (no live interaction needed)

#### Screen Layout
```
┌─────────────────────────────────────────────────────┐
│                                                     │
│   VS Code — dark theme, Python file open            │
│   Cursor blinking. No extensions visible.           │
│   Bottom right corner: empty, dark, silent          │
│                                                     │
│   [Slow zoom toward bottom-right corner]            │
│                                                     │
└─────────────────────────────────────────────────────┘
```

#### Visuals
- Open VS Code with a Python file (`main.py`) — no errors, just code
- Developer typing slowly, pausing, staring at screen
- Clock in corner shows late evening (optional: 11:47 PM)
- Bottom-right corner of screen is empty — this is where VibeCat will appear
- Slow, deliberate zoom toward that empty corner over 20 seconds
- Fade to slight desaturation to emphasize loneliness

#### Narration Script
> *[Quiet, reflective tone. No music yet — ambient keyboard sounds only.]*
>
> "Every solo developer knows this feeling."
>
> *[2-second pause]*
>
> "The empty chair next to you. No one to catch your typos. No one to notice when you're stuck."
>
> *[1-second pause]*
>
> "No one to celebrate when your tests finally pass."
>
> *[Zoom reaches the empty corner. Hold for 2 seconds. Fade to black.]*

#### Subtitle
```
[0:00] "Every solo developer knows this feeling."
[0:08] "The empty chair next to you."
[0:13] "No one to catch your typos. No one to notice when you're stuck."
[0:22] "No one to celebrate when your tests finally pass."
```

#### Director Notes
- No music in this segment — silence is the point
- The empty corner must be clearly visible and centered in the final zoom
- Fade to black at 0:28, hold 2 seconds before next segment

---

### [0:30 – 1:00] MEET VIBECAT — Zero Onboarding
**Duration:** 30 seconds  
**Recording Mode:** Live demo

#### Screen Layout
```
┌──────────────────────┬──────────────────────────────┐
│                      │                              │
│   macOS Desktop      │   Terminal A                 │
│   (VibeCat launching)│   (Gateway log streaming)    │
│                      │                              │
│   Cat sprite appears │   > deviceId: "a3f9c2..."    │
│   bottom-right corner│   > session_token issued     │
│                      │   > character: cat           │
└──────────────────────┴──────────────────────────────┘
```

#### Actions (Live)
1. **[0:30]** Click VibeCat app icon in Dock — app launches
2. **[0:32]** Cat sprite animates in at bottom-right corner (idle animation)
3. **[0:34]** Start Terminal A log tail — Gateway logs appear
4. **[0:38]** Highlight in log: `deviceId`, `session_token`, `character: cat`
5. **[0:45]** Hover over cat — speech bubble: *"안녕! 오늘도 같이 코딩해요 🐱"*
6. **[0:52]** Show: no login screen, no API key prompt, no onboarding wizard

#### Narration Script
> *[Upbeat, warm tone. Soft background music begins — lo-fi, gentle.]*
>
> "VibeCat fills that chair."
>
> *[App launches, cat appears]*
>
> "Install it, and it just works. No API keys. No sign-up. No configuration."
>
> *[Point to Gateway log]*
>
> "A device UUID authenticates you silently. Your session token is issued in the cloud — the API key never touches your machine."
>
> *[Cat waves]*
>
> "It watches your screen, hears your voice, and remembers yesterday's context."

#### Subtitle
```
[0:30] "VibeCat fills that chair."
[0:34] "Install it, and it just works. No API keys. No sign-up."
[0:42] "A device UUID authenticates you silently."
[0:48] "The API key never touches your machine."
[0:54] "It watches your screen, hears your voice, and remembers yesterday's context."
```

#### GCP Console Focus
- **Terminal A** showing Gateway log with `deviceId` and `session_token` fields highlighted
- Optional: briefly flash Secret Manager tab showing `gemini-api-key` exists but value is `[hidden]`

#### Director Notes
- Cat sprite must be clearly visible — ensure screen recording captures the corner
- Gateway log should scroll naturally — do not pause or freeze it
- Music fade-in should be subtle, not distracting

---

### [1:00 – 1:45] SEE — Screen Analysis
**Duration:** 45 seconds  
**Recording Mode:** Live demo + pre-recorded Cloud Trace (spliced)

#### Screen Layout
```
┌──────────────────────┬──────────────────────────────┐
│                      │                              │
│   VS Code            │   Cloud Trace                │
│   demo_code.py open  │   Waterfall view             │
│   Error highlighted  │                              │
│                      │   VisionAgent    ████ 0.4s   │
│   Cat speech bubble: │   MoodDetector   ██   0.2s   │
│   "어, 에러 났네요!  │   Mediator       ███  0.3s   │
│    고쳐드릴까요?"    │   EngagementAgent██   0.2s   │
│                      │   [Total: ~2.5s]             │
└──────────────────────┴──────────────────────────────┘
```

#### Demo Code File (`/tmp/demo_code.py`)
```python
# demo_code.py — pre-written with intentional error
import json

def process_user_data(data):
    result = []
    for item in data:
        parsed = json.loads(item)  # Will fail on non-JSON input
        result.append(parsed["name"])
    return result

# This will throw: json.JSONDecodeError
users = ["Alice", "Bob", "Charlie"]
print(process_user_data(users))
```

#### Actions (Live)
1. **[1:00]** Open `demo_code.py` in VS Code
2. **[1:05]** Run the file in terminal: `python demo_code.py`
3. **[1:07]** Error appears: `json.JSONDecodeError: Expecting value: line 1 column 1`
4. **[1:10]** VibeCat speech bubble appears: *"어, 에러 났네요! 고쳐드릴까요? 🐱"*
5. **[1:15]** Switch right panel to Cloud Trace — show waterfall
6. **[1:20]** Highlight agent execution order: Vision → Mood → Mediator → Engagement
7. **[1:35]** Show total trace duration: ~2.5 seconds from screen capture to speech

#### Narration Script
> *[Curious, engaged tone]*
>
> "VibeCat sees what you see."
>
> *[Error appears in terminal]*
>
> "When an error appears, it doesn't wait for you to ask."
>
> *[Cat speech bubble appears]*
>
> "The VisionAgent analyzes your screen. MoodDetector reads the context. Mediator decides whether to speak."
>
> *[Switch to Cloud Trace]*
>
> "Nine agents, executing in parallel waves — from screen capture to spoken response in under three seconds."

#### Subtitle
```
[1:00] "VibeCat sees what you see."
[1:07] "When an error appears, it doesn't wait for you to ask."
[1:15] "The VisionAgent analyzes your screen."
[1:20] "MoodDetector reads the context. Mediator decides whether to speak."
[1:30] "Nine agents, executing in parallel waves —"
[1:37] "from screen capture to spoken response in under three seconds."
```

#### GCP Console Focus
- **Cloud Trace** → Trace List → click most recent trace
- Waterfall must show: `VisionAgent`, `MoodDetector`, `Mediator`, `EngagementAgent` as named spans
- Total duration span visible: ~2.5s
- Pre-record this trace if live latency is unpredictable — splice at [1:15]

#### Director Notes
- The error must be clearly readable on screen — increase VS Code font size to 18pt
- Cat speech bubble must be legible — ensure contrast against VS Code dark theme
- Cloud Trace waterfall is the technical proof moment — hold on it for at least 10 seconds

---

### [1:45 – 2:15] HEAR — Voice Conversation
**Duration:** 30 seconds  
**Recording Mode:** Live demo (audio must be captured)

#### Screen Layout
```
┌──────────────────────┬──────────────────────────────┐
│                      │                              │
│   macOS Desktop      │   Terminal B                 │
│   VibeCat active     │   (Orchestrator log)         │
│                      │                              │
│   VAD waveform       │   > vad: speech_start        │
│   visible in corner  │   > transcript: "이 에러..."  │
│                      │   > barge_in: true           │
│   Cat responding     │   > affect: curious          │
│   with TTS audio     │   > tts_urgency: medium      │
└──────────────────────┴──────────────────────────────┘
```

#### Actions (Live)
1. **[1:45]** Speak aloud: *"이 에러 어떻게 고쳐?"* (Korean: "How do I fix this error?")
2. **[1:48]** VAD detects speech — show waveform activity in log
3. **[1:50]** VibeCat begins responding (TTS starts)
4. **[1:52]** Interrupt mid-sentence: *"잠깐, 그냥 try-except 쓰면 되는 거야?"*
5. **[1:54]** Barge-in works — VibeCat stops, listens, responds to new question
6. **[2:00]** VibeCat responds in Korean with corrected explanation
7. **[2:08]** Show Terminal B log: `barge_in: true`, `affect: curious`, `tts_urgency: medium`

#### Narration Script
> *[Conversational, natural tone — like showing a friend]*
>
> "And it hears you."
>
> *[Speak to VibeCat]*
>
> "Natural conversation with barge-in. Ask in any language — VibeCat responds in kind."
>
> *[Interrupt VibeCat mid-sentence]*
>
> "Interrupt it mid-sentence. It stops, listens, and adapts."
>
> *[Point to log]*
>
> "AffectiveDialog matches your tone. Urgency shapes the TTS delivery."

#### Subtitle
```
[1:45] "And it hears you."
[1:48] "Natural conversation with barge-in."
[1:52] "Ask in any language — VibeCat responds in kind."
[1:56] "Interrupt it mid-sentence. It stops, listens, and adapts."
[2:05] "AffectiveDialog matches your tone."
[2:10] "Urgency shapes the TTS delivery."
```

#### GCP Console Focus
- **Terminal B** showing Orchestrator log with `vad`, `transcript`, `barge_in`, `affect`, `tts_urgency` fields
- Scroll log naturally — do not pause

#### Director Notes
- This segment requires clean audio capture — use a directional microphone
- Barge-in must visibly work — practice this interaction 3+ times before recording
- If barge-in fails during live take, cut to pre-recorded version of this segment
- Korean speech is intentional — demonstrates multilingual capability

---

### [2:15 – 2:45] CARE — Mood Detection + Celebration
**Duration:** 30 seconds  
**Recording Mode:** Live demo + Cloud Logging (split)

#### Screen Layout
```
┌──────────────────────┬──────────────────────────────┐
│                      │                              │
│   VS Code            │   Cloud Logging              │
│   Tests running...   │                              │
│   ✓ All 5 passed     │   mood_detector:             │
│                      │     state: "frustrated"      │
│   Cat celebrating:   │     → "neutral"              │
│   "우와! 테스트가    │     → "success"              │
│    모두 성공했어요!  │                              │
│    대단해요! 🎉"     │   celebration_trigger:       │
│                      │     fired: true              │
│   [Confetti animation│     tts: "우와! 테스트가..." │
│    on cat sprite]    │                              │
└──────────────────────┴──────────────────────────────┘
```

#### Actions (Live)
1. **[2:15]** Apply the fix to `demo_code.py` (wrap in try-except)
2. **[2:20]** Run tests: `python -m pytest demo_test.py -v`
3. **[2:23]** Tests pass: `5 passed in 0.12s`
4. **[2:25]** VibeCat celebrates with TTS: *"우와, 테스트가 모두 성공했어요! 대단해요! 🎉"*
5. **[2:28]** Cat sprite plays celebration animation (confetti/wave)
6. **[2:32]** Switch right panel to Cloud Logging
7. **[2:35]** Show log entries: `mood_detector: frustrated → success`, `celebration_trigger: fired: true`

#### Demo Test File (`/tmp/demo_test.py`)
```python
# demo_test.py — pre-written, will pass after fix
import pytest
from demo_code import process_user_data

def test_valid_json():
    data = ['{"name": "Alice"}', '{"name": "Bob"}']
    assert process_user_data(data) == ["Alice", "Bob"]

def test_empty_list():
    assert process_user_data([]) == []
```

#### Narration Script
> *[Warm, emotionally resonant tone]*
>
> "VibeCat notices your frustration."
>
> *[Tests pass, cat celebrates]*
>
> "And it celebrates your victories."
>
> *[Switch to Cloud Logging]*
>
> "MoodDetector tracks your emotional arc — from frustrated, to neutral, to success."
>
> "CelebrationTrigger fires. The TTS urgency spikes. The chair is no longer silent."

#### Subtitle
```
[2:15] "VibeCat notices your frustration."
[2:25] "And it celebrates your victories."
[2:32] "MoodDetector tracks your emotional arc —"
[2:36] "from frustrated, to neutral, to success."
[2:40] "CelebrationTrigger fires. The TTS urgency spikes."
[2:43] "The chair is no longer silent."
```

#### GCP Console Focus
- **Cloud Logging** → Log Explorer → filter: `jsonPayload.agent="mood_detector" OR jsonPayload.agent="celebration_trigger"`
- Must show state transition: `frustrated → neutral → success`
- Must show `celebration_trigger.fired: true`
- Pre-record this log view if live filtering is slow

#### Director Notes
- The celebration TTS audio must be audible in the recording — check audio levels
- Cat animation must be visible — ensure sprite is not obscured by VS Code window
- Emotional arc is the key story beat here — linger on the log transition for 5 seconds

#### Optional Extension: Rest Reminder (if time allows — can replace part of [3:15-3:40])
- After 50+ minutes of continuous coding activity, EngagementAgent triggers a rest reminder
- VibeCat says: *"이미 50분 넘게 코딩했어요! 잠깐 쉬어가요~ 🐱"*
- Show log: `engagement_agent: rest_reminder: true, activity_minutes: 52`
- This demonstrates the proactive engagement pipeline: client tracks `activityMinutes` → Gateway passes to Orchestrator → EngagementAgent triggers rest suggestion
- **Demo tip**: Pre-set `sessionStartTime` to 50 minutes ago, or fast-forward with a code injection during demo prep

---

### [2:45 – 3:15] REMEMBER + SEARCH
**Duration:** 30 seconds  
**Recording Mode:** Pre-recorded (Firestore) + Live (Search grounding)

#### Screen Layout
```
┌──────────────────────┬──────────────────────────────┐
│                      │                              │
│   VS Code            │   Firestore Console          │
│   New file open      │   sessions/{deviceId}        │
│                      │   memories: [                │
│   Cat speech bubble: │     "Yesterday: json.loads   │
│   "어제 json.loads   │      error in process_user   │
│    에러 있었잖아요.  │      _data — resolved",      │
│    오늘도 비슷한     │     "Recurring: data parsing │
│    패턴이네요!"      │      issues in ETL scripts"  │
│                      │   ]                          │
│                      │                              │
│                      │   [Search grounding result]  │
│                      │   "Python json best practices│
│                      │    — 3 sources cited"        │
└──────────────────────┴──────────────────────────────┘
```

#### Actions (Live)
1. **[2:45]** Open a new file with similar data-processing code
2. **[2:48]** VibeCat proactively speaks: *"어제 json.loads 에러 있었잖아요. 오늘도 비슷한 패턴이네요!"*
3. **[2:52]** Switch right panel to Firestore — show `sessions/{deviceId}/memories` document
4. **[2:58]** Highlight memory entries referencing yesterday's session
5. **[3:02]** Ask VibeCat: *"Python에서 JSON 파싱 best practice가 뭐야?"*
6. **[3:05]** VibeCat responds with Google Search grounding — cites 3 sources
7. **[3:10]** Show search grounding metadata in Orchestrator log: `search_buddy: grounded: true, sources: 3`

#### Narration Script
> *[Thoughtful, slightly awed tone]*
>
> "It remembers."
>
> *[Cat references yesterday's error]*
>
> "Yesterday's unresolved issues. Last week's patterns. Cross-session context stored in Firestore, retrieved by the MemoryAgent."
>
> *[Ask question, search grounding fires]*
>
> "And when you're stuck, it searches for answers. Google Search grounding — not hallucination. Real sources, cited."

#### Subtitle
```
[2:45] "It remembers."
[2:50] "Yesterday's unresolved issues. Last week's patterns."
[2:56] "Cross-session context stored in Firestore, retrieved by the MemoryAgent."
[3:02] "And when you're stuck, it searches for answers."
[3:07] "Google Search grounding — not hallucination."
[3:12] "Real sources, cited."
```

#### GCP Console Focus
- **Firestore** → Data → `sessions` collection → click `{deviceId}` document → show `memories` array
- Memory entries must reference yesterday's session (pre-populate if needed)
- **Terminal B** log showing `search_buddy: grounded: true, sources: 3`

#### Director Notes
- Firestore view should be pre-populated with realistic memory data — do this before recording
- Search grounding response must include visible source citations in VibeCat's speech bubble
- This segment proves the "memory" and "search" ADK features — both must be clearly visible

---

### [3:15 – 3:40] CHARACTERS — Personality Switch
**Duration:** 25 seconds  
**Recording Mode:** Pre-recorded montage (3 character switches)

#### Screen Layout
```
┌─────────────────────────────────────────────────────┐
│                                                     │
│   [Full screen — app only, no split]                │
│                                                     │
│   Character 1: cat                                  │
│   "안녕! 같이 버그 잡아요 🐱"                        │
│   [Voice: Zephyr — bright, casual]                  │
│                                                     │
│   → Switch → Character 2: trump                     │
│   "This code is TREMENDOUS. The BEST code."         │
│   [Voice: Fenrir — energetic, superlative]          │
│                                                     │
│   → Switch → Character 3: jinwoo                    │
│   "..."  [long pause]  "고쳐."                       │
│   [Voice: Kore — low-calm, minimal]                 │
│                                                     │
└─────────────────────────────────────────────────────┘
```

#### Actions (Pre-recorded)
1. **[3:15]** Show `cat` character — idle animation, speak greeting
2. **[3:20]** Open character selector (right-click or settings menu)
3. **[3:21]** Switch to `trump` — sprite changes, voice changes instantly
4. **[3:24]** Trump speaks: *"This code is TREMENDOUS. The BEST code I've ever seen."*
5. **[3:29]** Switch to `jinwoo` — sprite changes, voice changes
6. **[3:32]** Jinwoo speaks: *"..."* [2-second pause] *"고쳐."* (Korean: "Fix it.")
7. **[3:36]** Quick flash of `saja` and `derpy` sprites (0.5s each)
8. **[3:38]** Return to `cat` — waving

#### Narration Script
> *[Playful, energetic tone — fastest-paced segment]*
>
> "Six unique characters. Each with their own voice, personality, and coding philosophy."
>
> *[trump appears]*
>
> "Trump: bombastic, superlative, relentlessly positive."
>
> *[jinwoo appears]*
>
> "Jinwoo: silent senior engineer. Speaks only when necessary."
>
> *[flash of others]*
>
> "Saja. Derpy. Kimjongun. All powered by the same nine agents — different souls."

#### Subtitle
```
[3:15] "Six unique characters."
[3:17] "Each with their own voice, personality, and coding philosophy."
[3:22] "Trump: bombastic, superlative, relentlessly positive."
[3:29] "Jinwoo: silent senior engineer. Speaks only when necessary."
[3:34] "Saja. Derpy. Kimjongun."
[3:37] "All powered by the same nine agents — different souls."
```

#### Director Notes
- This segment is entirely pre-recorded — no live interaction needed
- Character switches must be instant (< 0.5s) — if there's lag, cut in post
- Trump and Jinwoo are the strongest contrast — lead with these two
- Background music should briefly shift in energy when trump appears, then calm for jinwoo
- Keep this segment tight — do not linger on any single character more than 5 seconds

---

### [3:40 – 4:00] ARCHITECTURE + CLOSE
**Duration:** 20 seconds  
**Recording Mode:** Pre-recorded (architecture diagram) + GCP Console montage

#### Screen Layout — Phase 1 (3:40–3:50)
```
┌─────────────────────────────────────────────────────┐
│                                                     │
│   Architecture Diagram (full screen)                │
│                                                     │
│   ┌──────────┐    ┌──────────────┐    ┌──────────┐ │
│   │  macOS   │    │  Realtime    │    │   ADK    │ │
│   │  Client  │◄──►│  Gateway     │◄──►│  Orch.   │ │
│   │  Swift   │    │  Cloud Run   │    │  Cloud   │ │
│   └──────────┘    └──────────────┘    │  Run     │ │
│                                       │          │ │
│                   ┌──────────────┐    │ 9 Agents │ │
│                   │  Firestore   │◄──►│          │ │
│                   │  Secret Mgr  │    └──────────┘ │
│                   │  Cloud Trace │                  │
│                   └──────────────┘                  │
│                                                     │
└─────────────────────────────────────────────────────┘
```

#### Screen Layout — Phase 2 (3:50–3:58)
```
┌──────────────────────┬──────────────────────────────┐
│                      │                              │
│   VibeCat app        │   GCP Console montage        │
│   Cat waving         │   (rapid cuts, 1.5s each):   │
│                      │   1. Cloud Run — 2 services  │
│                      │   2. Firestore — data        │
│                      │   3. Cloud Trace — waterfall │
│                      │   4. Cloud Monitoring —      │
│                      │      custom metrics dashboard│
│                      │   5. Secret Manager — key    │
│                      │                              │
└──────────────────────┴──────────────────────────────┘
```

#### Actions (Pre-recorded)
1. **[3:40]** Fade in architecture diagram — animate layers appearing left to right
2. **[3:44]** Highlight: "3 layers" text appears
3. **[3:46]** Highlight: "9 agents" text appears, agent names listed
4. **[3:48]** Highlight: GCP services listed (Cloud Run, Firestore, Secret Manager, Trace)
5. **[3:50]** Split screen: cat waving on left, GCP Console rapid montage on right
6. **[3:50]** Cloud Run → both services `RUNNING` (green status)
7. **[3:51.5]** Firestore → sessions collection with data
8. **[3:53]** Cloud Trace → waterfall with agent spans
9. **[3:54.5]** Cloud Monitoring → Metrics Explorer showing `vibecat/agent_duration_ms` histogram + `vibecat/active_sessions` gauge
10. **[3:56]** Secret Manager → `gemini-api-key` (value hidden)
11. **[3:57.5]** Full screen: cat waving, speech bubble: *"같이 코딩해요! 🐱"*
11. **[4:00]** Fade to black. Title card: "VibeCat — Gemini Live Agent Challenge 2026"

#### Narration Script
> *[Confident, closing tone — music swells slightly]*
>
> "Three layers. Nine agents. Fourteen ADK features."
>
> *[Architecture diagram animates]*
>
> "Powered by Gemini Live API, Google ADK — with parallel agents, loop agents, retry-and-reflect self-healing, and BeforeModel callbacks — all deployed on Cloud Run with full observability."
>
> *[GCP Console montage — Cloud Run, Firestore, Trace, Monitoring, Secret Manager]*
>
> "Deployed. Monitored. Proven."
>
> *[Cat waves]*
>
> "The chair is no longer empty."
>
> *[Fade to black]*

#### Subtitle
```
[3:40] "Three layers. Nine agents."
[3:44] "All powered by Gemini Live API, Google ADK, and Cloud Run."
[3:50] "Deployed. Operational. Proven."
[3:56] "The chair is no longer empty."
```

#### GCP Console Focus
- **Cloud Run** → Services → both `realtime-gateway` and `adk-orchestrator` showing green `RUNNING`
- **Firestore** → Data → sessions collection with at least 1 document
- **Cloud Trace** → most recent trace with agent waterfall
- **Cloud Monitoring** → Metrics Explorer → custom metric `custom.googleapis.com/vibecat/agent_duration_ms` showing histogram of 9-agent execution times; also `vibecat/active_sessions` gauge and `vibecat/analysis_total` counter
- **Secret Manager** → `gemini-api-key` secret (value must be hidden/redacted)

#### Director Notes
- Architecture diagram should be a clean, pre-made graphic — not a screenshot
- GCP Console cuts should be rapid (2s each) — this is proof, not explanation
- Cat waving animation is the emotional close — ensure it's the last thing viewers see before fade
- End card should hold for 3 seconds after narration ends
- Music fades out with the end card

---

## PRODUCTION NOTES

### What to Pre-Record vs. Live

| Segment | Mode | Reason |
|---------|------|--------|
| [0:00–0:30] Before | Pre-recorded | Controlled atmosphere, no live risk |
| [0:30–1:00] Launch | Live | Authenticity of zero-onboarding |
| [1:00–1:45] Screen Analysis | Live + spliced Trace | Live error detection, pre-recorded Trace |
| [1:45–2:15] Voice | Live | Must capture real audio interaction |
| [2:15–2:45] Mood/Celebrate | Live + spliced Logging | Live test run, pre-recorded log view |
| [2:45–3:15] Memory/Search | Pre-recorded Firestore + Live search | Firestore pre-populated, search live |
| [3:15–3:40] Characters | Pre-recorded | Controlled, no live risk |
| [3:40–4:00] Architecture | Pre-recorded | Diagram + GCP montage |

### Contingency Plans

| Risk | Mitigation |
|------|-----------|
| Barge-in fails during live take | Pre-record [1:45–2:15] as backup, splice if needed |
| Cloud Trace slow to load | Pre-record trace waterfall, splice at [1:15] |
| Cat sprite doesn't appear | Restart app, have backup recording of launch |
| TTS audio not captured | Route system audio through BlackHole or Loopback |
| Network latency spikes | Record during off-peak hours (early morning KST) |
| GCP Console login required | Stay logged in, disable session timeout before recording |

### Audio Setup
- **Narration**: Record separately in post, sync to video
- **VibeCat TTS**: Captured via system audio routing (BlackHole 2ch recommended)
- **Microphone input**: Directional mic, 30cm distance, no noise cancellation
- **Background music**: Lo-fi, royalty-free, -18dB under narration
- **Music cues**: Fade in at [0:30], energy shift at [3:15] trump segment, swell at [3:40], fade at [4:00]

### Screen Recording Settings
- **Resolution**: 1920×1080 (do not use Retina 2x — file size too large)
- **Frame rate**: 60fps (smooth animations)
- **Software**: OBS Studio or QuickTime (macOS native)
- **Cursor**: Highlight cursor enabled, large size
- **Notifications**: Do Not Disturb ON, all notifications silenced
- **Dock**: Auto-hide disabled (VibeCat must be visible in Dock)

### Font Sizes for Readability
- VS Code: 18pt font, Fira Code or JetBrains Mono
- Terminal: 16pt font, white on black
- GCP Console: Browser zoom 125%
- VibeCat speech bubbles: Ensure legible at 1080p

### Timing Buffer
- Each segment has 1–2 seconds of buffer built in
- Total scripted content: ~3:45
- Buffer for transitions and pauses: ~0:15
- **Do not exceed 4:00** — cut from [3:15–3:40] characters segment if needed (reduce to 15s)

---

## AGENT REFERENCE (for log verification)

| Agent | Log Field | Expected Value |
|-------|-----------|----------------|
| VisionAgent | `agent: "vision_agent"` | `screen_analyzed: true` |
| MoodDetector | `agent: "mood_detector"` | `state: "frustrated\|neutral\|success"` |
| Mediator | `agent: "mediator"` | `should_speak: true\|false` |
| EngagementAgent | `agent: "engagement_agent"` | `trigger: "error_detected"` |
| MemoryAgent | `agent: "memory_agent"` | `memories_retrieved: N` |
| AdaptiveScheduler | `agent: "adaptive_scheduler"` | `cooldown_ms: N` |
| CelebrationTrigger | `agent: "celebration_trigger"` | `fired: true` |
| SearchBuddy | `agent: "search_buddy"` | `grounded: true, sources: N` |
| VAD | `vad: "speech_start\|speech_end"` | `barge_in: true\|false` |

---

## FINAL CHECKLIST (Day of Recording)

- [ ] GCP Console: all 5 tabs open and logged in
- [ ] Terminal A: Gateway log command staged (not yet running)
- [ ] Terminal B: Orchestrator log command staged (not yet running)
- [ ] `/tmp/demo_code.py` written with intentional error
- [ ] `/tmp/demo_test.py` written with passing tests
- [ ] Firestore `sessions/{deviceId}/memories` pre-populated with yesterday's data
- [ ] VibeCat app closed (will launch live at 0:30)
- [ ] Character set to `cat` (default)
- [ ] System audio routed through BlackHole
- [ ] Do Not Disturb: ON
- [ ] Screen recording software: ready, 1920×1080, 60fps
- [ ] Architecture diagram graphic: ready as PNG/SVG
- [ ] Background music track: loaded, cued to start at 0:30
- [ ] Microphone: tested, levels set
- [ ] Practice run: complete at least 2 full dry runs before final take
