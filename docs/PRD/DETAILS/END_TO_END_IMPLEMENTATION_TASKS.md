# End-to-End Implementation Tasks

This document defines a complete implementation task list for a new VibeCat development team.
The scope is documentation only (no code in this document), and each task includes concrete completion checks.

## Usage Rule

- Execute tasks in ID order unless a task explicitly states parallel execution is allowed.
- A task is complete only if its verification checks pass.
- Record task completion evidence in implementation notes and test logs.

## Phase 0 - Project Bootstrap

### T-001 Create workspace and target layout
- Goal: establish the package/workspace shape used by all downstream tasks.
- Required inputs: `docs/PRD/DETAILS/IMPLEMENTATION_EXECUTION_PLAN.md`.
- Implementation steps: create `Sources/Core`, `Sources/VibeCat`, `Tests/VibeCatTests`, `Assets`, `docs`.
- Verification: `swift build` succeeds with empty placeholders and expected target names.
- Depends on: none.

### T-002 Define app metadata and runtime permissions
- Goal: configure app identity and required runtime permission descriptions.
- Required inputs: `docs/reference/gemini/live-api/01_get_started_live_api.md`.
- Implementation steps: define bundle identifier, app name, screen recording and microphone permission descriptions.
- Verification: metadata file validates and app launches without permission-key crash.
- Depends on: T-001.

### T-003 Configure runtime configuration model
- Goal: define API key loading priority and validation boundaries.
- Required inputs: `docs/PRD/DETAILS/IMPLEMENTATION_REQUIREMENTS.md`.
- Implementation steps: document keychain-first lookup, environment fallback, and remote API-key verification flow.
- Verification: configuration tests cover empty key, malformed key, valid key, timeout, and server errors.
- Depends on: T-001.

### T-004 Establish settings persistence contract
- Goal: lock all persisted settings keys, defaults, and reset behavior.
- Required inputs: `docs/PRD/DETAILS/IMPLEMENTATION_STATUS_MATRIX.md`.
- Implementation steps: define settings domains for language, voice, capture, models, interaction, and music.
- Verification: round-trip persistence checks for each settings category.
- Depends on: T-001.

### T-005 Configure CI baseline for build and tests
- Goal: ensure build and test are mandatory before merge.
- Required inputs: `docs/PRD/DETAILS/TDD_VERIFICATION_PLAN.md`.
- Implementation steps: define CI jobs for build, test, and static checks.
- Verification: CI reports success on clean branch and fails on intentional test failure.
- Depends on: T-001, T-003, T-004.

## Phase 1 - Core Library

### T-010 Implement core message and settings domain models
- Goal: provide stable shared types used by app and tests.
- Required inputs: `docs/reference/gemini/genai-sdk/01_sdk_overview.md`.
- Implementation steps: define chat message model, settings enums, and strict value constraints.
- Verification: model construction and serialization tests pass.
- Depends on: T-004.

### T-011 Implement prompt composition primitives
- Goal: centralize prompt composition for live, fallback, and engagement flows.
- Required inputs: `docs/reference/gemini/live-api/02_live_capabilities_guide.md`.
- Implementation steps: define deterministic prompt composition with language-aware output constraints.
- Verification: prompt tests confirm required directives and language routing.
- Depends on: T-010.

### T-012 Implement image processing primitives
- Goal: provide resize and JPEG conversion functions for capture pipeline.
- Required inputs: `docs/PRD/DETAILS/IMPLEMENTATION_STATUS_MATRIX.md`.
- Implementation steps: define resize-if-needed and JPEG conversion paths with quality bounds.
- Verification: fixture tests validate dimensions, quality clamping, and output non-emptiness.
- Depends on: T-010.

### T-013 Implement image encoding and realtime payload helpers
- Goal: standardize text/image/audio payload construction for transport clients.
- Required inputs: `docs/reference/gemini/live-api/06_live_api_reference.md`.
- Implementation steps: define text-only, text+image, realtime image, and realtime audio payload builders.
- Verification: JSON structure tests match expected schema paths.
- Depends on: T-012.

