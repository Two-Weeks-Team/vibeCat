# VibeCat 해카톤 최종 전략 — Google 중심 + 전략적 로컬

> Historical note (2026-03-11): 이 문서는 Live Agent 중심 전략 시점의 기록입니다. 현재 제출 전략은 UI Navigator 기준으로 재정렬되었습니다.

**작성일**: 2026-03-06
**분석 근거**: 15개 전문가 에이전트 (코드베이스 7 + 외부 리서치 8) + 소스 직접 검증
**해카톤 마감**: 2026-03-16 17:00 PT (한국시간 3/17 09:00) — **11일**
**최대 점수**: 6.0 (기본 5.0 + 보너스 1.0)

---

## 전략 원칙

> **Google 서비스를 기본으로 사용. 로컬은 속도/기능에 악영향을 줄 때만 사용.**

| 판단 기준 | Google | Local |
|-----------|--------|-------|
| 레이턴시 추가 < 100ms | ✅ Google 사용 | |
| 레이턴시 추가 100-200ms | ✅ 해카톤 점수 > UX 시 | ⚠️ UX 우선 시 |
| 레이턴시 추가 > 200ms | | ✅ Local 사용 |
| 실시간 오디오/비디오 | | ✅ 무조건 Local |
| 보안 (API key) | ✅ Secret Manager (백엔드만) | |
| 사용자 인증 | ✅ 디바이스 UUID → Firestore 매핑 | |
| 심사위원에게 보이는 기능 | ✅ Google 사용 | |

---

## 1. 해카톤 채점 기준 (공식 추출)

### 기본 점수 (5.0 만점)

| 기준 | 비중 | 핵심 평가 항목 | 심사위원이 보는 것 |
|------|------|---------------|-----------------|
| **Innovation & Multimodal UX** | **40%** | 바지인, 페르소나/음성, "Live" 느낌, See/Hear/Speak | 데모 영상에서 자연스러운 대화 |
| **Technical Implementation** | **30%** | GenAI SDK/ADK 활용, **Cloud Run/Vertex AI/Firestore**, 에러 핸들링, 그라운딩 | 코드에서 GCP 서비스 import + 활용 |
| **Demo & Presentation** | **30%** | 문제→솔루션 스토리, 아키텍처 다이어그램, 실제 작동 증거 | GCP Console 스크린샷, 영상 |

### 보너스 (최대 +1.0)

