# Core And Models

Last checked: 2026-03-10

Use this file when choosing model lines, reasoning controls, speech generation, or embeddings.

## Core takeaways

- The current official models overview highlights Gemini 3 preview models for new reasoning-heavy work.
- Gemini 3 uses `thinking_level` as the preferred reasoning control and keeps `thinking_budget` mainly for backward compatibility.
- Gemini 3 thought-signature handling matters if chat state is replayed outside the official SDK helpers.
- Speech generation and Live native audio are separate surfaces and should not be treated as interchangeable.
- The current recommended stable embeddings model is `gemini-embedding-001`.

## Current snapshot details

- Gemini 3 preview models highlighted in the current docs:
  - `gemini-3-pro-preview`
  - `gemini-3-flash-preview`
- Gemini 3 docs currently call out:
  - 1M input context window
  - 64k output token limit
  - January 2025 knowledge cutoff
- Gemini 3 reasoning controls:
  - `thinking_level` is the preferred setting
  - default is `high`
  - Gemini 3 Flash supports `minimal` and `medium`
- Speech generation docs currently list:
  - `gemini-2.5-flash-preview-tts`
  - `gemini-2.5-pro-preview-tts`
- The speech-generation guide currently lists 30 prebuilt voices, including `Zephyr`, `Puck`, `Kore`, `Schedar`, `Zubenelgenubi`, and `Fenrir`.
- Embeddings docs currently recommend:
  - `gemini-embedding-001`
  - default dimension `3072`
  - reduced dimensions such as `1536` and `768` when storage or latency pressure matters

## Model selection guidance

- Prefer stable model IDs when the needed feature exists on a stable line.
- Re-check the changelog before trusting `-latest` aliases or preview suffixes in production code.
- Use preview-only models only when the required capability is unavailable elsewhere.

## Speech generation guidance

- Use the speech-generation guide for TTS model IDs, prebuilt voice names, and language-code behavior.
- Keep TTS configuration separate from Live session configuration.

## Embeddings guidance

- Use `taskType` intentionally.
- When reducing embedding dimensions, normalize vectors explicitly if the downstream retrieval flow depends on normalization.
- When using retrieval-style embeddings, add document titles where the official guide recommends them.
- The current guide explicitly calls out `RETRIEVAL_QUERY`, `RETRIEVAL_DOCUMENT`, `SEMANTIC_SIMILARITY`, and `CLASSIFICATION`.

## Original sources

- Models overview: [source](https://ai.google.dev/gemini-api/docs/models), [snapshot](originals/core/01_models.html)
- Gemini 3: [source](https://ai.google.dev/gemini-api/docs/gemini-3), [snapshot](originals/core/02_gemini_3.html)
- Speech generation: [source](https://ai.google.dev/gemini-api/docs/speech-generation), [snapshot](originals/core/03_speech_generation.html)
- Embeddings: [source](https://ai.google.dev/gemini-api/docs/embeddings), [snapshot](originals/core/04_embeddings.html)