### T-014 Implement visual change detection
- Goal: skip low-value capture uploads when screen content is unchanged.
- Required inputs: `docs/PRD/DETAILS/TDD_VERIFICATION_PLAN.md`.
- Implementation steps: define thumbnail generation and threshold-based pixel-difference logic.
- Verification: changed/unchanged fixture cases pass under multiple thresholds.
- Depends on: T-012.

### T-015 Implement audio message parsing
- Goal: decode server audio and transcription events from live stream messages.
- Required inputs: `docs/reference/gemini/live-api/06_live_api_reference.md`.
- Implementation steps: parse audio chunks, transcription updates, interruption, and turn-complete events.
- Verification: parser tests pass for valid, partial, and malformed server payloads.
- Depends on: T-013.

### T-016 Implement PCM conversion primitives
- Goal: convert PCM payloads to playback-friendly sample format.
- Required inputs: `docs/PRD/DETAILS/IMPLEMENTATION_STATUS_MATRIX.md`.
- Implementation steps: define Int16 to Float32 conversion with deterministic normalization.
- Verification: edge-value conversion tests pass (min, max, zero, odd byte length).
- Depends on: T-010.

### T-017 Implement secure key storage wrapper
- Goal: isolate keychain storage/retrieval for runtime configuration.
- Required inputs: `docs/PRD/DETAILS/IMPLEMENTATION_REQUIREMENTS.md`.
- Implementation steps: define save/load/delete operations with stable service/account names.
- Verification: local keychain tests pass in non-CI environment; CI skip policy is documented.
- Depends on: T-003.

### T-018 Complete core test suite gate
- Goal: enforce full core-module test coverage before app-layer work.
- Required inputs: `docs/PRD/DETAILS/TDD_VERIFICATION_PLAN.md`.
- Implementation steps: add and run all planned core tests.
- Verification: all core suites pass with zero failures.
- Depends on: T-011 to T-017.

## Phase 2 - Menu and Settings UX

### T-020 Build status-bar controller skeleton
- Goal: define the menu bar entry point and dynamic menu lifecycle.
- Required inputs: `docs/PRD/DETAILS/MENU_AND_RUNTIME_OPERATIONS_SPEC.md`.
- Implementation steps: create root menu, status row, and callback registration contract.
- Verification: menu appears and root state updates are reflected immediately.
- Depends on: T-004.

### T-021 Implement tray icon animation pipeline
- Goal: animate menu icon using emotion-state frame sets.
- Required inputs: `docs/PRD/DETAILS/ASSET_MIGRATION_PLAN.md`.
- Implementation steps: load tray icon frames, start frame timer, update icon by emotion.
- Verification: each emotion plays frame loop and falls back gracefully on missing frame.
- Depends on: T-020.

### T-022 Implement language, voice, and chattiness submenus
- Goal: expose core interaction behavior controls in menu.
- Required inputs: `docs/PRD/DETAILS/MENU_AND_RUNTIME_OPERATIONS_SPEC.md`.
- Implementation steps: build checkable submenus and bind each selection to persisted settings.
- Verification: selected option remains checked after app restart.
- Depends on: T-020.

### T-023 Implement model and reasoning controls
- Goal: allow runtime switching of vision/live/tts/thinking/media settings.
- Required inputs: `docs/reference/gemini/models/01_models_overview.md`.
- Implementation steps: add nested model menus and reconnect triggers for live model updates.
- Verification: changed model is persisted and reflected in next live setup payload.
- Depends on: T-020.

### T-024 Implement capture and appearance controls
- Goal: expose capture interval/sensitivity/quality and visual presentation controls.
- Required inputs: `docs/PRD/DETAILS/MENU_AND_RUNTIME_OPERATIONS_SPEC.md`.
- Implementation steps: connect capture-related items to capture service config and view model behavior.
- Verification: runtime changes apply without app relaunch.
- Depends on: T-020.

