---
title: why i stopped building nine equal agents
published: false
description: VibeCat started with a big multi-agent companion graph, but the product got better when I split it into one Live PM and one single-task action worker
tags: geminiliveagentchallenge, devlog, buildinpublic, architecture
cover_image:
---

I created this post for the purposes of entering the Gemini Live Agent Challenge.

---

there was a phase where VibeCat looked impressive on paper.

it had a graph. it had specialist agents. it had names like mediator, scheduler, celebration trigger, engagement. the architecture diagram got denser and denser, and every new box felt like progress.

it wasn't.

the product got better when I stopped trying to make nine equal agents cooperate in real time and started treating the system like what it actually is:

- one agent that talks to the user
- one worker that does the next concrete thing
- one local executor that can actually click, focus, and type

that's the version that feels fast, legible, and trustworthy.

## the mistake

the early VibeCat idea was "AI colleague." if you follow that idea too literally, you end up decomposing personality into a graph:

- something that sees
- something that remembers
- something that senses frustration
- something that decides whether to speak
- something that celebrates
- something that schedules
- something that searches

that architecture is defensible for background analysis. it is not a good hot path for UI action.

every extra agent adds:

- another model call
- another serialization boundary
- another state handoff
- another place for latency to hide
- another place for product intent to blur

for a desktop UI navigator, those costs are brutal. if the user says "open the official docs" or "type this in the search box," they do not care that your internal graph is elegant. they care whether the system moves now, and whether it moves safely.

## the split that finally worked

the architecture that made VibeCat click is much simpler:

```text
User
  ↕ voice / chat
Live PM (Gemini Live + VAD)
  ↕ structured handoff
Single-Task Action Worker
  ↕ step execution
Local AX Executor on macOS
```

the **Live PM** is the only thing that talks to the user. it handles:

- natural language intent
- short clarification questions
- progress narration
- results and next-step explanations

the **action worker** does not try to be charming. it does not improvise. it handles:

- task creation
- risk checks
- one-step planning
- verification
- completion or fallback

the **local executor** is even narrower. it just:

- finds the target
- focuses it
- clicks it
- types into it
- reports what happened

that is the boundary that matters.

## one task at a time

the real architectural pivot was this rule:

**only one executable task can be active at a time.**

if the user asks for something else mid-flight, VibeCat does not silently fork the work. it asks:

"I am already working on that. Do you want me to stop and switch?"

that sounds small, but it changes everything:

- state gets simpler
- verification gets more reliable
- traces get easier to read
- failure handling gets less magical
- the product feels more honest

you stop building a swarm and start building a control plane.

## the role of specialists now

I didn't delete the idea of specialists. I demoted them.

that's the important move.

specialist intelligence still matters for:

- research
- memory
- replay labeling
- session summaries
- low-confidence multimodal cross-checks

but it should sit **behind** the action worker, not beside the PM in the user-facing hot path.

in other words:

- background specialists: yes
- equal peer agents arguing over the next click: no

## why this is better for a voice-first product

VibeCat uses Gemini Live + VAD as the main interaction layer. that means the conversation channel is always on, low-latency, and interruption-friendly.

once you already have that, the cleanest structure is not "many speaking agents." it's one persistent PM that always owns the conversation, and one worker that gets called when the PM has something concrete to do.

the PM says:

- "I can do that now."
- "Do you want me to apply it, or just explain it?"
- "I found the input field."

the worker decides:

- what the task is
- what the next step is
- whether the request is risky
- whether the target is safe enough

the executor performs the step.

the PM comes back with the result.

that's a product architecture, not just a model architecture.

## what changed in the codebase

the current direction is explicit:

- Gemini Live stays as the user-facing PM session
- the gateway runs a single-task navigator worker
- the macOS client runs a local action worker and AX-first executor
- task ids tie planning, execution, and verification together
- new action requests do not run in parallel

the biggest product win from this pivot was not theoretical elegance. it was that text entry, focused input targeting, and action replacement finally became debuggable.

## the lesson

I still think multi-agent systems are useful.

I just think they are easy to over-apply.

for VibeCat, the right answer was not "how do I add more agents?" it was "which single agent should the user actually trust, and which single worker should actually act?"

that question killed a lot of architectural vanity.

good.

---

*Building VibeCat for the [Gemini Live Agent Challenge](https://geminiliveagentchallenge.devpost.com/). Source: [github.com/Two-Weeks-Team/vibeCat](https://github.com/Two-Weeks-Team/vibeCat)*
