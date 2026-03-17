# 12: Expand Navigator FC Tools

## Status

Implement in phases. Start with `navigate_click`, then `navigate_scroll`.

## Current Repo State

The Swift runtime already supports more action types than Gemini can currently call.

### Already implemented in Swift

- `paste_text`
- `copy_selection`
- `press_ax`
- `click_coordinates`
- `system_action`
- `wait_for`

### Already exposed as Gemini FC

- `navigate_text_entry`
- `navigate_hotkey`
- `navigate_focus_app`
- `navigate_open_url`
- `navigate_type_and_submit`

## The Actual Gap

The main gap is not raw runtime capability. It is FC exposure plus data-contract completeness.

Most importantly:

- `VibeCat/Sources/Core/AudioMessageParser.swift`
  - currently parses `NavigatorTargetDescriptor` without `clickX` / `clickY`
  - does not populate `screenBasisId`
  - does not populate `verificationCue`

Without that parser fix, coordinate-driven FC additions will not round-trip correctly to the client.

## Recommended Tool Design

Avoid overloading one tool with many unrelated modes. Use explicit tool names aligned with runtime actions.

### v0.2

1. `navigate_click`
2. parser/data-contract completion for coordinate clicks

### v0.3

1. `navigate_scroll`
2. `navigate_copy_selection`
3. `navigate_system_action`

## Tool 1: `navigate_click`

### Behavior

- preferred path: semantic/AX target (`press_ax`)
- fallback path: normalized coordinates (`click_coordinates`)

### Required parameters

- `target` when semantic click is intended
- `x`, `y` when coordinate click is intended
- optional `app`
- optional `double_click`
- optional `screen_basis_id`

### Backend changes

- `backend/realtime-gateway/internal/live/session.go`
  - add function declaration
- `backend/realtime-gateway/internal/ws/handler.go`
  - add handler
- `backend/realtime-gateway/internal/ws/navigator.go`
  - add step builder that prefers `press_ax`

### Client changes

- `VibeCat/Sources/Core/AudioMessageParser.swift`
  - parse `clickX`, `clickY`, `screenBasisId`, `verificationCue`

## Tool 2: `navigate_scroll`

### Behavior

- scroll active surface or frontmost app
- start with vertical scroll only

### Required runtime work

- add `scroll` to `NavigatorActionType`
- implement in `AccessibilityNavigator.execute(step:)`

## Do Not Ship Yet

- drag
- right click
- one giant `copy_paste` multipurpose tool

Those are valid later, but they are not required to unlock the current blocked flows.

## Concrete File Changes

### Backend

- `backend/realtime-gateway/internal/live/session.go`
- `backend/realtime-gateway/internal/ws/handler.go`
- `backend/realtime-gateway/internal/ws/navigator.go`

### Swift client

- `VibeCat/Sources/Core/NavigatorModels.swift`
- `VibeCat/Sources/Core/AudioMessageParser.swift`
- `VibeCat/Sources/VibeCat/AccessibilityNavigator.swift`

## Acceptance Criteria

1. Model can issue `navigate_click` for semantic button presses.
2. Coordinate clicks preserve `screenBasisId` and are rejected safely on stale screen state.
3. Scroll works on Chrome and code editor surfaces in manual acceptance tests.

## Risks

- coordinate clicks are dangerous without parser and stale-screen protections
- a combined multi-mode tool schema can confuse the model

## Sources

- Current repo runtime in `AccessibilityNavigator.swift`
