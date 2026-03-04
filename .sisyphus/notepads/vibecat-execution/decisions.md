# VibeCat Execution — Decisions

## [2026-03-03] Session Start

### Stack
- Backend: Go 1.24+ (non-negotiable)
- ADK: Go SDK (google.golang.org/adk v0.5.0)
- Client: Swift 6 / SwiftUI

### MVP Cuts
- ❌ Circle gesture (T-033) → hotkey only
- ❌ Search caching in Firestore (T-168) → always fresh search
- ❌ Topic detection real-time NLP (T-151) → keyword matching stub
- ❌ Background music fade transitions → binary on/off
- ❌ Launch-at-login feature
- ❌ Token refresh flow → static key for demo

### Sleeping Sprite
- No sleeping sprite exists in assets
- Decision: idle sprite with dimmed overlay (T-096)

### macOS Platform Target
- Package.swift: .macOS(.v26)
- README: macOS 15+
- Decision: PENDING — will resolve in Task 3 (scaffolding)
