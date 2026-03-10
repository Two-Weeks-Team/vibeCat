# VibeCat Pipeline Refactor Plan

## Goal
Fundamentally fix all pipeline coupling issues in VibeCat's Swift client and Go backend to achieve:
1. User voice barge-in always works (highest priority)
2. Ongoing speech maintained unless barge-in
3. Speech content matches bubble text exactly
4. Screen capture runs as independent 1Hz background pipeline

## User Priorities (in order)
1. Voice barge-in > 2. Speech maintenance > 3. Bubble sync > 4. Independent capture

---

## WAVE 1 (Parallel — mutually independent)

### Task 1A: Transcription-TTS Race Fix (Bubble Sync)
**File:** `VibeCat/Sources/VibeCat/AppDelegate.swift` (lines 260-288)
**Problem:** Late `.transcription` messages arriving after `.ttsStart` append to empty `pendingTranscription`, showing fragments like "!" instead of full text.
**Change:**
1. Add `ttsActive` guard in `.transcription` handler:
```swift
case .transcription(let text, let finished):
    // ... chatMode handling unchanged ...
    else if !text.isEmpty {
        guard !self.ttsActive else {
            if finished { self.pendingTranscription = "" }
            return
        }
        self.isTurnActive = true
        // ... existing pendingTranscription += displayText logic ...
    }
```
**Verification:** `make build && make test` (31 tests pass)
**Risk:** Low — guard addition only, no existing logic changed
**Addresses:** Priority 3 (bubble sync)

### Task 1B: Screen Capture Independence (Fast Path)
**File:** `VibeCat/Sources/VibeCat/ScreenAnalyzer.swift` (lines 127-173)
**Problem:** `isSpeechActive` check at line 145 completely suppresses capture during TTS + 5s cooldown. VibeCat is blind to screen changes while speaking.
**Change:**
1. Move `isSpeechActive` check to gate only Smart Path, not Fast Path:
```swift
private func runAnalysisCycle(forceSmartPath: Bool) async {
    guard isRunning, !isAnalyzing else {
        if isRunning { scheduleNextCapture() }
        return
    }
    guard gatewayClient.isConnected else {
        if isRunning { scheduleNextCapture() }
        return
    }
    isAnalyzing = true
    defer {
        isAnalyzing = false
        if isRunning { scheduleNextCapture() }
    }

    // REMOVED: speechActive check here — capture always proceeds

    let result = await captureService.captureAroundCursor()
    switch result {
    case .unchanged:
        NSLog("[CAPTURE] unchanged")
        return
    case .unavailable(let reason):
        NSLog("[CAPTURE] unavailable: %@", reason)
        return
    case .captured(let image):
        NSLog("[CAPTURE] captured: %dx%d", image.width, image.height)
        let now = Date()

        // Fast Path: ALWAYS send (speech state irrelevant)
        if now.timeIntervalSince(lastFastPathSend) >= fastPathCooldown {
            sendFastPath(image: image)
            lastFastPathSend = now
        }

        // Smart Path: suppress during speech (prevents new speech triggers)
        let speechActive = isSpeechActive
        if !speechActive {
            let smartPathReady = forceSmartPath || now.timeIntervalSince(lastSmartPathSend) >= smartPathCooldown
            if smartPathReady {
                await sendSmartPath(image: image, highSignificance: forceSmartPath)
                lastSmartPathSend = now
            }
        }
    }
}
```
2. Update `forceAnalysis()` similarly — Fast Path always, Smart Path gated:
```swift
func forceAnalysis() async {
    guard gatewayClient.isConnected else { return }
    let result = await captureService.forceCapture()
    if case .captured(let image) = result {
        sendFastPath(image: image)
        lastFastPathSend = Date()
        if !isSpeechActive {
            await sendSmartPath(image: image, highSignificance: true)
            lastSmartPathSend = Date()
        }
    }
}
```
**Verification:** `make build && make test`
**Risk:** Low — Fast Path sends video context to Live API only, does not trigger speech
**Addresses:** Priority 4 (independent capture)

