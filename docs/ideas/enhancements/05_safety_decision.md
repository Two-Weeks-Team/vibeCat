# P0-5: Safety Decision Handling

**SDK Verification**: No built-in safety_decision field in Go SDK. This is a custom implementation pattern from Google's computer-use web-agent example. VibeCat implements it as application-level logic.

**Live API compatible**: YES (custom logic, no SDK dependency)

**Current Code** (handler.go):
- `handleLiveToolCall()` at line 3645-3664 — dispatches tool calls WITHOUT safety checks
- `handleNavigateTextEntryToolCall()` at line 3666-3738 — directly executes text entry
- No confirmation dialog before risky actions

**Current Code** (session.go):
- System prompt at line 282-286 mentions safety: "Always wait for user confirmation before risky actions (git push, delete, submit)"
- But this is prompt-based only — no enforcement in code

**Implementation**:
1. Define risky action patterns (file delete, git push, system commands, password fields)
2. Add safety classification before executing each tool call
3. Send confirmation request to Swift client via WebSocket
4. Wait for user response before proceeding
5. Include safety_acknowledgement in tool response

**Go Code**:
```go
// internal/safety/classifier.go
package safety

import (
    "fmt"
    "strings"
)

type RiskLevel string
const (
    RiskLow    RiskLevel = "low"
    RiskMedium RiskLevel = "medium"
    RiskHigh   RiskLevel = "high"
)

type Assessment struct {
    Level       RiskLevel
    Reason      string
    RequiresAck bool
}

var highRiskPatterns = []string{
    "rm -rf", "git push", "git reset", "sudo", "delete",
    "DROP TABLE", "FORMAT", "password", "credentials",
}

func Classify(toolName string, args map[string]any) Assessment {
    text, _ := args["text"].(string)
    for _, pattern := range highRiskPatterns {
        if strings.Contains(strings.ToLower(text), strings.ToLower(pattern)) {
            return Assessment{
                Level:       RiskHigh,
                Reason:      fmt.Sprintf("Contains risky pattern: %s", pattern),
                RequiresAck: true,
            }
        }
    }
    // navigate_open_url with non-https
    if toolName == "navigate_open_url" {
        url, _ := args["url"].(string)
        if !strings.HasPrefix(url, "https://") {
            return Assessment{Level: RiskMedium, Reason: "Non-HTTPS URL", RequiresAck: true}
        }
    }
    return Assessment{Level: RiskLow, RequiresAck: false}
}
```

Handler integration:
```go
// In handleLiveToolCall(), before execution:
assessment := safety.Classify(fc.Name, fc.Args)
if assessment.RequiresAck {
    // Send confirmation request to Swift client
    sendJSON(conn, map[string]any{
        "type":   "safety_confirmation",
        "tool":   fc.Name,
        "reason": assessment.Reason,
        "taskId": taskID,
    })
    // Wait for user response (with 30s timeout)
    select {
    case approved := <-state.safetyResponseChan:
        if !approved {
            return &genai.FunctionResponse{
                Name:     fc.Name,
                Response: map[string]any{"result": "user_denied", "reason": "User declined risky action"},
            }
        }
    case <-time.After(30 * time.Second):
        return &genai.FunctionResponse{
            Name:     fc.Name,
            Response: map[string]any{"result": "timeout", "reason": "Safety confirmation timed out"},
        }
    }
}
```

**Verification**:
- Send "Run rm -rf /tmp/test" command
- Verify confirmation dialog appears in Swift UI
- Verify denial produces "user_denied" tool response
- Verify timeout produces "timeout" tool response

**Risks**:
- Blocking execution while waiting for user confirmation
- Need timeout to prevent deadlocks
- False positives may annoy users (too many confirmations)