### T-025 Implement advanced and background-music controls
- Goal: expose feature toggles and background audio controls.
- Required inputs: `docs/PRD/DETAILS/ASSET_MIGRATION_PLAN.md`.
- Implementation steps: implement Google Search, proactive audio, affective dialog, launch-at-login, track, and volume items.
- Verification: toggles persist and side effects trigger expected runtime behavior.
- Depends on: T-020.

### T-026 Implement API key onboarding and validation UX
- Goal: provide first-launch key setup and validation flow.
- Required inputs: `docs/reference/gemini/genai-sdk/01_sdk_overview.md`.
- Implementation steps: implement onboarding prompt, key entry window, format checks, online validation, and save-anyway branches.
- Verification: valid key path starts analysis flow; invalid path provides clear feedback.
- Depends on: T-003, T-020.

### T-027 Implement reconnect, pause, mute, and quit actions
- Goal: finalize operational controls required for daily use.
- Required inputs: `docs/PRD/DETAILS/MENU_AND_RUNTIME_OPERATIONS_SPEC.md`.
- Implementation steps: implement action handlers and tie to orchestrator/live/audio components.
- Verification: pause stops loops, mute clears speech, reconnect restarts live client, quit exits cleanly.
- Depends on: T-020.

## Phase 3 - Capture and Gesture Input

### T-030 Implement capture around cursor and window-under-cursor
- Goal: collect context image data for analysis with panel exclusion.
- Required inputs: `docs/reference/gemini/live-api/01_get_started_live_api.md`.
- Implementation steps: implement region/window capture with panel exclusion and JPEG output.
- Verification: captures succeed on active display and panel is not captured.
- Depends on: T-012, T-014.

### T-031 Implement changed-only capture path
- Goal: reduce unnecessary analysis traffic.
- Required inputs: `docs/PRD/DETAILS/IMPLEMENTATION_REQUIREMENTS.md`.
- Implementation steps: compare thumbnails and emit unchanged/unavailable/captured outcomes.
- Verification: unchanged screens do not trigger downstream analysis.
- Depends on: T-030.

### T-032 Implement active-window full capture path
- Goal: support high-significance deep context capture.
- Required inputs: `docs/PRD/DETAILS/IMPLEMENTATION_STATUS_MATRIX.md`.
- Implementation steps: capture frontmost app window with stricter max dimension settings.
- Verification: high-significance flow attaches full-window image payload.
- Depends on: T-030.

### T-033 Implement circle-gesture trigger
- Goal: allow manual force-analysis trigger.
- Required inputs: `docs/PRD/DETAILS/MENU_AND_RUNTIME_OPERATIONS_SPEC.md`.
- Implementation steps: implement continuous mouse tracking and three-circle detection in time window.
- Verification: gesture invokes force-capture callback once per successful detection.
- Depends on: T-030.

## Phase 4 - Live Transport and Audio

### T-040 Implement live websocket lifecycle
- Goal: establish stable bidirectional live session management.
- Required inputs: `docs/reference/gemini/live-api/06_live_api_reference.md`.
- Implementation steps: connect, send setup payload, receive loop, disconnect, and manual reconnect controls.
- Verification: setup-complete event received and connection state is accurate.
- Depends on: T-003, T-013.

### T-041 Implement setup payload policy toggles
- Goal: ensure setup payload reflects language, voice, tools, proactivity, and session resumption.
- Required inputs: `docs/reference/gemini/live-api/02_live_capabilities_guide.md`.
- Implementation steps: map settings to setup payload fields and optional sections.
- Verification: payload snapshots match selected settings combinations.
- Depends on: T-040.

### T-042 Implement server message handling
- Goal: route live audio, transcription, interruption, and turn-complete events.
- Required inputs: `docs/reference/gemini/live-api/04_live_session_management.md`.
- Implementation steps: parse server content and trigger callback handlers.
- Verification: callback contract tests pass for each event type.
- Depends on: T-015, T-040.

