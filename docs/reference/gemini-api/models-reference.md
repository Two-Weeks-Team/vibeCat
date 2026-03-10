# Gemini Models Reference

**Source:** https://ai.google.dev/gemini-api/docs/models  
**Extracted:** 2026-03-10

---

## Model Overview

This document provides key information about Gemini models relevant to the Live API and VibeCat project.

---

## Gemini 3 Flash Preview

**Model Code:** `gemini-3-flash-preview`

### Supported Data Types

**Inputs:**
- Text
- Image
- Video
- Audio
- PDF

**Output:**
- Text

### Token Limits

| Limit | Value |
|-------|-------|
| Input token limit | 1,048,576 |
| Output token limit | 65,536 |

### Capabilities

| Feature | Status |
|---------|--------|
| Audio generation | ❌ Not supported |
| Batch API | ✅ Supported |
| Caching | ✅ Supported |
| Code execution | ✅ Supported |
| Computer use | ✅ Supported |
| File search | ✅ Supported |
| Function calling | ✅ Supported |
| Grounding with Google Maps | ❌ Not supported |
| Image generation | ❌ Not supported |
| **Live API** | ❌ **Not supported** |
| Search grounding | ✅ Supported |
| Structured outputs | ✅ Supported |
| Thinking | ✅ Supported |
| URL context | ✅ Supported |

### Version Information

- **Preview:** `gemini-3-flash-preview`
- **Latest update:** December 2025
- **Knowledge cutoff:** January 2025

---

## Live API Models

Based on documentation, the Live API uses a specific model designed for real-time audio and video:

### Recommended Model for Live API

**Model:** `gemini-2.5-flash-native-audio-preview-12-2025`

This model is specifically designed for:
- Real-time audio streaming
- Native audio output (text-to-speech)
- Video input (up to 1 FPS)
- Multimodal interactions

### Live API Capabilities

| Feature | Status |
|---------|--------|
| Native audio output | ✅ Supported |
| Real-time audio input | ✅ Supported |
| Video input (≤1 FPS) | ✅ Supported |
| Text input/output | ✅ Supported |
| Function calling | ✅ Supported |
| WebSocket streaming | ✅ Supported |
| Barge-in/interruptions | ✅ Supported |
| Tool use | ✅ Supported |

---

## Audio Format Specifications

### Input Audio (to Live API)

| Property | Value |
|----------|-------|
| Format | Raw PCM |
| Sample Rate | 16 kHz |
| Bit Depth | 16-bit |
| Endianness | Little-endian |
| Channels | Mono |
| MIME Type | `audio/pcm;rate=16000` |

### Output Audio (from Live API)

| Property | Value |
|----------|-------|
| Format | Raw PCM |
| Sample Rate | 24 kHz |
| Bit Depth | 16-bit |
| Endianness | Little-endian |
| Channels | Mono |

---

## Video/Image Specifications

| Property | Value |
|----------|-------|
| Format | JPEG |
| Maximum Rate | 1 FPS (frames per second) |
| MIME Type | `image/jpeg` |

---

## Model Selection Guide

### For Live API (Real-time Voice/Video)

Use: `gemini-2.5-flash-native-audio-preview-12-2025`

**Use cases:**
- Voice agents
- Real-time transcription
- Video understanding
- Multimodal conversations

### For Batch Audio Processing

Use: `gemini-3-flash-preview`

**Use cases:**
- Audio transcription
- Audio analysis
- Speech-to-text (non-real-time)
- Audio summarization

---

## Important Notes

1. **Live API Model**: The standard Gemini 3 Flash Preview model does NOT support Live API. You must use the specific native audio model for real-time streaming.

2. **Audio Models**: For dedicated speech-to-text with real-time transcription, use the [Google Cloud Speech-to-Text API](https://cloud.google.com/speech-to-text) instead.

3. **Token Calculation**: Gemini represents each second of audio as 32 tokens (e.g., 1 minute = 1,920 tokens).

4. **Context Window**: The Live API model supports long-running sessions but has different limits than the standard API models.

---

## References

- [Gemini 3 Developer Guide](https://ai.google.dev/gemini-api/docs/gemini-3)
- [Live API Overview](https://ai.google.dev/gemini-api/docs/live-api)
- [Audio Understanding](https://ai.google.dev/gemini-api/docs/audio)
- [Token Counting](https://ai.google.dev/gemini-api/docs/tokens)

---

*Last updated: 2026-03-10*
