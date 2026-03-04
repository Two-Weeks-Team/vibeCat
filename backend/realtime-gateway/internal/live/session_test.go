package live

import (
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
			input:   `{"type":"setup","config":{"voice":"Zephyr","liveModel":"gemini-2.0-flash-live-001"}}`,
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
				if ad.StartOfSpeechSensitivity != genai.StartSensitivityLow {
					t.Fatalf("StartOfSpeechSensitivity = %v, want %v", ad.StartOfSpeechSensitivity, genai.StartSensitivityLow)
				}
				if ad.EndOfSpeechSensitivity != genai.EndSensitivityLow {
					t.Fatalf("EndOfSpeechSensitivity = %v, want %v", ad.EndOfSpeechSensitivity, genai.EndSensitivityLow)
				}
				if ad.PrefixPaddingMs == nil || *ad.PrefixPaddingMs != 20 {
					t.Fatalf("PrefixPaddingMs = %v, want 20", ad.PrefixPaddingMs)
				}
				if ad.SilenceDurationMs == nil || *ad.SilenceDurationMs != 100 {
					t.Fatalf("SilenceDurationMs = %v, want 100", ad.SilenceDurationMs)
				}
			},
		},
		{
			name: "with system prompt",
			cfg:  Config{SystemPrompt: "you are helpful"},
			check: func(t *testing.T, lc *genai.LiveConnectConfig) {
				t.Helper()
				if lc.SystemInstruction == nil || len(lc.SystemInstruction.Parts) != 1 {
					t.Fatal("expected system instruction with one part")
				}
				if got := lc.SystemInstruction.Parts[0].Text; got != "you are helpful" {
					t.Fatalf("system prompt = %q, want %q", got, "you are helpful")
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
