---
title: making Go speak real-time — the transport layer behind VibeCat's Live PM
published: false
description: VibeCat's Go gateway is more than a WebSocket proxy now: it keeps Gemini Live voice responsive while separating the PM conversation plane from the action worker
tags: geminiliveagentchallenge, devlog, buildinpublic, go
cover_image:
---

# making Go speak real-time — the transport layer behind VibeCat's Live PM

The first time I got the audio proxy working, the cat answered in a voice that sounded like a cheerful modem from a haunted office.

The sample rate was wrong.

That was a good introduction to this part of the system.

I created this post for the purposes of entering the Gemini Live Agent Challenge. I'm building [VibeCat](https://github.com/Two-Weeks-Team/vibeCat), a desktop UI navigator for developer workflows on macOS.

## the gateway is no longer just a proxy

The original job of the Go gateway was straightforward:

- accept a WebSocket from the macOS client
- forward audio to Gemini Live
- send model audio back

That part still exists.

But the gateway became more important as the product pivoted toward UI navigation.

It now sits between two very different responsibilities:

- a **Live PM** that talks naturally with the user through Gemini Live + VAD
- a **single-task action worker** that interprets executable intent, applies risk checks, plans the next step, and coordinates local execution

That means the gateway is not just transport anymore. It is the boundary between conversation and action.

## why the split matters

If you blend those two responsibilities together, the system gets fuzzy fast.

The conversation layer wants to:

- clarify
- summarize
- explain
- stay low-latency

The action layer wants to:

- classify intent
- enforce one active task
- require confirmation for risky steps
- track step ids and task ids
- verify results before continuing

Those are different jobs.

The gateway is where they meet, but they should not collapse into one thing.

## the transport path

At the transport level, the shape is still simple:

```text
macOS client <-> gateway websocket <-> Gemini Live session
```

Audio goes up from the client as PCM. Audio comes back from Gemini as streamed chunks. VAD, interruption, and model turn state all travel through the same session.

That part needs to stay lean because every extra delay shows up directly in the user's ear.

So the gateway keeps the Live session focused on:

- audio input forwarding
- audio output streaming
- barge-in handling
- session resumption
- reconnect handling

That is the PM plane.

## the action plane

The navigator path is different.

When the client sends a command like:

- "open the official docs"
- "find the search box"
- "type this here"

the gateway does not just forward that as plain conversation.

It routes the command through a navigator worker that:

1. classifies the intent
2. decides whether the request is explanatory or executable
3. asks a clarification question if it is ambiguous
4. blocks or confirms if it is risky
5. plans one step at a time
6. waits for verification before continuing

That is a very different control flow from an ordinary chat turn.

## one task at a time

The biggest architectural rule in this worker is simple:

**there is only one active executable task at a time.**

This solved a bunch of subtle problems at once:

- stale step refreshes
- overlapping plans
- ambiguous task ownership
- hidden concurrency in the UI loop

Now every planned action is tied to:

- `taskId`
- `command`
- `stepId`

and every verification message has to match all three before the worker accepts it.

That made the system stricter, but it also made it much easier to trust.

## why Go fit this well

Go turned out to be a good fit for this layer for boring reasons, which is usually a good sign.

It is good at:

- long-lived connections
- simple explicit state machines
- clear concurrency boundaries
- structured logging
- Cloud Run deployment

The important part of the gateway is not language cleverness. It is that the runtime stays understandable while juggling:

- WebSocket IO
- Gemini Live session state
- reconnect state
- task state
- step verification
- background analysis hooks

Go keeps that manageable.

## what stayed out of the hot path

One of the biggest improvements was deciding what **not** to keep in the gateway's real-time action loop.

The slower intelligence lane still matters:

- contextual analysis
- research
- memory
- summaries

but those belong behind the action worker or after a task completes, not in the immediate path of "the user said do this."

That boundary is what keeps the product responsive.

## the practical result

The current gateway is more credible than the earlier version because it has a clearer contract.

The Live PM remains conversational and fast.

The action worker remains explicit and narrow.

The local macOS client executes and verifies the actual step.

The gateway is the seam that keeps those planes separate while still letting them feel like one product.

That was the right evolution.

---

*Building VibeCat for the [Gemini Live Agent Challenge](https://geminiliveagentchallenge.devpost.com/). Source: [github.com/Two-Weeks-Team/vibeCat](https://github.com/Two-Weeks-Team/vibeCat)*
