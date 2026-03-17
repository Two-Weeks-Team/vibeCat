# P2-9: Always-On Memory Agent (Sleep/Consolidation Pattern)

## SDK Verification
- ADK v0.6.0 `memory.Service` interface — EXISTS (InMemoryService)
- ADK v0.6.0 `session.Service` with state prefixes `app:`, `user:`, `temp:` — EXISTS
- No built-in Firestore memory service — custom implementation required
- Live API compatible: N/A (ADK orchestrator feature)

## Current Code (adk-orchestrator)
- `internal/agents/memory/memory.go:28-40` — Memory Agent struct
- `internal/agents/memory/memory.go:99-114` — `retrieveMemory()` calls `store.GetMemory()`
- `internal/agents/memory/memory.go:118-152` — `SaveSessionSummary()` 
- `internal/agents/memory/memory.go:156-194` — `SaveTaskSummary()`
- `internal/store/firestore.go:84-100` — `GetMemory()` from `users/{userId}/memory/data`
- `internal/store/firestore.go:103-115` — `UpdateMemory()`
- `internal/store/models.go:54-59` — `MemoryEntry{UserID, RecentSummaries, KnownTopics, UpdatedAt}`
- `internal/store/models.go:62-66` — `SessionSummary{Date, Summary, UnresolvedIssues}`

## Current Limitations
- Passive storage only — memory is written at session end, read at session start
- No cross-memory pattern discovery
- No importance scoring
- Limited to 10 summaries and 20 topics (hard-coded limits)
- No entity extraction or relationship tracking

## Enhancement — Sleep/Consolidation Pattern
Inspired by `gemini/agents/always-on-memory-agent/`: human-like memory with active consolidation during idle periods.

## Implementation
1. Extend MemoryEntry with entity extraction, importance scores, and connections
2. Add consolidation agent that runs on a timer (every 30 min)
3. Consolidation discovers cross-memory patterns and generates insights
4. Add semantic search for memory retrieval (not just last-N)

## Go Code
```go
// internal/store/models.go — Extended memory model:
type MemoryEntry struct {
    UserID          string           `firestore:"userId"`
    RecentSummaries []SessionSummary `firestore:"recentSummaries"`
    KnownTopics     []Topic          `firestore:"knownTopics"`
    Entities        []Entity         `firestore:"entities"`
    Connections     []Connection     `firestore:"connections"`
    Insights        []Insight        `firestore:"insights"`
    UpdatedAt       time.Time        `firestore:"updatedAt"`
}

type Entity struct {
    Name       string    `firestore:"name"`
    Type       string    `firestore:"type"` // "project", "tool", "person", "concept"
    Mentions   int       `firestore:"mentions"`
    LastSeen   time.Time `firestore:"lastSeen"`
    Importance float64   `firestore:"importance"` // 0.0-1.0
}

type Connection struct {
    FromEntity string  `firestore:"from"`
    ToEntity   string  `firestore:"to"`
    Relation   string  `firestore:"relation"` // "uses", "part_of", "related_to"
    Strength   float64 `firestore:"strength"`
}

type Insight struct {
    Text      string    `firestore:"text"`
    Source    []string  `firestore:"source"` // summary IDs that led to this insight
    CreatedAt time.Time `firestore:"createdAt"`
}
```

```go
// internal/agents/memory/consolidator.go
package memory

type Consolidator struct {
    client      *genai.Client
    store       *store.Client
    interval    time.Duration
}

func NewConsolidator(client *genai.Client, store *store.Client) *Consolidator {
    return &Consolidator{
        client:   client,
        store:    store,
        interval: 30 * time.Minute,
    }
}

func (c *Consolidator) Start(ctx context.Context, userID string) {
    ticker := time.NewTicker(c.interval)
    go func() {
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                if err := c.consolidate(ctx, userID); err != nil {
                    slog.Error("memory consolidation failed", "err", err, "user", userID)
                }
            }
        }
    }()
}

func (c *Consolidator) consolidate(ctx context.Context, userID string) error {
    memory, err := c.store.GetMemory(ctx, userID)
    if err != nil {
        return err
    }

    // Generate consolidation prompt with all recent summaries
    prompt := buildConsolidationPrompt(memory)
    
    resp, err := c.client.Models.GenerateContent(ctx, geminiconfig.LiteTextModel, 
        []*genai.Content{{Role: "user", Parts: []*genai.Part{{Text: prompt}}}},
        &genai.GenerateContentConfig{
            ResponseMIMEType: "application/json",
            ResponseSchema: consolidationSchema(),
        },
    )
    if err != nil {
        return err
    }
    
    // Parse and merge new entities, connections, insights
    updates := parseConsolidationResult(resp)
    return c.store.UpdateMemory(ctx, userID, updates)
}

func buildConsolidationPrompt(memory *store.MemoryEntry) string {
    return fmt.Sprintf(`Analyze these session summaries and extract:
1. New entities (projects, tools, concepts the user works with)
2. Connections between entities
3. Cross-session insights (patterns, preferences, recurring issues)

Recent sessions:
%s

Known entities:
%s

Return JSON with: new_entities, new_connections, insights`,
        formatSummaries(memory.RecentSummaries),
        formatEntities(memory.Entities))
}
```

## Verification
- Run 3+ sessions with consistent topics
- Trigger consolidation manually
- Verify entities are extracted correctly
- Verify insights reference source summaries
- Check memory retrieval returns relevant context

## Risks
- Consolidation adds API cost (LLM calls during idle)
- Entity extraction quality depends on LLM accuracy
- Firestore write size limits may constrain large memory stores
- Need to handle race conditions between session writes and consolidation
