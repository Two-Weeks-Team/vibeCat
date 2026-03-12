<p align="center">
  <img src="Assets/Sprites/cat/idle_01.png" width="120" alt="VibeCat">
</p>

<h1 align="center">VibeCat</h1>

<p align="center">
  <strong>Your Proactive Desktop Companion — an AI that sees your screen, suggests before you ask, and acts with your permission.</strong>
</p>

<p align="center">
  <a href="https://geminiliveagentchallenge.devpost.com/"><img src="https://img.shields.io/badge/Gemini_Live_Agent_Challenge-2026-4285F4?style=flat-square&logo=google&logoColor=white" alt="Challenge"></a>
  <img src="https://img.shields.io/badge/category-UI_Navigator-0F9D58?style=flat-square&logo=googlechrome&logoColor=white" alt="Category">
  <img src="https://img.shields.io/badge/platform-macOS_15%2B-000000?style=flat-square&logo=apple&logoColor=white" alt="Platform">
  <img src="https://img.shields.io/badge/Swift-6.2-F05138?style=flat-square&logo=swift&logoColor=white" alt="Swift">
  <img src="https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/GCP-Cloud_Run-4285F4?style=flat-square&logo=googlecloud&logoColor=white" alt="GCP">
</p>

---

## What Is VibeCat?

VibeCat is a **native macOS desktop companion** that watches your screen, understands your context, and **proactively suggests actions before you ask**. Unlike traditional automation tools that wait for commands, VibeCat observes your workflow and offers help — like a senior colleague sitting next to you.

### Core Flow: OBSERVE → SUGGEST → WAIT → ACT → FEEDBACK

1. **OBSERVE** — VibeCat continuously watches your screen via Gemini Live API (screenshots + accessibility tree)
2. **SUGGEST** — When it spots an opportunity, it speaks up: *"I notice a bug in that function — want me to fix it?"*
3. **WAIT** — It always waits for your confirmation before acting
4. **ACT** — Upon approval, it executes precise actions across your desktop apps
5. **FEEDBACK** — After acting, it verifies the result and reports back: *"Done! The fix is applied. Looks good?"*

### What Makes VibeCat Different

| Feature | Traditional UI Agents | VibeCat |
|---------|----------------------|---------|
| **Interaction model** | Reactive — waits for commands | **Proactive** — suggests before you ask |
| **Interface** | Text-based CLI or scripted | **Voice-first** — natural conversation via Gemini Live API |
| **Platform** | Python + cross-platform wrappers | **Native macOS Swift** — first-class citizen |
| **Architecture** | Local-only execution | **Cloud-assisted reasoning** — Cloud Run + ADK |
| **Error handling** | Fails and reports | **Self-healing** — retries with alternative strategies |
| **Verification** | Basic state check | **Triple-source grounding** — Accessibility + CDP + Vision |
| **Transparency** | Silent processing | **Real-time feedback** — narrates every step |

## Demo Scenarios

VibeCat is optimized for three gold-tier surfaces:

### 1. Music Suggestion (Chrome/YouTube)
> *"You've been coding for a while — want me to put on some music?"*
> → User: *"Sure!"*
> → VibeCat opens YouTube, searches for focus music, starts playback
> → *"There you go! Let me know if you want something different."*

### 2. Code Fix Suggestion (Antigravity IDE)
> *"I see a potential null check missing in that function — should I add it?"*
> → User: *"Yeah, go ahead"*
> → VibeCat triggers inline edit, inserts the fix
> → *"Done! The null check is in place. Want me to run the tests?"*

### 3. Terminal Command Improvement (Terminal/iTerm2)
> *"That ls could show more detail with -la — want me to rerun it?"*
> → User: *"Do it"*
> → VibeCat types and executes the improved command
> → *"Here's the detailed listing. Notice the hidden files now showing?"*

## Architecture

```mermaid
flowchart TB
    subgraph Client["macOS Client (Swift 6)"]
        UI["Overlay UI + Cat Character"]
        AX["Accessibility Navigator<br/>80+ key codes"]
        SC["Screen Capture"]
        OV["Navigator Overlay Panel<br/>action · grounding badge · progress"]
    end

    subgraph Gateway["Realtime Gateway (Go · Cloud Run)"]
        WS["WebSocket Handler"]
        FC["Function Calling<br/>5 navigator tools"]
        PFC["pendingFC Mechanism<br/>sequential step execution"]
        SH["Self-Healing Engine<br/>max 2 retries"]
        VV["Vision Verification"]
        CDP["Chrome DevTools Protocol<br/>chromedp"]
        TF["Transparent Feedback<br/>processingState pipeline"]
    end

    subgraph GCP["Google Cloud Platform"]
        GM["Gemini Live API<br/>voice + vision + FC"]
        ADK["ADK Orchestrator<br/>escalation + analysis"]
        FS["Firestore"]
        CL["Cloud Logging / Trace"]
    end

    UI -->|voice + screen capture| WS
    WS -->|Gemini Live session| GM
    GM -->|function calls| FC
    FC --> PFC
    PFC -->|one step at a time| AX
    PFC -->|browser actions| CDP
    SH -->|retry on failure| PFC
    VV -->|screenshot analysis| ADK
    TF -->|status messages| OV
    WS --> FS
    WS --> CL
```

