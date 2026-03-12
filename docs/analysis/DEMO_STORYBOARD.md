# VibeCat Demo Storyboard — UI Navigator Submission

**Target length:** under 4 minutes
**Language:** English narration and on-screen copy

## Story Goal

Show that VibeCat is:

- a visual UI navigator
- safe but fast
- effective across Antigravity IDE, Terminal, and Chrome
- deployed and observable on Google Cloud
- reliable on a real multi-monitor developer desk

Every scene should also answer one judging question explicitly.

## Narrative Arc

### 0:00 - 0:20 Problem

Judging question answered: `What problem is this solving, and why is this not just chat?`

- Show Antigravity IDE with a failing test or broken behavior.
- Narration: “VibeCat is a desktop UI navigator for developers. It becomes your hands when your intent is clear, and asks when it is not.”

### 0:20 - 1:40 Hero Workflow

Judging question answered: `Can it act precisely on the real screen?`

- User says a natural request, not a rigid trigger phrase.
- VibeCat infers the action intent.
- It opens the relevant official docs in Chrome.
- It returns to the development surface.
- It opens the right place in Antigravity or prepares the next action.
- It runs a verification step in Terminal.

What must be visible:

- target highlight
- step status chip
- execution
- post-action verification
- concise narration of current action
- if using two displays, the target appears on the correct monitor

### 1:40 - 2:15 Ambiguity Gate

Judging question answered: `Does it ask when the user's intent is unclear?`

- Use a request that could mean “do it” or “explain it”.
- VibeCat asks one short clarification question.
- The user gives a short confirmation.
- VibeCat continues.

### 2:15 - 2:50 Safety / No Blind Click

Judging question answered: `Does it refuse unsafe or low-confidence action instead of guessing?`

- Show a low-confidence or unsupported target.
- VibeCat explicitly refuses to click blindly.
- It switches to guided mode instead.

### 2:50 - 3:25 Cross-App Continuity

Judging question answered: `Can it stay coherent across real developer tools?`

- Show Antigravity, Terminal, and Chrome all participating in a single flow.
- Close the loop with a successful verification or improved state.

### 3:25 - 3:50 Cloud Proof

Judging question answered: `Is the backend real, cloud-native, and tied to this same run?`

- Show Cloud Run services.
- Show recent trace or logs for the same navigator run shown in the hero flow.
- Show monitoring dashboard or runtime metrics.
- If possible, show replay or persisted task evidence from the same run.

### 3:50 - 4:00 Close

- Title card:
  - `VibeCat`
  - `Desktop UI Navigator for Developer Workflows`
  - `Gemini Live Agent Challenge 2026 — UI Navigator`

## Non-Negotiable Demo Beats

- one natural-language request without exact trigger phrases
- one ambiguity clarification
- one verified UI action
- one guided-mode downgrade instead of a blind click
- one cross-app flow across Antigravity IDE, Terminal, Chrome
- one Cloud proof shot
- one proof that the Cloud proof belongs to the same hero workflow, not generic infrastructure footage
- if multi-monitor is shown, one proof beat that VibeCat selected the correct display

## Screen Plan

### Hero Flow

1. Antigravity shows failure state.
2. User request: docs lookup / fix navigation intent.
3. Chrome official docs open.
4. Antigravity receives the next action.
5. Terminal runs validation.
6. VibeCat summarizes the result.

### Support Flow

1. GitHub issue in Chrome.
2. VibeCat moves the user back to the relevant code location in Antigravity.

## What Not To Show

- proactive companion chatter as the main differentiator
- emotionally themed celebration sequences
- politically charged character variants
- broad consumer-app automation claims
- any action that looks like a blind coordinate click
- multiple partially working workflows instead of one impeccable hero workflow
- silence during action if it removes the teammate feel
