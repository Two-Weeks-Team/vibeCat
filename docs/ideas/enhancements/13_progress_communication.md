# 13: Progress Display and User Communication Architecture

## Current Architecture (Already Implemented)

VibeCat already has a sophisticated progress display and user communication system. This document maps the existing architecture and defines enhancements to make it complete.

### Existing Message Protocol (AudioMessageParser.swift:3-33)

**32 ServerMessage types** defined in Swift enum. Key communication messages:

| Category | Message Type | Purpose | UI Rendering |
|---|---|---|---|
| Speech | `companionSpeech(text, emotion, urgency)` | Cat speaks | CatPanel speech bubble |
| Transcription | `transcription(text, finished)` | Assistant speech text | ChatBubbleView or ChatPanel |
| Input | `inputTranscription(text, finished)` | User speech text | Status bubble |
| Status | `turnState(state, source)` | speaking/idle | SpriteAnimator state |
| Progress | `processingState(flow, traceId, stage, label, detail, tool, sourceCount, active)` | Tool execution progress | Status bubble with spinner |
| Tool Result | `toolResult(tool, query, summary, sources)` | Tool completion | Bubble metadata |
| Navigator | `navigatorCommandAccepted(taskId, command, intentClass, intentConfidence)` | Task accepted | NavigatorOverlayPanel |
| Navigator | `navigatorStepPlanned(taskId, step, message)` | Step details | NavigatorOverlayPanel.showStep() |
| Navigator | `navigatorStepRunning(taskId, stepId, status)` | Step executing | Background log |
| Navigator | `navigatorStepVerified(taskId, stepId, status, observedOutcome)` | Step verified | ChatPanel |
| Safety | `navigatorRiskyActionBlocked(command, question, reason)` | Risk detected | Status bubble + ChatPanel |
| Clarification | `navigatorIntentClarificationNeeded(command, question, responseMode)` | Needs user input | ChatPanel |
| Navigator | `navigatorGuidedMode(taskId, reason, instruction)` | Fallback mode | Status bubble |
| Navigator | `navigatorCompleted(taskId, summary)` | Task done | NavigatorOverlayPanel.showCompletion() |
| Navigator | `navigatorFailed(taskId, reason)` | Task failed | NavigatorOverlayPanel.showCompletion() |
| Connection | `setupComplete(sessionId)` | Connected | Connection state |
| Connection | `liveSessionReconnecting(attempt, max)` | Reconnecting | Status update |
| TTS | `ttsStart(text)` / `ttsEnd` | Speech lifecycle | Speech state machine |

### Existing UI Components

| Component | File | Purpose |
|---|---|---|
| **CatPanel** | CatPanel.swift (699 lines) | Main character panel with speech/status bubbles, emotion indicator |
| **ChatBubbleView** | ChatBubbleView.swift (451 lines) | Speech and status bubbles (speech mode + status mode with spinner) |
| **NavigatorOverlayPanel** | NavigatorOverlayPanel.swift (285 lines) | Action icon + text + grounding badge + step progress + result |
| **TargetHighlightOverlay** | TargetHighlightOverlay.swift (62 lines) | Yellow rectangle around target AX elements |
| **DecisionOverlayHUD** | DecisionOverlayHUD.swift (90 lines) | Debug: trigger/vision/mediator/mood/cooldown |
| **CompanionChatPanel** | CompanionChatPanel.swift (238 lines) | Interactive chat panel (360x360) |
| **SpriteAnimator** | SpriteAnimator.swift | Cat states: idle, thinking, happy, surprised, frustrated, celebrating |

### Existing User Communication Flows

**1. Navigator Task Execution Flow** (AppDelegate.swift:1061-1178):
```
User speaks command
    -> navigatorCommandAccepted (ChatPanel: "Task accepted")
    -> navigatorStepPlanned (NavigatorOverlayPanel.showStep: action icon + grounding badge)
    -> AccessibilityNavigator executes (TargetHighlightOverlay shows target)
    -> navigatorStepVerified (NavigatorOverlayPanel.showResult: green/orange/red)
    -> navigatorCompleted (NavigatorOverlayPanel.showCompletion: "Done!")
```

**2. Risky Action Flow** (AppDelegate.swift:1122-1132):
```
Model requests risky action
    -> navigatorRiskyActionBlocked (Status bubble: "This action requires confirmation")
    -> CompanionChatPanel shows question
    -> User responds yes/no
    -> GatewayClient.sendNavigatorRiskConfirmation() (GatewayClient.swift:221-232)
```

**3. Ambiguous Intent Flow**:
```
User gives ambiguous command
    -> navigatorIntentClarificationNeeded (Status bubble + ChatPanel)
    -> User provides clarification
    -> GatewayClient.sendNavigatorClarificationResponse() (GatewayClient.swift:208-219)
```

**4. Processing State Flow** (AppDelegate.swift:1033-1057):
```
ADK orchestrator starts analysis
    -> processingState(stage: "analyzing", active: true) -> Status bubble with spinner
    -> processingState(stage: "searching", tool: "google_search") -> Tool icon in bubble
    -> toolResult(summary, sources) -> Metadata attached to next speech bubble
    -> processingState(active: false) -> Spinner stops
```

### Cat Emotion State Machine (SpriteAnimator.swift):
```
idle <-> thinking (on analysis start, 10s auto-timeout)
idle <-> happy (on task success)
idle <-> surprised (on unexpected input)
idle <-> frustrated (on repeated failures)
idle <-> celebrating (on milestone detection)
```

