# Source Registry

Last checked: 2026-03-10

Use this file to decide which official page to reopen before editing code. Treat these URLs as the authoritative source. Prefer the official page over any cached local summary, including this skill.

## Core model pages

- Models overview: https://ai.google.dev/gemini-api/docs/models
Use for current model IDs, stable/preview/latest alias rules, and capability tables.

- Gemini 3 guide: https://ai.google.dev/gemini-api/docs/gemini-3
Use for `thinking_level`, thought signatures, Gemini 3 migration rules, and Gemini 3 tool limits.

- Speech generation: https://ai.google.dev/gemini-api/docs/speech-generation
Use for `generateContent` text-to-speech, voice inventory, languages, and TTS model choice.

- Embeddings: https://ai.google.dev/gemini-api/docs/embeddings
Use for embedding model choice, `taskType`, `outputDimensionality`, and dimension guidance.

## Built-in tool pages

- Google Search grounding: https://ai.google.dev/gemini-api/docs/google-search
Use for `google_search`, pricing semantics, citations, and supported combinations.

- Google Maps grounding: https://ai.google.dev/gemini-api/docs/maps-grounding
Use for `googleMaps`, supported models, attribution duties, and widget behavior.

- Code execution: https://ai.google.dev/gemini-api/docs/code-execution
Use for built-in Python execution, library limits, and image/code workflows.

- URL Context: https://ai.google.dev/gemini-api/docs/url-context
Use for max URL count, supported content types, and current limitations.

- Computer Use: https://ai.google.dev/gemini-api/docs/computer-use
Use for browser-control loops, HITL requirements, and sandboxing expectations.

- File Search: https://ai.google.dev/gemini-api/docs/file-search
Use for hosted RAG/file store support, supported models, and retrieval constraints.

## Live API pages

The user-provided Live URLs may redirect to shorter canonical pages in search results. When browsing, accept either form if the content matches.

- Live API overview: https://ai.google.dev/gemini-api/docs/live-api
- Get started with SDK: https://ai.google.dev/gemini-api/docs/live-api/get-started-sdk
- Get started with raw WebSocket: https://ai.google.dev/gemini-api/docs/live-api/get-started-websocket
- Capabilities: https://ai.google.dev/gemini-api/docs/live-api/capabilities
- Tool use: https://ai.google.dev/gemini-api/docs/live-api/tools
- Session management: https://ai.google.dev/gemini-api/docs/live-api/session-management
- Ephemeral tokens: https://ai.google.dev/gemini-api/docs/live-api/ephemeral-tokens
- Best practices: https://ai.google.dev/gemini-api/docs/live-api/best-practices

Use the Live pages for:
- native audio model behavior
- VAD and interruption handling
- tool support in Live sessions
- session resumption and context window compression
- client-to-server auth with ephemeral tokens

## Examples

- AI Studio showcase: https://aistudio.google.com/app/apps?source=showcase&showcaseTag=gemini-3&showcaseTag=featured
Use for current end-to-end patterns and product examples. This page is dynamic; do not treat any copied example as stable API documentation.

## Maintenance pages

- Troubleshooting: https://ai.google.dev/gemini-api/docs/troubleshooting
Use for API-version mismatches, status-code handling, temperature guidance, and retry/fallback decisions.

- Changelog: https://ai.google.dev/gemini-api/docs/changelog
Open this before changing model IDs, depending on preview models, or assuming a Live/API feature still behaves the same way.

## Recheck Order

1. Changelog
2. Models overview
3. Feature-specific page
4. API reference or SDK reference if the exact field name matters

## When Not To Trust This Skill Snapshot

Reopen the official pages immediately when:
- the task says `latest`, `new`, `current`, or `supported`
- a preview model is involved
- a `-latest` alias is involved
- a Live API model name or auth flow is involved
- a tool is being combined with another tool or with function calling
