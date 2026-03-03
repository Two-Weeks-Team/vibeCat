# Implementation Requirements

## Mandatory Stack

- GenAI SDK
- ADK
- Gemini Live API
- VAD (`automaticActivityDetection`)

## Runtime Architecture

- Client app: input/output UI, local capture, transport only
- Realtime Gateway (Cloud Run): Live API session and stream handling
- ADK Orchestrator (Cloud Run): agent graph, tool routing, response policy

## Live API and VAD Baseline

```json
{
  "realtimeInputConfig": {
    "automaticActivityDetection": {
      "disabled": false,
      "startOfSpeechSensitivity": "START_SENSITIVITY_LOW",
      "endOfSpeechSensitivity": "END_SENSITIVITY_LOW",
      "prefixPaddingMs": 20,
      "silenceDurationMs": 100
    }
  }
}
```

## Acceptance Criteria

- User speech start is detected under target latency
- Interruption event correctly stops current output turn
- New user turn is accepted without deadlock
- All model calls originate from backend services

## Execution References

- Implementation status by feature: `docs/PRD/DETAILS/IMPLEMENTATION_STATUS_MATRIX.md`
- Test-first sequence and quality gates: `docs/PRD/DETAILS/TDD_VERIFICATION_PLAN.md`
- Build order and dependency constraints: `docs/PRD/DETAILS/IMPLEMENTATION_EXECUTION_PLAN.md`
- End-to-end implementation tasks: `docs/PRD/DETAILS/END_TO_END_IMPLEMENTATION_TASKS.md`
- Menu and runtime operations behavior: `docs/PRD/DETAILS/MENU_AND_RUNTIME_OPERATIONS_SPEC.md`
- Asset copy and integrity checks: `docs/PRD/DETAILS/ASSET_MIGRATION_PLAN.md`
