# Devpost 최종 제출 체크리스트

**데드라인:** March 16, 2026 5:00 PM PDT (= 3월 17일 09:00 KST)
**제출 URL:** https://geminiliveagentchallenge.devpost.com/

---

## 필수 제출 항목 (Stage One 통과 조건)

### 1. 카테고리 선택
- [x] **UI Navigator** 선택

### 2. 텍스트 설명
- [x] 준비 완료: `docs/DEVPOST_SUBMISSION.md`
- [ ] Devpost에 복사 & 붙여넣기
- 포함 확인:
  - [x] Inspiration
  - [x] What It Does
  - [x] How We Built It
  - [x] Challenges We Ran Into
  - [x] Accomplishments
  - [x] What We Learned
  - [x] What's Next
  - [x] Built With (기술 스택)

### 3. 공개 코드 저장소
- [x] GitHub URL: `https://github.com/Two-Weeks-Team/vibeCat`
- [x] 레포지토리가 Public인지 확인
- [x] README.md에 스핀업 인스트럭션 포함
  - Swift 빌드 명령어
  - Go 빌드 명령어
  - Cloud Run 배포 명령어

### 4. GCP 배포 증명
- [ ] 별도 영상 녹화 (가이드: `docs/GCP_PROOF_GUIDE.md`)
- [ ] YouTube에 업로드 (Unlisted 가능)
- [ ] 또는 코드 파일 링크로 대체:
  - `infra/deploy.sh` — 배포 자동화 스크립트
  - `backend/realtime-gateway/Dockerfile` — Cloud Run 컨테이너
  - `backend/adk-orchestrator/Dockerfile` — Cloud Run 컨테이너

### 5. 아키텍처 다이어그램
- [x] 준비 완료: `docs/architecture.png` (122KB)
- [ ] Devpost에 이미지로 첨부

### 6. 데모 영상
- [ ] 4분 이내
- [ ] 영어 또는 영어 자막 포함
- [ ] 실제 소프트웨어 작동 증명 (목업 아님)
- [ ] 문제와 솔루션의 가치 설명 포함
- [ ] YouTube 또는 Vimeo에 Public 업로드
- [ ] Devpost에 링크 첨부

---

## 심사 기준별 체크 (Stage Two)

### Innovation & Multimodal UX (40%)
- [x] "Beyond Text" — 음성 우선 인터랙션 (텍스트 박스 아님)
- [x] Visual precision — AX + CDP + Vision 트리플 그라운딩
- [x] UI Navigator 특화: 화면 컨텍스트 이해 (맹목 클릭 아님)
- [x] 프로액티브 제안 (AI가 먼저 말함)
- [x] Live & context-aware — 실시간 화면 분석
- [ ] 데모에서 이 모든 것이 보이는지 확인

### Technical Implementation (30%)
- [x] GenAI SDK 사용 (v1.49.0)
- [x] ADK 사용 (v0.6.0)
- [x] Cloud Run 호스팅
- [x] Firestore 사용
- [x] 에러 핸들링 + 자가치유
- [x] 그라운딩 증거 (환각 방지)
- [ ] 데모에서 자가치유가 보이는지 확인

### Demo & Presentation (30%)
- [ ] 문제 정의가 명확한가?
- [ ] 솔루션의 가치가 설명되는가?
- [x] 아키텍처 다이어그램 포함
- [ ] Cloud 배포 시각적 증명 포함
- [ ] 실제 소프트웨어 작동 영상

---

## 보너스 포인트 (Stage Three, 최대 +1.0점)

### 블로그 포스트 (+0.6)
- [x] 15개 게시물 존재: https://dev.to/combba
- [ ] 각 포스트에 "Created for the Gemini Live Agent Challenge" 문구 확인
- [ ] 소셜 미디어 공유 시 #GeminiLiveAgentChallenge 해시태그

### 자동화된 클라우드 배포 (+0.2)
- [x] `infra/deploy.sh` 스크립트 존재
- [x] 공개 레포지토리에 포함됨

### GDG 멤버십 (+0.2)
- [ ] https://gdg.community.dev/ 가입
- [ ] 공개 프로필 링크 제출

---

## 제출 순서 (D-Day)

1. **데모 영상 녹화** → YouTube Public 업로드
2. **GCP 증명 영상** → YouTube Unlisted 업로드
3. **블로그 포스트** → 해커톤 참가 문구 확인
4. **Devpost 제출 폼** 작성:
   - 카테고리: UI Navigator
   - 텍스트: `docs/DEVPOST_SUBMISSION.md` 복사
   - 코드 URL: GitHub 레포
   - 데모 영상 URL: YouTube 링크
   - GCP 증명: 영상 URL 또는 코드 링크
   - 아키텍처: `docs/architecture.png` 업로드
   - Built With 태그 입력
5. **최종 검토** → 제출

---

## 주의사항

- 제출 후에도 Submission Period 끝나기 전까지는 수정 가능
- 영상은 반드시 Public (또는 Unlisted가 아닌 Public)
- 코드 레포는 반드시 Public
- 모든 자료는 영어 또는 영어 번역 포함
- 제3자 도구 사용 시 description에 명시 (chromedp 등)
