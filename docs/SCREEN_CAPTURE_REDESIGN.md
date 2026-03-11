# VibeCat Screen Capture Pipeline Redesign

> Historical note (2026-03-11): this redesign document describes the earlier proactive companion pipeline. The current submission path is the command-driven UI Navigator flow.

**Date**: 2026-03-09
**Status**: Analysis Complete, Pre-Implementation Testing Phase
**Author**: Sisyphus (claude-opus-4-6)

---

## 1. Executive Summary

VibeCat's screen capture pipeline is being redesigned to:
1. **Remove local OCR** (335ms, garbage quality on terminals)
2. **Send images directly to Gemini Live API** via `SendRealtimeInput(Video: *Blob)`
3. **Dynamic resolution** based on display/window characteristics
4. **Dual-path architecture**: Fast Path (Live API) + Smart Path (ADK)

---

## 2. Gemini Image Tokenization Rules (Confirmed)

### 2.1 Tiling Formula

```
if width <= 384 AND height <= 384:
    tokens = 258  (single tile)
else:
    tiles_w = ceil(width / 768)
    tiles_h = ceil(height / 768)
    tokens = tiles_w * tiles_h * 258
```

**Source**: geminibyexample.com/027-calculate-input-tokens (official)

### 2.2 MediaResolution Enum (Go SDK `types.go`)

| Enum | Tokens | Notes |
|------|--------|-------|
| `MEDIA_RESOLUTION_LOW` | 64 | Gemini internal downscale |
| `MEDIA_RESOLUTION_MEDIUM` | 256 | Standard quality |
| `MEDIA_RESOLUTION_HIGH` | 256 | + zoomed reframing for detail |
| Unspecified (default) | 1,120 | Gemini 3.x models |

**Source**: googleapis/go-genai types.go:448-464

### 2.3 Token Cost Table

| Resolution | Aspect | Tiles (W*H) | Tokens | JPEG ~Size | Notes |
|-----------|--------|-------------|--------|-----------|-------|
| 384x216 | 16:9 | 1x1 | 258 | ~8KB | Too small for text |
| 640x480 | 4:3 | 1x1 | 258 | ~30KB | Official example default |
| **768x432** | **16:9** | **1x1** | **258** | **~35KB** | **Optimal 1-tile widescreen** |
| **768x480** | **16:10** | **1x1** | **258** | **~38KB** | **Optimal 1-tile MacBook** |
| 768x768 | 1:1 | 1x1 | 258 | ~50KB | Gemini recommended |
| 769x433 | 16:9 | 2x1 | 516 | ~36KB | 1px over = 2x tokens! |
| **1024x576** | **16:9** | **2x1** | **516** | **~55KB** | **Good text readability** |
| 1280x720 | 16:9 | 2x1 | 516 | ~70KB | HD 720p |
| **1536x864** | **16:9** | **2x2** | **1,032** | **~100KB** | **High-res analysis** |
| 1920x1080 | 16:9 | 3x2 | 1,548 | ~150KB | Full HD |
| 2048x1152 | 16:9 | 3x2 | 1,548 | ~180KB | Current (wasteful) |

**Critical insight**: 768px is the tile boundary. Staying at/below 768px in each dimension = 1 tile = 258 tokens.

---

## 3. Gemini Live API Video Constraints (Confirmed)

### 3.1 Hard Limits

| Aspect | Limit | Source |
|--------|-------|--------|
| Max FPS | **1 FPS** (server-enforced) | Firebase Live API Limits |
| Recommended resolution | **768x768** native | Firebase Live API Limits |
| Session (video+audio) | **2 minutes** (without compression) | Firebase Live API Limits |
| Session (audio-only) | **15 minutes** | Firebase Live API Limits |
| Context window | **128k tokens** | Firebase Live API Limits |
| Supported format | `image/jpeg` | go-genai SDK |

### 3.2 ContextWindowCompression

Available to extend session lifetime beyond 2 minutes:

