# Flow Audit 2026-03-10

> Historical snapshot note: this audit references the 2026-03-10 runtime and older Cloud Run revisions. Treat it as a dated analysis artifact, not the current operational source of truth.

## 기준

- 공식 문서
  - Gemini Live API Best Practices
  - Gemini Live API Session Management
  - Gemini Live API Tools
  - Gemini Google Search Tool
- 실제 기준 코드
  - `realtime-gateway-00038-ljq`
  - `adk-orchestrator-00036-j9f`

## 1. 세션 시작 / 재개

### 현재 흐름

1. 클라이언트가 `setup` 전송
2. Gateway가 `deviceId`, `language`, `searchEnabled`, `soul`을 수신
3. `resumptionHandle`이 없으면 MemoryAgent 컨텍스트를 조회
4. Live 세션을 연결하고 `setupComplete` 반환
5. 이후 `sessionResumptionUpdate`로 새 핸들을 저장

### 베스트 프랙티스 적합성

- 적합
  - 세션 재개를 실제로 사용
  - 최근 필수 컨텍스트만 압축 주입
  - 장기 기억은 Firestore/MemoryAgent로 분리
- 보완
  - cold start 첫 연결은 여전히 메모리 조회가 들어갈 수 있음
  - 다만 Gateway 메모리 캐시로 반복 연결 비용은 줄임

### 개선 적용

- `resumptionHandle` 필드 정정
- 메모리 컨텍스트 캐시 추가
- 세션 압축 기준을 `12000 -> 6000` 슬라이딩 윈도우로 조정

## 2. 일반 음성 대화

### 현재 흐름

1. 사용자가 마이크로 발화
2. Live VAD가 사용자 턴 종료 판단
3. Gateway가 `inputTranscription`과 `turnState`를 authoritative 하게 전달
4. Live가 오디오와 output transcription을 같은 턴에서 생성
5. 클라이언트는 음성/말풍선을 같은 transcription으로 렌더링

### 베스트 프랙티스 적합성

- 적합
  - 단일 Live 세션이 발화 엔진 역할
  - 출력 전사와 오디오가 같은 턴에서 동기화
  - 클라이언트는 추론보다 렌더링 중심
- 보완
  - 장시간 세션에서 prompt token 증가를 더 관찰할 필요가 있음

## 3. 음성 검색

### 현재 흐름

1. 사용자가 최신/검색성 질문을 음성으로 요청
2. Live 세션의 네이티브 `GoogleSearch` tool이 같은 턴 안에서 grounding
3. Gateway는 `grounding_metadata`와 `usage_metadata`를 로그로 남김
4. Live가 검색 결과를 포함한 음성/말풍선을 바로 반환

### 베스트 프랙티스 적합성

- 적합
  - Live tool을 세션 config에 직접 부착
  - 검색 후 별도 질문 없이 같은 턴에서 응답
  - grounding metadata를 관측 가능하게 수집
- 보완
  - 검색 질의는 외부 검색 지연 편차가 큼
  - 텍스트 `clientContent`는 아직 ADK tool routing 경로를 일부 유지

### 벤치마크

- 개선 전
  - `turn_started`: 평균 415ms
  - `grounding_metadata`: 평균 5567ms
  - `turn_complete`: 평균 11624ms
- 개선 후
  - `turn_started`: 평균 371ms
  - `grounding_metadata`: 평균 6797ms
  - `turn_complete`: 평균 10514ms

해석:

- 검색 자체는 외부 편차가 커서 grounding 완료 시점은 흔들린다.
- 대신 응답 길이를 줄여 전체 턴 종료는 단축됐다.

## 4. 화면 인식 / proactive 제안

### 현재 흐름

1. 클라이언트가 기본적으로 `window under cursor`를 캡처
2. Fast Path는 Live video context로 전송
3. Smart Path는 ADK `/analyze`로 전송
4. ADK가 `shouldSpeak / urgency / speechText`를 결정
5. Gateway가 Live 세션에 proactive prompt를 주입

### 베스트 프랙티스 적합성

- 적합
  - Vision/Memory/Decision 계층 분리
  - proactive는 별도 TTS가 아니라 Live 턴으로 통합
- 보완
  - ADK analyze는 현재도 수 초 단위 병목이 될 수 있음
  - 중요 이벤트만 보내도록 significance gating을 더 강화할 수 있음

## 5. 바지인

### 현재 흐름

1. 로컬 RMS가 빠른 candidate를 감지
2. 명시적 `bargeIn` 메시지가 Gateway로 전달
3. Gateway는 현재 모델 턴을 interrupt 처리
4. 모델 턴 중 일반 오디오는 자동 interrupt 하지 않고 버림
5. 사용자 후속 발화는 다시 Live로 이어짐

### 베스트 프랙티스 적합성

- 적합
  - 인터럽트 권한을 명시적 사용자 바지인으로 제한
  - stale audio가 모델 턴을 끊는 문제 제거
- 보완
  - 입력 장치별 RMS 적응형 튜닝은 여전히 후보

## 6. 메모리

### 현재 흐름

1. 세션 종료 시 Gateway가 history를 요약 저장
2. MemoryAgent가 최근 세션 요약, 미해결 이슈, 활성 토픽을 유지
3. 새 세션 시작 시 최근 필수 정보만 압축해 Live system instruction에 주입

### 베스트 프랙티스 적합성

- 적합
  - 장기 기억을 별도 저장소/에이전트로 분리
  - Live에는 압축된 최근 필수 정보만 넣음
- 보완
  - 첫 연결에서 메모리 miss 시 cold start를 더 줄이려면 background prefetch가 추가 후보

## 레이스 / 동시성

- `go test -race ./...`
  - `backend/realtime-gateway` 통과
  - `backend/adk-orchestrator` 통과
- 현재 구조상 가장 중요했던 race 성격 문제는 `모델 턴 시작 직후 남아 있던 일반 오디오가 자동 바지인으로 처리되던 경로`였고, 이 부분은 제거됐다.

## 다음 최적화 후보

1. 텍스트 `clientContent` 검색 경로도 Live 네이티브 검색 중심으로 재정렬
2. ADK analyze significance gating을 강화해 proactive 오탐과 지연을 줄이기
3. Memory context miss 시 background prefetch + non-speaking injection 전략 검토
4. 입력 장치별 adaptive barge-in threshold 도입
