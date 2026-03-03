# VibeCat Backend Architecture

## 1. Service Overview

Two Cloud Run services in `asia-northeast3`:

### 1.1 Realtime Gateway (`realtime-gateway`)

- **Role**: WebSocket proxy between macOS client and Gemini Live API
- **Stack**: Go (`google.golang.org/genai`)
- **Port**: 8080
- **Responsibilities**:
  - Accept WebSocket connections from Swift client
  - Authenticate client sessions (ephemeral tokens)
  - Initialize Gemini Live API session via GenAI SDK
  - Forward audio/text/image between client and Gemini
  - Apply VAD configuration (`automaticActivityDetection`)
  - Handle session resumption and context compression
  - Route screen capture analysis requests to ADK Orchestrator
  - Forward transcription events to client
  - Health endpoints: `/healthz`, `/readyz`

### 1.2 ADK Orchestrator (`adk-orchestrator`)

- **Role**: Agent graph execution, decision-making, tool routing
- **Stack**: Go (`google.golang.org/adk`)
- **Port**: 8080
- **Responsibilities**:
  - Define and run ADK agent graph (9 agents)
  - Vision analysis agent (screen captures → structured analysis)
  - Mediator agent (speech gating, cooldown, significance, mood-aware)
  - Engagement agent (proactive triggers after silence)
  - Adaptive scheduler agent (metric tracking, timing adjustments)
  - Memory agent (cross-session context, topic tracking, context bridge)
  - Mood detector (frustration sensing, supportive response routing)
  - Celebration trigger (success detection, positive reinforcement)
  - Search buddy (Google Search grounding, result summarization)
  - Persist agent state in Firestore
  - Return decisions to Realtime Gateway

## 2. Communication Flows

### 2.1 Client → Realtime Gateway (WebSocket)

```
Client opens WSS → Gateway
Client sends: audio (PCM 16kHz 16-bit mono), text, screen captures (JPEG base64)
Gateway sends: audio (PCM 24kHz), transcription, turn events, interruptions
```

### 2.2 Realtime Gateway → Gemini Live API (GenAI SDK)

```
Gateway creates Gemini Live session via GenAI SDK
Setup: voice, language, systemInstruction, tools, VAD config,
       sessionResumption, contextWindowCompression, proactiveAudio,
       affectiveDialog, outputAudioTranscription
Gateway proxies client audio/text ↔ Gemini audio/transcription
```

### 2.3 Realtime Gateway → ADK Orchestrator (HTTP)

```
POST /analyze
  Request: {image: base64, context: {appName, windowTitle, captureType}}
  Response: {shouldSpeak: bool, significance: int, emotion: string, content: string, urgency: string}
```

### 2.4 ADK Orchestrator → Firestore

```
Read/Write: sessions/{sessionId}/metrics
  - utteranceCount, responseCount, interruptCount
  - responseRate, interruptRate
  - silenceThreshold, cooldownSeconds
```

## 3. ADK Agent Graph (9 Agents)

### 3.1 Agent Philosophy

A chatbot answers. A colleague sees, hears, judges, adapts, remembers, cares, celebrates, and helps.

| Agent | Colleague Role | Category |
|---|---|---|
| VAD | Natural conversation | Core (5) |
| VisionAgent | Second pair of eyes | Core |
| Mediator | Social awareness | Core |
| AdaptiveScheduler | Rhythm awareness | Core |
| EngagementAgent | Initiative | Core |
| MemoryAgent | Long-term memory | Companion Intelligence (4) |
| MoodDetector | Emotional awareness | Companion Intelligence |
| CelebrationTrigger | Cheerleader | Companion Intelligence |
| SearchBuddy | Research assistant | Companion Intelligence |

### 3.2 Core Agent Flow

```
[Screen Capture Input]
        │
        ▼
  ┌─────────────┐
  │ VisionAgent  │ ← Gemini REST via GenAI SDK (gemini-3-flash)
  │ (structured) │   responseSchema enforced
  └──────┬──────┘
         │ VisionAnalysis {significance, content, emotion, shouldSpeak,
         │                 errorDetected, successDetected, repeatedError}
         ▼
  ┌─────────────┐
  │ MoodDetector │ ← Combines: VisionAgent signals + silence duration
  │ (sensing)    │   + interaction rate from AdaptiveScheduler
  └──────┬──────┘
         │ MoodState {mood, confidence, signals, suggestedAction}
         ▼
  ┌─────────────┐
  │  Mediator    │ ← Reads cooldown/threshold from AdaptiveScheduler
  │ (gating)     │   Checks: cooldown, significance, duplication,
  │              │   app-change, mood-adjusted threshold
  └──────┬──────┘
         │ MediatorDecision {shouldSpeak, reason, urgency, contextVersion}
         ▼
    ┌────┴────┐
 speak=true  speak=false → log only
    │
    ▼
  [Return to Gateway → forward to Gemini Live API]
```

### 3.3 Parallel Agents (timer/event-based)

