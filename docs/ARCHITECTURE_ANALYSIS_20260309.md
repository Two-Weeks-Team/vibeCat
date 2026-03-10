# VibeCat 전체 아키텍처 분석

**Date:** 2026-03-09
**Revision:** realtime-gateway rev 00012-qkl / adk-orchestrator rev 00013-qdj

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
│   Cloud Run rev 00012-qkl   │  Port 8080
│   GenAI SDK + Live API      │
└──────────┬──────────────────┘
           │ HTTP POST /analyze
           ▼
┌─────────────────────────────┐
│   ADK Orchestrator          │  backend/adk-orchestrator/ (Go)
│   Cloud Run rev 00013-qdj   │  Port 8080
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

| 단계 | 파일 | 포맷 | 동작 |
|------|------|------|------|
| ① 마이크 캡처 | `SpeechAudioCapture` (SpeechRecognizer.swift) | Float32, ~44100Hz, mono | `AVAudioEngine.inputNode` 탭, bufferSize=4096 |
| ② 노이즈 게이트 | 동일 파일 | RMS threshold 0.003 | 배경 소음 필터 (0.003 미만 drop) |
| ③ 포맷 변환 | `AppDelegate.convertAudioBufferToPCM16k()` | Int16, 16000Hz, mono | `AVAudioConverter` (serial queue) |
| ④ 전송 | `GatewayClient.sendAudio()` | binary WebSocket frame, 3200 bytes | `isTTSSpeaking=true`면 차단 |
| ⑤ 게이트웨이 수신 | `handler.go` BinaryMessage case | raw PCM bytes | `modelSpeaking=true`면 Gemini에 전달 안 함 |
| ⑥ Gemini 전달 | `session.SendAudio()` (session.go) | `audio/pcm;rate=16000` MIME | `SendRealtimeInput` → Gemini Live API |

### 출력 경로: Gemini → 스피커

| 단계 | 파일 | 포맷 | 동작 |
|------|------|------|------|
| ① Gemini 응답 | `receiveFromGemini()` (handler.go) | `InlineData.Data` (PCM) | 첫 오디오 → `ttsStart` 전송, `modelSpeaking=true` |
| ② 클라이언트 전송 | 동일 | binary WebSocket frame | `WriteMessage(BinaryMessage, data)` |
| ③ 파싱 | `GatewayClient` → `AudioMessageParser.parse()` | JSON 실패 → `.audio(Data)` | `.data` case에서 처리 |
| ④ 위임 | `CatVoice.enqueueAudio()` | 패스스루 | `AudioPlayer`의 thin wrapper |
| ⑤ 버퍼링 | `AudioPlayer.enqueue()` | 4800 bytes 이상 축적 후 재생 | ~100ms @24kHz coalesce |
| ⑥ 스피커 출력 | `AudioPlayer.scheduleAccumulatedSamples()` | Int16, 24000Hz, mono | `AVAudioPlayerNode` → `mainMixerNode` |

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
        │
        ▼
ScreenCaptureService.captureAroundCursor()
  └── SCScreenshotManager.captureImage() (ScreenCaptureKit)
  └── ImageDiffer.hasSignificantChange() (diff < 0.05면 unchanged)
        │ (변화 감지 시)
        ▼
ImageProcessor.toBase64JPEG() → base64 문자열
        │
        ▼
GatewayClient.sendScreenCapture() (JSON: type=screenCapture)
        │ WebSocket text frame
        ▼
handler.go "screenCapture" case
  └── modelSpeaking=true면 SKIP (발화 중 캡처 억제)
  └── adkClient.Analyze() → HTTP POST /analyze
        │
        ▼
ADK Orchestrator: 9-Agent Graph
  ├── Wave 1 (Parallel): VisionAgent ∥ MemoryAgent
  ├── Wave 2 (Parallel): MoodDetector ∥ CelebrationTrigger
  └── Wave 3 (Sequential): Mediator → Scheduler → Engagement → SearchLoop
        │
        ▼
