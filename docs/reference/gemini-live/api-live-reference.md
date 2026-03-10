# Live API - WebSockets API Reference

**Source:** https://ai.google.dev/api/live

The Live API is a stateful API that uses WebSockets. This section provides additional details regarding the WebSockets API.

## Sessions

A WebSocket connection establishes a session between the client and the Gemini server. After a client initiates a new connection the session can exchange messages with the server to:

- Receive audio, text, or function call requests from the Gemini server.
- Send text, audio, or video to the Gemini server.

### WebSocket Connection

To start a session, connect to this websocket endpoint:

```
wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1beta.GenerativeService.BidiGenerateContent
```

### Session Configuration

The initial message sent after establishing the WebSocket connection sets the session configuration, which includes the model, generation parameters, system instructions, and tools.

You cannot update the configuration while the connection is open. However, you can change the configuration parameters, except the model, when pausing and resuming a session.

## API Endpoints

### WebSocket /v1beta/models/{model}:bidiGenerateContent

A stateful WebSocket-based API for bi-directional streaming, designed for real-time conversational use cases. This Live API provides low-latency interactions suitable for applications requiring immediate responses.

#### Method
WebSocket

#### Endpoint
`wss://generativelanguage.googleapis.com/v1beta/models/{model}:bidiGenerateContent`

#### Parameters

**Path Parameters:**
- **model** (string) - Required - The model identifier (e.g., gemini-2.5-flash)

**Headers:**
- **x-goog-api-key** (string) - Required - Your API key for authentication

#### Connection
- Establishes a persistent WebSocket connection for bi-directional communication
- Supports real-time message exchange
- Maintains conversation state across multiple exchanges

#### Use Cases
- Real-time chatbots
- Interactive conversational applications
- Low-latency AI interactions

## Supported Models

The Live API uses specialized models optimized for real-time interactions:

- `gemini-2.5-flash-native-audio-preview-12-2025` - Latest native audio model

## Message Types

### Client to Server Messages

#### Config Message
Sets up the session configuration at the start of the connection.

```json
{
  "setup": {
    "model": "gemini-2.5-flash-native-audio-preview-12-2025",
    "generation_config": {
      "response_modalities": ["AUDIO"]
    }
  }
}
```

#### Realtime Input Message
Send audio, video, or text in real-time.

```json
{
  "realtime_input": {
    "media_chunks": [
      {
        "data": "base64encodedAudioBytes",
        "mime_type": "audio/pcm;rate=16000"
      }
    ]
  }
}
```

#### Client Content Message
Send text content as part of the conversation.

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

### Server to Client Messages

#### Server Content Message
Receive model responses including audio, text, and function calls.

```json
{
  "server_content": {
    "model_turn": {
      "parts": [
        {
          "text": "Hello! How can I help you today?"
        },
        {
          "inline_data": {
            "mime_type": "audio/pcm;rate=24000",
            "data": "base64encodedAudioBytes"
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

#### Session Resumption Update
Contains the session handle for resumption.

```json
{
  "session_resumption_update": {
    "resumable": true,
    "new_handle": "session-handle-string"
  }
}
```

## Audio Format Specifications

### Input Audio (Client to Server)
- **Format:** PCM (Pulse Code Modulation)
- **Sample Rate:** 16000 Hz
- **Bit Depth:** 16-bit
- **Channels:** Mono
- **MIME Type:** `audio/pcm;rate=16000`

### Output Audio (Server to Client)
- **Format:** PCM
- **Sample Rate:** 24000 Hz
- **Bit Depth:** 16-bit
- **Channels:** Mono
- **MIME Type:** `audio/pcm;rate=24000`

## Voice Activity Detection (VAD)

The Live API supports automatic Voice Activity Detection (VAD) for detecting when users start and stop speaking.

### VAD Configuration Options

```python
from google.genai import types

config = {
    "response_modalities": ["TEXT"],
    "realtime_input_config": {
        "automatic_activity_detection": {
            "disabled": False,  # default
            "start_of_speech_sensitivity": types.StartSensitivity.START_SENSITIVITY_LOW,
            "end_of_speech_sensitivity": types.EndSensitivity.END_SENSITIVITY_LOW,
            "prefix_padding_ms": 20,
            "silence_duration_ms": 100,
        }
    }
}
```

### Disable Automatic VAD

To take control of VAD yourself, disable automatic detection:

```json
{
  "setup": {
    "realtime_input_config": {
      "automatic_activity_detection": {
        "disabled": true
      }
    }
  }
}
```

When disabled, client must send:
- `activityStart` - When user starts speaking
- `activityEnd` - When user stops speaking or interrupts

## Interruption Handling

When VAD detects an interruption, the ongoing generation is canceled and a ServerContent message is sent with `interrupted: true`:

```python
async for response in session.receive():
    if response.server_content.interrupted is True:
        # The generation was interrupted
        # Stop audio playback and clear queued playback here
```

## Turn Management

### Detecting Turn Complete

```python
async for response in session.receive():
    if response.server_content.turn_complete is True:
        # User's turn is complete, model is responding
```

### Detecting Generation Complete

```python
async for response in session.receive():
    if response.server_content.generation_complete is True:
        # The generation is complete
```

## Related Documentation

- [Live API Capabilities Guide](capabilities.md)
- [Build with Live API](build-with-live-api.md)
- [Live API Reference](live-api-reference.md)
- [Session Management](session-management.md)
- [Tools with Live API](tools.md)
- [Ephemeral Tokens](ephemeral-tokens.md)

---

*Generated from Google AI Developer Documentation*
