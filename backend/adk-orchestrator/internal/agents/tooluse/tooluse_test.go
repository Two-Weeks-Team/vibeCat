package tooluse

import (
	"context"
	"testing"

	"vibecat/adk-orchestrator/internal/models"
)

func TestDetectFastPath(t *testing.T) {
	tests := []struct {
		name              string
		query             string
		fileSearchEnabled bool
		want              models.ToolKind
	}{
		{name: "url context", query: "이 페이지 요약해줘 https://ai.google.dev/gemini-api/docs/google-search", want: models.ToolKindURLContext},
		{name: "maps", query: "강남역 근처 카페 추천해줘", want: models.ToolKindMaps},
		{name: "code execution", query: "37도를 화씨로 계산해줘", want: models.ToolKindCodeExecution},
		{name: "file search", query: "업로드한 파일에서 auth 섹션 찾아줘", fileSearchEnabled: true, want: models.ToolKindFileSearch},
		{name: "search", query: "Go websocket close 1006 공식 문서 찾아줘", want: models.ToolKindSearch},
		{name: "none", query: "고마워", want: models.ToolKindNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectFastPath(tt.query, tt.fileSearchEnabled)
			if got != tt.want {
				t.Fatalf("detectFastPath(%q) = %q, want %q", tt.query, got, tt.want)
			}
		})
	}
}

func TestResolveNilClientFallsBackToFastPathOnly(t *testing.T) {
	a := New(nil, nil)
	if got := a.Resolve(context.Background(), models.ToolRequest{Query: "Go websocket close 1006 공식 문서 찾아줘"}); got != nil {
		t.Fatalf("Resolve() = %+v, want nil with no client", got)
	}
}

func TestDedupeParagraphs(t *testing.T) {
	input := "alpha\nbeta\nalpha\n\nbeta\nGamma"
	got := dedupeParagraphs(input)
	if got != "alpha\nbeta\nGamma" {
		t.Fatalf("dedupeParagraphs() = %q", got)
	}
}
