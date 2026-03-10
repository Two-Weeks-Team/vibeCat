# Build with Live API

**Source:** https://ai.google.dev/gemini-api/docs/live-api/build-with-live-api

This guide provides detailed instructions for building applications with the Gemini Live API.

## Getting Started

The Live API enables low-latency, real-time voice and video interactions with Gemini. It processes continuous streams of audio, video, or text to deliver immediate, human-like spoken responses.

## Quick Start

### Prerequisites

1. **Get an API Key:** Obtain your API key from Google AI Studio
2. **Choose a Model:** Use `gemini-2.5-flash-native-audio-preview-12-2025` for best results

### Basic Connection Flow

1. Connect to the WebSocket endpoint
2. Send configuration message
3. Stream audio/video/text
4. Receive model responses
5. Handle completion and interruptions

## WebSocket Connection

### Endpoint

```
wss://generativelanguage.googleapis.com/v1beta/models/{model}:bidiGenerateContent
```

### Authentication

Use your API key in the WebSocket handshake:

```
wss://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash-native-audio-preview-12-2025:bidiGenerateContent?key=YOUR_API_KEY
```

## Complete Example (Python)

```python
import asyncio
from pathlib import Path
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
        # Read audio file
        audio_data = Path("16000.pcm").read_bytes()
        
        # Send audio input
        await session.send_realtime_input(
            audio=types.Blob(data=audio_data, mime_type='audio/pcm;rate=16000')
        )
        
        # Receive responses
        async for msg in session.receive():
            if msg.server_content.model_turn:
                # Handle model response (audio/text)
                pass
            if msg.server_content.turn_complete:
                # Turn complete
                pass

if __name__ == "__main__":
    asyncio.run(main())
```

## Complete Example (JavaScript)

```javascript
import { GoogleGenAI, Modality } from '@google/genai';
import * as fs from "node:fs";

const ai = new GoogleGenAI({});
const model = 'gemini-2.5-flash-native-audio-preview-12-2025';
const config = { responseModalities: [Modality.AUDIO] };

async function live() {
  const responseQueue = [];

  async function waitMessage() {
    let done = false;
    let message = undefined;
    while (!done) {
      message = responseQueue.shift();
      if (message) {
        done = true;
      } else {
        await new Promise((resolve) => setTimeout(resolve, 100));
      }
    }
    return message;
  }

  async function handleTurn() {
    const turns = [];
    let done = false;
    while (!done) {
      const message = await waitMessage();
      turns.push(message);
      if (message.serverContent && message.serverContent.turnComplete) {
        done = true;
      }
    }
    return turns;
  }

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

  // Read and send audio file
  const fileBuffer = fs.readFileSync("sample.pcm");
  const base64Audio = Buffer.from(fileBuffer).toString('base64');

  session.sendRealtimeInput({
    audio: {
      data: base64Audio,
      mimeType: "audio/pcm;rate=16000"
    }
  });

  // Handle responses
  const turns = await handleTurn();
  for (const turn of turns) {
    if (turn.serverContent && turn.serverContent.modelTurn) {
      // Process model response
    }
  }

  session.close();
}

live().catch((e) => console.error('got error', e));
```

## Audio Streaming

### Sending Audio

Send base64-encoded PCM audio data:

```javascript
session.sendRealtimeInput({
  audio: {
    data: base64Audio,
    mimeType: "audio/pcm;rate=16000"
  }
});
```

### Receiving Audio

Process incoming audio from the model:

```javascript
async function messageLoop() {
  while (true) {
    const message = await waitMessage();
    
    // Handle interruption
    if (message.serverContent && message.serverContent.interrupted) {
      audioQueue.length = 0;  // Clear audio queue
      continue;
    }
    
    // Extract audio from model turn
    if (message.serverContent && message.serverContent.modelTurn) {
      for (const part of message.serverContent.modelTurn.parts) {
        if (part.inlineData && part.inlineData.data) {
          // Play audio
          const audioData = Buffer.from(part.inlineData.data, 'base64');
          audioQueue.push(audioData);
        }
      }
    }
  }
}
```

## Video Streaming

The Live API supports video input from cameras or screen capture. Video is sent as image frames at configurable intervals.

## Text Input/Output

Send text messages:

```python
await session.send_client_content(
    turns={"role": "user", "parts": [{"text": "Hello!"}]}, 
    turn_complete=True
)
```

Receive text responses:

```python
async for msg in session.receive():
    if msg.server_content.model_turn:
        for part in msg.server_content.model_turn.parts:
            if part.text:
                print(part.text)
```

## Error Handling

```python
async with client.aio.live.connect(model=model, config=config) as session:
    try:
        async for msg in session.receive():
            # Process message
            pass
    except Exception as e:
        print(f"Error: {e}")
```

## Best Practices

### 1. Audio Quality
- Use 16kHz, 16-bit, mono PCM for input
- Ensure low latency microphone capture
- Implement audio buffering for smooth playback

### 2. Connection Management
- Implement reconnection logic
- Use session resumption for long conversations
- Handle connection drops gracefully

### 3. Interruption Handling
- Monitor for interruption signals
- Stop audio playback immediately on interruption
- Clear audio queues on interruption

### 4. Resource Management
- Close connections properly
- Clean up audio resources
- Monitor memory usage

### 5. Security
- Use ephemeral tokens for client-side apps
- Never expose API keys in client code
- Validate all user input

## Common Patterns

### Real-time Conversation Loop

```python
async def conversation_loop():
    async with client.aio.live.connect(model=model, config=config) as session:
        while True:
            # Capture audio from microphone
            audio_data = capture_audio_chunk()
            
            # Send to API
            await session.send_realtime_input(
                audio=types.Blob(data=audio_data, mime_type='audio/pcm;rate=16000')
            )
            
            # Receive and play response
            async for msg in session.receive():
                if msg.server_content.model_turn:
                    # Play audio
                    play_audio(msg.server_content.model_turn)
                
                if msg.server_content.turn_complete:
                    break
```

### Handle Tool Calls

```python
async for msg in session.receive():
    if msg.server_content.tool_call:
        # Execute function
        result = execute_function(
            msg.server_content.tool_call.name,
            msg.server_content.tool_call.args
        )
        
        # Send result back
        await session.send_tool_response(result)
```

## Go SDK Usage

The Go SDK provides access to the Live API through the `google.golang.org/genai` package.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "google.golang.org/genai"
)

func main() {
    ctx := context.Background()
    client, err := genai.NewClient(ctx, nil)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Use client.LiveConnect for WebSocket-based live connections
    // See Go SDK documentation for detailed API
}
```

## Related Documentation

- [API Live Reference](api-live-reference.md)
- [Live API Capabilities](capabilities.md)
- [Session Management](session-management.md)
- [Tools with Live API](tools.md)
- [Ephemeral Tokens](ephemeral-tokens.md)

---

*Generated from Google AI Developer Documentation*
