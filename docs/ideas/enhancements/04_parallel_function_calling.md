# P0-4: Multi-Call Function Handling

## Status

Do not implement parallel desktop UI execution. First fix the current multi-call overwrite bug by serializing multiple function calls returned in one Live response.

## Why This Document Changed

The original draft treated this as a speed optimization. In the current repo it is first a correctness problem.

## Source-Verified Facts

- Live API supports multi-tool use.
- A Live response can contain multiple function calls.
- VibeCat's current Live navigator state stores only one pending function call sequence at a time.

## Current Repo Risk

`backend/realtime-gateway/internal/ws/handler.go`

- `handleLiveToolCall()` loops over every returned function call
- each handler can call `ls.setPendingFC(...)`
- `liveSessionState` stores only one pending FC:
  - `pendingFCID`
  - `pendingFCName`
  - `pendingFCTaskID`
  - `pendingFCSteps`

Result:

- if Gemini returns multiple navigator function calls in one response, the later call can overwrite the earlier one before it completes

## Implementation Decision

### Phase 1: serialize

Implement a queue of pending Live function calls and process them in order.

Rules:

- only one desktop navigator action chain runs at a time
- preserve response order from Gemini
- do not start call N+1 until call N is fully resolved or explicitly skipped

### Phase 2: optional future parallelism

Only consider parallel execution when all of the following are true:

- action types are read-only or non-UI
- targets are independent
- per-surface locking exists
- replay tests prove no race conditions

This does not apply to current desktop navigation actions.

## Concrete File Changes

### `backend/realtime-gateway/internal/ws/handler.go`

- replace single pending FC fields with:
  - active FC
  - queued FC list
- on Live tool call with `len(functionCalls) > 1`:
  - enqueue all
  - dispatch first only
- when one FC completes:
  - pop next queued FC
  - dispatch it

### `liveSessionState`

Add a queue type, for example:

- function call metadata
- prebuilt navigator steps
- original call ID and name

## Explicit Non-Goal

Do not launch multiple UI actions concurrently on macOS in v0.2.

Examples that must remain serialized:

- focus app -> type text
- open URL -> verify page -> click
- focus terminal -> paste command -> submit

## Acceptance Criteria

1. A single Live response containing multiple function calls no longer loses earlier calls.
2. Desktop actions still run one chain at a time.
3. Replay tests cover a two-call and three-call response.

## Risks

- Queue bugs can strand a call in pending state
- Cancellation behavior must clear both active and queued entries

## Sources

- [Gemini function calling guide](https://ai.google.dev/gemini-api/docs/function-calling)
