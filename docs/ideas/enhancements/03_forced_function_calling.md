# P0-3: Forced Function Calling (ANY/NONE Modes)

## Title
P0-3: Forced Function Calling (ANY/NONE Modes)

## SDK Verification (CONFIRMED via go doc v1.49.0)
- `genai.ToolConfig{FunctionCallingConfig *FunctionCallingConfig}` — EXISTS
- `genai.FunctionCallingConfig{Mode FunctionCallingConfigMode, AllowedFunctionNames []string}` — EXISTS
- `FunctionCallingConfigModeAuto` = "AUTO" — EXISTS
- `FunctionCallingConfigModeAny` = "ANY" — EXISTS
- `FunctionCallingConfigModeNone` = "NONE" — EXISTS
- `FunctionCallingConfigModeValidated` = "VALIDATED" — EXISTS
- Live API compatible: NO — LiveConnectConfig does NOT have ToolConfig field

## Applicability
ADK Orchestrator only (batch calls). The gateway's Live API does not support ToolConfig.

## Current Code (adk-orchestrator)
- `internal/agents/tooluse/tooluse.go:46-62` — classifyPrompt for tool selection
- `internal/agents/tooluse/tooluse.go:163-179` — detectFastPath for keyword routing
- `internal/agents/tooluse/tooluse.go:181-268` — execute() for tool invocation

## Implementation
1. Add ToolConfig to tool classification requests with `ModeAny` when fast-path detected
2. Add ToolConfig with `ModeNone` for casual conversation in mediator agent
3. Use `AllowedFunctionNames` to constrain which tools are available per context

## Go Code
```go
// In tooluse.go execute(), when fast-path detected:
func (a *Agent) execute(ctx context.Context, kind string, query string) (*Result, error) {
    config := &genai.GenerateContentConfig{
        Tools: a.toolsForKind(kind),
    }

    // If tool kind is known from fast-path detection, force FC
    if kind != models.ToolKindNone {
        config.ToolConfig = &genai.ToolConfig{
            FunctionCallingConfig: &genai.FunctionCallingConfig{
                Mode: genai.FunctionCallingConfigModeAny,
            },
        }
    }
    // ...
}

// In mediator agent, for casual conversation detection:
func (a *MediatorAgent) shouldSkipTools(ctx context.Context, input string) bool {
    config := &genai.GenerateContentConfig{
        ToolConfig: &genai.ToolConfig{
            FunctionCallingConfig: &genai.FunctionCallingConfig{
                Mode: genai.FunctionCallingConfigModeNone,
            },
        },
    }
    // Model will only produce text, never function calls
}
```

## Verification
- With ANY mode: verify model ALWAYS returns FunctionCall, never plain text
- With NONE mode: verify model NEVER returns FunctionCall
- Check tool_config appears in API request logs

## Risks
- ANY mode forces FC even when model has no good answer — may cause hallucinated tool calls
- NONE mode completely disables tools — must be certain conversation is casual
- NOT available in Live API — gateway sessions are unaffected
