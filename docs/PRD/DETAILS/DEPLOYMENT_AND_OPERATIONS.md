# Deployment and Operations

## GCP Targets

- Region: `asia-northeast3`
- Runtime: Cloud Run (realtime-gateway, adk-orchestrator)
- Build and Artifacts: Cloud Build + Artifact Registry
- Data and Security: Firestore + Secret Manager + Cloud Storage
- Observability: Cloud Logging + Cloud Monitoring + Cloud Trace

## Backend Services

### Realtime Gateway (`realtime-gateway`)

- **Role**: WebSocket proxy between macOS client and Gemini Live API
- **Stack**: Go (`google.golang.org/genai`)
- **Container Port**: 8080
- **Key Responsibilities**:
  - Accept WebSocket connections from Swift client
  - Authenticate client sessions using ephemeral tokens
  - Initialize Gemini Live API session via GenAI SDK
  - Forward audio, text, and images between client and Gemini
  - Apply VAD configuration (`automaticActivityDetection`)
  - Handle session resumption and context compression
  - Route screen capture analysis requests to ADK Orchestrator
  - Forward transcription events to client
- **Health Endpoints**: `/healthz` (liveness), `/readyz` (readiness)
- **Scaling**:
  - Min instances: 0
  - Max instances: 10
  - Memory: 512Mi
  - CPU: 1 vCPU
  - Concurrency: 80

### ADK Orchestrator (`adk-orchestrator`)

- **Role**: Agent graph execution, decision-making, tool routing
- **Stack**: Go (`google.golang.org/adk`)
- **Container Port**: 8080
- **Key Responsibilities**:
  - Define and run ADK agent graph
  - Vision analysis agent (screen captures to structured analysis)
  - Mediator agent (speech gating, cooldown, significance checks)
  - Engagement agent (proactive triggers after silence)
  - Adaptive scheduler agent (metric tracking, timing adjustments)
  - Memory agent (cross-session context, topic tracking)
  - Mood detector (frustration sensing, supportive responses)
  - Celebration trigger (success detection, positive reinforcement)
  - Search buddy (Google Search grounding, result summarization)
  - Persist agent state in Firestore
  - Return decisions to Realtime Gateway
- **Endpoint**: `POST /analyze`
  - Request: `{image: base64, context: {appName, windowTitle, captureType}}`
  - Response: `{shouldSpeak: bool, significance: int, emotion: string, content: string, urgency: string}`
- **Health Endpoints**: `/healthz` (liveness), `/readyz` (readiness)
- **Scaling**:
  - Min instances: 0
  - Max instances: 10
  - Memory: 1Gi
  - CPU: 1 vCPU
  - Concurrency: 100

## Firestore Schema

### `sessions/{sessionId}`

| Field | Type | Description |
|---|---|---|
| userId | string | Client user identifier |
| createdAt | timestamp | Session creation time |
| lastActiveAt | timestamp | Last activity timestamp |
| liveSessionHandle | string | Gemini session resumption handle |
| settings.voice | string | Selected voice |
| settings.language | string | Language code |
| settings.liveModel | string | Gemini model identifier |
| settings.chattiness | string | Interaction frequency level |
| settings.character | string | Character preset name |

### `sessions/{sessionId}/metrics`

| Field | Type | Description |
|---|---|---|
| utteranceCount | int | Total user utterances |
| responseCount | int | Total agent responses |
| interruptCount | int | Total interruptions |
| responseRate | float | responseCount / utteranceCount |
| interruptRate | float | interruptCount / utteranceCount |
| silenceThreshold | float | Seconds, bounded 10-45 |
| cooldownSeconds | float | Seconds, bounded 5-20 |
| lastUpdatedAt | timestamp | Last metrics update time |

### `sessions/{sessionId}/history`

| Field | Type | Description |
|---|---|---|
| timestamp | timestamp | Event time |
| type | string | vision_analysis, speech, engagement, interruption, celebration, mood_change, or search |
| content | string | Event content summary |
| significance | int | Significance score |

### `users/{userId}/memory` (cross-session, MemoryAgent)

