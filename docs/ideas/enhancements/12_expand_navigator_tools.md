# 12: Expand Navigator FC Tools for Complete macOS Desktop Control

## Background

VibeCat's Swift client (`AccessibilityNavigator.swift`, 2313 lines) already implements 8 action types via `NavigatorActionType` enum (NavigatorModels.swift:11-21):

| Action Type | Swift Implementation | Exposed as Gemini FC? |
|---|---|:---:|
| `focusApp` | NSWorkspace.shared.openApplication() | YES (navigate_focus_app) |
| `openURL` | Browser navigation | YES (navigate_open_url) |
| `hotkey` | CGEvent keyboard injection (80+ keys) | YES (navigate_hotkey) |
| `pasteText` | NSPasteboard + Cmd+V | Partially (navigate_text_entry) |
| `copySelection` | Cmd+C | NO |
| `pressAX` | AXUIElementPerformAction(kAXPressAction) at line 538 | NO |
| `clickCoordinates` | CGEvent mouse down/up at lines 553-596 | NO |
| `systemAction` | AppleScript volume/brightness at line 598 | NO |
| `waitFor` | Condition wait at line 601 | NO |

Additionally, `AutomationMCPClient.swift` (line 75-92) provides mouseClick(), mouseMove(), mouseDoubleClick() via MCP JSON-RPC.

**Critical gap**: Only 5 of 8+ capabilities are exposed to Gemini. The model cannot tell VibeCat to click a button, scroll a page, or copy text.

## New FC Tools to Add

### Tool 1: `navigate_click` (Priority: CRITICAL)

**Why**: Many macOS UI elements (buttons, checkboxes, menu items) have no keyboard shortcut. Without click, VibeCat cannot interact with dialog boxes, permission prompts, or non-standard UIs.

**Swift Implementation**: ALREADY EXISTS at AccessibilityNavigator.swift lines 553-596 (`clickCoordinates`) and lines 469-551 (`pressAX`). Uses:
- CGEvent(mouseEventSource: source, mouseType: .leftMouseDown, mouseCursorPosition: point) at line 584
- AXUIElementPerformAction(element, kAXPressAction) at line 538
- NavigatorTargetDescriptor already has clickX/clickY fields (NavigatorModels.swift:71-72)
- screenBasisId validation prevents blind clicks when screen state changes

**Go FC Declaration** (add to session.go navigatorToolDeclarations() after line 462):
```go
{
    Name:        "navigate_click",
    Description: "Click on a UI element on the screen. Preferred method: describe the element by role and label for AX-tree-based clicking (more reliable). Fallback: provide normalized coordinates (0.0-1.0) from the screenshot for coordinate-based clicking. Always describe what you see before clicking.",
    Parameters: &genai.Schema{
        Type: genai.TypeObject,
        Properties: map[string]*genai.Schema{
            "target": {
                Type:        genai.TypeString,
                Description: "Description of the UI element to click, e.g. 'OK button', 'Save dialog checkbox', 'Close button in top-left'. Used for AX tree element resolution.",
            },
            "app": {
                Type:        genai.TypeString,
                Description: "Target application name. If provided, the app is focused before clicking.",
            },
            "x": {
                Type:        genai.TypeNumber,
                Description: "Normalized X coordinate (0.0-1.0) from the screenshot. Use only when AX target description is insufficient.",
            },
            "y": {
                Type:        genai.TypeNumber,
                Description: "Normalized Y coordinate (0.0-1.0) from the screenshot. Use only when AX target description is insufficient.",
            },
            "double_click": {
                Type:        genai.TypeBoolean,
                Description: "Double-click instead of single click. Default false.",
            },
        },
        Required: []string{"target"},
    },
},
```

**Go Handler** (add to handler.go handleLiveToolCall() switch at line 3648):
```go
case "navigate_click":
    return h.handleNavigateClickToolCall(fc, taskID, state)
```

