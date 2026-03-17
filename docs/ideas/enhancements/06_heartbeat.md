# P0-6: Heartbeat Pattern

## Status

Implement as a refactor and tuning pass over the existing keepalive behavior.

## Current Repo State

The gateway already sends silent PCM periodically.

- `backend/realtime-gateway/internal/ws/handler.go`
  - starts a keepalive goroutine
  - currently sends `320` bytes of silence every `15s`

This means the feature is not new. The work is to make it correct, observable, and session-owned.

## Target Behavior

1. The heartbeat belongs to the `live.Session` lifecycle, not the outer handler loop.
2. Silent audio is sent only when the session has been idle long enough.
3. Heartbeat stops automatically when the session closes or reconnects.
4. Reconnection logic remains the fallback when heartbeat is insufficient.

## Implementation Decision

### Move ownership to `live.Session`

Refactor:

- start heartbeat from `Manager.Connect()`
- store `lastAudioSentAt` on the session
- update that timestamp in `SendAudio()`

### Add idle suppression

Only send silence when:

- no user audio has been sent recently
- model is not actively streaming a turn that would make the heartbeat redundant

### Keep cadence configurable

Use env/config instead of hardcoding.

Recommended starting values:

- interval: `10s` or `15s`
- silent chunk: `320` bytes

## Concrete File Changes

### `backend/realtime-gateway/internal/live/session.go`

- add heartbeat state to `Session`
- start heartbeat after successful Live connect
- update `lastAudioSentAt` inside `SendAudio()`

### `backend/realtime-gateway/internal/ws/handler.go`

- remove duplicated handler-owned keepalive loop after session refactor
- keep reconnect loop unchanged

## Observability

Add counters/logs for:

- heartbeat sent
- heartbeat skipped because active audio exists
- heartbeat send failure
- reconnects after idle periods

## Acceptance Criteria

1. Idle sessions remain connected for at least 10 minutes in staging.
2. Heartbeat stops after session close.
3. Reconnect count does not increase after refactor.

## Risks

- Sending silence too often adds noise and waste
- Sending silence too rarely may still allow idle disconnects

## Sources

- [Gemini Live API guide](https://ai.google.dev/gemini-api/docs/live-guide)
