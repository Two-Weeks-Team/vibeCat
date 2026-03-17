# P2-10: MCP Integration

## Status

Phase 1 for local development, later production rollout with remote or bundled transports only.

## Source-Verified Facts

- ADK v0.6.0 provides `google.golang.org/adk/tool/mcptoolset`.
- `mcptoolset.Config` includes:
  - `Transport mcp.Transport`
  - `RequireConfirmation bool`
  - `RequireConfirmationProvider`
- Cloud Run does not give us a free local Node/npm runtime for `npx`-based MCP servers.

## Critical Architecture Boundary

Filesystem MCP on Cloud Run cannot access a user's Mac filesystem.

Therefore:

- local filesystem MCP is valid for local development only
- production cloud MCP must target cloud-accessible systems
- user-local file access must remain client-side or use an explicit upload/sync path

## Recommended Rollout

### Phase 1: local dev only

Use stdio MCP transport for:

- local filesystem experiments
- local GitHub workflows
- tool UX validation

### Phase 2: production

Use one of:

- dedicated remote MCP service over HTTP
- bundled binary in the container image

Prefer cloud-safe MCP targets first:

- GitHub
- issue tracker
- remote knowledge sources

Do not start with local filesystem MCP as a production requirement.

## Concrete File Changes

### New package

- `backend/adk-orchestrator/internal/mcp/`
  - transport factory
  - env-driven config
  - health / lifecycle wrapper

### Graph integration

- `backend/adk-orchestrator/internal/agents/graph/graph.go`
  - register MCP-backed agents only behind feature flags

## Environment Modes

### Local

- stdio transport
- `RequireConfirmation = true` for write-capable tools

### Production

- remote HTTP transport or bundled binary
- connection health checks
- startup failure should disable MCP features cleanly, not crash the service

## Suggested First MCP Targets

1. GitHub MCP
2. remote docs/search MCP
3. client-side or uploaded-file workflows only after explicit design

## Acceptance Criteria

1. MCP failures degrade gracefully.
2. Production deployment does not depend on `npx`.
3. All write-capable MCP tools require confirmation.

## Risks

- cold start / child-process overhead
- transport instability
- security boundaries if tool permissions are too broad

## Sources

- [ADK MCP docs](https://google.github.io/adk-docs/tools/mcp-tools/)
