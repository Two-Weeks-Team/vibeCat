# Issue Triage Audit (2026-03-11)

This audit reduces the repository to issues that still represent meaningful unfinished work.

## Final state after 2026-03-11 cleanup

- closed as implemented or no longer meaningful: `#55`, `#60`, `#61`, `#68`
- closed as consolidated into `#57`: `#56`, `#58`
- remaining open: `#57`, `#64`, `#90`, `#91`, `#117`, `#120`, `#121`

## Closed as implemented

- Client foundation and UX: #1-#6, #8-#20, #22-#31, #33, #38-#43, #45-#53, #59, #62-#63
- Backend foundation and deployment: #65-#67, #69-#89, #92, #94-#116, #118-#119, #122

Representative evidence:

- Core settings, keychain, PCM, image, audio parsing, and localization primitives exist under `VibeCat/Sources/Core/`.
- Status bar, tray animation, onboarding, capture, live transport, chat, hotkey, overlay, and app wiring exist under `VibeCat/Sources/VibeCat/`.
- End-to-end and deployment-facing tests exist under `tests/e2e/`.
- Deployment automation exists under `infra/setup.sh`, `infra/deploy.sh`, and `infra/teardown.sh`.
- Realtime Gateway and ADK Orchestrator are deployed in `asia-northeast3`.

Deployment verification performed on 2026-03-11:

- `gcloud run services describe realtime-gateway --region asia-northeast3 --project vibecat-489105` returned a healthy URL with ready condition `True`
- `gcloud run services describe adk-orchestrator --region asia-northeast3 --project vibecat-489105` returned a healthy URL with ready condition `True`
- `curl https://realtime-gateway-163070481841.asia-northeast3.run.app/health` returned HTTP `200`
- `gcloud secrets list --project vibecat-489105` includes `vibecat-gemini-api-key` and `vibecat-gateway-auth-secret`
- `gcloud firestore databases list --project vibecat-489105` shows the default Firestore database in `asia-northeast3`

## Closed as superseded by the current architecture

- #7 prompt composition primitives
- #21 API key onboarding and validation UX
- #32 speech wrapper and client TTS fallback
- #34-#37 client-side agent implementations
- #44 client REST fallback speech path
- #54 startup behavior keyed off client API key presence

Reason:

- Prompt construction, agent logic, search grounding, mood, celebration, memory, and fallback TTS decisions now belong on the backend.
- The client now uses gateway-issued session tokens instead of storing or validating Gemini API keys locally.
- Client responsibilities are UI, capture, transport, playback, and local settings only.

## Closed as consolidated

- #93 deployment evidence pack

Reason:

- This is already subsumed by #57, which is the higher-level ops evidence task used by the final handoff path.

## Kept open

- #57 Complete deployment and operations evidence pack
- #64 Add privacy controls UI
- #90 Configure Cloud Logging and Monitoring
- #91 Configure Cloud Trace
- #117 End-to-end companion intelligence integration test
- #120 Implement graceful fallback for Gemini unavailability
- #121 Implement graceful fallback for ADK Orchestrator timeout

Rationale for the remaining open set:

- `#57`: evidence docs still need submission-grade artifact completion and final cross-linking.
- `#64`: pause/mute/reconnect exist, but always-visible capture indicator, manual-only analyze mode, and explicit “no screenshots stored” UI are still missing.
- `#90`: logging and metric exporters exist, but no Monitoring dashboard is configured.
- `#91`: traces exist, but end-to-end Gateway-to-Orchestrator trace propagation is not yet acceptance-proven.
- `#117`: current E2E coverage includes health/auth/search-memory/barge-in, but not the full companion session story.
- `#120`: Gemini reconnect behavior exists, but the TTS fallback path is not wired into Gemini-unavailable handling.
- `#121`: proactive analyze still uses a longer timeout path and does not yet return a fast silent fallback within the issue acceptance criteria.

## Closed in this cleanup

- `#55`: asset counts and runtime loaders are already verifiable from the current repo and code.
- `#56`, `#58`: these are better tracked as part of `#57` rather than as separate operational checklist issues.
- `#60`, `#61`: these are optional UX/stretch items, not meaningful blockers for the current shipped/deployed baseline.
- `#68`: Cloud Build YAML exists, but the active deployment path is GitHub Actions manual deploy plus Cloud Run; separate GCP triggers are not currently necessary.
