# GCP 배포 증명 녹화 가이드

**목적:** 심사위원에게 백엔드가 Google Cloud에서 실행 중임을 증명
**소요 시간:** 2~3분
**형식:** 별도 짧은 영상 (데모 영상과 분리)

---

## 녹화할 내용 (순서대로)

### 1. Cloud Run 서비스 목록 (30초)
1. Chrome에서 GCP Console 열기: https://console.cloud.google.com/run?project=vibecat-489105
2. 두 서비스가 녹색으로 표시되는 것을 보여주기:
   - `realtime-gateway` — ✅ active
   - `adk-orchestrator` — ✅ active
3. 각 서비스 클릭하여 세부 정보 보여주기:
   - 리전: `asia-northeast3`
   - URL 표시
   - 최근 배포 시간

### 2. Cloud Run 로그 (30초)
1. 서비스 중 하나 클릭 → "Logs" 탭
2. 최근 로그 엔트리 보여주기 (실시간 트래픽 증명)
3. 특히 `/health` 엔드포인트 호출 로그가 보이면 좋음

### 3. Firestore 데이터 (20초)
1. https://console.cloud.google.com/firestore?project=vibecat-489105
2. 컬렉션 목록 보여주기 (action_states, sessions 등)
3. 데이터가 존재함을 보여주기

### 4. Secret Manager (15초)
1. https://console.cloud.google.com/security/secret-manager?project=vibecat-489105
2. 시크릿 목록만 보여주기 (값은 노출하지 않음!)
3. GEMINI_API_KEY 등이 존재함을 증명

### 5. 터미널에서 Health Check (20초)
```bash
curl -s https://realtime-gateway-163070481841.asia-northeast3.run.app/health | python3 -m json.tool
```
결과: `{"connections": 0, "service": "realtime-gateway", "status": "ok"}`

### 6. 코드 레포지토리 배포 스크립트 (15초)
- `infra/deploy.sh` 파일을 에디터에서 열어 보여주기
- Cloud Run 배포 자동화 스크립트임을 증명

---

## 녹화 방법

```bash
# Cmd+Shift+5 → 전체 화면 기록
# 마이크: 없어도 됨 (음성 불필요, 화면만)
# 2~3분이면 충분
```

## 녹화 후

이 영상은 YouTube에 별도로 업로드하거나,
Devpost 제출 시 "Proof of Google Cloud Deployment" 필드에 링크로 첨부합니다.

또는 메인 데모 영상의 ACT 5에서 GCP Console 화면을 잠깐 보여주면
별도 영상이 필요 없을 수도 있습니다 (단, 5초로는 부족할 수 있음).

**권장:** 별도 짧은 영상으로 만들어 Unlisted로 YouTube 업로드.
