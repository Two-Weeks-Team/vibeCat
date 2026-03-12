# UI Navigator Execution Control Plane Design (2026-03-12)

## Goal

Strengthen VibeCat's weak action-execution layer without changing the narrow runtime thesis:

- keep one user-facing `Live PM`
- keep one gateway-owned planner
- keep one local macOS executor
- keep one narrow multimodal resolver
- add a dedicated **execution control plane** between planner output and local action execution

Locked product scope for this submission window:

- supported gold surfaces are only `Antigravity`, `Chrome`, and `Terminal`
- multi-monitor support is mandatory
- the always-on screenshot path must never add visible command delay
- the existing live conversational value must remain intact

The purpose of this layer is not to make VibeCat more verbose.

The purpose is to make VibeCat feel like a capable teammate who:

- understands what it is about to do
- tells the user what it is doing in short useful language
- acts carefully on the right surface
- notices when execution is shaky
- recovers once when possible
- falls back clearly when confidence is not good enough

## Why This Is The Next Step

Current strengths already exist:

- natural-language intent routing works
- clarification and risky-action gates work
- screenshot + AX context already inform planning
- visible-text extraction and resolved text insertion exist
- background replay/memory/research are already off the hot path

Current weakness is concentrated in the last mile:

- target activation is still fragile on gold-tier surfaces
- verification is too generic for strong proof
- retries are too thin and too implicit
- the planner emits steps, but the executor still lacks surface-specific operational intelligence
- the user sees final outcomes, but not enough of the action reasoning during execution

This design closes that gap.

## Product Thesis

VibeCat should feel like a pair programmer with hands.

That means the system must expose three qualities during action work:

1. **Intent transparency**
   - the user should know what VibeCat is trying to do next
2. **Surface awareness**
   - execution should adapt to Chrome, Terminal, and Antigravity/Codex instead of pretending all apps behave the same
3. **Trustworthy fallback**
   - when action confidence drops, VibeCat should explain the stuck point briefly and ask for the next smallest unblock, not collapse into vague guided mode text

## Runtime Position

The execution control plane sits between gateway planning and raw AX execution.

```text
User
  -> Live PM
  -> Realtime Gateway planner
  -> Execution Control Plane
      -> surface adapter
      -> local executor
      -> verifier
      -> bounded recovery
      -> user-facing narration/hud
  -> step result
  -> gateway state progression
  -> async replay/memory/research
```

## Boundaries

### Keep In Gateway

- intent classification
- ambiguity handling
- risky-action gating
- one-task-at-a-time policy
- step sequencing
- state persistence
- bounded replan after explicit execution failure reason

### Keep In Orchestrator

- low-confidence descriptor resolution from screenshot + AX context
- screen-derived text resolution
- async replay summaries
- async research enrichment
- async memory writes

### Move Into Execution Control Plane On Client

- surface-specific action selection
- activation strategy selection
- action-specific verification
- local bounded retry with alternative action strategy
- concise execution narration and proof surfaces

### Keep As A Separate Background Lane On Client

- continuous screenshot capture scheduling
- display-aware context caching
- diffing / deduplication
- JPEG/base64 preparation
- low-priority replay/proof snapshot preparation

The execution control plane must read from this lane instantly. It must not wait on a fresh screenshot unless confidence drops or verification explicitly demands it.

This keeps planning server-side and execution local.

## Core Design

### 1. Action Contract

Planner output must stop being just a raw step tuple.

Each executable step should become an **Action Contract** with enough structure for the client to act intelligently without inventing policy.

Suggested model additions:

```text
NavigatorStep
- id
- actionType
- targetApp
- targetDescriptor
- expectedOutcome
- inputText
- hotkey
- verifyHint
- confidence
- intentConfidence
- riskLevel
- executionPolicy
- fallbackPolicy

+ surface
+ macroID
+ narration
+ verifyContract
+ fallbackActionType
+ fallbackHotkey
+ maxLocalRetries
+ timeoutMs
+ proofLevel
```

Suggested `verifyContract` shape:

