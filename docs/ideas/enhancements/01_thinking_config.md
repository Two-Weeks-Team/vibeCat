# P0-1: ThinkingConfig + Thought Signatures

## Title
P0-1: ThinkingConfig + Thought Signatures

## SDK Verification (CONFIRMED via go doc v1.49.0)
- `genai.ThinkingConfig{IncludeThoughts bool, ThinkingBudget *int32, ThinkingLevel ThinkingLevel}` — EXISTS
- `LiveConnectConfig.ThinkingConfig *ThinkingConfig` — EXISTS (field confirmed in LiveConnectConfig)
- `Part.Thought bool` — EXISTS
- `Part.ThoughtSignature []byte` — EXISTS
- Live API compatible: YES

## Current Code (session.go)
- `buildLiveConfig()` at line 465-533 — LiveConnectConfig construction, currently NO ThinkingConfig set
- `Session.Receive()` at line 126-128 — receives LiveServerMessage
- Tool declarations at lines 367-462

## Current Code (handler.go)
- `receiveFromGemini()` at line 3338-3613 — processes response parts
- `handleLiveToolCall()` at line 3645-3664 — dispatches FC

## Implementation
1. Add `ThinkingConfig` to `buildLiveConfig()` in session.go after line 506 (after RealtimeInputConfig)
2. Handle `Part.Thought` and `Part.ThoughtSignature` in `receiveFromGemini()` in handler.go
3. Forward thought text to Swift client as a new message type `thinking`
4. Preserve ThoughtSignature in conversation history for multi-step FC workflows

## Go Code
```go
// In buildLiveConfig(), after line 506:
lc.ThinkingConfig = &genai.ThinkingConfig{
    IncludeThoughts: true,
}

// In receiveFromGemini(), handling response parts:
for _, part := range msg.ServerContent.ModelTurn.Parts {
    if part.Thought {
        // Forward thinking text to client for UI display
        sendJSON(conn, map[string]any{
            "type": "thinking",
            "text": part.Text,
        })
    }
    if len(part.ThoughtSignature) > 0 {
        // Store signature for conversation history preservation
        state.setLastThoughtSignature(part.ThoughtSignature)
    }
}
```

## Verification
After implementation, check Gemini logs for `thinking_config` in request. Verify `Part.Thought=true` parts appear in responses.

## Risks
- ThinkingConfig increases token usage (thinking budget)
- Only works with thinking-capable models (not gemini-2.5-flash-native-audio-preview)
- May need model upgrade to Gemini 3.x for full thinking support
