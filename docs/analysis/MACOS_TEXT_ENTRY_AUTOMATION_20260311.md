# macOS Text Entry Automation Notes

Date: 2026-03-11

## Apple AX facts used

- `kAXFocusedAttribute` is writable for focusable elements, and only `true` can be set to move keyboard focus.
- `kAXPositionAttribute` is the top-left corner of the element in global screen coordinates.
- `kAXSizeAttribute` is the visible size of the element.
- `kAXSelectedTextRangeAttribute` is writable for editable text elements and can be used to place a zero-length caret.
- `AXUIElementCopyElementAtPosition` uses the same top-left-relative global screen coordinates for hit testing.
- `kAXPressAction` simulates a click. `kAXConfirmAction` simulates Return in a text field, not focus acquisition.

Primary source in local SDK:

- `/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX26.2.sdk/System/Library/Frameworks/ApplicationServices.framework/Versions/A/Frameworks/HIServices.framework/Versions/A/Headers/AXAttributeConstants.h`
- `/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX26.2.sdk/System/Library/Frameworks/ApplicationServices.framework/Versions/A/Frameworks/HIServices.framework/Versions/A/Headers/AXUIElement.h`
- `/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX26.2.sdk/System/Library/Frameworks/ApplicationServices.framework/Versions/A/Frameworks/HIServices.framework/Versions/A/Headers/AXActionConstants.h`
- `/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX26.2.sdk/System/Library/Frameworks/CoreGraphics.framework/Headers/CGEvent.h`

## Runtime policy

Text entry should not be treated as a single `focus_input_field` step. The reliable flow is:

1. Resolve the best text-input candidate from the live AX tree.
2. Try AX focus (`kAXFocusedAttribute = true` / `AXPress`).
3. Try zero-length `selectedTextRange` at the end of the field to establish a caret.
4. If AX focus is insufficient, click a safe activation point inside the resolved field using global screen coordinates.
5. Verify that the focused element or current AX context now matches the intended text input.
6. Only then paste text.

## Why this exists

Some apps expose editable controls in AX but do not reliably accept keyboard focus from `kAXFocusedAttribute` alone. In those cases, a resolved target still needs explicit cursor activation before input. Codex-style composer fields are one of the known cases.
