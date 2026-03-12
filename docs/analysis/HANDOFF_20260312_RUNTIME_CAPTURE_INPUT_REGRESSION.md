# Runtime Capture / Input Regression Handoff

Date: 2026-03-12

## Read this after

- `docs/analysis/HANDOFF_20260312_NAVIGATOR_STABILIZATION.md`
- `docs/analysis/MACOS_TEXT_ENTRY_AUTOMATION_20260311.md`

This handoff covers the work done after the stabilization handoff: execution-control hardening, live speech/input fixes, capture-context fixes, and the still-open runtime regressions around multi-monitor window tracking and current-window panel behavior.

## Branch / local state

- Branch: `codex/navigator-stabilization-20260312`
- Worktree is dirty with substantial local changes across Swift client, gateway, orchestrator, docs, and tests.
- No commit was created in this session.

## Latest deployed backend revisions

- `realtime-gateway-00054-rxq`
- `adk-orchestrator-00043-jv9`

The backend was redeployed multiple times during this session. The revisions above are the latest confirmed ready revisions at handoff time.

## Latest local runtime state

- Latest log path: `/tmp/vibecat-run.log`
- Multiple local restarts happened during this session.
- The current process at one checkpoint was `90361 .build/arm64-apple-macosx/debug/VibeCat`.
- Later restarts also produced `85778`, `89615`, and `90361` / `90361`-successor style runs while iterating on panel/cat placement.
- Do not trust old visual behavior reports without checking the current PID + `/tmp/vibecat-run.log` header first.

## What changed in this session

## 1. Execution-control contract and client/runtime hardening

Swift client:

- added `SurfaceKind`, `ProofLevel`, `ExecutionFailureReason`, `ExecutionPhase`, `VerifyContract`
- expanded `NavigatorStep` and `NavigatorContextPayload`
- added structured execution results and richer refresh payloads
- added display metadata and screenshot provenance
- added target highlight overlay
- replaced several fixed sleeps with bounded wait/prove helpers
- added `NavigatorSurfaceProfile` and surface-aware preparation rules

Files:

- `VibeCat/Sources/Core/NavigatorModels.swift`
- `VibeCat/Sources/VibeCat/AccessibilityNavigator.swift`
- `VibeCat/Sources/VibeCat/NavigatorActionWorker.swift`
- `VibeCat/Sources/VibeCat/GatewayClient.swift`
- `VibeCat/Sources/Core/NavigatorSurfaceProfile.swift`
- `VibeCat/Sources/VibeCat/TargetHighlightOverlay.swift`

Gateway / orchestrator:

- planner emits `surface`, `macroID`, `narration`, `verifyContract`, `fallback*`, `timeoutMs`, `proofLevel`
- attempt logging/state/replay chain added

Files:

- `backend/realtime-gateway/internal/ws/navigator.go`
- `backend/realtime-gateway/internal/ws/navigator_confidence.go`
- `backend/realtime-gateway/internal/ws/handler.go`
- `backend/realtime-gateway/internal/ws/action_state_store.go`
- `backend/realtime-gateway/internal/ws/navigator_background.go`
- `backend/realtime-gateway/internal/adk/client.go`
- `backend/adk-orchestrator/internal/models/models.go`
- `backend/adk-orchestrator/internal/navigator/processor.go`
- `backend/adk-orchestrator/internal/store/models.go`

## 2. Speech / voice routing / input surfacing

- restored visible user speech bubble updates
- removed 300ms device-change delay for mic recovery
- added user-input transcription assembly path so raw fragments/noise are not shown directly
- added noise stripping for `<noise>`, `[noise]`, `<unk>`
- improved Korean text-entry command routing (`입력해죠`, `넣어줘`, `붙여줘`, etc.)
- added voice reroute queue so `sendBargeIn()` and `navigator.command` are no longer sent in the same unstable instant

Files:

- `VibeCat/Sources/VibeCat/AppDelegate.swift`
- `VibeCat/Sources/VibeCat/AssistantTranscriptionAssembler.swift`
- `VibeCat/Sources/Core/NavigatorVoiceCommandDetector.swift`

Tests:

- `VibeCat/Tests/VibeCatTests/AssistantTranscriptionAssemblerTests.swift`
- `VibeCat/Tests/VibeCatTests/NavigatorVoiceCommandDetectorTests.swift`

## 3. Capture/context work