### T-043 Implement heartbeat, protocol ping, and zombie detection
- Goal: protect against stale sockets and transient network loss.
- Required inputs: `docs/reference/adk/observability/01_observability_overview.md`.
- Implementation steps: implement heartbeat timer, ping timer, pong tracking, and forced reconnect when stale.
- Verification: simulated stale-connection case triggers reconnect path.
- Depends on: T-040.

### T-044 Implement audio playback engine
- Goal: play streaming PCM responses with stable state tracking.
- Required inputs: `docs/PRD/DETAILS/IMPLEMENTATION_STATUS_MATRIX.md`.
- Implementation steps: enqueue decoded buffers, start playback lazily, stop/clear states.
- Verification: `isPlaying` lifecycle matches scheduled buffer count transitions.
- Depends on: T-016.

### T-045 Implement speech wrapper and tts fallback client
- Goal: provide unified speech output path for live and fallback modes.
- Required inputs: `docs/reference/gemini/live-api/03_live_tool_use.md`.
- Implementation steps: speech wrapper delegates playback; fallback client synthesizes and returns PCM data.
- Verification: fallback path speaks successfully when live socket is unavailable.
- Depends on: T-044.

### T-046 Implement background music subsystem
- Goal: provide focus music with fade-in/fade-out around speaking events.
- Required inputs: `docs/PRD/DETAILS/ASSET_MIGRATION_PLAN.md`.
- Implementation steps: preload selected track, loop playback, and fade transitions on speech start/end.
- Verification: speaking event triggers fade-out and speech end restores configured volume.
- Depends on: T-044.

## Phase 5 - Agent Layer

### T-050 Implement vision analysis agent
- Goal: produce structured analysis from screen captures.
- Required inputs: `docs/reference/gemini/live-api/01_get_started_live_api.md`.
- Implementation steps: send image+prompt, enforce response schema, parse typed result, fallback model on failure.
- Verification: schema-valid output is produced for fixture screenshots.
- Depends on: T-013, T-030.

### T-051 Implement mediator decision engine
- Goal: gate speech by significance, cooldown, duplication, and app-change context.
- Required inputs: `docs/PRD/DETAILS/IMPLEMENTATION_STATUS_MATRIX.md`.
- Implementation steps: implement deterministic decision tree and context packet generation.
- Verification: decision-table tests cover all branches with expected reasons.
- Depends on: T-050.

### T-052 Implement adaptive scheduler
- Goal: tune silence threshold and cooldown from interaction metrics.
- Required inputs: `docs/PRD/DETAILS/TDD_VERIFICATION_PLAN.md`.
- Implementation steps: track utterances/responses/interruptions and apply bounded step adjustments.
- Verification: simulated metric profiles produce bounded expected values.
- Depends on: T-051.

### T-053 Implement engagement agent
- Goal: trigger proactive prompts after silence thresholds while respecting active turn.
- Required inputs: `docs/PRD/DETAILS/IMPLEMENTATION_REQUIREMENTS.md`.
- Implementation steps: schedule silence checks, build proactive packet, and suppress during active turn.
- Verification: clock-driven test shows trigger only after threshold.
- Depends on: T-052.

## Phase 6 - Orchestrator and Overlay UI

### T-060 Implement cat view model tracking loop
- Goal: move cat smoothly across monitors with boundary-safe positioning.
- Required inputs: `docs/PRD/DETAILS/MENU_AND_RUNTIME_OPERATIONS_SPEC.md`.
- Implementation steps: track cursor, lerp movement, handle jump threshold, manage panel migration across screens.
- Verification: cat remains within visible bounds during monitor transitions.
- Depends on: T-024.

### T-061 Implement sprite animator and character switching
- Goal: render animated emotional states using copied sprite assets.
- Required inputs: `docs/PRD/DETAILS/ASSET_MIGRATION_PLAN.md`.
- Implementation steps: resolve sprite root, load per-state frames, run frame timer, support character switching.
- Verification: each state animates and missing frames degrade gracefully.
- Depends on: T-021.

