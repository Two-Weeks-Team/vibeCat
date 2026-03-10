# VibeCat 전체 아키텍처 분석

**Date:** 2026-03-09
**Revision:** realtime-gateway rev 00014-sps / adk-orchestrator rev 00015-6zt
**Previous:** ARCHITECTURE_ANALYSIS_20260309.md (rev 00012-qkl / 00013-qdj)

---

## 변경 이력 (20260309 → 20260309-2)

| 항목 | 이전 | 현재 | 이유 |
|------|------|------|------|
| AudioPlayer coalesce | 4800 bytes (100ms) | 960 bytes (20ms) | 크래클링 수정 (버퍼 언더런 감소) |
| 바지인 방식 | 모델 발화 중 오디오 완전 차단 | RMS 에코 게이트 (bargeInThreshold=0.025) | 사용자 끼어들기 지원 |
| `isTTSSpeaking` guard | `sendAudio()` 차단 | 제거 (항상 전송) | 바지인 경로 개방 |
| `modelSpeaking` 차단 | handler.go에서 오디오 drop | 제거 (항상 Gemini 전달) | 바지인 경로 개방 |
| 화면 컨텍스트 | Gemini Live에 없음 | ADK Vision 결과를 `SendText()` 주입 | "화면 설명해줘" 응답 가능 |
| Gateway 리비전 | 00012-qkl | 00014-sps | 위 변경사항 배포 |
| Orchestrator 리비전 | 00013-qdj | 00015-6zt | 위 변경사항 배포 |

---

## 1. 3-Layer 시스템 구조

```
┌─────────────────────────────┐
│   macOS Swift Client        │  VibeCat/ (28 Swift files)
│   UI + Capture + Audio I/O  │
└──────────┬──────────────────┘
           │ WebSocket (wss://)
           │ Binary: PCM audio
           │ JSON: setup, screenCapture, clientContent, ping
           ▼
┌─────────────────────────────┐
│   Realtime Gateway          │  backend/realtime-gateway/ (Go)
│   Cloud Run rev 00014-sps   │  Port 8080
│   GenAI SDK + Live API      │
└──────────┬──────────────────┘
           │ HTTP POST /analyze
           ▼
┌─────────────────────────────┐
│   ADK Orchestrator          │  backend/adk-orchestrator/ (Go)
│   Cloud Run rev 00015-6zt   │  Port 8080
│   9-Agent Graph (ADK SDK)   │
└─────────────────────────────┘
```

| Layer | Technology | Location | Role |
|-------|-----------|----------|------|
| macOS Client | Swift 6 / SwiftUI / SPM | `VibeCat/` | UI, screen capture, audio I/O, gestures |
| Realtime Gateway | Go + `google.golang.org/genai` | `backend/realtime-gateway/` | WebSocket proxy to Gemini Live API + TTS |
| ADK Orchestrator | Go + `google.golang.org/adk` | `backend/adk-orchestrator/` | 9-agent decision graph |
| Persistence | Firestore | GCP `asia-northeast3` | Sessions, metrics, memory |

---

## 2. 음성 파이프라인 (Voice Pipeline)

### 입력 경로: 마이크 → Gemini

| 단계 | 파일 | 포맷 | 동작 | 소요시간 |
|------|------|------|------|----------|
| ① 마이크 캡처 | `SpeechAudioCapture` | Float32, ~44100Hz, mono | `AVAudioEngine.inputNode` 탭, bufferSize=4096 | ~93ms/buffer (4096 / 44100) |
| ② 노이즈 게이트 | 동일 파일 | RMS 계산 | 평시: threshold=0.003, 모델발화중: threshold=0.025 (에코게이트) | <0.1ms |
| ③ 포맷 변환 | `AppDelegate.convertAudioBufferToPCM16k()` | Float32→Int16, 44100→16000Hz | `AVAudioConverter`, serial DispatchQueue | ~1-2ms |
| ④ 전송 | `GatewayClient.sendAudio()` | binary WebSocket, ~3200 bytes/chunk | 항상 전송 (guard 없음) | ~1ms (로컬) |
| ⑤ 네트워크 | WebSocket → Cloud Run | encrypted TLS | 한국↔asia-northeast3 | ~5-15ms (국내) |
| ⑥ 게이트웨이 수신 | `handler.go` BinaryMessage | raw PCM bytes | 항상 Gemini에 전달 (차단 없음) | <1ms |
| ⑦ Gemini 전달 | `session.SendAudio()` | `audio/pcm;rate=16000` MIME | `SendRealtimeInput` | <1ms |
| ⑧ Gemini VAD 처리 | Gemini Live API | 내부 처리 | VAD 감지 → 응답 생성 시작 | ~300-1500ms |

