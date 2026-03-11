# VibeCat Winning Alignment Report

> Historical note (2026-03-11): this report optimizes for the former Live Agents framing. It is not the current submission-alignment source.

Updated: 2026-03-10

## Goal

The primary goal is not just to ship a working prototype, but to maximize the chance of winning the Gemini Live Agent Challenge in the Live Agents category.

The official judging weights are:

- Innovation & Multimodal User Experience: 40%
- Technical Implementation & Agent Architecture: 30%
- Demo & Presentation: 30%

This means VibeCat must optimize for:

- A seamless "see, hear, speak" experience that feels live.
- Natural interruption handling via Gemini Live API VAD.
- A clear and robust agent architecture on Google Cloud.
- Submission materials that prove the system is real, deployed, and reproducible.

## Architecture Direction

To align with the judging criteria, the speech architecture should follow these rules:

1. ADK analyzes screen changes and produces grounded suggestions.
2. Gemini Live API is the only runtime speech engine.
3. Backend is authoritative for assistant turn state.
4. Frontend renders state and audio, but does not invent turn semantics.
5. Bubble text should follow Live transcription, not a parallel text-only channel.

This avoids a disjointed experience where one subsystem decides text while another synthesizes separate audio.

## Implemented In This Refactor

### 1. Live-only assistant speech path

- Removed the separate Gateway TTS speaking path from the active runtime flow.
- Screen-analysis suggestions are now injected back into the Live session as grounded prompts.
- Voice-search answers are also injected into the same Live session instead of using a parallel TTS path.

Expected benefit:

- Higher "Live" factor.
- More seamless multimodal UX.
- Fewer race conditions between bubble text and audio.

### 2. Server-authoritative turn state

- Added a `turnState` WebSocket message.
- Gateway now emits `turnState=speaking` when Live model audio starts.
- Gateway now emits `turnState=idle` when the Live turn ends or is interrupted.

Expected benefit:

- Clear shared state across backend and frontend.
- More defensible architecture diagram and protocol story for judges.

### 3. Client state machine now prefers shared turn state

- Client parses the new `turnState` event.
- App state now uses server turn state as the primary signal for entering and exiting speaking/cooldown.
- Screen analysis suppression now uses `isModelTurnActive` and `lastModelTurnEndTime` instead of a TTS-specific concept.

Expected benefit:

- Cleaner speech gating.
- Better interruption semantics.
- More coherent explanation in demo and README.

## Why This Improves Winning Odds

### Innovation & Multimodal UX

- The product now better matches the "beyond text" requirement.
- Voice, vision, and interruption handling all flow through Gemini Live.
- The experience is less obviously stitched together.

### Technical Execution & Agent Architecture

- ADK remains valuable, but is positioned correctly as an analysis/orchestration layer.
- The backend protocol is easier to explain and defend.
- Fewer concurrent speech subsystems means fewer edge cases and less desync risk.

### Demo & Presentation

- The story becomes simpler:
  - "The cat sees your screen through ADK-backed vision analysis."
  - "The cat hears and talks through Gemini Live."
  - "The same Live session handles natural interruption."

That is much easier to demonstrate in a 4-minute video.

## Verification Completed

- `swift test`
- `go test ./...` in `backend/realtime-gateway`

Both passed after this refactor.

## Remaining Highest-Value Work

These are the highest-leverage tasks for actually winning:

1. Real-device barge-in verification and log capture.
2. Tight 4-minute demo showing real software, not narration over mockups.
3. Clear architecture diagram that highlights:
   - macOS client
   - Realtime Gateway on Cloud Run
   - ADK Orchestrator on Cloud Run
   - Gemini Live API
   - Firestore / memory if used in the flow
4. Public README with spin-up instructions and reproducibility proof.
5. Public blog post explicitly stating it was created for this hackathon and tagged `#GeminiLiveAgentChallenge`.

## Recommended Demo Story

Use a single scenario that proves all three weighted criteria quickly:

1. Developer is coding and the cat quietly watches.
2. A meaningful screen change occurs and the cat proactively helps.
3. The user interrupts naturally mid-speech.
4. The user asks a follow-up by voice.
5. The cat answers in the same Live conversation.
6. Show Cloud Run deployment proof and the architecture diagram.

This sequence demonstrates:

- multimodal input
- proactive screen-aware behavior
- natural barge-in
- distinct persona/voice
- Google Cloud deployment
- defensible agent architecture
