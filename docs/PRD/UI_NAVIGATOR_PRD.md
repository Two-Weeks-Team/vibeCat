# VibeCat UI Navigator PRD

## Product Definition

VibeCat is a **desktop UI navigator for developer workflows on macOS**.

It preserves the existing overlay character and voice/text interaction shell, but its primary value is no longer proactive companion speech. The primary value is acting on the user's intent across desktop apps.

## Core Promise

VibeCat becomes the user's hands when the intent is clear, and asks a short clarification question when it is not.

## Primary Submission Surfaces

- Antigravity IDE
- Terminal
- Chrome

## Core Requirements

1. infer execution intent from natural language, not just exact trigger phrases
2. distinguish `execute_now`, `open_or_navigate`, `find_or_lookup`, `analyze_only`, and `ambiguous`
3. use AX-first execution with visual fallback only when safe
4. avoid blind clicks
5. execute exactly one step at a time
6. verify each step before continuing
7. downgrade to guided mode on low confidence

## Safety Requirements

- low-risk actions may execute immediately
- ambiguous requests must prompt once before execution
- risky actions must require explicit confirmation or be blocked
- destructive shell commands, token entry, send/submit, delete, deploy, and `git push` are not safe-immediate actions

## Acceptance Workflow

Hero workflow:

1. Antigravity shows a failing test or broken state
2. the user asks naturally for help navigating or applying the next step
3. VibeCat opens the relevant docs or surface in Chrome
4. VibeCat returns to Antigravity or Terminal
5. VibeCat performs and verifies one step at a time

## Submission Message

`A desktop UI navigator that acts on natural intent, not just exact trigger phrases.`
