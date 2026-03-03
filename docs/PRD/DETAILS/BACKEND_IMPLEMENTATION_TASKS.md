# Backend Implementation Tasks

## Stack Decision

- **Language**: Go (Golang)
- **Realtime Gateway**: Go + `google.golang.org/genai` (GenAI SDK — Live API WebSocket)
- **ADK Orchestrator**: Go + `google.golang.org/adk` (ADK Go SDK v0.5.0)
- **GCP Project**: `vibecat-489105` (region: `asia-northeast3`)
- **Account**: `centisgood@gmail.com`
- **Gateway-Orchestrator Auth**: Cloud Run service-to-service IAM (identity tokens, no shared secrets)

## Go Module Dependencies

### Realtime Gateway (`backend/realtime-gateway/`)
```
google.golang.org/genai
github.com/gorilla/websocket
cloud.google.com/go/secretmanager
cloud.google.com/go/firestore
```

### ADK Orchestrator (`backend/adk-orchestrator/`)
```
google.golang.org/adk
google.golang.org/genai
cloud.google.com/go/firestore
```

## Usage Rule

- Execute tasks in ID order unless parallel execution is noted.
- A task is complete only if its verification checks pass.
- Backend tasks can run in parallel with client tasks where dependency allows.

## Phase B0 — Backend Bootstrap

### T-100 Initialize backend repository structure
- Goal: create directory structure for two Cloud Run services.
- Steps: create `backend/realtime-gateway/`, `backend/adk-orchestrator/`, `infra/`. Add Go project configs (go.mod, go.sum) for each service.
- Verification: project structure exists with dependency files.
- Depends on: none.

### T-101 Configure Secret Manager
- Goal: store Gemini API key securely in GCP.
- Ref: `docs/reference/gcp/secret-manager/01_secret_manager_overview.md`
- Steps: create secret `vibecat-gemini-api-key`, set IAM bindings for Cloud Run service accounts.
- Verification: secret accessible from Cloud Run service identity.
- Depends on: T-100.

### T-102 Configure Firestore
- Goal: set up session and metric persistence.
- Ref: `docs/reference/gcp/firestore/01_firestore_overview.md`
- Steps: create Firestore DB in asia-northeast3, define security rules, create collections (sessions, metrics, history).
- Verification: read/write succeeds from service account.
- Depends on: T-100.

### T-103 Set up Cloud Build pipeline
- Goal: automated build and deploy for both services.
- Ref: `docs/reference/gcp/cloud-build/01_cloud_build_overview.md`
- Steps: create cloudbuild.yaml per service, configure Artifact Registry, set up build triggers.
- Verification: push triggers build, image appears in registry.
- Depends on: T-100.

## Phase B1 — Realtime Gateway

### T-110 Implement WebSocket server
- Goal: accept client WebSocket connections on Cloud Run.
- Ref: `docs/reference/gemini/genai-sdk/01_sdk_overview.md`
- Steps: create WebSocket server (port 8080), handle open/message/close/error, add /healthz /readyz.
- Verification: client connects, sends message, receives echo.
- Depends on: T-100.

### T-111 Implement client authentication
- Goal: validate client sessions before WebSocket upgrade.
- Steps: token validation middleware, POST /api/v1/auth/register (validate key → Secret Manager → return token), POST /api/v1/auth/refresh.
- Verification: invalid tokens → 401, valid tokens → connection.
- Depends on: T-101, T-110.

### T-112 Implement GenAI SDK Live API session
- Goal: create Gemini Live session when client connects.
- Ref: `docs/reference/gemini/live-api/01_get_started_live_api.md`
- Steps: init GenAI client, create live session with setup config (voice, language, tools, VAD, resumption, compression, proactiveAudio, affectiveDialog, outputAudioTranscription).
- Verification: Gemini session established, setup-complete received.
- Depends on: T-101, T-110.

### T-113 Implement audio stream proxy
- Goal: bidirectional audio proxy between client and Gemini.
- Ref: `docs/reference/gemini/live-api/02_live_capabilities_guide.md`
- Steps: forward client PCM (16kHz 16-bit) → Gemini, forward Gemini audio (24kHz) → client. Handle interruptions, turn completion.
- Verification: send audio → receive audio response within target latency.
- Depends on: T-112.