**총 입력 지연**: ~400-1600ms (마이크→Gemini 응답 시작)

### 출력 경로: Gemini → 스피커

| 단계 | 파일 | 포맷 | 동작 | 소요시간 |
|------|------|------|------|----------|
| ① Gemini 응답 | `receiveFromGemini()` | `InlineData.Data` (PCM) | 첫 오디오 → `ttsStart` + `modelSpeaking=true` | 스트리밍 |
| ② 네트워크 | Cloud Run → WebSocket | binary frame | `WriteMessage(BinaryMessage, data)` | ~5-15ms |
| ③ 클라이언트 파싱 | `AudioMessageParser.parse()` | binary → `.audio(Data)` | JSON 실패 → audio fallback | <0.1ms |
| ④ CatVoice 위임 | `CatVoice.enqueueAudio()` | 패스스루 | `AudioPlayer`의 thin wrapper | <0.01ms |
| ⑤ 버퍼링 | `AudioPlayer.enqueue()` | 960 bytes 축적 후 재생 | ~20ms @24kHz coalesce | 0-20ms |
| ⑥ 스케줄링 | `scheduleAccumulatedSamples()` | `AVAudioPCMBuffer` | `playerNode.scheduleBuffer()` + `.play()` | <1ms |
| ⑦ 스피커 출력 | `AVAudioPlayerNode` → `mainMixerNode` | Int16, 24000Hz, mono | 하드웨어 재생 | ~5-10ms (OS 버퍼) |

**총 출력 지연**: ~30-60ms (게이트웨이 수신→스피커 출력)

### 바지인 (Barge-in) 경로

```
사용자 발화 (모델 발화 중)
    │
    ▼
SpeechAudioCapture: RMS ≥ 0.025? ──No──→ DROP (에코)
    │ Yes (의도적 음성)
    ▼
convertAudioBufferToPCM16k → GatewayClient.sendAudio() [항상 전송]
    │
    ▼
handler.go: BinaryMessage [항상 Gemini 전달]
    │
    ▼
Gemini VAD: StartOfActivityInterrupts → 모델 응답 중단
    │
    ▼
receiveFromGemini: sc.Interrupted=true → ttsEnd + modelSpeaking=false
    │
    ▼
클라이언트: .interrupted → catVoice.stop() + speechRecognizer.setModelSpeaking(false)
```

**바지인 지연**: ~200-500ms (사용자 발화→모델 중단)

### 포맷 요약

| 방향 | 포맷 | 샘플레이트 | 비트 | 채널 |
|------|------|-----------|------|------|
| 마이크 → 변환기 | Float32 | ~44100 Hz | 32-bit | mono |
| 클라이언트 → 서버 | Int16 | 16000 Hz | 16-bit | mono |
| 서버 → 클라이언트 (Live) | Int16 | 24000 Hz | 16-bit | mono |
| 서버 → 클라이언트 (TTS) | Int16 | 24000 Hz | 16-bit | mono |
| 재생 | Int16 | 24000 Hz | 16-bit | mono |

---

## 3. 화면 분석 파이프라인 (Vision Pipeline)

