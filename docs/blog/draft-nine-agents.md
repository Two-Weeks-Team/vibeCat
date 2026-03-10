---
title: teaching nine agents to think like a colleague
published: false
description: how VibeCat decomposes "being a good colleague" into 9 ADK agents running in 3 parallel waves — and why the Mediator agent is the hardest one to build
tags: geminiliveagentchallenge, devlog, buildinpublic, go
cover_image:
---

I created this post for the purposes of entering the Gemini Live Agent Challenge. In my last post I walked through what VibeCat actually does — a macOS cat that watches your screen, hears your voice, and knows when to shut up. But I glossed over *how* it does all that. The cat isn't one thing — it's nine things pretending to be one thing, and getting that pretense right is the actual engineering problem.

---

let me start with the question that shaped everything: what does a colleague actually *do*?

not a chatbot. not a search engine. a colleague. the person sitting next to you who catches your typo on line 23 before you do, notices you've been stuck for 40 minutes, and knows when to shut up because you're in flow.

I spent a while listing the behaviors:
- **See** your screen and notice errors
- **Remember** yesterday's context
- **Sense** frustration from patterns
- **Celebrate** when tests pass
- **Decide** whether to speak or stay silent
- **Adapt** timing to your rhythm
- **Reach out** when you've been too quiet
- **Search** for answers when you're stuck

that's not one model doing one thing. that's eight distinct behaviors plus voice (VAD makes nine). so I decomposed the colleague into nine agents.

## the graph

all nine agents run through Google ADK's workflow agents. the key insight: not all agents need each other's results. VisionAgent doesn't care about MemoryAgent's output. MoodDetector doesn't need CelebrationTrigger. so I split them into three waves:

```go
// Wave 1 — Perception (parallel)
wave1, _ := parallelagent.New(parallelagent.Config{
    AgentConfig: agent.Config{
        Name:      "wave1_perception",
        SubAgents: []agent.Agent{visionAgent, memoryAgent},
    },
})

// Wave 2 — Emotion (parallel)
wave2, _ := parallelagent.New(parallelagent.Config{
    AgentConfig: agent.Config{
        Name:      "wave2_emotion",
        SubAgents: []agent.Agent{moodAgent, celebrationAgent},
    },
})

// Wave 3 — Decision (sequential, because each depends on the previous)
wave3, _ := sequentialagent.New(sequentialagent.Config{
    AgentConfig: agent.Config{
        Name:      "wave3_decision",
        SubAgents: []agent.Agent{mediatorAgent, schedulerAgent, engagementAgent, searchLoop},
    },
})

// The full graph
graph, _ := sequentialagent.New(sequentialagent.Config{
    AgentConfig: agent.Config{
        Name:      "vibecat_graph",
        SubAgents: []agent.Agent{wave1, wave2, wave3},
    },
})
```

waves 1 and 2 run in parallel — `parallelagent` fires both sub-agents simultaneously. wave 3 runs sequentially because the Mediator needs mood + celebration results, the Scheduler needs the Mediator's decision, and so on.

the result: ~35% latency reduction compared to running all 9 sequentially. from ~3.5 seconds down to ~2.1-2.5 seconds for the full graph. that matters when a developer is waiting for the cat to react to their screen.

## the mediator problem

making AI talk is easy. every LLM wants to talk. the hard part is making it know when to *shut up*.

the Mediator agent is the gatekeeper. it reads everything — vision analysis, mood state, celebration events — and makes one binary decision: speak or stay silent. here's the core logic:

```go
const (
    defaultCooldown  = 10 * time.Second
    moodCooldown     = 180 * time.Second
    highSignificance = 7
)

func (a *Agent) decide(vision *models.VisionAnalysis, mood *models.MoodState, celebration *models.CelebrationEvent) *models.MediatorDecision {
    // ... read from state, check cooldown, check flow state

    // celebration always bypasses cooldown
    if celebration != nil && celebration.Message != "" {
        return &models.MediatorDecision{ShouldSpeak: true, Reason: "celebration"}
    }

    // high significance + error = speak immediately
    if vision != nil && vision.Significance >= highSignificance && vision.ErrorDetected {
        return &models.MediatorDecision{ShouldSpeak: true, Reason: "error_detected", Urgency: "high"}
    }

    // flow state = extend cooldown, stay silent
    if isInFlowState(ctx) {
        return &models.MediatorDecision{ShouldSpeak: false, Reason: "flow_state"}
    }

    // ... more rules
}
```

but it gets more nuanced. the Mediator also tracks recent speech to avoid repeating itself:

```go
func (a *Agent) isSimilarToRecent(text string) bool {
    // if we said something similar in the last 5 utterances, stay silent
}
```

and it generates mood-support messages dynamically using `gemini-3.1-flash-lite-preview` when it detects sustained frustration but hasn't spoken about mood in the last 3 minutes:

```go
if mood != nil && !decision.ShouldSpeak {
    sinceMood := time.Since(a.lastMoodSpoke)
    if sinceMood > moodCooldown {
        msg := a.generateMoodMessage(ctx, mood, vision, language)
        if msg != "" {
            decision.ShouldSpeak = true
            decision.Reason = "mood_support"
            a.lastMoodSpoke = time.Now()
        }
    }
}
```

