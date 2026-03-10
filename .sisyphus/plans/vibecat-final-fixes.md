# VibeCat Final Fixes Plan

**Created:** 2026-03-10
**Deadline:** 2026-03-16 (Gemini Live Agent Challenge)
**Base Commit:** `1aad51e`
**Status:** All tests passing (Swift 31/31, Go Gateway 6/6, Go Orchestrator 13/13)

---

## P0 — Must Fix Before Submission

### FIX-1: Implement consecutiveThreshold logic in SpeechRecognizer
- **File:** `VibeCat/Sources/VibeCat/SpeechRecognizer.swift`
- **Lines:** 87-88 (declaration), 120-144 (tap handler)
- **Problem:** `consecutiveAboveCount` and `consecutiveThreshold: Int = 3` are declared but NEVER USED. The tap handler at line 133 does `guard rms >= threshold else { return }` which immediately sends audio on the first buffer exceeding threshold. Single impulse sounds (keyboard thock, mouse click) trigger barge-in.
- **Fix:** Add counter logic in tap handler:
  ```swift
  if rms >= threshold {
      self.consecutiveAboveCount += 1
      guard self.consecutiveAboveCount >= self.consecutiveThreshold else { return }
  } else {
      self.consecutiveAboveCount = 0
      return
  }
  ```
- **Verify:** Build passes, keyboard typing no longer triggers audio send during manual test
- **Time:** 30min

### FIX-2: Enable voice processing for echo cancellation
- **File:** `VibeCat/Sources/VibeCat/SpeechRecognizer.swift`
- **Line:** 98
- **Problem:** `try inputNode.setVoiceProcessingEnabled(false)` explicitly disables macOS AEC. When cat speaks through speakers, mic captures speaker output causing self-interruption loop.
- **Fix:** Change `false` to `true`. Requires macOS 14+ (already our minimum target).
- **Risk:** May introduce ~10ms latency or change audio capture quality. Test with speakers at 50%/100% volume.
- **Verify:** Cat speaking does not trigger its own barge-in when using speakers (no headphones)
- **Time:** 5min

### FIX-3: Raise RMS thresholds
- **File:** `VibeCat/Sources/VibeCat/SpeechRecognizer.swift`
- **Lines:** 79-80
- **Problem:** `rmsThreshold = 0.01` (-40dB) is too low. Mechanical keyboard RMS is -30 to -20dB (0.03-0.1). `bargeInThreshold = 0.05` (-26dB) also allows keyboard sounds during model speech.
- **Fix:** `rmsThreshold: Float = 0.03`, `bargeInThreshold: Float = 0.08`
- **Verify:** Keyboard typing does not trigger audio send. Normal speech volume still works.
- **Time:** 5min

### FIX-4: Add sync.Mutex to 4 ADK Orchestrator agents
- **Files:**
  - `backend/adk-orchestrator/internal/agents/mood/mood.go` (lines 18-27: errorCount, lastErrorTime, silenceStart)
  - `backend/adk-orchestrator/internal/agents/celebration/celebration.go` (lines 25-33: lastCelebration, recentMessages)
  - `backend/adk-orchestrator/internal/agents/engagement/engagement.go` (lines 26-34: lastActivity, lastRestReminder)
  - `backend/adk-orchestrator/internal/agents/scheduler/scheduler.go` (lines 17-31: utterances, cooldown, silence)
- **Problem:** Each agent is instantiated once in `graph.go` but `Run()` is called concurrently from multiple `/analyze` HTTP requests. Mutable fields have no synchronization — `errorCount++`, `a.lastCelebration = time.Now()`, etc. are read-modify-write races.
- **Fix:** Add `mu sync.Mutex` to each Agent struct. Wrap all field reads/writes in `a.mu.Lock()`/`a.mu.Unlock()`.
- **Verify:** `go test -race ./...` passes for adk-orchestrator
- **Time:** 1hr

### FIX-5: Switch remaining pro models to flash-lite
- **Files:**
  - `backend/adk-orchestrator/internal/agents/memory/memory.go` line 173: `"gemini-3.1-pro-preview"` → `"gemini-3.1-flash-lite-preview"`
  - `backend/adk-orchestrator/internal/agents/search/search.go` line 163: `"gemini-3.1-pro-preview"` → `"gemini-3.1-flash-lite-preview"`
- **Problem:** Memory and Search agents use pro model (avg 8,242ms). Benchmark shows flash-lite is 5.3x faster (avg 1,550ms) with sufficient quality for all VibeCat use cases (correct JSON, appropriate Korean, valid reasoning).
- **Fix:** Replace model strings in 2 files.
- **Verify:** Go tests pass. Manual test confirms memory summaries and search results still coherent.
- **Time:** 15min

---

## P1 — Should Fix

### FIX-6: Hide bubble on interruption
- **File:** `VibeCat/Sources/VibeCat/AppDelegate.swift`
- **Lines:** 299-305
- **Problem:** On barge-in, `catVoice?.stop()` stops audio immediately but `panel?.hideBubble()` is never called. Bubble persists 2-10 seconds showing text that was never fully spoken.
- **Fix:** Add `panel?.showBubble(text: "")` or `panel?.hideBubble()` after `self.pendingTranscription = ""` in the `.interrupted` case handler.
- **Verify:** During cat speech, user speaks → audio stops AND bubble disappears simultaneously.
- **Time:** 10min

### FIX-7: Add mutex to Gateway Live Session
- **File:** `backend/realtime-gateway/internal/live/session.go`
- **Lines:** 83-116
- **Problem:** `SendAudio()`, `SendVideo()`, `SendText()` called from WebSocket handler goroutine. `Receive()` called from receiveFromGemini goroutine. `Close()` from cleanup. No synchronization on the underlying `*genai.Session`.
- **Fix:** Add `mu sync.Mutex` to Session struct, wrap all method calls.
- **Verify:** `go test -race ./...` passes for realtime-gateway
- **Time:** 30min

### FIX-8: Extract normalizeLanguage() to shared package
- **Files:** 9 Go files across both services
- **Problem:** Identical 15-line function duplicated 9 times.
- **Fix:** Create `backend/shared/lang.go` or extract to each service's `internal/utils/lang.go`.
- **Verify:** All Go tests pass after refactoring.
- **Time:** 30min

---

## Total Estimated Time
- P0: ~2 hours
- P1: ~1.5 hours
- **Total: ~3.5 hours**

## Post-Fix Verification
1. `swift build` — clean
2. `swift test` — 31/31
3. `go test ./...` (gateway) — all pass
4. `go test -race ./...` (orchestrator) — all pass, no races
5. `./infra/deploy.sh` — deploy both services
6. `make run` — manual device test:
   - Keyboard typing does NOT trigger barge-in
   - Speaker output does NOT cause self-interruption
   - Cat speech bubble matches spoken audio
   - Bubble disappears on interruption
   - Cat follows mouse smoothly
7. Commit + push
