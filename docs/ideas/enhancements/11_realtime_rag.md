# P2-11: Real-time RAG

## SDK Verification
- Gemini embedding model: `text-embedding-005` or `gemini-embedding-2-preview` — available via GenAI SDK
- `client.Models.EmbedContent()` — EXISTS in Go SDK
- No built-in vector store in Go SDK — custom implementation required
- Live API compatible: Partially (embedding is batch-only, but context injection works with Live)

## Current Code (adk-orchestrator)
- `internal/agents/tooluse/tooluse.go` — has `ToolKindSearch` for Google Search grounding
- Google Search already provides web-grounded answers
- No custom document embedding or vector search

## Concept
Embed user's project documents (README, docs, code comments) into vectors. During live conversation, match user queries against embedded docs and inject relevant chunks as context.

## Implementation
1. Create embedding pipeline for project documents
2. Store embeddings in Firestore (vector field) or in-memory
3. On user query, compute query embedding and find similar chunks
4. Inject top-K chunks into ADK orchestrator context before processing

## Go Code
```go
// internal/rag/embedder.go
package rag

import (
    "context"
    "google.golang.org/genai"
)

const embeddingModel = "text-embedding-005"

type Chunk struct {
    ID        string
    Text      string
    Source    string    // file path or URL
    Embedding []float32
}

type Store struct {
    chunks []Chunk
}

func NewStore() *Store {
    return &Store{}
}

func (s *Store) Embed(ctx context.Context, client *genai.Client, text, source string) error {
    // Split text into chunks (~500 tokens each)
    chunks := splitIntoChunks(text, 500)
    
    for _, chunk := range chunks {
        resp, err := client.Models.EmbedContent(ctx, embeddingModel, 
            &genai.EmbedContentConfig{
                Contents: []*genai.Content{{Parts: []*genai.Part{{Text: chunk}}}},
            },
        )
        if err != nil {
            return err
        }
        
        s.chunks = append(s.chunks, Chunk{
            ID:        generateID(),
            Text:      chunk,
            Source:    source,
            Embedding: resp.Embeddings[0].Values,
        })
    }
    return nil
}

func (s *Store) Search(ctx context.Context, client *genai.Client, query string, topK int) ([]Chunk, error) {
    // Embed query
    resp, err := client.Models.EmbedContent(ctx, embeddingModel,
        &genai.EmbedContentConfig{
            Contents: []*genai.Content{{Parts: []*genai.Part{{Text: query}}}},
        },
    )
    if err != nil {
        return nil, err
    }
    queryEmb := resp.Embeddings[0].Values

    // Cosine similarity search
    type scored struct {
        chunk Chunk
        score float64
    }
    var results []scored
    for _, c := range s.chunks {
        score := cosineSimilarity(queryEmb, c.Embedding)
        results = append(results, scored{c, score})
    }
    
    // Sort by score descending
    sort.Slice(results, func(i, j int) bool {
        return results[i].score > results[j].score
    })
    
    // Return top K
    var topResults []Chunk
    for i := 0; i < topK && i < len(results); i++ {
        topResults = append(topResults, results[i].chunk)
    }
    return topResults, nil
}

func cosineSimilarity(a, b []float32) float64 {
    var dot, normA, normB float64
    for i := range a {
        dot += float64(a[i]) * float64(b[i])
        normA += float64(a[i]) * float64(a[i])
        normB += float64(b[i]) * float64(b[i])
    }
    if normA == 0 || normB == 0 {
        return 0
    }
    return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func splitIntoChunks(text string, maxTokens int) []string {
    // Simple sentence-boundary chunking
    // Each chunk ~maxTokens tokens (approx 4 chars per token)
    maxChars := maxTokens * 4
    var chunks []string
    for len(text) > 0 {
        end := maxChars
        if end > len(text) {
            end = len(text)
        }
        // Find sentence boundary
        if end < len(text) {
            for i := end; i > end/2; i-- {
                if text[i] == '.' || text[i] == '\n' {
                    end = i + 1
                    break
                }
            }
        }
        chunks = append(chunks, strings.TrimSpace(text[:end]))
        text = text[end:]
    }
    return chunks
}
```

## Integration with ADK orchestrator
```go
// In main.go or orchestrator initialization:
ragStore := rag.NewStore()

// Index project docs at startup (or on-demand)
func (o *orchestrator) indexProjectDocs(ctx context.Context, projectPath string) error {
    files, _ := filepath.Glob(filepath.Join(projectPath, "**/*.md"))
    for _, f := range files {
        content, _ := os.ReadFile(f)
        if err := o.ragStore.Embed(ctx, o.genaiClient, string(content), f); err != nil {
            slog.Warn("failed to embed", "file", f, "err", err)
        }
    }
    return nil
}

// In analyzeHandler, before running agent graph:
func (o *orchestrator) analyzeHandler(w http.ResponseWriter, r *http.Request) {
    // ... parse request ...
    
    // RAG: search for relevant context
    chunks, _ := o.ragStore.Search(ctx, o.genaiClient, userQuery, 3)
    ragContext := formatChunksAsContext(chunks)
    
    // Inject into session state
    session.State().Set("rag_context", ragContext)
    
    // Run agent graph (memory agent can read rag_context from state)
    // ...
}
```

## Verification
- Index a project README
- Ask a question about the project
- Verify RAG returns relevant chunks (check cosine similarity scores)
- Compare answer quality with vs without RAG context

## Risks
- Embedding API cost (per-token pricing)
- In-memory vector store doesn't scale (use Firestore vector field for production)
- Cold start: embedding takes time for large projects
- Chunk quality depends on splitting strategy
- May conflict with Google Search grounding (duplicate information sources)