AnalyzeResult { Decision, Vision, Mood, SpeechText }
        │
        ▼
handler.go: shouldSpeak && SpeechText != ""
  ├── companionSpeech (JSON) → 클라이언트 말풍선
  └── urgency=high/critical → TTS 스트리밍
```

---

## 4. TTS 파이프라인 (Companion Speech)

| 단계 | 위치 | 동작 |
|------|------|------|
| ① 트리거 | handler.go `shouldSpeak && urgency=high` | `startTTSStream()` 호출 |
| ② TTS 요청 | tts/client.go `StreamSpeak()` | `gemini-2.5-flash-preview-tts` 모델, `GenerateContentStream` |
| ③ 스트리밍 | 동일 | 청크 단위로 `sink()` → binary WebSocket |
| ④ 래핑 | handler.go | `ttsStart` (JSON) → 오디오 청크들 → `ttsEnd` (JSON) |
| ⑤ 클라이언트 | GatewayClient | `ttsStart` → `isTTSSpeaking=true`, `ttsEnd` → 500ms 쿨다운 후 해제 |

---

## 5. 9-Agent Graph (ADK Orchestrator)

```
vibecat_graph (SequentialAgent)
  │
  ├── wave1_perception (ParallelAgent)
  │   ├── vision_agent    — 스크린샷 분석 (에러, 성공, 컨텍스트)
  │   └── memory_agent    — 크로스세션 컨텍스트 조회/저장
  │
  ├── wave2_emotion (ParallelAgent)
  │   ├── mood_detector       — 개발자 기분 분류 (focused/frustrated/stuck/idle)
  │   └── celebration_trigger — 성공 이벤트 감지 (테스트 통과, 빌드 성공)
  │
  └── wave3_decision (SequentialAgent)
      ├── mediator           — 말할지 결정 (significance, cooldown, mood 기반)
      ├── adaptive_scheduler — 타이밍 조정 (인터랙션 빈도 기반)
      ├── engagement_agent   — 장시간 침묵 시 프로액티브 개입
      └── search_refinement_loop (LoopAgent, max=2)
          ├── search_buddy      — Google Search (FunctionTool)
          └── llm_search_agent  — LLM Search (GeminiTool)
