# Live API Reference

**Source:** https://ai.google.dev/gemini-api/docs/live-api/reference

This document provides detailed API reference for the Gemini Live API.

## Overview

The Live API is a stateful, WebSocket-based API for real-time bi-directional communication with Gemini models. It supports audio, video, and text input/output with low latency.

## WebSocket Endpoint

```
wss://generativelanguage.googleapis.com/v1beta/models/{model}:bidiGenerateContent
```

**Parameters:**
- **model** (required): Model identifier (e.g., `gemini-2.5-flash-native-audio-preview-12-2025`)

## Authentication

### API Key
Pass your API key as a query parameter:
```
wss://generativelanguage.googleapis.com/v1beta/models/{model}:bidiGenerateContent?key=YOUR_API_KEY
```

### Ephemeral Token
For client-side applications, use ephemeral tokens instead of API keys:
```
wss://generativelanguage.googleapis.com/v1beta/models/{model}:bidiGenerateContent?token=EPHEMERAL_TOKEN
```

## Message Format

### Setup Message (Client → Server)

The first message sent after connection establishes the session:

```json
{
  "setup": {
    "model": "gemini-2.5-flash-native-audio-preview-12-2025",
    "generation_config": {
      "response_modalities": ["AUDIO"],
      "system_instruction": {
        "role": "system",
        "parts": [{"text": "You are a helpful assistant."}]
      }
    },
    "realtime_input_config": {
      "automatic_activity_detection": {
        "disabled": false,
        "start_of_speech_sensitivity": "START_SENSITIVITY_LOW",
        "end_of_speech_sensitivity": "END_SENSITIVITY_LOW",
        "prefix_padding_ms": 20,
        "silence_duration_ms": 100
      }
    }
  }
}
```

### Realtime Input Message (Client → Server)

Send audio, video, or text in real-time:

```json
{
  "realtime_input": {
    "media_chunks": [
      {
        "data": "base64EncodedAudioBytes",
        "mime_type": "audio/pcm;rate=16000"
      }
    ]
  }
}
```

Or with manual VAD:

```json
{
  "realtime_input": {
    "activity_start": {}
  }
}
```

```json
{
  "realtime_input": {
    "activity_end": {}
  }
}
```

### Client Content Message (Client → Server)

Send text content:

```json
{
  "client_content": {
    "turns": [
      {
        "role": "user",
        "parts": [{"text": "Hello!"}]
      }
    ],
    "turn_complete": true
  }
}
```

### Server Content Message (Server → Client)

Model responses:

```json
{
  "server_content": {
    "model_turn": {
      "parts": [
        {
          "text": "Hello! How can I help you?"
        },
        {
          "inline_data": {
            "mime_type": "audio/pcm;rate=24000",
            "data": "base64EncodedAudioBytes"
          }
        }
      ]
    },
    "turn_complete": false,
    "interrupted": false,
    "generation_complete": false
  }
}
```

### Tool Call Message (Server → Client)

```json
{
  "server_content": {
    "tool_call": {
      "id": "call_123",
      "name": "function_name",
      "args": {"param": "value"}
    }
  }
}
```

### Tool Response Message (Client → Server)

```json
{
  "tool_response": {
    "id": "call_123",
    "name": "function_name",
    "response": {"result": "value"}
  }
}
```

### Session Resumption Update (Server → Client)

```json
{
  "session_resumption_update": {
    "resumable": true,
    "new_handle": "session-handle-string"
  }
}
```

## Configuration Options

### Response Modalities

| Value | Description |
|-------|-------------|
| `TEXT` | Text-only responses |
| `AUDIO` | Audio responses (native audio models) |

### Voice Activity Detection

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `disabled` | boolean | false | Disable automatic VAD |
| `start_of_speech_sensitivity` | enum | START_SENSITIVITY_MEDIUM | Sensitivity for detecting speech start |
| `end_of_speech_sensitivity` | enum | END_SENSITIVITY_MEDIUM | Sensitivity for detecting speech end |
| `prefix_padding_ms` | integer | 0 | Audio to include before detected speech |
| `silence_duration_ms` | integer | 500 | Silence duration to trigger speech end |

### Sensitivity Levels

- `START_SENSITIVITY_LOW` - Requires more audio energy to start
- `START_SENSITIVITY_MEDIUM` - Balanced sensitivity
- `START_SENSITIVITY_HIGH` - More sensitive to quiet speech

- `END_SENSITIVITY_LOW` - Requires longer silence to end
- `END_SENSITIVITY_MEDIUM` - Balanced
- `END_SENSITIVITY_HIGH` - Ends quickly after silence

### Session Resumption

```json
{
  "session_resumption": {
    "handle": "previous-session-handle"
  }
}
```