```
ScreenAnalyzer.scheduleNextCapture() (captureInterval마다)
        │                                                    소요시간
        ▼
ScreenCaptureService.captureAroundCursor()                   ~50-100ms
  └── SCScreenshotManager.captureImage() (ScreenCaptureKit)
  └── ImageDiffer.hasSignificantChange() (diff < 0.05면 unchanged)
        │ (변화 감지 시)
        ▼
ImageProcessor.toBase64JPEG() → base64 문자열                ~10-30ms
        │
        ▼
GatewayClient.sendScreenCapture() (JSON: type=screenCapture)  ~1ms
        │ WebSocket text frame
        ▼
handler.go "screenCapture" case                               <1ms
  └── modelSpeaking=true면 regular capture만 SKIP
  └── forceCapture는 항상 허용
        │
        ▼ (async goroutine)
adkClient.Analyze() → HTTP POST /analyze                     ~5-15ms (네트워크)
        │
        ▼
ADK Orchestrator: 9-Agent Graph                              ~3-8s
  ├── Wave 1: VisionAgent ∥ MemoryAgent                      ~1-3s
  ├── Wave 2: MoodDetector ∥ CelebrationTrigger              ~0.5-1s
  └── Wave 3: Mediator → Scheduler → Engagement → Search     ~1-4s
        │
        ▼
AnalyzeResult { Decision, Vision, Mood, SpeechText }
        │
        ├──→ [NEW] sess.SendText("[Screen Context] ...") ←── Vision 결과를 Gemini Live에 주입
        │                                                     ~1ms
        ▼
handler.go: shouldSpeak && SpeechText != ""
  ├── companionSpeech (JSON) → 클라이언트 말풍선               ~5-15ms
  └── urgency=high/critical → TTS 스트리밍                    ~1-3s 추가
```

**총 화면분석 지연**: ~4-12s (캡처→클라이언트 결과)
**화면 컨텍스트 주입**: ADK 분석 완료 시 자동, Gemini가 "화면 설명해줘"에 응답 가능

---

## 4. TTS 파이프라인 (Companion Speech)

| 단계 | 위치 | 동작 | 소요시간 |
|------|------|------|----------|
| ① 트리거 | handler.go `shouldSpeak && urgency=high` | `startTTSStream()` 호출 | <1ms |
| ② TTS 요청 | tts/client.go `StreamSpeak()` | `gemini-2.5-flash-preview-tts`, `GenerateContentStream` | ~500-1500ms (첫 청크) |
| ③ 스트리밍 | 동일 | 청크 → `sink()` → binary WebSocket | ~20-50ms/청크 |
| ④ 래핑 | handler.go | `ttsStart` (JSON) → 오디오 청크들 → `ttsEnd` (JSON) | 메시지 오버헤드 |
| ⑤ 클라이언트 | AppDelegate | `ttsStart` → `setModelSpeaking(true)`, `ttsEnd` → 500ms 후 `setModelSpeaking(false)` | 500ms 쿨다운 |

**총 TTS 지연**: ~500-1500ms (트리거→첫 오디오 스피커 출력)

---

## 5. 9-Agent Graph (ADK Orchestrator)

```
vibecat_graph (SequentialAgent)
  │
  ├── wave1_perception (ParallelAgent)              ~1-3s
  │   ├── vision_agent    — 스크린샷 분석
  │   └── memory_agent    — 크로스세션 컨텍스트
  │
  ├── wave2_emotion (ParallelAgent)                 ~0.5-1s
  │   ├── mood_detector       — 개발자 기분 분류
  │   └── celebration_trigger — 성공 이벤트 감지
  │
  └── wave3_decision (SequentialAgent)              ~1-4s
      ├── mediator           — 말할지 결정
      ├── adaptive_scheduler — 타이밍 조정
      ├── engagement_agent   — 프로액티브 개입
      └── search_refinement_loop (LoopAgent, max=2)
          ├── search_buddy      — Google Search
          └── llm_search_agent  — LLM Search
```

