package topic

import (
	"strings"
)

var keywords = []string{
	"error", "bug", "crash", "fix", "test", "build", "deploy",
	"api", "database", "auth", "performance", "refactor", "review",
	"docker", "kubernetes", "cloud", "git", "merge", "pr",
	"typescript", "golang", "swift", "python", "javascript",
	"deadline", "meeting", "design", "architecture",
}

func Detect(text string) []string {
	lower := strings.ToLower(text)
	seen := map[string]bool{}
	var found []string
	for _, kw := range keywords {
		if strings.Contains(lower, kw) && !seen[kw] {
			seen[kw] = true
			found = append(found, kw)
		}
	}
	return found
}