```go
ContextWindowCompression: &genai.ContextWindowCompressionConfig{
    TriggerTokens: genai.Ptr[int64](100000),
    SlidingWindow: &genai.SlidingWindow{
        TargetTokens: genai.Ptr[int64](50000),
    },
}
```

- Sliding window evicts oldest tokens (including old video frames)
- System instructions + prefix_turns are preserved
- "Window shortening has some latency costs, avoid on every turn"

### 3.3 SDK Behavior

- `SendRealtimeInput` sends immediately via WebSocket (no client queuing)
- Only ONE of Media/Audio/Video/Text per call
- Non-deterministic ordering between audio and video
- Audio VAD drives responses; video frames add context but don't trigger responses
- "Interleaving SendClientContent and SendRealtimeInput is NOT recommended"

### 3.4 Critical Architecture Implication

**Video+Audio = 2 min session limit** means:
- Cannot send video frames continuously at 1Hz for long sessions
- Must use ContextWindowCompression (sliding window) for extended sessions
- OR send video frames sparingly (1 frame per 5-10 seconds)
- ADK Smart Path remains essential for long-running proactive analysis

---

## 4. Current Pipeline (Problems)

```
Client (1Hz):
  ScreenCaptureKit -> CGImage (display.width * 2, always retina 2x)
  -> ImageDiffer (32x32 thumbnail, ~1ms) -> changed?
  -> YES: OCR (335ms, garbage quality) -> TextContextBuffer -> sendToGateway
         base64 JPEG ~538KB + OCR text
  -> NO: skip

Gateway (handler.go):
  -> if modelSpeaking && type == "screenCapture": SKIP
  -> else: goroutine -> ADK /analyze (30s timeout)

ADK:
  -> VisionAgent -> Gemini 3.1 Flash Lite (genai.InlineData blob)
  -> JSON analysis (significance, shouldSpeak, content, emotion)
  -> shouldSpeak? -> companionSpeech -> client (bubble+TTS)
  -> Always: sess.SendText("[Screen Context] content") -> Gemini Live API
```

### Problems

| Problem | Impact |
|---------|--------|
| Local OCR 335ms, garbage quality | Wasted CPU, useless data |
| Always 2x retina capture | 5120x3200 -> 2048px -> 1,548 tokens always |
| Sends every significant change | Gemini generates new response -> interrupts speech |
| Client ignores audioActive | Wasteful transmission during speech |
| No display-adaptive resolution | Same cost for 1080p and 4K monitors |
| OCR context is redundant | Gemini VisionAgent reads image directly |
| 4 hops to screen understanding | Client->WS->HTTP->Gemini, ~3-5s latency |

---

## 5. Proposed Architecture: Dual-Path

```
Client (1Hz capture loop):
  ScreenCaptureKit -> CGImage
  -> ImageDiffer (32x32, ~1ms) -> changed?
  -> YES: Store in frameBuffer + metadata
  -> NO: skip
  * NO OCR. NO gateway send from capture loop.

Client (Send Strategy):
  +--> Fast Path (Live API Video):
  |    When: change detected + not audioActive + cooldown(2s) elapsed
  |    Resolution: 768x{aspect} (1 tile, 258 tokens)
  |    Format: raw JPEG bytes via binary WebSocket
  |    Gateway: sess.SendRealtimeInput(Video: *Blob)
  |    Gemini Live sees the screen natively alongside audio
  |
  +--> Smart Path (ADK Analysis):
       When: significant change + cooldown(5-10s) + not audioActive
       Resolution: DPI-adaptive (1024-1536px, 516-1032 tokens)
       Format: base64 JPEG via JSON WebSocket
       Gateway: ADK /analyze -> 9-agent graph
       -> shouldSpeak? -> proactive companionSpeech
       -> screen context injected to Live via SendText

Client (Immediate):
  User says "describe screen" -> forceAnalysis()
  -> Latest frame from frameBuffer
  -> Both Fast Path + Smart Path simultaneously
```

### 5.1 Fast Path Details

