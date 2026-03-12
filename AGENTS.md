# PROJECT KNOWLEDGE BASE

**Generated:** 2026-03-12
**Branch:** `codex/navigator-stabilization-20260312`
**HEAD:** `c1f35cc`

## OVERVIEW

VibeCat is a **Proactive Desktop Companion** — a native macOS AI that watches your screen, suggests actions before you ask, and acts with your permission. Built for the Gemini Live Agent Challenge (UI Navigator category).

Core identity: **OBSERVE → SUGGEST → WAIT → ACT → FEEDBACK**

The repo is a mixed Swift + Go workspace: the macOS client lives in `VibeCat/`, the Cloud Run services live in `backend/`, deployed smoke coverage lives in `tests/e2e/`, and deploy/observability scripts live in `infra/`.

Prefer the closest local `AGENTS.md` before relying on this root file.

## STRUCTURE

```text
vibeCat/
|-- VibeCat/                    # Swift package: Core + macOS app + tests
|-- backend/realtime-gateway/   # WebSocket gateway, FC tools, self-healing, CDP
|-- backend/adk-orchestrator/   # ADK graph, navigator escalation, memory/replay
|-- tests/e2e/                  # deployed smoke and live-path tests
|-- infra/                      # GCP bootstrap, deploy, observability
|-- docs/                       # current proof docs + historical analysis
|-- Assets/                     # sprites, tray icons, music
`-- voice_samples/              # sample audio assets
```

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Current submission truth | `README.md` | Proactive Companion framing |
| Current repo snapshot | `docs/CURRENT_STATUS_20260312.md` | Best high-level status doc |
| Runtime architecture | `docs/FINAL_ARCHITECTURE.md` | Current navigator flow |
| Swift client work | `VibeCat/AGENTS.md` | Local Swift/package guidance |
| Gateway work | `backend/realtime-gateway/AGENTS.md` | Local planner/runtime guidance |
| Orchestrator work | `backend/adk-orchestrator/AGENTS.md` | Local ADK/escalator guidance |
| Deployed smoke tests | `tests/e2e/AGENTS.md` | Env-driven live test rules |
| GCP scripts | `infra/AGENTS.md` | Bootstrap/deploy/observability |
| Documentation updates | `docs/AGENTS.md` | Current-vs-historical doc boundaries |
| Architecture research | `docs/AGENT_ARCHITECTURE_RESEARCH_20260312.md` | 5-option analysis + JAYU comparison |

## KEY COMPONENTS

| Component | Location | Description |
|-----------|----------|-------------|
| Proactive Companion prompt | `backend/realtime-gateway/internal/live/session.go` | System prompt + 5 FC tool declarations |
| FC tool handlers | `backend/realtime-gateway/internal/ws/handler.go` | pendingFC mechanism, self-healing, vision verification |
| Chrome controller | `backend/realtime-gateway/internal/cdp/chrome.go` | chromedp CDP for browser automation |
| Navigator overlay | `VibeCat/Sources/VibeCat/NavigatorOverlayPanel.swift` | Floating HUD with grounding badges |
| Key mapping | `VibeCat/Sources/VibeCat/AccessibilityNavigator.swift` | 80+ key codes for full keyboard control |

## CONVENTIONS

- Submission track remains `UI Navigator`; older companion-era docs are archival unless explicitly rewritten.
- Core identity is **Proactive Companion**: suggest before asked, confirm before acting, verify after acting.
- All model access stays server-side; the client owns UI, capture, transport, playback, and local execution only.
- `VibeCat/Sources/Core/` must stay UI-free.
- Production backend port remains `8080`; GCP region remains `asia-northeast3`.
- Gold-tier surfaces are `Antigravity IDE`, `Terminal`, and `Chrome`.
- Navigator uses 5 FC tools: `text_entry`, `hotkey`, `focus_app`, `open_url`, `type_and_submit`.
- Self-healing retries max 2 times with alternative grounding sources before failing.
- Vision verification via ADK screenshot analysis confirms action success.

## ANTI-PATTERNS

- direct client-to-Gemini calls
- client-side storage of Gemini API credentials
- moving prompt or agent logic back into the macOS client
- treating historical planning docs as proof of current implementation
- adding local guidance files under static asset trees
- sending multiple FC steps simultaneously (use pendingFC sequential mechanism)
- skipping vision verification for risky or complex actions

## COMMANDS

```bash
git status --short --branch
git log -1 --oneline

cd VibeCat && swift build && swift test
cd backend/realtime-gateway && go build ./... && go test ./... && go vet ./...
cd backend/adk-orchestrator && go build ./... && go test ./... && go vet ./...
cd tests/e2e && go test -v -count=1 ./...
```

## NOTES

- CI lives in `.github/workflows/ci.yml` and `.github/workflows/cd.yml`.
- `docs/evidence/` and `docs/deployment/` are the proof-oriented documentation paths.
- `Assets/` and `voice_samples/` are reference assets, not primary implementation surfaces.
