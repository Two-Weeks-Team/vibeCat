# VibeCat Execution — Learnings

## [2026-03-03] Session Start

### Project Context
- GCP Project: vibecat-489105 / centisgood@gmail.com
- Region: asia-northeast3
- Backend: Go 1.24+ (google.golang.org/genai + google.golang.org/adk)
- Client: Swift 6 / SwiftUI / SPM, macOS 26
- Deadline: March 16, 2026 5:00 PM PDT (13 days remaining)

### Scaffolding Already Done
- VibeCat/Package.swift (swift-tools-version 6.2, .macOS(.v26))
- VibeCat/Info.plist, VibeCat.entitlements
- Makefile, .github/workflows/ci.yml
- infra/setup.sh, infra/deploy.sh, infra/teardown.sh
- 6 character soul.md + preset.json
- Assets: cat sprites (idle/thinking/happy/surprised), TrayIcons_Clean, Music

### What Does NOT Exist Yet
- VibeCat/Sources/Core/ (no source files)
- VibeCat/Sources/VibeCat/ (no source files)
- VibeCat/Tests/VibeCatTests/ (no test files)
- backend/ directory (no Go code, no go.mod)

### GenAI Go SDK Reference
- Cached at /tmp/go-genai/ (live.go, types.go etc.)
- live.go has gorilla/websocket dependency → websocket proxy supported
- LSP errors in /tmp/go-genai/ are expected (missing go.mod context)

### Key Constraints
- ALL Gemini API calls go through backend — client never calls directly
- No API keys on client — session tokens only, key in Secret Manager
- All 9 agents required in ADK Orchestrator
- Sleeping sprite doesn't exist → idle + dimmed overlay (per user decision)
- English support required (challenge requirement)
- NEVER mention GeminiCat in committed files

## [2026-03-03T00:00:00Z] Task 2: ADK Go SDK Spike

- Correct Go module path is `google.golang.org/adk` (not `github.com/google/adk-go`), and version `v0.5.0` is published.
- Multi-agent graph support is confirmed via workflow agents: `sequentialagent.New(...)` with `agent.Config{SubAgents: []agent.Agent{...}}`.
- Agent definition pattern in Go ADK: `agent.New(agent.Config{... Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] { ... }})`.
- Graph execution pattern: create `runner.New(...)`, create session with `session.InMemoryService().Create(...)`, then iterate events from `r.Run(...)`.
- Structured outputs can be emitted as typed JSON payloads from custom agents and consumed from `event.Content`/`event.LLMResponse.Content`.
- Practical spike result: two custom agents (`vision_agent`, `memory_agent`) ran sequentially and emitted deterministic structured outputs.

## [2026-03-03 22:54:58 KST] Task 1: GenAI Live Spike

- `google.golang.org/genai` Live API is WebSocket-based in SDK implementation: `/tmp/go-genai/live.go` imports `github.com/gorilla/websocket` and dials via `websocket.DefaultDialer.Dial` inside `(*Live).Connect`.
- Live API client surface verified from source:
  - `func (r *Live) Connect(context context.Context, model string, config *LiveConnectConfig) (*Session, error)`
  - `func (s *Session) SendRealtimeInput(input LiveRealtimeInput) error`
  - `func (s *Session) Receive() (*LiveServerMessage, error)`
- Feature field mapping in current SDK (`/tmp/go-genai/types.go`):
  - proactiveAudio: `ProactivityConfig.ProactiveAudio`
  - affectiveDialog: `LiveConnectConfig.EnableAffectiveDialog`
  - outputAudioTranscription: `LiveConnectConfig.OutputAudioTranscription`
  - contextWindowCompression: `LiveConnectConfig.ContextWindowCompression`
- Spike module in `spike/genai-live` built and executed successfully (`go build . && go run .`) using `google.golang.org/genai v1.48.0`.
- `GEMINI_API_KEY` was effectively empty (count result `1`), so runtime probe used dummy key for connection-attempt behavior and still reached connect+SendRealtimeInput path in this environment.
## [2026-03-03 22:59] Task 4: Backend Init
- Gateway module: vibecat/realtime-gateway
- Orchestrator module: vibecat/adk-orchestrator
- Go version: 1.24
- Health endpoint format: {"status":"ok","service":"..."}
- Created: backend/realtime-gateway/go.mod, main.go
- Created: backend/adk-orchestrator/go.mod, main.go
- Health endpoints: /healthz, /readyz
- Logging: log/slog with JSON output
- Verified builds: go build ./... OK
- Verified endpoints: curl responds correctly