- **Purpose**: Gemini Live has real-time screen awareness
- **Resolution**: Always 768x{aspect} (1 tile = 258 tokens)
- **FPS**: ~1Hz but throttled by change detection + cooldowns
- **Session management**: ContextWindowCompression enabled
- **Token budget**: ~258 tokens/frame * ~30 frames/min = ~7,740 tokens/min
- **Interaction**: Gemini naturally references screen in voice responses

### 5.2 Smart Path Details

- **Purpose**: Proactive AI behavior (error detection, celebrations, mood)
- **Resolution**: DPI-adaptive (see section 6)
- **Frequency**: Every 5-10 seconds when significant changes
- **9-agent graph**: Vision -> Mediator -> Engagement -> Celebration -> Mood
- **Required**: Challenge judging criteria demands all 9 agents

### 5.3 Resolution Strategy

```
Fast Path (Live API): Always 1 tile
  16:9 display -> 768x432
  16:10 display -> 768x480

Smart Path (ADK): DPI-adaptive
  Retina (scale >= 2): 1536x864 (2x2 tiles, 1,032 tokens)
  Non-Retina (scale 1): 1024x576 (2x1 tiles, 516 tokens)
  Small window (<1000px): 768x432 (1x1 tile, 258 tokens)
```

---

## 6. Dynamic Resolution Implementation

### 6.1 Capture at Target Resolution (SCStreamConfiguration)

Instead of capturing at 2x retina then downscaling in ImageProcessor,
set SCStreamConfiguration directly to target resolution:

```swift
func optimalCaptureSize(for display: SCDisplay, path: CapturePathType) -> (width: Int, height: Int) {
    let screen = NSScreen.screens.first { /* mouse display */ }
    let scale = screen?.backingScaleFactor ?? 2.0
    let aspectRatio = CGFloat(display.width) / CGFloat(display.height)
    let is16by10 = abs(aspectRatio - 1.6) < 0.05

    switch path {
    case .fastPath:
        return is16by10 ? (768, 480) : (768, 432)
    case .smartPath:
        if scale >= 2.0 {
            return is16by10 ? (1536, 960) : (1536, 864)
        } else {
            return is16by10 ? (1024, 640) : (1024, 576)
        }
    }
}
```

### 6.2 Benefits

- No double processing (capture big -> downscale)
- SCStreamConfiguration does hardware-level scaling
- Exact tile boundary alignment -> no wasted tokens
- Smaller capture -> faster JPEG encoding -> less memory

---

## 7. Gateway Changes Required

### 7.1 New: videoFrame Binary Handler

```go
// Binary WebSocket message with type prefix byte
case websocket.BinaryMessage:
    if len(data) > 0 && data[0] == 0x01 { // Video frame prefix
        jpegData := data[1:]
        if sess := ls.getSession(); sess != nil {
            sess.SendRealtimeInput(genai.LiveRealtimeInput{
                Video: &genai.Blob{
                    Data:     jpegData,
                    MIMEType: "image/jpeg",
                },
            })
        }
    } else { // Audio (existing)
        // ... existing audio handling
    }
```

### 7.2 LiveConnectConfig Changes

```go
// Add to session setup
config.MediaResolution = genai.MediaResolutionHigh  // 256 tokens + zoom
config.ContextWindowCompression = &genai.ContextWindowCompressionConfig{
    TriggerTokens: genai.Ptr[int64](100000),
    SlidingWindow: &genai.SlidingWindow{
        TargetTokens: genai.Ptr[int64](50000),
    },
}
```

### 7.3 Existing: screenCapture Handler (Smart Path, unchanged)

Existing ADK analysis flow remains for proactive behavior.
Add audioActive guard and minimum 5s cooldown.

---

## 8. Client Changes Required

### 8.1 ScreenCaptureService

- Add `captureAtResolution(width:height:)` method
- Use `optimalCaptureSize()` for dynamic resolution
- Remove hardcoded `display.width * 2`