```text
VerifyContract
- expectedBundleId
- expectedWindowContains
- expectedFocusedRole
- expectedFocusedLabel
- expectedAXContains
- expectedSelectedTextPrefix
- requireWritableTarget
- requireFrontmostApp
- minCaptureConfidenceAfter
- proofStrategy
```

The planner still decides the goal. The client gains the structured rules to verify it safely.

### 2. Surface Adapter Layer

Add a client-side adapter registry:

```text
SurfaceAdapterRegistry
- ChromeAdapter
- TerminalAdapter
- AntigravityAdapter
- DefaultAXAdapter
```

For this submission, treat the three named adapters as the product surface. Everything else is fallback-only and not part of the winning claim set.

Each adapter supplies:

- target ranking hints
- preferred activation strategy
- preferred text entry strategy
- action-specific wait conditions
- proof strategy
- user-facing surface wording

Example responsibilities:

#### ChromeAdapter

- treat address-bar and search interactions as first-class actions
- prefer deterministic shortcuts like `Cmd+L` when the requested action matches
- verify by frontmost bundle + address/search focus + window change

#### TerminalAdapter

- treat command line readiness as a precondition
- prefer paste + return only when focused text area is stable
- fail closed if shell appears busy or focus is unclear

#### AntigravityAdapter

- specialize prompt/composer activation
- specialize command palette or inline-apply flows
- prefer app-specific activation points and post-focus checks
- treat first insert and follow-up insert as different flows

#### Multi-Monitor Requirement

All adapters must be display-aware.

Required inputs per action:

- frontmost bundle id
- focused window bounds
- active display id
- cursor display id
- target display id

The adapter must resolve and prove the correct display before any visible action executes.

#### DefaultAXAdapter

- preserve the existing generic AX-first flow
- remain the safe fallback for unsupported apps

### 3. Execution Transaction

Every action should execute inside an explicit transaction rather than an open-coded sequence.

Suggested phases:

```text
1. preflight
2. resolve_target
3. activate_target
4. prove_target_ready
5. perform_action
6. read_back
7. verify_outcome
8. recover_once_or_fail
```

Before `resolve_target`, the transaction must also bind to the intended display context so actions cannot drift to the wrong monitor.

Every phase emits:

- phase name
- start/end time
- success/failure
- structured reason
- proof snapshot summary

This replaces vague `guided_mode` collapse with meaningful statuses such as:

- `focus_not_ready`
- `wrong_target`
- `target_not_found`
- `target_not_writable`
- `verification_inconclusive`
- `paste_rejected`
- `surface_adapter_unavailable`

### 4. Bounded Local Recovery

Recovery must happen first on the client when it is purely mechanical.

Allowed recovery patterns:

- AX focus -> caret placement -> activation click
- paste -> typeText fallback
- adapter strategy A -> adapter strategy B
- waitUntil shorter signal -> waitUntil longer signal

Not allowed:

- repeated blind retries
- planner loops inside the client
- infinite fallback chains

Rule:

- one local retry family per step
- at most one gateway replan after the client returns a precise failure reason

## Background Screenshot Lane

The current screenshot pipeline should be split into two explicit roles:

### A. Context Cache Lane (Always On, Background)

Purpose:

- keep fresh per-display snapshots ready
- keep cursor-display and frontmost-window metadata ready
- keep pre-encoded fast-path and smart-path payloads ready when useful

Rules:

- never block the command hot path
- keep one latest trusted snapshot per display
- attach age/confidence metadata to every cached snapshot
- prepare data in the background, not on command submission

Suggested model:

```text
DisplayContextCache
- displays: [DisplaySnapshot]
- activeDisplayID
- activeAppBundleID
- activeWindowTitle
- activeWindowFrame
- lastTrustedSnapshotAgeMs
- lastTrustedCaptureConfidence
```

### B. Action Hot Path (On Demand, Deterministic)

Purpose:

- execute instantly using cached visual context + live AX context

Allowed fresh captures only when:

- low-confidence escalation is required
- verification mismatch occurs
- user explicitly requests analysis now