### T-114 Implement VAD configuration
- Goal: apply Voice Activity Detection to Gemini session.
- Ref: `docs/PRD/DETAILS/IMPLEMENTATION_REQUIREMENTS.md`
- Steps: set automaticActivityDetection: disabled=false, startSensitivity=LOW, endSensitivity=LOW, prefixPadding=20ms, silenceDuration=100ms.
- Verification: VAD detects speech start/end events.
- Depends on: T-112.

### T-115 Implement session resumption proxy
- Goal: handle Gemini session resumption transparently.
- Ref: `docs/reference/gemini/live-api/04_live_session_management.md`
- Steps: store resumption handles, use on client reconnect, handle GoAway messages.
- Verification: reconnect preserves conversation context.
- Depends on: T-112.

### T-116 Implement keepalive stack
- Goal: stable connections with zombie detection.
- Steps: protocol ping/pong (15s), heartbeat to Gemini (30s), respond to client pings, zombie detection (45s timeout).
- Verification: stale connections detected and cleaned up.
- Depends on: T-112.

### T-117 Implement ADK Orchestrator routing
- Goal: route screen captures to ADK for analysis.
- Steps: on screenCapture/forceCapture, POST to ADK Orchestrator, receive analysis result, use result to determine speech behavior.
- Verification: capture triggers analysis, structured result returned.
- Depends on: T-110, T-130.

### T-118 Implement transcription forwarding
- Goal: forward Gemini transcription to client for bubble display.
- Ref: `docs/reference/gemini/live-api/02_live_capabilities_guide.md`
- Steps: parse transcription events, forward with finished flag, handle turn completion.
- Verification: client receives sentence-level transcription correctly.
- Depends on: T-113.

### T-119 Implement settings update handling
- Goal: apply runtime settings from client.
- Steps: on settingsUpdate, apply changes requiring Gemini reconnect (voice, model, language) vs. pass-through to ADK (chattiness).
- Verification: voice change → new Gemini session with updated voice.
- Depends on: T-112.

## Phase B2 — ADK Orchestrator

### T-130 Initialize ADK project
- Goal: set up ADK Go project with agent definitions.
- Ref: ADK Go SDK (`google.golang.org/adk`), samples at `github.com/google/adk-samples/tree/main/go`
- Steps: add `google.golang.org/adk` to go.mod, create agent module structure, define HTTP endpoint for analysis requests.
- Verification: ADK server starts, responds to health check.
- Depends on: T-100.

### T-131 Implement VisionAgent (ADK)
- Goal: analyze screen captures via Gemini REST.
- Steps: ADK agent takes image+context, calls Gemini REST (gemini-3-flash) with responseSchema, returns VisionAnalysis {significance, content, emotion, shouldSpeak}.
- Verification: valid structured analysis for test screenshots.
- Depends on: T-130.

### T-132 Implement Mediator (ADK)
- Goal: gate speech by significance, cooldown, duplication.
- Steps: ADK agent receives VisionAnalysis, applies cooldown, significance threshold, duplicate detection. Returns MediatorDecision {shouldSpeak, reason, urgency}.
- Verification: decision-table tests cover all branches.
- Depends on: T-131.

### T-133 Implement AdaptiveScheduler (ADK)
- Goal: adjust timing from interaction metrics.
- Steps: ADK agent reads/writes Firestore metrics, tracks utterances/responses/interruptions, adjusts silenceThreshold (10-45s) and cooldownSeconds (5-20s).
- Verification: simulated profiles produce bounded values.
- Depends on: T-102, T-132.

### T-134 Implement EngagementAgent (ADK)
- Goal: proactive prompts after silence threshold.
- Steps: ADK agent monitors silence duration (Firestore), composes proactive prompt with context, suppresses during active turn.
- Verification: trigger fires only after threshold, suppressed during active turns.
- Depends on: T-133.

### T-135 Define ADK agent graph
- Goal: connect all agents into executable graph.
- Ref: `docs/reference/adk/streaming/01_adk_streaming_overview.md`
- Steps: graph: screenCapture → VisionAgent → Mediator → routing (speak/silent) → return. EngagementAgent runs parallel timer-based.
- Verification: full graph executes E2E with correct routing.
- Depends on: T-131, T-132, T-133, T-134.

### T-136 Implement Google Search grounding tool
- Goal: enable real-time web info in responses.
- Ref: `docs/reference/gemini/live-api/03_live_tool_use.md`
- Steps: define Google Search as ADK tool, configure in Gemini session, handle tool responses.
- Verification: queries needing web info receive grounded responses.
- Depends on: T-135.

