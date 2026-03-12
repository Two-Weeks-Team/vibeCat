# PROJECT KNOWLEDGE BASE

**Generated:** 2026-03-12
**Branch:** `codex/navigator-stabilization-20260312`
**HEAD:** `775b97f`

## OVERVIEW

VibeCat is a macOS desktop UI navigator for developer workflows. The repo is a mixed Swift + Go workspace: the macOS client lives in `VibeCat/`, the Cloud Run services live in `backend/`, deployed smoke coverage lives in `tests/e2e/`, and deploy/observability scripts live in `infra/`.

Prefer the closest local `AGENTS.md` before relying on this root file.

## STRUCTURE

```text
vibeCat/
|-- VibeCat/                    # Swift package: Core + macOS app + tests
|-- backend/realtime-gateway/   # WebSocket gateway, intent/risk/step planner
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
| Current submission truth | `README.md` | Current product framing |
| Current repo snapshot | `docs/CURRENT_STATUS_20260311.md` | Best high-level status doc |
| Runtime architecture | `docs/FINAL_ARCHITECTURE.md` | Current navigator flow |
| Swift client work | `VibeCat/AGENTS.md` | Local Swift/package guidance |
| Gateway work | `backend/realtime-gateway/AGENTS.md` | Local planner/runtime guidance |
| Orchestrator work | `backend/adk-orchestrator/AGENTS.md` | Local ADK/escalator guidance |
| Deployed smoke tests | `tests/e2e/AGENTS.md` | Env-driven live test rules |
| GCP scripts | `infra/AGENTS.md` | Bootstrap/deploy/observability |
| Documentation updates | `docs/AGENTS.md` | Current-vs-historical doc boundaries |

## CONVENTIONS

- Submission track remains `UI Navigator`; older companion-era docs are archival unless explicitly rewritten.
- All model access stays server-side; the client owns UI, capture, transport, playback, and local execution only.
- `VibeCat/Sources/Core/` must stay UI-free.
- Production backend port remains `8080`; GCP region remains `asia-northeast3`.
- Gold-tier surfaces are `Antigravity IDE`, `Terminal`, and `Chrome`.

## ANTI-PATTERNS

- direct client-to-Gemini calls
- client-side storage of Gemini API credentials
- moving prompt or agent logic back into the macOS client
- treating historical planning docs as proof of current implementation
- adding local guidance files under static asset trees

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
