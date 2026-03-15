# VibeCat 데모 리허설 프로토콜

**목표:** 3회 연속 성공해야 녹화 시작
**소요:** 리허설 1회당 ~5분

---

## 사전 확인 (자동 검증 완료됨)
- [x] VibeCat 프로덕션 연결 (connections=1)
- [x] 영어 응답 확인 (vibecat.language=en)
- [x] 프로액티브 발화 확인 (5회 연속 먼저 말함)
- [x] 화면 인식 확인 (코드, 에러, 앱 상태 인식)

## 오디오 설정
```bash
SwitchAudioSource -t output -s "MacBook Air 스피커"
SwitchAudioSource -t input -s "MacBook Air 마이크"
osascript -e "set volume output volume 70"
```

## 리허설 순서

### STEP 1: VibeCat 시작 + 첫 제안 (Chrome 시나리오)
1. Antigravity에서 `demo/UserService.swift` 열기
2. **기다리기** — VibeCat이 먼저 말할 때까지 (보통 10~30초)
3. VibeCat이 음악/코드/기타 제안을 하면:
   - 영어로 **"Yeah, go ahead"** 또는 **"Sure, play it"** 말하기
4. **확인:** VibeCat이 FC 도구를 실행하는가?
   - 오버레이 패널에 [AX], [CDP], [Hotkey] 등 뱃지가 보이는가?
   - 실제 앱 전환/액션이 발생하는가?

### STEP 2: 코드 수정 시나리오 (Antigravity)
1. Antigravity로 돌아가기 (`Cmd+Tab`)
2. `demo/UserService.swift`가 보이는 상태로 기다리기
3. VibeCat이 코드 문제를 지적하면:
   - **"Yeah, fix it"** 말하기
4. **확인:** VibeCat이 코드를 수정하려고 시도하는가?

### STEP 3: 터미널 시나리오
1. Terminal 열고 `ls` 입력 후 Enter
2. 기다리기 — VibeCat이 `ls -la` 제안하는지
3. VibeCat이 제안하면:
   - **"Yeah, run it"** 또는 **"Do it"** 말하기
4. **확인:** VibeCat이 Terminal에 `ls -la` 입력하는가?

---

## 합격 기준

각 리허설에서 아래 3가지 중 **최소 2가지** 성공:

| # | 시나리오 | 성공 조건 |
|---|----------|----------|
| 1 | Chrome/음악 | VibeCat이 먼저 제안 → 승인 → YouTube Music 열림 |
| 2 | 코드 수정 | VibeCat이 코드 문제 지적 → 승인 → IDE에서 액션 |
| 3 | 터미널 | VibeCat이 더 나은 명령 제안 → 승인 → 터미널에서 실행 |

**VibeCat이 제안하지 않는 경우:**
- 30초 더 기다리기
- 그래도 안 되면 직접 영어로 요청: "Hey VibeCat, can you play some music?"
- 프로액티브 제안이 아니어도 FC 실행이 작동하면 OK (데모에서는 프로액티브 + 직접 요청 혼합 가능)

---

## 결과 기록

### 리허설 1
- [ ] Chrome 시나리오: ___
- [ ] 코드 시나리오: ___
- [ ] 터미널 시나리오: ___

### 리허설 2
- [ ] Chrome 시나리오: ___
- [ ] 코드 시나리오: ___
- [ ] 터미널 시나리오: ___

### 리허설 3
- [ ] Chrome 시나리오: ___
- [ ] 코드 시나리오: ___
- [ ] 터미널 시나리오: ___

---

## 로그 확인 명령어 (리허설 중 터미널에서 실행)
```bash
# VibeCat이 뭐라고 했는지
strings /tmp/vibecat-rehearsal.log | grep "transcription FINALIZED" | tail -5

# FC 도구 호출 확인
strings /tmp/vibecat-rehearsal.log | grep -i "navigate_\|stepPlanned\|pendingFC" | tail -10

# 프로액티브 흐름 확인
strings /tmp/vibecat-rehearsal.log | grep "flow=proactive" | tail -5
```

---

## 리허설 후

3회 통과 시:
1. 실제로 작동한 시나리오 기반으로 `SCRIPT.md` 수정
2. `Cmd+Shift+5`로 녹화 시작
3. 녹화 후 `./scripts/demo-post-process.sh` 실행
