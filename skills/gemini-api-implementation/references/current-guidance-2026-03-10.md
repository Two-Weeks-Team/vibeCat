# Current Guidance (2026-03-10)

This is a compact snapshot of the official Gemini docs checked on 2026-03-10. Use it to orient yourself quickly, then reopen the official page before changing production code.

## Table of Contents

1. Model-selection rules
2. Tool-support rules
3. Live API rules
4. Speech generation rules
5. Embeddings rules
6. Change-management rules
7. Source map

## 1. Model-selection rules

- Gemini 3 is currently preview-only. The current text models highlighted in the docs are `gemini-3-pro-preview` and `gemini-3-flash-preview`.
- Gemini 3 models have a 1M input context window and 64k output window, with a January 2025 knowledge cutoff.
- Gemini 3 reasoning uses `thinking_level`. Default is `high`. Gemini 3 Flash also supports `minimal` and `medium`.
- `thinking_budget` still exists for backward compatibility, but the Gemini 3 guide recommends migrating toward `thinking_level` and not sending both in the same request.
- Thought signatures matter for Gemini 3. If you are not relying on the official SDK to replay chat history, preserve signature-bearing parts exactly, especially around function calls and image editing.
- For production code, prefer stable model IDs when they support the feature. Use preview IDs only when the feature surface requires them.

## 2. Tool-support rules

### Gemini 3

- Gemini 3 supports Google Search, File Search, Code Execution, URL Context, and standard function calling.
- Gemini 3 does not support Google Maps grounding or Computer Use.
- Gemini 3 currently does not support combining built-in tools with custom function calling in one request.

### Google Search grounding

- Use `google_search` for current models.
- Older Gemini 1.5 guidance still mentions `google_search_retrieval`, but it is legacy.
- Gemini 3 search grounding billing changed: the docs state billing is per search query for Gemini 3, and this began on 2026-01-05. Older 2.5-and-earlier behavior is billed per prompt.
- Search grounding can be combined with URL Context and Code Execution.

### Google Maps grounding

- Not available on Gemini 3.
- Supported models listed in the current docs: Gemini 2.5 Pro, Gemini 2.5 Flash, Gemini 2.5 Flash-Lite, and Gemini 2.0 Flash.
- Maps results come with source metadata and optional widget tokens. Attribution rules are stricter than normal search citations.

### URL Context

- Supported models listed in the current docs: Gemini 3 Flash, Gemini 3 Pro, Gemini 2.5 Pro, Gemini 2.5 Flash, and Gemini 2.5 Flash-Lite.
- URL Context currently supports up to 20 URLs per request.
- The per-URL content size limit is 34 MB.
- It handles text, image, and PDF retrieval, but not paywalled content, YouTube videos, Google Workspace docs, or audio/video files.
- The current docs say built-in tool use such as URL Context is currently unsupported with function calling.

### File Search

- The current docs list support for `gemini-3-pro-preview`, `gemini-3-flash-preview`, `gemini-2.5-pro`, `gemini-2.5-flash` and preview versions, and `gemini-2.5-flash-lite` and preview versions.
- Treat File Search as hosted RAG over a file store, not as an ad hoc filesystem crawler.

### Code Execution

- The models page shows Code Execution as supported on Gemini 3 Pro Preview and Gemini 3 Flash Preview.
- The code-execution guide highlights Gemini 3 Flash "visual thinking", where the model writes and runs Python to inspect images.
- The execution environment is constrained. You cannot install arbitrary libraries.

### Computer Use

- Computer Use is safety-sensitive and should run in a sandboxed environment with user confirmation, logs, and allowlists/blocklists.
- Use it only when the task genuinely requires browser/GUI control. Do not substitute it for deterministic local automation.

## 3. Live API rules

- Live API is still preview.
- Current guides emphasize two integration modes:
  - server-to-server proxying through your backend
  - direct client-to-Live connections using ephemeral tokens for production safety
- Current server-side examples still use 16-bit PCM mono input and 24 kHz output audio.
- Live sessions allow exactly one response modality per session: `TEXT` or `AUDIO`.
- Native-audio Live output models auto-select language and the current guide says not to set an explicit language code for them.
- Current limits in the Live docs:
  - audio-only sessions: 15 minutes without compression
  - audio+video sessions: 2 minutes without compression
  - connection lifetime: about 10 minutes
  - context window: 128k for native-audio output models, 32k for other Live models