### Component Overview

| Layer | Technology | Role |
|-------|-----------|------|
| **macOS Client** | Swift 6 / AppKit | Screen capture, AX executor (80+ key codes), navigator overlay with grounding badges, voice transport |
| **Realtime Gateway** | Go 1.24 / Cloud Run | WebSocket handler, Proactive Companion prompt, 5 FC tool handlers, pendingFC sequential execution, self-healing (max 2 retries), vision verification, transparent feedback pipeline |
| **Gemini Live API** | Google GenAI SDK v1.48 | Real-time multimodal conversation (voice + vision), function calling, session resumption, VAD |
| **ADK Orchestrator** | Go / Cloud Run | Confidence escalation (`/navigator/escalate`), vision verification, visible-text extraction, async summary/memory/replay (`/navigator/background`) |
| **Chrome Controller** | chromedp v0.11.3 (CDP) | Click, Type, Navigate, Scroll, Screenshot, Close — lazy connect with graceful fallback |

### Proactive Companion System Prompt

The gateway injects a **Proactive Companion** system prompt into every Gemini Live session. This prompt defines VibeCat's core identity — an attentive colleague, not a passive tool:

- **Proactive observation**: notices errors, long work sessions, inefficient commands, missing code
- **Natural suggestion**: *"You've been coding for a while — want me to play some music?"*
- **Confirmation gate**: always waits for user approval before acting
- **Friendly feedback**: *"Done! How does that look?"* after every action
- **Emotion tags**: `[happy]`, `[thinking]`, `[concerned]`, etc. for character expression
- **Language matching**: responds in the user's language (Korean, English, Japanese)

### Navigator Tools (Function Calling)

VibeCat registers 5 tools with Gemini via `navigatorToolDeclarations()` for precise desktop control:

| Tool | Parameters | Purpose | Example |
|------|-----------|---------|---------|
| `navigate_text_entry` | `text`, `target`, `submit` | Type text into focused field | Search queries, code snippets, form input |
| `navigate_hotkey` | `keys[]`, `target` | Send keyboard shortcuts | `Cmd+S`, `Space` (YouTube play/pause), `Cmd+I` (IDE inline) |
| `navigate_focus_app` | `app` | Switch to a specific application | Open Chrome, switch to Terminal, focus Antigravity |
| `navigate_open_url` | `url` | Open a URL in the default browser | YouTube links, documentation pages |
| `navigate_type_and_submit` | `text`, `submit` | Type text and press Enter | Terminal commands (`ls -la`), search submissions |

Each tool has a dedicated handler in the gateway: `handleNavigateHotkeyToolCall`, `handleNavigateFocusAppToolCall`, `handleNavigateOpenURLToolCall`, `handleNavigateTypeAndSubmitToolCall`.

### pendingFC: Sequential Multi-Step Execution

Complex actions (e.g., "open YouTube and search for music") require multiple steps. The **pendingFC mechanism** ensures steps execute one at a time:

1. Gateway receives Gemini's function call → queues all steps in `pendingFCSteps`
2. Sends **only the first step** to the client
3. Client executes and confirms → gateway sends next step via `advancePendingFC()`
4. Repeat until all steps complete or self-healing exhausts retries
5. `clearPendingFC()` resets all state (8 fields + retry counter) on completion

This prevents race conditions in multi-step workflows and enables per-step retry/verification.

### Self-Healing Navigation

When an action fails, VibeCat retries with alternative grounding strategies (max 2 retries per step via `stepRetryCount`):

1. **First attempt** — Execute via primary method (AX tree or hotkey)
2. **Retry 1** — Try alternative grounding source (CDP for browser elements, different AX path)
3. **Retry 2** — Fall back to vision-based coordinate targeting via ADK screenshot analysis
4. **Each retry** — `incrementStepRetry()` tracks count; vision verification confirms success before proceeding
5. **Exhausted** — Graceful fallback to guided mode or human explanation

### Vision Verification

After executing risky or complex actions, VibeCat verifies success through ADK screenshot analysis:

