# Asset Migration Plan

This document defines which local assets are required in VibeCat and where they are stored.

## Required Asset Groups

- `Assets/Sprites/`: character sprite frames used by overlay animation
- `Assets/TrayIcons_Clean/`: menu/tray animated icon frames
- `Assets/Music/`: background music resources
- `Assets/SPRITE_LICENSE.md`: sprite attribution and usage constraints
- `voice_samples/`: local voice sample assets for voice-related validation

## Copy Policy

1. Copy assets exactly as local files (no remote download at runtime).
2. Preserve directory names and frame naming for runtime compatibility.
3. Keep license file with assets.
4. Record counts after copy for integrity checks.

## Inventory Template

Current local inventory:

| Path | Type | Count Rule | Current Count |
|---|---|---|
| `Assets/Sprites/` | PNG sprite frames | count by character/state/frame | 97 PNG files |
| `Assets/TrayIcons_Clean/` | PNG tray icon frames | 8 frames x 3 scales | 24 PNG files |
| `Assets/Music/` | audio files | count all runtime-used tracks | 2 audio files |
| `Assets/SPRITE_LICENSE.md` | text | must exist exactly once | 1 file |
| `voice_samples/` | audio samples | count all local voice sample files | 13 audio files |

## Verification

- `Assets/` exists in VibeCat root.
- All required asset groups exist.
- File count is non-zero for each required group.
- Asset paths match runtime configuration assumptions.