no hardcoded messages. every utterance is generated by LLM, considering the developer's current context, mood, language, and what they're working on. the hardcoded pool exists only as a fallback if LLM generation fails.

## multimodal mood detection

the MoodDetector doesn't just look at text. it fuses three signals:

1. **Vision signals** — error frequency, repeated errors (same error 3+ times = frustrated), app switches
2. **Voice tone** — from Gemini's AffectiveDialog, the Live API reports the emotional tone of the user's voice
3. **Temporal patterns** — how long since last interaction, silence duration, error-to-fix time

```go
voiceTone, voiceConfidence := readVoiceToneFromState(ctx)
mood := a.classify(vision, voiceTone, voiceConfidence)
```

the voice tone comes from ADK session state — the gateway extracts it from the Live API's AffectiveDialog output and writes it to `voice_tone` in the session state. the MoodDetector reads it alongside the vision analysis to produce a fused mood classification.

this is genuinely multimodal — not just "look at the screen" or "listen to the voice" but both, simultaneously, informing a single emotional model.

## rest reminders and proactive engagement

the EngagementAgent handles two kinds of proactive behavior:

**silence engagement** — if the developer hasn't interacted in 3 minutes, it speaks up:

```go
if sinceLast > silenceThreshold {
    result.Decision.ShouldSpeak = true
    result.Decision.Reason = "silence_engagement"
    result.SpeechText = a.generateSilenceMessage(ctx, language)
}
```

**rest reminders** — the client tracks `activityMinutes` from session start and sends it with every screen capture. after 50 minutes of continuous coding:

```go
const restReminderInterval = 50 * time.Minute
const restReminderCooldown = 30 * time.Minute

if activityMin >= int(restReminderInterval.Minutes()) && sinceLastReminder > restReminderCooldown {
    result.Decision.ShouldSpeak = true
    result.Decision.Reason = "rest_reminder"
    result.SpeechText = a.generateRestMessage(ctx, lang, activityMin)
}
```

the full pipeline: macOS client calculates minutes since session start → sends `activityMinutes` in the WebSocket payload → Gateway passes it to Orchestrator in `POST /analyze` → EngagementAgent reads it from session state → triggers LLM-generated rest suggestion in the developer's language.

## adk advanced features

VibeCat doesn't just use ADK's basic agents. it uses the advanced stuff:

**`retryandreflect` plugin** — if an agent fails (network timeout, LLM error), it automatically reflects on why it failed and retries:

```go
import "google.golang.org/adk/plugin/retryandreflect"

r, _ := runner.New(runner.Config{
    Agent:   graphAgent,
    Plugins: []runner.Plugin{retryandreflect.New(retryandreflect.WithTrackingScope(retryandreflect.Invocation))},
})
```

**`loopagent`** — the SearchBuddy is wrapped in a loop agent that runs up to 2 iterations, refining search results:

```go
searchLoop, _ := loopagent.New(loopagent.Config{
    AgentConfig: agent.Config{
        Name:      "search_refinement_loop",
        SubAgents: searchSubAgents,
    },
    MaxIterations: 2,
})
```

**`BeforeModel/AfterModel` callbacks** — the LLM search agent has callbacks for logging and guard-rails:

```go
llmSearchAgent, _ := llmagent.New(llmagent.Config{
    BeforeModelCallback: func(ctx agent.CallbackContext, req *model.LLMRequest) (*model.LLMResponse, error) {
        slog.Info("[LLM_SEARCH] before model", "agent", ctx.AgentName())
        return nil, nil
    },
    AfterModelCallback: func(ctx agent.CallbackContext, resp *model.LLMResponse, err error) (*model.LLMResponse, error) {
        slog.Info("[LLM_SEARCH] after model", "agent", ctx.AgentName(), "has_error", err != nil)
        return resp, nil
    },
})
```

14 ADK features total. `agent.New`, `sequentialagent`, `parallelagent`, `loopagent`, `llmagent`, `session.InMemoryService`, `memory.InMemoryService`, `runner.New`, `telemetry`, `session.State`, `functiontool`, `geminitool.GoogleSearch`, `retryandreflect`, and `BeforeModel/AfterModel` callbacks.

## what I learned

the hardest thing about building a multi-agent system isn't the graph. it's the boundaries. when does MoodDetector's responsibility end and Mediator's begin? who owns the "should I speak" decision when both EngagementAgent and Mediator have opinions?

the answer that worked: each agent writes to session state, and downstream agents read from it. no agent calls another agent directly. the graph topology IS the API contract. Vision writes `vision_analysis` to state. Mood reads it and writes `mood_state`. Mediator reads both. clean, testable, and you can swap any agent without touching the others.

nine agents. three waves. one decision. and a cat that knows when to shut up.

---

*Building VibeCat for the [Gemini Live Agent Challenge](https://geminiliveagentchallenge.devpost.com/). Source: [github.com/Two-Weeks-Team/vibeCat](https://github.com/Two-Weeks-Team/vibeCat)*
