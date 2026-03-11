# Submission and Demo Plan

## Submission Artifacts

- Devpost product description aligned to `UI Navigator`
- public repository URL
- setup and verification instructions in `README.md`
- proof of Google Cloud deployment
- architecture diagram showing `Live PM`, `single-task worker`, `confidence escalator`, `background lane`, and observability
- demo video under 4 minutes

## Locked Demo Story

VibeCat is no longer pitched as a proactive desk companion.

The demo story is:

> “Talk naturally. If the intent is clear, VibeCat acts. If it is not, VibeCat asks once. Then it executes one verified step at a time across real macOS apps.”

## Locked Hero Flow

The demo stays on the three gold-tier surfaces:

1. **Antigravity IDE**
2. **Chrome**
3. **Terminal**

Recommended sequence:

1. Start in Antigravity with a failing test or broken state visible.
2. Ask naturally for the next concrete action.
3. Show one clarification or confidence-safe resolution when the target is not fully clear.
4. Open Chrome to the relevant official docs.
5. Return to Antigravity or Terminal for the next actionable step.
6. Execute exactly one step at a time with verification bubbles.
7. Show Cloud Run / Trace / logs / replay proof.

## Demo Timeline (Under 4 Minutes)

| Time | Scene | Runtime Truth To Show |
|---|---|---|
| 0:00–0:20 | Hook | “Desktop UI navigator for developer workflows on macOS” |
| 0:20–0:45 | Intent | Natural-language request in Antigravity |
| 0:45–1:10 | Ambiguity Gate | One clarification or low-confidence safe handling |
| 1:10–1:40 | Chrome Docs | VibeCat opens the official docs search in Chrome |
| 1:40–2:10 | Antigravity / Terminal | Return to the work surface and execute one next step |
| 2:10–2:35 | Verification | Show step verification, guided mode fallback, or replacement handoff |
| 2:35–3:05 | Architecture | Live PM + single-task worker + confidence escalator + background lane |
| 3:05–3:35 | Cloud Proof | Cloud Run services, logs, trace, monitoring exporter |
| 3:35–4:00 | Close | “Acts when intent is clear, asks when it is not.” |

## Submission Copy Anchors

The submission text should stay close to these claims:

- natural-language intent, not exact trigger phrases
- one task at a time
- safe immediate execution for low-risk steps
- asks once when ambiguous
- no blind clicks
- multimodal confidence escalator only when AX confidence is low
- async background intelligence outside the action hot path

## Proof Checklist

- hero demo only uses runtime-supported flows
- architecture diagram matches current code
- Cloud deployment proof shows both services on Cloud Run
- logs / traces / metrics are visible
- replay fixtures and regression tests are present in the repo
- blog drafts describe the navigator pivot, not a proactive companion-first product
