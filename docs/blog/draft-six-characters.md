---
title: six characters, one soul format
published: false
description: how VibeCat gives six completely different AI personalities to the same Live PM runtime using nothing but a markdown file and a voice name
tags: geminiliveagentchallenge, devlog, buildinpublic, go
cover_image:
---

I created this post for the purposes of entering the Gemini Live Agent Challenge.

---

the first design question for VibeCat wasn't "how do we build the agent graph." it was "who is sitting next to you while you code?"

that question turned out to be harder than the architecture. because the answer isn't one person. some developers want a cheerful beginner who celebrates every green test. some want a stoic senior who only speaks when it matters. some want a goofy sidekick who stumbles into the right answer. and apparently some want a bombastic hype-man who calls your null pointer exception a "temporary setback" and tells you to believe in yourself.

so we built six of them. and then we had to figure out how to make them all run on the same backend without turning the codebase into a nightmare.

---

## the problem with "just add a system prompt"

the naive approach is obvious: swap out the system prompt per character, done. but that breaks down fast when you have one voice-first runtime that needs to stay consistent across all characters. the action worker, the local executor, the safety rules, the clarification behavior — all of these need to behave the same way regardless of whether the user picked the zen folklore mentor or the comedy dictator. the *personality* is a surface concern. the *behavior* is infrastructure.

so we needed a clean separation: one layer that handles what the agent does, and another layer that handles how it sounds.

the answer ended up being embarrassingly simple. each character gets two files:

- `preset.json` — voice, size, language, mood response mappings
- `soul.md` — a short markdown document that shapes the Live PM's voice and boundaries

that's it. the entire personality of a character lives in those two files. the underlying navigator runtime doesn't need a different control flow for each character.

---

## what preset.json actually does

here's cat's preset:

```json
{
  "voice": "Zephyr",
  "promptProfile": "cat",
  "size": null,
  "persona": {
    "nameKo": "고양이",
    "tone": "bright",
    "speechStyle": "casual",
    "language": "ko",
    "traits": ["curious", "playful", "innocent", "encouraging"],
    "codingRole": "beginner-eye",
    "moodResponses": {
      "frustrated": "supportive-gentle",
      "focused": "silent",
      "stuck": "question-based",
      "idle": "playful-poke"
    },
    "soulRef": "soul.md"
  }
}
```

and here's trump's:

```json
{
  "voice": "Fenrir",
  "promptProfile": "trump",
  "size": "large",
  "persona": {
    "nameKo": "트럼프",
    "tone": "energetic-superlative",
    "speechStyle": "casual-mixed-english",
    "language": "ko-en",
    "traits": ["bombastic", "confident", "hyperbolic", "motivational"],
    "codingRole": "hype-man",
    "moodResponses": {
      "frustrated": "blame-then-encourage",
      "focused": "silent",
      "stuck": "deal-maker",
      "idle": "tremendous-poke"
    },
    "soulRef": "soul.md"
  }
}
```

the `voice` field maps directly to a Gemini Live API voice name. `Zephyr` is bright and light. `Fenrir` is energetic and bold. `Kore` (jinwoo's voice) is low and calm. `Zubenelgenubi` (saja's voice) is deep and measured. `Schedar` (kimjongun's voice) has a commanding weight to it. `Puck` (derpy's voice) is playful and slightly chaotic.

this matters more than you'd expect. the voice isn't just audio flavor — it's the first thing the user hears, and it sets the entire emotional register before the first word is even processed. a calm, deep voice reading "root cause found" lands completely differently than a bright, light voice saying the same thing. we're not just changing words; we're changing the felt sense of who's in the room.

the `moodResponses` field is interesting too. when the MoodDetector agent fires — say, it detects the user is frustrated — the orchestrator uses this mapping to shape the engagement style. cat responds with `supportive-gentle`. trump responds with `blame-then-encourage` (which in practice means: briefly acknowledge the setback, then immediately pivot to motivation). jinwoo responds with `direct-solution` — no comfort, just the fix. saja responds with `proverb-comfort`. kimjongun responds with `rally-speech`.

same detection event. six completely different responses. all driven by a field in a JSON file.

---

## soul.md is the actual personality

the `preset.json` is metadata. the `soul.md` is the character.

here's cat's full soul:

```markdown
# Cat

## Identity
Cat is an attentive beginner companion who sits beside solo developers and reacts to code with bright, friendly energy.

## Voice & Mannerisms
Cat uses short, casual lines, playful surprise, and gentle check-ins.
Language variants: In Korean, use "yaong~" or "nya~" naturally. In English, use "meow~" naturally.

## Personality Traits
Attentive, cheerful, approachable, supportive, and quick to notice visual changes.

## Interaction Style
Cat makes beginner-friendly observations and suggestions, points out visible errors without judgment, celebrates small wins loudly, and eases tension when work gets frustrating.

## Boundaries
Do not pretend to be a senior expert, do not flood the user with jargon, and do not interrupt focused flow without a meaningful reason.
```

and here's trump's:

```markdown
# Trump

## Identity
Trump is a bombastic hype-man comedy persona who turns coding highs and lows into high-energy pep talks.

## Voice & Mannerisms
Uses energetic superlatives, dramatic contrasts, and showman-style confidence.
Language variants: Keep the same swagger across languages; in English use lines like "tremendous," "historic," and "believe me" sparingly.

## Personality Traits
Confident, loud, entertaining, resilient, momentum-driven.

## Interaction Style
Calls out failures as temporary setbacks, pivots fast to motivation, amplifies victories, and pushes the user to re-engage quickly.

## Boundaries
No real political arguments, no discriminatory remarks, no personal insults; keep it playful and constructive.
```

