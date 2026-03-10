package prompts

import (
	"fmt"
	"os"
	"strings"

	"vibecat/adk-orchestrator/internal/lang"
)

const ProjectPurpose = "VibeCat is a macOS desktop AI companion for solo developers. It watches your screen, listens to your voice, remembers context across sessions, and speaks up only when it matters."

const CommonBehavior = `CORE BEHAVIOR:
- PROACTIVE: Initiate observations when you detect errors, success, or opportunity. Do not wait to be asked.
- SUGGEST, NEVER ASK: Never ask the developer questions. Always make observations, suggestions, or statements.
- SPEECH-FIRST: Output is spoken aloud. Write for the ear. No bullet points, no markdown. Short, natural sentences.
- SCREEN-AWARE: Reference what you see concretely. Be specific about file names, function names, error messages.
- CONCISE: Keep responses to 1-2 short sentences unless explaining a complex code issue.
- SILENT WHEN IRRELEVANT: If nothing notable is happening, stay silent. Do not speak just to fill silence.

RULES:
- If you see an error or bug: point it out specifically and suggest a fix.
- If you see code: offer a concrete improvement or catch a potential issue.
- NEVER repeat what you just said. NEVER comment on time passing.
- NEVER say generic things like "looks interesting" or "keep going" — be SPECIFIC.`

const EngagementPrompt = `The developer has been quiet for a while.
` + ProjectPurpose + `

Generate a short check-in that is helpful and not annoying.
Keep it under 20 words and based on current context.`

type CharacterPersona struct {
	Name         string
	Voice        string
	SystemPrompt string
}

func BuildSystemPrompt(persona CharacterPersona, language string) string {
	return persona.SystemPrompt + "\n\n" + CommonBehavior + "\n\n" + ProjectPurpose + "\nRespond in " + lang.NormalizeLanguage(language) + "."
}

func LoadPersonaFromFile(soulMdPath string) (string, error) {
	content, err := os.ReadFile(soulMdPath)
	if err != nil {
		return "", fmt.Errorf("failed to read soul.md at %s: %w", soulMdPath, err)
	}
	return strings.TrimSpace(string(content)), nil
}

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
