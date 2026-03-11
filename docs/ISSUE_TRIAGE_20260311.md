# Issue Triage Audit (2026-03-11)

This audit reduces the repository to issues that still represent meaningful unfinished work.

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

- #55 Validate copied assets and runtime path resolution
- #56 Run full operation scenarios from menu to speech output
- #57 Complete deployment and operations evidence pack
- #58 Final handoff gate
- #60 Implement sprite screen pointing
- #61 Implement session farewell and sleep sprite
- #64 Add privacy controls UI
- #68 Set up Cloud Build pipeline
- #90 Configure Cloud Logging and Monitoring
- #91 Configure Cloud Trace
- #117 End-to-end companion intelligence integration test
- #120 Implement graceful fallback for Gemini unavailability
- #121 Implement graceful fallback for ADK Orchestrator timeout

Rationale for the remaining open set:

- Asset validation, manual operations proof, and handoff evidence still need explicit artifact generation rather than code presence alone.
- Grand-prize UX items (`#60`, `#61`, `#64`) do not have matching implementation in the client.
- Cloud Build triggers are not configured yet even though deployment scripts exist.
- Logging/Monitoring/Trace exporters are wired, but dashboard-level and trace-verification acceptance still needs explicit completion.
- The full companion-intelligence integration test is broader than the current deployed search/memory/barge-in coverage.
- Graceful degraded-mode behavior for Gemini or ADK outages is not complete enough to satisfy the issue acceptance criteria.
