# VibeCat Package Guide

## OVERVIEW

`VibeCat/` is a Swift 6 package with a UI-free core target, a macOS executable target, and package tests.

## STRUCTURE

```text
VibeCat/
|-- Package.swift
|-- Sources/Core/        # shared models, localization, parsers, utilities
|-- Sources/VibeCat/     # AppKit app, capture, transport, AX executor
`-- Tests/VibeCatTests/  # package tests
```

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Package boundaries | `VibeCat/Package.swift` | `VibeCatCore`, `VibeCat`, `VibeCatTests` |
| App entrypoint | `VibeCat/Sources/VibeCat/VibeCat.swift` | `@main`, single-instance lock, hidden Edit menu |
| App runtime wiring | `VibeCat/Sources/VibeCat/AppDelegate.swift` | capture, gateway, navigator, UI shell |
| Local action execution | `VibeCat/Sources/VibeCat/AccessibilityNavigator.swift` | AX-first execution + verification |
| Step worker | `VibeCat/Sources/VibeCat/NavigatorActionWorker.swift` | step execution loop |
| Gateway transport | `VibeCat/Sources/VibeCat/GatewayClient.swift` | `/ws/live` client and navigator messages |
| Shared navigator models | `VibeCat/Sources/Core/NavigatorModels.swift` | cross-boundary payloads |
| Voice routing heuristics | `VibeCat/Sources/Core/NavigatorVoiceCommandDetector.swift` | voice-to-navigator interception |
| Navigator overlay UI | `VibeCat/Sources/VibeCat/NavigatorOverlayPanel.swift` | floating HUD: action label, grounding badge, progress, result |
| Navigator models | `VibeCat/Sources/Core/NavigatorModels.swift` | GroundingSource, NavigatorActionType, ExecutionPhase enums |
| Localization | `VibeCat/Sources/Core/Localization.swift` | navigator action labels, step progress, status strings |

## CONVENTIONS

- Keep `Sources/Core/` free of AppKit/UI dependencies.
- Put UI, AX, capture, playback, and app wiring under `Sources/VibeCat/`.
- Keep model/API payload structs in Core when both client and tests need them.
- Navigator mode is safety-first: verify targets, prefer guided mode over blind actions.
- Navigator overlay shows grounding source (AX/CDP/Vision/Keyboard) for every action.
- All navigator status strings go through Localization.swift for i18n readiness.

## ANTI-PATTERNS

- importing app-layer UI types into `Sources/Core/`
- storing Gemini credentials or model calls in the client
- bypassing `GatewayClient` for gateway protocol changes
- adding unverified blind click or typing paths to `AccessibilityNavigator`
- hardcoding UI strings outside Localization.swift

## COMMANDS

```bash
cd VibeCat && swift build
cd VibeCat && swift test
make build
make test
```
