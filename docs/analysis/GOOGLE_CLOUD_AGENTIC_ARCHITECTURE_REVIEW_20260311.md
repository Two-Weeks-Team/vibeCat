# Google Cloud Agentic Architecture Review (2026-03-11)

This note maps Google Cloud's published agentic AI architecture guidance to the current VibeCat runtime.

Reviewed sources:

- `choose-agentic-ai-architecture-components`
- `choose-design-pattern-agentic-ai-system`
- `multiagent-ai-system`
- `single-agent-ai-system-adk-cloud-run`
- `agentic-ai-interactive-learning`
- `agentic-ai-classify-multimodal-data`

## Summary

The guidance supports VibeCat's current direction:

- keep a single user-facing agent
- keep execution deterministic
- isolate risky actions with human approval
- use multi-agent or parallel specialists only behind a coordinator and only when the task truly needs it
- externalize state for resilience

For VibeCat, the right target is:

- `Live PM` as the only user-facing agent
- `single-task action worker` as the only executor-facing coordinator
- optional specialist intelligence as background tools, not as equal peers in the hot path

## What Fits VibeCat Well

### 1. Single-agent first

Google's design guidance recommends starting with simpler patterns and only moving to multi-agent when the task complexity demands it.

VibeCat should therefore keep:

- one user-facing conversation plane
- one action coordination plane
- one deterministic local executor

This matches the current `Live PM + single-task worker` split.

### 2. Human-in-the-loop

The design-pattern guidance explicitly supports human approval for risky operations.

This directly validates:

- clarification before ambiguous actions
- explicit confirmation for risky actions
- guided mode fallback instead of blind interaction

### 3. External state

The component guidance recommends externalizing session and task state for production resilience on Cloud Run.

This supports the next planned change for VibeCat:

- move active task state out of per-connection memory
- persist `taskId`, `currentStepId`, `riskState`, and prompt state

### 4. Background learning and asynchronous feedback

The interactive-learning guidance separates the real-time path from post-task learning and feedback loops.

For VibeCat this means:

- keep action execution hot path small
- move memory writes, summaries, and replay labeling to the background lane

### 5. Narrow multimodal escalation

The multimodal classification guidance is useful, but not as a default architecture.

It is most valuable when VibeCat has low confidence about a target and needs to cross-check:

- AX snapshot
- screenshot
- selected text
- focus metadata

This should remain an exception path, not the baseline.

## What Does Not Fit the Hot Path

### 1. Full multi-agent coordination

The multi-agent guidance is helpful for complex distributed business workflows, but it adds:

- more model calls
- more latency
- more coordination overhead
- harder debugging

That makes it a poor fit for VibeCat's interactive action loop.

### 2. Parallel agent fan-out by default

Parallel subagents are useful only when there is a clear accuracy gain.

For VibeCat's action path, the default must remain:

- deterministic planning
- one step at a time
- one active task at a time

### 3. Treating execution and conversation as one agent

The reviewed guidance consistently separates orchestration, tools, and execution concerns.

That reinforces the current VibeCat pivot:

- Live PM handles speech
- action worker handles state and decisions
- local executor handles UI actions

## Concrete Recommendations

### Recommended now

- introduce `ActionStateStore`
- persist active task state
- add step-level metrics and replay traces
- formalize `before_action_context`
- keep ADK research and memory off the action hot path

### Recommended later

- add narrow multimodal confidence escalation for low-confidence target resolution
- expand replay-based eval coverage
- optionally standardize tool boundaries if the number of specialists grows significantly

### Not recommended now

- replacing Cloud Run
- general multi-agent hot-path orchestration
- broad parallel planner fan-out
- making the Live PM itself responsible for step execution

## Target Architecture Statement

The target architecture for VibeCat should be:

`one Live PM, one single-task action worker, one deterministic local executor, optional specialist intelligence behind the worker.`

That is the most defensible path for:

- latency
- reliability
- observability
- safety
- submission clarity