| Agent | ADK Type | Gemini 모델 | 소요시간 |
|-------|----------|------------|----------|
| VisionAgent | Custom (Run func) | `gemini-3.1-flash-lite-preview` | ~1-2s |
| MemoryAgent | Custom (Run func) | `gemini-3.1-flash-lite-preview` | ~0.5-1s |
| MoodDetector | Custom (Run func) | 없음 (규칙 기반) | <10ms |
| CelebrationTrigger | Custom (Run func) | `gemini-3.1-flash-lite-preview` | ~0.5-1s |
| Mediator | Custom (Run func) | `gemini-3.1-flash-lite-preview` | ~0.5-1s |
| AdaptiveScheduler | Custom (Run func) | 없음 (규칙 기반) | <10ms |
| EngagementAgent | Custom (Run func) | `gemini-3.1-flash-lite-preview` | ~0.5-1s |
| SearchBuddy | Custom (Run func) | `gemini-3.1-flash-lite-preview` | ~1-3s (검색 포함) |
| LLMSearchAgent | LLMAgent | `gemini-3.1-flash-lite-preview` | ~1-2s |

---

## 6. Gemini Live API VAD 설정

| 설정 | 값 | 의미 |
|------|-----|------|
| `StartOfSpeechSensitivity` | Low | 발화 시작 감지 덜 민감 |
| `EndOfSpeechSensitivity` | Low | 발화 종료 판단 보수적 |
| `PrefixPaddingMs` | 300ms | 발화 시작 전 300ms 오디오 포함 |
| `SilenceDurationMs` | 500ms | 500ms 침묵 시 발화 종료 |
| `ActivityHandling` | StartOfActivityInterrupts | 사용자 발화 → 모델 응답 즉시 중단 (바지인) |
| `TurnCoverage` | TurnIncludesOnlyActivity | 활성 음성만 turn에 포함 |
| `ResponseModalities` | [Audio] | 오디오로만 응답 |
| `OutputAudioTranscription` | 활성화 | 모델 음성 텍스트 변환 |
| `InputAudioTranscription` | 활성화 | 사용자 음성 텍스트 변환 |
| `EnableAffectiveDialog` | true (설정 시) | 감정 인식 대화 |
| `Proactivity.ProactiveAudio` | true (설정 시) | 모델 자발적 발화 |
| `ContextWindowCompression` | trigger=4096, target=2048 | 긴 세션 컨텍스트 압축 |
| `SessionResumption` | 활성화 | 세션 재연결 지원 |

---

## 7. 상태 플래그 & 인터럽트 제어

| 플래그 | 위치 | 역할 | 변경사항 |
|--------|------|------|----------|
| `modelSpeaking` (클라이언트) | `SpeechAudioCapture` | true면 RMS 임계값 0.003→0.025 (에코 게이트) | **NEW** |
| `modelSpeaking` (게이트웨이) | `liveSessionState` | true면 ① 일반 screenCapture 억제 ② cancelTTS 억제. 오디오는 항상 전달 | **CHANGED** |
| `isTTSSpeaking` | `GatewayClient` | `sendAudio()`에서 guard 제거됨. ttsStart/ttsEnd 추적용으로만 유지 | **CHANGED** |
| `ttsEndCooldownTask` | `GatewayClient` | ttsEnd 후 500ms 딜레이 → `isTTSSpeaking=false` | 유지 |
| `ttsCancel` | `liveSessionState` | companion TTS 취소 (BinaryMessage 수신 시) | 유지 |
| `turnHasAudio` | `receiveFromGemini` | Gemini turn 내 오디오 존재 여부 추적 | 유지 |

### 에코 게이트 동작 원리

```
모델 발화 중:
  스피커 → 마이크: 에코 RMS ~0.005-0.03
  사용자 직접 발화: RMS ~0.03-0.5

  SpeechAudioCapture threshold:
    modelSpeaking=false → 0.003 (거의 모든 음성 통과)
    modelSpeaking=true  → 0.025 (에코 차단, 의도적 음성만 통과)

  ttsStart → AppDelegate: speechRecognizer.setModelSpeaking(true)
  ttsEnd   → AppDelegate: 500ms 후 speechRecognizer.setModelSpeaking(false)
  interrupted → AppDelegate: 즉시 speechRecognizer.setModelSpeaking(false)
  turnComplete → AppDelegate: 즉시 speechRecognizer.setModelSpeaking(false)
```

---

## 8. 화면-음성 브리지 (Screen Context Injection)

**문제**: Gemini Live API는 음성 전용 — 화면 데이터 접근 불가
**해결**: ADK VisionAnalysis.Content를 Gemini Live 세션에 텍스트로 주입