### T-137 Implement prompt management
- Goal: centralize prompt templates for all agents.
- Steps: VisionAgent system prompt, cat personality prompt, engagement directive, fallback personality. Language-aware output.
- Verification: prompts include required directives.
- Depends on: T-130.

## Phase B3 — Deployment and Integration

### T-140 Containerize Realtime Gateway
- Steps: multi-stage Dockerfile, port 8080, health check, WebSocket support.
- Verification: docker build → container starts → health check passes.
- Depends on: T-119.

### T-141 Containerize ADK Orchestrator
- Steps: multi-stage Dockerfile, port 8080, google-adk dependencies.
- Verification: docker build → container starts → health check passes.
- Depends on: T-136.

### T-142 Deploy to Cloud Run
- Ref: `docs/reference/gcp/cloud-run/02_deploying.md`
- Steps: configure services (min 0, max 10), set env vars, bind Secret Manager, service-to-service auth.
- Verification: both services running, health endpoints → 200.
- Depends on: T-140, T-141.

### T-143 Configure Cloud Logging and Monitoring
- Ref: `docs/reference/gcp/logging-monitoring-trace/01_cloud_logging_overview.md`
- Steps: structured JSON logging, create dashboard (connections, latency, errors).
- Verification: logs in Cloud Logging, metrics in Monitoring dashboard.
- Depends on: T-142.

### T-144 Configure Cloud Trace
- Ref: `docs/reference/gcp/logging-monitoring-trace/03_cloud_trace_overview.md`
- Steps: trace context propagation Gateway↔Orchestrator, instrument key operations.
- Verification: traces showing client→Gateway→Orchestrator→Gemini spans.
- Depends on: T-142.

### T-145 End-to-end integration test
- Steps: client connects → sends capture → receives analysis → audio plays. Test: capture cycle, voice chat, gesture, interruption, reconnection.
- Verification: all flows produce expected results.
- Depends on: T-142, client T-091.

### T-146 Deployment evidence pack
- Ref: `docs/PRD/DETAILS/SUBMISSION_AND_DEMO_PLAN.md`
- Steps: screenshots of Cloud Run, Firestore, Logging, Monitoring, Trace.
- Verification: all evidence artifacts exist and documented.
- Depends on: T-143, T-144, T-145.

## Dependency Summary

```
T-100 → T-101, T-102, T-103
T-100 → T-110 → T-111, T-112 → T-113~T-119
T-100 → T-130 → T-131 → T-132 → T-133 → T-134 → T-135 → T-136
T-117 depends on T-130 (ADK must be ready)
T-145 depends on T-142 (backend) + client T-091 (client ready)
```

## Phase B4 — Companion Intelligence Agents

### T-147 Define Firestore schema for cross-session memory
- Goal: create `users/{userId}/memory` collection with recentSummaries, knownTopics, codingPatterns fields.
- Ref: `docs/reference/gcp/firestore/01_firestore_overview.md`
- Steps: define collection structure, set Firestore security rules for authenticated service access, create indexes for topic queries.
- Verification: Firestore emulator accepts read/write to memory collection with expected field types.
- Depends on: T-101.

### T-148 Implement end-of-session summary generation
- Goal: when WebSocket closes, generate a concise session summary and store it in Firestore.
- Steps: on session end event, collect conversation highlights and VisionAgent key findings, call Gemini to generate summary with unresolved issues, write to `users/{userId}/memory.recentSummaries`, cap at last 10 summaries.
- Verification: after session close, Firestore contains new summary with date, text, and unresolved issues list.
- Depends on: T-147, T-113.

### T-149 Implement session-start memory retrieval
- Goal: on new session, load recent memory from Firestore and identify unresolved issues.
- Steps: on session open, read `users/{userId}/memory`, select most recent summary and unresolved topics, format as context bridge text.
- Verification: new session receives memory context within 500ms of connection.
- Depends on: T-147.

### T-150 Implement MemoryAgent in ADK graph
- Goal: ADK agent that reads memory context and injects it into conversation.
- Ref: `docs/reference/adk/streaming/01_adk_streaming_overview.md`
- Steps: define MemoryAgent in ADK agent graph, on session start inject context bridge into system instruction ("어제 그 API 이슈, 오늘은 괜찮아?"), register memory tool for topic lookup during conversation.
- Verification: new session with prior memory produces context bridge message. Session without prior memory starts clean.
- Depends on: T-149, T-130.

