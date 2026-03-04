// Package graph wires all 9 VibeCat agents into an ADK sequential agent graph.
package graph

import (
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/workflowagents/sequentialagent"
	"google.golang.org/genai"

	"vibecat/adk-orchestrator/internal/agents/celebration"
	"vibecat/adk-orchestrator/internal/agents/engagement"
	"vibecat/adk-orchestrator/internal/agents/mediator"
	"vibecat/adk-orchestrator/internal/agents/memory"
	"vibecat/adk-orchestrator/internal/agents/mood"
	"vibecat/adk-orchestrator/internal/agents/scheduler"
	"vibecat/adk-orchestrator/internal/agents/search"
	"vibecat/adk-orchestrator/internal/agents/vision"
	"vibecat/adk-orchestrator/internal/store"
)

// New creates the full VibeCat agent graph.
// Execution order:
//  1. MemoryAgent   — inject cross-session context at session start
//  2. VisionAgent   — analyze screenshot
//  3. MoodDetector  — classify developer mood from vision signals
//  4. CelebrationTrigger — detect success events
//  5. Mediator      — decide whether to speak (mood-aware, celebration bypass)
//  6. AdaptiveScheduler — adjust cooldown/silence thresholds
//  7. EngagementAgent — proactive engagement on silence
//  8. SearchBuddy   — Google Search when stuck/frustrated
//
// storeClient may be nil — MemoryAgent will run in stub mode.
func New(genaiClient *genai.Client, storeClient *store.Client) (agent.Agent, error) {
	memoryAgent, err := agent.New(agent.Config{
		Name:        "memory_agent",
		Description: "Retrieves cross-session context and stores end-of-session summaries",
		Run:         memory.New(genaiClient, storeClient).Run,
	})
	if err != nil {
		return nil, err
	}

	visionAgent, err := agent.New(agent.Config{
		Name:        "vision_agent",
		Description: "Analyzes developer screen captures for errors, success, and context",
		Run:         vision.New(genaiClient).Run,
	})
	if err != nil {
		return nil, err
	}

	moodAgent, err := agent.New(agent.Config{
		Name:        "mood_detector",
		Description: "Classifies developer mood from vision signals and interaction patterns",
		Run:         mood.New().Run,
	})
	if err != nil {
		return nil, err
	}

	celebrationAgent, err := agent.New(agent.Config{
		Name:        "celebration_trigger",
		Description: "Detects success events and triggers celebration responses",
		Run:         celebration.New().Run,
	})
	if err != nil {
		return nil, err
	}

	mediatorAgent, err := agent.New(agent.Config{
		Name:        "mediator",
		Description: "Decides when to speak based on significance, cooldown, mood, and celebration",
		Run:         mediator.New().Run,
	})
	if err != nil {
		return nil, err
	}

	schedulerAgent, err := agent.New(agent.Config{
		Name:        "adaptive_scheduler",
		Description: "Adjusts timing thresholds based on interaction rate",
		Run:         scheduler.New().Run,
	})
	if err != nil {
		return nil, err
	}

	engagementAgent, err := agent.New(agent.Config{
		Name:        "engagement_agent",
		Description: "Proactively engages when developer has been silent too long",
		Run:         engagement.New().Run,
	})
	if err != nil {
		return nil, err
	}

	searchAgent, err := agent.New(agent.Config{
		Name:        "search_buddy",
		Description: "Searches Google for solutions when developer is stuck or frustrated",
		Run:         search.New(genaiClient).Run,
	})
	if err != nil {
		return nil, err
	}

	graph, err := sequentialagent.New(sequentialagent.Config{
		AgentConfig: agent.Config{
			Name:        "vibecat_graph",
			Description: "VibeCat 9-agent orchestration graph for developer companion intelligence",
			SubAgents: []agent.Agent{
				memoryAgent,
				visionAgent,
				moodAgent,
				celebrationAgent,
				mediatorAgent,
				schedulerAgent,
				engagementAgent,
				searchAgent,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return graph, nil
}
