# VibeCat 4-Mode 상호작용 설계 및 해카톤 최종 전략

**작성일**: 2026-03-06
**분석 근거**: 10개 전문가 에이전트 (코드베이스 4 + 외부 리서치 6) + 코드 직접 검증
**해카톤 마감**: 2026-03-16 17:00 PT (한국시간 3/17 09:00) — 11일 남음
**최대 점수**: 6.0 (기본 5.0 + 보너스 1.0)

---

## 해카톤 채점 기준

| 기준 | 비중 | Live Agent 카테고리 핵심 |
|------|------|-------------------------|
| **Innovation & Multimodal UX** | **40%** | 바지인 자연스러움, 고유 페르소나/음성, "Live" 느낌, See/Hear/Speak |
| **Technical Implementation** | **30%** | GenAI SDK/ADK 활용, Cloud Run/Firestore, 에러 핸들링, 그라운딩 |
| **Demo & Presentation** | **30%** | 문제-솔루션 스토리, 아키텍처 다이어그램, 실제 작동 증거 |
| 보너스: 블로그 | +0.6 | dev.to/Medium에 #GeminiLiveAgentChallenge 포함 공개 게시 |
| 보너스: 자동 배포 | +0.2 | infra/deploy.sh 또는 Terraform |
| 보너스: GDG 멤버십 | +0.2 | gdg.community.dev 프로필 |

---

## 현재 Live API 기능 활용 검증 결과

### 이미 활성화된 기능 (변경 불필요)

| 기능 | 코드 위치 | 설정 |
|------|----------|------|
| VAD | `session.go:158-169` | Low/Low, 300ms/500ms |
| Barge-in | `session.go:167` | StartOfActivityInterrupts |
| ProactiveAudio | `session.go:148-153` | 클라이언트 default true |
| OutputAudioTranscription | `session.go:155` | 항상 활성 |
| InputAudioTranscription | `session.go:156` | 항상 활성 |
| ContextWindowCompression | `session.go:171-178` | trigger 4096, target 2048 |
| SessionResumption | `session.go:58-64` | 핸들 기반 재연결 |
| 6개 캐릭터 페르소나 | `Assets/Sprites/{name}/preset.json` | 각각 고유 음성+성격 |

### 비활성 상태 (활성화 필요)

| 기능 | 상태 | 원인 | 필요한 변경 |
|------|------|------|-----------|
| **AffectiveDialog** | 백엔드 코드 존재, 클라이언트 미전송 | `GatewayClient.swift:288-293`에 `affectiveDialog` 키 누락 | setup 메시지에 1줄 추가 |

### ADK 공식 SDK 활용 현황

| 패키지 | 사용 여부 | 설명 |
|--------|---------|------|
| `google.golang.org/adk v0.5.0` | ✅ go.mod에 직접 의존 | |
| `adk/agent` — `agent.New()`, `agent.Config` | ✅ 8개 에이전트 모두 사용 | |
| `adk/agent/workflowagents/sequentialagent` | ✅ 그래프 오케스트레이션 | |
| `adk/session` — State, Event | ✅ 에이전트 간 상태 공유 | |
| `adk/model` — LLMResponse | ✅ 응답 구조 | |
| `adk/agent/workflowagents/parallelagent` | ❌ 미사용 | 독립 에이전트 병렬 실행 가능 |
| `adk/tool/functiontool` | ❌ 미사용 | 도구 정의 패턴 |
| `adk/tool/geminitool` — GoogleSearch | ❌ 미사용 | 네이티브 구글 검색 |
| `adk/memory` — MemoryService | ❌ 미사용 | 내장 메모리 서비스 |
| `adk/runner` | ❌ 미사용 | 에이전트 실행 런타임 |

---

## 4개 모드 최종 설계

### Mode 1: VAD 음성 대화 + 음성 답변

**현재 상태**: 대부분 완성. 핵심 파이프라인 작동 중.

**레이턴시**: ~1.5-3s (음성 → 음성 응답)

**즉시 적용 가능한 개선 (P0)**:

1. **AffectiveDialog 활성화** — `GatewayClient.swift` setup 메시지에 추가
   - Gemini가 사용자 음성 톤에서 좌절/흥분/불확실함 자동 감지
   - MoodDetector의 텍스트 기반 감정 분석을 보완
   - 구현: 1줄 추가 (`"affectiveDialog": true`)

