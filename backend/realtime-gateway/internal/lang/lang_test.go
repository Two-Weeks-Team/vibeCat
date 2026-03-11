package lang

import "testing"

func TestNormalizeLanguage(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "Korean"},
		{"ko", "Korean"},
		{"English", "English"},
		{"ja", "Japanese"},
		{"日本語", "Japanese"},
	}

	for _, tt := range tests {
		if got := NormalizeLanguage(tt.input); got != tt.want {
			t.Fatalf("NormalizeLanguage(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
