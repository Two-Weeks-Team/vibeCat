# Live API

Last checked: 2026-03-10

Use this file when changing realtime audio/video transport, session resumption, VAD, Live tool use, or auth.

## Core takeaways

- Live API remains a preview surface and model IDs can move more often than the unary `generateContent` surface.
- The current docs treat direct client connections as a production case only when ephemeral tokens are used.
- Live sessions are single-modality on output: choose `TEXT` or `AUDIO`.
- Native audio output models have specific language and audio-format rules that differ from text-only TTS.
- Session resumption, context window compression, and interruption handling are all first-class parts of a production integration.

## Current snapshot details

- Current Live guides emphasize:
  - 16-bit PCM mono input
  - 24 kHz output audio
  - one output modality per session
- Current limits called out in the Live docs:
  - audio-only sessions: 15 minutes without compression
  - audio+video sessions: 2 minutes without compression
  - connection lifetime: about 10 minutes
- Current context-window guidance:
  - 128k for native-audio output models
  - 32k for other Live models
- Current resumption guidance:
  - resumption handles remain valid for 2 hours after the last session termination
- The current capabilities and tools docs call out `v1alpha` for:
  - affective dialog
  - proactive audio
  - ephemeral token provisioning
- The current Live tool docs list support on `gemini-2.5-flash-native-audio-preview-12-2025` for:
  - Google Search: yes
  - function calling: yes
  - Google Maps: no
  - Code Execution: no
  - URL Context: no

## Implementation rules

- Re-check Live model IDs before edits. Do not trust memory here.
- Keep 16-bit PCM input and current output-audio assumptions aligned with the guide you are using.
- Clear queued audio immediately on interruption events.
- Use `v1alpha` when the feature requires it, including affective dialog, proactive audio, and ephemeral-token provisioning.
- Do not assume all unary built-in tools are available in Live sessions.
- Do not set an explicit language code for native-audio output models unless the current official guide changes.

## Original sources

- Live API overview: [source](https://ai.google.dev/gemini-api/docs/live-api), [snapshot](originals/live-api/01_live_api_overview.html)
- Get started with SDK: [source](https://ai.google.dev/gemini-api/docs/live-api/get-started-sdk), [snapshot](originals/live-api/02_get_started_sdk.html)
- Get started with WebSocket: [source](https://ai.google.dev/gemini-api/docs/live-api/get-started-websocket), [snapshot](originals/live-api/03_get_started_websocket.html)
- Capabilities: [source](https://ai.google.dev/gemini-api/docs/live-api/capabilities), [snapshot](originals/live-api/04_capabilities.html)
- Tools: [source](https://ai.google.dev/gemini-api/docs/live-api/tools), [snapshot](originals/live-api/05_tools.html)
- Session management: [source](https://ai.google.dev/gemini-api/docs/live-api/session-management), [snapshot](originals/live-api/06_session_management.html)
- Ephemeral tokens: [source](https://ai.google.dev/gemini-api/docs/live-api/ephemeral-tokens), [snapshot](originals/live-api/07_ephemeral_tokens.html)
- Best practices: [source](https://ai.google.dev/gemini-api/docs/live-api/best-practices), [snapshot](originals/live-api/08_best_practices.html)
