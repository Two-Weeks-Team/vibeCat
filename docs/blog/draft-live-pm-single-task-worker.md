---
title: the architecture pivot that made vibecat credible
published: false
description: after reviewing Google Cloud's agentic AI guidance, I stopped treating VibeCat like a swarm and rebuilt it as one Live PM plus one single-task action worker
tags: geminiliveagentchallenge, architecture, googlecloud, buildinpublic
cover_image:
---

I created this post for the purposes of entering the Gemini Live Agent Challenge.

---

I had a vague feeling that VibeCat was getting harder to reason about, even when the code was technically improving.

the issue wasn't capability. it was shape.

every time I described the system, I reached for phrases like "graph," "agents," "parallel waves," and "specialists." those were not wrong. they just were not the most important truth anymore.

the more relevant truth was simpler:

- there is one always-on voice agent talking to the user
- there is one action worker deciding the next executable step
- there is one local executor that can actually operate the UI

once I looked at the system that way, the architecture stopped fighting the product.

## what changed my mind

I reviewed Google Cloud's published guidance on:

- choosing agentic AI components
- choosing design patterns
- single-agent systems on ADK + Cloud Run
- multi-agent systems
- interactive learning
- multimodal classification

the strongest takeaway was not "use more agents."

it was:

- start simple
- keep the user-facing plane clear
- separate execution from conversation
- put humans in the loop for risky actions
- externalize state
- only use parallel specialists when there is a real accuracy reason

that fits VibeCat almost perfectly.

## the new shape

the current architecture is easiest to explain in three planes.

**1. Live PM**

Gemini Live + VAD stays on all the time and handles:

- natural speech
- interruption
- clarification
- short summaries
- handoff language

**2. Single-task action worker**

the worker handles:

- intent classification
- ambiguity gate
- risk gate
- one-step planning
- task replacement
- verification

there is only one active task at a time.

**3. Local executor**

the macOS client handles:

- AX targeting
- input field focus
- text insertion
- hotkeys
- post-action context refresh

that split is easier to implement, easier to trace, and easier to trust.

## what I stopped doing

I stopped treating every internal capability as an equal peer agent.

memory, research, summaries, replay labeling, and multimodal escalation still matter. they just do not all belong in the hot path of "the user asked for a thing, now do the next safe step."

that is the core pivot.

the product did not get dumber.

it got stricter about where intelligence belongs.

## why single-task matters

desktop action agents fail in boring ways:

- they click the right thing in the wrong app
- they type into the wrong field
- they continue an old plan after the UI changed
- they start two tasks and lose the thread

the fix is not more personality. it is stronger control.

so VibeCat now assumes:

- one task id
- one current step
- one verification cycle
- one replacement decision if the user asks for something else

that turns a fuzzy assistant into a system you can actually debug.

## the part I kept from the old architecture

I did not throw away the intelligence lane.

I kept it, but moved it behind the worker.

that means:

- research can still help
- memory can still help
- low-confidence multimodal checks can still help
- session summaries can still help

but none of those should slow down the basic action loop unless they are truly necessary.

## the broader lesson

I think a lot of agent architecture mistakes come from confusing "how many capabilities exist" with "how many active agents the product needs."

VibeCat has many capabilities.

it should still feel like one PM and one worker.

that is a better product shape.

and honestly, it is a better engineering shape too.

---

*Building VibeCat for the [Gemini Live Agent Challenge](https://geminiliveagentchallenge.devpost.com/). Source: [github.com/Two-Weeks-Team/vibeCat](https://github.com/Two-Weeks-Team/vibeCat)*
