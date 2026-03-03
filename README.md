# VibeCat

A macOS desktop companion for solo developers — an animated character that watches your screen, hears your voice, remembers context across sessions, and proactively helps.

Built for the [Gemini Live Agent Challenge](https://geminiliveagentchallenge.devpost.com/) using GenAI SDK, Google ADK, Gemini Live API, and VAD.

## Architecture

| Layer | Technology | Location |
|-------|-----------|----------|
| macOS Client | Swift 6 / SwiftUI | `VibeCat/` |
| Realtime Gateway | Go + GenAI SDK | `backend/realtime-gateway/` |
| ADK Orchestrator | Go + ADK Go SDK | `backend/adk-orchestrator/` |
| Persistence | Firestore | GCP `asia-northeast3` |

## Quick Start

### Prerequisites
- macOS 15.0+, Xcode 16+
- Go 1.24+
- GCP project with Firestore, Secret Manager, Cloud Run enabled
- Gemini API key stored in Secret Manager as `vibecat-gemini-api-key`

### Build & Run (Client)
```bash
make build   # Build Swift package
make sign    # Codesign for dev
make run     # Build + sign + run
make test    # Run tests
```

### Build & Run (Backend — Local)
```bash
cd backend/realtime-gateway && go run .
cd backend/adk-orchestrator && go run .
```

### Deploy (Cloud Run)
```bash
./infra/deploy.sh    # Deploy both services
./infra/teardown.sh  # Remove deployment
```

## 9 Agents

| Agent | Role |
|-------|------|
| VAD | Natural conversation with barge-in |
| VisionAgent | Screen capture analysis |
| Mediator | Speech gating and cooldown |
| AdaptiveScheduler | Timing adjustments |
| EngagementAgent | Proactive triggers |
| MemoryAgent | Cross-session context |
| MoodDetector | Frustration sensing |
| CelebrationTrigger | Success detection |
| SearchBuddy | Google Search grounding |

## Characters

6 animated characters with unique voices and personalities. Each has a `soul.md` defining their persona:

| Character | Role | Voice |
|-----------|------|-------|
| `cat` | Curious beginner companion | Zephyr |
| `derpy` | Goofy accidental debugger | Puck |
| `jinwoo` | Silent senior engineer | Kore |
| `kimjongun` | Supreme debugger (comedy) | Schedar |
| `saja` | Zen mentor from folklore | Zubenelgenubi |
| `trump` | Bombastic hype-man (comedy) | Fenrir |

## License

[TODO: Add license]
