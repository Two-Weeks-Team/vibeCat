# VibeCat Execution — Issues & Gotchas

## [2026-03-03] Session Start

### Known Issues
1. Package.swift platform target mismatch (.macOS(.v26) vs README macOS 15+) — resolve in Task 3
2. /tmp/go-genai/ files show LSP errors (expected — they need go.mod context)
3. Screen recording entitlement may need additional entry for ScreenCaptureKit sandbox mode
4. CI uses self-hosted macOS runner — verify runner is active before relying on CI

### Risk Areas
- GenAI Go SDK Live API actual support (spike Task 1 will confirm)
- ADK Go SDK agent graph maturity (spike Task 2 will confirm)
- proactiveAudio / affectiveDialog availability in current API

### Anti-Patterns to Avoid
- Client making direct Gemini API calls
- Storing API key on client
- Mentioning GeminiCat anywhere
- Over-engineering before demo happy path works
