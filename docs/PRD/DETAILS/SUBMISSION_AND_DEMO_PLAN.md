# Submission and Demo Plan

## Submission Artifacts

- Text description (features, stack, learnings)
- Public repository URL
- README spin-up instructions
- Proof of Google Cloud deployment (screen recording of Cloud Run console + logs)
- Architecture diagram (9-agent graph with GCP services)
- Demo video (under 4 minutes)

## Demo Narrative

> "When you code alone, the chair next to you is empty. No one catches your typos. No one notices you are stuck. No one celebrates when your tests finally pass. VibeCat fills that chair."

## Demo Flow (4 minutes)

| Time | Scene | What Happens | Agents Shown | 6.0 Feature |
|---|---|---|---|---|
| 0:00–0:25 | **Hook** | "혼자 개발할 때, 옆자리가 비어있다" — problem framing | — | Story |
| 0:25–0:45 | **Session Start** | Open VibeCat → cat appears → "어제 인증 모듈 하다 멈췄지, 이어서 할래?" | MemoryAgent | Cross-session memory |
| 0:45–1:05 | **Natural Conversation** | Talk to VibeCat while coding, interrupt mid-sentence (barge-in) | VAD | affectiveDialog tone |
| 1:05–1:25 | **Screen Analysis + Pointing** | Error on screen → cat moves toward error → "23번째 줄 타입 미스매치" | VisionAgent | Screen pointing |
| 1:25–1:40 | **Decision HUD** | Toggle overlay → show why agent spoke (trigger, analysis, confidence) | Mediator | Grounding evidence |
| 1:40–1:55 | **Search Help** | "이 에러 뭔지 모르겠어" → "찾아봤는데, Stack Overflow에서..." | SearchBuddy | proactiveAudio |
| 1:55–2:10 | **Mood Detection** | Debugging continues → worried voice tone: "힘들어 보이는데, 같이 볼까?" | MoodDetector | affectiveDialog |
| 2:10–2:25 | **Celebration** | Fix → tests pass → bright voice: "통과! 고생했어" (happy sprite) | CelebrationTrigger | affectiveDialog |
| 2:25–2:35 | **Quiet Moment** | Coding in flow → cat stays quiet (privacy indicator visible) | Mediator | Privacy controls |
| 2:35–2:50 | **Resilience** | Network disconnect → reconnecting indicator → reconnects → "돌아왔어!" | Fallback | Graceful recovery |
| 2:50–3:15 | **Architecture** | 9-agent diagram: "사람 한 명이 하는 일을 분해하면 9가지" | — | Agent philosophy |
| 3:15–3:35 | **Cloud Proof** | Cloud Run console + Cloud Trace span for live request + automated deploy script | — | Trace + IaC |
| 3:35–3:50 | **Farewell** | "오늘 고생했어, 내일 보자" → cat sleeping sprite | MemoryAgent | Emotional ending |
| 3:50–4:00 | **Closing** | "The chair is no longer empty." | — | — |

## Optional Bonus Contributions (Score Boosters)

- **Content Publishing** (up to +0.6 points): Publish a blog, podcast, or video covering how the project was built using Google AI models and Google Cloud on a public platform (e.g., medium.com, dev.to, YouTube). Content must be public (not unlisted) and include language stating it was created for this hackathon. Use hashtag `#GeminiLiveAgentChallenge` on social media.
- **Automated Cloud Deployment** (up to +0.2 points): Demonstrate automated deployment using scripts or infrastructure-as-code tools. Code must be in the public repository (see `infra/` in Required Artifacts).
- **GDG Membership** (up to +0.2 points): Provide a public GDG community profile link if an active Google Developer Group member.

## Pre-Submission Verification

- Required stack evidence linked in code
- Deployment evidence linked in docs
- Demo timeline and script validated
- All source references mapped in `SOURCE_REFERENCE_MAP.md`
- Optional bonus contributions prepared if applicable