```go
func (h *Handler) handleNavigateClickToolCall(fc *genai.FunctionCall, taskID string, state *liveSessionState) {
    target, _ := fc.Args["target"].(string)
    app, _ := fc.Args["app"].(string)
    x, hasX := fc.Args["x"].(float64)
    y, hasY := fc.Args["y"].(float64)
    doubleClick, _ := fc.Args["double_click"].(bool)

    var actionType string
    if hasX && hasY {
        actionType = "click_coordinates"
    } else {
        actionType = "press_ax"
    }

    step := navigatorStep{
        ID:         generateStepID(),
        ActionType: actionType,
        TargetApp:  app,
        TargetDescriptor: targetDescriptor{
            Label:  target,
            ClickX: x,
            ClickY: y,
        },
        ExpectedOutcome: fmt.Sprintf("Clicked %s", target),
        Confidence:      0.8,
        RiskLevel:       "low",
    }
    // ... queue step via setPendingFC
}
```

### Tool 2: `navigate_scroll` (Priority: HIGH)

**Why**: Cannot navigate long code files, web pages, or lists without scroll.

**Swift Implementation**: NOT YET EXISTS. Need to add to AccessibilityNavigator.swift.

**macOS API** (CGEvent scroll wheel):
```swift
// Add to NavigatorActionType enum in NavigatorModels.swift:
case scroll

// Add to AccessibilityNavigator.swift execute(step:) switch:
case .scroll:
    let deltaY = step.systemAmount  // positive = scroll up, negative = scroll down
    let deltaX = 0  // horizontal scroll (future)
    guard let event = CGEvent(scrollWheelEvent2Source: nil,
                               units: .pixel,
                               wheelCount: 1,
                               wheel1: Int32(deltaY)) else {
        return .failed("Could not create scroll event", reason: .focusNotReady, phase: .performAction)
    }
    event.post(tap: .cghidEventTap)
    return .success("Scrolled \(deltaY > 0 ? "up" : "down") by \(abs(deltaY)) pixels", phase: .performAction)
```

**Go FC Declaration**:
```go
{
    Name:        "navigate_scroll",
    Description: "Scroll the active window or element. Use positive amount for scrolling up, negative for scrolling down. Default unit is 'lines' (about 3 lines per scroll unit).",
    Parameters: &genai.Schema{
        Type: genai.TypeObject,
        Properties: map[string]*genai.Schema{
            "direction": {
                Type:        genai.TypeString,
                Description: "Scroll direction: 'up', 'down', 'left', 'right'.",
            },
            "amount": {
                Type:        genai.TypeInteger,
                Description: "Number of scroll units. Default 3 (about one page section).",
            },
            "target": {
                Type:        genai.TypeString,
                Description: "Target app or area to scroll. If omitted, scrolls the frontmost window.",
            },
        },
        Required: []string{"direction"},
    },
},
```

### Tool 3: `navigate_copy_paste` (Priority: MEDIUM)

**Why**: Cannot read selected text or paste content from clipboard without keyboard shortcut workarounds.

**Swift Implementation**: ALREADY EXISTS. `copySelection` at line 456, `pasteText` at line 389.

**Go FC Declaration**:
```go
{
    Name:        "navigate_copy_paste",
    Description: "Copy selected text from the active app or paste text into it. Use action='copy' to read the current selection, action='paste' to paste provided text.",
    Parameters: &genai.Schema{
        Type: genai.TypeObject,
        Properties: map[string]*genai.Schema{
            "action": {
                Type:        genai.TypeString,
                Description: "Either 'copy' (read selection) or 'paste' (write text).",
            },
            "text": {
                Type:        genai.TypeString,
                Description: "Text to paste. Required when action='paste', ignored when action='copy'.",
            },
        },
        Required: []string{"action"},
    },
},
```

### Tool 4: `navigate_system` (Priority: MEDIUM)

**Why**: Cannot control volume, brightness, or other system settings.

**Swift Implementation**: ALREADY EXISTS at line 598 (`systemAction`). Uses AppleScript.

