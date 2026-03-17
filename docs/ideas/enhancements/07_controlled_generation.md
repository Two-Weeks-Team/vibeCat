# P1-7: Controlled Generation

## Status

Implement now for direct batch calls in the orchestrator.

## Source-Verified Facts

- Current structured-output guidance centers on `responseMimeType = "application/json"` plus schema.
- Go SDK v1.49.0 supports both:
  - `ResponseSchema *genai.Schema`
  - `ResponseJsonSchema any`
- ADK `llmagent.Config` has `OutputSchema`, but its own config docs warn that tool use should not be mixed with that path as a primary design.

## Implementation Decision

### Use JSON objects, not enum-only text mode, as the default contract

For classifier-style outputs, return:

```json
{"decision":"OBSERVE"}
```

not a bare enum string.

Reason:

- easier extensibility
- better backward compatibility
- easier logging and validation

### Prefer `ResponseJsonSchema` for direct `GenerateContent` calls

Apply first to:

- `backend/adk-orchestrator/internal/agents/vision/vision.go`
- `backend/adk-orchestrator/internal/agents/tooluse/tooluse.go`
- `backend/adk-orchestrator/internal/agents/memory/memory.go`

### Use ADK `OutputSchema` only for no-tool agents

Do not make tool-using `llmagent` paths depend on `OutputSchema` for v0.2.

## Concrete File Changes

### `vision.go`

- define a JSON schema matching `models.VisionAnalysis`
- request `application/json`
- reject malformed model output instead of silently accepting free-form text

### `tooluse.go`

- classifier returns:
  - `tool_kind`
  - `confidence`
  - `reason`
- validate against schema before routing

### `memory.go`

- `generateSummary()` should request schema:
  - `summary`
  - optional `unresolved_issues`

## Suggested Contracts

### Tool router

```json
{
  "tool_kind": "search",
  "confidence": 0.94,
  "reason": "current docs / external grounding required"
}
```

### Vision analysis

```json
{
  "significance": 8,
  "content": "The test run failed on auth middleware.",
  "emotion": "concerned",
  "shouldSpeak": true,
  "errorDetected": true,
  "repeatedError": false,
  "successDetected": false,
  "errorMessage": "..."
}
```

## Acceptance Criteria

1. Parse failures drop to zero on covered agents.
2. Schema-invalid outputs are logged and retried or downgraded deterministically.
3. Existing tests are updated to assert typed JSON payloads.

## Risks

- Slight latency increase from stricter constraints
- Overly strict schema can increase fallback frequency if not tuned

## Sources

- [Gemini structured output guide](https://ai.google.dev/gemini-api/docs/structured-output)
