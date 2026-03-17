# P2-10: MCP (Model Context Protocol) Integration

## SDK Verification (CONFIRMED)
- ADK v0.6.0 `google.golang.org/adk/tool/mcptoolset` — EXISTS
- `mcptoolset.New(Config) (tool.Toolset, error)` — EXISTS
- `Config{Client, Transport, ToolFilter, RequireConfirmation}` — EXISTS
- Agent types for multi-agent MCP: llmagent, parallelagent — ALL EXISTS
- Live API compatible: N/A (ADK orchestrator feature)

## Current Code (adk-orchestrator)
- `internal/agents/graph/graph.go:49-216` — agent graph with 9 agents
- `internal/agents/tooluse/tooluse.go:64-81` — Tool Agent struct
- `internal/models/models.go:59-68` — ToolKind enum (search, maps, url_context, code_execution, file_search)
- `main.go:284-289` — agent graph build
- `main.go:292-310` — ADK runner initialization

## Implementation
1. Create MCP server wrappers for external services
2. Register MCP toolsets in the ADK agent graph
3. Add new tool kinds for MCP-backed tools
4. Route queries to appropriate MCP-equipped agents

## Phase 1 — File System MCP (most relevant for desktop companion)
```go
// internal/agents/tooluse/mcp.go
package tooluse

import (
    "google.golang.org/adk/tool/mcptoolset"
    "google.golang.org/adk/tool"
)

func NewFileSystemMCPToolset() (tool.Toolset, error) {
    return mcptoolset.New(mcptoolset.Config{
        Transport: mcptoolset.StdioTransport{
            Command: "npx",
            Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/Users"},
        },
        RequireConfirmation: true, // Safety: confirm before file operations
    })
}
```

## Phase 2 — GitHub MCP
```go
func NewGitHubMCPToolset() (tool.Toolset, error) {
    return mcptoolset.New(mcptoolset.Config{
        Transport: mcptoolset.StdioTransport{
            Command: "npx",
            Args:    []string{"-y", "@modelcontextprotocol/server-github"},
            Env:     []string{"GITHUB_PERSONAL_ACCESS_TOKEN=" + os.Getenv("GITHUB_TOKEN")},
        },
    })
}
```

## Integration in agent graph
```go
// In graph.go, add MCP-equipped agents:
func buildGraph(genaiClient *genai.Client) (*graph.Graph, error) {
    // ... existing agents ...

    // New: File system agent with MCP
    fsMCPToolset, err := tooluse.NewFileSystemMCPToolset()
    if err != nil {
        return nil, fmt.Errorf("filesystem MCP: %w", err)
    }

    fileAgent, err := llmagent.New(llmagent.Config{
        Name:        "file_agent",
        Description: "Manages project files: search, read, create, modify",
        Model:       geminiconfig.ToolModel,
        Toolsets:    []tool.Toolset{fsMCPToolset},
        Instruction: "You help users manage their project files...",
    })
    if err != nil {
        return nil, err
    }

    // Add to Wave 3 or create new Wave 4
    // ... graph composition ...
}
```

## New ToolKind routing
```go
// In models.go:
const (
    // ... existing kinds ...
    ToolKindFileSystem = "filesystem"
    ToolKindGitHub     = "github"
)

// In tooluse.go detectFastPath():
var fileSystemKeywords = []string{
    "file", "directory", "folder", "create file", "read file",
    "find file", "project structure", "list files",
    "파일", "디렉토리", "폴더", "파일 찾기", "파일 읽기",
}
```

## Verification
- Ask "Show me the project structure"
- Verify file_agent is routed via MCP toolset
- Verify RequireConfirmation prompts user before writes
- Check MCP server process starts and stops correctly

## Risks
- MCP server as child process — need lifecycle management (start/stop/restart)
- npx cold start adds latency (~2-5s first call)
- File system access needs security sandboxing
- Memory overhead per MCP server process
