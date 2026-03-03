# TDD Verification Plan

This plan defines Red-Green-Refactor order to implement VibeCat behavior with test-first execution.

## Phase Order

1. Core models and pure utilities
2. Image and audio processing primitives
3. Prompt and policy primitives
4. Mediation and adaptive timing logic
5. Realtime client contract and reconnect semantics
6. Vision analysis contract and schema validation
7. Orchestrator flow and interruption behavior
8. UI state and interaction layer
9. Deployment checks and operations proof

## Red-Green-Refactor Matrix

| Phase | Red (write failing tests) | Green (minimal code) | Refactor (stabilize) |
|---|---|---|---|
| 1 | model invariants, parser edge cases | implement structs/enums/parsers | normalize naming and shared helpers |
| 2 | fixed image/audio fixture tests | implement transforms and converters | reduce allocation and duplicate transforms |
| 3 | prompt policy assertions | implement prompt composition | extract reusable prompt blocks |
| 4 | decision-table tests for speak/skip | implement gating and scheduling | separate policy constants |
| 5 | mock websocket reconnect tests | implement session lifecycle and ping/pong | isolate transport concerns |
| 6 | invalid schema and retry tests | implement typed response decoding | consolidate schema validators |
| 7 | full pipeline interruption tests | implement orchestration hooks | split orchestration into focused coordinators |
| 8 | state transition tests for overlay/chat | implement state reducers and views | move side effects behind interfaces |
| 9 | deployment script and health checks | implement CI/CD and health endpoints | tighten observability and alert thresholds |

## Required Test Categories

- Unit: deterministic pure logic and transformation code
- Contract: external API request/response format and schema
- Integration: realtime transport, orchestration, persistence
- Smoke: executable startup, core interaction path, graceful shutdown

## Initial Test File Scaffold

Create these test files first and implement assertions in phase order:

- `Tests/VibeCatTests/AudioMessageParserTests.swift`
- `Tests/VibeCatTests/SettingsTypesTests.swift`
- `Tests/VibeCatTests/PromptBuilderTests.swift`
- `Tests/VibeCatTests/ImageProcessorTests.swift`
- `Tests/VibeCatTests/ImageDifferTests.swift`
- `Tests/VibeCatTests/PCMConverterTests.swift`
- `Tests/VibeCatTests/MediatorTests.swift`
- `Tests/VibeCatTests/AdaptiveSchedulerTests.swift`
- `Tests/VibeCatTests/GeminiLiveClientTests.swift`
- `Tests/VibeCatTests/ScreenAnalyzerTests.swift`

## Quality Gates Per Merge

1. All new tests for changed area are present and passing.
2. Typecheck/build passes.
3. No failing gate in `IMPLEMENTATION_STATUS_MATRIX.md`.
4. No skipped test without documented reason and follow-up task.
