# VibeCat — Devpost Submission
## Gemini Live Agent Challenge 2026 · UI Navigator Category

---

## Inspiration

Every developer knows the pain: you're deep in a flow state, and suddenly you need to look something up, fix a typo in a terminal command, or find that one Stack Overflow answer. The context switch costs you minutes of mental re-entry time. We kept asking ourselves — what if your desktop had a senior colleague sitting right next to you, watching your screen, and quietly saying *"hey, want me to handle that?"* before you even had to ask?

That question became VibeCat. Most AI tools are reactive — they wait for you to type a command. We wanted to flip that model entirely. A truly proactive companion doesn't wait; it observes, recognizes opportunity, and offers help. Gemini Live API's real-time multimodal capabilities made this vision technically achievable for the first time: an AI that can simultaneously see your screen, hear your voice, and speak back — all in real time.

---

## What It Does

VibeCat is a **Proactive Desktop Companion** for macOS. Its core loop is **OBSERVE → SUGGEST → WAIT → ACT → FEEDBACK**. It continuously watches your screen via Gemini Live API, and when it spots an opportunity — a fixable bug, a long command that could be improved, a YouTube search you're about to type — it speaks up first: *"I notice a null check missing in that function — want me to add it?"*

Critically, VibeCat always waits for your confirmation before touching anything. Once you say yes, it executes precise desktop actions across three gold-tier surfaces: Antigravity IDE, Terminal, and Chrome. It uses 5 Function Calling tools (`text_entry`, `hotkey`, `focus_app`, `open_url`, `type_and_submit`) for structured, verifiable control. If an action fails, a self-healing engine retries up to 2 times with alternative grounding strategies. After every action, vision verification via ADK screenshot analysis confirms the result before VibeCat reports back.

---

## How We Built It

**macOS Client (Swift 6.2 + AppKit):** The native client handles screen capture, microphone input, voice playback, and local action execution. We mapped 80+ key codes in `AccessibilityNavigator.swift` for full keyboard control across apps. A floating `NavigatorOverlayPanel` HUD shows the current action, grounding source badge, and progress in real time — so users always know what VibeCat is doing and why.

**Realtime Gateway (Go 1.26.1, Cloud Run):** A WebSocket server bridges the Swift client to Gemini Live API. The gateway hosts all 5 FC tool handlers and the `pendingFC` sequential execution mechanism — ensuring multi-step actions never race. A `chromedp`-based Chrome controller handles browser automation via CDP for elements invisible to the Accessibility tree (like canvas-rendered YouTube controls).

**Self-Healing + Vision Verification:** On failure, the gateway retries with an alternative grounding source (AX → CDP → vision coordinates). After each action, it calls the ADK Orchestrator to analyze a fresh screenshot and confirm success before proceeding. All model logic stays server-side; the client owns only UI, capture, transport, and local execution.

**Cloud Infrastructure:** Cloud Run (asia-northeast3), Firestore for session state, Secret Manager for credentials, Cloud Logging and Trace for observability.

---

## Challenges We Ran Into

**Canvas-rendered browser controls:** YouTube's player controls are drawn on a `<canvas>` element — completely invisible to the macOS Accessibility tree. We solved this by routing YouTube playback through hotkey-based control (`Space` for play/pause, `k` for toggle) rather than element targeting, with CDP as a fallback for other browser interactions.

**Multi-step FC race conditions:** Gemini can issue multiple function calls in rapid succession. Naively executing them in parallel caused state corruption — a `focus_app` call would race with a `text_entry` call targeting the wrong window. We solved this with the `pendingFC` mechanism: each FC result is queued and executed strictly sequentially, with vision verification gating each step.

**High-DPI coordinate misalignment:** Screenshot coordinates from Gemini's vision analysis didn't map correctly to AX tree coordinates on Retina displays. We solved this using native Swift coordinate APIs that account for display scale factors, and by preferring semantic AX element targeting over raw pixel coordinates wherever possible.