### T-062 Implement bubble and emotion indicators
- Goal: show speech content and emotional cues during runtime.
- Required inputs: `docs/PRD/DETAILS/MENU_AND_RUNTIME_OPERATIONS_SPEC.md`.
- Implementation steps: render bubble text, visibility animation, and edge-safe placement behavior.
- Verification: bubble hides according to configured duration when inactive.
- Depends on: T-060, T-061.

### T-063 Implement screen analyzer orchestration loop
- Goal: connect capture, agents, transport, and speech outputs.
- Required inputs: `docs/PRD/DETAILS/MENU_AND_RUNTIME_OPERATIONS_SPEC.md`.
- Implementation steps: run periodic capture cycle, analyze, evaluate, and route speak/skip decisions.
- Verification: no concurrent-cycle overlap and no deadlock during sustained operation.
- Depends on: T-031, T-050, T-051.

### T-064 Implement high-significance full-window branch
- Goal: attach deeper context when significance exceeds threshold.
- Required inputs: `docs/PRD/DETAILS/IMPLEMENTATION_REQUIREMENTS.md`.
- Implementation steps: capture active window on threshold condition and include in send path.
- Verification: threshold scenarios include full-window payload; lower scores do not.
- Depends on: T-032, T-063.

### T-065 Implement live transcription handling for bubble updates
- Goal: map streaming transcription to user-visible sentence updates.
- Required inputs: `docs/reference/gemini/live-api/04_live_session_management.md`.
- Implementation steps: buffer partial text, handle sentence finished flag, hold and flush behavior.
- Verification: sentence-level display behavior is correct for finished and turn-complete events.
- Depends on: T-042, T-063.

### T-066 Implement REST fallback speech path
- Goal: maintain voice output when live socket is unavailable.
- Required inputs: `docs/reference/gemini/genai-sdk/01_sdk_overview.md`.
- Implementation steps: generate fallback text, map emotion tags, synthesize TTS, and play audio.
- Verification: simulated disconnected-live condition still produces voice response.
- Depends on: T-045, T-063.

## Phase 7 - Voice and Chat Interaction

### T-070 Implement global hotkey manager
- Goal: provide system-wide interaction trigger.
- Required inputs: `docs/PRD/DETAILS/MENU_AND_RUNTIME_OPERATIONS_SPEC.md`.
- Implementation steps: monitor global/local key events and trigger callback on configured hotkey.
- Verification: hotkey opens chat/voice interaction from foreground and background apps.
- Depends on: T-024.

### T-071 Implement speech recognizer wake-word path
- Goal: allow wake-word activation and query extraction.
- Required inputs: `docs/reference/gemini/live-api/01_get_started_live_api.md`.
- Implementation steps: request permissions, start recognition, detect wake words, extract trailing query.
- Verification: wake-word variants trigger callback and pass correct query text.
- Depends on: T-002, T-070.

### T-072 Implement chat panel container
- Goal: provide borderless panel near cat for text chat entry and message display.
- Required inputs: `docs/PRD/DETAILS/MENU_AND_RUNTIME_OPERATIONS_SPEC.md`.
- Implementation steps: panel show/dismiss lifecycle, screen-bound clamped positioning, state reset on dismiss.
- Verification: panel opens near cat and always stays on-screen.
- Depends on: T-060.

### T-073 Implement chat view and message list behavior
- Goal: support user and assistant message rendering with auto-scroll and submit flow.
- Required inputs: `docs/PRD/DETAILS/MENU_AND_RUNTIME_OPERATIONS_SPEC.md`.
- Implementation steps: render role-based message rows, input field submit behavior, and scroll-to-latest update.
- Verification: message list autoscrolls and submit clears input.
- Depends on: T-072.

### T-074 Integrate hotkey, speech recognizer, and chat panel in app lifecycle
- Goal: finalize interaction mode from menu to runtime execution.
- Required inputs: `docs/PRD/DETAILS/MENU_AND_RUNTIME_OPERATIONS_SPEC.md`.
- Implementation steps: wire callbacks for hotkey press, wake-word detection, panel show, and send path.
- Verification: menu-selected interaction mode controls runtime trigger behavior.
- Depends on: T-070, T-071, T-072, T-073.

