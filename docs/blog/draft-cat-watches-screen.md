---
title: the cat that watches your screen
published: false
description: building vibecat for the gemini live agent challenge — a macOS AI companion that sees your code, hears your voice, and knows when to shut up
tags: geminiliveagentchallenge, devlog, buildinpublic, go
cover_image:
---

I created this post for the purposes of entering the Gemini Live Agent Challenge, but this one's different from the earlier posts. Those were about missless dying and VibeCat being born. This one is about what VibeCat actually *is* now that it's built.

---

in my last post I split "being a good colleague" into 9 Go agents. but that post was about *what* each agent does — not *how* the whole system actually runs. this post is the how. the three-layer architecture, the Live API config, the 3-wave parallel execution, and the Mediator problem — making AI know when to shut up.

let me tell you what VibeCat actually does.

it's a macOS desktop companion that sits on your screen — an animated sprite in the corner. it watches your screen via ScreenCaptureKit, hears your voice through the microphone, remembers what you were working on yesterday, and speaks up when something matters. not when you ask it to. when *it* decides to.

that last part is the hard part.

## the three-layer split

the Gemini Live Agent Challenge requires four things: GenAI SDK, Google ADK, Gemini Live API, and VAD. all of them. and the client can never talk to Gemini directly — everything goes through a backend.

so VibeCat is three layers:

```
macOS Client (Swift 6 / SwiftUI)
    ↕ WebSocket (wss://)
Realtime Gateway (Go + google.golang.org/genai v1.48.0)
    ↕ HTTP POST /analyze
ADK Orchestrator (Go + google.golang.org/adk v0.5.0)
```

the client does UI, screen capture, audio playback, and sprite animation. it never touches Gemini. the gateway proxies WebSocket audio to Gemini's Live API and forwards screen captures to the orchestrator. the orchestrator runs a 9-agent graph and returns decisions.

both backend services run on Cloud Run in `asia-northeast3`. the API key lives in Secret Manager. the client doesn't even know the key exists — it authenticates with a device UUID, and the gateway handles everything else.

## the live API setup

the voice conversation runs on `gemini-2.5-flash-native-audio-latest` through the GenAI SDK's Live API:

```go
session, err := m.client.Live.Connect(ctx, model, &genai.LiveConnectConfig{
    ResponseModalities: []genai.Modality{genai.ModalityAudio},
    SpeechConfig: &genai.SpeechConfig{
        VoiceConfig: &genai.VoiceConfig{
            PrebuiltVoiceConfig: &genai.PrebuiltVoiceConfig{
                VoiceName: cfg.Voice,
            },
        },
    },
})
```

but the real magic is in the config flags. we enable *everything*:

- **VAD** with `automaticActivityDetection` — 300ms prefix padding, 500ms silence duration. low sensitivity so it doesn't cut you off mid-sentence.
- **Barge-in** with `StartOfActivityInterrupts` — interrupt the cat mid-sentence, it stops and listens.
- **AffectiveDialog** — Gemini reads your tone. frustrated? it adjusts. excited? it matches.
- **ProactiveAudio** — the AI can speak without being prompted. this is what makes VibeCat a companion, not a chatbot.
- **SessionResumption** — resume sessions with a handle instead of re-establishing context.
- **ContextWindowCompression** — trigger at 4096 tokens, target 2048. long sessions don't run out of context.
- **Output/Input transcription** — we get text versions of both sides of the conversation, which feeds back into the agent graph.

eight Live API features in one connection. every one of them matters.

## the 9-agent graph

a chatbot answers questions. VibeCat has opinions. the difference is nine agents running in three parallel waves:

**Wave 1 — Perception (parallel):**
- `VisionAgent` analyzes your screen capture for errors, patterns, and context
- `MemoryAgent` retrieves yesterday's unresolved issues from Firestore

**Wave 2 — Emotion (parallel):**
- `MoodDetector` classifies your mood from vision signals + voice tone (multimodal fusion)
- `CelebrationTrigger` checks if your tests just passed or a deploy succeeded

**Wave 3 — Decision (sequential):**
- `Mediator` decides whether to speak — this is the hardest agent
- `AdaptiveScheduler` adjusts cooldown based on interaction frequency
- `EngagementAgent` handles proactive outreach: rest reminders after 50 minutes, silence-breaking after extended quiet
- `SearchBuddy` (wrapped in a `loopagent` for iterative refinement) searches Google when you're stuck

the graph is built with ADK's `parallelagent` and `sequentialagent`:

```go
graph, _ := sequentialagent.New(sequentialagent.Config{
    AgentConfig: agent.Config{
        Name:      "vibecat_graph",
        SubAgents: []agent.Agent{wave1, wave2, wave3},
    },
})
```

waves 1 and 2 run in parallel because they're independent — vision doesn't need memory, mood doesn't need celebration. wave 3 runs sequentially because each decision depends on the previous one. the mediator needs mood and celebration results. the scheduler needs the mediator's decision. engagement needs the scheduler's cooldown.

## the mediator problem

making AI talk is easy. making AI know when to *shut up* is the real engineering challenge.

the Mediator agent is the gatekeeper. it looks at everything — urgency score, mood state, celebration flags, time since last speech, developer's flow state — and decides: speak or stay silent.

if you're in flow state (long continuous coding, no errors, no frustration), the mediator extends the cooldown. it won't interrupt you. if it detects frustration and an error simultaneously, it speaks immediately — urgency override. if a celebration fires (tests passed), it always speaks — celebration bypass.

this is what makes VibeCat feel like a colleague instead of a notification system.

## what's next

I've written separate deep-dives for the pieces that deserve their own posts:
- the 9-agent architecture and ADK advanced features (retryandreflect, loopagent, BeforeModel/AfterModel callbacks)
- the 6 characters and how one soul.md file creates completely different personalities
- the Cloud Run deployment with full GCP observability (Cloud Trace, Cloud Logging, Cloud Monitoring)

the chair is no longer empty. now I need to film it.

---

*Building VibeCat for the [Gemini Live Agent Challenge](https://geminiliveagentchallenge.devpost.com/). Source: [github.com/Two-Weeks-Team/vibeCat](https://github.com/Two-Weeks-Team/vibeCat)*