2. **OutputTranscription → 말풍선 연동 강화**
   - 현재: Live API 음성 응답의 텍스트가 `transcription` 메시지로 클라이언트에 전달됨
   - 현재: chatPanel에만 표시, CatPanel 말풍선에는 미연동
   - 개선: `transcription(finished: true)` 시 CatPanel 말풍선에도 표시
   - 효과: 음성 대화 중에도 말풍선이 자연스럽게 동기화

**해카톤 점수 영향**: Innovation 40% — "natural, immersive" + "distinct persona"

---

### Mode 2: 스크린 제안 — 상황별 모달리티

**현재 상태**: 모든 스크린 제안에 TTS 음성 동반 (8.8s 레이턴시)

**최선의 설계 — 모든 스크린 제안이 말풍선만인 것이 아님**:

Mediator가 urgency/mood 기반으로 모달리티를 결정:

| 조건 | 모달리티 | 이유 |
|------|---------|------|
| significance < 7 | **말풍선만** (4.5s) | 개발 흐름 보존 |
| errorDetected = true | **음성 + 말풍선** (8.8s) | 에러는 즉각 알림 필요 |
| celebration = true | **음성 + 애니메이션** (8.8s) | 축하는 에너지 필요 |
| mood = frustrated | **음성 + 말풍선** (8.8s) | 감정적 지지는 음성이 효과적 |
| mood = idle (10분+) | **음성 + 말풍선** | 주의 환기 필요 |
| 그 외 | **말풍선만** (4.5s) | 기본은 비방해적 |

**구현**:
- `handler.go`: Mediator 결과에 `urgency` 필드 활용하여 TTS 호출 조건부 실행
- 현재 urgency 필드 이미 존재 (`result.Decision.Urgency`)
- `urgency == "high"` → TTS + 말풍선
- `urgency == "low" || urgency == ""` → 말풍선만

**말풍선 전용 시 표시 시간**:
- 현재: `bubbleDuration = 2.0s` (TTS 동반 시 turnActive로 유지)
- 말풍선 전용: 텍스트 길이 기반 동적 계산 `max(3.0, Double(text.count) / 5.0)` (최대 10초)

**레이턴시 개선**: 대부분의 스크린 제안이 significance < 7 → **8.8s → 4.5s (49% 감소)**

---

### Mode 3: Apple Watch형 프로액티브 인게이지먼트

**현재 상태**: 기본 인프라 존재 (EngagementAgent 3분 침묵, MoodDetector 4상태, CelebrationTrigger)

**최선의 설계**:

#### 인게이지먼트 계층 (4-Tier)

| Tier | 주기 | 모달리티 | 트리거 | 현재 구현 |
|------|------|---------|--------|----------|
| **1. 앰비언트** | 상시 | 고양이 애니메이션만 | 상태 변화 | ⚠️ 스프라이트 감정만 |
| **2. 마이크로 넛지** | 시간당 최대 3회, 최소 간격 5분 | 짧은 음성 (3-5초) | 좌절 감지, 막힘 상태 | ⚠️ MoodDetector 있으나 음성 연동 미약 |
| **3. 휴식 알림** | 50분마다 | 음성 + 말풍선 (10초) | 활동 시간 경과 | ❌ 미구현 |
| **4. 축하** | 이벤트 기반 | 음성 + 애니메이션 (5초) | 성공 감지 (significance ≥ 9) | ✅ CelebrationTrigger |

#### 쿨다운 규칙 (알림 피로 방지)

```
global_cooldown: 5분     # 모든 프로액티브 간 최소 간격
session_grace: 10분      # 앱 시작 후 유예
daily_maximum: 15회      # 일일 하드 캡
break_interval: 50분     # 휴식 알림 간격
focus_mode: 비활성화      # macOS 방해금지 모드 존중
```

#### 필요한 변경

**백엔드 (ADK Orchestrator)**:
1. `SessionMetrics` 확장: `ActiveMinutes int`, `LastBreakReminder time.Time`
2. `EngagementAgent` 확장: 시간 기반 휴식 트리거 (현재 침묵 3분만)
3. `MoodDetector` 확장: `MoodTired` 상태 추가 (ActiveMinutes > 120)
4. Mediator: 인게이지먼트 종류별 urgency 결정 (Tier 2 = low, Tier 3 = high)

