# VibeCat Client ↔ Backend Protocol Specification

## 1. Connection Architecture

### 1.1 Primary: WebSocket (Client ↔ Realtime Gateway)
- URL: `wss://{GATEWAY_HOST}/ws/live`
- Protocol: RFC 6455 WebSocket
- Encoding: JSON text frames (control), binary frames (audio)
- Auth: Bearer token in HTTP upgrade headers

### 1.2 Secondary: REST (Client ↔ Realtime Gateway)
- Base URL: `https://{GATEWAY_HOST}/api/v1`
- Used for: auth, settings sync, health checks
- Auth: Bearer token in Authorization header

## 2. Authentication Flow

### 2.1 First-Time Setup
1. Client opens onboarding / connect UI when backend session is not established
2. Client → `POST /api/v1/auth/register` with `{"deviceId": "<stable-device-id>"}`
3. Backend issues an application session token
4. Backend uses the server-side Gemini key from Secret Manager / environment
5. Backend returns `{"sessionToken": "tok_...", "expiresAt": "ISO8601"}`
6. Client uses the session token for WebSocket and REST requests

### 2.2 Normal Session Start
1. Client requests or reuses a valid session token
2. Client opens WebSocket with `Authorization: Bearer {token}`
3. Gateway validates token
4. Gateway initializes Gemini Live API session
5. Gateway sends `setupComplete` to client

### 2.3 Token Refresh
- Tokens expire after 24 hours
- Client receives WebSocket close / 401 → `POST /api/v1/auth/refresh`

## 3. WebSocket Message Protocol

### 3.1 Client → Gateway

#### `setup` (first message after connection)
```json
{
  "type": "setup",
  "config": {
    "voice": "Zephyr",
    "language": "ko",
    "liveModel": "gemini-2.5-flash-native-audio-preview-12-2025",
    "proactiveAudio": true,
    "searchEnabled": true,
    "affectiveDialog": true,
    "deviceId": "device-uuid"
  }
}
```

#### `audio` — Binary frame
Raw PCM 16kHz 16-bit mono

#### `clientContent`
```json
{
  "clientContent": {
    "turnComplete": true,
    "turns": [
      {
        "role": "user",
        "parts": [{"text": "user message text"}]
      }
    ]
  }
}
```

#### `screenCapture`
```json
{
  "type": "screenCapture",
  "image": "<base64 JPEG>",
  "context": "[app=Xcode target=window_under_cursor window=MyProject.swift]"
}
```

#### `forceCapture` (circle gesture)
```json
{
  "type": "forceCapture",
  "image": "<base64 JPEG>",
  "context": "[app=Xcode target=frontmost_window window=MyProject.swift]"
}
```

#### `bargeIn`
```json
{"type": "bargeIn"}
```

#### `settingsUpdate`
```json
{
  "type": "settingsUpdate",
  "changes": {"voice": "Puck", "chattiness": "chatty"}
}
```

#### `ping`
```json
{"type": "ping"}
```

### 3.2 Gateway → Client

#### `setupComplete`
```json
{"type": "setupComplete", "sessionId": "ses_abc123", "resumptionHandle": "h_xyz"}
```

#### `audio` — Binary frame
Raw PCM 24kHz Float32 mono

#### `transcription`
```json
{"type": "transcription", "text": "partial text...", "finished": false}
```

#### `transcription` (sentence boundary)
```json
{"type": "transcription", "text": "Complete sentence.", "finished": true}
```

#### `turnComplete`
```json
{"type": "turnComplete"}
```

#### `interrupted`
```json
{"type": "interrupted"}
```

#### `analysisResult`
```json
{
  "type": "analysisResult",
  "analysis": {
    "significance": 7,
    "emotion": "surprised",
    "shouldSpeak": true,
    "content": "Build error detected in line 42",
    "errorDetected": true,
    "successDetected": false,
    "repeatedError": false,
    "errorRegion": {"x": 0.65, "y": 0.45}
  }
}
```

#### `error`
```json
{
  "type": "error",
  "code": "GEMINI_RATE_LIMIT",
  "message": "Rate limited",
  "retryAfterMs": 5000
}
```

#### `pong`
```json
{"type": "pong"}
```

