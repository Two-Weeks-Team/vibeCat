# Live-First Hardening 2026-03-11

## Goal

Align `voice + text + search + tool use + bubble UX` around a single Live-first runtime so the demo feels trustworthy, responsive, and visibly grounded.

This document records the state after deploying:

- `realtime-gateway-00039-xw7`
- `adk-orchestrator-00037-psz`
- git commit `234004f`

## What Changed

### 1. Single routing policy

- `plain_live`: normal chat goes directly to Gemini Live.
- `live_search`: freshness/search queries use Live-native Google Search.
- `adk_tool`: `maps`, `url_context`, `code_execution`, `file_search` remain ADK-driven, then get injected back into Live for the spoken reply.

This removes the old split where voice and text could take different search paths.

### 2. User-visible waiting states

The gateway now emits `processingState` messages so the client can show a non-silent waiting bubble:

- `기억 불러오는 중...`
- `화면 읽는 중...`
- `검색 중...`
- `근거 확인 중...`
- `도구 실행 중...`
- `답변 정리 중...`
- `다시 연결 중...`

The bubble switches from `status` mode to `speech` mode as soon as real output begins.

### 3. Grounding metadata surfaced to UI

The client now accepts:

- `processingState`
- `toolResult`
- `goAway`

This allows the UI to show metadata such as:

- `Google Search · 근거 3개`
- `URL Context`
- `Code Execution`

### 4. Session resumption hardening

- Empty `sessionResumptionUpdate` handles no longer wipe out a valid handle on the client.
- The resume handle remains reusable across reconnects.

### 5. Bubble layout hardening

- Bubble text is measured using its actual rendered width.
- Speech bubbles keep metadata below body text.
- Status bubbles render spinner + text as one centered block.
- Bubble content is vertically centered inside the body instead of sticking to the top edge.

## Active Runtime Parameters

### Client

| Parameter | Current value | Where |
|---|---:|---|
| `bargeInThreshold` | `0.04` | `SpeechRecognizer.swift` |
| `consecutiveThreshold` | `2` | `SpeechRecognizer.swift` |
| `minimumSpeechGap` | `3.0s` | `AppDelegate.swift` |
| `bubbleDuration` | `2.0s` | `CatPanel.swift` |
| `maxBubbleDisplayTime` | `15.0s` | `CatPanel.swift` |
| `chattiness` | user setting, default `normal` | `Settings.swift` / `GatewayClient.swift` |

### Gateway / Live

Current active profile: `baseline`

| Parameter | Baseline | Memory-light experiment | VAD-relaxed experiment |
|---|---:|---:|---:|
| `MaxMemoryChars` | `1200` | `900` | `1200` |
| `PrefixPaddingMs` | `20` | `20` | `40` |
| `SilenceDurationMs` | `200` | `200` | `250` |
| `CompressionTriggerTokens` | `12000` | `10000` | `12000` |
| `CompressionTargetTokens` | `6000` | `5000` | `6000` |

Fixed VAD sensitivities:

- `StartOfSpeechSensitivity = Low`
- `EndOfSpeechSensitivity = Low`

Cache:

- `memoryContextCacheTTL = 5m`

## Best-Practice Alignment

### Live API

Aligned with official guidance:

- Keep one long-lived Live session per interaction lane.
- Use built-in tools where Live supports them directly.
- Feed compact memory summaries instead of raw history.
- resume via `sessionResumptionUpdate`.
- expose grounding to reduce invisible model behavior.

References:

- [Live API Best Practices](https://ai.google.dev/gemini-api/docs/live-api/best-practices)
- [Session Management](https://ai.google.dev/gemini-api/docs/live-api/session-management)
- [Live API Tools](https://ai.google.dev/gemini-api/docs/live-api/tools)
- [Google Search Tool](https://ai.google.dev/gemini-api/docs/google-search)

### Challenge fit

This is better for the challenge because it increases all three judging signals:

- multimodal clarity: screen + speech + bubble now tell one coherent story
- technical rigor: one authoritative real-time runtime instead of split search paths
- demo readability: waiting states and grounding metadata are visible on screen

## Verified

### Local tests

- `go test ./...` in `backend/realtime-gateway`
- `go test ./...` in `backend/adk-orchestrator`
- `swift test`
- `make build`

### Real API E2E

- health
- auth register / refresh
- websocket upgrade
- Live setup
- `/search`
- `/tool`
- tool auto-response
- Live-native search processing state
- session resumption
- memory save / retrieve

Voice barge-in automation remains skipped unless `E2E_VOICE_BARGE_IN` is explicitly enabled.

## Remaining High-Value Experiments

These are the next candidates, ordered by likely value for judging impact:

1. Adaptive barge-in threshold
   - replace fixed `0.04` with rolling noise-floor estimation per input device
   - expected win: fewer headset-dependent misses

2. Auto-select tuning profile
   - compare `baseline`, `memory_light`, `vad_relaxed` against live telemetry
   - promote the profile only if `p95 turn_complete` improves without barge-in regressions

3. Bubble-side source chips
   - render compact chips like `Search`, `Maps`, `Code`, `3 sources`
   - expected win: demo viewers instantly understand “this was grounded”

4. Progress escalation for long waits
   - if no model output after `1.2s`, show status bubble
   - if still waiting after `3.5s`, switch detail text to a more explicit stage
   - expected win: less “dead air” perception during search or tool execution

5. Prompt tightening by chattiness
   - `quiet`: enforce one-sentence cap more strongly
   - `chatty`: allow at most one additional concrete suggestion, not generic filler
   - expected win: cleaner demo pacing
