# P0-2: Context Caching

## Title
P0-2: Context Caching

## SDK Verification (CONFIRMED via go doc v1.49.0)
- `genai.Client.Caches` service — EXISTS with Create, Get, Update, Delete, List, All
- `genai.CreateCachedContentConfig{TTL, Contents, SystemInstruction, Tools, ToolConfig}` — EXISTS
- `genai.CachedContent{Name, DisplayName, Model, CreateTime, UpdateTime, ExpireTime}` — EXISTS
- `GenerateContentConfig.CachedContentName string` — EXISTS (field name is `CachedContentName` in Go, JSON tag `cachedContent`)
- Live API compatible: NO (batch-only)

## Applicability
ADK Orchestrator only (not gateway). The orchestrator uses batch `GenerateContent` calls for vision analysis, mood detection, tool classification, etc.

## Current Code (adk-orchestrator)
- `internal/geminiconfig/models.go` lines 3-16 — model constants
- `internal/agents/tooluse/tooluse.go` — uses batch GenerateContent for tool classification
- `internal/agents/memory/memory.go:216` — uses LiteTextModel for memory summarization
- System prompts are rebuilt per-request across all 9 agents

## Implementation
1. Create a cache manager in `internal/cache/manager.go`
2. Cache common system instructions + tool declarations for each model
3. Set TTL to 1 hour, auto-refresh before expiry
4. Use `CachedContentName` in `GenerateContentConfig` for batch requests

## Go Code
```go
// internal/cache/manager.go
package cache

import (
    "context"
    "sync"
    "time"
    "google.golang.org/genai"
)

type Manager struct {
    client     *genai.Client
    mu         sync.RWMutex
    caches     map[string]*genai.CachedContent // model -> cache
    refreshTTL time.Duration
}

func New(client *genai.Client) *Manager {
    return &Manager{
        client:     client,
        caches:     make(map[string]*genai.CachedContent),
        refreshTTL: 50 * time.Minute, // Refresh before 1h TTL expires
    }
}

func (m *Manager) GetOrCreate(ctx context.Context, model string, systemInstruction string, tools []*genai.Tool) (string, error) {
    m.mu.RLock()
    if cached, ok := m.caches[model]; ok {
        if time.Until(cached.ExpireTime) > 5*time.Minute {
            m.mu.RUnlock()
            return cached.Name, nil
        }
    }
    m.mu.RUnlock()

    m.mu.Lock()
    defer m.mu.Unlock()

    cached, err := m.client.Caches.Create(ctx, model, &genai.CreateCachedContentConfig{
        TTL:               time.Hour,
        DisplayName:       "vibecat-" + model,
        SystemInstruction: &genai.Content{Parts: []*genai.Part{{Text: systemInstruction}}},
        Tools:             tools,
    })
    if err != nil {
        return "", err
    }
    m.caches[model] = cached
    return cached.Name, nil
}
```

## Minimum token requirement
2,048 tokens. VibeCat's system prompts exceed this.

## Cost
~75% reduction on cached input tokens.

## Verification
Check `response.UsageMetadata.CachedContentTokenCount > 0` in responses.

## Risks
- Cache is model-specific — separate cache per model variant
- Not applicable to Live API sessions (gateway)
- Adds complexity to startup (cache warm-up)