**Proactive without being annoying:** An AI that speaks up constantly is worse than one that stays silent. We tuned the suggestion threshold and added a mandatory confirmation gate — VibeCat never acts without explicit user approval, which also makes the proactive model feel safe rather than intrusive.

---

## Accomplishments That We're Proud Of

**VibeCat speaks first.** Proactive suggestion is a fundamentally different UX from reactive command agents — and it required rethinking the entire interaction model, not just adding a feature. Getting that feel right, where suggestions land as helpful rather than annoying, was the hardest design problem we solved.

**Self-healing navigation that actually recovers.** Most automation tools fail and stop. VibeCat fails, diagnoses, retries with a different strategy, and verifies the result — all transparently narrated to the user. Watching it recover from a failed AX lookup by switching to CDP in real time is genuinely satisfying.

**Triple-source grounding eliminates blind clicking.** By combining Accessibility API, Chrome DevTools Protocol, and Gemini vision analysis, VibeCat always knows *why* it's targeting something — not just where. This makes actions reliable across apps that expose their UI very differently.

**Native macOS integration.** VibeCat feels like a real desktop app — not a web wrapper or a Python script with a thin GUI. Swift 6.2 + AppKit gives it first-class system integration, proper permission handling, and a UI that respects macOS conventions.

---

## What We Learned

Gemini's Function Calling with Live API is a genuinely powerful primitive for real-time agent architectures. The combination of streaming audio, vision, and structured tool invocation in a single session — with sub-second latency — enables interaction patterns that simply weren't possible before.

macOS Accessibility APIs are incredibly capable, but every app exposes them differently. Chrome, Terminal, and Antigravity IDE each required custom handling; there's no universal "click this button" abstraction. Building reliable cross-app navigation means understanding each app's AX tree structure individually.

The biggest UX improvement we made wasn't a technical one — it was adding transparent, real-time narration of every action. Users tolerate failures and delays far better when they understand what's happening. Silent processing feels broken; narrated processing feels collaborative.

Proactive AI requires careful safety design. The confirm-before-acting model isn't a limitation — it's what makes proactive suggestions feel trustworthy rather than alarming. Users need to feel in control even when the AI is doing the work.

---

## What's Next for VibeCat

**Expanded surface support:** VS Code, Safari, and Slack are the next gold-tier targets. Each requires custom AX tree mapping and app-specific hotkey profiles, but the gateway architecture makes adding new surfaces straightforward.

**Persistent user preferences and session learning:** VibeCat currently treats each session independently. Adding Firestore-backed preference memory — "user prefers lo-fi music for focus sessions," "user always wants tests run after code edits" — would make suggestions dramatically more relevant over time.

**Multi-monitor workflow orchestration:** Power users work across 2-3 monitors with different apps on each. VibeCat should understand the full multi-display context and coordinate actions across screens.

**Plugin system for deep app integration:** A plugin API would let app developers expose richer semantic context to VibeCat — enabling suggestions that go beyond what's visible on screen to what the app actually knows about the user's current task.

**Linux support via AT-SPI:** The gateway architecture is platform-agnostic. Replacing the macOS Accessibility API with AT-SPI (Linux accessibility) would bring VibeCat to developer workstations running Ubuntu or Fedora.

---

## Built With

- **Google GenAI SDK (v1.49.0)** — Gemini Live API client for real-time multimodal voice + vision conversation
- **Google Cloud Run** — Serverless backend hosting (asia-northeast3)
- **Google ADK (Agent Development Kit, v0.6.0)** — Screenshot analysis and confidence escalation
- **Google Firestore** — Session state and memory persistence
- **Google Secret Manager** — Secure credential management
- **Swift 6.2** — Native macOS client (AppKit, Accessibility API, AVFoundation)
- **Go 1.26.1** — Realtime Gateway and ADK Orchestrator backends
- **chromedp** — Go-native Chrome DevTools Protocol client for browser automation
- **WebSocket** — Low-latency bidirectional transport between client and gateway
- **macOS Accessibility API** — Native UI element discovery and action execution