### Task 1C: Gateway Video Guard Removal
**File:** `backend/realtime-gateway/internal/ws/handler.go` (lines 233-241)
**Problem:** Gateway drops JPEG video frames when `isModelSpeaking()` is true. This blocks Gemini from having real-time screen context during speech.
**Change:**
1. Remove `isModelSpeaking()` guard for video (JPEG) frames only:
```go
// BEFORE (line 233-241):
if isJPEG(data) {
    if sess := ls.getSession(); sess != nil {
        if !ls.isModelSpeaking() {
            if sendErr := sess.SendVideo(data); sendErr != nil { ... }
        }
    }
}

// AFTER:
if isJPEG(data) {
    if sess := ls.getSession(); sess != nil {
        if sendErr := sess.SendVideo(data); sendErr != nil { ... }
    }
}
```
2. Keep `isModelSpeaking()` guard for Smart Path (screenCapture JSON) at line 388-391 — defensive layer since client also suppresses.
**Verification:** `cd backend/realtime-gateway && go test ./...` (6 packages pass)
**Risk:** Low — Gemini Live API accepts video during model speech for context
**Addresses:** Priority 4 (independent capture)

---

## WAVE 2 (After Wave 1 — Task 2A then 2B sequentially)

### Task 2A: SpeechState Consolidation
**File:** `VibeCat/Sources/VibeCat/AppDelegate.swift`
**Problem:** 4 boolean state variables (isTurnActive, ttsActive, isTTSSpeaking, modelSpeaking) scattered across 3 files with timing mismatches.
**Change:**
1. Add `SpeechState` enum to AppDelegate:
```swift
private enum SpeechState {
    case idle
    case modelSpeaking
    case cooldown
}
private var speechState: SpeechState = .idle
```
2. Add computed properties for backward compatibility:
```swift
private var isTurnActive: Bool { speechState == .modelSpeaking }
private var ttsActive: Bool { speechState == .modelSpeaking }
```
3. Add centralized transition method:
```swift
private func transitionSpeech(to newState: SpeechState) {
    let old = speechState
    guard old != newState else { return }
    speechState = newState
    NSLog("[SPEECH] transition: %@ -> %@", String(describing: old), String(describing: newState))
    switch newState {
    case .idle:
        speechRecognizer?.setModelSpeaking(false)
        catPanel?.setTurnActive(false)
    case .modelSpeaking:
        speechRecognizer?.setModelSpeaking(true)
        catPanel?.setTurnActive(true)
    case .cooldown:
        speechRecognizer?.setModelSpeaking(false)
        catPanel?.setTurnActive(false)
    }
}
```
4. Update all message handlers to use `transitionSpeech()`:
- `.companionSpeech` → `transitionSpeech(to: .modelSpeaking)`
- `.transcription` (non-empty, non-finished) → `transitionSpeech(to: .modelSpeaking)`
- `.ttsStart` → `transitionSpeech(to: .modelSpeaking)` + stop catVoice + show bubble
- `.ttsEnd` → `transitionSpeech(to: .cooldown)` + schedule `.idle` after 1s
- `.turnComplete` → `transitionSpeech(to: .idle)` + flush catVoice + clear pendingTranscription
- `.interrupted` → `transitionSpeech(to: .idle)` + stop catVoice + hide bubble + clear pendingTranscription
5. Remove direct `self.isTurnActive = ...` and `self.ttsActive = ...` assignments
6. Remove direct `speechRecognizer?.setModelSpeaking(...)` calls from handlers (moved to transitionSpeech)
**Verification:** `make build && make test` — all 31 tests must pass identically
**Risk:** Medium — state logic consolidation touches multiple handlers. Computed properties ensure existing code paths produce identical behavior.
**Addresses:** Priorities 1, 2 (barge-in reliability, speech maintenance)

