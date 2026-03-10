# Live Architecture Best Practices

## 목적

VibeCat의 핵심 목적은 Gemini Live Agent Challenge 심사 기준에 맞게, 화면 인식과 음성 상호작용이 하나의 자연스러운 실시간 에이전트로 동작하는 것이다. 이 문서는 공식 Gemini Live API 베스트 프랙티스에 맞춰 현재 구조를 정리하고, 실제 코드에 반영한 방향을 고정한다.

## 최종 구조

1. Live API는 유일한 음성 턴 엔진이다.
2. Realtime Gateway는 턴 상태와 세션 정책의 authoritative 계층이다.
3. ADK Orchestrator는 화면 분석, 장기 메모리, proactive 제안을 담당한다.
4. MemoryAgent는 장기 기억을 요약하고, Gateway는 이를 Live 세션의 최근 필수 컨텍스트로 압축 주입한다.

## 공식 베스트 프랙티스 반영

### 1. Live 세션은 단일 도구/음성 세션으로 유지

- 음성 검색은 Live 세션 내부의 네이티브 `GoogleSearch` tool을 사용한다.
- 사용자가 현재 정보, 최신 정보, 웹 문서, GitHub, 검색 요청을 하면 Live가 같은 턴 안에서 검색하고 답한다.
- "검색해볼게요"만 말하고 끝나는 비동기 우회 경로를 제거하는 것이 목적이다.

### 2. 최근 컨텍스트는 짧게 압축해서 system instruction에 포함

- 장기 기억 전체를 넣지 않는다.
- 최근 1~2개 세션 요약, 미해결 이슈, 활성 토픽만 넣는다.
- 현재 화면과 최신 사용자 발화가 충돌하면 최신 입력을 우선한다.

### 3. 세션 재개를 실제로 사용

- 클라이언트는 `sessionHandle`을 저장하되, setup 요청에는 `resumptionHandle`로 다시 보낸다.
- Live 재연결 시 기존 턴 맥락을 최대한 유지한다.

### 4. 검색/grounding 여부는 로그에서 직접 검증 가능해야 함

- Gateway는 `grounding_metadata` trace를 남긴다.
- 검색 쿼리 수, 출처 수, retrieval score를 로그에서 확인할 수 있어야 한다.

## 구현 반영

### Gateway

- setup 시 `MemoryAgent`로부터 최근 컨텍스트를 가져와 Live config에 포함
- 음성 턴은 ADK 검색 우회 대신 Live 네이티브 검색을 사용
- grounding metadata, usage metadata, voice activity를 서버 로그로 수집

### Live Session

- `GoogleSearch` tool 활성화
- 검색이 필요할 때만 검색하고, 검색 후 같은 턴에서 답하도록 system instruction 강화
- 최근 필수 컨텍스트를 `RECENT ESSENTIAL CONTEXT` 블록으로 주입

### Client

- `resumptionHandle` 전송 오탈자 수정
- 서버 authoritative turn state 유지
- 기존 말풍선/발화 동기화 흐름 유지

## 남은 확장 후보

1. `URLContext`를 Live built-in tool로 추가
2. `CodeExecution`을 음성 디버깅 턴에 조건부 추가
3. grounding metadata를 Recent Speech나 디버그 UI에도 표면화
4. 메모리 컨텍스트에 코드베이스 관련 고정 관심사와 최근 실패 패턴을 별도 슬롯으로 분리