- command submissions now prefer fresh capture, not only stale cached screenshot reuse
- `display_fallback` is rejected for navigator command context
- proactive analysis now skips `display_fallback` to avoid poisoning the live session with `loginwindow` / full-display junk
- fast current-window panel probe introduced and later refactored repeatedly

Files:

- `VibeCat/Sources/VibeCat/ScreenAnalyzer.swift`
- `VibeCat/Sources/VibeCat/ScreenCaptureService.swift`
- `VibeCat/Sources/VibeCat/CatPanel.swift`
- `VibeCat/Sources/VibeCat/CatViewModel.swift`

## 4. Docs / strategy / process

- hierarchical `AGENTS.md` files added
- execution control plane / winning strategy / judge proof docs added
- account-bound Gemini docs guard skill added:
  - `~/.claude/skills/gemini-official-docs-guard/SKILL.md`

## What is proven

- `cd VibeCat && swift build && swift test` passed repeatedly
- final Swift state reached `85 tests, 1 skipped, 0 failures`
- `cd backend/realtime-gateway && go build ./... && go test ./... && go vet ./...` passed repeatedly
- `cd backend/adk-orchestrator && go build ./... && go test ./... && go vet ./...` passed repeatedly
- synthesized voice-input round-trip test passed against deployed gateway:
  - `tests/e2e/voice_input_transcription_test.go`
- real Terminal integration smoke passed at one point:
  - `VIBECAT_REAL_TERMINAL_INPUT=1 swift test --filter TerminalInputIntegrationTests`

## Current confirmed open problems

### 1. Current-window panel only behaves correctly on the main display and one secondary display

User-observed symptom:

- the lower panel showing the currently recognized window updates on the main monitor and one secondary monitor, but not reliably on monitors 3 and 4.

What we found:

- the capture system itself can see non-main displays in logs (`display=3`, `display=5` appeared in earlier runtime evidence)
- the worse bug is the panel/cat coordinate model: it was partly switched to union-of-displays coordinates and then partially reverted
- multiple regressions were introduced while trying to fix this:
  - union-sized `CatPanel` made the cat disappear or move to the wrong area
  - `activeScreenFrame` semantics were accidentally changed to mean union bounds
  - later this was reverted so visibility recovered, but the panel still is not structurally unified across all displays

Key files:

- `VibeCat/Sources/VibeCat/CatViewModel.swift`
- `VibeCat/Sources/VibeCat/CatPanel.swift`
- `VibeCat/Sources/VibeCat/AppDelegate.swift`

Current status:

- latest stable runtime uses current-screen-sized `CatPanel` again
- that restored cat visibility, but likely reintroduced the limitation that the panel only truly belongs to a single active screen at a time
- this is not fully solved yet

### 2. Current-window title panel flickers between `window` and `window + panel-name`

User-observed symptom:

- the lower panel flickers between just the underlying window title and a composite title that includes the panel / app itself.

Confirmed root cause chain:

1. fast probe originally used screenshot/capture path too directly
2. then fast probe started using window-under-cursor checks
3. later AX-first probe was added
4. `VibeCat` self-hit in the probe path caused the panel/app to sometimes be recognized as the current target

What was fixed:

- self-hit filter was added by excluding `Bundle.main.bundleIdentifier` from probe results
- stale-title persistence on `displayFallback` was also fixed by clearing the panel when fallback happens

What is still not fully trustworthy:

- because the panel is itself a floating object and the cat/panel geometry still changes with monitor layout, the full flicker problem may not be completely dead until panel/cat multi-monitor ownership is made unambiguous

Key files:

- `VibeCat/Sources/VibeCat/ScreenCaptureService.swift`
- `VibeCat/Sources/VibeCat/ScreenAnalyzer.swift`
- `VibeCat/Sources/VibeCat/CatPanel.swift`

### 3. Speech/voice recognition quality is still weak in real use

User-observed symptom:

- speech recognition quality is low
- some user commands still drift into unrelated live/chat replies

What is now true:

- Live API audio input / `inputTranscription` works end-to-end
- server-side Live input transcription is active
- automatic activity detection / interruption path is configured
- synthetic voice injection test passed

What remains weak:

- real-world Korean speech variants and noisy partials still cause routing misses or misclassification sometimes
- voice reroute is better, but not fully robust under natural real usage

Current model/config facts:

- current Live audio model: `gemini-2.5-flash-native-audio-preview-12-2025`
- Live session config uses `InputAudioTranscription` and automatic activity detection style handling
- local `SpeechRecognizer` is not the STT model; it captures mic audio and forwards PCM

Relevant files:

