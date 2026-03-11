# Proof of GCP Deployment

This checklist tracks the submission-grade proof package for the **UI Navigator** version of VibeCat.

## Required Proof

- deployed Cloud Run gateway
- deployed Cloud Run orchestrator
- recent logs for navigator requests
- recent traces for navigator turns
- monitoring/dashboard evidence
- public repo and README
- demo video alignment with the deployed product

## Required Screens / Artifacts

- Cloud Run service details for `realtime-gateway`
- Cloud Run service details for `adk-orchestrator`
- log entry showing navigator command handling
- trace view showing navigator turn classification and execution events
- monitoring dashboard with runtime health
- repository README showing UI Navigator framing

## Narrative Requirements

The proof package must match the submission story:

- category: UI Navigator
- product type: desktop UI navigator
- hero surfaces: Antigravity IDE, Terminal, Chrome
- interaction contract: acts when intent is clear, asks when it is not

## Completion Standard

This proof document is complete only when:

1. every screenshot or recording uses the UI Navigator framing
2. at least one navigator step is visible in logs or trace evidence
3. URLs, service names, and revision identifiers match the current deployment
