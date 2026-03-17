# P0-5: Safety Decision Handling

## Status

Implement as hardening of the existing navigator safety system.

## Current Repo State

This feature already exists in partial form.

### Backend

- `backend/realtime-gateway/internal/ws/navigator.go`
  - `planRiskReason(...)`
  - `stepsRequireRiskConfirmation(...)`
  - `buildRiskQuestion(...)`
- `backend/realtime-gateway/internal/ws/handler.go`
  - sends `navigator.riskyActionBlocked`
  - already supports clarification / confirmation message flow

### Client

- `VibeCat/Sources/Core/AudioMessageParser.swift`
  - parses `navigatorRiskyActionBlocked`
- `VibeCat/Sources/VibeCat/GatewayClient.swift`
  - sends navigator risk confirmation response
- `VibeCat/Sources/VibeCat/AppDelegate.swift`
  - shows current status/chat reaction for risky actions

## Problem To Solve

The current logic is split across planner heuristics and UI messaging. It needs:

- one shared classifier
- richer risk categories
- consistent handling for Live FC paths and navigator planner paths
- better confirmation UX

## Implementation Decision

### 1. Introduce a shared safety package

Add `backend/realtime-gateway/internal/safety/classifier.go` with:

- `RiskLevel`: `low`, `medium`, `high`
- `Assessment`
  - level
  - reason
  - requires confirmation
  - user-facing question
  - machine-readable category

### 2. Run the classifier in two places

- navigator plan building
- direct Live function-call handlers

This prevents drift between:

- planned navigator steps
- model-issued FC tools

### 3. Expand patterns beyond raw text match

Include:

- destructive shell patterns
- secret/token/password handling
- submit/send/publish/deploy verbs
- non-HTTPS or suspicious URLs
- system-level changes
- multi-step actions where the last step commits or submits

### 4. Standardize tool responses

When blocked:

- `status=blocked`
- `reason=<classifier reason>`
- `risk_level=<level>`

When denied:

- `status=user_denied`

When timeout:

- `status=confirmation_timeout`

## Concrete File Changes

### Backend

- `backend/realtime-gateway/internal/safety/classifier.go`
- `backend/realtime-gateway/internal/ws/navigator.go`
  - replace ad hoc keyword logic with shared classifier
- `backend/realtime-gateway/internal/ws/handler.go`
  - run shared classifier before executing risky FC handlers

### Client

- `VibeCat/Sources/VibeCat/AppDelegate.swift`
  - show richer approval UI
- `VibeCat/Sources/VibeCat/CompanionChatPanel.swift`
  - add explicit allow / block buttons

## UX Requirement

The confirmation prompt must show:

- what action will run
- which app or target it affects
- why it is considered risky
- clear approve / deny actions

Do not hide this inside a generic speech bubble only.

## Acceptance Criteria

1. Risk classification is shared between planner and FC runtime.
2. Risk confirmation UI shows explicit approve / deny choices.
3. Denied and timed-out actions return deterministic tool responses.
4. Existing safe flows remain zero-confirmation.

## Risks

- Over-classification can annoy users
- Under-classification weakens product trust
- Confirmation deadlocks if timeout handling is missing

## Sources

- [Gemini computer use guide](https://ai.google.dev/gemini-api/docs/computer-use)
