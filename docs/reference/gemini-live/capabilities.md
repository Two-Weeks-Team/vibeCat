# Live API Capabilities Guide

**Source:** https://ai.google.dev/gemini-api/docs/live-guide

**Preview:** The Live API is in preview.

This is a comprehensive guide that covers capabilities and configurations available with the Live API.

## Overview

The Live API enables low-latency, real-time voice and video interactions with Gemini. It processes continuous streams of audio, video, or text to deliver immediate, human-like spoken responses, creating a natural conversational experience for your users.

Live API offers a comprehensive set of features such as:
- Voice Activity Detection (VAD)
- Tool use and function calling
- Session management (for managing long-running conversations)
- Ephemeral tokens (for secure client-side authentication)

## Before You Begin

- **Familiarize yourself with core concepts:** If you haven't already done so, review the core concepts documentation.

## Implementation Approaches

When integrating with Live API, you can choose from the following implementation approaches:

### 1. Server-to-Server
Your backend connects to the Live API using WebSockets. Typically, your client sends stream data (audio, video, text) to your server, which then forwards it to the Live API.

### 2. Client-to-Server
Your frontend code connects directly to the Live API using WebSockets to stream data, bypassing your backend.

## Voice Activity Detection (VAD)

Voice Activity Detection (VAD) can be configured with several parameters to control how the API detects and responds to speech.

### VAD Configuration Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `disabled` | boolean | Set to `true` to disable automatic VAD |
| `start_of_speech_sensitivity` | enum | Controls detection threshold for speech start (LOW, MEDIUM, HIGH) |
| `end_of_speech_sensitivity` | enum | Controls detection threshold for speech end (LOW, MEDIUM, HIGH) |
| `prefix_padding_ms` | integer | Include audio before speech detection begins (in milliseconds) |
| `silence_duration_ms` | integer | How long silence must persist before API considers speech ended |

### Configure Automatic VAD (Python)

```python
from google.genai import types

config = {
    "response_modalities": ["TEXT"],
    "realtime_input_config": {
        "automatic_activity_detection": {
            "disabled": False,  # default
            "start_of_speech_sensitivity": types.StartSensitivity.START_SENSITIVITY_LOW,
            "end_of_speech_sensitivity": types.EndSensitivity.END_SENSITIVITY_LOW,
            "prefix_padding_ms": 20,
            "silence_duration_ms": 100,
        }
    }
}
```

### Configure Automatic VAD (JavaScript)

```javascript
import { GoogleGenAI, Modality, StartSensitivity, EndSensitivity } from '@google/genai';

const config = {
  responseModalities: [Modality.TEXT],
  realtimeInputConfig: {
    automaticActivityDetection: {
      disabled: false,  // default
      startOfSpeechSensitivity: StartSensitivity.START_SENSITIVITY_LOW,
      endOfSpeechSensitivity: EndSensitivity.END_SENSITIVITY_LOW,
      prefixPaddingMs: 20,
      silenceDurationMs: 100,
    }
  }
};
```

### Disable Automatic VAD

This configuration allows clients to take control of Voice Activity Detection (VAD) by disabling the automatic VAD feature. Clients are then responsible for sending `activityStart` and `activityEnd` messages.

```python
config = {
    "response_modalities": ["TEXT"],
    "realtime_input_config": {"automatic_activity_detection": {"disabled": True}},
}

async with client.aio.live.connect(model=model, config=config) as session:
    await session.send_realtime_input(activity_start=types.ActivityStart())
    await session.send_realtime_input(
        audio=types.Blob(data=audio_bytes, mime_type="audio/pcm;rate=16000")
    )
    await session.send_realtime_input(activity_end=types.ActivityEnd())
```

```javascript
const config = {
  responseModalities: [Modality.TEXT],
  realtimeInputConfig: {
    automaticActivityDetection: {
      disabled: true,
    }
  }
};

session.sendRealtimeInput({ activityStart: {} })
session.sendRealtimeInput({
  audio: {
    data: base64Audio,
    mimeType: "audio/pcm;rate=16000"
  }
});
session.sendRealtimeInput({ activityEnd: {} })
```

## Audio Configuration

### Input Audio Format
- **Format:** PCM
- **Sample Rate:** 16000 Hz
- **Bit Depth:** 16-bit
- **Channels:** Mono
- **MIME Type:** `audio/pcm;rate=16000`

### Output Audio Format
- **Format:** PCM  
- **Sample Rate:** 24000 Hz
- **Bit Depth:** 16-bit
- **Channels:** Mono

### Configure Microphone (Node.js)

```javascript
const micInstance = mic({
  rate: '16000',
  bitwidth: '16',
  channels: '1',
});
const micInputStream = micInstance.getAudioStream();

micInputStream.on('data', (data) => {
  // API expects base64 encoded PCM data
  session.sendRealtimeInput({
    audio: {
      data: data.toString('base64'),
      mimeType: "audio/pcm;rate=16000"
    }
  });
});

micInputStream.on('error', (err) => {
  console.error('Microphone error:', err);
});

micInstance.start();
console.log('Microphone started. Speak now...');
```

## Response Modalities

### Text-Only Responses

```python
config = {
    "response_modalities": ["TEXT"],
}
```

### Audio Responses

```python
config = {
    "response_modalities": ["AUDIO"],
}
```

