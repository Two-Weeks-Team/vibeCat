# Reference Sources Manifest

Generated: 2026-03-03

## Scope

- This manifest tracks the curated local reference library under `docs/reference/`.
- All saved source pages are Markdown snapshots converted from official documentation pages and official repositories.
- `gemini-official-20260310/originals/` is an exception: it stores raw HTML snapshots fetched from the requested official URLs on 2026-03-10.

## Stored Domains

- Challenge:
  - `challenge/gemini-live-agent-challenge/`
- ADK:
  - `adk/streaming/`
  - `adk/get-started/`
  - `adk/components/`
  - `adk/deployment/`
  - `adk/observability/`
  - `adk/runtime/`
  - `adk/reference/`
  - `adk/safety/`
- Gemini:
  - `gemini/live-api/`
  - `gemini/genai-sdk/`
  - `gemini/models/`
  - `gemini-official-20260310/`
- GCP:
  - `gcp/cloud-run/`
  - `gcp/firestore/`
  - `gcp/secret-manager/`
  - `gcp/logging-monitoring-trace/`
  - `gcp/cloud-build/`
  - `gcp/artifact-registry/`
- Samples:
  - `samples/`

## Source Mapping

- Canonical source-to-local mapping is maintained in:
  - `docs/PRD/DETAILS/SOURCE_REFERENCE_MAP.md`

## Notes

- Naming convention:
  - Directories: lowercase + kebab-case
  - Files: `NN_topic_name.md`
- Challenge pages were normalized from legacy folder naming to:
  - `challenge/gemini-live-agent-challenge/`
- Link integrity validation output is stored in:
  - `docs/reference/LINK_VALIDATION_REPORT.md`
