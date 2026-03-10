# TTS Pipeline Implementation Plan

**Created:** 2026-03-05
**Status:** Ready for review
**Scope:** Add streaming TTS pipeline using `gemini-2.5-flash-preview-tts` via Go SDK

## Problem

`sendTextAndTriggerAudio()` in `backend/realtime-gateway/internal/live/session.go:111-123` sends text to the Live API's `gemini-2.5-flash-native-audio-latest` model, but this model ONLY generates audio from AUDIO input. Text input produces "thinking" text responses, not audio. The companion can analyze the screen and generate speech text, but CANNOT speak it.

## Architecture: Before vs After

```
BEFORE (broken):
  ADK SpeechText → session.sendTextAndTriggerAudio()
                    → Live API SendRealtimeInput(text)  ← WRONG: native-audio model ignores text
                    → silence trigger
                    → returns thinking text, NOT audio

AFTER (working):
  ADK SpeechText → tts.Client.StreamSpeak(ctx, config, audioSink)
                    → GenAI GenerateContentStream("gemini-2.5-flash-preview-tts")
                    → PCM 24kHz 16-bit mono audio chunks
                    → WebSocket BinaryMessage → AudioPlayer.enqueue() → speaker
                 → session.SendText("[COMPANION_CONTEXT] ...") [context only, no response]

  Echo Prevention:
    Gateway sends {"type":"ttsStart"} → GatewayClient.isTTSSpeaking = true → sendAudio() suppressed
    Gateway sends {"type":"ttsEnd"}   → GatewayClient.isTTSSpeaking = false → mic resumes
    User barge-in: binary audio arrives → cancelTTS() → forced ttsEnd sent → mic resumes
```

## Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| TTS vs Live API | Separate TTS API call | Live API native-audio model doesn't support text→audio |
| Streaming vs Unary | `GenerateContentStream` | Sub-5s first-chunk latency requirement |
| Audio coordination | TTS cancellation on user voice input | User barge-in interrupts TTS naturally |
| Client changes | Minimal (~20 lines) | Binary format is identical (24kHz PCM) |
| Context injection | `SendText()` without silence trigger | Live API gets context but doesn't respond |
| Timeout mechanism | `context.WithTimeout()` | SDK bug #689: `HTTPOptions.Timeout` drops streaming chunks |
| genai.Client sharing | Shared between Live and TTS | Thread-safe, confirmed by SDK source |

## Dependency Graph

```
Wave 1  (Parallel — no dependencies)
├── T1: Create backend/realtime-gateway/internal/tts/client.go
└── T2: Create backend/realtime-gateway/internal/tts/client_test.go

Wave 2  (Sequential — depends on Wave 1, atomic change)
└── T3: Backend integration wiring
     ├── T3a: Update backend/realtime-gateway/main.go
     ├── T3b: Update backend/realtime-gateway/internal/ws/handler.go
     └── T3c: Update backend/realtime-gateway/internal/live/session.go

Wave 3  (Parallel — depends on Wave 2)
├── T4: Go build verification (go build && go test && go vet)
└── T5: Swift client echo prevention
     ├── T5a: Update VibeCat/Sources/Core/AudioMessageParser.swift
     ├── T5b: Update VibeCat/Sources/VibeCat/GatewayClient.swift
     └── T5c: Update VibeCat/Sources/VibeCat/AppDelegate.swift

Wave 4  (Depends on Wave 3)
└── T6: Swift build verification (swift build)
```

## T1: NEW — `backend/realtime-gateway/internal/tts/client.go`

**~100 lines**

```go
package tts

// Client struct:
//   genai  *genai.Client  — shared with Live API (thread-safe)
//   model  string         — "gemini-2.5-flash-preview-tts"

// Config struct:
//   Voice    string  — "Zephyr", "Puck", etc. (from session config)
//   Language string  — "ko-KR" for Korean
//   Text     string  — the text to speak

// AudioSink = func(chunk []byte) error

// StreamSpeak(ctx, Config, AudioSink) error
//   1. context.WithTimeout(ctx, 15s)  — NOT HTTPOptions.Timeout (SDK bug #689)
//   2. Build GenerateContentConfig:
//      - ResponseModalities: []string{"AUDIO"}
//      - SpeechConfig with voice + language code
//   3. Range over GenerateContentStream iterator
//   4. Extract InlineData.Data from each chunk
//   5. Call sink(chunk) for each non-empty chunk
//   6. Log first-chunk latency, total bytes, voice, text length
//   7. Return nil on success, wrapped error on failure

// BuildConfig(Config) *genai.GenerateContentConfig  — exported for testing
// normalizeLanguageCode(string) string — "Korean"→"ko-KR", "English"→"en-US"
```

**Critical**: `GenerateContentConfig.ResponseModalities` is `[]string`, NOT `[]genai.Modality` (different from `LiveConnectConfig`).

## T2: NEW — `backend/realtime-gateway/internal/tts/client_test.go`

**~80 lines**

