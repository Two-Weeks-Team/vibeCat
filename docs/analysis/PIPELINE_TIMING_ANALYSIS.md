# VibeCat Pipeline Timing Analysis

## Revision History

| Date | Status | Key Change |
|------|--------|------------|
| 2026-03-05 | Initial | Live API TTS dead, vision-only pipeline measured |
| **2026-03-06** | **Current** | **TTS pipeline working, bubble-voice sync fixed, full E2E measured** |

---

## 2026-03-06 Analysis (Current)

**Measured from:** Real runtime logs (10:42:16 ~ 10:42:25 KST)
**Character:** cat (Zephyr) | **Active App:** Google Chrome
**TTS Model:** gemini-2.5-flash-preview-tts
**Live Model:** gemini-2.5-flash-native-audio-latest
**Vision Model:** gemini-3.1-flash-lite-preview

### Architecture Flow (Current)

```
Timer(5s) → ScreenCapture → ImageDiffer → WebSocket → Gateway → ADK(9 agents)
  → companionSpeech JSON (bubble) + ttsStart JSON (bubble hold)
  → TTS API call → PCM chunks → ttsEnd JSON (bubble release)
```

### Measured Timings (10:42:16 Event)

| Step | Component | Timestamp | Delta | % of Total |
|------|-----------|-----------|-------|------------|
| 1 | ImageDiffer (change detection) | 10:42:16.240 | — | — |
| 2 | JPEG encode + base64 + WS send | 10:42:16.358 | +0.12s | 1.4% |
| 3 | Gateway receive + ADK POST start | 10:42:16.360 | +0.002s | <0.1% |
| **4** | **ADK 9-agent analysis** | **10:42:20.701** | **+4.34s** | **49.3%** |
| 5 | companionSpeech → client bubble | 10:42:20.702 | +0.001s | <0.1% |
| 6 | ttsStart → client (turnActive) | 10:42:20.703 | +0.001s | <0.1% |
| **7** | **TTS API → first audio chunk** | **10:42:25.067** | **+4.36s** | **49.5%** |
| 8 | ttsEnd | 10:42:25.070 | +0.003s | <0.1% |

### Summary

| Metric | Value |
|--------|-------|
| **Capture → Bubble (text visible)** | **~4.5s** |
| **Capture → First Audio (voice heard)** | **~8.8s** |
| **Bubble → First Audio gap** | **~4.4s** |
| ADK 9-agent analysis | ~4.3s (49%) |
| TTS API response | ~4.4s (50%) |
| Network/framework overhead | ~0.1s (1%) |
| TTS audio payload | 230,926 bytes (single chunk) |

### Pipeline Breakdown

```
T+0.00s  Screen capture (ImageDiffer comparison)
T+0.12s  WebSocket send to Gateway
T+0.12s  Gateway → ADK POST /analyze
         ├── VisionAgent (Gemini 3.1 Flash Lite)  ~4.0s
         ├── MoodDetector                          <1ms
         ├── CelebrationTrigger                    <1ms
         ├── Mediator (speak decision)             <1ms
         ├── AdaptiveScheduler                     <1ms
         └── EngagementAgent                       <1ms
T+4.46s  ADK response → companionSpeech JSON
T+4.46s  ★ BUBBLE SHOWN (user sees text)
T+4.46s  ttsStart JSON → turnActive=true (bubble held)
T+4.46s  TTS API call start (gemini-2.5-flash-preview-tts)
T+8.83s  TTS audio arrives (230KB PCM, single chunk)
T+8.83s  ★ VOICE PLAYS (user hears audio)
T+8.83s  ttsEnd JSON → turnActive=false
T+~13s   Audio playback finishes
T+~15s   Bubble hides (2s after audio ends, KDC pattern)
```

### Bottleneck Analysis

| Bottleneck | Time | Cause | Optimization Path |
|------------|------|-------|-------------------|
| ADK 9-agent | 4.3s | VisionAgent REST call dominates | Parallel agent execution, smaller image |
| TTS cold start | 4.4s | First call to gemini-2.5-flash-preview-tts | Warm-up call on connect, streaming chunks |
| Serial execution | 8.8s total | ADK must complete before TTS starts | Cannot parallelize (need speech text first) |

### Bubble-Voice Sync Status

| Behavior | Before Fix | After Fix (KDC pattern) |
|----------|-----------|------------------------|
| Bubble shows on companionSpeech | Yes | Yes |
| Bubble stays during TTS | No (2s auto-hide) | Yes (turnActive=true) |
| Bubble stays during audio playback | No | Yes (audioPlayer.isPlaying) |
| Bubble hides after audio ends | N/A (already gone) | Yes (+2s delay) |
| Voice without bubble | **Frequent** | **Fixed** |

### Log Evidence

**Gateway:**
```
10:42:16.359  websocket text frame type=screenCapture
10:42:16.360  [HANDLER] >>> ADK analyze request
10:42:20.701  [ADK-CLIENT] <<< response OK (elapsed=4.341s)
10:42:20.701  [HANDLER] >>> sending companionSpeech to client
10:42:25.067  [TTS] first chunk (latency=4.365s)
10:42:25.067  [TTS] stream complete (bytes=230926)
```

**Client:**
```
10:42:16.357  [CAPTURE] sendToGateway
10:42:20.702  [GW-IN] companionSpeech
10:42:20.702  [BUBBLE] showBubble: 야옹~ 다시...
10:42:20.703  [GW-IN] ttsStart, hasText=1
10:42:20.703  [BUBBLE] showBubble: 야옹~ 다시... (refresh, turnActive=true)
10:42:25.067  [GW-IN] onAudioData: 230926 bytes
10:42:25.070  [GW-IN] ttsEnd
```

---

## 2026-03-05 Analysis (Historical)

**Measured from:** Real runtime logs (13:39:43 ~ 13:39:52 KST)
**Character:** saja | **Active App:** Terminal
**Status:** TTS was broken (Live API session dead)

### Measured Timings

| Step | Component | Measured | Status |
|------|-----------|----------|--------|
| Screen capture + change detection | ~216ms | OK |
| JPEG encode + base64 (6016x3384 → 443KB) | ~105ms | OK |
| WebSocket send (Client → Gateway) | ~7ms | OK |
| **Vision Agent (Gemini 3.1 Flash Lite REST)** | **~2480ms** | **BOTTLENECK** |
| Local agents (Mood+Celeb+Mediator+Sched+Engage+Search) | ~3ms | OK |
| companionSpeech to client | <1ms | OK |
| **Live API TTS** | **DEAD** | **CRITICAL BUG** |

**Total Capture → ADK Decision:** ~2.8s (vision only, no TTS)

### Issues (All Resolved as of 2026-03-06)

1. ~~Live API Session Dead~~ → Fixed: Independent TTS pipeline via gemini-2.5-flash-preview-tts
2. ~~No audio output~~ → Fixed: TTS streaming with barge-in support
3. ~~Bubble-voice desync~~ → Fixed: KDC-pattern turnActive/audioPlayer gating
