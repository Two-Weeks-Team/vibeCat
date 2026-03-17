# P1-8: Native `computer_use`

## Status

Deferred.

## Source-Verified Facts

- Current Gemini computer-use guidance is for browser control agents.
- Current Go SDK exposes `genai.ComputerUse{Environment Environment}`.
- In v1.49.0 the available environment constants are:
  - `EnvironmentUnspecified`
  - `EnvironmentBrowser`
- There is no `EnvironmentOS`.

## What This Means For VibeCat

VibeCat is a macOS desktop navigator first.

The native `computer_use` tool does not replace:

- AX-based control
- local hotkey injection
- local click/paste/system actions
- multi-app desktop focus changes

## Decision

Do not spend roadmap budget replacing the current navigator with browser `computer_use`.

If the team wants to experiment:

- isolate it to a Chrome/browser-only branch
- compare it only against the current CDP path
- keep the main desktop runtime unchanged

## Valid Experimental Scope

- browser-only fallback for hard-to-target web controls
- side-by-side comparison with current CDP + AX + vision approach

## Invalid Scope

- replacing the 5+ desktop navigator tools
- replacing AX desktop control
- assuming normalized coordinates solve desktop targeting generally

## Revisit Condition

Reopen this only if Google ships desktop/OS environment support in the official SDK/docs.

## Sources

- [Gemini computer use guide](https://ai.google.dev/gemini-api/docs/computer-use)