## Phase 8 - Full App Integration

### T-080 Implement main entrypoint and duplicate-instance guard
- Goal: prevent duplicate process instances and configure accessory app mode.
- Required inputs: `docs/PRD/DETAILS/MENU_AND_RUNTIME_OPERATIONS_SPEC.md`.
- Implementation steps: duplicate detection by bundle identifier, accessory activation policy, base edit menu.
- Verification: second launch exits cleanly and first instance remains active.
- Depends on: T-001.

### T-081 Implement app delegate wiring sequence
- Goal: ensure component initialization order prevents nil dependency failures.
- Required inputs: `docs/PRD/DETAILS/MENU_AND_RUNTIME_OPERATIONS_SPEC.md`.
- Implementation steps: instantiate all major components, inject dependencies, and attach callback pathways.
- Verification: startup completes without runtime nil-access errors.
- Depends on: T-063, T-074.

### T-082 Configure floating panel behavior
- Goal: keep overlay visible across spaces while allowing click-through behavior.
- Required inputs: `docs/PRD/DETAILS/MENU_AND_RUNTIME_OPERATIONS_SPEC.md`.
- Implementation steps: configure panel level, transparency, collection behavior, and mouse-event policy.
- Verification: panel stays across desktop spaces and does not block normal app interaction.
- Depends on: T-081.

### T-083 Complete status-bar callback wiring
- Goal: connect menu actions to runtime changes safely.
- Required inputs: `docs/PRD/DETAILS/MENU_AND_RUNTIME_OPERATIONS_SPEC.md`.
- Implementation steps: wire pause/mute/reconnect/model/capture/music/settings-reset callbacks.
- Verification: each menu action produces expected runtime side effect.
- Depends on: T-027, T-081.

### T-084 Implement startup mode behavior (with and without API key)
- Goal: produce predictable first-launch and normal-launch behavior.
- Required inputs: `docs/PRD/DETAILS/MENU_AND_RUNTIME_OPERATIONS_SPEC.md`.
- Implementation steps: onboarding when key missing, direct analysis start when key exists and app is not paused.
- Verification: both startup branches execute correctly in isolated runs.
- Depends on: T-026, T-081.

## Phase 9 - Assets, Operations, and Handoff

### T-090 Validate copied assets and runtime path resolution
- Goal: guarantee local assets are sufficient and discoverable.
- Required inputs: `docs/PRD/DETAILS/ASSET_MIGRATION_PLAN.md`.
- Implementation steps: validate sprite/tray/music/voice-sample counts and runtime path resolution order.
- Verification: asset count checks pass and runtime loaders resolve existing files.
- Depends on: T-061, T-046.

### T-091 Run full operation scenarios from menu to speech output
- Goal: validate end-to-end behavior using only documented runtime paths.
- Required inputs: `docs/PRD/DETAILS/MENU_AND_RUNTIME_OPERATIONS_SPEC.md`.
- Implementation steps: execute first-launch onboarding, live connection, capture cycle, proactive trigger, fallback mode, pause/resume, mute/unmute, reconnect.
- Verification: all scenario checks pass and produce expected state transitions.
- Depends on: T-084, T-090.

### T-092 Complete deployment and operations evidence pack
- Goal: prepare deployment and observability artifacts.
- Required inputs: `docs/PRD/DETAILS/DEPLOYMENT_AND_OPERATIONS.md`.
- Implementation steps: produce deployment proof, health checks, logging/monitoring evidence, and secrets policy notes.
- Verification: all required evidence files exist and are cross-linked.
- Depends on: T-091.