- Use context window compression and session resumption for long-lived sessions.
- Resumption handles remain valid for 2 hours after the last session termination.
- On interruption events, clear queued playback immediately. The docs explicitly say generated audio that has not reached the user is discarded.
- Affective dialog and proactive audio require `v1alpha`.
- The current Live tool docs list support on `gemini-2.5-flash-native-audio-preview-12-2025` for:
  - Google Search: yes
  - function calling: yes
  - Google Maps: no
  - Code Execution: no
  - URL Context: no
- Live API tool responses are manual. Unlike unary `generateContent`, automatic tool-response handling is not available.

## 4. Speech generation rules

- `generateContent` TTS and Live native audio are different surfaces. Do not conflate them.
- The TTS guide currently lists two preview TTS models:
  - `gemini-2.5-flash-preview-tts`
  - `gemini-2.5-pro-preview-tts`
- TTS takes text input only and produces audio output only.
- TTS output uses 24 kHz PCM in the examples.
- The current TTS guide lists 30 prebuilt voices including `Zephyr`, `Puck`, `Kore`, `Schedar`, `Zubenelgenubi`, and `Fenrir`.
- The guide lists Korean as `ko-KR` and supports automatic language detection across the supported language set.

## 5. Embeddings rules

- The current recommended stable embedding model is `gemini-embedding-001`.
- The embeddings quickstart says the default dimension is 3072 and recommends 768, 1536, or 3072 depending on storage/latency needs.
- Reduced dimensions should be normalized manually for similarity work; the docs call out 3072 as normalized by default.
- Use `taskType` intentionally:
  - `RETRIEVAL_QUERY`
  - `RETRIEVAL_DOCUMENT`
  - `SEMANTIC_SIMILARITY`
  - `CLASSIFICATION`
- When using `RETRIEVAL_DOCUMENT`, add a title for better retrieval quality.
- The older experimental embedding model `gemini-embedding-exp-03-07` is already called out as deprecated in the docs snapshot; do not start new work on it.

## 6. Change-management rules

- Open the changelog before changing model strings. The 2025 release notes already show multiple Live model shutdowns and replacement preview IDs.
- Do not trust `-latest` aliases in long-lived production code without verifying current behavior on the models page.
- Treat preview models as volatile even if they are billable and permitted in production.
- The troubleshooting guide currently recommends:
  - checking API version mismatches on 400s
  - switching to another model temporarily on 500/503s
  - increasing timeout on 504s
  - keeping Gemini 3 temperature at its default unless you have a clear reason to change it

## 7. Source Map

- Models overview: https://ai.google.dev/gemini-api/docs/models
- Gemini 3 guide: https://ai.google.dev/gemini-api/docs/gemini-3
- Speech generation: https://ai.google.dev/gemini-api/docs/speech-generation
- Embeddings: https://ai.google.dev/gemini-api/docs/embeddings
- Google Search: https://ai.google.dev/gemini-api/docs/google-search
- Google Maps: https://ai.google.dev/gemini-api/docs/maps-grounding
- Code Execution: https://ai.google.dev/gemini-api/docs/code-execution
- URL Context: https://ai.google.dev/gemini-api/docs/url-context
- Computer Use: https://ai.google.dev/gemini-api/docs/computer-use
- File Search: https://ai.google.dev/gemini-api/docs/file-search
- Live API overview: https://ai.google.dev/gemini-api/docs/live-api
- Live get started: https://ai.google.dev/gemini-api/docs/live-api/get-started-sdk
- Live WebSocket: https://ai.google.dev/gemini-api/docs/live-api/get-started-websocket
- Live capabilities: https://ai.google.dev/gemini-api/docs/live-api/capabilities
- Live tools: https://ai.google.dev/gemini-api/docs/live-api/tools
- Live session management: https://ai.google.dev/gemini-api/docs/live-api/session-management
- Live ephemeral tokens: https://ai.google.dev/gemini-api/docs/live-api/ephemeral-tokens
- Troubleshooting: https://ai.google.dev/gemini-api/docs/troubleshooting
- Changelog: https://ai.google.dev/gemini-api/docs/changelog
