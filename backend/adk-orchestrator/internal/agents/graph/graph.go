package graph

import (
	"log/slog"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/workflowagents/loopagent"
	"google.golang.org/adk/agent/workflowagents/parallelagent"
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
func beforeLog() agent.BeforeAgentCallback {
	return func(ctx agent.CallbackContext) (*genai.Content, error) {
		slog.Info("[ADK] agent started", "agent", ctx.AgentName())
		return nil, nil
	}
}

func afterLog() agent.AfterAgentCallback {
	return func(ctx agent.CallbackContext) (*genai.Content, error) {
		slog.Info("[ADK] agent completed", "agent", ctx.AgentName())
		return nil, nil
	}
}

func New(genaiClient *genai.Client, storeClient *store.Client, apiKey ...string) (agent.Agent, error) {
	cbs := agent.Config{
		BeforeAgentCallbacks: []agent.BeforeAgentCallback{beforeLog()},
		AfterAgentCallbacks:  []agent.AfterAgentCallback{afterLog()},
	}

	memoryAgent, err := agent.New(agent.Config{
		Name:                 "memory_agent",
		Description:          "Retrieves cross-session context and stores end-of-session summaries",
		Run:                  memory.New(genaiClient, storeClient).Run,
		BeforeAgentCallbacks: cbs.BeforeAgentCallbacks,
		AfterAgentCallbacks:  cbs.AfterAgentCallbacks,
	})
	if err != nil {
		return nil, err
	}

	visionAgent, err := agent.New(agent.Config{
		Name:                 "vision_agent",
		Description:          "Analyzes developer screen captures for errors, success, and context",
		Run:                  vision.New(genaiClient).Run,
		BeforeAgentCallbacks: cbs.BeforeAgentCallbacks,
		AfterAgentCallbacks:  cbs.AfterAgentCallbacks,
	})
	if err != nil {
		return nil, err
	}

	moodAgent, err := agent.New(agent.Config{
		Name:                 "mood_detector",
		Description:          "Classifies developer mood from vision signals and interaction patterns",
		Run:                  mood.New().Run,
		BeforeAgentCallbacks: cbs.BeforeAgentCallbacks,
		AfterAgentCallbacks:  cbs.AfterAgentCallbacks,
	})
	if err != nil {
		return nil, err
	}

	celebrationAgent, err := agent.New(agent.Config{
		Name:                 "celebration_trigger",
		Description:          "Detects success events and triggers celebration responses",
		Run:                  celebration.New(genaiClient).Run,
		BeforeAgentCallbacks: cbs.BeforeAgentCallbacks,
		AfterAgentCallbacks:  cbs.AfterAgentCallbacks,
	})
	if err != nil {
		return nil, err
	}

	mediatorAgent, err := agent.New(agent.Config{
		Name:                 "mediator",
		Description:          "Decides when to speak based on significance, cooldown, mood, and celebration",
		Run:                  mediator.New(genaiClient).Run,
		BeforeAgentCallbacks: cbs.BeforeAgentCallbacks,
		AfterAgentCallbacks:  cbs.AfterAgentCallbacks,
	})
	if err != nil {
		return nil, err
	}

	schedulerAgent, err := agent.New(agent.Config{
		Name:                 "adaptive_scheduler",
		Description:          "Adjusts timing thresholds based on interaction rate",
		Run:                  scheduler.New().Run,
		BeforeAgentCallbacks: cbs.BeforeAgentCallbacks,
		AfterAgentCallbacks:  cbs.AfterAgentCallbacks,
	})
	if err != nil {
		return nil, err
	}

	engagementAgent, err := agent.New(agent.Config{
		Name:                 "engagement_agent",
		Description:          "Proactively engages when developer has been silent too long",
		Run:                  engagement.New(genaiClient).Run,
		BeforeAgentCallbacks: cbs.BeforeAgentCallbacks,
		AfterAgentCallbacks:  cbs.AfterAgentCallbacks,
	})
	if err != nil {
		return nil, err
	}

	searchAgent, err := agent.New(agent.Config{
		Name:                 "search_buddy",
		Description:          "Searches Google for solutions when developer is stuck or frustrated",
		Run:                  search.New(genaiClient).Run,
		BeforeAgentCallbacks: cbs.BeforeAgentCallbacks,
		AfterAgentCallbacks:  cbs.AfterAgentCallbacks,
	})
	if err != nil {
		return nil, err
	}

	key := ""
	if len(apiKey) > 0 {
		key = apiKey[0]
	}

	searchSubAgents := []agent.Agent{searchAgent}
	llmSearchAgent, llmSearchErr := search.NewLLMSearchAgent(key, "Korean")
	if llmSearchErr != nil {
		slog.Warn("failed to create LLM search agent — using custom search only", "error", llmSearchErr)
	} else {
		searchSubAgents = append(searchSubAgents, llmSearchAgent)
		slog.Info("LLM search agent wired into search loop (llmagent + functiontool + geminitool)")
	}

	searchLoop, err := loopagent.New(loopagent.Config{
		AgentConfig: agent.Config{
			Name:        "search_refinement_loop",
			Description: "Iterative search refinement — runs search agents up to 2 times for better results",
			SubAgents:   searchSubAgents,
		},
		MaxIterations: 2,
	})
	if err != nil {
		return nil, err
	}

	wave3SubAgents := []agent.Agent{mediatorAgent, schedulerAgent, engagementAgent, searchLoop}

	wave1, err := parallelagent.New(parallelagent.Config{
		AgentConfig: agent.Config{
			Name:        "wave1_perception",
			Description: "Parallel: Vision analysis + Memory retrieval",
			SubAgents:   []agent.Agent{visionAgent, memoryAgent},
		},
	})
	if err != nil {
		return nil, err
	}

	wave2, err := parallelagent.New(parallelagent.Config{
		AgentConfig: agent.Config{
			Name:        "wave2_emotion",
			Description: "Parallel: Mood detection + Celebration check",
			SubAgents:   []agent.Agent{moodAgent, celebrationAgent},
		},
	})
	if err != nil {
		return nil, err
	}

	wave3, err := sequentialagent.New(sequentialagent.Config{
		AgentConfig: agent.Config{
			Name:        "wave3_decision",
			Description: "Sequential: Decision agents that depend on perception + emotion results",
			SubAgents:   wave3SubAgents,
		},
	})
	if err != nil {
		return nil, err
	}

	graph, err := sequentialagent.New(sequentialagent.Config{
		AgentConfig: agent.Config{
			Name:        "vibecat_graph",
			Description: "VibeCat 9-agent orchestration: perception(parallel) → emotion(parallel) → decision(sequential)",
			SubAgents:   []agent.Agent{wave1, wave2, wave3},
		},
	})
	if err != nil {
		return nil, err
	}

	return graph, nil
}