### T-151 Implement topic detection and tracking
- Goal: detect significant topics from conversation and VisionAgent output, update knownTopics in real-time.
- Steps: monitor conversation transcripts and VisionAgent analysis for recurring themes (error types, file names, API names), update `knownTopics` in Firestore with topic, lastMentioned, resolved status.
- Verification: after discussing "authentication" 3+ times, topic appears in knownTopics. After user says "solved", topic marked resolved.
- Depends on: T-150, T-147.

### T-152 Add memory context to Gateway session initialization
- Goal: pass memory summary to Gemini system instruction at session start.
- Steps: Gateway requests memory from ADK Orchestrator at session open, injects memory summary into Gemini Live API systemInstruction field, handles MEMORY_UNAVAILABLE gracefully (proceed without context).
- Verification: Gemini session system instruction contains memory context. Without memory service, session starts normally.
- Depends on: T-150, T-112.

### T-153 Define mood classification schema
- Goal: establish mood categories, detection signals, and scoring thresholds.
- Steps: define 4 mood states (focused, frustrated, stuck, idle) with signal weights: repeated error screen (0.3), same file revisited 5+ times (0.2), long silence after error (0.25), interaction rate drop (0.25). Define confidence threshold (0.7) for mood change.
- Verification: schema document reviewed, signal weights sum to 1.0 per mood state.
- Depends on: T-130.

### T-154 Implement frustration signal detection in VisionAgent
- Goal: extend VisionAgent structured output to include error repetition and success detection.
- Steps: add fields to VisionAgent response schema: `errorDetected` (bool), `repeatedError` (bool, same error seen 3+ times), `successDetected` (bool, test pass/build success), `errorMessage` (string, extracted error text).
- Verification: VisionAgent returns repeatedError=true when same error screen appears 3+ times. Returns successDetected=true for build success screen.
- Depends on: T-131.

### T-155 Implement mood scoring in ADK Orchestrator
- Goal: combine vision signals, silence duration, and interaction rate into mood classification.
- Ref: `docs/reference/adk/components/01_agents.md`
- Steps: create MoodDetector agent in ADK graph, input: VisionAgent signals + silence duration from EngagementAgent + interaction rate from AdaptiveScheduler, output: MoodState {mood, confidence, signals, suggestedAction}, update `sessions/{sessionId}/metrics.currentMood`.
- Verification: repeated error + long silence → mood=frustrated with confidence > 0.7. Steady coding + no errors → mood=focused.
- Depends on: T-153, T-154, T-134.

### T-156 Integrate mood into Mediator decision flow
- Goal: mood-aware speech gating in Mediator.
- Steps: Mediator reads MoodState from MoodDetector, frustrated → lower speak threshold (speak more supportive messages), focused → raise threshold (disturb less), stuck → trigger SearchBuddy auto-search, define supportive message templates per mood state.
- Verification: Mediator speaks supportive message when mood=frustrated. Mediator stays quiet when mood=focused and significance < 8.
- Depends on: T-155, T-132.

### T-157 Add mood field to session metrics in Firestore
- Goal: track mood changes over session lifetime.
- Steps: add currentMood, moodConfidence fields to `sessions/{sessionId}/metrics`, log mood changes to `sessions/{sessionId}/history` with type=mood_change.
- Verification: Firestore metrics show mood transitions. History log contains mood_change events.
- Depends on: T-155, T-101.

### T-158 Define supportive message templates per mood state
- Goal: create natural, non-annoying supportive messages for each mood.
- Steps: define message pools per mood: frustrated (5+ variants: "같이 한번 볼까?", "힘들어 보이는데, 도와줄까?", ...), stuck (5+ variants: "잠깐 쉬어볼래?", "한 발짝 물러서서 보면 보일 수도 있어", ...). Ensure variety to avoid repetition. Include Korean and English variants.
- Verification: message pool has 5+ unique variants per mood. No duplicate messages within 10 consecutive uses.
- Depends on: T-153.

### T-159 Define celebration detection patterns in VisionAgent
- Goal: VisionAgent recognizes success screens.
- Steps: add detection patterns for: "tests passed" / "All tests passed" / green checkmarks, "Build succeeded" / "Build successful", "Deployed" / "Deploy complete", "PR merged" / "Pull request merged", green CI badges.
- Verification: VisionAgent returns successDetected=true for each pattern. Returns false for unrelated screens.
- Depends on: T-131.