### 8.2 ScreenAnalyzer (Dual-Path)

- Remove all OCR references
- Add frameBuffer (ring buffer of last 10 CGImages)
- Fast Path: send JPEG bytes via binary WebSocket
- Smart Path: send base64 JPEG via JSON WebSocket (existing)
- Add audioActive guard + cooldowns
- Add app switch detection (existing, keep)

### 8.3 GatewayClient

- Add `sendVideoFrame(jpegData: Data)` for binary video frames
- Keep existing `sendScreenCapture()` for Smart Path

### 8.4 Settings

- Remove captureInterval (now implicit at 1Hz)
- Add captureQuality enum (low/medium/high) if needed

---

## 9. Session Lifetime Management

### 9.1 The 2-Minute Problem

With video+audio, sessions are limited to ~2 min context window.
At 258 tokens/frame * 1 FPS = 15,480 tokens/min.
128k token context / 15,480 = ~8.3 minutes before exhaustion.

But audio also consumes tokens (32 tokens/second = 1,920/min).
Combined: ~17,400 tokens/min -> 128k / 17,400 = ~7.4 minutes.

### 9.2 Solution: ContextWindowCompression

Configure sliding window:
- TriggerTokens: 100,000 (trigger at ~78% capacity)
- TargetTokens: 50,000 (compress to ~39% capacity)
- This evicts oldest ~50k tokens of audio+video
- System instructions preserved

### 9.3 Smart Throttling

Don't send video every second. Strategy:
- Send 1 frame after change detected + 2s cooldown
- Max ~1 frame per 3-5 seconds during active use
- ~0 frames when screen unchanged
- Actual token budget: ~258 * 12 frames/min = ~3,096 tokens/min
- Combined with audio: ~5,016 tokens/min -> 128k / 5,016 = ~25 minutes

---

## 10. Testing Plan

### 10.1 Pre-Implementation Verification (10+ tests)

1. **Resolution/quality tests**: Capture at each tier, verify Gemini text readability
2. **Latency tests**: Measure SendRealtimeInput(Video) -> response time
3. **MediaResolution comparison**: LOW vs MEDIUM vs HIGH quality
4. **ContextWindowCompression**: Verify session extends beyond 2 minutes
5. **Concurrent audio+video**: Verify no interference
6. **Frame drop behavior**: Send faster than 1 FPS, verify graceful handling
7. **Different display types**: Retina, non-retina, 4K, ultrawide
8. **Text readability at 768px**: Screenshot of code editor at 768x432
9. **App switch detection**: Verify immediate frame on Cmd+Tab
10. **Audio guard**: Verify no sends during speech playback

### 10.2 Test Program Structure

Go test in `backend/realtime-gateway/cmd/videotest/` that:
1. Connects to Gemini Live API
2. Sends a screenshot at various resolutions
3. Asks "What do you see on screen?" via audio or text
4. Measures time to first response token
5. Evaluates response quality (did it read text correctly?)

---

## 11. Risk Assessment

| Risk | Mitigation |
|------|-----------|
| 2-min session limit with video | ContextWindowCompression + smart throttling |
| 768px too small for code | Test first; fallback to 1024px (516 tokens) |
| Frame drops at server | 1 FPS limit + change detection = well under limit |
| SendRealtimeInput ordering | Non-deterministic is fine for screen context |
| Mixed SendClientContent + SendRealtimeInput | Already using SendRealtimeInput for audio; keep same pattern |
| Token budget overflow | Monitor with usageMetadata; auto-throttle |

---

## 12. References

- Go GenAI SDK: github.com/googleapis/go-genai (types.go, live.go)
- Firebase Live API Limits: firebase.google.com/docs/ai-logic/live-api/limits-and-specs
- Image tokenization: geminibyexample.com/027-calculate-input-tokens
- Gemini Live API examples: github.com/google-gemini/gemini-live-api-examples
- Google Cloud docs: docs.cloud.google.com/vertex-ai/generative-ai/docs/live-api
