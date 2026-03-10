package tts

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/genai"

	"vibecat/realtime-gateway/internal/geminiconfig"
)

const defaultModel = geminiconfig.TextToSpeechModel
const defaultVoice = "Zephyr"
const ttsTimeout = 15 * time.Second

type AudioSink func(chunk []byte) error

type Config struct {
	Voice    string
	Language string
	Text     string
}

type Client struct {
	genai *genai.Client
	model string
}

func NewClient(genaiClient *genai.Client) *Client {
	if genaiClient == nil {
		return nil
	}
	return &Client{genai: genaiClient, model: defaultModel}
}

func (c *Client) StreamSpeak(ctx context.Context, cfg Config, sink AudioSink) error {
	ttsCtx, cancel := context.WithTimeout(ctx, ttsTimeout)
	defer cancel()

	genConfig := BuildConfig(cfg)
	text := cfg.Text
	if text == "" {
		return fmt.Errorf("tts: empty text")
	}

	start := time.Now()
	firstChunk := true
	totalBytes := 0

	for resp, err := range c.genai.Models.GenerateContentStream(ttsCtx, c.model, genai.Text(text), genConfig) {
		if err != nil {
			if ttsCtx.Err() != nil {
				slog.Info("[TTS] stream cancelled", "elapsed", time.Since(start).String(), "bytes", totalBytes)
				return nil
			}
			return fmt.Errorf("tts stream: %w", err)
		}

		if resp == nil || len(resp.Candidates) == 0 {
			continue
		}
		candidate := resp.Candidates[0]
		if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
			continue
		}
		for _, part := range candidate.Content.Parts {
			if part.InlineData == nil || len(part.InlineData.Data) == 0 {
				continue
			}
			if firstChunk {
				firstChunk = false
				slog.Info("[TTS] first chunk", "latency", time.Since(start).String(), "mime", part.InlineData.MIMEType, "voice", voiceOrDefault(cfg.Voice), "text_len", len(text))
			}
			totalBytes += len(part.InlineData.Data)
			if err := sink(part.InlineData.Data); err != nil {
				return fmt.Errorf("tts sink: %w", err)
			}
		}
	}

	slog.Info("[TTS] stream complete", "elapsed", time.Since(start).String(), "bytes", totalBytes, "voice", voiceOrDefault(cfg.Voice))
	return nil
}

func BuildConfig(cfg Config) *genai.GenerateContentConfig {
	voice := voiceOrDefault(cfg.Voice)
	return &genai.GenerateContentConfig{
		ResponseModalities: []string{"AUDIO"},
		SpeechConfig: &genai.SpeechConfig{
			VoiceConfig: &genai.VoiceConfig{
				PrebuiltVoiceConfig: &genai.PrebuiltVoiceConfig{
					VoiceName: voice,
				},
			},
			LanguageCode: NormalizeLanguageCode(cfg.Language),
		},
	}
}

func NormalizeLanguageCode(lang string) string {
	trimmed := strings.TrimSpace(lang)
	if trimmed == "" {
		return "ko-KR"
	}
	lower := strings.ToLower(trimmed)
	switch lower {
	case "ko", "kr", "korean", "korean language", "한국어":
		return "ko-KR"
	case "en", "eng", "english", "english language":
		return "en-US"
	default:
		return trimmed
	}
}

func voiceOrDefault(voice string) string {
	if voice == "" {
		return defaultVoice
	}
	return voice
}
