# Gemini Official Sync (2026-03-10)

This folder stores a dated Gemini documentation sync checked on 2026-03-10.

It has two goals:

1. Provide a compact project-facing summary of the official Gemini API pages the project depends on.
2. Preserve original HTML snapshots of the source pages under `originals/`.

## Contents

- `01_core_and_models.md`: model, Gemini 3, speech generation, embeddings
- `02_tool_use.md`: Google Search, Maps, Code Execution, URL Context, Computer Use, File Search
- `03_live_api.md`: Live API behavior, session rules, tools, auth, best practices
- `04_examples_and_operations.md`: examples, troubleshooting, changelog, refresh workflow
- `SOURCE_MANIFEST.md`: human-readable source-to-file mapping
- `originals/manifest.json`: fetched URL metadata
- `originals/`: raw HTML snapshots grouped by category

## How To Use This Folder

1. Read the summary file that matches the feature you are touching.
2. Open the matching HTML snapshot in `originals/` if you need the original page body preserved at fetch time.
3. Re-open the live official page if the task depends on the latest model support, preview IDs, or changelog changes after 2026-03-10.

## Notes

- The AI Studio showcase page redirected to Google sign-in during capture. The saved HTML is still preserved, but it is an auth-gated landing page instead of the showcase body.
- The original HTML snapshots are not normalized Markdown. They are preserved as fetched.
- Existing reference folders such as `docs/reference/gemini/`, `docs/reference/gemini-api/`, and `docs/reference/gemini-live/` remain untouched. This folder is an additional dated sync.
