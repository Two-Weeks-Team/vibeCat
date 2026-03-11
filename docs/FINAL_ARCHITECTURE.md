# VibeCat — Final Architecture Document

**Last Reviewed:** 2026-03-11
**Repo HEAD:** `7356f8c`
**Status:** Architecture reference. For live deployment, CI, and open-issue truth, use `docs/CURRENT_STATUS_20260311.md` and the deployment evidence docs.

> Snapshot note: this file is a detailed architecture reference, not the authoritative source for fast-changing operational values. Deployment revisions, CI outcomes, and issue triage should be read from the current-status and evidence documents first.

---

## Table of Contents

1. [System Overview](#1-system-overview)
2. [Three-Layer Architecture](#2-three-layer-architecture)
3. [Layer 1: macOS Swift Client](#3-layer-1-macos-swift-client)
4. [Layer 2: Realtime Gateway](#4-layer-2-realtime-gateway)
5. [Layer 3: ADK Orchestrator](#5-layer-3-adk-orchestrator)
6. [9-Agent Graph](#6-9-agent-graph)
7. [Data Flow](#7-data-flow)
8. [Character System](#8-character-system)
9. [Infrastructure & Deployment](#9-infrastructure--deployment)
10. [Configuration Reference](#10-configuration-reference)
11. [File Inventory](#11-file-inventory)

---

## 1. System Overview

VibeCat is a **macOS desktop AI companion for solo developers** — an animated character that watches your screen, hears your voice, remembers context across sessions, and proactively helps. Built for the [Gemini Live Agent Challenge 2026](https://geminiliveagentchallenge.devpost.com/).

**Required Stack:** GenAI SDK + Google ADK + Gemini Live API + VAD (all four mandatory)

```
┌──────────────────────────────────────────────────────────────────────────┐
│                          VibeCat System                                  │
│                                                                          │
│  ┌─────────────┐    WebSocket     ┌──────────────┐    HTTP POST         │
│  │ macOS Client │◄──────────────►│   Realtime    │──────────────►┌─────┐│
│  │  (Swift 6)   │   PCM + JPEG   │   Gateway    │   /analyze    │ ADK ││
│  │              │                 │  (Go+GenAI)  │               │Orch.││
│  └──────┬───────┘                 └──────┬───────┘               └──┬──┘│
│         │                                │                          │   │
│    Screen Capture                   Live API                   Firestore│
│    Voice Input                    TTS Streaming              9 Agents  │
│    UI Overlay                     VAD Config                 Memory    │
│    Animation                      Session Resumption         Search   │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Three-Layer Architecture

| Layer | Technology | Location | Port | Role |
|-------|-----------|----------|------|------|
| **macOS Client** | Swift 6, SwiftUI, SPM | `VibeCat/` | — | UI, screen capture, audio I/O, gestures |
| **Realtime Gateway** | Go 1.24 + GenAI SDK v1.48 | `backend/realtime-gateway/` | 8080 | WebSocket proxy to Gemini Live API |
| **ADK Orchestrator** | Go 1.24 + ADK Go SDK | `backend/adk-orchestrator/` | 8080 | 9-agent graph, Firestore persistence |

**Non-Negotiable Rules:**
- ALL model calls go through backend — client never touches Gemini API directly
- API key lives in GCP Secret Manager — never on client
- Client does UI/capture/playback only
- Prompt logic is server-side only

---

## 3. Layer 1: macOS Swift Client

### Module Structure (Package.swift)

```
VibeCat (executable)
  └── depends on: VibeCatCore (library)

VibeCatCore (library)
  └── Pure Swift/Foundation — no UI dependencies
```

### Core Module (`Sources/Core/`)

| File | Role |
|------|------|
| `Core.swift` | Module entry point, shared type exports |
| `Settings.swift` | `AppSettings` singleton, UserDefaults persistence |
| `Models.swift` | `ChatMessage`, `CharacterPresetConfig`, `CompanionEmotion`, `CompanionMood`, `CompanionSpeechEvent` |
| `AudioMessageParser.swift` | WebSocket message parsing, `ServerMessage` enum (17 message types), emotion tag parsing |
| `ImageProcessor.swift` | CGImage → JPEG encoding, resizing, base64 conversion |
| `ImageDiffer.swift` | Pixel-level screen change detection using 32×32 thumbnails (~1ms) |
| `PCMConverter.swift` | Audio format conversions: Int16 ↔ Float32, bytes ↔ samples |
| `KeychainHelper.swift` | Secure API key storage via Keychain Services |

### App Module (`Sources/VibeCat/`)

| File | Lines | Role |
|------|-------|------|
| `VibeCat.swift` | — | App entry point, singleton enforcement via `flock()` |
| `AppDelegate.swift` | 597 | **Main orchestrator** — wires all components, handles lifecycle, message routing |
| `GatewayClient.swift` | 686 | WebSocket client, reconnection with circuit breaker, session resumption |
| `ScreenAnalyzer.swift` | — | Dual-path capture: Fast Path (Live API) + Smart Path (ADK Orchestrator) |
| `ScreenCaptureService.swift` | — | ScreenCaptureKit wrapper, cursor-region capture, multi-monitor |
| `CatPanel.swift` | 331 | Borderless overlay window, sprite rendering, chat bubble, emotion indicators |
| `CatViewModel.swift` | 120 | 60 FPS mouse tracking, screen bounds, home position management |
| `SpriteAnimator.swift` | 237 | Animation state machine, idle behavior, preset loading |
| `ChatBubbleView.swift` | — | Custom NSView, speech bubble rendering, spring animations |
| `SpeechRecognizer.swift` | — | AVAudioEngine mic capture, VAD with barge-in, PCM16 conversion |
| `AudioPlayer.swift` | — | AVAudioEngine PCM playback with ~20ms buffer coalescing |
| `CatVoice.swift` | — | Thin wrapper over AudioPlayer for voice output |
| `CircleGestureDetector.swift` | — | Mouse circle gesture (3 circles in 6s → screen analysis trigger) |
| `StatusBarController.swift` | 608 | Menu bar UI, settings, session tracking, speech/emotion history |
| `TrayIconAnimator.swift` | — | Menu bar icon animation (8 frames) |
| `BackgroundMusicPlayer.swift` | — | Lo-fi background music playback |
| `CompanionChatPanel.swift` | — | Chat UI panel for text interaction |
| `OnboardingWindowController.swift` | — | API key entry dialog with Keychain |
| `ErrorReporter.swift` | — | Global error reporting singleton |
| `DecisionOverlayHUD.swift` | — | Debug HUD for agent decision visibility |

### Client Callback Architecture

Components communicate via closures (no delegate protocols):

```
AppDelegate (central wiring)
  ├── GatewayClient.onMessage → routes ServerMessage to UI
  ├── GatewayClient.onAudioData → AudioPlayer.play()
  ├── GatewayClient.onStateChange → StatusBar/ErrorReporter
  ├── SpeechRecognizer.onAudioBufferCaptured → GatewayClient.sendAudio()
  ├── ScreenAnalyzer.onSpeechEvent → CatPanel.showBubble()
  ├── ScreenAnalyzer.onBackgroundSpeech → notification
  ├── CatViewModel.onPositionUpdate → CatPanel.updatePosition()
  ├── CircleGestureDetector.onCircleGesture → ScreenAnalyzer.forceCapture()
  └── CompanionChatPanel.onTextSubmitted → GatewayClient.sendText()
```

### Concurrency Model

| Pattern | Where | Purpose |
|---------|-------|---------|
| `@MainActor` | All UI classes | Thread-safe UI access |
| `MainActor.assumeIsolated` | Timer callbacks | Swift 6 compliance for RunLoop.main timers |
| `DispatchQueue.global(qos: .userInitiated)` | ScreenAnalyzer | JPEG/base64 encoding off main thread |
| `DispatchQueue(label: "vibecat.audio.conversion")` | AppDelegate | Audio format conversion |
| `NWPathMonitor` + dedicated queue | GatewayClient | Network reachability |
| `NSLock` | SpeechRecognizer | Thread-safe `modelSpeaking` flag (audio tap thread) |
| `Timer` + `RunLoop.main.add(.common)` | CatViewModel, SpriteAnimator, CatPanel | 60fps animation (survives tracking mode) |
| `withCheckedContinuation` | ScreenAnalyzer | Bridge GCD to async/await |

### External Framework Dependencies

| Framework | Usage |
|-----------|-------|
| AppKit | NSWindow, NSPanel, NSMenu, NSImage |
| AVFoundation | Audio engine, capture, playback |
| ScreenCaptureKit | Screen recording via SCScreenshotManager |
| CoreGraphics | CGImage, display handling |
| Network | NWPathMonitor for reachability |
| UserNotifications | macOS notification center |
| Security | Keychain Services (via Core module) |
| ImageIO | JPEG encoding via CGImageDestination (via Core module) |

---

## 4. Layer 2: Realtime Gateway

### Package Structure

```
backend/realtime-gateway/
├── main.go                 # Entry: Cloud Trace, Cloud Logging, JWT, GenAI client, HTTP routes
├── internal/
│   ├── live/               # Gemini Live API session management
│   │   ├── session.go      # Manager.Connect(), Session.Send{Audio,Video,Text}(), Receive()
│   │   └── session_test.go
│   ├── ws/                 # WebSocket handling
│   │   ├── handler.go      # Main handler (773 lines): upgrade, routing, ADK integration
│   │   ├── registry.go     # Thread-safe active connection registry
│   │   └── *_test.go
│   ├── adk/                # ADK Orchestrator HTTP client
│   │   ├── client.go       # POST /analyze, POST /search with Cloud Run ID tokens
│   │   └── client_test.go
│   ├── auth/               # JWT authentication
│   │   ├── jwt.go          # HS256, 24h duration
│   │   ├── handler.go      # POST /api/v1/auth/{register,refresh}, Bearer middleware
│   │   └── *_test.go
│   ├── tts/                # Text-to-Speech
│   │   ├── client.go       # StreamSpeak() via GenAI SDK, 15s timeout
│   │   └── client_test.go
│   └── secrets/            # GCP Secret Manager
│       ├── secrets.go      # LoadSecret(), LoadSecretWithVersion()
│       └── secrets_test.go
└── cmd/videotest/
    └── main.go             # Live API testing tool (682 lines, 21+ tests)
```

### WebSocket Flow

```
┌──────────────┐         ┌──────────────────┐         ┌──────────────────┐
│ macOS Client │  WS     │    ws.Handler    │  gRPC   │  Gemini Live API │
│              │────────►│   (handler.go)   │────────►│                  │
│              │◄────────│                  │◄────────│                  │
└──────────────┘         └────────┬─────────┘         └──────────────────┘
                                  │
                           HTTP POST /analyze
                                  │
                                  ▼
                         ┌──────────────────┐
                         │ ADK Orchestrator │
                         └──────────────────┘
```

**Message Types (Client → Gateway):**

| Type | Format | Content |
|------|--------|---------|
| Binary (audio) | Raw bytes | PCM 16kHz 16-bit mono |
| Binary (video) | JPEG bytes | Screen capture frame (detected by JPEG magic bytes) |
| Text `setup` | JSON | Character, voice, language, soul, session handle |
| Text `screenCapture` | JSON | Base64 JPEG + context for ADK analysis |
| Text `text` | JSON | Chat text input |
| Text `voiceSearch` | JSON | Voice query for Google Search |

**Message Types (Gateway → Client):**

| Type | Content |
|------|---------|
| Binary | PCM 24kHz audio from Live API |
| `setupComplete` | Session established |
| `transcription` | User/model speech transcription |
| `companionSpeech` | Agent-generated speech with emotion |
| `liveSessionReconnecting` | Connection recovery in progress |
| `liveSessionReconnected` | Connection restored |
| `error` | Error with code |

### GenAI SDK Usage

| Feature | Pattern |
|---------|---------|
| Live API connect | `client.Live.Connect(ctx, model, liveConfig)` |
| Send audio | `session.SendRealtimeInput(LiveRealtimeInput{Audio: &Blob{...}})` |
| Send video (Fast Path) | `session.SendRealtimeInput(LiveRealtimeInput{Video: &Blob{MIMEType:"image/jpeg"}})` |
| Send text | `session.SendRealtimeInput(LiveRealtimeInput{Text: text})` |
| Receive | `session.Receive()` → `*LiveServerMessage` |
| TTS streaming | `client.Models.GenerateContentStream()` with `ResponseModalities: ["AUDIO"]` |
| Session resumption | `SessionResumptionConfig{Handle: resumptionHandle}` |
| Context window compression | `ContextWindowCompressionConfig{TriggerTokens: 100000, SlidingWindow{TargetTokens: 50000}}` |
| Affective dialog | `EnableAffectiveDialog: &true` |
| Proactive audio | `Proactivity: &ProactivityConfig{ProactiveAudio: &true}` |

### Reconnection Logic

```
Error detected → errChan (buffered)
  └── Reconnection goroutine:
        for attempt 1..3:
          delay = 2^(attempt-1) seconds (1s, 2s, 4s)
          → Connect with resumption handle
          → Success: restart receiver goroutine
          → Failure: retry or send LIVE_SESSION_LOST
```

### Graceful Degradation

| Condition | Behavior |
|-----------|----------|
| `liveMgr == nil` | Stub mode — echoes audio, no Gemini |
| `adkClient == nil` | Screen captures ignored, search disabled |
| `ttsClient == nil` | Text bubbles only, no audio for urgent messages |

---

## 5. Layer 3: ADK Orchestrator

### Package Structure

```
backend/adk-orchestrator/
├── main.go                        # HTTP handlers, Runner setup, observability
├── internal/
│   ├── agents/
│   │   ├── graph/graph.go         # 9-agent graph construction (3 waves)
│   │   ├── vision/vision.go       # VisionAgent: screen analysis
│   │   ├── memory/memory.go       # MemoryAgent: cross-session context
│   │   ├── mood/mood.go           # MoodDetector: frustration sensing
│   │   ├── celebration/celebration.go  # CelebrationTrigger: success detection
│   │   ├── mediator/mediator.go   # Mediator: speech gating
│   │   ├── scheduler/scheduler.go # AdaptiveScheduler: timing
│   │   ├── engagement/engagement.go # EngagementAgent: proactive triggers
│   │   ├── search/search.go       # SearchBuddy: custom search
│   │   ├── search/llmsearch.go    # LLM SearchBuddy: llmagent + tools
│   │   ├── search/classifier.go   # Search intent classifier
│   │   └── topic/topic.go         # Topic detector
│   ├── models/models.go           # Request/response types
│   ├── prompts/prompts.go         # Prompt templates
│   └── store/
│       ├── firestore.go           # Firestore client (6 collections)
│       └── models.go              # Firestore schema types
```

### ADK SDK Usage

```go
import (
    "google.golang.org/adk/agent"
    "google.golang.org/adk/agent/llmagent"
    "google.golang.org/adk/agent/workflowagents/loopagent"
    "google.golang.org/adk/agent/workflowagents/parallelagent"
    "google.golang.org/adk/agent/workflowagents/sequentialagent"
    "google.golang.org/adk/memory"
    "google.golang.org/adk/model/gemini"
    "google.golang.org/adk/plugin/retryandreflect"
    "google.golang.org/adk/runner"
    "google.golang.org/adk/session"
    "google.golang.org/adk/telemetry"
    "google.golang.org/adk/tool/functiontool"
    "google.golang.org/adk/tool/geminitool"
)
```

### Runner Configuration

```go
runner.New(runner.Config{
    AppName:        "vibecat",
    Agent:          agentGraph,       // 9-agent sequential/parallel graph
    SessionService: session.InMemoryService(),
    MemoryService:  memory.InMemoryService(),
    PluginConfig: runner.PluginConfig{
        Plugins: []*plugin.Plugin{retryandreflect.MustNew(...)},
    },
})
```

### HTTP Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/analyze` | POST | Run 9-agent graph on screen capture |
| `/search` | POST | Direct voice search query |
| `/health` | GET | Liveness probe |
| `/readyz` | GET | Readiness probe |

### `/analyze` Request/Response

```go
// Request
type AnalysisRequest struct {
    Image           string // base64 JPEG
    Context         string // "editor", "browser", etc.
    Language        string
    AppName         string
    SessionID       string
    UserID          string
    Character       string // "cat", "derpy", etc.
    Soul            string // Character persona from soul.md
    ActivityMinutes int    // Time since last activity
}

// Response
type AnalysisResult struct {
    Vision      *VisionAnalysis   // What was seen on screen
    Decision    *MediatorDecision // Whether to speak
    Mood        *MoodState        // Developer mood
    Celebration *CelebrationEvent // Success detection
    Search      *SearchResult     // Search results
    SpeechText  string            // Text to speak/display
}
```

---

## 6. 9-Agent Graph

### Graph Structure (3 Waves)

```
SEQUENTIAL: vibecat_graph
│
├── PARALLEL: wave1_perception          ← Independent analysis
│   ├── VisionAgent                     ← Screen capture → significance score
│   └── MemoryAgent                     ← Cross-session context retrieval
│
├── PARALLEL: wave2_emotion             ← Emotional processing
│   ├── MoodDetector                    ← Frustration/stuck/idle detection
│   └── CelebrationTrigger             ← Success event detection
│
└── SEQUENTIAL: wave3_decision          ← Decision chain
    ├── Mediator                        ← Speech gating (speak or stay silent)
    ├── AdaptiveScheduler               ← Dynamic cooldown adjustment
    ├── EngagementAgent                 ← Proactive silence engagement
    └── LOOP: search_refinement_loop    ← Search (max 2 iterations)
        ├── SearchBuddy                 ← Custom search agent
        └── LLM SearchBuddy            ← llmagent + GoogleSearch tool
```

### Agent Specifications

#### Wave 1: Perception (Parallel)

| | VisionAgent | MemoryAgent |
|---|---|---|
| **Model** | `gemini-3.1-flash-lite-preview` | `gemini-2.5-flash-lite` |
| **Input** | Base64 screenshot, context, character, soul | UserID, session history |
| **Output** | `VisionAnalysis`: significance (0-10), content, emotion, shouldSpeak, errorDetected, successDetected | Cross-session context string |
| **Tools** | None (direct GenAI) | Firestore read/write |
| **ADK Type** | `agent.New()` | `agent.New()` |

#### Wave 2: Emotion (Parallel)

| | MoodDetector | CelebrationTrigger |
|---|---|---|
| **Model** | Rule-based (no LLM) | `gemini-3.1-flash-lite-preview` |
| **Input** | VisionAnalysis, voice_tone, voice_confidence | VisionAnalysis (successDetected + significance ≥ 9) |
| **Output** | `MoodState`: mood, confidence, signals, suggestedAction | `CelebrationEvent`: triggerType, emotion, message |
| **Logic** | Error count tracking, silence detection, voice tone fusion | 10-minute cooldown, deduplication |
| **ADK Type** | `agent.New()` | `agent.New()` |

#### Wave 3: Decision (Sequential)

| | Mediator | AdaptiveScheduler | EngagementAgent | SearchBuddy |
|---|---|---|---|---|
| **Model** | `gemini-3.1-flash-lite-preview` | Rule-based | `gemini-3.1-flash-lite-preview` | `gemini-2.5-flash` |
| **Input** | Vision + Mood + Celebration | Utterance rate | Activity minutes | Errors, vision content, mood |
| **Output** | `MediatorDecision` + speechText | Adjusted cooldown/silence | Check-in messages, rest reminders | `SearchResult`: query, summary, sources |
| **Logic** | Cooldown (10s default, 180s mood), significance threshold, duplicate detection | Rate > 2/min → ↑cooldown; < 0.5/min → ↓cooldown | After 180s silence, 50min rest | Mood-based triggers |
| **ADK Type** | `agent.New()` | `agent.New()` | `agent.New()` | `agent.New()` + `llmagent.New()` (loop) |
| **Tools** | — | — | — | `geminitool.GoogleSearch{}`, `functiontool.New()` |

### Firestore Collections

| Collection | Path | Purpose |
|------------|------|---------|
| `sessions` | `sessions/{sessionId}` | Session metadata, settings |
| `metrics` | `sessions/{sessionId}/metrics/current` | Behavioral metrics |
| `history` | `sessions/{sessionId}/history/{entryId}` | Interaction event log |
| `searches` | `sessions/{sessionId}/searches/{query}` | Cached search results |
| `users` | `users/{userId}` | User profiles |
| `memory` | `users/{userId}/memory/data` | Cross-session summaries |

---

## 7. Data Flow

### Screen Capture Pipeline

```
1Hz ScreenCaptureKit
       │
       ▼
  ImageDiffer (32×32 thumbnail, ~1ms)
       │
       ├── No change → skip
       │
       ├── Fast Path (every 5s cooldown)
       │     └── DispatchQueue.global → JPEG encode → main thread
       │           └── GatewayClient.sendVideoFrame()
       │                 └── session.SendRealtimeInput(Video: JPEG blob)
       │                       └── Gemini Live API (realtime vision)
       │
       └── Smart Path (every 15s cooldown)
             └── DispatchQueue.global → base64 encode → main thread
                   └── GatewayClient.sendScreenCapture(base64)
                         └── Gateway → POST /analyze → ADK Orchestrator
                               └── 9-agent graph → AnalysisResult
```

### Voice Pipeline

```
Microphone (AVAudioEngine)
       │
       ▼
  SpeechRecognizer (PCM 16kHz 16-bit mono)
       │
       ├── VAD Check: isSpeechActive?
       │     ├── audioPlayer.isPlaying → suppress
       │     ├── isTTSSpeaking → suppress
       │     └── postSpeechCooldown (5s) → suppress
       │
       ├── RMS Threshold (0.01) → below = ignore
       │
       └── GatewayClient.sendAudio(pcmData)
             └── Gateway → session.SendRealtimeInput(Audio)
                   └── Gemini Live API
                         │
                         ├── Audio response → PCM 24kHz → Client AudioPlayer
                         ├── Transcription → Client StatusBar
                         └── Model turn complete → ADK analysis trigger
```

### Speech Protection (3-Layer)

```
Layer 1: Client (SpeechRecognizer)
  └── rmsThreshold=0.01, bargeInThreshold=0.05, consecutiveThreshold=3

Layer 2: Gateway (ws/handler.go)
  └── modelSpeaking guard, JPEG frame gating during speech

Layer 3: Gemini Live API Config
  └── PrefixPaddingMs=500, SilenceDurationMs=500
  └── StartOfSpeechSensitivity=Low, EndOfSpeechSensitivity=Low
```

---

## 8. Character System

### 6 Characters

| Character | Voice | Tone | Coding Role |
|-----------|-------|------|-------------|
| `cat` | Zephyr | bright, casual | beginner-eye |
| `derpy` | Puck | goofy, clumsy | accidental debugger |
| `jinwoo` | Kore | low-calm, concise | silent senior engineer |
| `kimjongun` | Schedar | authoritative-warm | supreme debugger (comedy) |
| `saja` | Zubenelgenubi | calm-deep, archaic | zen mentor |
| `trump` | Fenrir | energetic-superlative | bombastic hype-man (comedy) |

### Character Files

```
Assets/Sprites/{character}/
├── preset.json     # Voice, size, persona config
├── soul.md         # Personality prompt (injected server-side into system prompt)
├── idle_01..04.png
├── happy_01..04.png
├── surprised_01..04.png
└── thinking_01..04.png
```

### Persona Injection Flow

```
Client reads preset.json + soul.md
  └── Sends in WebSocket "setup" message
        └── Gateway injects soul.md into Live API SystemInstruction
        └── Gateway forwards to ADK Orchestrator in AnalysisRequest.Soul
              └── VisionAgent includes persona in analysis prompt
```

---

## 9. Infrastructure & Deployment

### GCP Services

| Service | Purpose | Region |
|---------|---------|--------|
| Cloud Run | 2 services (gateway + orchestrator) | asia-northeast3 |
| Firestore | Sessions, metrics, memory (6 collections) | asia-northeast3 |
| Secret Manager | `vibecat-gemini-api-key`, `vibecat-gateway-auth-secret` | asia-northeast3 |
| Artifact Registry | Docker images (`vibecat-images`) | asia-northeast3 |
| Cloud Build | Service YAML present, no active triggers | global |
| Cloud Logging | Structured JSON logs | — |
| Cloud Monitoring | Custom metrics, dashboards | — |
| Cloud Trace | OpenTelemetry distributed tracing | — |

### Cloud Run Configuration

| Setting | Gateway | Orchestrator |
|---------|---------|-------------|
| Memory | 512Mi | 1Gi |
| Min instances | 0 | 0 |
| Max instances | 20 | 20 |
| Concurrency | 80 | 100 |
| Auth | Public (`--allow-unauthenticated`) | Authenticated invocation required |
| Session affinity | Yes | No |

### Docker Build

Both services use identical multi-stage builds:
```dockerfile
FROM golang:1.24-alpine AS builder  →  CGO_ENABLED=0 go build
FROM gcr.io/distroless/static-debian12  →  minimal runtime
```

### CI/CD Pipeline

```
GitHub Push/PR → CI (ci.yml)
  ├── Go Gateway: build + vet + test (ubuntu-latest, Go 1.24)
  ├── Go Orchestrator: build + vet + test (ubuntu-latest, Go 1.24)
  ├── Swift: build + test (self-hosted macOS ARM64)
  └── Docker: build both images

Manual Trigger → CD (cd.yml)
  ├── Deploy orchestrator → Cloud Run
  ├── Deploy gateway → Cloud Run (with orchestrator URL)
  └── Smoke test against deployed gateway
```

**Auth**: Workload Identity Federation (`github-actions@vibecat-489105.iam.gserviceaccount.com`)

### Deployment Commands

```bash
# One-time setup
./infra/setup.sh

# Deploy both services
./infra/deploy.sh

# Swift client
make build && make sign && make run

# Teardown
./infra/teardown.sh
```

---

## 10. Configuration Reference

### Models

| Model | Purpose | Used By |
|-------|---------|---------|
| `gemini-2.5-flash-native-audio-preview-12-2025` | Live API (voice + vision) | Gateway |
| `gemini-2.5-flash-preview-tts` | Text-to-Speech streaming | Gateway |
| `gemini-3.1-flash-lite-preview` | Vision analysis and multimodal perception | ADK Orchestrator |
| `gemini-2.5-flash-lite` | Memory summarization and lightweight text generation | ADK Orchestrator |
| `gemini-2.5-flash` | Search grounding and tool-driven generation | ADK Orchestrator |

### VAD Configuration

| Parameter | Value |
|-----------|-------|
| PrefixPaddingMs | 500 |
| SilenceDurationMs | 500 |
| StartOfSpeechSensitivity | Low |
| EndOfSpeechSensitivity | Low |
| ActivityHandling | StartOfActivityInterrupts |
| TurnCoverage | TurnIncludesOnlyActivity |

### Client VAD Thresholds

| Parameter | Value |
|-----------|-------|
| rmsThreshold | 0.01 |
| bargeInThreshold | 0.05 |
| consecutiveThreshold | 3 |
| Post-speech cooldown | 5 seconds |

### Timing

| Parameter | Value |
|-----------|-------|
| Screen capture interval | 1 second |
| Fast Path cooldown | 5 seconds |
| Smart Path cooldown | 15 seconds |
| Cat animation | 60 FPS |
| Sprite animation | 8.3 FPS |
| Mouse event throttle | 30 FPS |
| WebSocket ping | 54 seconds |
| WebSocket pong timeout | 60 seconds |
| ADK request timeout | 30 seconds |
| TTS request timeout | 15 seconds |
| JWT token duration | 24 hours |
| Engagement silence trigger | 180 seconds |
| Rest reminder | 50 minutes |
| Celebration cooldown | 10 minutes |
| Mediator default cooldown | 10 seconds |
| Mediator mood cooldown | 180 seconds |

### Context Window

| Parameter | Value |
|-----------|-------|
| Trigger tokens | 100,000 |
| Target tokens (after compression) | 50,000 |
| MediaResolution | Medium |

---

## 11. File Inventory

### Source Code Statistics

| Layer | Files | Primary Language |
|-------|-------|-----------------|
| Core Module | 8 | Swift |
| App Module | 20 | Swift |
| Realtime Gateway | 12 | Go |
| ADK Orchestrator | 17 | Go |
| Tests | 12 | Swift + Go |
| Infrastructure | 7 | Shell + YAML + Dockerfile |
| **Total** | **76** | — |

### Asset Statistics

| Category | Count |
|----------|-------|
| Sprite PNGs | 97 |
| Character presets (JSON) | 6 |
| Character personas (soul.md) | 6 |
| Tray icon PNGs | 72 (48 raw + 24 clean) |
| Music tracks | 2 |
| Voice samples | 13 |
| **Total assets** | **196** |

### Test Coverage

| Suite | Tests | Status |
|-------|-------|--------|
| Swift (VibeCatTests) | 31 | All passing |
| Go Gateway | 6 packages | All passing |
| Go Orchestrator | 13 packages | All passing |

### Current Deployment

| Service | Revision | URL |
|---------|----------|-----|
| Gateway | `realtime-gateway-00040-gcd` | `https://realtime-gateway-a4akw2crra-du.a.run.app` |
| Orchestrator | `adk-orchestrator-00038-t4c` | `https://adk-orchestrator-a4akw2crra-du.a.run.app` |

---

*Reviewed against codebase and deployment state at commit `7356f8c` on 2026-03-11.*