This is the key latency rule for the submission build.

## User-Facing Collaboration Layer

This is the part that makes the system feel like a real teammate instead of a hidden automation engine.

This layer stays.

Execution hardening must not remove:

- live speech output
- clarification prompts
- risk confirmation flow
- narration/status bubbles
- interruption handling
- conversational continuity across surface switches

### Principles

- say what matters, not every internal step
- use short concrete action language
- reveal uncertainty honestly
- distinguish `thinking`, `acting`, `stuck`, and `done`

### Required Behaviors

#### Before action

VibeCat gives a short forward action statement:

- "Opening Chrome for the official docs."
- "Focusing the Codex composer first."
- "I found the likely input field. Trying the safer path."

#### During action

Show a lightweight visible step HUD:

- current step
- target surface
- confidence band
- if recovery is in progress, say which one

#### When stuck

Do not say only "guided mode".

Say the smallest useful truth:

- "I found the window, but the composer still isn't writable."
- "I can see two likely fields and can't prove which one is yours."
- "Paste was blocked, so I stopped before typing into the wrong place."

#### After success

Summarize observable outcome, not just action name:

- "The docs search is open in Chrome."
- "The command is in Terminal and ready to run."
- "The Codex composer is focused and the follow-up text is inserted."

This layer is product-critical for judge trust and teammate feel.

## Proof Layer

The action system needs visible evidence.

### Add

- target highlight ring before critical press/paste actions
- step HUD for current action and confidence
- replay card after completion with step count and surface summary
- per-phase trace logs with stable names
- display badge or target-monitor cue when multiple monitors are active

### Proof Levels

```text
basic
- frontmost app changed as expected

strong
- focused target matches descriptor and app/window

strict
- writable target proven and post-action readback confirms likely success
```

The planner chooses proof level based on risk and surface.

## File-Level Design

### `VibeCat/Sources/VibeCat/AccessibilityNavigator.swift`

Add or refactor:

- `SurfaceAdapter` protocol
- `SurfaceAdapterRegistry`
- `ExecutionTransaction`
- `VerifyContract` evaluation
- `waitUntil` polling helper
- `typeText` fallback path
- expanded keyboard map
- phase-level structured result reporting
- target highlight support
- display binding + multi-monitor target proof

### `VibeCat/Sources/VibeCat/NavigatorActionWorker.swift`

Change from a thin wrapper into the entrypoint for:

- execution transaction start/end
- structured failure reason propagation
- step HUD lifecycle
- local retry envelope

### `VibeCat/Sources/VibeCat/AppDelegate.swift`

Add hooks for:

- concise narration updates
- step HUD state
- replay card after completion
- user-facing stuck-state phrasing
- command path reads latest trusted screenshot context instead of forcing new capture by default

### `VibeCat/Sources/VibeCat/ScreenAnalyzer.swift`

Refactor from a mostly main-actor scheduler into:

- thin UI-facing scheduler
- separate background capture/cache lane
- per-display snapshot cache
- pre-encoded payload preparation
- explicit cache age/confidence reporting

The command hot path must not depend on synchronous fresh encode work from this file.

### `VibeCat/Sources/Core/NavigatorModels.swift`

Add shared contract types:

- `VerifyContract`
- `ExecutionFailureReason`
- `ProofLevel`
- `SurfaceKind`

### `backend/realtime-gateway/internal/ws/navigator.go`

Upgrade planning output to include:

- `surface`
- `macroID`
- `narration`
- `verifyContract`
- `fallbackActionType`
- `fallbackHotkey`
- `maxLocalRetries`
- `proofLevel`

Keep rule-based planning narrow and deterministic.

### `backend/realtime-gateway/internal/ws/handler.go`

Use richer client outcomes to support:

- one bounded replan after precise failure reason
- stronger per-phase metrics
- clearer guided-mode text when recovery is exhausted

### `backend/realtime-gateway/internal/ws/navigator_confidence.go`

Keep escalation narrow, but allow escalated output to improve:

