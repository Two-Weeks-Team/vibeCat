package lang

import "strings"

func NormalizeLanguage(language string) string {
	trimmed := strings.TrimSpace(language)
	if trimmed == "" {
		return "Korean"
	}
	lower := strings.ToLower(trimmed)
	switch lower {
	case "ko", "kr", "korean", "korean language", "한국어":
		return "Korean"
	case "en", "eng", "english", "english language":
		return "English"
	case "ja", "jp", "jpn", "japanese", "japanese language", "日本語":
		return "Japanese"
	default:
		return trimmed
	}
}