| Field | Type | Description |
|---|---|---|
| recentSummaries | array | [{date, summary, unresolvedIssues}] — last N session summaries |
| knownTopics | array | [{topic, lastMentioned, resolved}] — tracked discussion topics |
| codingPatterns | array | [{pattern, frequency}] — observed coding behaviors |
| lastSessionAt | timestamp | When the user last used VibeCat |

### `sessions/{sessionId}/searches` (SearchBuddy cache)

| Field | Type | Description |
|---|---|---|
| timestamp | timestamp | Search time |
| query | string | Search query |
| summary | string | Summarized result |
| sources | array | [{title, url}] — source references |
| triggeredBy | string | user_request or auto_mood |

## Secret Manager

| Secret Name | Purpose |
|---|---|
| `vibecat-gemini-api-key` | Gemini API key for GenAI SDK |
| `vibecat-gateway-auth-secret` | Client session token signing key |

Access pattern: each Cloud Run service reads secrets at startup via Secret Manager API. No secrets are stored in environment variables or container images.

## VAD Configuration

Applied in Realtime Gateway when initializing Gemini Live API session:

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
  }
}
```

## Observability

### Logging

- Tool: Cloud Logging
- Format: Structured JSON
- Required log fields: `severity`, `message`, `service`, `sessionId`, `traceId`
- Both services emit request logs, error logs, and decision logs

### Monitoring

- Tool: Cloud Monitoring
- Key metrics:
  - **Gateway**: WebSocket connection count, audio throughput (bytes/sec), Gemini API latency (p50/p95/p99), reconnection rate
  - **Orchestrator**: Agent graph execution time, decision distribution (speak/silent ratio), Firestore write latency
- Alerting: configure alerts for error rate > 5% and Gemini latency p99 > 5s

### Tracing

- Tool: Cloud Trace
- Span coverage: Client to Gateway to Orchestrator to Gemini
- Required spans: WebSocket message receive, Gemini API call, ADK agent graph execution, Firestore read/write

## Keepalive Stack

| Layer | Direction | Interval | Purpose |
|---|---|---|---|
| Client ping | Client to Gateway | 15s | Detect dead client connection |
| Gateway pong | Gateway to Client | on ping | Confirm alive |
| Gateway to Gemini ping | Gateway to Gemini | 15s | Keep Gemini session alive |
| App heartbeat | Gateway to Gemini | 30s | Prevent Gemini timeout |
| Zombie detection | Gateway | 45s no pong | Tear down stale connections |

## Error Handling

| Error | Gateway Action | Client Receives |
|---|---|---|
| Gemini rate limit | Queue and retry after delay | `error: GEMINI_RATE_LIMIT, retryAfterMs` |
| Gemini unavailable | Retry with exponential backoff | `error: GEMINI_UNAVAILABLE` |
| ADK orchestrator timeout | Return default silent decision | No speech (silent path) |
| Client invalid message | Log and skip | `error: INVALID_MESSAGE` |
| Session expired | Close WebSocket | `error: SESSION_EXPIRED` (client re-authenticates) |

## Required Artifacts

- `backend/realtime-gateway/Dockerfile`
- `backend/adk-orchestrator/Dockerfile`
- `infra/` (IaC or deployment scripts)
- `docs/deployment/PROOF_OF_GCP_DEPLOYMENT.md`

## Build and Deploy Pipeline

1. Code push triggers Cloud Build
2. Cloud Build builds Docker images for both services
3. Images pushed to Artifact Registry
4. Cloud Run services updated with new image revisions
5. Health checks confirm readiness before traffic shift

## Operational Checklist

- [ ] Secret rotation policy defined
- [ ] Structured logging enabled on both backend services
- [ ] Health endpoints available (`/healthz`, `/readyz`) on both services
- [ ] Core SLO metrics visible in Cloud Monitoring dashboard
- [ ] Alerting configured for error rate and latency thresholds
- [ ] Keepalive intervals validated against Gemini session timeout
- [ ] Firestore security rules restrict access to authenticated services only
- [ ] Cloud Build triggers configured for automated deployment
- [ ] Deployment proof artifacts recorded in `docs/deployment/PROOF_OF_GCP_DEPLOYMENT.md`
