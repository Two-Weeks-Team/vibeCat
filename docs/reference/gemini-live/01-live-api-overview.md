# Gemini Live API Overview

**Source:** https://ai.google.dev/gemini-api/docs/live-api  
**Extracted:** 2026-03-10

---

## Overview

The Live API enables low-latency, real-time voice and vision interactions with Gemini. It processes continuous streams of audio, images, and text to deliver immediate, human-like spoken responses, creating a natural conversational experience for your users.

---

## Use Cases

Live API can be used to build real-time voice agents for a variety of industries, including:

- **E-commerce and retail**: Shopping assistants that offer personalized recommendations and support agents that resolve customer issues.
- **Gaming**: Interactive non-player characters (NPCs), in-game help assistants, and real-time translation of in-game content.
- **Next-gen interfaces**: Voice- and video-enabled experiences in robotics, smart glasses, and vehicles.
- **Healthcare**: Health companions for patient support and education.
- **Financial services**: AI advisors for wealth management and investment guidance.
- **Education**: AI mentors and learner companions that provide personalized instruction and feedback.

---

## Key Features

Live API offers a comprehensive set of features for building robust voice agents:

### Multilingual support
Converse in 70 supported languages.

### Barge-in
Users can interrupt the model at any time for responsive interactions.

### Tool use
Integrates tools like function calling and Google Search for dynamic interactions.

### Audio transcriptions
Provides text transcripts of both user input and model output.

### Proactive audio
Lets you control when the model responds and in what contexts.

### Affective dialog
Adapts response style and tone to match the user's input expression.

---

## Technical Specifications

| Category | Details |
|----------|---------|
| **Input modalities** | Audio (raw 16-bit PCM audio, 16kHz, little-endian), images (JPEG <= 1FPS), text |
| **Output modalities** | Audio (raw 16-bit PCM audio, 24kHz, little-endian) |
| **Protocol** | Stateful WebSocket connection (WSS) |

### Audio Format Details

**Input Audio:**
- Format: Raw 16-bit PCM
- Sample Rate: 16kHz
- Endianness: Little-endian
- Channels: Mono

**Output Audio:**
- Format: Raw 16-bit PCM  
- Sample Rate: 24kHz
- Endianness: Little-endian
- Channels: Mono

**Video/Image:**
- Format: JPEG
- Rate: <= 1 FPS (frames per second)

---

## Implementation Approaches

When integrating with Live API, you'll need to choose one of the following implementation approaches:

### Server-to-server
Your backend connects to the Live API using WebSockets. Typically, your client sends stream data (audio, video, text) to your server, which then forwards it to the Live API.

### Client-to-server
Your frontend code connects directly to the Live API using WebSockets to stream data, bypassing your backend.

**Note:** Client-to-server generally offers better performance for streaming audio and video, since it bypasses the need to send the stream to your backend first. It's also easier to set up since you don't need to implement a proxy that sends data from your client to your server and then your server to the API. However, for production environments, in order to mitigate security risks, we recommend using ephemeral tokens instead of standard API keys.

---

## Get Started

Select the guide that matches your development environment:

### Server-to-server: GenAI SDK tutorial
Connect to the Gemini Live API using the GenAI SDK to build a real-time multimodal application with a Python backend.

**URL:** `/gemini-api/docs/live-api/get-started-sdk`

### Client-to-server: WebSocket tutorial
Connect to the Gemini Live API using WebSockets to build a real-time multimodal application with a JavaScript frontend and ephemeral tokens.

**URL:** `/gemini-api/docs/live-api/get-started-websocket`

### Agent development kit: ADK tutorial
Create an agent and use the Agent Development Kit (ADK) Streaming to enable voice and video communication.

**URL:** `https://google.github.io/adk-docs/streaming/`

---

## Partner Integrations

To streamline the development of real-time audio and video apps, you can use a third-party integration that supports the Gemini Live API over WebRTC or WebSockets.

| Partner | Description | URL |
|---------|-------------|-----|
| **LiveKit** | Use the Gemini Live API with LiveKit Agents. | https://docs.livekit.io/agents/models/realtime/plugins/gemini/ |
| **Pipecat by Daily** | Create a real-time AI chatbot using Gemini Live and Pipecat. | https://docs.pipecat.ai/guides/features/gemini-live |
| **Fishjam by Software Mansion** | Create live video and audio streaming applications with Fishjam. | https://docs.fishjam.io/tutorials/gemini-live-integration |
| **Vision Agents by Stream** | Build real-time voice and video AI applications with Vision Agents. | https://visionagents.ai/integrations/gemini |
| **Voximplant** | Connect inbound and outbound calls to Live API with Voximplant. | https://voximplant.com/products/gemini-client |
| **Firebase AI SDK** | Get started with the Gemini Live API using Firebase AI Logic. | https://firebase.google.com/docs/ai-logic/live-api?api=dev |

---

## Related Documentation

- [Live API Capabilities](/gemini-api/docs/live-api/capabilities)
- [Tool Use](/gemini-api/docs/live-api/tools)
- [Session Management](/gemini-api/docs/live-api/session-management)
- [Ephemeral Tokens](/gemini-api/docs/live-api/ephemeral-tokens)
- [Best Practices](/gemini-api/docs/live-api/best-practices)
- [Audio Understanding](/gemini-api/docs/audio)
- [Speech Generation](/gemini-api/docs/speech-generation)

---

*Last updated: 2026-03-09 UTC*
