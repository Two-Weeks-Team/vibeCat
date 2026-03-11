---
title: Swift 6, ScreenCaptureKit, and why the macOS client became an executor
published: false
description: VibeCat's macOS client started as a screen-aware frontend, but the real breakthrough came when it became a local action worker with ScreenCaptureKit, AX execution, and input recovery
tags: geminiliveagentchallenge, devlog, buildinpublic, swift
cover_image:
---

# Swift 6, ScreenCaptureKit, and why the macOS client became an executor

I created this post for the purposes of entering the Gemini Live Agent Challenge. I'm building [VibeCat](https://github.com/Two-Weeks-Team/vibeCat), a desktop UI navigator for developer workflows on macOS.

There was a point where I thought the macOS client was the easy part.

The backend had Gemini Live, Cloud Run, WebSockets, and all the usual networking pain. The client just had to capture the screen, listen to the microphone, play the voice back, and draw a cat in the corner.

That assumption was wrong.

The macOS client became one of the most important architectural pieces in the project because it is the only place where VibeCat can actually touch the desktop safely.

## the client is no longer just a frontend

The current VibeCat split looks like this:

```text
Gemini Live + VAD = Live PM
Gateway navigator = single-task action worker
macOS client = local action worker + AX-first executor
```

That last line is the real change.

The Swift app does not just render UI anymore. It now owns:

- ScreenCaptureKit capture
- current app and window context
- focused element state
- accessibility-backed execution
- text input targeting
- post-action verification
- audio input recovery when devices switch

That means the client is part of the product's control plane, not just the shell.

## ScreenCaptureKit was the start, not the end

The first reason to build the macOS client carefully was screen capture.

VibeCat needs current context from the desktop, but not as a constant flood of screenshots. It needs useful captures that line up with the active workflow.

ScreenCaptureKit was the right foundation because it gives you:

- access to visible displays and windows
- exclusion of your own app windows
- good enough image quality for downstream reasoning
- control over when and how often captures happen

That solved the "what's on the screen?" problem.

It did not solve the more important question:

"where exactly is the thing I should act on?"

## the real problem was input targeting

The moment VibeCat started moving from companion behavior to UI navigation, the client had to do more than observe.

It had to:

- identify the current input field
- confirm that the field was safe to target
- focus it
- insert text
- verify that the action actually landed in the right place

This is where pure screenshot understanding stops being enough.

If the user says:

- "type this here"
- "find the search box"
- "paste that into the input field"

the app can't just answer conversationally. It has to act on the current macOS UI with precision.

That means the client must combine:

- frontmost app info
- window title
- focused accessibility role
- focused label
- selected text
- an AX snapshot

and turn that into a safe local execution step.

## why AX-first matters

For real desktop interaction, Accessibility is the safest primary interface.

VibeCat now treats AX as the default path for actions like:

- focus app
- press control
- focus text field
- paste text
- send hotkey

Only after the target is resolved locally does the action go through.

That gives the system a better contract than "the model thinks the search box is somewhere near the top left."

The key rule is simple:

**do not click blindly.**

If AX resolution is weak or the target is ambiguous, the client drops to guided mode instead of pretending confidence.

That rule matters more than any amount of "smartness."

## text entry was the turning point

One of the most important client-side changes was handling text entry as a real execution path instead of a generic response.

The old behavior could end up saying something like:

"I can see the page, but it's hard to tell where the input field is."

That is a reasonable model answer. It is not a good navigator answer.

Now the runtime treats text entry as its own path:

1. build a text-entry-aware target descriptor
2. resolve or focus the input field locally
3. insert the requested text
4. verify the result from updated context

That shift forced the client to stop being a passive observer and become a deterministic executor.

## audio input recovery was a real product bug

The other surprisingly difficult client problem was microphone recovery.

If I switched from MacBook speakers to AirPods and back, the recognition path could quietly die. The system looked alive, but nothing happened because the audio engine was still bound to the wrong device state.

That led to three changes:

- track the current system input device continuously
- rebind listening when the default input changes
- surface the current audio input and capture state in the menu

This sounds operational, but it is product-critical. A voice-first navigator that silently stops listening is not reliable enough to demo, let alone ship.

## Swift 6 was annoying and right

Swift 6 pushed hard on actor isolation, thread safety, and UI boundaries.

In the moment, that felt expensive.

In hindsight, it was exactly what this client needed.

The client now has:

- UI work on the main actor
- execution boundaries that are clearer
- less casual cross-thread mutation
- more explicit async transitions around capture, playback, and execution

That pressure made the runtime less sloppy.

## what the macOS app is now

The best way to describe the current client is this:

it is a local operating surface for a cloud-backed navigator.

It captures context, executes verified actions, and reports what happened back to the worker. The cat, the speech bubble, and the tray icon still matter. But they are no longer the most interesting thing about the app.

The interesting thing is that the macOS client now has real responsibility.

That was the right architectural turn.

---

*Building VibeCat for the [Gemini Live Agent Challenge](https://geminiliveagentchallenge.devpost.com/). Source: [github.com/Two-Weeks-Team/vibeCat](https://github.com/Two-Weeks-Team/vibeCat)*