| Test | Validates |
|------|-----------|
| `TestBuildConfig_WithVoice` | VoiceName set correctly |
| `TestBuildConfig_DefaultVoice` | Empty voice → "Zephyr" |
| `TestBuildConfig_ResponseModalities` | Always `[]string{"AUDIO"}` |
| `TestBuildConfig_LanguageCode` | Korean variants → "ko-KR" |
| `TestNormalizeLanguageCode` | Table-driven all variants |
| `TestNewClient_NilGuard` | nil input → nil output |

## T3a: MODIFY — `backend/realtime-gateway/main.go`

**+8 lines**

1. Add import `"vibecat/realtime-gateway/internal/tts"`
2. Create TTS client from shared genaiClient (after liveMgr creation)
3. Pass ttsClient to Handler

## T3b: MODIFY — `backend/realtime-gateway/internal/ws/handler.go`

**+70/-25 lines** — largest change

1. Handler signature: add `ttsClient *tts.Client` parameter
2. `liveSessionState`: add `ttsMu sync.Mutex`, `ttsCancel context.CancelFunc`, helper methods
3. Barge-in: `ls.cancelTTS()` on BinaryMessage receipt
4. Replace companion speech: context injection + `startTTSStream()` helper
5. Replace search result: same TTS pattern
6. New `startTTSStream()` function: sends ttsStart → streams audio chunks → sends ttsEnd (defer guarantees)
7. Emotion tag prepending from ADK analysis result

## T3c: MODIFY — `backend/realtime-gateway/internal/live/session.go`

**-27 lines**

Remove:
- `SendCompanionSpeech()` (lines 97-102)
- `SendSearchResult()` (lines 104-109)
- `sendTextAndTriggerAudio()` (lines 111-123)

Keep: `SendAudio()`, `SendText()`, `Receive()`, `Close()`, `buildLiveConfig()`, `normalizeLanguage()`

## T5: Swift Client Echo Prevention

### T5a: `VibeCat/Sources/Core/AudioMessageParser.swift` (+4 lines)
- Add `.ttsStart` and `.ttsEnd` to ServerMessage enum
- Add parse cases in switch

### T5b: `VibeCat/Sources/VibeCat/GatewayClient.swift` (+12 lines)
- Add `isTTSSpeaking: Bool` property
- Guard `sendAudio()` with `!isTTSSpeaking`
- Handle ttsStart/ttsEnd in message handler

### T5c: `VibeCat/Sources/VibeCat/AppDelegate.swift` (+6 lines)
- Handle ttsStart: `catVoice?.stop()` to clear queued audio
- Handle ttsEnd: no-op (audio plays naturally)

## File Inventory

| # | File | Action | Lines | Wave |
|---|------|--------|-------|------|
| 1 | `backend/realtime-gateway/internal/tts/client.go` | NEW | ~100 | 1 |
| 2 | `backend/realtime-gateway/internal/tts/client_test.go` | NEW | ~80 | 1 |
| 3 | `backend/realtime-gateway/main.go` | MODIFY | +8 | 2 |
| 4 | `backend/realtime-gateway/internal/ws/handler.go` | MODIFY | +70/-25 | 2 |
| 5 | `backend/realtime-gateway/internal/live/session.go` | MODIFY | -27 | 2 |
| 6 | `VibeCat/Sources/Core/AudioMessageParser.swift` | MODIFY | +4 | 3 |
| 7 | `VibeCat/Sources/VibeCat/GatewayClient.swift` | MODIFY | +12 | 3 |
| 8 | `VibeCat/Sources/VibeCat/AppDelegate.swift` | MODIFY | +6 | 3 |

**Total: 2 new files, 6 modified files, ~280 LOC**

## Test Strategy

| Level | Command | What |
|-------|---------|------|
| Unit | `go test ./internal/tts/...` | Config builder, language normalization |
| Unit | `go test ./internal/live/...` | Existing session tests pass |
| Unit | `go test ./internal/ws/...` | Existing handler tests pass |
| Build | `go build ./...` | Go compilation |
| Vet | `go vet ./...` | Static analysis |
| Build | `swift build` | Swift compilation |

## Risk Assessment

| Risk | Severity | Mitigation |
|------|----------|------------|
| SDK bug #689 drops streaming chunks | Critical | `context.WithTimeout()` instead of HTTPOptions |
| Audio interleave TTS + Live API | Medium | Clear queue on ttsStart + barge-in cancel |
| Echo loop speaker → mic | Medium | `isTTSSpeaking` flag suppresses sendAudio() |
| TTS preview model changes | Medium | Model name is constant, easy swap |
| First-chunk latency > 5s | Medium | Streaming + 15s timeout |
| Context loss in Live API | Low | `SendText("[COMPANION_CONTEXT]...")` |

## Verification Plan

### Automated
```bash
cd backend/realtime-gateway && go build ./... && go test ./... && go vet ./...
cd VibeCat && swift build
```

### Manual E2E Checklist
1. Start gateway → "tts client initialized" in logs
2. Connect WebSocket, send setup → setupComplete
3. Trigger screen analysis → companionSpeech JSON → ttsStart → binary chunks → ttsEnd
4. Time first audio chunk < 5 seconds
5. Send mic audio during TTS → TTS cancelled, ttsEnd sent
6. Verify voice conversation still works after TTS
7. Trigger voice search → search result spoken via TTS