### T-160 Implement CelebrationTrigger agent in ADK graph
- Goal: agent that receives VisionAgent success signals and fires celebration events.
- Steps: create CelebrationTrigger agent, input: VisionAgent.successDetected, check cooldown (last celebration > 5 minutes ago), output: celebration event with trigger type, emotion=happy, celebratory message.
- Verification: celebration fires on test pass. Second test pass within 5 minutes does not fire. Test pass after 5+ minutes fires again.
- Depends on: T-159, T-130.

### T-161 Add celebration cooldown logic
- Goal: prevent celebration spam with per-session cooldown tracking.
- Steps: track celebrationCount and lastCelebrationAt in `sessions/{sessionId}/metrics`, enforce minimum 5-minute gap between celebrations, reset count on session end.
- Verification: rapid success events produce only 1 celebration per 5 minutes.
- Depends on: T-160, T-101.

### T-162 Integrate celebration into Mediator
- Goal: celebration events bypass normal significance gating.
- Steps: Mediator receives celebration event from CelebrationTrigger, bypasses significance threshold check, directly routes to Gateway, triggers happy sprite state on client.
- Verification: celebration is spoken even when Mediator would normally suppress (low significance window).
- Depends on: T-160, T-132.

### T-163 Add celebration event to client protocol
- Goal: client receives and renders celebration events.
- Steps: define celebration WebSocket message type (trigger, emotion, message, spriteState), client transitions to happy sprite, plays celebration voice, returns to idle after animation.
- Verification: client receives celebration message, displays happy sprite, plays voice.
- Depends on: T-162.

### T-164 Configure Google Search grounding tool in ADK Orchestrator
- Goal: enable Google Search tool in ADK for SearchBuddy.
- Ref: `docs/reference/gemini/live-api/03_live_tool_use.md`
- Steps: register Google Search grounding tool in ADK agent graph, configure search parameters (language, region, result count), test search tool returns results.
- Verification: ADK agent can call Google Search and receive results.
- Depends on: T-130.

### T-165 Implement search trigger detection
- Goal: detect when user wants help searching.
- Steps: monitor conversation for trigger phrases: "이거 뭐야", "왜 안돼", "어떻게 해", "이 에러 뭐지", explicit search requests. Also accept searchRequest WebSocket messages from client.
- Verification: trigger phrases activate SearchBuddy. Non-trigger phrases do not.
- Depends on: T-164.

### T-166 Implement auto-search trigger from MoodDetector
- Goal: automatically search when developer is stuck on same error for 3+ minutes.
- Steps: MoodDetector tracks time spent on same error (using VisionAgent.errorMessage), after 3 minutes stuck on same error → auto-trigger SearchBuddy with error message as query, only auto-trigger once per unique error.
- Verification: 3 minutes on same error → search auto-fires. Different error resets timer.
- Depends on: T-155, T-164.

### T-167 Implement search result summarization
- Goal: convert raw search results into voice-friendly summary.
- Steps: take Google Search results, use Gemini to summarize into 2-3 sentence voice-friendly response, format: "찾아봤는데, [source]에서 [summary]", include source attribution.
- Verification: search results summarized into < 100 words. Source mentioned. Response is natural spoken language.
- Depends on: T-164.

### T-168 Add search result caching in Firestore
- Goal: cache recent searches to avoid duplicate queries.
- Steps: store searches in `sessions/{sessionId}/searches` with query, summary, sources, triggeredBy, check cache before executing new search (same query within 10 minutes → return cached).
- Verification: duplicate query within 10 minutes returns cached result. New query after 10 minutes executes fresh search.
- Depends on: T-167, T-101.

### T-169 Integrate SearchBuddy response into Gateway voice output
- Goal: search results are spoken to user through Gateway.
- Steps: SearchBuddy sends searchResult to Gateway, Gateway injects result into Gemini Live API as tool response, Gemini speaks summarized result in current voice/persona.
- Verification: search result is spoken in configured voice. Client receives searchResult message with sources.
- Depends on: T-167, T-113.