```
ADK Orchestrator /analyze 완료
    │
    ▼
handler.go: result.Vision.Content != ""
    │
    ▼
sess.SendText("[Screen Context] {vision content}")
    │
    ▼
Gemini Live 세션: 화면 컨텍스트 보유
    │
    ▼
사용자: "지금 화면에 뭐가 보여?"
    │
    ▼
Gemini: [Screen Context] 기반 응답 생성
```

**제약**: ADK 분석이 최소 1회 완료되어야 컨텍스트 존재. 앱 시작 직후에는 captureInterval 대기 필요.

---

## 9. WebSocket 메시지 프로토콜

### 클라이언트 → 게이트웨이

| 타입 | 형식 | 설명 |
|------|------|------|
| `setup` | JSON | 초기 설정 (voice, language, model, soul, deviceId) |
| binary | Binary | PCM 16kHz 16-bit mono 오디오 (바지인 포함 항상 전송) |
| `screenCapture` | JSON | base64 스크린샷 + context |
| `forceCapture` | JSON | 강제 분석 (modelSpeaking 중에도 허용) |
| `clientContent` | JSON | 텍스트 입력 (채팅 모드) |
| `settingsUpdate` | JSON | 설정 변경 → 세션 재연결 |
| `ping` | JSON | 앱 레벨 heartbeat (30초) |

### 게이트웨이 → 클라이언트

| 타입 | 형식 | 설명 |
|------|------|------|
| `setupComplete` | JSON | 연결 완료 + sessionId |
| binary | Binary | PCM 24kHz 16-bit mono 오디오 |
| `transcription` | JSON | 모델 음성 → 텍스트 |
| `inputTranscription` | JSON | 사용자 음성 → 텍스트 |
| `companionSpeech` | JSON | ADK 분석 결과 텍스트 + emotion + urgency |
| `ttsStart` | JSON | 오디오 시작 → 에코 게이트 활성화 |
| `ttsEnd` | JSON | 오디오 종료 → 500ms 후 에코 게이트 해제 |
| `turnComplete` | JSON | Gemini turn 완료 |
| `interrupted` | JSON | Gemini 응답 중단 (바지인 성공) |
| `sessionResumptionUpdate` | JSON | 세션 재연결 핸들 |
| `liveSessionReconnecting` | JSON | Gemini 세션 재연결 중 |
| `liveSessionReconnected` | JSON | Gemini 세션 재연결 완료 |
| `goAway` | JSON | Gemini 세션 타임아웃 경고 |
| `pong` | JSON | heartbeat 응답 |
| `error` | JSON | 에러 (code + message) |

---

## 10. 클라이언트 Swift 모듈 구조

### Core 모듈 (UI 의존성 없음)

| 파일 | 역할 |
|------|------|
| `AudioMessageParser.swift` | 서버 메시지 파싱 (JSON ↔ enum) |
| `Models.swift` | 공유 타입 (ChatMessage, CompanionEmotion, CompanionSpeechEvent) |
| `Settings.swift` | AppSettings (UserDefaults 기반) |
| `ImageDiffer.swift` | 스크린샷 변화 감지 (pixel diff) |
| `ImageProcessor.swift` | CGImage → base64 JPEG 변환 |
| `PCMConverter.swift` | Int16 ↔ Float32 유틸리티 |
| `KeychainHelper.swift` | Keychain API 키 저장 |
| `Core.swift` | 모듈 설명 (placeholder 제거) |

### App 모듈 (UI + 시스템)