```
  ┌──────────────┐
  │EngagementAgent│ ← Monitors silence duration from Firestore
  │(proactive)    │ ← Threshold from AdaptiveScheduler
  └──────┬───────┘
         │ Triggers when silence > threshold
         ▼
  [Send proactive prompt to Gateway]

  ┌───────────────────┐
  │ CelebrationTrigger │ ← Receives VisionAgent.successDetected
  │ (positive)         │ ← Cooldown: max 1 per 5 minutes
  └──────┬────────────┘
         │ Fires celebration event (happy sprite + voice)
         ▼
  [Send celebration to Gateway, bypass Mediator gating]

  ┌──────────────┐
  │  SearchBuddy  │ ← Triggered by: user request OR MoodDetector (stuck 3+ min)
  │ (research)    │ ← Uses Google Search grounding tool
  └──────┬───────┘
         │ SearchResult {query, summary, sources}
         ▼
  [Send summarized result to Gateway via voice]
```

### 3.4 Session Lifecycle Agents

```
  ┌──────────────┐
  │ MemoryAgent   │
  │ (memory)      │
  └──────┬───────┘
         │
  On session START:
    → Load users/{userId}/memory from Firestore
    → Inject context bridge into system instruction
    → "어제 인증 모듈 하다 멈췄지, 이어서 할래?"
         │
  On session END:
    → Summarize session (topics, unresolved issues)
    → Store summary in users/{userId}/memory
    → Update knownTopics
         │
  During session:
    → Detect significant topics from conversation
    → Update knownTopics in real-time
```

### 3.5 Agent Interaction Matrix

| Source Agent | Target Agent | Data Exchanged |
|---|---|---|
| VisionAgent | MoodDetector | errorDetected, repeatedError, successDetected |
| VisionAgent | CelebrationTrigger | successDetected (test pass, build success) |
| VisionAgent | Mediator | VisionAnalysis (significance, content, emotion) |
| MoodDetector | Mediator | MoodState (mood, suggestedAction) — adjusts speak threshold |
| MoodDetector | SearchBuddy | stuck signal (same error 3+ minutes) → auto-search |
| AdaptiveScheduler | Mediator | cooldown, silence threshold |
| AdaptiveScheduler | EngagementAgent | proactive trigger threshold |
| MemoryAgent | Gateway | session context (injected into system instruction) |
| SearchBuddy | Gateway | search result (voice output) |
| CelebrationTrigger | Gateway | celebration event (sprite + voice) |

### 3.6 New Agent Specifications

#### MemoryAgent

- **Purpose**: Cross-session memory — remembers past conversations, topics, unresolved issues
- **Triggers**: session start (load), session end (save), significant topic detected (update)
- **Storage**: Firestore `users/{userId}/memory`
- **Context Bridge**: On new session, constructs a brief summary of relevant past context and injects it into the Gemini system instruction
- **Topic Detection**: Identifies recurring themes (e.g., "authentication", "database migration") from conversation and VisionAgent output
- **Summary Generation**: At session end, uses Gemini to generate a concise session summary with unresolved issues listed

#### MoodDetector

- **Purpose**: Detects developer mood from screen patterns + interaction patterns
- **Input Signals**:
  - VisionAgent: repeated error screens (same error 3+ times), failed build patterns
  - EngagementAgent: silence duration after error
  - AdaptiveScheduler: interaction rate drop
- **Output**: MoodState with classification (focused, frustrated, stuck, idle) and confidence score
- **Integration**: Feeds into Mediator — frustrated mood lowers speak threshold for supportive messages, focused mood raises threshold to avoid disturbance
- **Supportive Messages**: "힘들어 보이는데, 같이 한번 볼까?" / "잠깐 쉬어볼래?" / "한 발짝 물러서서 보면 보일 수도 있어"

#### CelebrationTrigger

- **Purpose**: Detects positive moments and celebrates with the developer
- **Detection Patterns**: VisionAgent sees "tests passed", "build succeeded", "deployed", green CI status, PR merged
- **Response**: Triggers happy sprite state + celebratory voice message ("오 통과했네! 고생했어")
- **Cooldown**: Maximum 1 celebration per 5 minutes (prevents annoyance), tracked in session metrics
- **Mediator Bypass**: Celebration events bypass normal significance gating in Mediator

#### SearchBuddy

- **Purpose**: Searches for solutions when developer is stuck
- **Trigger Sources**:
  - User explicit: "이거 뭐야", "왜 안돼", "어떻게 해", or asks about specific error
  - Auto from MoodDetector: stuck on same error for 3+ minutes
- **Tool**: Google Search grounding (already available in ADK)
- **Output**: Voice-friendly summarized result: "찾아봤는데, Stack Overflow에서..."
- **Caching**: Recent searches cached in `sessions/{sessionId}/searches` to avoid duplicates

## 4. Firestore Schema

### `sessions/{sessionId}`
```
{
  userId: string,
  createdAt: timestamp,
  lastActiveAt: timestamp,
  liveSessionHandle: string,    // Gemini session resumption
  settings: {
    voice: string,
    language: string,
    liveModel: string,
    chattiness: string,
    character: string
  }
}
```

