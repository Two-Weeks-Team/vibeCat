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

type AnalysisRequest struct {
	Image     string `json:"image"`
	Context   string `json:"context"`
	AppName   string `json:"appName,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	UserID    string `json:"userId,omitempty"`
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

var SupportiveMessages = map[string][]string{
	MoodFrustrated: {
		"힘들어 보이는데, 같이 한번 볼까요?",
		"잠깐 쉬어가는 건 어때요? 새로운 눈으로 보면 보일 수도 있어요.",
		"이런 에러, 저도 처음엔 헷갈렸어요. 같이 디버깅해봐요.",
	},
	MoodStuck: {
		"막힌 것 같은데, 검색해볼까요?",
		"비슷한 문제를 겪은 사람들이 있을 거예요. 찾아볼게요.",
		"다른 접근 방법을 시도해볼까요?",
	},
}

var CelebrationMessages = []string{
	"오예! 해냈어요! 🎉",
	"완벽해요! 정말 잘했어요!",
	"드디어! 이 순간을 위해 달려왔잖아요!",
}
