# Gemini Live API Documentation Index

**Project:** VibeCat  
**Purpose:** Real-time voice and vision interactions with Gemini  
**Extracted:** 2026-03-10

---

## Documentation Structure

```
docs/reference/
├── gemini-live/                    # Live API specific documentation
│   ├── NAV_STRUCTURE.md            # Full navigation structure
│   ├── INDEX.md                    # This file
│   ├── 01-live-api-overview.md     # Live API Overview
│   ├── 02-get-started-sdk.md         # GenAI SDK tutorial
│   ├── 03-get-started-websocket.md   # WebSocket tutorial
│   └── (more files to be added)    # Other Live API pages
│
└── gemini-api/                     # General Gemini API documentation
    └── audio-understanding.md      # Audio understanding guide
```

---

## Quick Reference

### Live API Core Documentation

| Document | Description | Priority |
|----------|-------------|----------|
| [01-live-api-overview.md](01-live-api-overview.md) | Overview of Live API features, use cases, technical specs | ⭐⭐⭐ HIGH |
| [02-get-started-sdk.md](02-get-started-sdk.md) | GenAI SDK tutorial for server-to-server | ⭐⭐⭐ HIGH |
| [03-get-started-websocket.md](03-get-started-websocket.md) | WebSocket tutorial for client-to-server | ⭐⭐⭐ HIGH |

### Related API Documentation

| Document | Description | Location |
|----------|-------------|----------|
| [Audio Understanding](../gemini-api/audio-understanding.md) | Audio file processing, transcription | gemini-api/ |

---

## Key Technical Specifications

### Audio Format

| Direction | Format | Sample Rate | Bit Depth | Endianness |
|-----------|--------|-------------|-----------|------------|
| **Input** | Raw PCM | 16 kHz | 16-bit | Little-endian |
| **Output** | Raw PCM | 24 kHz | 16-bit | Little-endian |

### Video/Image

- Format: JPEG
- Rate: <= 1 FPS (frames per second)

### Protocol

- **WebSocket**: Stateful WebSocket connection (WSS)
- **Endpoint (API Key)**: `wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1beta.GenerativeService.BidiGenerateContent?key=YOUR_API_KEY`
- **Endpoint (Ephemeral Token)**: `wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1alpha.GenerativeService.BidiGenerateContentConstrained?access_token={token}`

### Supported Models

- `gemini-2.5-flash-native-audio-preview-12-2025`
- `gemini-3-flash-preview` (for non-live audio)

---

## Implementation Approaches

### 1. Server-to-Server (GenAI SDK)

- **Best for**: Production applications, secure API key management
- **Approach**: Backend connects to Live API using WebSockets
- **Security**: API keys stored securely on server
- **Guide**: [02-get-started-sdk.md](02-get-started-sdk.md)

### 2. Client-to-Server (WebSocket)

- **Best for**: Lower latency, real-time streaming
- **Approach**: Frontend connects directly to Live API
- **Security**: Use ephemeral tokens for production
- **Guide**: [03-get-started-websocket.md](03-get-started-websocket.md)

---

## Key Features

1. **Multilingual support**: 70 supported languages
2. **Barge-in**: Users can interrupt the model at any time
3. **Tool use**: Function calling and Google Search integration
4. **Audio transcriptions**: Text transcripts of user input and model output
5. **Proactive audio**: Control when the model responds
6. **Affective dialog**: Adapt response style and tone to user expression

---

## Go Backend Integration

For VibeCat's Go backend, the Live API integration involves:

1. **GenAI SDK for Go**: Use `google.golang.org/genai` package
2. **WebSocket Proxy**: Server-to-server WebSocket connection
3. **Audio Format Conversion**: Convert between client audio (16kHz) and Live API requirements
4. **Session Management**: Handle long-running conversations
5. **Tool Integration**: Connect to ADK Orchestrator for 9-agent graph

### Relevant Go Packages

- `google.golang.org/genai`: Official GenAI SDK for Go
- `google.golang.org/adk`: Agent Development Kit for Go

---

## External Resources

### Official Documentation

- [Gemini API Docs](https://ai.google.dev/gemini-api/docs)
- [Live API Overview](https://ai.google.dev/gemini-api/docs/live-api)
- [WebSocket API Reference](https://ai.google.dev/api/live)

### Example Repositories

- [Gemini Live API Examples](https://github.com/google-gemini/gemini-live-api-examples)
- [Gemini Cookbook](https://github.com/google-gemini/cookbook)

### Tools & Integrations

- [Google AI Studio](https://aistudio.google.com/live): Try Live API interactively
- [LiveKit](https://docs.livekit.io/agents/models/realtime/plugins/gemini/)
- [Pipecat](https://docs.pipecat.ai/guides/features/gemini-live)
- [Fishjam](https://docs.fishjam.io/tutorials/gemini-live-integration)

---

## VibeCat Implementation Notes

### Audio Pipeline

```
macOS Client (Swift)
    ↓ (PCM 16kHz 16-bit)
VibeCat Transport Layer
    ↓ (WebSocket)
Realtime Gateway (Go)
    ↓ (WebSocket)
Gemini Live API
    ↓ (PCM 24kHz 16-bit)
VibeCat Transport Layer
    ↓ (WebSocket)
macOS Client (Swift)
```

### Key Integration Points

1. **VAD (Voice Activity Detection)**: Configured via Live API's `automaticActivityDetection`
2. **Session Management**: Long-running WebSocket sessions with reconnection logic
3. **Ephemeral Tokens**: Generated by backend for secure client connections
4. **Tool Use**: Integration with ADK Orchestrator's 9-agent graph
5. **Character Voices**: Native audio output with custom voice presets

---

*Generated: 2026-03-10*
*For VibeCat Project - Gemini Live Agent Challenge*
