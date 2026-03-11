---
title: the cat no longer just watches your screen
published: false
description: VibeCat started as a screen-aware companion, but it became more compelling when it turned into a desktop UI navigator that can safely act on natural intent
tags: geminiliveagentchallenge, devlog, buildinpublic, macos
cover_image:
---

I created this post for the purposes of entering the Gemini Live Agent Challenge.

---

the original VibeCat pitch was easy to say:

"it's a cat on your desktop that watches your screen and speaks up when something matters."

that got attention fast.

it also hid the real product problem.

watching is interesting for about thirty seconds. acting is where the product starts.

so the current VibeCat is not best described as a screen-watching companion anymore. it is better described as a **desktop UI navigator for developer workflows on macOS**.

it still has the cat. it still has voice. it still sees the current screen and hears what you say. but the center of gravity moved from "I noticed something" to "I can do the next safe thing for you."

## what the product does now

the current interaction contract is:

- if your intent is clear and low-risk, VibeCat acts
- if your intent is ambiguous, VibeCat asks one short question
- if your request is risky, VibeCat stops and asks for explicit confirmation
- if the UI target is unclear, VibeCat drops to guided mode instead of guessing

that's a very different contract from a proactive companion.

the user doesn't have to memorize exact trigger phrases either. these all count:

- "go to the official docs"
- "run that again"
- "take me to the right place"
- "type this here"
- "find the search box"

the important thing is intent, not wording.

## why the screen still matters

the screen is still a first-class input. it just isn't the whole story anymore.

VibeCat builds action context from:

- the frontmost app
- the current window title
- the focused element
- selected text
- an accessibility snapshot

that turns the screen from a passive observation channel into execution context.

if you're in Chrome and say "type `gemini live api` here," the system does not just hallucinate what "here" means. it checks the focused element and AX tree, confirms that the target is a text input, focuses it if needed, and only then inserts the text.

that is the difference between a companion demo and a navigator product.

## the architecture that made it work

the runtime split is now very explicit:

```text
Gemini Live + VAD = Live PM
Gateway navigator = single-task action worker
macOS accessibility layer = local executor
```

the PM talks.

the worker plans.

the executor acts.

and everything runs one step at a time.

that last rule matters a lot. a desktop agent that silently starts two tasks at once does not feel magical. it feels unsafe.

## what changed in the user experience

the old version had a stronger "cat notices things on its own" flavor.

the current version is more grounded:

- it can still explain what it sees
- it can still summarize the current screen
- but it shines when you tell it to do something

the best demos now are not "look, the cat has opinions."

they are:

- "open the official docs"
- "find the input field"
- "run that command again"
- "take me back to the right file"

those are small actions, but they compound into actual workflow movement.

## the tradeoff

there is one thing I gave up in exchange for this pivot: some of the original ambient companion magic.

I think that trade was right.

for a challenge entry and for a real product, "acts safely on natural intent" is a stronger promise than "sometimes notices things on its own."

the cat is still there.

it just has a job now.

---

*Building VibeCat for the [Gemini Live Agent Challenge](https://geminiliveagentchallenge.devpost.com/). Source: [github.com/Two-Weeks-Team/vibeCat](https://github.com/Two-Weeks-Team/vibeCat)*
