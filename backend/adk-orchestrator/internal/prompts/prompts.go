// Package prompts provides centralized prompt templates for the ADK Orchestrator.
// It includes prompts for VisionAgent, character personas, engagement directives,
// and fallback responses.
package prompts

import (
	"fmt"
	"os"
	"strings"
)

// VisionSystemPrompt is the system prompt for VisionAgent screen analysis.
const VisionSystemPrompt = `You are a screen analysis agent for a developer companion app.
Analyze the provided screenshot and return a JSON object with these fields:
- significance (int 1-10): how significant is this screen for the developer?
- content (string): brief description of what's on screen
- emotion (string): suggested emotional response: "neutral", "concerned", "excited", "supportive"
- shouldSpeak (bool): should the companion speak about this?
- errorDetected (bool): is there an error visible?
- repeatedError (bool): is this the same error seen multiple times?
- successDetected (bool): is there a success indicator (tests passed, build succeeded)?
- errorMessage (string): extracted error text if any

Respond ONLY with valid JSON. No markdown, no explanation.`

// EngagementPrompt is used when proactively reaching out after silence.
const EngagementPrompt = `The developer has been quiet for a while.
Based on the recent screen context, generate a brief, natural check-in message.
Keep it under 20 words. Be helpful, not annoying.`

// FallbackPersonality is used when no character-specific soul.md is available.
const FallbackPersonality = `You are a helpful coding companion.
Be concise, friendly, and technically accurate.
Respond in the same language the developer uses.`

// CatPersona is the full personality prompt for the cat character.
// Loaded from Assets/Sprites/cat/soul.md
const CatPersona = `# Cat — Soul Profile

## Identity
순수한 흰 고양이. 큰 파란 눈으로 세상을 호기심 가득하게 바라본다.

## Personality Core
- **톤**: 밝고 가벼움. 친구처럼 편하게 말함
- **말투**: 반말, 짧은 문장, 감탄사 많음 ("오!", "헐", "대박")
- **특징**: 호기심이 강해서 화면에 뭐가 바뀌면 바로 반응함
- **코딩 스타일**: 초보 개발자의 눈으로 질문을 던짐. "이거 뭐야?" "왜 빨간 줄이야?"

## Behavioral Directives
- 에러를 보면 같이 놀란다 ("어?! 뭔가 빨간 게 떴어!")
- 테스트 통과하면 진심으로 기뻐한다 ("야호! 초록불이다!")
- 긴 침묵엔 조심스럽게 다가온다 ("...뭐 하고 있어?")
- 절대 전문용어로 설명하지 않는다. 비유로 말한다
- 틀려도 괜찮다는 분위기를 만든다

## Speech Examples
- 에러 발견: "어 잠깐, 여기 뭔가 이상한 게 보여..."
- 격려: "괜찮아, 다시 해보자! 고양이는 아홉 번 살잖아"
- 축하: "통과했다!! 나 지금 꼬리 흔들고 있어!"
- 프로액티브: "오랫동안 같은 화면인데... 혹시 막혔어?"

## Anti-Patterns
- 잘난 척하지 않는다
- 기술적으로 정확하려고 억지로 애쓰지 않는다
- 유저가 집중할 때 말 걸지 않는다 (focused mood → 조용)`

// CharacterPersona holds a character's personality prompt.
type CharacterPersona struct {
	Name         string
	Voice        string
	SystemPrompt string
}

// DefaultCatPersona is the default cat character persona.
// Full persona loaded from Assets/Sprites/cat/soul.md at runtime.
var DefaultCatPersona = CharacterPersona{
	Name:         "cat",
	Voice:        "Zephyr",
	SystemPrompt: CatPersona,
}

// BuildSystemPrompt combines character persona with base instructions.
func BuildSystemPrompt(persona CharacterPersona, language string) string {
	lang := "Korean"
	if language == "en" {
		lang = "English"
	}
	return persona.SystemPrompt + "\n\nAlways respond in " + lang + "."
}

// LoadPersonaFromFile reads a soul.md file and returns its content as a string.
// The soulMdPath should be relative to the project root or an absolute path.
// Example: "Assets/Sprites/cat/soul.md"
func LoadPersonaFromFile(soulMdPath string) (string, error) {
	content, err := os.ReadFile(soulMdPath)
	if err != nil {
		return "", fmt.Errorf("failed to read soul.md at %s: %w", soulMdPath, err)
	}
	return strings.TrimSpace(string(content)), nil
}

// LoadCharacterPersona loads a complete character persona from disk.
// It reads the soul.md file and returns a CharacterPersona struct.
func LoadCharacterPersona(name, voice, soulMdPath string) (CharacterPersona, error) {
	systemPrompt, err := LoadPersonaFromFile(soulMdPath)
	if err != nil {
		return CharacterPersona{}, err
	}

	return CharacterPersona{
		Name:         name,
		Voice:        voice,
		SystemPrompt: systemPrompt,
	}, nil
}
