# VibeCat 데모 녹화 완전 가이드

**환경:** MacBook Pro 단일 화면 (3456×2234 Retina = 1728×1117 논리)
**목표:** 4분 이내, 영어 자막, 실제 작동 증명

---

## 1단계: BlackHole 설치 (터미널에서 직접 실행)

```bash
# 터미널을 열고 아래 명령어 실행
brew install blackhole-2ch
# sudo 비밀번호 입력 요청됨 → 입력
# 완료 후 재부팅 필요
```

재부팅 후:
1. **Audio MIDI Setup** 열기 (`/Applications/Utilities/Audio MIDI Setup.app`)
2. 좌측 하단 `+` → "Create Multi-Output Device" 클릭
3. 체크:
   - ☑ Built-in Output (MacBook Pro Speakers)
   - ☑ BlackHole 2ch
4. "Built-in Output"를 Master Device로 설정
5. **시스템 설정 → 사운드 → 출력** → "Multi-Output Device" 선택

---

## 2단계: 녹화 전 환경 정리

### 앱 준비
```bash
# 1. 프로덕션 서비스 확인
curl -s https://realtime-gateway-163070481841.asia-northeast3.run.app/health

# 2. VibeCat 빌드 + 실행
cd /Users/kimsejun/GitHub/vibeCat/VibeCat
swift build
.build/arm64-apple-macosx/debug/VibeCat &

# 3. 연결 확인 (10초 대기 후)
sleep 10
curl -s https://realtime-gateway-163070481841.asia-northeast3.run.app/health
# connections: 1 이어야 함

# 4. 필요한 앱 열기
open -a "Antigravity"
open -a "Google Chrome" "https://music.youtube.com"
open -a Terminal

# 5. 아키텍처 다이어그램 열기
open /Users/kimsejun/GitHub/vibeCat/docs/architecture.png
```

### 데스크탑 정리
- [ ] Dock 숨기기: `Cmd+Option+D`
- [ ] 방해금지 모드 ON: 제어센터 → 집중 모드 → 방해금지
- [ ] 모든 채팅 앱 종료 (Slack, Discord, KakaoTalk 등)
- [ ] 알림 배너 모두 닫기
- [ ] 메뉴바 정리 (불필요한 아이콘 최소화)

### 오디오 설정
- **BlackHole 설치됨:** 시스템 설정 → 사운드 → 출력 → Multi-Output Device
- **BlackHole 없음:** 기본 스피커 유지 (마이크로 간접 녹음)

### 창 배치 (미리 설정)
1. **Antigravity IDE** — Swift 파일 열기, `getUserData()` 함수가 보이도록
2. **Chrome** — YouTube Music 열기 (재생 안 함)
3. **Terminal** — 깨끗한 프롬프트, 홈 디렉토리
4. **Preview** — `docs/architecture.png` 열어두기
5. **GCP Console** — Chrome 탭에 Cloud Run 페이지 열어두기

---

## 3단계: 녹화 시작

### 스크린 녹화 설정
1. `Cmd+Shift+5` 누르기
2. **"전체 화면 기록"** 선택
3. **옵션** 클릭:
   - 마이크: **BlackHole 2ch** (또는 MacBook Pro 마이크)
   - 타이머: **5초** (준비 시간)
   - 저장 위치: 바탕화면
4. **"기록"** 클릭

### 녹화 중 시나리오 (docs/challenge/video-1-demo/SCRIPT.md 참조)

#### ACT 1: 훅 (0:00~0:25)
- 빠르게 앱 전환하며 컨텍스트 스위칭 보여주기
- 내레이션: "Context switching. The silent killer of deep work."

#### ACT 2: 코드 수정 (0:25~1:15) ⭐ 핵심
- Antigravity IDE에서 코드 작성
- VibeCat이 먼저 말함: "I noticed a null check missing..."
- 사용자: "Yeah, go ahead"
- 오버레이 패널에서 진행 상황 표시
- 코드 수정 완료

#### ACT 3: 음악 검색 (1:15~2:15) ⭐ 멀티앱
- "Can you find some focus music?"
- Chrome → YouTube → 검색 → 재생
- 4개 앱을 한 문장으로

#### ACT 4: 터미널 + 자가치유 (2:15~3:00) ⭐ 기술력
- VibeCat: "ls -la would show hidden files..."
- 첫 시도 실패 → 자가치유 → 성공
- 실패를 보여주는 게 신뢰를 쌓음

#### ACT 5: 아키텍처 + 마무리 (3:00~4:00)
- 아키텍처 다이어그램 보여주기
- GCP Console 5초 플래시
- VibeCat 인사: "I'll be here when you need me."

### 녹화 중 팁
- **말을 또박또박** 하세요 (마이크 자동 녹음)
- `Cmd+Tab`으로 앱 전환
- VibeCat 고양이 캐릭터가 항상 보이는지 확인
- 오버레이 패널이 보이는지 확인
- 너무 빠르게 진행하지 마세요 — 심사위원이 볼 수 있어야 함

### 녹화 종료
- `Cmd+Shift+5` → 중지 (또는 메뉴바 아이콘 클릭)
- 바탕화면에 .mov 파일로 저장됨

---

## 4단계: 후처리

```bash
cd /Users/kimsejun/GitHub/vibeCat

# 자동 후처리 스크립트 실행
./scripts/demo-post-process.sh ~/Desktop/"Screen Recording"*.mov

# 결과물:
# docs/video/vibecat-demo-clean.mp4      (자막 없음, 깔끔)
# docs/video/vibecat-demo-subtitled.mp4  (자막 번인)
# docs/video/vibecat-demo-youtube.mp4    (YouTube 최적화)
```

---

## 5단계: YouTube 업로드

### 업로드 정보
- **제목:** `VibeCat — Proactive Desktop Companion | Gemini Live Agent Challenge 2026`
- **공개 범위:** Public (공개)
- **자막:** docs/demo_subtitles.srt 파일 업로드 (CC 자막)

### 설명:
```
VibeCat is a proactive desktop companion that watches your screen,
suggests actions before you ask, and acts with your permission.
Built with Gemini Live API + ADK on Google Cloud Run.

Key Features:
- Proactive suggestions (AI speaks first)
- Voice-first interaction via Gemini Live API
- Triple-source grounding (Accessibility + CDP + Vision)
- Self-healing navigation with vision verification
- Native macOS Swift app with animated cat companion

GitHub: https://github.com/Two-Weeks-Team/vibeCat
Blog: https://dev.to/combba

Built for the Gemini Live Agent Challenge 2026.
#GeminiLiveAgentChallenge #GeminiAPI #GoogleCloud #AIAgent #UINavigator
```

### 태그:
`GeminiLiveAgentChallenge, Gemini, GoogleCloud, AIAgent, UINavigator, macOS, Swift, desktopcompanion`

---

## 중요 체크리스트

- [ ] 영상이 4분 이내인가?
- [ ] 영어 자막이 있는가? (SRT 또는 번인)
- [ ] 실제 소프트웨어가 작동하는 모습이 보이는가? (목업 아님)
- [ ] VibeCat이 먼저 말하는 장면이 있는가? (프로액티브)
- [ ] 자가치유 장면이 있는가? (기술력)
- [ ] 멀티앱 워크플로우가 보이는가? (혁신성)
- [ ] 아키텍처 다이어그램이 보이는가?
- [ ] GCP 콘솔 화면이 보이는가? (클라우드 증명)
- [ ] YouTube에 Public으로 업로드했는가?
