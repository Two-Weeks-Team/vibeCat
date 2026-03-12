package store

import "time"

// Session represents a user's active session.
// Maps to Firestore document at sessions/{sessionId}
type Session struct {
	ID                string          `firestore:"id"`
	UserID            string          `firestore:"userId"`
	StartedAt         time.Time       `firestore:"startedAt"`
	UpdatedAt         time.Time       `firestore:"updatedAt"`
	LiveSessionHandle string          `firestore:"liveSessionHandle"`
	Settings          SessionSettings `firestore:"settings"`
}

// SessionSettings holds user preferences for the session.
type SessionSettings struct {
	Voice      string `firestore:"voice"`
	Language   string `firestore:"language"`
	LiveModel  string `firestore:"liveModel"`
	Chattiness string `firestore:"chattiness"`
	Character  string `firestore:"character"`
}

// SessionMetrics tracks per-session behavioral metrics.
// Maps to Firestore document at sessions/{sessionId}/metrics
type SessionMetrics struct {
	SessionID         string     `firestore:"sessionId"`
	Utterances        int        `firestore:"utterances"`
	Responses         int        `firestore:"responses"`
	Interruptions     int        `firestore:"interruptions"`
	ResponseRate      float64    `firestore:"responseRate"`
	InterruptRate     float64    `firestore:"interruptRate"`
	SilenceThreshold  float64    `firestore:"silenceThreshold"`
	CooldownSeconds   float64    `firestore:"cooldownSeconds"`
	CurrentMood       string     `firestore:"currentMood"`
	MoodConfidence    float64    `firestore:"moodConfidence"`
	CelebrationCount  int        `firestore:"celebrationCount"`
	LastCelebrationAt *time.Time `firestore:"lastCelebrationAt,omitempty"`
	UpdatedAt         time.Time  `firestore:"updatedAt"`
}

// HistoryEntry represents a single interaction event in session history.
// Maps to Firestore documents at sessions/{sessionId}/history/{entryId}
type HistoryEntry struct {
	Timestamp    time.Time `firestore:"timestamp"`
	Type         string    `firestore:"type"` // vision_analysis | speech | engagement | interruption | celebration | mood_change | search
	Content      string    `firestore:"content"`
	Significance int       `firestore:"significance"`
}

// MemoryEntry represents a cross-session memory summary.
// Maps to Firestore document at users/{userId}/memory
type MemoryEntry struct {
	UserID          string           `firestore:"userId"`
	RecentSummaries []SessionSummary `firestore:"recentSummaries"`
	KnownTopics     []Topic          `firestore:"knownTopics"`
	UpdatedAt       time.Time        `firestore:"updatedAt"`
}

// SessionSummary stores a summary of a past session.
type SessionSummary struct {
	Date             time.Time `firestore:"date"`
	Summary          string    `firestore:"summary"`
	UnresolvedIssues []string  `firestore:"unresolvedIssues"`
}

// Topic represents a detected coding topic.
type Topic struct {
	Name          string    `firestore:"name"`
	LastMentioned time.Time `firestore:"lastMentioned"`
	Resolved      bool      `firestore:"resolved"`
}

// MoodState represents the current detected mood.
type MoodState struct {
	Mood            string    `firestore:"mood"` // focused | frustrated | stuck | idle
	Confidence      float64   `firestore:"confidence"`
	Signals         []string  `firestore:"signals"`
	SuggestedAction string    `firestore:"suggestedAction"`
	UpdatedAt       time.Time `firestore:"updatedAt"`
}

// SearchCacheEntry caches recent search results to avoid duplicates.
// Maps to Firestore documents at sessions/{sessionId}/searches/{searchId}
type SearchCacheEntry struct {
	Query     string    `firestore:"query"`
	Result    string    `firestore:"result"`
	Timestamp time.Time `firestore:"timestamp"`
}

// NavigatorReplay stores a completed navigator task replay and labels.
type NavigatorReplay struct {
	TaskID                  string             `firestore:"taskId"`
	UserID                  string             `firestore:"userId,omitempty"`
	SessionID               string             `firestore:"sessionId,omitempty"`
	Command                 string             `firestore:"command"`
	Outcome                 string             `firestore:"outcome"`
	OutcomeDetail           string             `firestore:"outcomeDetail,omitempty"`
	Surface                 string             `firestore:"surface,omitempty"`
	Summary                 string             `firestore:"summary,omitempty"`
	ReplayLabel             string             `firestore:"replayLabel,omitempty"`
	ResearchSummary         string             `firestore:"researchSummary,omitempty"`
	ResearchSources         []string           `firestore:"researchSources,omitempty"`
	Tags                    []string           `firestore:"tags,omitempty"`
	InitialAppName          string             `firestore:"initialAppName,omitempty"`
	InitialWindowTitle      string             `firestore:"initialWindowTitle,omitempty"`
	InitialContextHash      string             `firestore:"initialContextHash,omitempty"`
	LastVerifiedContextHash string             `firestore:"lastVerifiedContextHash,omitempty"`
	StartedAt               time.Time          `firestore:"startedAt,omitempty"`
	CompletedAt             time.Time          `firestore:"completedAt,omitempty"`
	UpdatedAt               time.Time          `firestore:"updatedAt,omitempty"`
	Attempts                []NavigatorAttempt `firestore:"attempts,omitempty"`
}

type NavigatorAttempt struct {
	ID               string    `firestore:"id"`
	TaskID           string    `firestore:"taskId,omitempty"`
	Command          string    `firestore:"command"`
	Surface          string    `firestore:"surface,omitempty"`
	Route            string    `firestore:"route"`
	RouteReason      string    `firestore:"routeReason,omitempty"`
	ContextHash      string    `firestore:"contextHash,omitempty"`
	ScreenshotSource string    `firestore:"screenshotSource,omitempty"`
	ScreenshotCached bool      `firestore:"screenshotCached,omitempty"`
	ScreenBasisID    string    `firestore:"screenBasisId,omitempty"`
	ActiveDisplayID  string    `firestore:"activeDisplayId,omitempty"`
	TargetDisplayID  string    `firestore:"targetDisplayId,omitempty"`
	Outcome          string    `firestore:"outcome,omitempty"`
	OutcomeDetail    string    `firestore:"outcomeDetail,omitempty"`
	StartedAt        time.Time `firestore:"startedAt,omitempty"`
	CompletedAt      time.Time `firestore:"completedAt,omitempty"`
}
