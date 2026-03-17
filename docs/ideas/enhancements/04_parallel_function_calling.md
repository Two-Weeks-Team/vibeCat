# P0-4: Parallel Function Calling

## Title
P0-4: Parallel Function Calling

## SDK Verification
- Gemini can return multiple `Part.FunctionCall` entries in a single response — confirmed in SDK
- `LiveToolResponseInput.FunctionResponses []*FunctionResponse` — already accepts multiple responses
- Live API compatible: YES (model returns multiple FCs, handler must process all)

## Current Code (handler.go)
- `pendingFC` fields at lines 175-184 — tracks ONE pending FC at a time
- `setPendingFC()` at line 462 — sets single pending FC
- `hasPendingFCForTask()` at line 475 — checks for single FC
- `advancePendingFCStep()` at line 481 — advances single FC step
- `handleLiveToolCall()` at line 3645 — handles one tool call at a time
- `receiveFromGemini()` at line 3338 — extracts tool calls from response

## Current Pattern
Sequential FC execution — one FC at a time, wait for result, then next.

## Implementation
1. In `receiveFromGemini()`, collect ALL FunctionCall parts from a single response
2. Execute independent FCs in parallel using goroutines
3. Collect all results and send back via `SendToolResponse()` (which already accepts slice)
4. Keep sequential mode for dependent FCs (when one FC's output feeds another)

## Go Code
```go
// In receiveFromGemini(), when multiple FCs detected:
func (h *Handler) handleParallelToolCalls(calls []*genai.FunctionCall) []*genai.FunctionResponse {
    results := make([]*genai.FunctionResponse, len(calls))
    var wg sync.WaitGroup

    for i, call := range calls {
        wg.Add(1)
        go func(idx int, fc *genai.FunctionCall) {
            defer wg.Done()
            result, err := h.executeSingleToolCall(fc)
            if err != nil {
                results[idx] = &genai.FunctionResponse{
                    Name:     fc.Name,
                    Response: map[string]any{"error": err.Error()},
                }
                return
            }
            results[idx] = result
        }(i, call)
    }
    wg.Wait()
    return results
}

// Send all results at once:
if err := session.SendToolResponse(results); err != nil {
    slog.Error("failed to send parallel tool responses", "err", err)
}
```

## When to use parallel vs sequential
- Parallel: Independent tools (e.g., focus_app + open_url for different apps)
- Sequential: Dependent tools (e.g., focus_app then text_entry in same app)
- Detection: If tool calls target different apps/contexts, they are independent

## Verification
- Send a prompt like "Open Terminal and Chrome at the same time"
- Verify both navigate_focus_app calls execute concurrently
- Measure latency reduction vs sequential execution

## Risks
- Race conditions if parallel FCs target the same app/resource
- Need conflict detection: if two FCs target same app, execute sequentially
- Error in one FC should not block others