```

| Agent | ADK Type | Gemini 모델 사용 | 역할 |
|-------|----------|-----------------|------|
| VisionAgent | Custom (Run func) | `gemini-3.1-flash-lite-preview` | 스크린샷 → 에러/컨텍스트/감정 분석 |
| MemoryAgent | Custom (Run func) | `gemini-3.1-flash-lite-preview` | Firestore 세션 기억 조회/저장 |
| MoodDetector | Custom (Run func) | 없음 (규칙 기반) | Vision 결과 → 기분 분류 |
| CelebrationTrigger | Custom (Run func) | `gemini-3.1-flash-lite-preview` | 성공 패턴 감지 |
| Mediator | Custom (Run func) | `gemini-3.1-flash-lite-preview` | shouldSpeak + SpeechText 결정 |
| AdaptiveScheduler | Custom (Run func) | 없음 (규칙 기반) | captureInterval 동적 조정 |
| EngagementAgent | Custom (Run func) | `gemini-3.1-flash-lite-preview` | 침묵 임계값 초과 시 대화 생성 |
| SearchBuddy | Custom (Run func) | `gemini-3.1-flash-lite-preview` | Google Search grounding |
| LLMSearchAgent | LLMAgent | `gemini-3.1-flash-lite-preview` | GeminiTool(GoogleSearch) 사용 |

---

## 6. Gemini Live API VAD 설정

| 설정 | 값 | 의미 |
|------|-----|------|
| `StartOfSpeechSensitivity` | Low | 발화 시작 감지 덜 민감 (배경소음 무시) |
| `EndOfSpeechSensitivity` | Low | 발화 종료 판단 보수적 (짧은 침묵 무시) |
| `PrefixPaddingMs` | 300ms | 발화 시작 전 300ms 오디오 포함 |
| `SilenceDurationMs` | 500ms | 500ms 침묵 시 발화 종료 판단 |
| `ActivityHandling` | StartOfActivityInterrupts | 사용자 발화 감지 시 모델 응답 중단 |
| `TurnCoverage` | TurnIncludesOnlyActivity | 활성 음성만 turn에 포함 |
| `ResponseModalities` | [Audio] | 오디오로만 응답 |
| `OutputAudioTranscription` | 활성화 | 모델 음성의 텍스트 변환 |
| `InputAudioTranscription` | 활성화 | 사용자 음성의 텍스트 변환 |
| `EnableAffectiveDialog` | true (설정 시) | 감정 인식 대화 |
| `Proactivity.ProactiveAudio` | true (설정 시) | 모델 자발적 발화 |
| `ContextWindowCompression` | trigger=4096, target=2048 | 긴 세션 컨텍스트 압축 |
| `SessionResumption` | 활성화 | 세션 재연결 지원 |

---

## 7. 상태 플래그 & 인터럽트 제어

| 플래그 | 위치 | 역할 |
|--------|------|------|
| `isTTSSpeaking` | GatewayClient (클라이언트) | true면 `sendAudio()` 차단 → 서버에 오디오 안 보냄 |
| `modelSpeaking` | liveSessionState (게이트웨이) | true면 ① 클라이언트 오디오 Gemini 전달 차단 ② cancelTTS 억제 ③ screenCapture 억제 |
| `ttsEndCooldownTask` | GatewayClient (클라이언트) | ttsEnd 후 500ms 딜레이 → 스피커 잔향 방지 |
| `ttsCancel` | liveSessionState (게이트웨이) | companion TTS 취소 (사용자 barge-in 시) |
| `turnHasAudio` | receiveFromGemini (게이트웨이) | Gemini turn 내 오디오 존재 여부 추적 |

---

## 8. WebSocket 메시지 프로토콜

### 클라이언트 → 게이트웨이

| 타입 | 형식 | 설명 |
|------|------|------|
| `setup` | JSON | 초기 설정 (voice, language, model, soul, deviceId) |
| binary | Binary | PCM 16kHz 16-bit mono 오디오 |
| `screenCapture` | JSON | base64 스크린샷 + context |
| `forceCapture` | JSON | 강제 분석 (modelSpeaking 중에도 허용) |
| `clientContent` | JSON | 텍스트 입력 (채팅 모드) |
| `settingsUpdate` | JSON | 설정 변경 → 세션 재연결 |
| `ping` | JSON | 앱 레벨 heartbeat (30초) |

### 게이트웨이 → 클라이언트

| 타입 | 형식 | 설명 |
|------|------|------|
| `setupComplete` | JSON | 연결 완료 + sessionId |
| binary | Binary | PCM 24kHz 16-bit mono 오디오 (Gemini Live 또는 TTS) |
| `transcription` | JSON | 모델 음성 → 텍스트 (outputTranscription) |
| `inputTranscription` | JSON | 사용자 음성 → 텍스트 (inputTranscription) |
| `companionSpeech` | JSON | ADK 분석 결과 텍스트 + emotion + urgency |
| `ttsStart` | JSON | TTS/모델 오디오 시작 → 클라이언트 마이크 억제 |
| `ttsEnd` | JSON | TTS/모델 오디오 종료 → 클라이언트 마이크 재개 (500ms 후) |
| `turnComplete` | JSON | Gemini turn 완료 |
| `interrupted` | JSON | Gemini 응답 중단 (barge-in) |
| `sessionResumptionUpdate` | JSON | 세션 재연결 핸들 |
| `liveSessionReconnecting` | JSON | Gemini 세션 재연결 중 (attempt/max) |
| `liveSessionReconnected` | JSON | Gemini 세션 재연결 완료 |
| `goAway` | JSON | Gemini 세션 타임아웃 경고 |
| `pong` | JSON | heartbeat 응답 |
| `error` | JSON | 에러 (code + message) |

---

## 9. 클라이언트 Swift 모듈 구조

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

### App 모듈 (UI + 시스템)

| 파일 | 역할 |
|------|------|
| `AppDelegate.swift` | 앱 진입점, 모든 컴포넌트 조립, 오디오 변환 |
| `GatewayClient.swift` | WebSocket 연결, 메시지 송수신, 재연결 |
| `SpeechRecognizer.swift` | 마이크 캡처 (AVAudioEngine + 노이즈 게이트) |
| `AudioPlayer.swift` | 스피커 출력 (AVAudioPlayerNode, 24kHz) |
| `CatVoice.swift` | AudioPlayer thin wrapper |
| `CatPanel.swift` | 캐릭터 UI (스프라이트 + 말풍선 + 이모지 애니메이션) |
| `SpriteAnimator.swift` | 스프라이트 상태 머신 (idle/thinking/happy/surprised/frustrated/celebrating) |
| `ScreenAnalyzer.swift` | 화면 캡처 주기 관리 + ADK 결과 처리 |
| `ScreenCaptureService.swift` | ScreenCaptureKit 래퍼 |
| `CatViewModel.swift` | 캐릭터 위치/화면 관리 |
| `CompanionChatPanel.swift` | 채팅 모드 UI |
| `StatusBarController.swift` | 메뉴바 아이콘 + 상태 |
| `TrayIconAnimator.swift` | 메뉴바 아이콘 애니메이션 |
| `BackgroundMusicPlayer.swift` | 배경 음악 (별도 파이프라인) |
| `DecisionOverlayHUD.swift` | ADK 결정 오버레이 |
| `OnboardingWindowController.swift` | 초기 설정 UI |
| `CircleGestureDetector.swift` | 원형 제스처 감지 |
| `ErrorReporter.swift` | 에러 로깅 |
| `ChatBubbleView.swift` | 말풍선 뷰 |

---

## 10. 사용 모델

| 용도 | 모델 | 위치 |
|------|------|------|
| Live Voice (VAD + 대화) | `gemini-2.5-flash-native-audio-latest` | Gateway → Live API |
| TTS (Companion Speech) | `gemini-2.5-flash-preview-tts` | Gateway → TTS Client |
| Vision / Search / Agent | `gemini-3.1-flash-lite-preview` | ADK Orchestrator |

---

## 11. 인프라 & GCP

| 리소스 | 서비스 |
|--------|--------|
| Cloud Run | `realtime-gateway`, `adk-orchestrator` (asia-northeast3) |
| Firestore | 세션, 메트릭, 크로스세션 메모리 |
| Secret Manager | `vibecat-gemini-api-key`, `vibecat-gateway-auth-secret` |
| Artifact Registry | `vibecat-images` 컨테이너 |
| Cloud Trace | OpenTelemetry spans |
| Cloud Logging | 구조화 로깅 |

---

## 12. 알려진 이슈 (2026-03-09)

| 이슈 | 심각도 | 설명 |
|------|--------|------|
| 에코 캔슬링 미완 | ⚠️ | `setVoiceProcessingEnabled(false)` → Apple AEC 비활성. `modelSpeaking` 플래그 + 500ms 쿨다운으로 대체 |
| 지직거리는 잡음 | ❓ | 사용자 보고. AudioPlayer 재생 관련 조사 필요 |
| Gemini 음성 응답 | ❓ | `inputTranscription` 수신 확인 (Gemini가 듣고 있음). 오디오 응답 사용자 확인 필요 |
| Barge-in 불가 | 설계상 | `modelSpeaking=true` 동안 모든 오디오 차단 → 사용자 끼어들기 불가 |
