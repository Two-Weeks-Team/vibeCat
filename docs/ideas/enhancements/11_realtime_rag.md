# P2-11: Real-time RAG

## Status

Implement in two scopes:

- Scope A: server-owned or uploaded docs
- Scope B: local workspace docs only after adding an explicit client upload/sync path

## Source-Verified Facts

- Go SDK v1.49.0 supports `Models.EmbedContent(...)`.
- `EmbedContentConfig` supports `TaskType` and `OutputDimensionality`.
- Embeddings docs recommend `RETRIEVAL_DOCUMENT` for documents and `RETRIEVAL_QUERY` for queries.
- Firestore now supports vector storage and KNN search (`FindNearest` in Go samples/docs).

## Critical Architecture Boundary

The backend cannot directly scan arbitrary local project files on the user's Mac.

So this original pattern is invalid for production:

- cloud service receives a local path
- cloud service runs `WalkDir()` on that local path

That only works for files already present on the server.

## Implementation Decision

### Phase 1: server-owned / uploaded docs

Support:

- repo docs already present in the deployed service image
- docs explicitly uploaded or synced by the client

Use:

- stable embedding model: `gemini-embedding-001`
- task types:
  - `RETRIEVAL_DOCUMENT` for chunks
  - `RETRIEVAL_QUERY` for queries

### Phase 2: local workspace sync

Add a client-originated ingestion path:

- client extracts allowed files
- client chunks and uploads, or uploads raw docs for server-side chunking
- backend embeds and stores under:
  - `user_id`
  - `workspace_id`
  - `source_path`

## Recommended Storage

Use Firestore vector fields instead of an in-memory store for production.

Suggested collection:

- `rag_chunks/{chunkId}`

Fields:

- `userId`
- `workspaceId`
- `sourcePath`
- `chunkText`
- `embedding`
- `updatedAt`
- `visibility`

## Concrete File Changes

- `backend/adk-orchestrator/internal/rag/`
  - chunker
  - embedder
  - Firestore repository
  - retriever
- `backend/adk-orchestrator/main.go`
  - upload/index endpoints
  - retrieval hook for analysis requests
- client upload/sync path later in:
  - `VibeCat`
  - or a separate helper/CLI if preferred

## Retrieval Flow

```text
User query
  -> query embedding (RETRIEVAL_QUERY)
  -> Firestore FindNearest on scoped workspace/user chunks
  -> top-K chunks
  -> inject compact context into orchestrator request
```

## Scoping Rules

- never mix one user's chunks with another user's results
- separate workspaces
- cap injected context size
- log which chunks were used

## Acceptance Criteria

1. Scope A works without assuming local filesystem access on Cloud Run.
2. Firestore vector search returns relevant top-K chunks.
3. Injected context stays bounded and traceable.

## Risks

- ingestion complexity for local workspaces
- stale chunks without update/delete strategy
- noisy retrieval if chunking is poor

## Sources

- [Gemini embeddings guide](https://ai.google.dev/gemini-api/docs/embeddings)
- [Firestore vector search](https://cloud.google.com/firestore/native/docs/vector-search)