**Go FC Declaration**:
```go
{
    Name:        "navigate_system",
    Description: "Control macOS system settings: volume, brightness, dark mode, do-not-disturb.",
    Parameters: &genai.Schema{
        Type: genai.TypeObject,
        Properties: map[string]*genai.Schema{
            "command": {
                Type:        genai.TypeString,
                Description: "System command: 'volume_up', 'volume_down', 'volume_mute', 'volume_set', 'brightness_up', 'brightness_down'.",
            },
            "value": {
                Type:        genai.TypeString,
                Description: "Value for 'set' commands. E.g., '50' for 50% volume.",
            },
        },
        Required: []string{"command"},
    },
},
```

## Future Tools (v0.4.0+)

### Tool 5: `navigate_drag` (NOT YET IN SWIFT)
```swift
// CGEvent drag sequence:
let source = CGEventSource(stateID: .hidSystemState)
let down = CGEvent(mouseEventSource: source, mouseType: .leftMouseDown, mouseCursorPosition: startPoint, mouseButton: .left)
down?.post(tap: .cghidEventTap)
// Intermediate moves
for point in interpolatedPoints {
    let move = CGEvent(mouseEventSource: source, mouseType: .leftMouseDragged, mouseCursorPosition: point, mouseButton: .left)
    move?.post(tap: .cghidEventTap)
    usleep(10000) // 10ms between moves
}
let up = CGEvent(mouseEventSource: source, mouseType: .leftMouseUp, mouseCursorPosition: endPoint, mouseButton: .left)
up?.post(tap: .cghidEventTap)
```

### Tool 6: `navigate_right_click` (NOT YET IN SWIFT)
```swift
let down = CGEvent(mouseEventSource: source, mouseType: .rightMouseDown, mouseCursorPosition: point, mouseButton: .right)
let up = CGEvent(mouseEventSource: source, mouseType: .rightMouseUp, mouseCursorPosition: point, mouseButton: .right)
// Or via AX: AXUIElementPerformAction(element, kAXShowMenuAction)
```

## Implementation Order

| Phase | Tools | Effort | Impact |
|---|---|---|---|
| v0.2.0 | navigate_click + navigate_scroll | Medium | Unlocks 80% of blocked UI interactions |
| v0.3.0 | navigate_copy_paste + navigate_system | Low | Clipboard + system control |
| v0.4.0 | navigate_drag + navigate_right_click | Medium | Full desktop control |

## Files to Modify

### Go Backend (realtime-gateway)
1. `internal/live/session.go:367-462` — Add new FC declarations to navigatorToolDeclarations()
2. `internal/ws/handler.go:3645-3663` — Add cases to handleLiveToolCall() switch
3. `internal/ws/handler.go` — Add handler functions for each new tool
4. `internal/ws/navigator.go` — Add step planning for new action types

### Swift Client (VibeCat)
1. `Sources/Core/NavigatorModels.swift:11-21` — Add `scroll` case to NavigatorActionType enum
2. `Sources/VibeCat/AccessibilityNavigator.swift:174-605` — Add scroll handler to execute() switch
3. No changes needed for click/copy/paste/system — already implemented

### System Prompt (session.go)
Update `commonLivePrompt` (lines 147-287) to document new tools:
```
NAVIGATOR TOOLS:
Available: navigate_text_entry, navigate_hotkey, navigate_focus_app, navigate_open_url, 
navigate_type_and_submit, navigate_click, navigate_scroll, navigate_copy_paste, navigate_system.

navigate_click: Click on a UI element. Describe the target for AX-tree clicking (preferred),
or provide x,y coordinates from the screenshot (fallback). Always verify the click result.

navigate_scroll: Scroll the active window. Direction: up/down/left/right. Amount: number of scroll units.
```

## Verification
- "Click the OK button" -> verify navigate_click FC is generated with target="OK button"
- "Scroll down to see more" -> verify navigate_scroll FC with direction="down"
- "Copy this code" -> verify navigate_copy_paste with action="copy"
- "Turn the volume down" -> verify navigate_system with command="volume_down"
- Measure tool call accuracy across 20 test scenarios
<!-- OMO_INTERNAL_INITIATOR -->
