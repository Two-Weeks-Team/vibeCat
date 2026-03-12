# UI Navigator Judge Proof Matrix (2026-03-12)

## Goal

Translate the official judging criteria into exact VibeCat proof artifacts so the final submission is easy for judges to score highly.

Use this document as the final submission workboard for video, screenshots, architecture, and runtime evidence.

## Rule

Every major claim in the submission must be backed by at least one of:

- a live demo scene
- a screenshot-grade proof asset
- a repo path
- a Cloud proof asset
- a trace/log/replay artifact

## Stage-One Pass / Fail Requirements

Before weighted judging matters, the submission must clearly include:

- public code repository
- text description
- architecture diagram
- proof of Google Cloud deployment
- demo video under 4 minutes

Assume judges may score without running the project.

## Judging Matrix

| Criterion | Judge Question | Required Proof | Repo / Asset Anchor |
|-----------|----------------|----------------|----------------------|
| Innovation & Multimodal UX | Is this beyond a text box? | voice request + screen-aware action in one live sequence | demo video hero scene |
| Innovation & Multimodal UX | Does it demonstrate visual precision rather than blind clicking? | target highlight + step HUD + verified target outcome | `docs/analysis/DEMO_STORYBOARD.md`, runtime capture |
| Innovation & Multimodal UX | Does the precision hold across real workstation setups? | one multi-monitor action clip with correct-display targeting | hero workflow support asset |
| Innovation & Multimodal UX | Does it feel live and context-aware? | uninterrupted live action flow, short narration, immediate step progression | demo hero scene |
| Technical Execution & Agent Architecture | Is the backend really Google Cloud-native? | Cloud Run service view + architecture diagram + backend code anchors | `docs/FINAL_ARCHITECTURE.md`, `docs/evidence/DEPLOYMENT_EVIDENCE.md` |
| Technical Execution & Agent Architecture | Is the system design sound? | one current architecture diagram + request path narrative | `docs/FINAL_ARCHITECTURE.md` |
| Technical Execution & Agent Architecture | Does it handle uncertainty and edge cases safely? | one ambiguity clip + one low-confidence downgrade clip + structured failure reason | demo support scenes + runtime logs |
| Technical Execution & Agent Architecture | Is there grounding and anti-hallucination behavior? | screenshot/AX-backed targeting, verify contract, no blind click story | `docs/PRD/DETAILS/UI_NAVIGATOR_EXECUTION_CONTROL_PLANE_DESIGN_20260312.md` |
| Demo & Presentation | Is the problem and solution obvious fast? | first 20 seconds state the problem and value clearly | demo opening |
| Demo & Presentation | Is there visual proof of deployment and architecture? | Cloud proof scene tied to same hero run + diagram | demo 3:25-3:50 segment |
| Demo & Presentation | Is the software real? | actual desktop footage only, no mockups | all demo scenes |

## Locked Claim Set

The submission should make only these strongest claims:

1. VibeCat is a desktop UI navigator for developer workflows on macOS.
2. It uses Gemini multimodal understanding to interpret the current screen and output executable actions.
3. It acts one verified step at a time across Antigravity, Chrome, and Terminal.
4. It asks when intent is unclear and refuses blind interaction when confidence is too low.
5. Its backend control plane is hosted on Google Cloud.
6. It remains live and conversational while acting.
7. It supports multi-monitor developer setups for those three surfaces.

Do not broaden claims beyond what is shown in the hero workflow and proof package.

## Hero Workflow Proof Bundle

The same hero run should supply as many score signals as possible.

### Required from one primary run

- natural English voice request
- visible target highlight before at least one critical action
- visible step HUD or status chip during execution
- preserved live conversational continuity while moving across surfaces
- Chrome docs open from the current coding context
- return to Antigravity/Codex for the next actionable step
- Terminal verification step or command-ready state
- final spoken or onscreen summary
- corresponding Cloud log/trace evidence for the same run

### Required from short support clips

- one ambiguity clarification
- one low-confidence refusal / guided fallback

## Proof Asset Inventory

## 1. Demo Video

Required:

- under 4 minutes
- English narration/copy
- actual software only
- hero workflow + ambiguity + safety + cloud proof

## 2. Architecture Diagram

Must show clearly:

- macOS client
- Gemini Live / multimodal input path
- Realtime Gateway on Cloud Run
- ADK Orchestrator on Cloud Run
- Firestore / replay / memory path if mentioned
- execution verification loop

## 3. Cloud Deployment Proof

Must capture:

- gateway service
- orchestrator service
- region / project
- proof they are active

Preferred addition:

- log or trace snippet showing navigator activity from the demo run

## 4. Runtime Trust Proof

Must capture at least one of each:

- verified success
- ambiguity clarification
- safe downgrade instead of blind click
- multi-monitor correct-target proof if more than one display is used in the demo

## 5. Reproducibility Proof

Must include:

- public repo
- README instructions
- deployment automation evidence if used for bonus points

## Bonus Contribution Matrix

| Bonus | Relative Value | Proof Needed |
|-------|----------------|--------------|
| Public content | highest | public blog/devlog/video explicitly saying it was created for this hackathon |
| Automated deployment | medium | repo path to deployment automation and brief submission mention |
| GDG membership | medium-low | public GDG profile if applicable |

These should improve the score, but they should never take time away from the hero run or proof package.

## Submission Read Path

Judges should be able to consume the project in this order:

1. demo video
2. short text description
3. architecture diagram
4. Cloud deployment proof
5. public repo / README

All five should tell the same story with the same terminology.

## Story Consistency Rules

Use the same phrases across all materials:

- `desktop UI navigator`
- `acts when intent is clear, asks when it is not`
- `visual precision, not blind clicking`
- `one verified step at a time`
- `Google Cloud-hosted control plane`
- `live collaborator, not silent automation`
- `multi-monitor aware`

Avoid mixing in older companion-era wording.

## Go / No-Go Gates

The submission is not ready until all are true:

- hero workflow succeeds reliably on camera
- one ambiguity flow is short and clean
- one safe refusal flow is short and clean
- target highlight and current-step proof are visible
- live narration/clarification value remains visible during action work
- Cloud proof is tied to the same runtime story
- architecture diagram matches what the code really does
- README and Devpost text do not over-claim beyond the demo

## Highest-Leverage Missing Assets

At this moment, the most valuable remaining assets are:

1. final hero-run recording
2. screenshot-grade target highlight / HUD capture
3. Cloud log/trace screenshots from the same run
4. explicit guided-fallback capture
5. one multi-monitor correct-target capture
6. final public submission copy aligned to this matrix

## Final Decision

The winning path is now concrete:

- ship visible precision
- prove safety and grounding
- keep architecture narrow and explainable
- tie demo proof to Cloud proof
- claim only what the hero workflow can prove impeccably
