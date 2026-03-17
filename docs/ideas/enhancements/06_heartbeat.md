# P0-6: Heartbeat Pattern

**SDK Verification**: No SDK support needed. Client-side implementation using existing `Session.SendAudio()`.

**Live API compatible**: YES (this IS a Live API feature)

**Current Code** (session.go):
- `SendAudio(pcmData []byte)` at line 89-98 — sends PCM audio to Gemini
- Audio format: `audio/pcm;rate=16000` (16kHz, 16-bit PCM)

**Current Code** (handler.go):
- Keepalive goroutine at lines 2062-2080 — EXISTS but does WebSocket ping, NOT Gemini audio keepalive
- Reconnection logic at lines 2001-2058 — handles goaway with 3 retry attempts

**Current Problem**: 
- Gemini Live API disconnects after 2-3 minutes of silence
- Current mitigation: session resumption on disconnect
- Better approach: prevent disconnect with heartbeat

**Implementation**:
1. Add heartbeat goroutine that sends silent PCM every 5 seconds
2. Start when session connects, stop on close
3. Silent PCM = 320 bytes of zeros (16-bit samples at 16kHz = 10ms of silence)
4. Only send when no user audio has been sent recently (avoid unnecessary traffic)

**Go Code**:
```go
// In session.go, add heartbeat method:

const (
    heartbeatInterval = 5 * time.Second
    silentPCMSize     = 320 // 10ms of 16kHz 16-bit PCM silence
)

func (s *Session) StartHeartbeat(ctx context.Context) {
    silence := make([]byte, silentPCMSize)
    ticker := time.NewTicker(heartbeatInterval)
    go func() {
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                if s.shouldSendHeartbeat() {
                    if err := s.SendAudio(silence); err != nil {
                        slog.Warn("heartbeat send failed", "err", err)
                        return
                    }
                }
            }
        }
    }()
}

// In Connect(), after session creation (line 85):
sess := &Session{
    gemini: geminiSession,
    cancel: cancel,
    Cfg:    cfg,
}
sess.StartHeartbeat(ctx)
return sess, nil
```

**Optimization** — skip heartbeat when user is actively sending:
```go
type Session struct {
    // ... existing fields ...
    lastAudioSent time.Time
    heartbeatMu   sync.Mutex
}

func (s *Session) SendAudio(pcmData []byte) error {
    s.heartbeatMu.Lock()
    s.lastAudioSent = time.Now()
    s.heartbeatMu.Unlock()
    // ... existing send logic
}

func (s *Session) shouldSendHeartbeat() bool {
    s.heartbeatMu.Lock()
    defer s.heartbeatMu.Unlock()
    return time.Since(s.lastAudioSent) > heartbeatInterval
}
```

**Verification**:
- Start session, stay silent for 5+ minutes
- Verify session stays connected (no goaway/reconnection)
- Check Gemini metrics for heartbeat traffic volume
- Compare reconnection frequency before/after

**Risks**:
- Minimal bandwidth overhead (320 bytes every 5s = 64 bytes/s)
- Silent audio might trigger VAD if not configured correctly
- May conflict with proactive audio mode