**Important Notes:**
- Resumption tokens are valid for 2 hours after session termination
- Sessions can be kept alive for up to 24 hours

## Supported Tools

| Tool | Supported |
|------|-----------|
| Google Search | Yes |
| Function Calling | Yes |
| Google Maps | No |
| Code Execution | No |
| URL Context | No |

### Tool Configuration

```json
{
  "tools": [
    {"google_search": {}},
    {
      "function_declarations": [
        {
          "name": "function_name",
          "description": "Function description",
          "parameters": {
            "type": "object",
            "properties": {
              "param": {
                "type": "string",
                "description": "Parameter description"
              }
            },
            "required": ["param"]
          }
        }
      ]
    }
  ]
}
```

## Audio Formats

### Input

| Property | Value |
|----------|-------|
| Format | PCM |
| Sample Rate | 16000 Hz |
| Bit Depth | 16-bit |
| Channels | Mono |
| MIME Type | `audio/pcm;rate=16000` |

### Output

| Property | Value |
|----------|-------|
| Format | PCM |
| Sample Rate | 24000 Hz |
| Bit Depth | 16-bit |
| Channels | Mono |
| MIME Type | `audio/pcm;rate=24000` |

## Session Management

### Start New Session

Connect to WebSocket without session handle.

### Resume Session

1. Store the `new_handle` from `SessionResumptionUpdate` messages
2. On reconnect, pass the handle in the setup:

```json
{
  "setup": {
    "model": "gemini-2.5-flash-native-audio-preview-12-2025",
    "session_resumption": {
      "handle": "stored-handle"
    }
  }
}
```

### Session Lifecycle Events

1. **Connection established** - WebSocket connected
2. **Setup complete** - Configuration accepted
3. **Active** - Exchanging messages
4. **Turn complete** - User turn finished
5. **Generation complete** - Model finished responding
6. **Interrupted** - User interrupted (VAD)
7. **Closed** - Connection terminated

## Error Handling

### Error Message Format

```json
{
  "error": {
    "code": 429,
    "message": "Rate limit exceeded",
    "status": "RESOURCE_EXHAUSTED"
  }
}
```

### Common Error Codes

| Code | Status | Description |
|------|--------|-------------|
| 400 | INVALID_ARGUMENT | Invalid request format |
| 401 | UNAUTHENTICATED | Invalid API key |
| 403 | PERMISSION_DENIED | Access denied |
| 404 | NOT_FOUND | Model not found |
| 429 | RESOURCE_EXHAUSTED | Rate limit exceeded |
| 500 | INTERNAL_ERROR | Server error |

## SDK Reference

### Python SDK

```python
from google import genai
from google.genai import types

client = genai.Client()

# Connect to Live API
session = client.aio.live.connect(
    model="gemini-2.5-flash-native-audio-preview-12-2025",
    config={...}
)

# Send realtime input
await session.send_realtime_input(audio=...)

# Receive responses
async for msg in session.receive():
    ...

# Send client content
await session.send_client_content(turns=..., turn_complete=True)

# Handle tool calls
await session.send_tool_response(...)

# Close session
session.close()
```

### JavaScript SDK

```javascript
import { GoogleGenAI } from '@google/genai';

const ai = new GoogleGenAI({});

// Connect to Live API
const session = await ai.live.connect({
  model: 'gemini-2.5-flash-native-audio-preview-12-2025',
  config: {...},
  callbacks: {...}
});

// Send realtime input
session.sendRealtimeInput({ audio: ... });

// Send client content
session.sendClientContent({ turns: ..., turnComplete: true });

// Handle tool calls
session.sendToolResponse(...);

// Close session
session.close();
```

### Go SDK

```go
import "google.golang.org/genai"

// Create client
client, err := genai.NewClient(ctx, &genai.ClientConfig{...})
if err != nil {
    log.Fatal(err)
}

// Use LiveConnect for WebSocket connections
// See Go SDK documentation for details
```

## Rate Limits

- See Google AI Studio for current rate limits
- Use ephemeral tokens with usage limits for production

## Best Practices

1. **Always handle interruptions** - Check for `interrupted` flag
2. **Implement session resumption** - For reliable long conversations
3. **Use appropriate audio format** - 16kHz PCM for input
4. **Monitor turn completion** - Use `turnComplete` and `generationComplete`
5. **Handle tool responses** - Required for function calling

## Related Documentation

- [API Live Reference](api-live-reference.md)
- [Live API Capabilities](capabilities.md)
- [Build with Live API](build-with-live-api.md)
- [Session Management](session-management.md)
- [Tools with Live API](tools.md)
- [Ephemeral Tokens](ephemeral-tokens.md)

---

*Generated from Google AI Developer Documentation*