### T-093 Final handoff gate
- Goal: confirm a new team can implement and operate VibeCat using docs and assets only.
- Required inputs: all PRD detail documents and `docs/reference`.
- Implementation steps: perform doc-only dry-run walkthrough and resolve any ambiguous instruction.
- Verification: no open blocker remains in task sequence, link validation is clean, and checklists are complete.
- Depends on: T-092.

## Phase 7 — Grand Prize Features

### T-094 Implement Decision Overlay HUD
- Goal: show real-time "why the agent spoke" information on screen for grounding evidence.
- Required inputs: `docs/PRD/DETAILS/BACKEND_ARCHITECTURE.md` (Agent Interaction Matrix).
- Implementation steps: create a translucent overlay panel (toggle with hotkey) showing: trigger source (silence/gesture/vision/mood), what VisionAgent detected (structured fields), Mediator decision (speak/skip + reason), MoodDetector state (mood + confidence), cooldown timer remaining.
- Verification: overlay shows correct trigger source and decision for each agent action. Toggle on/off works. Does not interfere with main UI.
- Depends on: T-066.

### T-095 Implement sprite screen pointing
- Goal: animate the cat sprite to move toward the location of a detected error on screen.
- Required inputs: `docs/PRD/DETAILS/BACKEND_ARCHITECTURE.md` (VisionAgent).
- Implementation steps: extend VisionAgent response to include error region coordinates (approximate x,y from screen capture analysis), client receives coordinates in analysisResult message, animate sprite movement toward error region, sprite returns to default position after speaking.
- Verification: error detected in bottom-right of screen → cat moves toward bottom-right. Returns to home position after 5 seconds.
- Depends on: T-066, T-040.

### T-096 Implement session farewell and sleep sprite
- Goal: when session ends, the cat says goodbye and shows a sleeping animation.
- Required inputs: asset inventory (need sleeping sprite).
- Implementation steps: on session end (app quit or explicit close), send farewell message ("오늘 고생했어, 내일 보자"), transition to sleeping sprite state (reuse idle with dimmed overlay or create new sleep sprite), play gentle closing animation, save session summary via MemoryAgent.
- Verification: app close triggers farewell voice + sleep animation. Summary is saved in Firestore.
- Depends on: T-066, backend T-148.

### T-097 Handle companion intelligence message types
- Goal: client supports new WebSocket message types from companion intelligence agents.
- Required inputs: `docs/PRD/DETAILS/CLIENT_BACKEND_PROTOCOL.md`.
- Implementation steps: handle `memoryContext` (display context bridge text in chat panel on session start), handle `moodUpdate` (adjust sprite expression, show supportive message in chat), handle `celebration` (transition to happy sprite, play celebration animation), handle `searchResult` (display sources in chat panel with clickable links), send `searchRequest` when user types search query, send `memoryFeedback` when user confirms/dismisses memory.
- Verification: each message type produces correct UI response. Memory context shows on session start. Celebration triggers happy sprite.
- Depends on: T-040, T-066.

### T-098 Implement graceful reconnection UX
- Goal: when connection drops, show reconnecting state with smooth recovery.
- Required inputs: `docs/PRD/DETAILS/BACKEND_ARCHITECTURE.md` (Keepalive Stack).
- Implementation steps: on WebSocket disconnect, show reconnecting indicator (sprite shows thinking state), attempt reconnect with 1s delay, on reconnect success send resumptionHandle for session continuity, show "돌아왔어!" message on successful reconnect, if TTS fallback active show fallback indicator.
- Verification: network disconnect → reconnecting indicator within 1s. Reconnect → session resumes seamlessly. 3 consecutive failures → show error state.
- Depends on: T-040, backend T-173.

### T-099 Add privacy controls UI
- Goal: always-visible capture indicator and one-click pause for user trust.
- Implementation steps: add always-visible indicator showing when screen capture is active (small dot in menu bar), add one-click pause button to stop all screen capture, add "analyze now" manual mode (capture only on user request), show "no screenshots stored" statement in settings panel.
- Verification: capture indicator reflects actual capture state. Pause stops all captures. Manual mode captures only on click.
- Depends on: T-023, T-066.