### `sessions/{sessionId}/metrics`
```
{
  utteranceCount: int,
  responseCount: int,
  interruptCount: int,
  responseRate: float,          // responseCount / utteranceCount
  interruptRate: float,         // interruptCount / utteranceCount
  silenceThreshold: float,      // seconds (bounded 10-45)
  cooldownSeconds: float,       // seconds (bounded 5-20)
  currentMood: string,          // focused | frustrated | stuck | idle
  moodConfidence: float,        // 0.0-1.0
  celebrationCount: int,        // celebrations fired this session
  lastCelebrationAt: timestamp, // cooldown tracking
  lastUpdatedAt: timestamp
}
```

### `sessions/{sessionId}/history` (recent interactions)
```
{
  timestamp: timestamp,
  type: string,                 // vision_analysis | speech | engagement | interruption
                                // | celebration | mood_change | search
  content: string,
  significance: int
}
```

### `users/{userId}/memory` (cross-session, MemoryAgent)
```
{
  recentSummaries: [            // last N session summaries
    {
      date: timestamp,
      summary: string,
      unresolvedIssues: [string]
    }
  ],
  knownTopics: [                // tracked discussion topics
    {
      topic: string,
      lastMentioned: timestamp,
      resolved: bool
    }
  ],
  codingPatterns: [              // observed behaviors
    {
      pattern: string,
      frequency: int
    }
  ],
  lastSessionAt: timestamp
}
```

### `sessions/{sessionId}/searches` (SearchBuddy cache)
```
{
  timestamp: timestamp,
  query: string,
  summary: string,
  sources: [{title: string, url: string}],
  triggeredBy: string           // user_request | auto_mood
}
```

## 5. Secret Manager Keys

| Secret Name | Purpose |
|---|---|
| `vibecat-gemini-api-key` | Gemini API key for GenAI SDK |
| `vibecat-gateway-auth-secret` | Client session token signing key |

## 6. VAD and Live API Configuration

Applied in Realtime Gateway's Gemini Live API setup message:

```json
{
  "realtimeInputConfig": {
    "automaticActivityDetection": {
      "disabled": false,
      "startOfSpeechSensitivity": "START_SENSITIVITY_LOW",
      "endOfSpeechSensitivity": "END_SENSITIVITY_LOW",
      "prefixPaddingMs": 20,
      "silenceDurationMs": 100
    }
  },
  "outputAudioTranscription": {},
  "sessionResumption": {},
  "contextWindowCompression": {
    "slidingWindow": {}
  },
  "proactiveAudio": {
    "enabled": true
  },
  "affectiveDialog": {
    "enabled": true
  }
}
```

### proactiveAudio

Enables the agent to speak first without waiting for user input. Essential for the "colleague who checks in" behavior — EngagementAgent and MoodDetector both rely on this to initiate conversation.

### affectiveDialog

Enables emotionally expressive voice output. The agent's voice tone adapts to context:
- Worried tone when MoodDetector senses frustration
- Bright, celebratory tone when CelebrationTrigger fires
- Calm, supportive tone during proactive engagement
- Natural conversational tone during regular interaction

This is the difference between a robotic assistant and a colleague who **sounds** like they care.

## 7. Observability

| Layer | Tool | Key Metrics |
|---|---|---|
| Logging | Cloud Logging | Structured JSON, request/error logs |
| Monitoring | Cloud Monitoring | Connection count, message throughput, Gemini latency |
| Tracing | Cloud Trace | Client→Gateway→Orchestrator→Gemini spans |

### Key Metrics
- **Gateway**: WebSocket connections, audio throughput, Gemini API latency, reconnection rate
- **Orchestrator**: Agent graph execution time, decision distribution (speak/silent ratio), Firestore write latency

## 8. Deployment Specification

| Attribute | Realtime Gateway | ADK Orchestrator |
|---|---|---|
| Region | asia-northeast3 | asia-northeast3 |
| Min instances | 0 | 0 |
| Max instances | 10 | 10 |
| Memory | 512Mi | 1Gi |
| CPU | 1 vCPU | 1 vCPU |
| Concurrency | 80 | 100 |
| Build | Cloud Build → Artifact Registry | Cloud Build → Artifact Registry |

## 9. Keepalive Stack

| Layer | Direction | Interval | Purpose |
|---|---|---|---|
| Client ping | Client→Gateway | 15s | Detect dead client connection |
| Gateway pong | Gateway→Client | on ping | Confirm alive |
| Gateway→Gemini ping | Gateway→Gemini | 15s | Keep Gemini session alive |
| App heartbeat | Gateway→Gemini | 30s | Prevent Gemini timeout |
| Zombie detection | Gateway | 45s no pong | Tear down stale connections |

## 10. Error Handling

| Error | Gateway Action | Client Receives |
|---|---|---|
| Gemini rate limit | Queue and retry after delay | `error: GEMINI_RATE_LIMIT, retryAfterMs` |
| Gemini unavailable | Retry with backoff | `error: GEMINI_UNAVAILABLE` |
| ADK orchestrator timeout | Return default (silent) decision | No speech (silent path) |
| Client invalid message | Log and skip | `error: INVALID_MESSAGE` |
| Session expired | Close WebSocket | `error: SESSION_EXPIRED` → client re-auth |