**프론트엔드 (Swift)**:
1. 활동 시간 추적 (ScreenAnalyzer에 타이머)
2. macOS Focus Mode 연동 (`NSWorkspace.shared.frontmostApplication`)

**해카톤 점수 영향**: Innovation 40% — **최대 차별화 포인트**. 경쟁 프로젝트 대부분은 "말하면 답하는" 수준. Apple Watch형 프로액티브는 "동료" 느낌을 줌.

---

### Mode 4: 컨텍스트 메모리

**현재 상태**: Firestore 기반 MemoryAgent (ADK Orchestrator). 50-100ms 레이턴시.

**최선의 설계**: 로컬 SQLite가 주 저장소, Firestore는 관찰성/분석 전용

#### 아키텍처

```
Swift Client (macOS)
├── Local SQLite (GRDB.swift) ← 메모리 source of truth
│   - 세션 요약: sessions 테이블
│   - 활동 패턴: activity_patterns 테이블
│   - 기억 청크: memory_chunks 테이블
│   - 조회 레이턴시: < 1ms
│
├── 세션 시작 시:
│   SQLite에서 최근 맥락 조회 → Gateway setup 메시지에 포함
│   → Gateway가 Gemini system instruction에 주입
│
├── 세션 종료 시:
│   Gateway → ADK MemoryAgent가 AI 요약 생성 (Gemini 사용)
│   → 요약 결과를 클라이언트에 반환
│   → 클라이언트가 SQLite에 저장
│
└── Firestore 유지 (해카톤 채점용):
    - SessionMetrics (발화 수, 응답 수, 축하 수)
    - 에이전트 실행 로그
    - "Google Cloud Native: Firestore" 체크박스 충족
```

#### 기술 선택: GRDB.swift

- SPM 호환: swift-tools-version 6.1+ ✅
- Swift 6 strict concurrency ✅
- DatabaseMigrator (스키마 마이그레이션) ✅
- 성능: 인덱스 조회 < 1ms, 200K rows fetch 0.06s
- WAL 모드: 읽기/쓰기 병행

#### 스키마

```sql
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    character_id TEXT NOT NULL,
    started_at DATETIME NOT NULL,
    ended_at DATETIME,
    context_summary TEXT,
    unresolved_issues TEXT,
    mood_summary TEXT
);

CREATE TABLE activity_patterns (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT REFERENCES sessions(id) ON DELETE CASCADE,
    activity_type TEXT NOT NULL,
    app_name TEXT,
    duration_seconds INTEGER,
    timestamp DATETIME NOT NULL
);

CREATE TABLE memory_chunks (
    id TEXT PRIMARY KEY,
    content TEXT NOT NULL,
    category TEXT CHECK (category IN ('fact', 'preference', 'issue', 'decision')),
    importance REAL DEFAULT 0.5,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_accessed DATETIME
);

CREATE INDEX idx_sessions_time ON sessions(started_at DESC);
CREATE INDEX idx_activity_app ON activity_patterns(app_name, timestamp);
CREATE INDEX idx_memory_importance ON memory_chunks(importance DESC);
```

**해카톤 점수 영향**: Technical 30% — "system design" + "robustness"

---

## 해카톤 100점 실행 계획

### P0: 즉시 적용 (1일) — 최소 노력, 최대 점수

| 작업 | 변경량 | 점수 영향 |
|------|-------|----------|
| AffectiveDialog 활성화 (클라이언트 setup에 1줄 추가) | 1줄 | Innovation ★★★★★ |
| 스크린 제안 urgency 기반 모달리티 분기 (handler.go) | ~10줄 | UX ★★★★☆ |
| OutputTranscription → CatPanel 말풍선 연동 | ~15줄 | UX ★★★☆☆ |
| 말풍선 전용 시 동적 표시 시간 (CatPanel.swift) | ~5줄 | UX ★★★☆☆ |

### P1: 핵심 차별화 (3일) — 해카톤 승리 핵심

