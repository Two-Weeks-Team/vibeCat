# PROJECT KNOWLEDGE BASE

**Generated:** 2026-03-03
**Branch:** master (no commits)

## OVERVIEW

VibeCat is a macOS desktop companion for solo developers — an animated cat that watches your screen, hears your voice, remembers context across sessions, and proactively helps. Built for the Gemini Live Agent Challenge using GenAI SDK, Google ADK, Gemini Live API, and VAD. Project scaffolding (Package.swift, Makefile, CI, infra scripts) is in place; source implementation has not yet started.

## STRUCTURE

```
vibeCat/
├── Assets/
│   ├── Sprites/          # 6 characters (cat, derpy, jinwoo, kimjongun, saja, trump), ~97 PNGs
│   ├── TrayIcons/        # Raw tray icons (48 PNGs)
│   ├── TrayIcons_Clean/  # Production tray icons (24 PNGs, 8 frames x 3 scales)
│   ├── Music/            # Background music (2 audio files)
│   └── SPRITE_LICENSE.md
├── docs/
│   ├── PRD/
│   │   ├── LIVE_AGENTS_PRD.md   # Master PRD — start here
│   │   ├── INDEX.md             # Document map
│   │   └── DETAILS/             # 13 detailed spec documents
│   ├── reference/               # External SDK/GCP/ADK reference docs (~40 files)
│   └── ideas/                   # Gitignored local design notes
├── voice_samples/               # 13 voice sample files (AIFF/WAV)
└── .gitignore
```

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Understand the product | `docs/PRD/LIVE_AGENTS_PRD.md` | Business model, agent philosophy, architecture diagram |
| Find all spec documents | `docs/PRD/INDEX.md` | Complete document map |
| Implementation order | `docs/PRD/DETAILS/IMPLEMENTATION_EXECUTION_PLAN.md` | Build sequence, module inventory, dependency rules |
| Backend architecture | `docs/PRD/DETAILS/BACKEND_ARCHITECTURE.md` | 9-agent graph, Firestore schema, service contracts (413 lines) |
| Client-backend protocol | `docs/PRD/DETAILS/CLIENT_BACKEND_PROTOCOL.md` | WebSocket/REST spec, message types, error codes (281 lines) |
| Implementation tasks | `docs/PRD/DETAILS/END_TO_END_IMPLEMENTATION_TASKS.md` | Task-by-task guide |
| Backend tasks | `docs/PRD/DETAILS/BACKEND_IMPLEMENTATION_TASKS.md` | Backend-specific tasks T-100 to T-146 |
| TDD plan | `docs/PRD/DETAILS/TDD_VERIFICATION_PLAN.md` | Red-Green-Refactor order |
| Asset inventory | `docs/PRD/DETAILS/ASSET_MIGRATION_PLAN.md` | Required assets, counts, verification |
| Deployment | `docs/PRD/DETAILS/DEPLOYMENT_AND_OPERATIONS.md` | GCP Cloud Run, Firestore, observability (215 lines) |
| Architecture migration | `docs/ideas/GEMINICAT_TO_VIBECAT_MAPPING.md` | Monolithic→split mapping (gitignored) |
| SDK reference | `docs/reference/` | Gemini, ADK, GCP official docs |
| Character presets | `Assets/Sprites/{name}/preset.json` | Voice + prompt profile per character |

## TARGET STRUCTURE (Post-Implementation)

```
vibeCat/
├── VibeCat/
│   ├── Package.swift          # SPM manifest (swift-tools-version 6.2)
│   ├── Sources/Core/          # Pure Swift modules (no UI deps)
│   ├── Sources/VibeCat/       # macOS app (UI, capture, transport)
│   └── Tests/VibeCatTests/
├── backend/
│   ├── realtime-gateway/      # Cloud Run: Go + GenAI SDK (Live API WebSocket proxy)
│   └── adk-orchestrator/      # Cloud Run: Go + ADK Go SDK (9-agent graph)
├── infra/                     # IaC deployment scripts
├── Assets/
│   └── Sprites/{name}/
│       ├── preset.json        # Voice, size, persona config
│       └── soul.md            # Character personality prompt
├── docs/
└── voice_samples/
```

## ARCHITECTURE

### Three-Layer Split (Non-Negotiable)

