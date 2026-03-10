# VibeCat Notes

Last checked against repo: 2026-03-10

Use this file only when working in the `vibeCat` repo or in a repo with the same architectural shape.

## Current repo shape

- Swift client in `VibeCat/`
- Go realtime gateway in `backend/realtime-gateway/`
- Go orchestrator in `backend/adk-orchestrator/`
- Gemini calls are backend-side, which matches the project rules in `AGENTS.md`

## Current Gemini usage in code

### SDKs

- Go SDK: `google.golang.org/genai`
- Go ADK: `google.golang.org/adk`
- Current module version in both backend services: `google.golang.org/genai v1.48.0`

### Live API

- File: `backend/realtime-gateway/internal/live/session.go`
- Current default Live model: `gemini-2.5-flash-native-audio-latest`
- The gateway already configures:
  - audio output modality
  - prebuilt voice selection
  - affective dialog and proactive audio toggles
  - automatic activity detection
  - output and input audio transcription
  - context window compression
  - session resumption
- The gateway intentionally does not attach Google Search directly to the Live session because it would add latency on every voice turn. Search is handled in the orchestrator instead.

### TTS

- File: `backend/realtime-gateway/internal/tts/client.go`
- Current TTS model: `gemini-2.5-flash-preview-tts`
- Current voice default: `Zephyr`
- Current code explicitly maps app languages to `ko-KR` and `en-US`

### Search/tool usage

- File: `backend/adk-orchestrator/internal/agents/search/search.go`
- Current search model: `gemini-3.1-flash-lite-preview`
- Current search path uses the `GoogleSearch` tool in `GenerateContentConfig`
- This means the repo currently mixes Gemini 2.5 Live/TTS surfaces with a Gemini 3.x-style search model

## Codebase-fit rules for this repo

- Preserve the backend-only Gemini boundary unless the user explicitly asks for architectural migration.
- If you update Live model IDs, also re-check:
  - interruption handling in `backend/realtime-gateway/internal/ws/handler.go`
  - session-resumption handling in `backend/realtime-gateway/internal/live/session.go`
  - audio parsing in `VibeCat/Sources/Core/AudioMessageParser.swift`
  - microphone suppression behavior in `VibeCat/Sources/VibeCat/GatewayClient.swift`
- If you update TTS model or voice handling, also re-check:
  - `backend/realtime-gateway/internal/tts/client.go`
  - character voice presets in `Assets/Sprites/*/preset.json`
- If you update search/tool behavior, also re-check:
  - `backend/adk-orchestrator/internal/agents/search/search.go`
  - any PRD or architecture docs that mention tool latency or agent responsibilities

## Immediate watchlist

- `gemini-2.5-flash-native-audio-latest` is convenient but volatile. Verify the concrete model behind that alias before changing Live behavior.
- `gemini-3.1-flash-lite-preview` is a preview identifier already outside the user-supplied document set. Re-check the models page and changelog before keeping or replacing it.
- The repo relies on Live-session interruption behavior for barge-in. Any model migration here needs explicit verification of interruption semantics.