- `VibeCat/Sources/VibeCat/SpeechRecognizer.swift`
- `VibeCat/Sources/VibeCat/AppDelegate.swift`
- `backend/realtime-gateway/internal/live/session.go`
- `VibeCat/Sources/Core/GeminiModels.swift`

## Most important runtime evidence

### A. Current process / latest stable log checkpoints

- one older runtime showing the union-frame regression:
  - `/tmp/vibecat-run.log:20`
  - `CatPanel shown. frame={{-1890, -339}, {7906, 3360}}`
- later runtime after visibility rollback:
  - `/tmp/vibecat-run.log:11`
  - `CatPanel shown. frame={{3008, 1329}, {3008, 1692}}`

### B. Window-under-cursor logs

- example successful capture evidence:
  - `target=window_under_cursor app=Google Chrome window=vibeCat 프로그램 분석`
- example ignored fallback evidence:
  - `ignoring display fallback snapshot for proactive analysis app=Google Chrome display=5`
  - `ignoring display fallback snapshot for proactive analysis app=한컴오피스 한글 display=5`

### C. Voice route / input evidence

- synthetic voice-input test passed and returned:
  - `이 터미널의 A부터 Z까지 다시 입력해 줘.`

## Current best root-cause summary

The session uncovered three intertwined but separate issues:

1. **Capture targeting correctness**
   - improved substantially
   - no longer the only root cause
   - `display_fallback` poisoning has been reduced

2. **Panel/cat geometry ownership across multiple displays**
   - still the largest unresolved structural issue
   - the cat and its lower panel need a single, explicit monitor-ownership model that works across all displays without switching between single-screen and union semantics ad hoc

3. **Voice routing quality**
   - improved and now testable
   - but still not “done” for natural speech reliability

## Recommended next-session debug order

Do these in order. Do not jump around.

### Step 1: Freeze and observe panel ownership

- add temporary logs for:
  - cat global position
  - cat local position
  - panel frame
  - window-title badge text
  - visible screen containing cat
- verify on each monitor whether the panel is being attached to the same screen as the cat

Goal:

- prove whether the bug is screen ownership, clamp logic, or panel frame sizing

### Step 2: Choose one geometry model and stick to it

Recommended direction:

- keep the actual floating `CatPanel` bound to a single real screen at a time
- when the cat crosses into another screen, explicitly move the whole panel to that screen
- do not keep a union-sized panel if visibility and hit-testing regress
- use union/global coordinates only as an internal math tool, not as the live panel frame

### Step 3: Make the current-window badge a pure follower of the cat's owning screen

- panel badge layout should be clamped against the screen containing the cat
- panel title updates should be based on a probe point derived from the cat, not the raw mouse, but only after screen ownership is explicit

### Step 4: Re-run real runtime checks

- test on all displays:
  - main monitor
  - vertical monitor
  - right-side monitors
- confirm:
  - cat visible
  - lower panel visible
  - panel text updates to the current window on that monitor
  - no `window -> window+panel` flicker

### Step 5: Only after that, continue voice/input quality work

- do not mix geometry and speech fixes in the same patch set again

## Files most worth reading first next session

- `VibeCat/Sources/VibeCat/CatViewModel.swift`
- `VibeCat/Sources/VibeCat/CatPanel.swift`
- `VibeCat/Sources/VibeCat/ScreenCaptureService.swift`
- `VibeCat/Sources/VibeCat/ScreenAnalyzer.swift`
- `VibeCat/Sources/VibeCat/AppDelegate.swift`
- `VibeCat/Tests/VibeCatTests/CatViewModelGeometryTests.swift`
- `VibeCat/Tests/VibeCatTests/ScreenCaptureServiceTests.swift`
- `tests/e2e/voice_input_transcription_test.go`

## Official references already consulted

- Google Cloud architecture docs on orchestration / interactive learning / single-agent ADK on Cloud Run
- Gemini Live API docs for Live input transcription, interruption, session handling, best practices, and changelog checks
- local Apple SDK AX references already noted in `MACOS_TEXT_ENTRY_AUTOMATION_20260311.md`
- Apple/macOS guidance relevant to window-server coordinates and AX hit-testing was consulted to justify the AX-first under-cursor probe and the later direct-screen-space bounds comparison

## Final note for the next session

Do not assume the current panel bug is still a capture bug.

The latest evidence suggests:

- capture improved
- speech improved
- attempt logging improved
- the remaining highest-risk issue is **panel/cat multi-monitor ownership semantics**

Start there.
