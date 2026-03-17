# P0-1: ThinkingConfig + Thought Signatures

## Status

Implement now in the Live gateway, but split it into two stages:

- Stage 1: enable/tune thinking on the Live session and measure it
- Stage 2: optionally surface thought summaries in the client UI behind a debug or feature flag

## Source-Verified Facts

- `genai.LiveConnectConfig` in `google.golang.org/genai v1.49.0` has `ThinkingConfig *ThinkingConfig`.
- The current Live model `gemini-2.5-flash-native-audio-preview-12-2025` officially supports thinking, with dynamic thinking enabled by default.
- `genai.Part` includes `Thought bool` and `ThoughtSignature []byte`.
- Official thinking docs say the Google GenAI SDK automatically handles thought-signature return unless you manually modify conversation history.

## Current Repo State

- `backend/realtime-gateway/internal/live/session.go`
  - `buildLiveConfig()` does not set `ThinkingConfig`
- `backend/realtime-gateway/internal/ws/handler.go`
  - `receiveFromGemini()` does not inspect `Part.Thought` or thought-signature fields
- The Swift client already has a generic visual thinking state through sprite/status UI
- There is no verified end-user thought-summary protocol yet

## Implementation Decision

### Stage 1: ship

1. Add a Live thinking feature flag.
2. Set `ThinkingConfig` in `buildLiveConfig()` when enabled.
3. Start with an explicit `ThinkingBudget` value and log token usage.
4. Record `UsageMetadata.ThoughtsTokenCount` in gateway logs/metrics.
5. Do not change user-facing UI copy until actual thought parts are observed in production or staging logs.

### Stage 2: optional

1. If the current Live session returns `Part.Thought` text reliably, forward it as a new debug-oriented message type.
2. Keep this off by default in production.
3. Never expose raw thought text as required UX for task completion.

## Concrete File Changes

### `backend/realtime-gateway/internal/live/session.go`

- Extend `live.Config` with a thinking control:
  - `EnableThinking bool`
  - `ThinkingBudget int32`
- In `buildLiveConfig()`:
  - set `ThinkingConfig` only when `EnableThinking` is true
  - start with `ThinkingBudget = 1024`
- Keep the default disabled behind a feature flag until staging verification passes

### `backend/realtime-gateway/internal/ws/handler.go`

- In `receiveFromGemini()`:
  - inspect `sc.ModelTurn.Parts`
  - when `part.Thought == true`, log it at debug level
  - when `len(part.ThoughtSignature) > 0`, log presence/count only
- In `UsageMetadata` logging:
  - include `ThoughtsTokenCount`

### Optional client changes

- Add `thinking(text)` only if Stage 2 is enabled
- Render in existing status bubble style, not as a blocking modal

## Non-Goals

- Manual thought-signature persistence for the Live session
- Reconstructing or storing internal reasoning across sessions
- Shipping raw chain-of-thought as a primary UI feature

## Acceptance Criteria

1. Live session connects successfully with thinking enabled on the current model.
2. Gateway logs show `thoughts_token_count` for turns where thinking is used.
3. No regressions in audio streaming, tool calling, or reconnect flow.
4. If Stage 2 is enabled, `Part.Thought` messages are actually observed before UI is exposed broadly.

## Risks

- Increased token cost and latency
- Thought summaries may not appear in every turn even when thinking is enabled
- End-user thought UI can create noisy UX if shown indiscriminately

## Sources

- [Gemini Live API guide](https://ai.google.dev/gemini-api/docs/live-guide)
- [Gemini thinking guide](https://ai.google.dev/gemini-api/docs/thinking)
