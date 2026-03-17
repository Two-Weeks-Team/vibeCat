# 13: Progress Display and User Communication

## Status

Implement protocol and UX refinements, not a full rewrite.

## Current Repo State

VibeCat already has most of this system:

- parsed navigator messages
- status bubbles
- overlay panel
- clarification and risky-action prompts
- reconnect status handling

The remaining work is mostly protocol completeness and UI refinement.

## What Is Already Good

- `processingState` gives non-blocking execution visibility
- navigator task lifecycle messages already exist
- risky action and clarification flows are already wired end-to-end
- status bar already tracks connection state

## What Is Still Missing

### 1. Planned-step totals

`NavigatorOverlayPanel.showStep(...)` supports `totalSteps`, but the current call path passes `nil`.

To support real multi-step progress:

- backend must send `totalSteps`, or
- backend must send a new `navigator.planStarted` message containing the full plan length

### 2. Safer confirmation UX

Current risky-action UX is still light-weight. It should expose:

- action summary
- reason
- explicit allow / deny controls

### 3. Thinking UI only if real thought text is present

Do not promise end-user thought summaries by default.

If doc 01 Stage 2 is enabled:

- add `thinking(text)` message type
- keep it debug-only or feature-flagged first

### 4. Connection visibility near the cat panel

Status bar already knows connection state. A cat-panel indicator is optional polish.

## Concrete Protocol Changes

### Backend

- extend `navigator.stepPlanned` payload with:
  - `stepNumber`
  - `totalSteps`

or add:

- `navigator.planStarted`
  - task id
  - command
  - total steps

### Client

- `VibeCat/Sources/Core/AudioMessageParser.swift`
  - parse the new metadata
- `VibeCat/Sources/VibeCat/AppDelegate.swift`
  - pass actual totals into overlay panel
- `VibeCat/Sources/VibeCat/NavigatorOverlayPanel.swift`
  - render real multi-step progress

## Suggested Priority

1. total-steps protocol
2. risky confirmation UI
3. optional thought text UI
4. optional cat-panel connection dot

## Acceptance Criteria

1. A 3-step navigator task shows correct `1/3`, `2/3`, `3/3`.
2. Risk confirmation uses explicit allow / deny controls.
3. Reconnect state remains visible during session recovery.

## Risks

- adding many transient UI states can make the panel noisy
- thought text can distract if enabled without strong filtering

## Sources

- current repo message protocol in `AudioMessageParser.swift`