### T-171 Enable affectiveDialog in Gemini Live API session config
- Goal: activate emotionally expressive voice output so the agent sounds worried, celebratory, or calm depending on context.
- Ref: `docs/reference/gemini/live-api/02_live_capabilities_guide.md`
- Steps: add `affectiveDialog: {enabled: true}` to Gemini Live API setup config in Gateway. Verify voice tone changes based on conversation context.
- Verification: agent voice sounds noticeably different when delivering supportive vs celebratory messages.
- Depends on: T-112.

### T-172 Enable proactiveAudio in Gemini Live API session config
- Goal: allow the agent to speak first without waiting for user input.
- Ref: `docs/reference/gemini/live-api/02_live_capabilities_guide.md`
- Steps: add `proactiveAudio: {enabled: true}` to Gemini Live API setup config in Gateway. This enables EngagementAgent and MoodDetector to initiate conversation.
- Verification: agent speaks proactively after silence period without user prompt.
- Depends on: T-112.

### T-173 Implement graceful fallback for Gemini unavailability
- Goal: when Gemini Live API is down, fall back to local TTS so the agent can still speak.
- Steps: detect Gemini connection failure or timeout, switch to local TTS endpoint (gemini-2.5-flash-preview-tts via REST), queue pending messages and deliver via TTS fallback, show reconnecting indicator to client, auto-resume Gemini Live session when available.
- Verification: kill Gemini connection → agent speaks via TTS fallback within 3s. Gemini reconnects → seamless switch back.
- Depends on: T-112, T-113.

### T-174 Implement graceful fallback for ADK Orchestrator timeout
- Goal: when ADK Orchestrator is slow or unresponsive, return safe default response.
- Steps: set 5s timeout on Gateway→ADK HTTP call, on timeout return default decision (shouldSpeak: false, silent path), log timeout event, do not block audio pipeline.
- Verification: ADK response delayed 10s → Gateway returns silent decision within 5s. Audio pipeline continues unblocked.
- Depends on: T-117.

### T-175 Create infra/ automated deployment scripts
- Goal: automated Cloud Run deployment for both services using gcloud CLI or Terraform.
- Ref: `docs/reference/gcp/cloud-run/02_deploying.md`
- Steps: create `infra/deploy.sh` with: (1) build Docker images via Cloud Build, (2) push to Artifact Registry, (3) deploy realtime-gateway to Cloud Run, (4) deploy adk-orchestrator to Cloud Run, (5) configure Secret Manager access, (6) set environment variables. Add `infra/teardown.sh` for cleanup. Add `infra/README.md` with usage instructions.
- Verification: `./infra/deploy.sh` deploys both services to Cloud Run. `./infra/teardown.sh` removes them. Scripts are idempotent.
- Depends on: T-140, T-141.

### T-170 End-to-end companion intelligence integration test
- Goal: verify all 4 new agents work together in a realistic session flow.
- Steps: simulate full session: (1) session start → memory loads → context bridge spoken, (2) coding → error appears → VisionAgent detects → mood shifts to frustrated, (3) 3 minutes stuck → auto-search fires → result spoken, (4) fix applied → tests pass → celebration fires, (5) session end → summary saved to memory. Test each agent individually and in combination.
- Verification: all 5 steps complete without error. Memory persists across session restart. Mood transitions follow expected sequence.
- Depends on: T-152, T-156, T-162, T-169.

## Dependency Summary

```
T-100 → T-101, T-102, T-103
T-100 → T-110 → T-111, T-112 → T-113~T-119
T-100 → T-130 → T-131 → T-132 → T-133 → T-134 → T-135 → T-136
T-117 depends on T-130 (ADK must be ready)
T-145 depends on T-142 (backend) + client T-091 (client ready)

Phase B4 (Companion Intelligence):
T-147 → T-148, T-149 → T-150 → T-151, T-152
T-153 → T-154, T-155 → T-156, T-157, T-158
T-159 → T-160 → T-161, T-162, T-163
T-164 → T-165, T-166, T-167 → T-168, T-169
T-170 depends on T-152, T-156, T-162, T-169 (full integration)
```

## Cross-Dependencies with Client Tasks

- T-117 (Gateway→ADK routing) needs T-130 (ADK initialized)
- T-145 (E2E integration) needs T-142 (backend deployed) + T-091 (client operational)
- T-163 (celebration client protocol) needs client sprite state handling
- T-170 (E2E companion test) needs client to support new message types (memoryContext, moodUpdate, celebration, searchResult)
- Client T-040 (live websocket) must be modified to connect to Gateway instead of direct Gemini