| 작업 | 변경량 | 점수 영향 |
|------|-------|----------|
| Apple Watch 인게이지먼트: 휴식 알림 + 시간 추적 | 백엔드 ~100줄, 클라이언트 ~50줄 | Innovation ★★★★★ |
| 로컬 SQLite 메모리 (GRDB.swift) | Package.swift + ~200줄 | Technical ★★★★☆ |
| ADK ParallelAgent: Vision/Memory 병렬화 | graph.go ~30줄 | Technical ★★★☆☆ |

### P2: 마무리 및 제출 (4일)

| 작업 | 점수 영향 |
|------|----------|
| GCP Cloud Run 배포 + 배포 자동화 스크립트 | Demo 30% + 보너스 +0.2 |
| 아키텍처 다이어그램 작성 | Demo 30% |
| 데모 영상 촬영 (4분 이내, 영어 자막) | Demo 30% |
| dev.to 블로그 게시 (#GeminiLiveAgentChallenge) | 보너스 +0.6 |
| GDG 멤버십 가입 | 보너스 +0.2 |

### P3: 버퍼 (3일)

| 작업 |
|------|
| 통합 테스트 + 버그 수정 |
| 실기기 UX 테스트 |
| 영상 재촬영 (필요 시) |

---

## 사용자 경험 플로우 (완성 후)

```
[사용자가 코딩 시작]
  └→ 고양이 앰비언트 애니메이션 (숨쉬기, 귀 움직임)
  └→ SQLite에서 어제 맥락 로드 (<1ms)
  └→ 음성: "어제 authentication 모듈 작업하고 있었지? 오늘도 화이팅!"

[10분 후, 스크린 변화 감지 — significance 5]
  └→ 말풍선만: "오 새 함수 만들고 있구나!" (4.5s, 비방해적)

[에러 반복 감지 — errorDetected, MoodDetector: frustrated]
  └→ AffectiveDialog가 음성 톤에서 좌절 감지
  └→ 음성+말풍선: "괜찮아, 같이 보자. 에러 메시지 보니까 타입 불일치인 것 같아" (8.8s)

[사용자: "야 이 에러 뭐야?" — VAD 감지]
  └→ Live API 즉시 음성 응답 (~2s)
  └→ OutputTranscription → 말풍선 자동 표시
  └→ 응답 중 사용자가 말함 → Barge-in 즉시 취소

[50분 경과 — 활동 시간 추적]
  └→ 음성+말풍선: "벌써 50분! 잠깐 스트레칭 어때?" (Tier 3 휴식)

[테스트 통과 — CelebrationTrigger]
  └→ 음성+흥분 애니메이션: "테스트 통과! 대박!" (Tier 4 축하)

[다음날 세션]
  └→ SQLite 맥락 로드: "어제 CORS 이슈 미해결"
  └→ 음성: "어제 CORS 이슈 있었는데 해결됐어?"
```

---

## 참고 자료

### Live API 공식 문서
- Gemini Live API Overview: https://ai.google.dev/gemini-api/docs/live
- Live API Guide: https://ai.google.dev/gemini-api/docs/live-guide
- Go SDK: https://pkg.go.dev/google.golang.org/genai
- Go SDK GitHub: https://github.com/googleapis/go-genai

### ADK 공식 문서
- ADK Docs: https://google.github.io/adk-docs/
- Go Quickstart: https://google.github.io/adk-docs/get-started/go/
- Workflow Agents: https://google.github.io/adk-docs/agents/workflow-agents/
- Parallel Agents: https://google.github.io/adk-docs/agents/workflow-agents/parallel-agents/
- Memory: https://google.github.io/adk-docs/sessions/memory/
- Streaming/Live: https://google.github.io/adk-docs/streaming/

### 해카톤
- 공식 페이지: https://geminiliveagentchallenge.devpost.com/
- 규칙: https://geminiliveagentchallenge.devpost.com/rules
- GCP 크레딧 신청 (3/13 마감): https://forms.gle/rKNPXA1o6XADvQGb7
- GDG 가입: https://gdg.community.dev/

### UX 리서치
- Calm Technology Principles: https://principles.design/examples/principles-of-calm-technology
- Time Out (macOS): https://apps.apple.com/us/app/time-out-break-reminders/id402592703
- 알림 피로 연구: 시간당 최대 3회, 일 15회, 세션 시작 10분 유예
