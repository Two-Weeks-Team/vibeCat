# Gemini Live Agent Challenge — 제출 녹화 가이드

**데드라인:** March 16, 2026 5:00 PM PDT (= 3/17 09:00 KST)

---

## 찍어야 하는 영상

| # | 영상 | 길이 | 필수 | 폴더 |
|---|------|------|------|------|
| 1 | **메인 데모** | 4분 이내 | ✅ 필수 | `video-1-demo/` |
| 2 | **GCP 배포 증명** | 2~3분 | 선택 (코드 링크로 대체 가능) | `video-2-gcp-proof/` |

---

## 녹화 순서

### 1. VibeCat 실행
```bash
cd /Users/kimsejun/GitHub/vibeCat/VibeCat
swift build && .build/arm64-apple-macosx/debug/VibeCat &
sleep 10
curl -s https://realtime-gateway-163070481841.asia-northeast3.run.app/health
```
→ `connections: 1` 확인

### 2. 영상 1 녹화 (메인 데모)
`video-1-demo/RECORDING_GUIDE.md` 따라 진행
- 스크립트: `video-1-demo/SCRIPT.md`
- 자막: `video-1-demo/subtitles.srt`

### 3. 영상 2 녹화 (GCP 증명) — 선택
`video-2-gcp-proof/GUIDE.md` 따라 진행

### 4. 후처리
```bash
./scripts/demo-post-process.sh ~/Desktop/"Screen Recording"*.mov
```

### 5. YouTube 업로드
- 영상 1 → **Public** + SRT 자막 첨부
- 영상 2 → **Unlisted** (선택)

### 6. Devpost 제출
- 제출 텍스트: `docs/DEVPOST_SUBMISSION.md`
- 아키텍처: `docs/architecture.png`
- GDG 프로필: `https://gdg.community.dev/u/m5n58q/`

### YouTube 메타데이터
- **제목:** `VibeCat — Proactive Desktop Companion | Gemini Live Agent Challenge 2026`
- **설명:** `video-1-demo/RECORDING_GUIDE.md` 5단계 참조
- **태그:** `GeminiLiveAgentChallenge, Gemini, GoogleCloud, AIAgent, UINavigator, macOS, Swift`
