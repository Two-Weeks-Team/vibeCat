# Get Started with Gemini Live API using the Google GenAI SDK

**Source:** https://ai.google.dev/gemini-api/docs/live-api/get-started-sdk  
**Extracted:** 2026-03-10

---

## Overview

The Gemini Live API allows for real-time, bidirectional interaction with Gemini models, supporting audio, video, and text inputs and native audio outputs. This guide explains how to integrate with the API using the Google GenAI SDK on your server.

### Key Concepts

- **Session**: A persistent connection to the model.
- **Config**: Setting up modalities (audio/text), voice, and system instructions.
- **Real-time Input**: Sending audio and video frames as blobs.

---

## Connecting to the Live API

Start a Live API session with an API key:

### Python

```python
import asyncio
from google import genai

client = genai.Client(api_key="YOUR_API_KEY")
model = "gemini-2.5-flash-native-audio-preview-12-2025"
config = {"response_modalities": ["AUDIO"]}

async def main():
    async with client.aio.live.connect(model=model, config=config) as session:
        print("Session started")
        # Send content...

if __name__ == "__main__":
    asyncio.run(main())
```

### JavaScript

```javascript
// See WebSocket guide for JavaScript implementation
```

---

## Sending Text

Text can be sent using `send_realtime_input` (Python) or `sendRealtimeInput` (JavaScript).

### Python

```python
await session.send_realtime_input(text="Hello, how are you?")
```

---

## Sending Audio

Audio needs to be sent as raw PCM data (raw 16-bit PCM audio, 16kHz, little-endian).

### Python

```python
# Assuming 'chunk' is your raw PCM audio bytes
await session.send_realtime_input(
    audio=types.Blob(
        data=chunk,
        mime_type="audio/pcm;rate=16000"
    )
)
```

**Note:** For an example of how to get the audio from the client device (e.g. the browser) see the end-to-end example on [GitHub](https://github.com/google-gemini/gemini-live-api-examples/blob/main/gemini-live-genai-python-sdk/frontend/media-handler.js#L31-L70).

---

## Sending Video

Video frames are sent as individual images (e.g., JPEG or PNG) at a specific frame rate (max 1 frame per second).

### Python

```python
# Assuming 'frame' is your JPEG-encoded image bytes
await session.send_realtime_input(
    video=types.Blob(
        data=frame,
        mime_type="image/jpeg"
    )
)
```

**Note:** For an example of how to get the video from the client device (e.g. the browser) see the end-to-end example on [GitHub](https://github.com/google-gemini/gemini-live-api-examples/blob/main/gemini-live-genai-python-sdk/frontend/media-handler.js#L84-L120).

---

## Receiving Audio

The model's audio responses are received as chunks of data.

### Python

```python
async for response in session.receive():
    if response.server_content and response.server_content.model_turn:
        for part in response.server_content.model_turn.parts:
            if part.inline_data:
                audio_data = part.inline_data.data
                # Process or play the audio data
```

**See also:**
- [Receive audio on your server example](https://github.com/google-gemini/gemini-live-api-examples/blob/main/gemini-live-genai-python-sdk/gemini_live.py#L86-L98)
- [Play audio in the browser example](https://github.com/google-gemini/gemini-live-api-examples/blob/main/gemini-live-genai-python-sdk/frontend/media-handler.js#L145-L174)

---

## Receiving Text

Transcriptions for both user input and model output are available in the server content.

### Python

```python
async for response in session.receive():
    content = response.server_content
    if content:
        if content.input_transcription:
            print(f"User: {content.input_transcription.text}")
        if content.output_transcription:
            print(f"Gemini: {content.output_transcription.text}")
```

---

## Handling Tool Calls

The API supports tool calling (function calling). When the model requests a tool call, you must execute the function and send the response back.

### Python

```python
async for response in session.receive():
    if response.tool_call:
        function_responses = []
        for fc in response.tool_call.function_calls:
            # 1. Execute the function locally
            result = my_tool_function(**fc.args)
            
            # 2. Prepare the response
            function_responses.append(types.FunctionResponse(
                name=fc.name,
                id=fc.id,
                response={"result": result}
            ))
        
        # 3. Send the tool response back to the session
        await session.send_tool_response(function_responses=function_responses)
```

---

## What's Next

- Read the full Live API [Capabilities](/gemini-api/docs/live-guide) guide for key capabilities and configurations; including Voice Activity Detection and native audio features.
- Read the [Tool use](/gemini-api/docs/live-tools) guide to learn how to integrate Live API with tools and function calling.
- Read the [Session management](/gemini-api/docs/live-session) guide for managing long running conversations.
- Read the [Ephemeral tokens](/gemini-api/docs/ephemeral-tokens) guide for secure authentication in client-to-server applications.
- For more information about the underlying WebSockets API, see the [WebSockets API reference](/api/live).

---

## Example Resources

- [Try the Live API in Google AI Studio](https://aistudio.google.com/live)
- [Clone the example app from GitHub](https://github.com/google-gemini/gemini-live-api-examples/tree/main/gemini-live-genai-python-sdk)
- [Use coding agent skills](/gemini-api/docs/coding-agents)

---

*Last updated: 2026-03-09 UTC*
