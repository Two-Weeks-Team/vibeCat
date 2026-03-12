# Docs Guide

## OVERVIEW

`docs/` mixes current submission-truth documents with historical analysis, planning, and reference material. When updating docs, prefer the files explicitly rewritten for the UI Navigator pivot.

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Current snapshot | `docs/CURRENT_STATUS_20260311.md` | best concise status doc |
| Runtime architecture | `docs/FINAL_ARCHITECTURE.md` | current navigator architecture |
| Deployment evidence | `docs/evidence/DEPLOYMENT_EVIDENCE.md` | proof-oriented deploy state |
| Submission proof checklist | `docs/deployment/PROOF_OF_GCP_DEPLOYMENT.md` | final evidence checklist |
| Current product plan | `docs/PRD/UI_NAVIGATOR_PRD.md` | current PRD |
| Historical design notes | `docs/analysis/` | useful, but often archival |
| Reference dumps | `docs/reference/` | source material, not project truth |

## CONVENTIONS

- Treat docs as current only if they were explicitly updated for the navigator pivot.
- Use `docs/evidence/` and `docs/deployment/` for proof and operations claims.
- Keep submission-facing wording aligned to `UI Navigator`, not the older companion framing.

## ANTI-PATTERNS

- citing `docs/analysis/` or old PRD files as proof of current implementation
- updating README/proof docs with companion-era language
- using reference snapshots as if they were maintained design docs
- duplicating the same operational facts across multiple proof docs without checking current status