Emotion tags in speech: `[happy]`, `[surprised]`, `[thinking]`, `[concerned]`, `[idle]` (session.go line 287).

---

## Enhancements for Complete User Communication

### Enhancement A: ThinkingConfig UI Integration

**Current**: Cat enters `thinking` sprite state on analysis start. No thought text displayed.
**Enhanced**: Display actual model thought process from ThinkingConfig (Doc 01).

**New message type** (add to AudioMessageParser.swift after line 33):
```swift
case thinking(text: String)
```

**Go backend** (handler.go receiveFromGemini):
```go
// When Part.Thought == true:
sendJSON(conn, map[string]any{
    "type": "thinking",
    "text": part.Text,
})
```

**Swift UI** (AppDelegate.swift onMessage handler):
```swift
case .thinking(let text):
    spriteAnimator.setState(.thinking)
    catPanel.showStatusBubble(text, style: .thinking) // Dimmed text, italic font
```

**ChatBubbleView enhancement** (ChatBubbleView.swift):
- Add `.thinking` style: semi-transparent background, italic font, smaller text
- Auto-hide after 5 seconds or when speech starts

### Enhancement B: Enhanced Safety Confirmation UI

**Current**: `navigatorRiskyActionBlocked` shows status bubble + CompanionChatPanel question.
**Enhanced**: Dedicated confirmation dialog with clear approve/deny buttons and risk explanation.

The backend already sends risk data. The enhancement is purely Swift UI:

```swift
// New confirmation panel (or enhance CompanionChatPanel):
class SafetyConfirmationView: NSView {
    let riskLabel: NSTextField     // "This action may be risky"
    let reasonLabel: NSTextField   // "Contains: rm -rf"
    let commandLabel: NSTextField  // "Run rm -rf /tmp/test"
    let approveButton: NSButton    // "Allow" (green)
    let denyButton: NSButton       // "Block" (red)
}
```

**Integration point**: AppDelegate.swift line 1122-1132 already handles `navigatorRiskyActionBlocked`. Enhance the UI displayed there.

### Enhancement C: Multi-Step Progress Visualization

**Current**: NavigatorOverlayPanel shows "Step X of Y" with action icon and grounding badge.
**Enhanced**: Timeline view showing all planned steps with current progress.

**Existing step progress** (NavigatorOverlayPanel.swift:162-166):
```swift
progressLabel.stringValue = VibeCatL10n.navigatorStepProgress(current: stepNumber, total: total)
```

**Enhancement**: When navigatorStepPlanned arrives with total > 1, show a horizontal step indicator:
```
[1: Focus] -> [2: Type] -> [3: Submit]
   done        active       pending
```

**Grounding badges** already color-coded (NavigatorOverlayPanel.swift:271-284):
- AX = blue, Vision = purple, Hotkey = gray, System = teal

### Enhancement D: Connection Status Indicator

**Current**: GatewayClient.ConnectionState (disconnected, connecting, connected, failed).
**Enhanced**: Persistent status indicator in the cat panel.

Already tracked via `onStateChange` callback (GatewayClient.swift:15). Enhancement is visual:
- Small colored dot near cat: green=connected, yellow=reconnecting, red=disconnected
- Tooltip showing session details
- Reconnection progress: "Reconnecting 2/30..."

### Enhancement E: Voice Command Feedback

**Current**: User speech is transcribed via `inputTranscription`. No explicit "I heard you" feedback.
**Enhanced**: Brief visual acknowledgment when speech is detected.

Already exists in `NavigatorVoiceCommandDetector` (Core/NavigatorVoiceCommandDetector.swift). Enhancement:
- Flash the cat's ears or show a brief listening indicator
- Display detected command text in status bubble before processing starts

## Implementation Priority

| Enhancement | Effort | Impact | Depends On |
|---|---|---|---|
| A: ThinkingConfig UI | Low | Medium | Doc 01 (ThinkingConfig backend) |
| B: Safety Confirmation UI | Medium | High | Doc 05 (Safety backend) |
| C: Multi-Step Progress | Low | Medium | Already implemented in part |
| D: Connection Status | Low | Low | None |
| E: Voice Command Feedback | Low | Medium | None |

## Files to Modify

### Swift Client
1. `Sources/Core/AudioMessageParser.swift:3-33` — Add `thinking` message type
2. `Sources/VibeCat/AppDelegate.swift:933-1291` — Handle new message types in onMessage
3. `Sources/VibeCat/CatPanel.swift` — Add thinking bubble style, connection indicator
4. `Sources/VibeCat/ChatBubbleView.swift` — Add thinking style (italic, semi-transparent)
5. `Sources/VibeCat/NavigatorOverlayPanel.swift` — Enhanced multi-step timeline
6. `Sources/VibeCat/SpriteAnimator.swift` — Potentially add ear animation for voice detection

### Go Backend
1. `internal/ws/handler.go` (receiveFromGemini) — Forward Part.Thought as "thinking" message
2. `internal/live/session.go` — ThinkingConfig addition (see Doc 01)

## Verification
- Enable ThinkingConfig -> verify "thinking" messages appear in Swift client
- Trigger risky action -> verify confirmation UI appears with approve/deny
- Execute 3-step navigation -> verify timeline shows all steps with progress
- Disconnect network -> verify connection indicator turns red
- Speak command -> verify brief visual acknowledgment before processing