1. Gateway requests a screenshot from the client (`requestScreenCapture`)
2. Screenshot is sent to ADK Orchestrator (`POST /navigator/escalate`)
3. ADK's Gemini-powered vision agent analyzes the post-action state
4. Result feeds back into the step pipeline — success advances, failure triggers self-healing retry

### Grounding Sources

VibeCat uses **triple-source grounding** to prevent blind clicking:

| Source | Badge | Technology | Use Case |
|--------|-------|-----------|----------|
| **Accessibility** | AX (blue) | macOS Accessibility API | Native UI element discovery and manipulation |
| **CDP** | CDP (orange) | Chrome DevTools Protocol (chromedp) | Precise browser element interaction |
| **Vision** | Vision (purple) | Gemini/ADK screenshot analysis | Visual verification and coordinate targeting |
| **Keyboard** | ⌨ (green) | CGEvent key injection (80+ codes) | App-specific hotkeys (YouTube, IDE, Terminal) |

### Transparent Feedback Pipeline

VibeCat narrates every processing step in real-time via `processingState` messages — no silent processing:

| Stage | English | Korean | When |
|-------|---------|--------|------|
| `analyzing_command` | Analyzing command... | 명령 분석 중... | FC received from Gemini |
| `planning_steps` | Planning steps... | 실행 계획 중... | Multi-step plan created |
| `executing_step` | Executing action... | 실행 중... | Step sent to client |
| `verifying_result` | Verifying result... | 결과 확인 중... | Post-action screenshot check |
| `retrying_step` | Retrying with alternative... | 재시도 중... | Self-healing retry triggered |
| `completing` | Completing task... | 작업 완료 중... | Final step succeeded |
| `observing_screen` | Observing screen... | 화면 관찰 중... | Proactive screen analysis |

The client displays these as real-time status bubbles in the navigator overlay panel.

### Execution Contract

The full execution pipeline:

1. VibeCat **proactively observes** the user's screen via Gemini Live (screenshots + AX context)
2. Gemini identifies opportunities and **suggests an action** via voice
3. User confirms ("yeah, go ahead") or declines
4. Gateway receives Gemini's function call (one of 5 tools)
5. **Transparent feedback**: `processingState` streams to client at each stage
6. **pendingFC** queues multi-step plans and sends one step at a time
7. macOS client executes the step via AX, CDP, or keyboard
8. **Self-healing**: on failure, retries up to 2 times with alternative grounding source
9. **Vision verification**: ADK analyzes post-action screenshot to confirm success
10. On success → next step (if any) or completion with voice feedback
11. On persistent failure → graceful fallback to guided mode or human explanation
12. Completed tasks enqueue async summary/replay/memory work off the hot path

## Quick Start

### Prerequisites

- macOS 15+ (Sequoia)
- Xcode 16+ (for local client builds)
- Go 1.24+
- A Google Cloud project with:
  - Gemini API key
  - Cloud Run enabled
  - Firestore database
  - Secret Manager configured

### Build & Test

```bash
# Swift client
cd VibeCat
swift build
swift test          # 91 tests

# Go gateway
cd backend/realtime-gateway
go build ./...
go test ./...       # all packages pass
go vet ./...

# Go orchestrator
cd backend/adk-orchestrator
go build ./...
go test ./...
```

### Deploy to Cloud Run

```bash
# Deploy gateway
./infra/deploy.sh gateway

# Deploy orchestrator
./infra/deploy.sh orchestrator
```

### Runtime Permissions

VibeCat requires three macOS permissions:

| Permission | Required For |
|-----------|-------------|
| **Screen Recording** | Capturing screen content for Gemini vision analysis |
| **Accessibility** | Reading UI elements and executing navigation actions |
| **Microphone** | Voice input for Gemini Live API conversation |

## Project Structure

```text
vibeCat/
├── VibeCat/                          # Swift package: Core + macOS app + tests
│   ├── Sources/Core/                 # UI-free models, localization, parsers
│   ├── Sources/VibeCat/              # AppKit app, AX navigator, overlay UI
│   └── Tests/VibeCatTests/           # 91 package tests
├── backend/realtime-gateway/         # Go: WebSocket gateway, FC tools, self-healing
│   └── internal/
│       ├── ws/                       # Handler, navigator, metrics
│       ├── live/                     # Gemini Live session management
│       └── cdp/                      # Chrome DevTools Protocol controller
├── backend/adk-orchestrator/         # Go: ADK graph, escalation, memory/replay
├── tests/e2e/                        # Deployed smoke and live-path tests
├── infra/                            # GCP bootstrap, deploy scripts, observability
├── docs/                             # Architecture, status, evidence, research
└── Assets/                           # Sprites, tray icons, audio samples
```

