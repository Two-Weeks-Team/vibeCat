# P1-8: Native computer_use Tool

**SDK Verification (CONFIRMED via go doc v1.49.0)**:
- `genai.ComputerUse{Environment, ExcludedPredefinedFunctions}` — EXISTS
- `genai.EnvironmentBrowser` — EXISTS
- `genai.EnvironmentOS` — DOES NOT EXIST (browser-only)
- Live API compatible: NO

**CRITICAL FINDING**: The native computer_use tool only supports `EnvironmentBrowser`. There is NO `EnvironmentOS` environment. This means it CANNOT control the macOS desktop — only web browsers via Playwright/CDP.

**Applicability**: LIMITED — Could only be used for VibeCat's CDP-based browser control, not for desktop navigation which is VibeCat's primary use case.

**Current Approach** (VibeCat's 5 custom FC tools):
- navigate_text_entry, navigate_hotkey, navigate_focus_app, navigate_open_url, navigate_type_and_submit
- These work across ALL macOS apps (not just browsers)
- Uses AX tree + CDP + Vision for grounding
- Already has self-healing and verification

**Assessment**: VibeCat's current custom FC approach is SUPERIOR to native computer_use for desktop apps because:
1. computer_use is browser-only (no macOS desktop support)
2. VibeCat already handles 5 action types across all apps
3. Triple-source grounding (AX + CDP + Vision) provides better context than computer_use's screenshot-only approach

**Potential Partial Use**: Could use computer_use for browser-specific interactions in CDP mode as an alternative to current navigate_* tools when operating within Chrome.

**Implementation** (if pursued for browser-only):
```go
// Browser-only computer use tool declaration:
browserTools := &genai.Tool{
    ComputerUse: &genai.ComputerUse{
        Environment:                genai.EnvironmentBrowser,
        ExcludedPredefinedFunctions: []string{"drag_and_drop"},
    },
}

// Normalized coordinate handling (0-1000 scale):
type CoordinateNormalizer struct {
    Width  int
    Height int
}
func (cn *CoordinateNormalizer) ToPixel(normX, normY int) (int, int) {
    return int(float64(normX) / 1000.0 * float64(cn.Width)),
           int(float64(normY) / 1000.0 * float64(cn.Height))
}
```

**Recommendation**: DEFER this enhancement. Current custom FC approach is better suited for VibeCat's macOS desktop use case. Revisit when/if Google adds `EnvironmentOS` support.

**Verification**: N/A (deferred)

**Risks**:
- Would require major refactor of tool handling for limited browser-only benefit
- EnvironmentOS may never be added to Go SDK
- Normalized coordinates (0-1000) vs VibeCat's AX-tree-based targeting are fundamentally different approaches
