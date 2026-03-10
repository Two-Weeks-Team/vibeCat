# Menu and Runtime Operations Spec

This specification defines operational behavior from menu setup to full runtime execution.
It is implementation guidance for a new team and excludes source code.

## 1) Menu Information Architecture

### Top-level menu sections
- Status
- Language
- Voice
- Chattiness
- Models
- Capture
- Appearance
- Advanced
- Background Music
- Set API Key
- Reset Settings
- About
- Reconnect
- Pause/Resume
- Mute/Unmute
- Quit

### Required submenu groups
- Models: Vision Model, Live Model, TTS Model, Thinking Level, Media Resolution
- Capture: Size, Interval, Sensitivity, Image Quality
- Appearance: Character, Cat Size, Follow Speed, Bubble Duration
- Advanced: Google Search, Proactive Audio, Affective Dialog, Launch at Login
- Background Music: Enabled, Track, Volume

## 2) Menu State Rules

### Status row
- Connected: green indicator + running message
- Reconnecting: yellow indicator + attempt/max value
- Disconnected: red indicator + key-required or disconnected message

### Action labels
- Pause item switches label between `Pause` and `Resume`.
- Mute item switches label between `Mute Voice` and `Unmute Voice`.
- Checkmarks reflect persisted settings on app launch and after each change.

## 3) Connection UX and Validation Flow

### Onboarding trigger
- If backend session is not established at startup, show onboarding prompt before analysis loop begins.

### Validation pipeline
1. client sends device registration request
2. backend issues session token
3. branch handling
   - valid: start normal operation
   - invalid / rejected: show error and keep onboarding state
   - rate/network/server issues: allow retry without storing any Gemini key on device

### Persistence rule
- Successful connection updates the current session token and triggers live reconnect.

## 4) Runtime Lifecycle Sequence

### Startup sequence
1. create state and settings objects
2. initialize animation and UI support modules
3. initialize capture, live transport, audio, and agent modules
4. wire callbacks and dependencies
5. create floating overlay panel and host UI
6. initialize status bar menu and callbacks
7. branch by connection and pause state

### Analysis loop sequence
1. background capture task runs every 1 second minimum
2. capture the window under the cursor
3. changed-content filter
4. send changed frame to Gemini fast path
5. send smart-path analysis only when speech is not active or capture is forced
6. update transcription/bubble state from live turn events

### Force-capture sequence
1. user circle gesture detected
2. capture frontmost window immediately
3. analyze and evaluate
4. route through live or fallback speech path

## 5) Speech and Audio Routing Rules

### Primary path (live connected)
- Send state packet and optional full-window image to live stream.
- Receive streaming audio and transcription events.
- Keep turn state active until completion/interruption.

### Fallback path (live disconnected)
- Generate text response through fallback request.
- Apply emotion tagging policy for voice style.
- Synthesize and play local PCM audio.

### Background music coordination
- Speech start: fade out music
- Speech end: fade in music
- Initial greeting complete: start background music if enabled

## 6) Interruption and Turn Safety Rules

### Interruption handling
- User speech barge-in has highest priority.
- On local barge-in detection: stop queued audio immediately, clear current bubble, and notify gateway.
- On interruption event: keep the user input path active, reset assistant turn state, and update adaptive metrics.

### Timeout handling
- If active turn exceeds timeout window, force reset turn state.

### Duplicate speech prevention
- Mediator must suppress duplicate or low-value outputs during cooldown windows.

## 7) Multi-Monitor and Boundary Rules

### Overlay positioning
- Overlay panel tracks the current screen under cursor.
- Cat sprite remains within visible bounds for all size options.

### Bubble and emotion indicators
- Default offsets apply in normal positions.
- Near screen edges, indicator placement adapts to remain visible.

## 8) Operational Scenarios Checklist

### Scenario A - First launch without backend session
- expected: onboarding prompt appears, analysis does not start

### Scenario B - Valid connect request
- expected: session token issued, connection starts, status becomes connected

### Scenario C - Pause and resume
- expected: pause stops capture and proactive triggers; resume restarts loops

### Scenario D - Mute and unmute
- expected: mute clears voice output immediately; unmute restores speech playback

### Scenario E - Live disconnect and reconnect
- expected: reconnect status updates, retry attempts visible, operation resumes on recovery

### Scenario F - Gesture-triggered analysis
- expected: force capture path executes and produces response if decision permits

### Scenario G - High-significance event
- expected: full-window capture branch is used before speech routing

### Scenario H - Background music behavior
- expected: fade-out on speech start and fade-in on speech end

## 9) Acceptance Gates

- Menu controls exist and persist state.
- Startup and runtime branch behavior matches this spec.
- Live and fallback speech paths both operate.
- Gesture, interruption, timeout, and reconnection safety paths operate.
- Multi-monitor and edge placement behavior remain stable.
