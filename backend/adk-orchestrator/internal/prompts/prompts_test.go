package prompts

import (
	"strings"
	"testing"
)

func TestBuildSystemPrompt_IncludesLanguageDirective(t *testing.T) {
	tests := []struct {
		name     string
		language string
		want     string
	}{
		{
			name:     "Korean language",
			language: "ko",
			want:     "Always respond in Korean",
		},
		{
			name:     "English language",
			language: "en",
			want:     "Always respond in English",
		},
		{
			name:     "Empty defaults to Korean",
			language: "",
			want:     "Always respond in Korean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			persona := CharacterPersona{
				Name:         "test",
				Voice:        "Zephyr",
				SystemPrompt: "Test persona prompt.",
			}

			result := BuildSystemPrompt(persona, tt.language)

			if !strings.Contains(result, tt.want) {
				t.Errorf("BuildSystemPrompt() = %q, want it to contain %q", result, tt.want)
			}
		})
	}
}

func TestBuildSystemPrompt_AppendsToPersona(t *testing.T) {
	persona := CharacterPersona{
		Name:         "cat",
		Voice:        "Zephyr",
		SystemPrompt: "Original persona content.",
	}

	result := BuildSystemPrompt(persona, "en")

	expectedPrefix := "Original persona content."
	if !strings.HasPrefix(result, expectedPrefix) {
		t.Errorf("BuildSystemPrompt() should start with persona content, got %q", result)
	}

	if !strings.Contains(result, "Always respond in English") {
		t.Error("BuildSystemPrompt() should contain language directive")
	}
}

func TestDefaultCatPersona(t *testing.T) {
	if DefaultCatPersona.Name != "cat" {
		t.Errorf("DefaultCatPersona.Name = %q, want %q", DefaultCatPersona.Name, "cat")
	}

	if DefaultCatPersona.Voice != "Zephyr" {
		t.Errorf("DefaultCatPersona.Voice = %q, want %q", DefaultCatPersona.Voice, "Zephyr")
	}

	if !strings.Contains(DefaultCatPersona.SystemPrompt, "Cat — Soul Profile") {
		t.Error("DefaultCatPersona.SystemPrompt should contain Cat soul profile")
	}

	if !strings.Contains(DefaultCatPersona.SystemPrompt, "Identity") {
		t.Error("DefaultCatPersona.SystemPrompt should contain Identity section")
	}
}

func TestLoadPersonaFromFile_Success(t *testing.T) {
	// Use the actual soul.md file from the project
	content, err := LoadPersonaFromFile("../../../../Assets/Sprites/cat/soul.md")
	if err != nil {
		t.Fatalf("LoadPersonaFromFile() error = %v", err)
	}

	if !strings.Contains(content, "Cat — Soul Profile") {
		t.Error("Loaded content should contain Cat soul profile header")
	}

	if !strings.Contains(content, "Identity") {
		t.Error("Loaded content should contain Identity section")
	}
}

func TestLoadPersonaFromFile_FileNotFound(t *testing.T) {
	_, err := LoadPersonaFromFile("/nonexistent/path/soul.md")
	if err == nil {
		t.Error("LoadPersonaFromFile() should return error for non-existent file")
	}
}

func TestLoadCharacterPersona(t *testing.T) {
	// Use the actual soul.md file
	persona, err := LoadCharacterPersona("cat", "Zephyr", "../../../../Assets/Sprites/cat/soul.md")
	if err != nil {
		t.Fatalf("LoadCharacterPersona() error = %v", err)
	}

	if persona.Name != "cat" {
		t.Errorf("persona.Name = %q, want %q", persona.Name, "cat")
	}

	if persona.Voice != "Zephyr" {
		t.Errorf("persona.Voice = %q, want %q", persona.Voice, "Zephyr")
	}

	if !strings.Contains(persona.SystemPrompt, "Cat — Soul Profile") {
		t.Error("persona.SystemPrompt should contain Cat soul profile")
	}
}

func TestVisionSystemPrompt(t *testing.T) {
	if !strings.Contains(VisionSystemPrompt, "screen analysis agent") {
		t.Error("VisionSystemPrompt should mention screen analysis")
	}

	if !strings.Contains(VisionSystemPrompt, "JSON") {
		t.Error("VisionSystemPrompt should mention JSON")
	}
}

func TestEngagementPrompt(t *testing.T) {
	if !strings.Contains(EngagementPrompt, "quiet for a while") {
		t.Error("EngagementPrompt should mention quiet period")
	}
}

func TestFallbackPersonality(t *testing.T) {
	if !strings.Contains(FallbackPersonality, "helpful coding companion") {
		t.Error("FallbackPersonality should mention helpful coding companion")
	}
}
