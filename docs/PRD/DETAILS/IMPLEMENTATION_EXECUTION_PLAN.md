# Implementation Execution Plan

This document defines a start-now implementation order for VibeCat.

## Target Repository Skeleton

```text
vibeCat/
  Assets/
  Sources/
    Core/
    VibeCat/
  Tests/
    VibeCatTests/
  docs/
    PRD/
    reference/
```

## Build Sequence

1. Bootstrap package/workspace and CI baseline.
2. Implement `Sources/Core` pure modules and their tests.
3. Implement settings/config/state foundation in `Sources/VibeCat`.
4. Implement capture, image pipeline, and gesture input.
5. Implement realtime transport and audio playback path.
6. Implement analysis agents (vision, mediation, scheduler, engagement).
7. Implement orchestrator to connect capture, agents, transport, and UI.
8. Implement overlay and chat UI integration.
9. Wire app startup and service lifecycle.
10. Finalize operational docs and deployment proof artifacts.

## Module Inventory (Target)

### Core module (`Sources/Core`)

- `AudioMessageParser.swift`
- `ChatMessage.swift`
- `ImageDiffer.swift`
- `ImageEncoder.swift`
- `ImageProcessor.swift`
- `KeychainHelper.swift`
- `PCMConverter.swift`
- `PromptBuilder.swift`
- `SettingsTypes.swift`
- `CharacterPresetConfig.swift`

### App module (`Sources/VibeCat`)

- App/bootstrap: `main.swift`, `AppDelegate.swift`, `Config.swift`
- UI: `CatViewModel.swift`, `CatView.swift`, `SpriteAnimator.swift`, `ChatBubbleView.swift`
- Capture and analysis: `ScreenCaptureService.swift`, `VisionAgent.swift`
- Decision agents: `Mediator.swift`, `EngagementAgent.swift`, `AdaptiveScheduler.swift`
- Orchestration: `ScreenAnalyzer.swift`
- Audio and transport: `GeminiLiveClient.swift`, `AudioPlayer.swift`, `CatVoice.swift`, `TTSClient.swift`, `BackgroundMusicPlayer.swift`
- Voice/chat: `GeminiChatPanel.swift`, `GeminiChatView.swift`, `GlobalHotkeyManager.swift`, `SpeechRecognizer.swift`
- Runtime support: `StatusBarController.swift`, `Settings.swift`, `Prompts.swift`, `VibeCatState.swift`, `ErrorReporter.swift`, `CircleGestureDetector.swift`

### Tests (`Tests/VibeCatTests`)

- Start with full Core coverage, then add app-layer tests in phase order from `TDD_VERIFICATION_PLAN.md`.

## Dependency Rules

- `Core` must not depend on app-layer UI modules.
- transport layer must be testable with mock server interfaces.
- orchestration depends on typed contracts, not concrete UI views.
- UI observes typed app state and does not own policy logic.

## Done Definition

VibeCat is implementation-ready when:

1. Source skeleton exists and compiles.
2. Required assets are present under `Assets/` and documented.
3. TDD plan phases 1-9 are tracked with passing tests.
4. Deployment checklist artifacts exist and are linked in docs.

## Immediate Next Execution Steps

1. Create source skeleton and placeholder modules.
2. Add phase-1 failing tests and implement to green.
3. Continue strictly phase by phase per `TDD_VERIFICATION_PLAN.md`.