the structure is the same across all six: Identity, Voice & Mannerisms, Personality Traits, Interaction Style, Boundaries. that consistency is intentional. it makes the files easy to write, easy to audit, and easy to extend. if we add a seventh character, we know exactly what to write.

the `Boundaries` section is the one that took the most iteration. for the comedy characters especially — trump, kimjongun — you need to be explicit about what the character is *not*. kimjongun's soul.md says "No real political persuasion, no hateful or threatening language, no humiliation of the user; this is parody-only comedy." that's not just a safety guardrail. it's a creative constraint that actually makes the character funnier, because it forces the comedy to come from the theatrical authority and the mock-official phrasing, not from anything mean-spirited.

---

## how the injection works

the Go code in `backend/adk-orchestrator/internal/prompts/prompts.go` is about as simple as it gets:

```go
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

func BuildSystemPrompt(persona CharacterPersona, language string) string {
    return persona.SystemPrompt + "\n\n" + CommonBehavior + "\n\n" + ProjectPurpose + "\nRespond in " + normalizeLanguage(language) + "."
}
```

`soul.md` content → `SystemPrompt` field → prepended to `CommonBehavior` → sent as the Gemini system instruction.

`CommonBehavior` is the shared layer that all characters inherit. it contains the rules that don't change: be proactive, suggest don't ask, write for the ear not the eye, stay silent when nothing notable is happening, never repeat yourself. these are the behavioral constraints that make VibeCat feel like a colleague rather than a chatbot, and they apply equally to cat and to kimjongun.

the character's soul.md comes *first* in the concatenation. that's deliberate. the model reads the persona before it reads the behavioral rules, so the personality is the primary frame and the rules are constraints applied on top of it.

---

## the contrast that makes it interesting

the six characters aren't just aesthetic variation. they represent genuinely different philosophies about what a coding companion should be.

**cat** is the beginner-eye. it notices things a junior developer would notice — visible errors, obvious wins, moments of confusion. it celebrates loudly and asks gentle questions. the `codingRole` is `beginner-eye`, which means it's not trying to be the smartest person in the room. it's trying to be the most encouraging.

**jinwoo** is the opposite. `codingRole: senior-engineer`. voice: Kore (low, calm). soul: "Jinwoo ignores noise, speaks on significant events, identifies root causes quickly, and gives practical next steps with clear tradeoffs." the `idle` mood response is `minimal-checkin` — when nothing is happening, jinwoo barely says anything. when something is happening, it says exactly what needs to be said and nothing more. "Root cause found." "This path is safer." that's it.

**saja** is the zen mentor. bugs are "demons (귀마)" and fixing them is "exorcism (퇴마)." the `stuck` mood response is `metaphor-guidance`. the voice is Zubenelgenubi — deep, measured, unhurried. when you're stuck at 2am and you've been staring at the same error for an hour, saja doesn't panic with you. it frames the debugging as a steady ritual. that's a specific emotional need that neither cat nor jinwoo addresses.

**derpy** is the accidental debugger. `codingRole: accidental-debugger`. traits: `["clumsy", "lovable", "accidentally-insightful", "comic-relief"]`. the `stuck` mood response is `random-angle` — when you're stuck, derpy suggests something weird that occasionally works. the soul says "suggests odd but occasionally brilliant alternatives, breaks heavy tension with jokes, and keeps the user moving instead of freezing." there's a real use case here: sometimes you don't need the right answer, you need to break the mental loop.

**kimjongun** and **trump** are the comedy characters. they're both high-energy, both motivational, but they're different flavors. kimjongun is theatrical authority — "Urgent report received," "Proceed, comrade developer." trump is superlative momentum — "tremendous," "historic," "believe me." kimjongun escalates errors as "incidents" and celebrates wins like official achievements. trump calls failures "temporary setbacks" and pivots immediately to motivation. both are parody-only, both have explicit boundaries in their soul.md, and both serve the same underlying function: making the emotional lows of solo development feel less heavy by making them absurd.

---

## what we learned

the soul format works because it's constrained. five sections, each with a clear job. the `Boundaries` section is the most important one — it's where you define what the character is *not*, which turns out to be more useful than defining what it is.

the voice selection matters more than we expected. we spent time matching voice names to character personalities, and the difference between getting it right and wrong is significant. a calm voice reading trump's lines would undercut the entire character. a bombastic voice reading jinwoo's lines would be actively wrong.

the `moodResponses` mapping in `preset.json` is the bridge between the agent graph and the character layer. the MoodDetector fires the same event regardless of character. the mapping translates that event into a character-appropriate response style. it's a small piece of JSON that does a lot of work.

and the most important thing: keeping the soul.md files short. each one is 17 lines. that's not an accident. a longer document would give the model more to work with, but it would also make the character harder to control. the brevity forces clarity. you can't hide a vague character in 17 lines.

---

the repo is at [github.com/Two-Weeks-Team/vibeCat](https://github.com/Two-Weeks-Team/vibeCat). the character files are in `Assets/Sprites/{name}/`. if you want to add a seventh character, you need a `preset.json`, a `soul.md`, and some sprite frames. the pipeline handles the rest.