- `targetDescriptor`
- `resolvedText`
- optional surface-specific hint fields later if needed

### `backend/adk-orchestrator/internal/navigator/processor.go`

Do not turn this into a planner.

Only sharpen:

- target resolution quality
- screen-derived text extraction quality
- reason strings that help the gateway choose safe fallback behavior

## Gemini API Adoption Rules

The execution control plane should assume Gemini integrations stay current, pinned, and capability-checked.

### Production Rules

- pin Live model versions explicitly
- avoid `*-latest` in production
- keep all Gemini credentials and calls server-side
- use ephemeral tokens if any client-facing session token flow is introduced
- keep Live API tool execution manual and explicit
- treat computer-use style capabilities as orchestrator/planner inputs, not direct blind client control

### Codebase Process

Create a lightweight recurring update loop:

1. review `ai.google.dev` changelog and deprecations
2. review model and Live capability pages
3. update pinned model map in code/config
4. update local reference docs under `docs/reference/` when major changes land
5. run a capability checklist against current runtime assumptions

Suggested maintained checklist:

- live model pin
- tool support matrix
- session resumption support
- context compression assumptions
- TTS path and voice config
- embeddings usage assumptions
- retry/troubleshooting policy

## Implementation Phases

### Phase A - Transaction + Visibility

Ship first:

- `waitUntil`
- structured execution phases
- phase-level failure reasons
- target highlight
- step HUD
- concise narration strings
- command hot path no longer blocks on fresh screenshot capture/encoding

Success bar:

- current runtime behaves the same or safer
- every step emits observable proof
- users can tell what VibeCat is attempting
- command submission latency is no longer tied to screenshot work

### Phase B - Gold Surface Adapters

Ship next:

- `ChromeAdapter`
- `TerminalAdapter`
- `AntigravityAdapter`
- first-insert vs follow-up-insert handling for Codex/Antigravity composer flows
- multi-monitor target binding for all three surfaces

Success bar:

- guided mode rate drops on gold surfaces
- follow-up insert reliability improves materially
- actions land on the correct monitor consistently

### Phase C - Gateway Contract Upgrade

Ship next:

- `verifyContract`
- `surface`
- `macroID`
- bounded local retry fields
- better failure-reason routing back into planner

Success bar:

- planner output is richer but still narrow
- gateway remains single planner, not orchestration soup

### Phase D - Recovery + Replay Quality

Ship next:

- one bounded replan after local recovery exhaustion
- replay cards and per-phase trace summaries
- eval fixtures for Codex/Antigravity, Chrome, and Terminal hero flows

Success bar:

- failed runs explain where they failed
- successful runs produce reusable proof and regression fixtures

## Acceptance Criteria

This design is successful when:

1. the runtime shape still matches `docs/FINAL_ARCHITECTURE.md`
2. gold-surface actions feel more deterministic without adding agent sprawl
3. action results return precise failure reasons instead of vague guided mode
4. the user can follow what the system is doing in real time
5. demo observers can see proof of target selection and outcome verification
6. Gemini integration stays pinned and reviewable rather than drifting implicitly

## Immediate Build Order

1. add shared types in `NavigatorModels.swift`
2. split screenshot pipeline into background `DisplayContextCache` lane and hot-path lookup
3. add structured result + failure reason plumbing in `NavigatorActionWorker.swift`
4. refactor `AccessibilityNavigator.swift` around `ExecutionTransaction`
5. add `waitUntil`, target highlight, display proof, and `typeText` fallback
6. add `ChromeAdapter` and `TerminalAdapter`
7. add `AntigravityAdapter` with explicit composer flows
8. upgrade gateway step contract fields
9. add bounded replan logic in gateway refresh handling
10. add HUD + narration + replay card and lock hero eval runs on Antigravity, Terminal, and Chrome

## Final Note

The design goal is not to make VibeCat look more agentic on paper.

The design goal is to make a user feel:

- "it understood what I meant"
- "it told me what it was doing"
- "it acted carefully in the right place"
- "it stopped honestly when it could not prove safety"

That is the teammate feeling worth shipping.