```javascript
const config = {
  responseModalities: [Modality.AUDIO],
};
```

## Interruption Handling

Detect and handle user interruptions during model generation using VAD. When VAD detects an interruption, the ongoing generation is canceled.

```python
async for response in session.receive():
    if response.server_content.interrupted is True:
        # The generation was interrupted
        # Stop audio playback and clear queued playback here
```

```javascript
const turns = await handleTurn();

for (const turn of turns) {
  if (turn.serverContent && turn.serverContent.interrupted) {
    // The generation was interrupted
    // Stop audio playback and clear queued playback here
  }
}
```

## Turn Management

### Send Text Content

```python
message = "Hello, how are you?"
await session.send_client_content(turns=message, turn_complete=True)
```

```javascript
const message = 'Hello, how are you?';
session.sendClientContent({ turns: message, turnComplete: true });
```

### Send Incremental Turns

```python
turns = [
    {"role": "user", "parts": [{"text": "What is the capital of France?"}]},
    {"role": "model", "parts": [{"text": "Paris"}]},
]

await session.send_client_content(turns=turns, turn_complete=False)

turns = [{"role": "user", "parts": [{"text": "What is the capital of Germany?"}]}]

await session.send_client_content(turns=turns, turn_complete=True)
```

```javascript
let inputTurns = [
  { "role": "user", "parts": [{ "text": "What is the capital of France?" }] },
  { "role": "model", "parts": [{ "text": "Paris" }] },
]

session.sendClientContent({ turns: inputTurns, turnComplete: false })

inputTurns = [{ "role": "user", "parts": [{ "text": "What is the capital of Germany?" }] }]

session.sendClientContent({ turns: inputTurns, turnComplete: true })
```

### Detect Turn Complete

```python
async for response in session.receive():
    if response.server_content.turn_complete is True:
        # User's turn is complete
```

### Detect Generation Complete

```python
async for response in session.receive():
    if response.server_content.generation_complete is True:
        # The generation is complete
```

## Input Audio Transcription

Configure the Live API to transcribe user audio input:

```python
config = {
    "response_modalities": ["AUDIO"],
    "input_audio_transcription": {}
}

async with client.aio.live.connect(model=model, config=config) as session:
    audio_data = Path("16000.pcm").read_bytes()
    
    await session.send_realtime_input(
        audio=types.Blob(data=audio_data, mime_type='audio/pcm;rate=16000')
    )
    
    async for msg in session.receive():
        if msg.server_content.input_transcription:
            print('Transcript:', msg.server_content.input_transcription.text)
```

## Output Audio Transcription

Configure the Live API to transcribe the model's audio responses:

```python
config = {
    "response_modalities": ["AUDIO"],
    "output_audio_transcription": {}
}

async with client.aio.live.connect(model=model, config=config) as session:
    await session.send_client_content(
        turns={"role": "user", "parts": [{"text": message}]}, 
        turn_complete=True
    )
    
    async for response in session.receive():
        if response.server_content.model_turn:
            print("Model turn:", response.server_content.model_turn)
        if response.server_content.output_transcription:
            print("Transcript:", response.server_content.output_transcription.text)
```

## Thinking Configuration

Set up thinking capabilities for the native audio model:

```python
config = types.LiveConnectConfig(
    response_modalities=["AUDIO"],
    thinking_config=types.ThinkingConfig(
        thinking_budget=1024,
    )
)
```

```javascript
const config = {
  responseModalities: [Modality.AUDIO],
  thinkingConfig: {
    thinkingBudget: 1024,
  },
};
```

## Model

Use the native audio model for best results:

```python
model = "gemini-2.5-flash-native-audio-preview-12-2025"
```

```javascript
const model = 'gemini-2.5-flash-native-audio-preview-12-2025';
```

## Connect to Live API

### Python

```python
import asyncio
from google import genai
from google.genai import types

client = genai.Client()
model = "gemini-2.5-flash-native-audio-preview-12-2025"

config = {
    "response_modalities": ["AUDIO"],
    "system_instruction": "You are a helpful and friendly AI assistant.",
}

async def main():
    async with client.aio.live.connect(model=model, config=config) as session:
        # Send and receive audio
        pass

asyncio.run(main())
```

### JavaScript

```javascript
import { GoogleGenAI, Modality } from '@google/genai';

const ai = new GoogleGenAI({});
const model = 'gemini-2.5-flash-native-audio-preview-12-2025';
const config = { responseModalities: [Modality.TEXT] };

const session = await ai.live.connect({
  model: model,
  callbacks: {
    onopen: function () {
      console.debug('Opened');
    },
    onmessage: function (message) {
      responseQueue.push(message);
    },
    onerror: function (e) {
      console.debug('Error:', e.message);
    },
    onclose: function (e) {
      console.debug('Close:', e.reason);
    }
  },
  config: config
});
```

## Session Management

The Live API supports session resumption for maintaining conversations across disconnections. Sessions can be kept alive for up to 24 hours through server-side session state storage.

See [Session Management with Live API](session-management.md) for details.

## Tools and Function Calling

The Live API supports:
- Google Search (grounding)
- Function calling

See [Tool Use with Live API](tools.md) for details.

## Security

For client-side implementations, use ephemeral tokens instead of API keys to enhance security.

See [Ephemeral Tokens](ephemeral-tokens.md) for details.

---

*Generated from Google AI Developer Documentation*