| 파일 | 역할 |
|------|------|
| `AppDelegate.swift` | 앱 진입점, 모든 컴포넌트 조립, 오디오 변환, **setModelSpeaking 연결** |
| `GatewayClient.swift` | WebSocket 연결, 메시지 송수신, 재연결 |
| `SpeechRecognizer.swift` | 마이크 캡처 + **에코 게이트 (bargeInThreshold=0.025)** + **setModelSpeaking()** |
| `AudioPlayer.swift` | 스피커 출력 (AVAudioPlayerNode, 24kHz, **coalesce=960bytes/20ms**) |
| `CatVoice.swift` | AudioPlayer thin wrapper |
| `CatPanel.swift` | 캐릭터 UI (스프라이트 + 말풍선 + 이모지 애니메이션) |
| `SpriteAnimator.swift` | 스프라이트 상태 머신 (idle/thinking/happy/surprised/frustrated/celebrating) |
| `ScreenAnalyzer.swift` | 화면 캡처 주기 관리 + ADK 결과 처리 |
| `ScreenCaptureService.swift` | ScreenCaptureKit 래퍼 |
| `CatViewModel.swift` | 캐릭터 위치/화면 관리 |
| `CompanionChatPanel.swift` | 채팅 모드 UI |
| `StatusBarController.swift` | 메뉴바 아이콘 + 상태 |
| `TrayIconAnimator.swift` | 메뉴바 아이콘 애니메이션 |
| `BackgroundMusicPlayer.swift` | 배경 음악 |
| `DecisionOverlayHUD.swift` | ADK 결정 오버레이 |
| `OnboardingWindowController.swift` | 초기 설정 UI |
| `CircleGestureDetector.swift` | 원형 제스처 감지 |
| `ErrorReporter.swift` | 에러 로깅 |
| `ChatBubbleView.swift` | 말풍선 뷰 |

---

## 11. 사용 모델

| 용도 | 모델 | 위치 |
|------|------|------|
| Live Voice (VAD + 대화) | `gemini-2.5-flash-native-audio-latest` | Gateway → Live API |
| TTS (Companion Speech) | `gemini-2.5-flash-preview-tts` | Gateway → TTS Client |
| Vision / Search / Agent | `gemini-3.1-flash-lite-preview` | ADK Orchestrator |

---

## 12. 인프라 & GCP

| 리소스 | 서비스 |
|--------|--------|
| Cloud Run | `realtime-gateway` (rev 00014-sps), `adk-orchestrator` (rev 00015-6zt) |
| Firestore | 세션, 메트릭, 크로스세션 메모리 |
| Secret Manager | `vibecat-gemini-api-key`, `vibecat-gateway-auth-secret` |
| Artifact Registry | `vibecat-images` 컨테이너 |
| Cloud Trace | OpenTelemetry spans |
| Cloud Logging | 구조화 로깅 |
| Region | `asia-northeast3` |
| Project | `vibecat-489105` |

---

## 13. End-to-End 지연 시간 요약

| 경로 | 단계 수 | 총 지연 |
|------|---------|---------|
| 마이크 → Gemini 응답 시작 | 8 | ~400-1600ms |
| Gemini 오디오 → 스피커 | 7 | ~30-60ms |
| 바지인 (발화→중단) | 전체 | ~200-500ms |
| 화면 캡처 → ADK 결과 | 전체 | ~4-12s |
| TTS 트리거 → 첫 오디오 | 5 | ~500-1500ms |
| 화면 컨텍스트 → Gemini 인지 | ADK 후 | ~1ms (SendText) |

---

## 14. 알려진 이슈 (2026-03-09 Rev.2)

| 이슈 | 심각도 | 설명 | 상태 |
|------|--------|------|------|
| 에코 캔슬링 | ⚠️ | `setVoiceProcessingEnabled(false)` → Apple AEC 비활성. 에코 게이트(RMS 0.025)로 대체 | 부분 해결 |
| 크래클링 | ⚠️ | coalesce 100ms→20ms로 감소. 사용자 확인 필요 | 부분 해결 |
| 바지인 | ⚠️ | 에코 게이트 + ActivityHandling 방식으로 전환. threshold=0.025 튜닝 필요할 수 있음 | 구현 완료 |
| 화면 인식 | ✅ | Vision 결과를 Gemini Live에 SendText로 주입. 첫 캡처 전까지 컨텍스트 없음 | 해결 |
| ~~Barge-in 불가~~ | ✅ | ~~modelSpeaking 동안 모든 오디오 차단~~ → 에코 게이트로 전환 | 해결 |
| ~~Gemini 음성 응답~~ | ✅ | inputTranscription 수신 확인 + 음성 응답 작동 | 해결 |
