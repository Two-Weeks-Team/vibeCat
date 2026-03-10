---
name: gemini-api-implementation
description: Keep Gemini API integrations aligned with the latest official Google AI docs and changelog. Use when Codex is implementing, reviewing, refactoring, debugging, or upgrading Gemini model calls, Google Gen AI SDK usage, Live API sessions, speech generation, embeddings, built-in tool use, or ephemeral-token flows, especially when model IDs, tool support, or auth/session behavior may have changed.
---

# Gemini API Implementation

Use this skill to turn "use Gemini here" into code that matches both the current official docs and the existing repo architecture.

Treat Gemini model IDs, built-in tool support, Live API behavior, and auth flows as time-sensitive. The references in this skill are a dated snapshot, not a permanent source of truth.

## Start Here

1. Run `python3 scripts/scan_gemini_usage.py <repo-root>` before editing code.
2. Read [references/source-registry.md](references/source-registry.md) to pick the official pages that govern the task.
3. Read [references/current-guidance-2026-03-10.md](references/current-guidance-2026-03-10.md).
4. Read [references/vibecat-notes.md](references/vibecat-notes.md) when working in this repo or a similar split client/backend voice-agent architecture.
5. If the task says `latest`, touches preview models, or depends on tool/model support, browse the official pages again before patching code.

## Workflow

1. Inventory the repo first.
Use the scan script output to identify:
- SDKs already in use
- exact model IDs already shipped
- Live API vs `generateContent` vs embeddings usage
- built-in tools already enabled
- whether the client talks to Gemini directly or through a backend

2. Re-check the authoritative docs for the feature you are touching.
Always re-open the changelog and the relevant feature page when the work depends on:
- current model names or aliases
- preview/stable availability
- built-in tool support
- Live API session/auth behavior
- speech generation or embeddings configuration

3. Fit the fix to the existing codebase.
- Preserve the repo's current language and SDK unless the task explicitly requires migration.
- Prefer changing the smallest surface that brings the code back in line with the docs.
- Keep transport/auth boundaries intact. Do not move Gemini calls into a client just because a browser example exists.

4. Explain the implementation choice in code-review terms.
State which official pages governed the change, which model/tool choice you made, and any preview/stability tradeoff.

## Working Rules

- Prefer stable model IDs in production code. Use preview or experimental IDs only when the needed feature is unavailable on a stable line.
- Check the changelog before trusting any `-latest` alias or older preview suffix.
- Keep long-lived API keys off clients. Use ephemeral tokens only for direct client-to-Live connections.
- Preserve thought signatures and exact turn ordering when you touch Gemini 3 tool-calling flows outside the official SDK helpers.
- Do not combine unsupported built-in tools and function calling.
- Keep Live sessions single-modality: `TEXT` or `AUDIO`, never both.
- For native-audio Live sessions, send raw little-endian 16-bit PCM and expect 24 kHz output audio.
- On Live interruption events, stop playback and clear queued audio immediately.

## Decision Rules

### Standard generation
- Prefer Gemini 3 preview models for new reasoning-heavy text/code work only after rechecking current support on the models page.
- Migrate Gemini 3 reasoning controls toward `thinking_level`; keep `thinking_budget` only where the docs still require or allow it.
- If you are not using the official SDK's normal chat/history flow, explicitly preserve thought-signature parts.

### Built-in tools
- Use `google_search`, not legacy `google_search_retrieval`, for current models.
- Do not assume Gemini 3 supports every built-in tool. Maps grounding and Computer Use are excluded from Gemini 3 according to the current docs snapshot.
- Treat URL Context and File Search as constrained retrieval tools, not general crawling.
- Use Computer Use only with a sandboxed executor, user confirmation, and logs.

### Live API
- Re-check Live docs before editing model IDs. Live preview model names have already changed multiple times.
- Enable session resumption and context window compression for long-running sessions.
- Use `v1alpha` when the feature requires it, including affective dialog, proactive audio, and ephemeral-token provisioning.
- Do not set an explicit language code for native-audio Live output models.

### Speech generation and embeddings
- Keep `generateContent` TTS separate from Live native audio. They are different products with different constraints.
- Choose TTS voice and language settings from the current speech-generation guide, not from memory.
- Prefer `gemini-embedding-001` for embeddings. Use task types intentionally and normalize reduced-dimension vectors when needed.

## References

- [references/source-registry.md](references/source-registry.md)
- [references/current-guidance-2026-03-10.md](references/current-guidance-2026-03-10.md)
- [references/vibecat-notes.md](references/vibecat-notes.md)
- [scripts/scan_gemini_usage.py](scripts/scan_gemini_usage.py)
