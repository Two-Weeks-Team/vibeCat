package models

import "time"

type VisionAnalysis struct {
	Significance    int     `json:"significance"`
	Content         string  `json:"content"`
	Emotion         string  `json:"emotion"`
	ShouldSpeak     bool    `json:"shouldSpeak"`
	ErrorDetected   bool    `json:"errorDetected"`
	RepeatedError   bool    `json:"repeatedError"`
	SuccessDetected bool    `json:"successDetected"`
	RepeatedSuccess bool    `json:"repeatedSuccess,omitempty"`
	ErrorMessage    string  `json:"errorMessage,omitempty"`
	ErrorRegion     *Region `json:"errorRegion,omitempty"`
}

type Region struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type MediatorDecision struct {
	ShouldSpeak bool   `json:"shouldSpeak"`
	Reason      string `json:"reason"`
	Urgency     string `json:"urgency"`
}

type MoodState struct {
	Mood            string    `json:"mood"`
	Confidence      float64   `json:"confidence"`
	Signals         []string  `json:"signals"`
	SuggestedAction string    `json:"suggestedAction"`
	UpdatedAt       time.Time `json:"updatedAt"`
	VoiceTone       string    `json:"voiceTone,omitempty"`
	VoiceConfidence float64   `json:"voiceConfidence,omitempty"`
}

type CelebrationEvent struct {
	TriggerType string `json:"triggerType"`
	Emotion     string `json:"emotion"`
	Message     string `json:"message"`
}

type SearchResult struct {
	Query   string   `json:"query"`
	Summary string   `json:"summary"`
	Sources []string `json:"sources,omitempty"`
}

type SearchRequest struct {
	Query     string `json:"query"`
	Language  string `json:"language,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	UserID    string `json:"userId,omitempty"`
}

type AnalysisRequest struct {
	Image     string `json:"image"`
	Context   string `json:"context"`
	Language  string `json:"language,omitempty"`
	AppName   string `json:"appName,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	UserID    string `json:"userId,omitempty"`
	Character string `json:"character,omitempty"`
	Soul      string `json:"soul,omitempty"`
}

type AnalysisResult struct {
	Vision      *VisionAnalysis   `json:"vision,omitempty"`
	Decision    *MediatorDecision `json:"decision,omitempty"`
	Mood        *MoodState        `json:"mood,omitempty"`
	Celebration *CelebrationEvent `json:"celebration,omitempty"`
	Search      *SearchResult     `json:"search,omitempty"`
	SpeechText  string            `json:"speechText,omitempty"`
}

const (
	MoodFocused    = "focused"
	MoodFrustrated = "frustrated"
	MoodStuck      = "stuck"
	MoodIdle       = "idle"
)

var SupportiveMessages = map[string]map[string][]string{
	MoodFrustrated: {
		"Korean": {
			"이거 꽤 까다로운 에러네요. 같이 디버깅해 볼까요?",
			"잠깐 쉬었다 보면 해결책이 보일 수도 있어요.",
			"이 에러 까다롭긴 한데, 하나씩 짚어보면 풀릴 거예요.",
			"에러 메시지를 자세히 보면 힌트가 있을 수 있어요.",
			"스택 트레이스 한번 같이 볼까요?",
			"혹시 최근에 바꾼 코드가 있으면 거기부터 확인해 보세요.",
			"이런 에러는 보통 원인이 좁혀져 있어요. 하나씩 봅시다.",
			"물 한 잔 마시고 다시 보면 의외로 답이 보일 때가 있어요.",
			"커밋 히스토리에서 뭐가 바뀌었는지 확인해 볼까요?",
			"이 부분 로그를 조금 더 찍어보면 원인을 좁힐 수 있을 거예요.",
		},
		"English": {
			"This looks frustrating. I can help debug it together.",
			"A short break might help. Fresh eyes often reveal the fix.",
			"This error is tricky, but we can work through it step by step.",
			"The error message might have a clue. Let's read it carefully.",
			"Want to check the stack trace together?",
			"If you changed something recently, that's a good place to start.",
			"These errors usually narrow down quickly once you isolate the cause.",
			"A glass of water and a second look can work wonders.",
			"Let's check the commit history to see what changed.",
			"Adding a few more log statements might help narrow this down.",
		},
	},
	MoodStuck: {
		"Korean": {
			"좀 막힌 것 같아 보여요. 검색해서 해결법 찾아볼까요?",
			"다른 접근 방식을 시도해 보면 풀릴 수도 있어요.",
			"이 문제 관련 공식 문서를 한번 확인해 볼까요?",
			"잠시 다른 작업 하다 오면 실마리가 잡힐 수도 있어요.",
			"문제를 더 작은 단위로 쪼개서 하나씩 확인해 보는 건 어때요?",
			"테스트 코드를 작성해서 동작을 확인해 보면 도움이 될 거예요.",
			"혹시 관련 이슈가 GitHub에 올라와 있는지 찾아볼까요?",
			"처음부터 다시 짚어보면 놓친 부분이 보일 수도 있어요.",
			"이런 상황에서는 러버덕 디버깅이 의외로 효과 있어요.",
			"한 발 물러서서 전체 흐름을 다시 그려보는 건 어때요?",
		},
		"English": {
			"It looks like you might be stuck. I can search for proven fixes.",
			"A different approach might help unblock this.",
			"Want me to check the official docs for this?",
			"Sometimes stepping away briefly helps the answer click.",
			"Breaking the problem into smaller pieces might make it clearer.",
			"Writing a quick test case could help verify what's going on.",
			"I can look for related GitHub issues if you'd like.",
			"Retracing from the beginning might reveal something you missed.",
			"Rubber duck debugging actually works surprisingly well here.",
			"How about stepping back and re-mapping the overall flow?",
		},
	},
}

var CelebrationMessages = map[string][]string{
	"Korean": {
		"오! 해냈네요!",
		"깔끔하게 잘 해결했어요!",
		"드디어! 이게 바로 우리가 원하던 거예요!",
	},
	"English": {
		"Yes! You nailed it!",
		"Great work. That was clean and solid.",
		"Finally! This is exactly what we were aiming for.",
	},
}