1. **macOS Swift Client** — UI, screen capture, audio playback, gestures, local settings
2. **Realtime Gateway** (Cloud Run) — Go + `google.golang.org/genai` — WebSocket proxy to Gemini Live API
3. **ADK Orchestrator** (Cloud Run) — Go + `google.golang.org/adk` — 9-agent graph

### 9 Agents

| Agent | Role | Where |
|-------|------|-------|
| VAD | Natural conversation, barge-in | Gemini Live API config |
| VisionAgent | Screen capture analysis | ADK Orchestrator |
| Mediator | Speech gating, cooldown | ADK Orchestrator |
| AdaptiveScheduler | Timing adjustments | ADK Orchestrator |
| EngagementAgent | Proactive triggers | ADK Orchestrator |
| MemoryAgent | Cross-session context | ADK Orchestrator + Firestore |
| MoodDetector | Frustration sensing | ADK Orchestrator |
| CelebrationTrigger | Success detection | ADK Orchestrator |
| SearchBuddy | Google Search grounding | ADK Orchestrator |

### Key Protocols

- Client↔Gateway: WebSocket (`wss://{host}/ws/live`) + REST (`/api/v1/`)
- Gateway↔Orchestrator: HTTP `POST /analyze`
- Audio format: PCM 16kHz 16-bit mono (client→server), PCM 24kHz (server→client)
- Auth: Ephemeral tokens, API key stored in GCP Secret Manager (never on client)

## CONVENTIONS

- **Mandatory stack**: GenAI SDK + ADK + Gemini Live API + VAD — all four required
- **Client/backend separation**: ALL model calls from backend. Client does UI/capture/playback only
- **Core module**: Must not depend on app-layer UI modules
- **Transport**: Must be testable with mock server interfaces
- **Orchestration**: Depends on typed contracts, not concrete UI views
- **Asset copy**: Exact local files, no runtime downloads. Preserve directory names and frame naming
- **GCP region**: `asia-northeast3`
- **Backend port**: 8080 for both Cloud Run services

## ANTI-PATTERNS (THIS PROJECT)

- Client making direct Gemini API calls — ALL API calls go through backend
- Storing Gemini API key on client — use session tokens, key lives in Secret Manager
- Monolithic architecture — must be client/backend split per challenge rules
- Skipping any of the 9 agents — all are required
- Ignoring VAD config — `automaticActivityDetection` must be enabled
- Moving `PromptBuilder` to client — prompt logic is server-side only

## CHARACTERS

6 sprite characters, each with `preset.json` (voice + persona config) and `soul.md` (personality prompt):

| Character | Voice | Persona | Tone |
|-----------|-------|---------|------|
| `cat` | Zephyr | Curious beginner companion | bright, casual |
| `derpy` | Puck | Goofy accidental debugger | goofy, clumsy |
| `jinwoo` | Kore | Silent senior engineer | low-calm, concise |
| `kimjongun` | Schedar | Supreme debugger (comedy) | authoritative-warm |
| `saja` | Zubenelgenubi | Zen mentor from folklore | calm-deep, archaic |
| `trump` | Fenrir | Bombastic hype-man (comedy) | energetic-superlative |

Each has ~16 sprite frames as PNGs. Character persona is injected server-side via `soul.md` content.

## COMMANDS

```bash
# Swift Client (via Makefile)
make build        # Build Swift package
make sign         # Codesign for dev
make run          # Build + sign + run
make test         # Run Swift tests

# Backend (Go — local)
cd backend/realtime-gateway && go run .
cd backend/adk-orchestrator && go run .

# Backend (Go — tests)
cd backend/realtime-gateway && go test ./...
cd backend/adk-orchestrator && go test ./...

# Docker
make docker-build  # Build both backend images

# Deploy (Cloud Run)
./infra/deploy.sh     # Deploy both services to asia-northeast3
./infra/teardown.sh   # Remove deployment
```

## NOTES

- `docs/ideas/` is gitignored — contains local-only design notes (never commit)
- `.cursor/rules/` contains Taskmaster workflow rules (from cursor config, not committed)
- 6 characters share the same animation system, differ only in sprites and voice preset
- Backend stack: **Go** — both Gateway (`google.golang.org/genai`) and Orchestrator (`google.golang.org/adk`)
- GCP Project: `vibecat-489105`, Account: `centisgood@gmail.com`
- Challenge submission requires deployment evidence, demo video, and operational proof artifacts