## Deployment

| Service | Region | Technology | URL |
|---------|--------|-----------|-----|
| Realtime Gateway | asia-northeast3 | Go / Cloud Run | `realtime-gateway-163070481841.asia-northeast3.run.app` |
| ADK Orchestrator | asia-northeast3 | Go / Cloud Run | `adk-orchestrator-163070481841.asia-northeast3.run.app` |
| Firestore | `(default)` | Native database | — |

**GCP Project**: `vibecat-489105` · **Infrastructure**: Cloud Run, Firestore, Secret Manager, Cloud Logging, Cloud Trace, Cloud Monitoring

## Gold-Tier Surfaces

Submission-critical reliability is concentrated on three surfaces:

| Surface | Capabilities | Key Shortcuts |
|---------|-------------|---------------|
| **Antigravity IDE** | Code editing, inline fixes, symbol navigation | `Cmd+P` (file picker), `Cmd+Shift+O` (symbols), `Cmd+I` (inline prompt) |
| **Terminal / iTerm2** | Command execution, output interpretation | `Cmd+T` (new tab), type + Enter |
| **Chrome** | URL navigation, YouTube playback, search, form filling | `Space` (play/pause), `F` (fullscreen), `Shift+N` (next) |

## Safety Model

VibeCat uses **safe-immediate execution** with mandatory confirmation for proactive suggestions:

- Proactive suggestions **always wait** for user confirmation before acting
- Low-risk, well-targeted steps may execute immediately after confirmation
- Ambiguous intent never auto-executes
- Low-confidence targets downgrade to clarification or guided mode

**Immediate (low-risk):** focus changes, page navigation, search entry, tab switching, hotkeys

**Confirmation required:** passwords/tokens, deploy/publish/send, destructive shell commands, `git push`, bulk code insertion

## Technology Stack

| Technology | Version | Role |
|-----------|---------|------|
| **Gemini Live API** | via GenAI SDK v1.48 | Real-time multimodal conversation (voice + vision + FC) |
| **Gemini Function Calling** | 5 tools registered | Structured tool invocation for desktop actions |
| **ADK (Agent Development Kit)** | Go / Cloud Run | Confidence escalation, vision verification, memory/replay |
| **Google Cloud Run** | asia-northeast3 | Serverless backend hosting (gateway + orchestrator) |
| **chromedp** | v0.11.3 | Go-native Chrome DevTools Protocol client |
| **macOS Accessibility API** | AppKit / AX | Native UI element discovery and action execution |
| **Swift** | 6.2 | macOS client with 91 package tests |
| **Go** | 1.24+ | Backend services with full test/vet coverage |
| **Firestore** | Native mode | Action state persistence, session memory, replay fixtures |
| **Cloud Logging / Trace** | GCP | Navigator telemetry, self-healing metrics, processingState transitions |

## Observability

The navigator path emits proof-oriented telemetry:

- Task acceptance, clarification prompts, replacement prompts
- Time to first action (`time_to_first_action_ms`)
- Guided-mode outcomes, verification failures
- Input-field focus success/failure, wrong-target detections
- **Self-healing retry counts and outcomes**
- **Vision verification results**
- **processingState stage transitions**

These feed Cloud Logging, Cloud Trace, and Cloud Monitoring. Completed task replays are persisted in Firestore for regression comparison.

## Submission Assets

| Asset | Location |
|-------|----------|
| Architecture diagram | This README (Mermaid) + `docs/FINAL_ARCHITECTURE.md` |
| Demo video script | `docs/DEMO_VIDEO_SCRIPT.md` |
| Devpost submission text | `docs/DEVPOST_SUBMISSION.md` |
| Current status | `docs/CURRENT_STATUS_20260312.md` |
| Deployment evidence | `docs/evidence/DEPLOYMENT_EVIDENCE.md` |
| GCP proof | `docs/deployment/PROOF_OF_GCP_DEPLOYMENT.md` |
| Architecture research | `docs/AGENT_ARCHITECTURE_RESEARCH_20260312.md` |

### Dev.to Blog Posts

15 published articles documenting the development journey:

- [the moment vibecat stopped waiting and started suggesting](https://dev.to/combba/the-moment-vibecat-stopped-waiting-and-started-suggesting-4ek) — Proactive Companion pivot
- [six characters, one soul](https://dev.to/combba/six-characters-one-soul-5008) — Character system architecture
- Full series: [dev.to/combba](https://dev.to/combba)

## License

This project is submitted to the [Gemini Live Agent Challenge](https://geminiliveagentchallenge.devpost.com/) (2026).