#### `goAway`
```json
{"type": "goAway", "reason": "session_timeout", "timeLeftMs": 30000}
```

#### `memoryContext` — sent at session start with previous session context
```json
{
  "type": "memoryContext",
  "summary": "어제 인증 모듈 작업 중 OAuth 토큰 갱신 이슈가 있었습니다",
  "unresolvedIssues": ["OAuth token refresh timeout", "CORS configuration"],
  "lastSessionDate": "2026-03-02T23:15:00Z",
  "knownTopics": ["authentication", "OAuth2", "REST API"]
}
```

#### `moodUpdate` — sent when mood state changes
```json
{
  "type": "moodUpdate",
  "mood": "frustrated",
  "confidence": 0.82,
  "signals": ["repeated_error_screen", "long_pause"],
  "suggestedAction": "offer_help",
  "message": "힘들어 보이는데, 같이 한번 볼까?"
}
```

#### `celebration` — sent when success is detected
```json
{
  "type": "celebration",
  "trigger": "test_pass",
  "emotion": "happy",
  "message": "오 통과했네! 고생했어",
  "spriteState": "happy"
}
```

#### `searchResult` — sent when SearchBuddy finds information
```json
{
  "type": "searchResult",
  "query": "OAuth token refresh best practices",
  "summary": "Stack Overflow에서 찾아봤는데, refresh token은 httpOnly cookie에 저장하는 게 권장됩니다",
  "sources": [{"title": "OAuth2 Best Practices", "url": "https://..."}],
  "triggeredBy": "user_request"
}
```

### 3.3 Client → Gateway (Companion Intelligence)

#### `searchRequest` — client explicitly requests a search
```json
{
  "type": "searchRequest",
  "query": "이 에러 어떻게 해결해?"
}
```

#### `memoryFeedback` — client confirms or dismisses memory context
```json
{
  "type": "memoryFeedback",
  "action": "confirm",
  "topic": "OAuth token refresh"
}
```

## 4. REST API Endpoints

| Method | Path | Auth | Purpose |
|---|---|---|---|
| POST | /api/v1/auth/register | None | Register API key, get session token |
| POST | /api/v1/auth/refresh | Bearer (expired) | Refresh session token |
| GET | /api/v1/health | None | Health check |
| POST | /api/v1/settings | Bearer | Sync settings to backend |
| GET | /api/v1/memory/{userId} | Bearer | Retrieve cross-session memory |
| POST | /api/v1/memory/{userId}/feedback | Bearer | Update memory based on user feedback |

## 5. Error Codes

| Code | Meaning | Client Action |
|---|---|---|
| GEMINI_RATE_LIMIT | Gemini API rate limited | Wait retryAfterMs |
| GEMINI_UNAVAILABLE | Gemini API down | Auto-retry, show reconnecting |
| SESSION_EXPIRED | Backend session expired | Re-authenticate via REST |
| INVALID_MESSAGE | Malformed client message | Log and skip |
| INTERNAL_ERROR | Backend crash | Reconnect after 1s |
| MEMORY_UNAVAILABLE | Memory service unavailable | Proceed without context |
| SEARCH_TIMEOUT | Search took too long | Notify user, continue |
| SEARCH_QUOTA_EXCEEDED | Google Search API quota hit | Disable auto-search for session |

## 6. Connection Lifecycle

- **Reconnect delay**: 1s immediate (no exponential backoff)
- **Max attempts**: 50
- **Session resumption**: Client sends last `resumptionHandle` on reconnect
- **Gateway handles**: Gemini session resumption transparently

## 7. Keepalive Protocol

| Layer | Direction | Interval | Timeout |
|---|---|---|---|
| Client ping | Client→Gateway | 15s | - |
| Gateway pong | Gateway→Client | on ping | - |
| Zombie detection | Client-side | - | 45s no pong → reconnect |

## 8. Settings That Require Backend Reconnect

These settings changes cause the Gateway to tear down and recreate the Gemini Live API session:

- `voice`, `language`, `liveModel`, `googleSearch`, `proactiveAudio`, `affectiveDialog`

These take effect immediately without reconnect:
- `chattiness` (sent to ADK Orchestrator), all UI/capture settings (client-local only)