| 항목 | 점수 | 현재 상태 | 필요한 작업 |
|------|------|----------|-----------|
| 블로그 게시 (#GeminiLiveAgentChallenge) | +0.6 | ✅ dev.to 4개 게시 + 1 드래프트 + 3개 추가 예정 | 사용자가 공개 전환 |
| 자동 배포 (IaC) | +0.2 | ✅ `infra/deploy.sh` + `setup.sh` 존재 | 검증만 |
| GDG 멤버십 | +0.2 | ⏳ 미완료 | gdg.community.dev 가입 |

---

## 2. 현재 Google 서비스 활용 현황 (코드 검증)

### ✅ 활성 (5개)

| 서비스 | 패키지 | 위치 | 용도 |
|--------|--------|------|------|
| **GenAI SDK** | `google.golang.org/genai v1.48.0` | Gateway + Orchestrator | Gemini Live API, TTS, Search |
| **ADK** | `google.golang.org/adk v0.5.0` | Orchestrator | 9-agent 그래프 |
| **Firestore** | `cloud.google.com/go/firestore v1.19.0` | Orchestrator | 세션, 메모리, 메트릭스 |
| **Secret Manager** | `cloud.google.com/go/secretmanager v1.14.5` | Gateway | API key, JWT secret |
| **Google Search** | `genai.GoogleSearch{}` | SearchBuddy agent | 웹 검색 그라운딩 |

### ✅ 활성 (추가 4개 — 총 9개)

| 서비스 | 상태 | 비고 |
|--------|------|------|
| **Cloud Run** | ✅ 배포 완료 | Gateway `00010-m9p`, Orchestrator `00011-qj4` |
| **Cloud Logging** | ✅ 명시적 클라이언트 | `cloud.google.com/go/logging` 구조화 로그 |
| **Cloud Trace** | ✅ 명시적 span | OpenTelemetry → Cloud Trace exporter |
| **ADK Telemetry** | ✅ 활성 | `google.golang.org/adk/telemetry` |

### ❌ 미사용 (문서에만 언급)

| 서비스 | 문서 위치 | 코드 |
|--------|----------|------|
| Cloud Storage | `DEPLOYMENT_AND_OPERATIONS.md` | 없음 (P3 — 필요 시) |

---

## 3. 최종 Google 서비스 전략

### ✅ 추가 완료 (5개 → 9개 활성)

| 추가 서비스 | 구현 시간 | 코드량 | 점수 영향 | 데모 가시성 | 상태 |
|------------|----------|--------|----------|-----------|------|
| **Cloud Trace** (명시적 span) | 20분 | ~30줄 | ★★★★★ | 높음 — Trace Explorer 워터폴 | ✅ 완료 |
| **Cloud Logging** (구조화 로그) | 15분 | ~20줄 | ★★★★☆ | 중간 — Logs Explorer | ✅ 완료 |
| **Cloud Monitoring** (커스텀 대시보드) | 30분 | ~25줄 | ★★★★☆ | 매우 높음 — 대시보드 스크린샷 | ✅ 완료 (OTel metric exporter) |
| **ADK Telemetry** (OpenTelemetry) | 10분 | ~10줄 | ★★★☆☆ | 중간 — Trace에 통합 | ✅ 완료 |
| **Cloud Run 배포** | 30분 | 스크립트 실행 | ★★★★★ | 필수 — 배포 증명 | ✅ 완료 |

### 서비스별 구현 상세

#### Cloud Trace — 분산 추적 (Gateway + Orchestrator)
```go
// backend/realtime-gateway/main.go
import (
    texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
    "go.opentelemetry.io/otel"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

exporter, _ := texporter.New(texporter.WithProjectID("vibecat-489105"))
tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter))
otel.SetTracerProvider(tp)
```
```go
// handler.go — 각 핵심 경로에 span 추가
ctx, span := otel.Tracer("gateway").Start(ctx, "websocket.screenCapture")
defer span.End()
span.SetAttributes(attribute.String("session_id", sessionID))
```
**데모 효과**: Trace Explorer에서 Client→Gateway→Orchestrator→Gemini 전체 요청 흐름 시각화

#### Cloud Logging — 구조화 로그
```go
import "cloud.google.com/go/logging"
client, _ := logging.NewClient(ctx, "vibecat-489105")
logger := client.Logger("realtime-gateway")
logger.Log(logging.Entry{
    Payload: map[string]any{
        "event": "agent_decision",
        "agent": "mediator",
        "decision": "speak",
        "urgency": "high",
    },
    Severity: logging.Info,
})
```
**데모 효과**: Logs Explorer에서 에이전트 결정 과정 실시간 추적

#### Cloud Monitoring — 커스텀 메트릭
```go
import monitoring "cloud.google.com/go/monitoring/apiv3/v2"
// 커스텀 메트릭: 에이전트별 실행 시간, 세션 활성 수, 감정 상태 분포
```
**데모 효과**: Monitoring 대시보드에서 에이전트 성능 시각화

---

## 4. ADK 기능 확장 전략

### ✅ 현재 사용 (14개 기능 — 모두 구현 완료)

| 기능 | Import | 용도 | 상태 |
|------|--------|------|------|
| `agent.New()` | `google.golang.org/adk/agent` | 커스텀 에이전트 생성 | ✅ |
| `sequentialagent.New()` | `.../workflowagents/sequentialagent` | 순차 그래프 | ✅ |
| `parallelagent.New()` | `.../workflowagents/parallelagent` | 병렬 에이전트 (Wave 1, 2) | ✅ |
| `llmagent.New()` | `google.golang.org/adk/agent/llmagent` | LLM 기반 검색 에이전트 | ✅ |
| `session.InMemoryService()` | `google.golang.org/adk/session` | 세션 관리 | ✅ |
| `memory.InMemoryService()` | `google.golang.org/adk/memory` | 크로스 세션 메모리 | ✅ |
| `runner.New()` | `google.golang.org/adk/runner` | 공식 런타임 | ✅ |
| `telemetry.New()` | `google.golang.org/adk/telemetry` | OpenTelemetry 연동 | ✅ |
| `session.State/Event` | `google.golang.org/adk/session` | 에이전트 간 상태 공유 | ✅ |
| `functiontool.New()` | `google.golang.org/adk/tool/functiontool` | 타입 안전 도구 정의 | ✅ |
| `geminitool.GoogleSearch{}` | `google.golang.org/adk/tool/geminitool` | 네이티브 검색 그라운딩 | ✅ |
| `loopagent.New()` | `.../workflowagents/loopagent` | 검색 반복 정제 | ✅ |
| `retryandreflect` plugin | `google.golang.org/adk/plugin/retryandreflect` | 자동 반성+재시도 | ✅ |
| `BeforeModel/AfterModel` | `google.golang.org/adk/agent/llmagent` | 가드레일, 로깅 콜백 | ✅ |

### 향후 추가 가능한 ADK 기능

| 기능 | Import | 코드량 | 점수 영향 | 용도 |
|------|--------|--------|----------|------|
| **runner.New()** | `adk/runner` | ~25줄 | ★★★★★ | 공식 런타임 — 세션/메모리/텔레메트리 통합 |
| **parallelagent.New()** | `.../parallelagent` | ~15줄 | ★★★★☆ | Vision+Memory 병렬, Mood+Celebration 병렬 |
| **functiontool.New()** | `adk/tool/functiontool` | ~25줄/도구 | ★★★★★ | 타입 안전 도구 정의 (스크린 분석, 메모리 조회) |
| **geminitool.GoogleSearch** | `adk/tool/geminitool` | ~5줄 | ★★★★☆ | 네이티브 검색 그라운딩 |
| **Callbacks** | `agent/llmagent` | ~30줄 | ★★★★★ | BeforeModel/AfterModel — 가드레일, 로깅 |
| **memory.InMemoryService()** | `adk/memory` | ~15줄 | ★★★★☆ | 크로스 세션 컨텍스트 |
| **telemetry** | `adk/telemetry` | ~10줄 | ★★★☆☆ | OpenTelemetry → Cloud Trace 연동 |

### ADK 아키텍처 (현재 — 구현 완료 ✅)

```
runner.New(Config{
  Agent: sequentialagent → [
    parallelagent → [Vision, Memory]        ← Wave 1: 독립 에이전트 병렬 ✅
    parallelagent → [Mood, Celebration]      ← Wave 2: 독립 에이전트 병렬 ✅
    sequentialagent → [Mediator, Scheduler, Engagement, SearchBuddy, LLMSearchBuddy]  ← Wave 3: 순차 ✅
  ],
  SessionService: session.InMemoryService(),  ✅
  MemoryService: memory.InMemoryService(),    ✅
})
```

**레이턴시 개선**: Wave 1+2 병렬화로 에이전트 그래프 실행 시간 ~35% 감소 (3.5s → 2.1-2.5s)

### Dynamic Message Generation (하드코딩 제거 완료 ✅)

| 에이전트 | 기능 | LLM 모델 |
|---------|------|----------|
| Mediator | 기분 지원 메시지 동적 생성 | `gemini-3.1-flash-lite-preview` |
| CelebrationTrigger | 축하 메시지 동적 생성 | `gemini-3.1-flash-lite-preview` |
| EngagementAgent | 침묵 깨기 메시지 동적 생성 | `gemini-3.1-flash-lite-preview` |

모든 에이전트가 사용자의 맥락(현재 작업, 분위기, 언어)에 맞춤 메시지를 LLM으로 생성. 하드코딩된 메시지 풀은 LLM 실패 시 fallback으로만 사용.

### ADK Runner 구현

```go
import (
    "google.golang.org/adk/runner"
    "google.golang.org/adk/session"
    "google.golang.org/adk/memory"
)

r, _ := runner.New(runner.Config{
    AppName:        "vibecat-orchestrator",
    Agent:          graphAgent,  // 위의 parallelagent 포함 그래프
    SessionService: session.InMemoryService(),
    MemoryService:  memory.InMemoryService(),
})

// 에이전트 실행
for event, err := range r.Run(ctx, userID, sessionID, userMsg, agent.RunConfig{}) {
    // 스트리밍 이벤트 처리
}
```

### FunctionTool 예시

```go
import "google.golang.org/adk/tool/functiontool"

type ScreenInput struct {
    ImageBase64 string `json:"image_base64" description:"Base64 encoded screenshot"`
    AppName     string `json:"app_name" description:"Foreground application name"`
}
type ScreenOutput struct {
    Description string `json:"description"`
    ErrorFound  bool   `json:"error_found"`
    Significance int   `json:"significance"`
}

screenTool, _ := functiontool.New(functiontool.Config{
    Name:        "analyze_screen",
    Description: "Analyze a screenshot for errors, changes, and context",
}, func(ctx tool.Context, input ScreenInput) (ScreenOutput, error) {
    // Gemini Vision 호출
    return analyzeWithGemini(ctx, input)
})
```

### Callbacks 예시

```go
import "google.golang.org/adk/agent/llmagent"

agent, _ := llmagent.New(llmagent.Config{
    // ...
    BeforeModelCallback: func(ctx agent.CallbackContext, req *model.LLMRequest) (*model.LLMResponse, error) {
        // Cloud Logging으로 요청 로깅
        logger.Log(logging.Entry{Payload: map[string]any{"agent": ctx.Agent().Name(), "action": "before_model"}})
        // Cloud Trace span 추가
        _, span := otel.Tracer("adk").Start(ctx.Context(), "llm_call."+ctx.Agent().Name())
        defer span.End()
        return nil, nil  // nil = 정상 진행
    },
    AfterModelCallback: func(ctx agent.CallbackContext, resp *model.LLMResponse, err error) (*model.LLMResponse, error) {
        // 토큰 사용량 메트릭 기록
        logger.Log(logging.Entry{Payload: map[string]any{"agent": ctx.Agent().Name(), "tokens": resp.UsageMetadata}})
        return resp, nil
    },
})
```

---

## 5. Live API 기능 전략

### 현재 상태 (코드 검증 완료)

| 기능 | 상태 | 코드 위치 |
|------|------|----------|
| VAD | ✅ 활성 | `session.go:158-169` Low/Low, 300ms/500ms |
| Barge-in | ✅ 활성 | `session.go:167` StartOfActivityInterrupts |
| ProactiveAudio | ✅ 활성 | `session.go:148-153`, Settings.swift default true |
| OutputAudioTranscription | ✅ 활성 | `session.go:155` |
| InputAudioTranscription | ✅ 활성 | `session.go:156` |
| ContextWindowCompression | ✅ 활성 | `session.go:171-178` trigger 4096, target 2048 |
| SessionResumption | ✅ 활성 | `session.go:58-64` 핸들 기반 |
| **AffectiveDialog** | ✅ **활성** | `session.go:143-146` 백엔드 + 클라이언트 전송 완료 |

### 수정 사항 (1건)

**AffectiveDialog 활성화** — `GatewayClient.swift:288-293` setup 메시지에 추가:
```swift
// 현재
let setup: [String: Any] = [
    "voice": settings.voice,
    "language": settings.language,
    "liveModel": settings.liveModel,
    "proactiveAudio": settings.proactiveAudio,
    "searchEnabled": settings.searchEnabled
]

// 수정 후
let setup: [String: Any] = [
    "voice": settings.voice,
    "language": settings.language,
    "liveModel": settings.liveModel,
    "proactiveAudio": settings.proactiveAudio,
    "searchEnabled": settings.searchEnabled,
    "affectiveDialog": true  // ← 추가
]
```

**효과**: Gemini가 사용자 음성 톤에서 감정 자동 감지 (좌절, 흥분, 불확실함) → MoodDetector 텍스트 분석 보완

---

## 6. 로컬 유지 항목 (Google로 이동하면 안 되는 것)

### ✅ 무조건 로컬 (속도/기능 악영향)

| 항목 | 프레임워크 | 레이턴시 | 이유 |
|------|-----------|---------|------|
| 마이크 캡처 | AVAudioEngine | 실시간 | 클라우드 스트리밍 시 50-200ms 추가 |
| 오디오 재생 | AVAudioPlayerNode | 실시간 | 네트워크 지터 시 재생 품질 붕괴 |
| RMS 노이즈 게이팅 | AVAudioPCMBuffer | ~1ms | 대역폭 최적화 (노이즈 미전송) |
| 이미지 차이 감지 | CoreGraphics 32x32 | ~2-5ms | 중복 프레임 전송 방지 |
| 스크린 캡처 | ScreenCaptureKit | ~50ms | OS 레벨 API, 클라우드 불가 |
| JPEG 인코딩 | CoreGraphics/ImageIO | ~10-30ms | 클라우드 시 원본 전송 필요 (10x 대역폭) |
| API Key 저장 | Keychain | 즉시 | 보안 — 네트워크 전송 금지 |
| WebSocket 관리 | URLSessionWebSocketTask | 즉시 | 네트워킹 레이어 |
| JSON 파싱 | JSONSerialization | <1ms | 라운드트립 불필요 |
| 스프라이트/음악 로딩 | FileManager | 즉시 | 앱 번들 에셋 |

### ✅ 로컬 우선 + Google 동기화 (하이브리드)

| 항목 | 로컬 | Google | 이유 |
|------|------|--------|------|
| **컨텍스트 메모리** | SQLite (GRDB.swift) — source of truth, <1ms | Firestore — 관찰성/심사용 | 세션 시작 시 즉시 로드 필요 |
| **설정** | UserDefaults — 11개 키, 즉시 읽기 | Firestore — 데모 시 "클라우드 동기화" 보여주기 | 설정 변경은 드물어 클라우드 레이턴시 무관 |
| **세션 메트릭스** | 인메모리 (StatusBarController) | Firestore — 세션 종료 시 벌크 저장 | 실시간 집계는 로컬, 영구 저장은 클라우드 |

---

## 7. 4개 모드 최종 설계

### Mode 1: VAD 음성 대화 (~2s)

- **Google**: Gemini Live API (VAD, AffectiveDialog, Barge-in, ProactiveAudio, OutputTranscription)
- **Local**: 마이크 캡처, RMS 게이팅, 오디오 재생, PCM 변환
- **변경**: AffectiveDialog 활성화 (1줄), OutputTranscription → CatPanel 말풍선 연동

### Mode 2: 스크린 제안 — urgency 기반 모달리티

- **Google**: ADK 9-agent 그래프 (Vision, Mediator, Mood, Celebration), Gemini TTS
- **Local**: 스크린 캡처, 이미지 차이 감지, JPEG 인코딩
- **변경**: `handler.go`에서 urgency 기반 TTS 분기
  - `significance < 7` → 말풍선만 (4.5s)
  - `errorDetected / frustrated / celebration` → 음성+말풍선 (8.8s)
  - 동적 말풍선 표시 시간: `max(3.0, text.count / 5.0)`

### Mode 3: Apple Watch형 프로액티브 인게이지먼트

- **Google**: EngagementAgent (ADK), MoodDetector (ADK), CelebrationTrigger (ADK)
- **Local**: 활동 시간 추적 (타이머), macOS Focus Mode 확인
- **4-Tier 계층**:
  - Tier 1 앰비언트: 고양이 애니메이션만 (상시, 로컬)
  - Tier 2 마이크로 넛지: 짧은 음성 3-5초 (좌절/막힘, Google ADK)
  - Tier 3 휴식 알림: 음성+말풍선 (50분, Google ADK)
  - Tier 4 축하: 음성+흥분 애니메이션 (성공 감지, Google ADK)

### Mode 4: 컨텍스트 메모리 — 하이브리드

- **Local (주)**: SQLite (GRDB.swift) — sessions, activity_patterns, memory_chunks
- **Google (부)**: Firestore — 메트릭스 저장, AI 요약 생성 (ADK MemoryAgent)
- **Google (추가)**: ADK `memory.InMemoryService()` — 크로스 세션 검색
- **플로우**:
  1. 세션 시작 → SQLite에서 최근 맥락 즉시 로드 (<1ms)
  2. Setup 메시지에 맥락 포함 → Gateway → Gemini system instruction 주입
  3. 세션 종료 → ADK MemoryAgent가 AI 요약 생성 (Gemini)
  4. 요약 → 클라이언트 SQLite 저장 + Firestore 메트릭스 저장

---

## 8. 구현 타임라인 (11일)

### Day 1-2: P0 — 즉시 적용 (최소 노력, 최대 점수) — ✅ 전체 완료

| 작업 | 코드량 | Google 서비스 | 점수 영향 | 상태 |
|------|-------|-------------|----------|------|
| AffectiveDialog 활성화 | 1줄 | Live API | Innovation ★★★★★ | ✅ |
| **디바이스 UUID 인증 플로우** (Keychain 제거) | ~50줄 | Firestore + Secret Manager | Technical ★★★★★ | ✅ |
| Cloud Trace 명시적 span 추가 (Gateway + Orchestrator) | ~30줄 | Cloud Trace | Technical ★★★★★ | ✅ |
| Cloud Logging 구조화 로그 (에이전트 결정 로깅) | ~20줄 | Cloud Logging | Technical ★★★★☆ | ✅ |
| urgency 기반 모달리티 분기 | ~10줄 | ADK (Mediator) | Innovation ★★★★☆ | ✅ |
| OutputTranscription → CatPanel 말풍선 | ~15줄 | Live API | Innovation ★★★☆☆ | ✅ |
| 동적 말풍선 표시 시간 | ~5줄 | — | UX ★★★☆☆ | ✅ |

### Day 3-5: P1 — ADK 기능 확장 + 메모리 — 대부분 완료

| 작업 | 코드량 | Google 서비스 | 점수 영향 | 상태 |
|------|-------|-------------|----------|------|
| ADK Runner 도입 | ~25줄 | ADK Runner | Technical ★★★★★ | ✅ |
| parallelagent 적용 (Vision+Memory, Mood+Celebration) | ~15줄 | ADK ParallelAgent | Technical ★★★★☆ | ✅ |
| functiontool로 에이전트 도구 정의 (3개) | ~75줄 | ADK FunctionTool | Technical ★★★★★ | ✅ |
| geminitool.GoogleSearch 네이티브 통합 | ~5줄 | ADK GeminiTool | Technical ★★★★☆ | ✅ |
| BeforeModel/AfterModel 콜백 (로깅/가드레일) | ~30줄 | ADK Callbacks | Technical ★★★★★ | ✅ 완료 (search agent) |
| **ADK `retryandreflect` 플러그인** | ~10줄 | ADK Plugin | Technical ★★★★☆ | ✅ 완료 (main.go plugin) |
| **ADK `loopagent` 반복 정제** | ~15줄 | ADK LoopAgent | Technical ★★★☆☆ | ✅ 완료 (search refinement) |
| ADK memory.InMemoryService 연동 | ~15줄 | ADK Memory | Technical ★★★★☆ | ✅ |
| **멀티모달 감정 융합** (AffectiveDialog + MoodDetector + 스크린) | ~40줄 | ADK (Orchestrator) | Innovation ★★★★★ | ✅ 완료 (voice fusion in mood.go) |
| 로컬 SQLite (GRDB.swift) 구현 | ~200줄 | — (로컬) | UX ★★★★☆ | ❌ 미구현 (P3) |
| 휴식 알림 인게이지먼트 (시간 추적 + 프로액티브 알림) | ~150줄 | ADK (확장) | Innovation ★★★★★ | ✅ 완료 (rest reminder pipeline) |
| **컨텍스트 인식 침묵** (Flow state 감지 → 쿨다운 연장) | ~15줄 | ADK (Mediator) | Innovation ★★★★☆ | ✅ 완료 (flow state cooldown) |

### Day 6-8: P2 — 배포 + 데모 준비 — 대부분 완료

| 작업 | 점수 영향 | 상태 |
|------|----------|------|
| Cloud Run 배포 (`./infra/deploy.sh`) | Technical ★★★★★ (필수) | ✅ 배포 완료 (00010/00011) |
| Cloud Monitoring 커스텀 대시보드 | Technical ★★★★☆ | ✅ 완료 (OTel metric exporter) |
| 아키텍처 다이어그램 (Mermaid) | Demo ★★★★★ | ✅ README.md에 포함 |
| GCP Console 증거 수집 (Trace, Logs, Monitoring, Firestore) | Demo ★★★★★ | ⏳ 사용자 작업 |
| 데모 영상 스토리보드 작성 | Demo ★★★★☆ | ✅ `docs/analysis/DEMO_STORYBOARD.md` |
| GDG 멤버십 가입 | Bonus +0.2 | ⏳ 사용자 작업 |

### Day 9-10: P3 — 최종 제출물

| 작업 | 점수 영향 |
|------|----------|
| 데모 영상 촬영 (4분, 영어 자막) | Demo 30% |
| dev.to 최종 블로그 게시 (#GeminiLiveAgentChallenge) | Bonus +0.6 |
| README.md 최종 업데이트 (아키텍처, 스크린샷) | Demo ★★★★☆ |
| DevPost 제출 | — |

### Day 11: 버퍼

| 작업 |
|------|
| 통합 테스트, 버그 수정 |
| 실기기 UX 테스트 |
| 영상 재촬영 (필요 시) |

---

## 9. 6.0 만점 달성 전략

### 이전 예측: 5.2/6.0 — 갭 분석

| 영역 | 이전 예측 | 만점 | 갭 | 원인 |
|------|----------|------|-----|------|
| Innovation | 1.8 | 2.0 | -0.2 | 감정 융합/제로 온보딩 부재 |
| Technical | 1.3 | 1.5 | -0.2 | ADK 심화 활용 부족, 클라이언트 보안 |
| Demo | 1.1 | 1.5 | -0.4 | 스토리텔링 약, GCP 증거 부족 |
| 보너스 | +1.0 | +1.0 | 0 | 완성 |
| **합계** | **5.2** | **6.0** | **-0.8** | |

### 갭 해소 전략

#### Innovation +0.2 → 2.0/2.0

| 추가 구현 | 효과 | 코드량 |
|-----------|------|--------|
| **제로 온보딩**: 디바이스 UUID 자동 인식 → API key 입력 불필요 | "설치 즉시 사용" — 경쟁작 대비 차별화 | ~30줄 (Swift) + ~20줄 (Go) |
| **멀티모달 감정 융합**: AffectiveDialog(음성톤) + MoodDetector(텍스트) + 스크린(에러빈도) → 통합 감정 점수 | "3채널 감정 인식" — 단일 채널 대비 정확도 상승 | ~40줄 (Orchestrator) |
| **컨텍스트 인식 침묵**: Flow state 감지 시 자동 쿨다운 연장 | "방해하지 않는 AI" — Apple Watch 원칙 | ~15줄 (Mediator) |

#### Technical +0.2 → 1.5/1.5

| 추가 구현 | 효과 | 코드량 |
|-----------|------|--------|
| **디바이스 인증 플로우**: UUID → Firestore `devices/` → Secret Manager | 클라이언트에 API key 완전 부재 — 보안 만점 | ~50줄 |
| **ADK `retryandreflect` 플러그인**: 에이전트 실패 시 자동 반성+재시도 | self-healing — 프로덕션 품질 | ~10줄 |
| **ADK `loopagent`**: 복잡한 스크린 분석 시 반복 정제 | 정밀 분석 — ADK 기능 커버리지 확대 | ~15줄 |
| **Firestore `devices/` 컬렉션**: 사용자-디바이스 매핑 + 설정 동기화 | Google Cloud 활용도 추가 | ~20줄 |

#### Demo +0.4 → 1.5/1.5

| 개선 | 효과 |
|------|------|
| **Before/After 구성**: 첫 30초 "VibeCat 없이 코딩" (조용, 혼자) → "VibeCat 켜기" → 고양이 등장+인사 | 감정적 대비 → 심사위원에게 "아 이게 필요하겠다" |
| **스플릿 스크린**: 앱 화면 + GCP Console 동시 표시 (Trace 워터폴 실시간) | 기술력 + 실제 작동 동시 증명 |
| **9-agent 실행 시각화**: Cloud Trace에서 Vision→Mood→Celebration→Mediator 워터폴 | "9개 에이전트가 실제로 돌아간다" 증거 |
| **블로그 콜백**: "the chair is still empty, but not for long" → 데모 마지막에 "the chair is no longer empty" | 스토리 마무리 |
| **캐릭터 실시간 전환**: cat→trump 전환 시 음성/성격 즉시 변화 데모 | 6개 캐릭터의 실제 차이 증명 |

### 수정된 점수 예측: 6.0/6.0

#### Innovation & Multimodal UX (40% = 2.0점)

| 요소 | 구현 | 예상 |
|------|------|------|
| VAD + Barge-in + AffectiveDialog | P0 완성 | 0.35/0.4 |
| 멀티모달 감정 융합 (음성+텍스트+스크린) | P1 구현 | 0.35/0.4 |
| 6개 캐릭터 페르소나 + 실시간 전환 | ✅ 완성 | 0.35/0.4 |
| See/Hear/Speak 완전 통합 | ✅ 완성 | 0.3/0.3 |
| Apple Watch 프로액티브 + 컨텍스트 인식 침묵 | P1 구현 | 0.35/0.4 |
| 제로 온보딩 (디바이스 UUID) | P0 구현 | 0.1/0.1 |
| **소계** | | **1.8/2.0** |

#### Technical Implementation (30% = 1.5점)

| 요소 | 구현 | 예상 |
|------|------|------|
| GenAI SDK: Live API + TTS + Vision (3개 모델) | ✅ + P0 | 0.25/0.3 |
| ADK: Runner + ParallelAgent + FunctionTool + Callbacks + Memory + LoopAgent + RetryReflect | P1 확장 | 0.3/0.3 |
| Cloud Run 배포 (2 서비스) | P2 | 0.15/0.15 |
| Firestore (세션/메모리/메트릭/디바이스) | ✅ + P1 | 0.15/0.15 |
| Cloud Trace + Logging + Monitoring (풀 옵저버빌리티) | P0-P2 | 0.2/0.2 |
| Secret Manager + 디바이스 인증 (zero-client-secret) | P0 | 0.15/0.15 |
| Google Search 그라운딩 (geminitool 네이티브) | P1 | 0.1/0.1 |
| 에러 핸들링 (Circuit breaker + retryandreflect) | ✅ + P1 | 0.15/0.15 |
| **소계** | | **1.45/1.5** |

#### Demo & Presentation (30% = 1.5점)

| 요소 | 구현 | 예상 |
|------|------|------|
| Before/After 스토리 ("빈 의자" → "동료") | P3 영상 | 0.4/0.4 |
| 아키텍처 다이어그램 (3-layer + 9-agent) | P2 | 0.2/0.2 |
| GCP 실제 작동 증거 (스플릿 스크린 + Console) | P3 | 0.35/0.4 |
| Cloud Trace 9-agent 워터폴 시각화 | P2 | 0.15/0.15 |
| 코드 품질 + README | ✅ | 0.2/0.2 |
| 캐릭터 실시간 전환 데모 | P3 | 0.1/0.15 |
| **소계** | | **1.4/1.5** |

#### 총합

| 항목 | 점수 |
|------|------|
| Innovation | 1.9/2.0 |
| Technical | 1.5/1.5 |
| Demo | 1.4/1.5 |
| **기본 합계** | **4.8/5.0** |
| 블로그 보너스 | +0.6 |
| 자동 배포 보너스 | +0.2 |
| GDG 보너스 | +0.2 |
| **최종 예측** | **5.8/6.0** |

> **보수적 예측 5.8, 낙관적 예측 6.0**. 0.2 차이는 데모 영상 품질과 심사위원 주관에 의존.
> **업데이트 (2026-03-09)**: Cloud Monitoring, ADK BeforeModel/AfterModel, retryandreflect, loopagent, 멀티모달 감정 융합, 휴식 알림, 컨텍스트 인식 침묵 모두 구현 완료. Technical 만점 달성.

---

## 10. Google 서비스 전체 목록 (최종)

### 런타임 서비스 (10개)

| # | 서비스 | 용도 | 새로 추가? |
|---|--------|------|----------|
| 1 | **Gemini Live API** (GenAI SDK) | 음성 대화, VAD, Barge-in, AffectiveDialog | 기존 (AffectiveDialog 활성화) |
| 2 | **Gemini TTS** (GenAI SDK) | 텍스트→음성 변환 | 기존 |
| 3 | **Gemini Vision** (GenAI SDK) | 스크린 캡처 분석 | 기존 |
| 4 | **Google ADK** | 9-agent + Runner + ParallelAgent + LLMAgent + FunctionTool + Memory + Telemetry (11개 기능) | 기존 (**11개 기능으로 확장**) |
| 5 | **Cloud Firestore** | 세션, 메모리, 메트릭스, **디바이스 매핑**, 설정 동기화 | 기존 (**devices/ 추가**) |
| 6 | **Secret Manager** | API key, JWT secret (**클라이언트 부재, 백엔드만**) | 기존 (보안 강화) |
| 7 | **Google Search** | 웹 검색 그라운딩 (geminitool 네이티브) | 기존 (geminitool 전환) |
| 8 | **Cloud Trace** | 분산 추적 (Gateway→Orchestrator→9-agent 워터폴) | ✅ 구현 완료 |
| 9 | **Cloud Logging** | 구조화 에이전트 결정 로그 | ✅ 구현 완료 |
| 10 | **Cloud Monitoring** | 커스텀 메트릭 대시보드 (OTel metric exporter) | ✅ 구현 완료 |

### 인프라 서비스 (4개)

| # | 서비스 | 용도 |
|---|--------|------|
| 11 | **Cloud Run** | Gateway + Orchestrator 호스팅 |
| 12 | **Cloud Build** | 컨테이너 빌드 |
| 13 | **Artifact Registry** | 컨테이너 이미지 저장 |
| 14 | **IAM** | 서비스 계정, 역할 바인딩 |

### 로컬 전용 (Google로 이동하지 않음)

| # | 항목 | 프레임워크 | 이유 |
|---|------|-----------|------|
| 1 | 마이크 캡처 | AVAudioEngine | 실시간 — 클라우드 불가 |
| 2 | 오디오 재생 | AVAudioPlayerNode | 실시간 — 지터 불가 |
| 3 | 스크린 캡처 | ScreenCaptureKit | OS API — 클라우드 불가 |
| 4 | 이미지 처리 | CoreGraphics | 레이턴시 <30ms — 클라우드 시 3x |
| 5 | RMS 노이즈 게이팅 | AVAudioPCMBuffer | 대역폭 최적화 |
| 6 | 컨텍스트 메모리 (주) | SQLite (GRDB.swift) | <1ms 조회 — Firestore 50-100ms |
| 7 | 설정 (즉시 읽기) | UserDefaults | <1ms — Firestore 동기화는 비동기 |

> **Keychain 제거**: API key는 클라이언트에 저장하지 않음. Secret Manager(백엔드)에만 존재.
> 사용자 식별은 디바이스 UUID → Firestore `devices/{uuid}` 매핑으로 처리.

---

## 11. 데모 영상 전략 (4분)

### 구성 — "Before/After" 내러티브 (30초 단위)

| 시간 | 장면 | 보여줄 것 | 감정 |
|------|------|----------|------|
| 0:00-0:20 | **Before**: 빈 의자 | 혼자 코딩, 에러 반복, 좌절, 아무도 도와주지 않음 | 공감 |
| 0:20-0:50 | **After**: VibeCat 켜기 | 앱 실행 → 고양이 등장 → "어제 인증 모듈 작업했지?" (SQLite 메모리) | 놀라움 |
| 0:50-1:20 | 음성 대화 | 자연스러운 대화 + Barge-in + AffectiveDialog 감정 반응 | 자연스러움 |
| 1:20-1:50 | 스크린 분석 | 에러 감지 → 음성+말풍선, 일반 변화 → 말풍선만 (urgency 분기) | 유용함 |
| 1:50-2:10 | 캐릭터 전환 | cat → trump 실시간 전환, 음성/성격 즉시 변화 | 재미 |
| 2:10-2:40 | 프로액티브 | 50분 휴식 알림 + 테스트 통과 축하 + Flow state 침묵 | 세심함 |
| 2:40-3:20 | **GCP 스플릿 스크린** | 앱 화면 + Cloud Trace 9-agent 워터폴 동시 표시 | 기술력 |
| 3:20-3:50 | GCP Console 증거 | Firestore 데이터 + Logs Explorer + Monitoring 대시보드 | 신뢰 |
| 3:50-4:00 | 클로징 | "the chair is no longer empty" — 블로그 콜백 | 감동 |

### GCP Console 증거 스크린샷 (필수)

| 스크린샷 | 무엇을 보여주는지 |
|---------|----------------|
| Cloud Run 서비스 목록 | 두 서비스가 asia-northeast3에서 실행 중 |
| Firestore 데이터 | sessions 컬렉션에 실제 세션 데이터 |
| Trace Explorer | Client→Gateway→Orchestrator 요청 워터폴 |
| Logs Explorer | 에이전트 결정 구조화 로그 |
| Monitoring 대시보드 | 에이전트 성능 커스텀 메트릭 |
| Secret Manager | 시크릿 목록 (값 가림) |

---

## 12. 디바이스 인증 플로우 (Keychain 대체)

### 아키텍처

```
macOS Client                          Backend (Gateway)
    │                                      │
    ├─ identifierForVendor ──────────────► │
    │  (디바이스 UUID, 자동)                 │
    │                                      ├─ Firestore devices/{uuid} 조회
    │                                      │  └─ 없으면 자동 생성 (첫 연결)
    │                                      │
    │                                      ├─ Secret Manager에서 API key 조회
    │                                      │  (클라이언트에 전달 안 함)
    │                                      │
    │  ◄── WebSocket 연결 승인 ────────────┤
    │  (세션 토큰 발급)                      │
    │                                      ├─ genai.Live.Connect(apiKey)
    │                                      │
    └─ 음성/스크린 데이터 ────────────────► └─ Gemini Live API
```

### 구현 상세

**클라이언트 (Swift)**:
```swift
// Keychain 제거 — API key 없음
// 디바이스 UUID 자동 생성
let deviceID = Host.current().localizedName ?? UUID().uuidString

// Gateway 연결 시 UUID만 전송
let setup: [String: Any] = [
    "deviceId": deviceID,
    "voice": settings.voice,
    "language": settings.language,
    // ... (API key 없음)
]
```

**백엔드 (Go Gateway)**:
```go
// 디바이스 UUID로 사용자 식별
deviceID := setupMsg["deviceId"].(string)

// Firestore에서 디바이스 설정 조회 (없으면 자동 생성)
deviceDoc := firestoreClient.Collection("devices").Doc(deviceID)
// ...

// API key는 Secret Manager에서만 조회
apiKey := secretsClient.LoadSecret("vibecat-gemini-api-key")
```

### 해카톤 점수 효과

| 항목 | 개선 |
|------|------|
| Technical: 보안 | "API key never on client" — 심사위원 체크리스트 충족 |
| Technical: Google Cloud | Firestore `devices/` 컬렉션 추가 활용 |
| Innovation: UX | 제로 온보딩 — 설치 즉시 사용 |
| Demo: 스토리 | "보안까지 고려한 프로덕션 수준 설계" |

---

## 13. 리스크 및 대응

| 리스크 | 확률 | 영향 | 대응 |
|--------|------|------|------|
| Cloud Run 배포 실패 | 중간 | 높음 | Day 6에 배포, Day 11 버퍼 |
| ADK Runner 호환성 이슈 | 낮음 | 중간 | 기존 sequentialagent 유지 가능 |
| SQLite + GRDB.swift SPM 충돌 | 낮음 | 중간 | UserDefaults 폴백 |
| 데모 영상 촬영 실패 | 낮음 | 높음 | Day 11 재촬영 버퍼 |
| GCP 크레딧 부족 | 중간 | 높음 | 3/13까지 크레딧 신청 |

---

## 참고 자료

### Live API
- https://ai.google.dev/gemini-api/docs/live
- https://ai.google.dev/gemini-api/docs/live-guide
- https://pkg.go.dev/google.golang.org/genai

### ADK Go SDK
- https://google.github.io/adk-docs/
- https://google.github.io/adk-docs/get-started/go/
- https://pkg.go.dev/google.golang.org/adk
- https://github.com/google/adk-go

### GCP 서비스
- Cloud Trace: https://cloud.google.com/trace/docs
- Cloud Logging: https://cloud.google.com/logging/docs
- Cloud Monitoring: https://cloud.google.com/monitoring/docs
- Firestore: https://cloud.google.com/firestore/docs

### 해카톤
- https://geminiliveagentchallenge.devpost.com/
- GCP 크레딧 신청 (3/13 마감): https://forms.gle/rKNPXA1o6XADvQGb7
- GDG 가입: https://gdg.community.dev/
