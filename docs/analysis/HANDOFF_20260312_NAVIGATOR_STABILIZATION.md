# Navigator Stabilization Handoff

Date: 2026-03-12

## Branch and commits

- Branch: `codex/navigator-stabilization-20260312`
- Commit 1: `61c92e7` `Add navigator clarification and system action routing`
- Commit 2: `68c7265` `Harden macOS action worker focus and audio stability`

## What changed

### Cross-service navigator contract

- gateway planner now separates `literal`, `intrinsic`, and `screen-derived` text-entry payloads
- ambiguous insertion requests no longer complete as `focus-only`
- ADK escalator can now return `resolvedText`
- client/server clarification flow now distinguishes `confirmation` vs `provide_details`
- voice command rerouting now catches navigator-style speech more aggressively
- deterministic `system_action` handling exists for basic macOS volume control

### Local macOS runtime

- AX candidate search now walks the live AX tree instead of relying only on the short snapshot summary
- text entry execution is now `resolve target -> AX focus -> caret placement -> click activation point -> verify -> paste`
- text-input descriptor caching no longer gets poisoned by arbitrary non-input focused labels
- assistant transcription assembly merges overlapping partials instead of duplicating text
- audio device hot-plug churn is debounced and duplicate same-device snapshots are ignored

## Deployment/runtime state

- Cloud Run gateway revision already deployed during this work: `realtime-gateway-00050-fq2`
- App smoke log after the latest local restart: `/tmp/vibecat-live-20260311-231357.log`
- macOS text-entry activation attempts now log with `[NAV-AX]`

## Verification already run

- `cd VibeCat && swift test`
- `cd backend/realtime-gateway && go test ./...`
- `cd tests/e2e && go test ./...`
- `git diff --check`

## Next session priorities

1. Run live user-driven smoke again on Codex with:
   - no cursor in the composer
   - first direct insert
   - second follow-up insert
2. Check `/tmp/vibecat-live-20260311-231357.log` for `[NAV-AX]` lines to see which activation stage actually succeeds.
3. If text entry still fails without an existing cursor, dump the frontmost Codex AX tree around the chosen candidate and confirm the chosen activation point is inside the real editable region.
4. If the `Codex` proactive hint reappears, verify the connected gateway revision and inspect `proactiveContextHintText` runtime output rather than only the static code path.

## Local references used for the macOS text-entry redesign

- `/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX26.2.sdk/System/Library/Frameworks/ApplicationServices.framework/Versions/A/Frameworks/HIServices.framework/Versions/A/Headers/AXAttributeConstants.h`
- `/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX26.2.sdk/System/Library/Frameworks/ApplicationServices.framework/Versions/A/Frameworks/HIServices.framework/Versions/A/Headers/AXUIElement.h`
- `/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX26.2.sdk/System/Library/Frameworks/ApplicationServices.framework/Versions/A/Frameworks/HIServices.framework/Versions/A/Headers/AXActionConstants.h`
- `/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX26.2.sdk/System/Library/Frameworks/CoreGraphics.framework/Headers/CGEvent.h`
