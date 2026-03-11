package tts

import "testing"

func TestBuildConfig_WithVoice(t *testing.T) {
	cfg := Config{Voice: "Puck", Language: "Korean", Text: "test"}
	gc := BuildConfig(cfg)
	if gc.SpeechConfig.VoiceConfig.PrebuiltVoiceConfig.VoiceName != "Puck" {
		t.Errorf("expected Puck, got %s", gc.SpeechConfig.VoiceConfig.PrebuiltVoiceConfig.VoiceName)
	}
}

func TestBuildConfig_DefaultVoice(t *testing.T) {
	cfg := Config{Voice: "", Language: "ko", Text: "test"}
	gc := BuildConfig(cfg)
	if gc.SpeechConfig.VoiceConfig.PrebuiltVoiceConfig.VoiceName != "Zephyr" {
		t.Errorf("expected Zephyr default, got %s", gc.SpeechConfig.VoiceConfig.PrebuiltVoiceConfig.VoiceName)
	}
}

func TestBuildConfig_ResponseModalities(t *testing.T) {
	cfg := Config{Voice: "Kore", Text: "test"}
	gc := BuildConfig(cfg)
	if len(gc.ResponseModalities) != 1 || gc.ResponseModalities[0] != "AUDIO" {
		t.Errorf("expected [AUDIO], got %v", gc.ResponseModalities)
	}
}

func TestBuildConfig_LanguageCode(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"Korean", "ko-KR"},
		{"ko", "ko-KR"},
		{"한국어", "ko-KR"},
		{"English", "en-US"},
		{"en", "en-US"},
		{"ja", "ja-JP"},
		{"Japanese", "ja-JP"},
		{"", "ko-KR"},
	}
	for _, tt := range tests {
		cfg := Config{Language: tt.input, Text: "test"}
		gc := BuildConfig(cfg)
		if gc.SpeechConfig.LanguageCode != tt.expected {
			t.Errorf("NormalizeLanguageCode(%q) = %q, want %q", tt.input, gc.SpeechConfig.LanguageCode, tt.expected)
		}
	}
}

func TestNormalizeLanguageCode(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"ko", "ko-KR"},
		{"Korean", "ko-KR"},
		{"한국어", "ko-KR"},
		{"kr", "ko-KR"},
		{"en", "en-US"},
		{"English", "en-US"},
		{"eng", "en-US"},
		{"ja", "ja-JP"},
		{"Japanese", "ja-JP"},
		{"日本語", "ja-JP"},
		{"", "ko-KR"},
		{"  ", "ko-KR"},
		{"ja-JP", "ja-JP"},
	}
	for _, tt := range tests {
		got := NormalizeLanguageCode(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizeLanguageCode(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestNewClient_NilGuard(t *testing.T) {
	c := NewClient(nil)
	if c != nil {
		t.Error("expected nil client for nil genai input")
	}
}
