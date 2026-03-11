# VibeCat 발화·말풍선 아키텍처 (2026-03-10 기준)

> Historical note (2026-03-11): 이 문서는 기존 companion speech 경로를 설명하는 기록입니다. 현재 제출 기준의 진실 소스는 아닙니다.

> 이 문서는 현재 구현된 음성 파이프라인, 말풍선 동기화, 상태 머신의 정확한 동작을 기술합니다.
> 새 세션에서 작업을 이어갈 때 이 문서를 참조하세요.

---

## 0. 프로젝트 목적 (Goal)

### 0.1 VibeCat이란

VibeCat은 **솔로 개발자를 위한 macOS 데스크톱 AI 코딩 동반자**입니다. 화면 위에 애니메이션 캐릭터가 상주하며, 개발자의 화면을 보고, 목소리를 듣고, 세션 간 맥락을 기억하고, 필요할 때만 먼저 말을 겁니다. 챗봇이 아니라 **옆자리 동료**입니다.

### 0.2 챌린지 요구사항

[Gemini Live Agent Challenge 2026](https://geminiliveagentchallenge.devpost.com/)에 출품하는 프로젝트로, 다음 4개 기술을 **모두** 사용해야 합니다:

| 필수 기술 | VibeCat 적용 |
|-----------|-------------|
| **GenAI SDK** | `google.golang.org/genai` v1.48 — Gateway에서 Live API·TTS 세션 관리 |
| **Google ADK** | `google.golang.org/adk` — Orchestrator의 9-agent 그래프 |
| **Gemini Live API** | `gemini-2.5-flash-native-audio` — 양방향 실시간 음성+비디오 스트리밍 |
| **VAD** | `AutomaticActivityDetection` — 자연스러운 대화, 바지인 |

### 0.3 제출 산출물

1. **데모 영상** (4분 이내) — 실사용 시나리오 녹화
2. **블로그 포스트** — `#GeminiLiveAgentChallenge` 태그
3. **배포 증거** — Cloud Run 서비스 가동 확인
4. **소스코드** — GitHub 공개 저장소

### 0.4 핵심 목표 (사용자 경험)

> "비용은 무시하고 속도와 안정성. 사용자의 경험이 최우선."

| 목표 | 설명 |
|------|------|
| **안정적 음성 파이프라인** | 이중 오디오 소스 충돌 없이, 발화와 말풍선이 항상 일치 |
| **자연스러운 대화 흐름** | 발화 중에는 사용자 바지인만 인터럽 가능. 새 companionSpeech는 드랍 |
| **적절한 타이밍** | 첫 연결 10초 안정화, 발화 간 3초 갭, 발화 후 2초 말풍선 유지 |
| **프로액티브 동반자** | 에러/성공/막힘을 감지하여 먼저 도움. 질문하지 않고 제안 |
| **6캐릭터 × 고유 페르소나** | 각 캐릭터별 음성·성격·말투가 서버 사이드로 주입 |

### 0.5 현재 단계

- ✅ macOS 클라이언트 완성 (UI, 캡처, 음성, 제스처, 60fps 애니메이션)
- ✅ Backend 2서비스 완성 + Cloud Run 배포 완료
- ✅ 9-agent 그래프 동작 확인
- ✅ 이중 오디오 소스 레이스 컨디션 해결 + 배포
- ✅ 발화 상태 머신 구현 (SpeechState, 바지인, 쿨다운, 말풍선 동기화)
- 🔲 실기기 음성 바지인 최종 테스트
- 🔲 데모 영상 촬영 (4분)
- 🔲 블로그 포스트 작성
- 🔲 DevPost 제출

---

## 1. 전체 아키텍처 개요

```
┌──────────────────────────────────────────────────────────────┐
│                    macOS Client (Swift 6)                     │
│                                                              │
│  ┌──────────┐   ┌──────────────┐   ┌───────────────────┐    │
│  │SpeechRec │──>│ AppDelegate  │──>│    CatPanel        │    │
│  │ ognizer  │   │ (Orchestrator│   │  ┌─────────────┐  │    │
│  │          │   │  SpeechState) │   │  │ChatBubbleView│  │    │
│  └──────────┘   └──────┬───────┘   │  └─────────────┘  │    │
│       │                │           │  ┌─────────────┐  │    │
│       │                │           │  │SpriteAnimator│  │    │
│       │                │           │  └─────────────┘  │    │
│       │         ┌──────┴───────┐   └───────────────────┘    │
│       │         │GatewayClient │                             │
│       │         │ (WebSocket)  │                             │
│       └────────>│              │                             │
│   PCM 16kHz     └──────┬───────┘                             │
│                        │                                     │
│                 ┌──────┴───────┐                             │
│                 │  CatVoice    │                             │
│                 │  AudioPlayer │   PCM 24kHz playback        │
│                 └──────────────┘                             │
└──────────────────────┬───────────────────────────────────────┘
                       │ WebSocket (wss://)
┌──────────────────────┴───────────────────────────────────────┐
│              Realtime Gateway (Go, Cloud Run)                 │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐   │
│  │  ws/handler  │  │ live/session  │  │   tts/client     │   │
│  │  (WebSocket  │──│ (Gemini Live  │  │  (gemini-2.5-    │   │
│  │   handler)   │  │  API proxy)   │  │  flash-tts)      │   │
│  └──────┬───────┘  └──────────────┘  └──────────────────┘   │
│         │                                                    │
│         │ HTTP POST /analyze                                 │
│  ┌──────┴───────────────────────────────────────────────┐   │
│  │           ADK Orchestrator (Go, Cloud Run)            │   │
│  │  9-agent graph (Vision, Memory, Mood, Celebration,    │   │
│  │  Mediator, Scheduler, Engagement, SearchBuddy)        │   │
│  └───────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────┘
```

---

## 2. 오디오 소스 (이중 소스 → 단일 소스 전환 완료)

### 2.1 두 가지 오디오 소스

| 소스 | 생성 위치 | 프로토콜 | 용도 |
|------|----------|---------|------|
| **Live API (네이티브 오디오)** | Gemini Live API | `onAudioData` (binary PCM 24kHz) | 실시간 대화, 프로액티브 발화 |
| **ADK TTS** | Gateway `tts/client` → `gemini-2.5-flash-preview-tts` | `ttsStart`/`ttsEnd` + PCM chunks | ADK 분석 결과 음성화 (고긴급도만) |

### 2.2 이중 소스 레이스 컨디션 해결 (커밋 `0d52ecf`)

**문제**: ADK가 `shouldSpeak=true` 판단 시 Gateway가 동시에:
1. `sess.SendText("[Screen Context]...")` → Gemini가 context에 반응하여 네이티브 오디오 생성
2. `startTTSStream()` → 별도 TTS 오디오 생성
→ 두 오디오가 동시 재생되어 충돌

**해결** (`handler.go` 라인 480-549):
```go
if shouldSpeak && result.SpeechText != "" {
    // companionSpeech 전송 + urgency가 high/critical이면 TTS 시작
    // SendText() 호출하지 않음 → 네이티브 오디오 미생성
} else if result.Vision.Content != "" {
    // shouldSpeak=false일 때만 context injection
    sess.SendText("[Screen Context] ...")
}
```

### 2.3 TTS 발동 조건

```
ADK shouldSpeak=true AND urgency ∈ {high, critical}
  → companionSpeech 메시지 + ttsStart/오디오/ttsEnd 전송

ADK shouldSpeak=true AND urgency ∈ {low, medium}
  → companionSpeech 메시지만 (bubble-only, 음성 없음)

ADK shouldSpeak=false
  → context injection (SendText) → Live API 판단에 위임
```

---

## 3. SpeechState 상태 머신 (AppDelegate)

### 3.1 상태 정의

```swift
// AppDelegate.swift (라인 44-80)
private enum SpeechSource {
    case live   // Gemini Live API 네이티브 오디오
    case tts    // ADK TTS 오디오
}

private enum SpeechState: CustomStringConvertible {
    case idle                           // 아무것도 재생 안 함
    case modelSpeaking(SpeechSource)    // 모델이 발화 중 (소스 추적)
    case cooldown                       // 발화 종료 후 쿨다운 (1초)

    var isSpeaking: Bool    // modelSpeaking인지
    var isCooldown: Bool    // cooldown인지
    var source: SpeechSource?  // 현재 소스
}
```

### 3.2 상태 전이도

```
                    ┌─────────────────────────┐
                    │                         │
                    v                         │
              ┌──────────┐                    │
     ┌───────>│   idle   │<───────┐           │
     │        └────┬─────┘        │           │
     │             │              │           │
     │    transcription/     turnComplete     │
     │    companionSpeech/   (no cooldown)    │
     │    ttsStart                            │
     │             │              │           │
     │             v              │           │
     │   ┌─────────────────┐     │           │
     │   │ modelSpeaking   │─────┘           │
     │   │ (.live / .tts)  │                 │
     │   └────────┬────────┘                 │
     │            │                          │
     │         ttsEnd                        │
     │            │                          │
     │            v                          │
     │     ┌──────────┐      1초 후          │
     │     │ cooldown │──────────────────────┘
     │     └──────────┘
     │
     │  user barge-in (유일한 인터럽)
     │  interrupted 메시지
     │  disconnect
     └────────────────────────────────
```

### 3.3 전이 함수 (`transitionSpeech`)

```swift
// AppDelegate.swift (라인 88-108)
private func transitionSpeech(to newState: SpeechState) {
    switch newState {
    case .idle:
        bubbleLockedByTTS = false
        cooldownTask?.cancel()
        spriteIdleTask?.cancel()
        speechRecognizer?.setModelSpeaking(false)
        catPanel?.setTurnActive(false)
    case .modelSpeaking:
        cooldownTask?.cancel()
        spriteIdleTask?.cancel()
        speechRecognizer?.setModelSpeaking(true)    // mic 게이팅 활성화
        catPanel?.setTurnActive(true)               // 말풍선 auto-hide 억제
    case .cooldown:
        speechRecognizer?.setModelSpeaking(false)
        catPanel?.setTurnActive(false)              // auto-hide 타이머 시작
    }
}
```

---

## 4. 메시지 핸들러별 동작 (AppDelegate `gateway.onMessage`)

### 4.1 `companionSpeech` (ADK 분석 결과 텍스트)

```
도착 → isSpeaking || isCooldown? → DROP (로그: "DROPPED: speech active")
      → minimumSpeechGap(3초) 미충족? → DROP (로그: "DROPPED: too soon")
      → bubbleLockedByTTS = true
      → transitionSpeech(.modelSpeaking(.tts))
      → chatMode? → chatPanel 업데이트
         아니면 → ScreenAnalyzer.handleCompanionSpeech() → TTS 요청
```

**핵심**: 발화 중이거나 쿨다운 중이면 새 companionSpeech는 무조건 드랍. 큐잉 없음.

### 4.2 `transcription` (Live API 출력 텍스트)

```
도착 → chatMode? → chatPanel 업데이트
      → text 비어있으면 무시
      → !isSpeaking이면 → transitionSpeech(.modelSpeaking(.live))
      → emotion 태그 파싱 → displayText 추출
      → pendingTranscription += displayText (누적)
      → bubbleLockedByTTS? → 말풍선 업데이트 스킵
         아니면 → showBubble(pendingTranscription)
      → finished? → recentSpeechStore 저장, pendingTranscription 리셋
```

### 4.3 `ttsStart` (TTS 오디오 시작)

```
도착 → source 결정: text 있으면 .tts, 없으면 .live
     → transitionSpeech(.modelSpeaking(source))
     → catVoice.stop() (기존 오디오 버퍼 클리어)
     → source == .tts && text 있으면:
         → bubbleLockedByTTS = true
         → pendingTranscription = ""
         → showBubble(text)   ← TTS 텍스트로 말풍선 고정
```

### 4.4 `ttsEnd` (TTS 오디오 종료)

```
도착 → !isSpeaking이면 → pendingTranscription 리셋, return
     → lastSpeechEndTime = Date()
     → transitionSpeech(.cooldown)
     → cooldownTask 시작: 1초 후 → cooldown이면 → transitionSpeech(.idle)
     → spriteIdleTask 시작: 2초 후 → idle이면 → spriteAnimator.setState(.idle)
```

### 4.5 `turnComplete` (Live API 턴 완료)

```
도착 → catVoice.flush() (남은 오디오 재생)
     → lastSpeechEndTime = Date()
     → cooldown 상태가 아니면 → transitionSpeech(.idle)
     → pendingTranscription = ""
```

### 4.6 `interrupted` (서버 인터럽)

```
도착 → catVoice.stop()
     → transitionSpeech(.idle)
     → pendingTranscription = ""
     → hideBubble()
```

### 4.7 `audio` (PCM 오디오 청크)

```
도착 → onAudioData → CatVoice.enqueueAudio() → AudioPlayer.enqueue()
```

---

## 5. 사용자 바지인 (Barge-in)

### 5.1 3단계 보호

| 레이어 | 위치 | 설정 | 동작 |
|--------|------|------|------|
| **1. Client** | `SpeechAudioCapture` | rmsThreshold=0.03, bargeInThreshold=0.06, consecutiveThreshold=4 | RMS > bargeInThreshold이 4프레임 연속이면 `onBargeInDetected` |
| **2. Gateway** | `GatewayClient.isTTSSpeaking` | 500ms cooldown | ttsStart 시 `isTTSSpeaking=true`, ttsEnd 후 500ms 대기 후 false |
| **3. Live API** | `session.go` VAD | PrefixPadding=20ms, Silence=200ms, Sensitivity=Low, StartOfActivityInterrupts | Gemini 서버 측 음성 활동 감지 |

### 5.2 바지인 처리 (`handleUserBargeIn`)

```swift
// AppDelegate.swift (라인 575-583)
private func handleUserBargeIn() {
    guard speechState.isSpeaking else { return }
    NSLog("[SPEECH] local barge-in detected")
    gatewayClient?.sendBargeIn()      // Gateway에 bargeIn JSON 전송
    catVoice?.stop()                   // 오디오 즉시 정지
    pendingTranscription = ""          // 누적 텍스트 리셋
    catPanel?.hideBubble()             // 말풍선 즉시 닫기
    transitionSpeech(to: .idle)        // idle로 복귀
}
```

---

## 6. 말풍선 시스템 (CatPanel)

### 6.1 핵심 프로퍼티

```swift
// CatPanel.swift
private var bubbleDuration: TimeInterval = 2.0    // 고정 2초 (발화 종료 후)
private var turnActive = false                      // AppDelegate가 제어
private let maxBubbleDisplayTime: TimeInterval = 15.0  // 최대 표시 시간
```

### 6.2 `showBubble(text:)` 동작

```
호출 → bubbleShownAt = Date()
     → bubbleDuration = 2.0 (고정)
     → hideCountdownTimer 취소
     → 이미 보이면 → updateText()
        아니면 → show()
     → ensureSmartHidePolling() 시작 (0.5초 폴링)
```

### 6.3 Auto-hide 로직 (`evaluateBubbleHide`)

```
0.5초마다 평가:
  → 15초 초과? → 강제 hideBubble()
  → turnActive || audioPlayer.isPlaying? → 타이머 취소 (말풍선 유지)
  → 둘 다 false이면:
      → hideCountdownTimer 없으면 → 2초 타이머 시작
      → 2초 후 → hideBubble()
```

### 6.4 `bubbleLockedByTTS` 플래그

- **목적**: TTS가 말풍선에 표시한 텍스트를 Live API transcription이 덮어쓰지 못하게 잠금
- **true 시점**: `companionSpeech` 핸들러 / `ttsStart` 핸들러 (text 있을 때)
- **false 시점**: `transitionSpeech(.idle)` 호출 시

---

## 7. ScreenAnalyzer (화면 캡처)

### 7.1 캡처 흐름

```
1Hz 캡처 루프 (configurable)
  → 첫 연결 후 10초 안정화 딜레이 (initialStabilizationDelay)
  → ImageDiffer로 변경 감지
  → Fast Path (1초 쿨다운): JPEG → sendVideoFrame() → Gemini Live API
  → Smart Path (15초 쿨다운): base64 → sendScreenCapture() → Gateway → ADK
```

### 7.2 Speech-aware 캡처 억제

```swift
// ScreenAnalyzer.swift (라인 138-143)
private var isSpeechActive: Bool {
    let audioPlaying = audioPlayer?.isPlaying ?? false
    let ttsSpeaking = gatewayClient.isTTSSpeaking
    let recentlyStopped = Date().timeIntervalSince(gatewayClient.lastSpeechEndTime) < postSpeechCooldown  // 5초
    return audioPlaying || ttsSpeaking || recentlyStopped
}
```

Smart Path는 `isSpeechActive`가 true이면 스킵. Fast Path(비디오 프레임)는 항상 전송.

---

## 8. 오디오 재생 (AudioPlayer / CatVoice)

### 8.1 AudioPlayer

```swift
// AudioPlayer.swift
- PCM 24kHz 16-bit mono 포맷
- coalesceThreshold = 960 bytes (~20ms) — 최소 버퍼링
- isPlaying: 스케줄된 버퍼 개수로 자동 관리
- scheduleBuffer 완료 콜백에서 scheduledBufferCount-- → 0이면 isPlaying=false
- 외부 완료 콜백 없음 (CatPanel이 isPlaying 폴링으로 확인)
```

### 8.2 CatVoice

```swift
// CatVoice.swift — AudioPlayer의 thin wrapper
enqueueAudio() → audioPlayer.enqueue()
flush()        → audioPlayer.flush()
stop()         → audioPlayer.clear()  // 버퍼 전부 클리어 + 재생 정지
```

---

## 9. GatewayClient 메시지 파싱

### 9.1 수신 메시지 타입 (ServerMessage)

| 타입 | 소스 | 주요 데이터 |
|------|------|------------|
| `companionSpeech` | ADK 분석 결과 | text, emotion, urgency |
| `transcription` | Live API 출력 텍스트 | text, finished |
| `inputTranscription` | 사용자 음성 텍스트 | text, finished |
| `audio` | Live API / TTS | PCM data |
| `ttsStart` | Gateway TTS 시작 | text (optional) |
| `ttsEnd` | Gateway TTS 종료 | — |
| `turnComplete` | Live API 턴 끝 | — |
| `interrupted` | Live API 인터럽 | — |
| `setupComplete` | Gateway 연결 완료 | sessionId |
| `sessionResumptionUpdate` | Live API | handle |
| `pong` | Gateway heartbeat | — |
| `error` | Gateway | code, message |

### 9.2 GatewayClient TTS 추적

```swift
// GatewayClient.swift
isTTSSpeaking: Bool          // ttsStart → true, ttsEnd + 500ms → false
lastSpeechEndTime: Date      // ttsEnd 후 500ms cooldown 완료 시 갱신
```

---

## 10. SpriteAnimator

```swift
// 상태: idle, thinking, happy, surprised, frustrated, celebrating
// thinking 상태 10초 후 자동 idle 복귀 (thinkingTimeoutTask)
// celebrating → idle 전환 시 happy 4초 오버라이드
// onStateTransition 콜백으로 EmotionTransitionStore에 기록
```

---

## 11. 핵심 설정값 요약

### 클라이언트 (Swift)

| 설정 | 값 | 위치 |
|------|---|------|
| `minimumSpeechGap` | 3.0초 | AppDelegate.swift:42 |
| `bubbleDuration` | 2.0초 (고정) | CatPanel.swift:104 |
| `maxBubbleDisplayTime` | 15.0초 | CatPanel.swift:22 |
| `initialStabilizationDelay` | 10초 | ScreenAnalyzer.swift:111 |
| `fastPathCooldown` | 1.0초 | ScreenAnalyzer.swift:25 |
| `smartPathCooldown` | 15.0초 | ScreenAnalyzer.swift:26 |
| `postSpeechCooldown` | 5.0초 | ScreenAnalyzer.swift:27 |
| `rmsThreshold` | 0.03 | SpeechRecognizer.swift:81 |
| `bargeInThreshold` | 0.06 | SpeechRecognizer.swift:82 |
| `consecutiveThreshold` | 4 | SpeechRecognizer.swift:89 |
| `coalesceThreshold` | 960 bytes (~20ms) | AudioPlayer.swift:15 |
| `ttsEnd cooldown` | 500ms | GatewayClient.swift:404-410 |
| `cooldownTask 지속시간` | 1초 | AppDelegate.swift:418 |
| `spriteIdleTask 지속시간` | 2초 | AppDelegate.swift:425 |

### 서버 (Go)

| 설정 | 값 | 위치 |
|------|---|------|
| VAD PrefixPaddingMs | 20 | session.go:194 |
| VAD SilenceDurationMs | 200 | session.go:195 |
| VAD StartSensitivity | Low | session.go:198 |
| VAD EndSensitivity | Low | session.go:199 |
| ActivityHandling | StartOfActivityInterrupts | session.go:203 |
| TurnCoverage | TurnIncludesOnlyActivity | session.go:204 |
| ContextWindowCompression Trigger | 100K tokens | session.go:209 |
| ContextWindowCompression Target | 50K tokens | session.go:210 |
| MediaResolution | Medium | session.go:207 |

---

## 12. 현재 구현된 발화 규칙 요약

| 규칙 | 상태 | 구현 위치 |
|------|------|----------|
| 발화 중 새 companionSpeech 도착 시 드랍 | ✅ 완료 | AppDelegate.swift:317-319 |
| 쿨다운 중 새 companionSpeech 도착 시 드랍 | ✅ 완료 | AppDelegate.swift:317-319 |
| minimumSpeechGap 3초 미만 시 드랍 | ✅ 완료 | AppDelegate.swift:321-324 |
| 사용자 바지인만 인터럽 가능 | ✅ 완료 | AppDelegate.swift:575-583 |
| Live API transcription 말풍선 누적 표시 | ✅ 완료 | AppDelegate.swift:348-356 |
| TTS 말풍선 잠금 (bubbleLockedByTTS) | ✅ 완료 | AppDelegate.swift:326, 404-408 |
| 발화 종료 후 2초 뒤 말풍선 자동 닫힘 | ✅ 완료 | CatPanel.swift:104, 144-166 |
| 첫 연결 후 10초 안정화 딜레이 | ✅ 완료 | ScreenAnalyzer.swift:111, 118-121 |
| ttsEnd 후 1초 cooldown | ✅ 완료 | AppDelegate.swift:416-423 |
| ttsEnd 후 2초 뒤 sprite idle 복귀 | ✅ 완료 | AppDelegate.swift:424-430 |
| 이중 오디오 소스 충돌 방지 | ✅ 완료 (배포됨) | handler.go:513-549 |

---

## 13. 배포 상태

| 서비스 | 리비전 | 상태 |
|--------|--------|------|
| realtime-gateway | `realtime-gateway-00027-ww9` | ✅ 100% traffic |
| adk-orchestrator | `adk-orchestrator-00028-xmr` | ✅ 100% traffic |
| Swift 클라이언트 | 로컬 빌드 | ✅ 빌드 성공 (미커밋 변경사항 있음) |

---

## 14. 미커밋 변경사항 (2026-03-10)

### AppDelegate.swift
- `SpeechState.isCooldown` 프로퍼티 추가 (컴파일 에러 수정)
- `companionSpeech` 핸들러에서 `isCooldown` 상태도 드랍 조건에 포함

### CatPanel.swift
- `bubbleDuration` 계산을 텍스트 길이 기반에서 고정 2.0초로 변경

---

## 15. 알려진 한계 / 향후 개선 가능 사항

1. **AudioPlayer에 완료 콜백 없음**: `isPlaying`만 auto-reset. CatPanel이 0.5초 폴링으로 확인. 정밀 타이밍이 필요하면 콜백 추가 필요.
2. **turnComplete와 ttsEnd 경쟁**: 두 이벤트가 거의 동시에 올 수 있음. turnComplete는 cooldown을 건너뛰므로, ttsEnd가 먼저 와서 cooldown 진입 후 turnComplete가 idle로 덮어쓸 수 있음.
3. **bubble-only 모드의 말풍선 닫기 타이밍**: urgency가 low/medium인 companionSpeech는 TTS 없이 말풍선만 표시. 이 경우 오디오가 없으므로 turnActive=false 직후 2초 타이머가 바로 시작됨. 텍스트가 길면 읽기 전에 닫힐 수 있음.
4. **Live API proactiveAudio + companionSpeech 동시 발생**: Live API가 자발적으로 발화하는 시점에 ADK도 발화를 결정하면 겹칠 수 있음. 현재는 companionSpeech가 isSpeaking이면 드랍되므로 Live API가 우선.