### Task 2B: ttsEnd Cooldown Cleanup
**File:** `VibeCat/Sources/VibeCat/AppDelegate.swift`
**Problem:** `.ttsEnd` sets `ttsActive = false` immediately but delays `setModelSpeaking(false)` by 500ms, creating a window where barge-in is harder than necessary.
**Change:**
1. Replace `.ttsEnd` handler with clean transition:
```swift
case .ttsEnd:
    NSLog("[GW-IN] onMessage: ttsEnd")
    transitionSpeech(to: .cooldown)
    // Cooldown period: modelSpeaking=false (barge-in uses lower threshold)
    // but bubble still evaluates hide. After 1s, transition to idle.
    Task { @MainActor [weak self] in
        try? await Task.sleep(nanoseconds: 1_000_000_000)
        guard let self, self.speechState == .cooldown else { return }
        self.transitionSpeech(to: .idle)
    }
    Task { @MainActor [weak self] in
        try? await Task.sleep(nanoseconds: 2_000_000_000)
        guard let self, self.speechState == .idle else { return }
        self.spriteAnimator?.setState(.idle)
    }
```
**Verification:** `make build && make test`
**Risk:** Low — cooldown is shorter than before (modelSpeaking=false immediately vs 500ms delay), making barge-in EASIER (aligns with Priority 1)
**Dependency:** Task 2A must be completed first (uses transitionSpeech)
**Addresses:** Priority 1 (barge-in always works)

---

## WAVE 3 (After Wave 2)

### Task 3A: Repeating Capture Timer
**File:** `VibeCat/Sources/VibeCat/ScreenAnalyzer.swift`
**Problem:** Non-repeating timer creates variable cadence — capture completion time affects next trigger.
**Change:**
1. Replace `scheduleNextCapture()` with `startCaptureTimer()`:
```swift
private func startCaptureTimer() {
    analysisTimer?.invalidate()
    let interval = AppSettings.shared.captureInterval
    analysisTimer = Timer.scheduledTimer(withTimeInterval: interval, repeats: true) { [weak self] _ in
        Task { @MainActor [weak self] in
            guard let self, self.isRunning, !self.isAnalyzing else { return }
            await self.runAnalysisCycle(forceSmartPath: false)
        }
    }
}
```
2. Update `start()` and `resume()` to call `startCaptureTimer()` instead of `scheduleNextCapture()`
3. Update `pause()` to invalidate timer
4. Remove `scheduleNextCapture()` calls from `runAnalysisCycle()` defer block and guard blocks
**Verification:** `make build && make test`
**Risk:** Low — timing behavior only, capture logic unchanged
**Dependency:** Task 1B must be completed first (isSpeechActive guard changes)
**Addresses:** Priority 4 (consistent 1Hz capture)

### Task 3B: Integration Build + Test + Commit + Push
**Changes:**
1. `make build && make test` — Swift 31/31
2. `cd backend/realtime-gateway && go test ./...` — 6 packages
3. `cd backend/adk-orchestrator && go test -race ./...` — 13 packages
4. Commit with descriptive message
5. Push to origin/master
**Verification:** All tests green, clean build
**Risk:** None — verification only

---

## Execution Order
```
Wave 1: [1A, 1B, 1C] — parallel, independent
Wave 2: [2A] → [2B] — sequential (2B depends on 2A)
Wave 3: [3A] → [3B] — sequential (3A depends on 1B; 3B is final verification)
```

## Out of Scope (with rationale)
1. **Gateway session.go mutex split** — GenAI SDK concurrent send support unverified. Risk > benefit for challenge deadline.
2. **CatPanel showBubble debouncing** — MainActor sequential execution prevents real race. No practical benefit.
3. **GatewayClient.isTTSSpeaking removal** — Still needed for ScreenAnalyzer Smart Path suppression.
4. **Priority queue for messages** — Server-authoritative interruption already provides correct priority. Client priority queue adds complexity without benefit.

## Test Plan
### Objective: Verify all 4 user priorities work after refactoring
### Prerequisites: VibeCat built and running, Gateway connected
### Test Cases:
1. **Barge-in during speech**: Model speaks → user says word → audio stops, bubble hides
2. **Speech maintenance**: Model speaks → keyboard typing → speech continues uninterrupted
3. **Bubble matches speech**: Model starts speaking → bubble shows full text → text accumulates correctly → bubble auto-hides after speech ends
4. **Independent capture**: Model speaks → screen changes → Console shows "[CAPTURE] captured" and "[CAPTURE] Fast Path" logs (not "suppressed")
5. **Late transcription rejection**: Fast speech → bubble doesn't flicker or show fragments
### Success Criteria: ALL 5 test cases pass
### How to Execute: `make run`, observe Console.app logs + visual behavior
