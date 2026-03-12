package live

import (
	"strings"
	"testing"

	"google.golang.org/genai"
)

func TestParseSetup(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid json",
			input:   `{"type":"setup","config":{"voice":"Zephyr","liveModel":"gemini-2.5-flash-native-audio-preview-12-2025"}}`,
			wantErr: false,
		},
		{
			name:    "invalid json",
			input:   `{`,
			wantErr: true,
		},
		{
			name:    "wrong type",
			input:   `{"type":"ping","config":{"voice":"Zephyr"}}`,
			wantErr: true,
		},
		{
			name:    "missing fields",
			input:   `{"config":{"voice":"Zephyr"}}`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg, err := ParseSetup([]byte(tc.input))
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseSetup() error = %v", err)
			}
			if msg == nil {
				t.Fatal("expected setup message, got nil")
			}
			if msg.Type != "setup" {
				t.Fatalf("Type = %q, want setup", msg.Type)
			}
		})
	}
}

func TestBuildLiveConfig(t *testing.T) {
	tests := []struct {
		name  string
		cfg   Config
		check func(t *testing.T, lc *genai.LiveConnectConfig)
	}{
		{
			name: "with voice",
			cfg:  Config{Voice: "Zephyr"},
			check: func(t *testing.T, lc *genai.LiveConnectConfig) {
				t.Helper()
				if lc.SpeechConfig == nil || lc.SpeechConfig.VoiceConfig == nil || lc.SpeechConfig.VoiceConfig.PrebuiltVoiceConfig == nil {
					t.Fatal("expected speech config with prebuilt voice")
				}
				if got := lc.SpeechConfig.VoiceConfig.PrebuiltVoiceConfig.VoiceName; got != "Zephyr" {
					t.Fatalf("VoiceName = %q, want Zephyr", got)
				}
			},
		},
		{
			name: "without voice",
			cfg:  Config{},
			check: func(t *testing.T, lc *genai.LiveConnectConfig) {
				t.Helper()
				if lc.SpeechConfig != nil {
					t.Fatalf("SpeechConfig = %#v, want nil", lc.SpeechConfig)
				}
			},
		},
		{
			name: "with vad settings",
			cfg:  Config{},
			check: func(t *testing.T, lc *genai.LiveConnectConfig) {
				t.Helper()
				if lc.RealtimeInputConfig == nil || lc.RealtimeInputConfig.AutomaticActivityDetection == nil {
					t.Fatal("expected automatic activity detection config")
				}
				ad := lc.RealtimeInputConfig.AutomaticActivityDetection
				if ad.StartOfSpeechSensitivity != genai.StartSensitivityHigh {
					t.Fatalf("StartOfSpeechSensitivity = %v, want %v", ad.StartOfSpeechSensitivity, genai.StartSensitivityHigh)
				}
				if ad.EndOfSpeechSensitivity != genai.EndSensitivityLow {
					t.Fatalf("EndOfSpeechSensitivity = %v, want %v", ad.EndOfSpeechSensitivity, genai.EndSensitivityLow)
				}
				if ad.PrefixPaddingMs == nil || *ad.PrefixPaddingMs != 20 {
					t.Fatalf("PrefixPaddingMs = %v, want 20", ad.PrefixPaddingMs)
				}
				if ad.SilenceDurationMs == nil || *ad.SilenceDurationMs != 200 {
					t.Fatalf("SilenceDurationMs = %v, want 200", ad.SilenceDurationMs)
				}
			},
		},
		{
			name: "with system prompt",
			cfg:  Config{Soul: "you are helpful", Language: "en"},
			check: func(t *testing.T, lc *genai.LiveConnectConfig) {
				t.Helper()
				if lc.SystemInstruction == nil || len(lc.SystemInstruction.Parts) != 1 {
					t.Fatal("expected system instruction with one part")
				}
				got := lc.SystemInstruction.Parts[0].Text
				if !strings.Contains(got, commonLivePrompt) {
					t.Fatal("system prompt should include common live prompt")
				}
				if !strings.Contains(got, "=== CHARACTER PERSONA ===\nyou are helpful") {
					t.Fatal("system prompt should include soul content section")
				}
				if !strings.Contains(got, "Respond in English.") {
					t.Fatal("system prompt should include normalized language directive")
				}
				if !strings.Contains(got, "Keep responses to 1-2 short sentences") {
					t.Fatal("system prompt should include concise speech guidance")
				}
			},
		},
		{
			name: "with affective dialog",
			cfg:  Config{AffectiveDialog: true},
			check: func(t *testing.T, lc *genai.LiveConnectConfig) {
				t.Helper()
				if lc.EnableAffectiveDialog == nil || !*lc.EnableAffectiveDialog {
					t.Fatal("expected affective dialog enabled")
				}
			},
		},
		{
			name: "with proactive audio",
			cfg:  Config{ProactiveAudio: true},
			check: func(t *testing.T, lc *genai.LiveConnectConfig) {
				t.Helper()
				if lc.Proactivity == nil || lc.Proactivity.ProactiveAudio == nil || !*lc.Proactivity.ProactiveAudio {
					t.Fatal("expected proactive audio enabled")
				}
			},
		},
		{
			name: "google search tool enabled for live api",
			cfg:  Config{GoogleSearch: true},
			check: func(t *testing.T, lc *genai.LiveConnectConfig) {
				t.Helper()
				if len(lc.Tools) != 2 {
					t.Fatalf("expected navigator + search tools, got %d", len(lc.Tools))
				}
				hasNavigator := false
				hasSearch := false
				for _, tool := range lc.Tools {
					if tool.FunctionDeclarations != nil {
						hasNavigator = true
					}
					if tool.GoogleSearch != nil {
						hasSearch = true
					}
				}
				if !hasNavigator {
					t.Fatal("expected navigator function declarations tool")
				}
				if !hasSearch {
					t.Fatal("expected Google Search live tool")
				}
			},
		},
		{
			name: "navigator tools always present",
			cfg:  Config{},
			check: func(t *testing.T, lc *genai.LiveConnectConfig) {
				t.Helper()
				if len(lc.Tools) != 1 {
					t.Fatalf("expected navigator tool only, got %d", len(lc.Tools))
				}
				if lc.Tools[0].FunctionDeclarations == nil || len(lc.Tools[0].FunctionDeclarations) == 0 {
					t.Fatal("expected navigator function declarations")
				}
				if lc.Tools[0].FunctionDeclarations[0].Name != "navigate_text_entry" {
					t.Fatalf("expected navigate_text_entry, got %q", lc.Tools[0].FunctionDeclarations[0].Name)
				}
			},
		},
		{
			name: "memory context included in system prompt",
			cfg:  Config{Language: "ko", MemoryContext: "Previous session: websocket search routing fix was in progress."},
			check: func(t *testing.T, lc *genai.LiveConnectConfig) {
				t.Helper()
				if lc.SystemInstruction == nil || len(lc.SystemInstruction.Parts) != 1 {
					t.Fatal("expected system instruction")
				}
				got := lc.SystemInstruction.Parts[0].Text
				if !strings.Contains(got, "=== RECENT ESSENTIAL CONTEXT ===") {
					t.Fatal("expected recent essential context section")
				}
				if !strings.Contains(got, "websocket search routing fix was in progress") {
					t.Fatal("expected memory context content")
				}
			},
		},
		{
			name: "google search guidance included in system prompt",
			cfg:  Config{Language: "ko", GoogleSearch: true},
			check: func(t *testing.T, lc *genai.LiveConnectConfig) {
				t.Helper()
				got := lc.SystemInstruction.Parts[0].Text
				if !strings.Contains(got, "For grounded search answers, give the direct answer") {
					t.Fatal("expected grounded search brevity guidance")
				}
			},
		},
		{
			name: "context compression tuned for realtime sessions",
			cfg:  Config{},
			check: func(t *testing.T, lc *genai.LiveConnectConfig) {
				t.Helper()
				if lc.ContextWindowCompression == nil || lc.ContextWindowCompression.TriggerTokens == nil {
					t.Fatal("expected context window compression config")
				}
				if *lc.ContextWindowCompression.TriggerTokens != 12000 {
					t.Fatalf("TriggerTokens = %d, want 12000", *lc.ContextWindowCompression.TriggerTokens)
				}
				if lc.ContextWindowCompression.SlidingWindow == nil || lc.ContextWindowCompression.SlidingWindow.TargetTokens == nil {
					t.Fatal("expected sliding window target tokens")
				}
				if *lc.ContextWindowCompression.SlidingWindow.TargetTokens != 6000 {
					t.Fatalf("TargetTokens = %d, want 6000", *lc.ContextWindowCompression.SlidingWindow.TargetTokens)
				}
			},
		},
		{
			name: "quiet chattiness tightens response length guidance",
			cfg:  Config{Language: "ko", Chattiness: "quiet"},
			check: func(t *testing.T, lc *genai.LiveConnectConfig) {
				t.Helper()
				got := lc.SystemInstruction.Parts[0].Text
				if !strings.Contains(got, "exactly one short spoken sentence") {
					t.Fatal("expected quiet chattiness guidance")
				}
			},
		},
		{
			name: "chatty chattiness allows two sentences",
			cfg:  Config{Language: "ko", Chattiness: "chatty"},
			check: func(t *testing.T, lc *genai.LiveConnectConfig) {
				t.Helper()
				got := lc.SystemInstruction.Parts[0].Text
				if !strings.Contains(got, "up to two short spoken sentences") {
					t.Fatal("expected chatty chattiness guidance")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lc := buildLiveConfig(tc.cfg)
			if lc == nil {
				t.Fatal("buildLiveConfig() returned nil")
			}

			if lc.OutputAudioTranscription == nil {
				t.Fatal("expected OutputAudioTranscription to be always set")
			}

			tc.check(t, lc)
		})
	}
}

func TestTuningProfilesMatchPlan(t *testing.T) {
	if baselineTuningProfile.Name != "baseline" {
		t.Fatalf("baseline name = %q", baselineTuningProfile.Name)
	}
	if memoryLightTuningProfile.TriggerTokens != 10000 || memoryLightTuningProfile.TargetTokens != 5000 || memoryLightTuningProfile.MaxMemoryChars != 900 {
		t.Fatalf("memory_light profile mismatch: %#v", memoryLightTuningProfile)
	}
	if vadRelaxedTuningProfile.PrefixPaddingMs != 40 || vadRelaxedTuningProfile.SilenceDurationMs != 250 {
		t.Fatalf("vad_relaxed profile mismatch: %#v", vadRelaxedTuningProfile)
	}
}
