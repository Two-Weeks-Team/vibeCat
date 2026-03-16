# VibeCat UI Navigator PRD

## Product Definition

VibeCat is a **desktop UI navigator for developer workflows on macOS**.

The submission keeps the existing overlay character and voice/text interaction shell, but the primary value is no longer proactive companion speech. The target value is acting on the user's intent across desktop apps.

## Core Promise

VibeCat becomes the user's hands when the intent is clear, and asks a short clarification question when it is not.

## Runtime Thesis

VibeCat is not a swarm of equal agents.

It is:

- one `Live PM` that speaks with the user through Gemini Live + VAD
- one `single-task action worker` that plans and tracks executable work
- one local `AX-first executor` that performs UI actions
- optional background intelligence that can enrich the experience without blocking execution

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

## Plan Of Record

The implementation plan is tracked in:

- `docs/FINAL_ARCHITECTURE.md` — Authoritative architecture reference
- `docs/evidence/DEPLOYMENT_EVIDENCE.md` — Live deployment proof
- `docs/CURRENT_STATUS_20260316.md` — Current project status

Implementation status against that plan:

1. `Live PM + single-task worker` boundaries are locked in code
2. action task state is externalized through `ActionStateStore`
3. pre-action context includes screenshot, AX hashes, focus stability, and input-field descriptors
4. low-confidence target resolution goes through a narrow multimodal escalator
5. post-task summary, replay labeling, research enrichment, and memory writes are on the async background lane
6. step-level metrics and replay fixtures are in place for regression coverage
7. submission docs are aligned to the navigator runtime

## Submission Message

`A desktop UI navigator that acts on natural intent, not just exact trigger phrases.`
