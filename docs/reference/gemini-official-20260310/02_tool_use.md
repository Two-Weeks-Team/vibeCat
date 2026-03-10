# Tool Use

Last checked: 2026-03-10

Use this file when adding or reviewing Gemini built-in tools.

## Core takeaways

- Google Search is the current built-in grounding path for current Gemini lines.
- Google Maps grounding is model-limited and should be checked explicitly before use.
- URL Context and File Search are constrained retrieval features, not arbitrary crawlers.
- Code Execution is useful when the model must compute or inspect data, but it does not replace deterministic local automation.
- Computer Use is high-risk and requires a sandbox plus explicit human confirmation.

## Current snapshot details

- Gemini 3 currently supports:
  - Google Search
  - File Search
  - Code Execution
  - URL Context
  - function calling
- Gemini 3 currently does not support:
  - Google Maps grounding
  - Computer Use
- The Google Search guide currently recommends `google_search` over legacy retrieval naming.
- The Maps grounding guide currently lists support on:
  - Gemini 2.5 Pro
  - Gemini 2.5 Flash
  - Gemini 2.5 Flash-Lite
  - Gemini 2.0 Flash
- The URL Context guide currently states:
  - up to 20 URLs per request
  - up to 34 MB per URL
- The File Search guide currently lists support for:
  - `gemini-3-pro-preview`
  - `gemini-3-flash-preview`
  - `gemini-2.5-pro`
  - `gemini-2.5-flash`
  - `gemini-2.5-flash-lite`

## Tool fit rules

- Re-check model support before combining a tool with a given model family.
- Treat built-in tools and custom function calling as a compatibility check, not an assumption.
- Keep source attribution rules in mind for Search and Maps outputs.
- Use Computer Use only when browser or GUI control is genuinely required.
- Treat URL Context and File Search as bounded retrieval systems with strict limits.
- Treat Code Execution as a controlled model-side runtime, not a place to install arbitrary libraries.

## Original sources

- Google Search: [source](https://ai.google.dev/gemini-api/docs/google-search), [snapshot](originals/tools/01_google_search.html)
- Maps grounding: [source](https://ai.google.dev/gemini-api/docs/maps-grounding), [snapshot](originals/tools/02_maps_grounding.html)
- Code Execution: [source](https://ai.google.dev/gemini-api/docs/code-execution), [snapshot](originals/tools/03_code_execution.html)
- URL Context: [source](https://ai.google.dev/gemini-api/docs/url-context), [snapshot](originals/tools/04_url_context.html)
- Computer Use: [source](https://ai.google.dev/gemini-api/docs/computer-use), [snapshot](originals/tools/05_computer_use.html)
- File Search: [source](https://ai.google.dev/gemini-api/docs/file-search), [snapshot](originals/tools/06_file_search.html)
