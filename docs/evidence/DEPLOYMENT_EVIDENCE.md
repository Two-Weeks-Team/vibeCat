# Deployment Evidence

This file tracks the current evidence baseline for the **UI Navigator** submission.

## Deployed Services

- project: `vibecat-489105`
- region: `asia-northeast3`
- gateway: `https://realtime-gateway-a4akw2crra-du.a.run.app`
- orchestrator: `https://adk-orchestrator-a4akw2crra-du.a.run.app`

## Evidence Categories

### Runtime

- websocket gateway is deployed
- orchestrator is deployed
- authentication and health paths exist
- Cloud Logging is active
- Cloud Trace is active
- Monitoring/dashboard evidence is tracked separately

### Submission Alignment

Evidence must now support the following claims:

- VibeCat is a desktop UI navigator
- it executes real UI actions through `VibeCat/Sources/VibeCat/AccessibilityNavigator.swift`, including `AXUIElementPerformAction`-backed control presses and guarded keyboard automation
- it supports a gold-tier workflow on Antigravity IDE, Terminal, and Chrome
- it is hosted on Google Cloud
- it emits observable runtime evidence for navigator turns

### Remaining Evidence Work

- final screenshot-grade trace capture for navigator flows
- final monitoring screenshots aligned with navigator metrics
- final public demo/proof asset links

## Final Proof Asset Checklist

The award-maximizing submission should capture these final artifacts:

### Hero Run Assets

- one recording of the primary Antigravity -> Chrome -> Terminal hero workflow
- one still image showing target highlight + step HUD during execution
- one still image or clip showing explicit ambiguity clarification
- one still image or clip showing safe downgrade instead of blind click

### Cloud-Native Assets

- Cloud Run service view for gateway and orchestrator
- Cloud Logging or Cloud Trace evidence from the same hero run
- monitoring screenshot for navigator metrics
- replay or persisted summary evidence if used in final architecture/proof story

### Submission Assets

- final public demo link
- final proof-of-deployment link or recording
- final public repository link
- final public blog/devlog link if used for bonus points

## Historical Note

Older evidence that refers to companion-intelligence or proactive-speech acceptance should be treated as implementation history, not current submission truth.
