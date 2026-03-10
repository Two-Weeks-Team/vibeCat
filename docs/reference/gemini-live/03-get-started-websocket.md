# Get Started with Gemini Live API using WebSockets

**Source:** https://ai.google.dev/gemini-api/docs/live-api/get-started-websocket  
**Extracted:** 2026-03-10

---

## Overview

The Gemini Live API allows for real-time, bidirectional interaction with Gemini models, supporting audio, video, and text inputs and native audio outputs. This guide explains how to integrate directly with the API using raw WebSockets.

### Key Concepts

- **WebSocket Endpoint**: The specific URL to connect to.
- **Message Format**: All communication is done via JSON messages conforming to `LiveSessionRequest` and `LiveSessionResponse` structures.
- **Session Management**: You are responsible for maintaining the WebSocket connection.

---

## Authentication

Authentication is handled by including your API key as a query parameter in the WebSocket URL.

The endpoint format is:

```
wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1beta.GenerativeService.BidiGenerateContent?key=YOUR_API_KEY
```

Replace `YOUR_API_KEY` with your actual API key.

---

## Authentication with Ephemeral Tokens

If you are using ephemeral tokens, you need to connect to the `v1alpha` endpoint. The ephemeral token needs to be passed as an `access_token` query parameter.

The endpoint format for ephemeral keys is:

```
wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1alpha.GenerativeService.BidiGenerateContentConstrained?access_token={short-lived-token}
```

Replace `{short-lived-token}` with the actual ephemeral token.

---

## Connecting to the Live API

To start a live session, establish a WebSocket connection to the authenticated endpoint. The first message sent over the WebSocket must be a `LiveSessionRequest` containing the `config`. For the full configuration options, see the [Live API - WebSockets API reference](/api/live).

### Python Example

```python
import asyncio
import websockets
import json

API_KEY = "YOUR_API_KEY"
MODEL_NAME = "gemini-2.5-flash-native-audio-preview-12-2025"
WS_URL = f"wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1beta.GenerativeService.BidiGenerateContent?key={API_KEY}"

async def connect_and_configure():
    async with websockets.connect(WS_URL) as websocket:
        print("WebSocket Connected")
        
        # 1. Send the initial configuration
        config_message = {
            "config": {
                "model": f"models/{MODEL_NAME}",
                "responseModalities": ["AUDIO"],
                "systemInstruction": {
                    "parts": [{"text": "You are a helpful assistant."}]
                }
            }
        }
        await websocket.send(json.dumps(config_message))
        print("Configuration sent")
        
        # Keep the session alive for further interactions
        await asyncio.sleep(3600)  # Example: keep open for an hour

async def main():
    await connect_and_configure()

if __name__ == "__main__":
    asyncio.run(main())
```

---

## Sending Text

To send text input, construct a `LiveSessionRequest` with the `realtimeInput` field populated with text.

### Python Example

```python
# Inside the websocket context
async def send_text(websocket, text):
    text_message = {
        "realtimeInput": {
            "text": text
        }
    }
    await websocket.send(json.dumps(text_message))
    print(f"Sent text: {text}")

# Example usage:
# await send_text(websocket, "Hello, how are you?")
```

---

## Sending Audio

Audio needs to be sent as raw PCM data (raw 16-bit PCM audio, 16kHz, little-endian). Construct a `LiveSessionRequest` with the `realtimeInput` field, containing a `Blob` with the audio data. The `mimeType` is crucial.

### Python Example

```python
# Inside the websocket context
async def send_audio_chunk(websocket, chunk_bytes):
    import base64
    encoded_data = base64.b64encode(chunk_bytes).decode('utf-8')
    audio_message = {
        "realtimeInput": {
            "audio": {
                "data": encoded_data,
                "mimeType": "audio/pcm;rate=16000"
            }
        }
    }
    await websocket.send(json.dumps(audio_message))

# Assuming 'chunk' is your raw PCM audio bytes
# await send_audio_chunk(websocket, chunk)
```

**Note:** For an example of how to get the audio from the client device (e.g. the browser) see the end-to-end example on [GitHub](https://github.com/google-gemini/gemini-live-api-examples/blob/main/gemini-live-ephemeral-tokens-websocket/frontend/mediaUtils.js#L38-L74).

---

## Sending Video

Video frames are sent as individual images (e.g., JPEG or PNG). Similar to audio, use `realtimeInput` with a `Blob`, specifying the correct `mimeType`.

### Python Example

```python
# Inside the websocket context
async def send_video_frame(websocket, frame_bytes, mime_type="image/jpeg"):
    import base64
    encoded_data = base64.b64encode(frame_bytes).decode('utf-8')
    video_message = {
        "realtimeInput": {
            "video": {
                "data": encoded_data,
                "mimeType": mime_type
            }
        }
    }
    await websocket.send(json.dumps(video_message))

# Assuming 'frame' is your JPEG-encoded image bytes
# await send_video_frame(websocket, frame)
```

**Note:** For an example of how to get the video from the client device (e.g. the browser) see the end-to-end example on [GitHub](https://github.com/google-gemini/gemini-live-api-examples/blob/main/gemini-live-ephemeral-tokens-websocket/frontend/mediaUtils.js#L185-L222).

---

## Receiving Responses

The WebSocket will send back `LiveSessionResponse` messages. You need to parse these JSON messages and handle different types of content.

### Python Example

```python
# Inside the websocket context, in a receive loop
async def receive_loop(websocket):
    async for message in websocket:
        response = json.loads(message)
        print("Received:", response)
        
        if "serverContent" in response:
            server_content = response["serverContent"]
            
            # Receiving Audio
            if "modelTurn" in server_content and "parts" in server_content["modelTurn"]:
                for part in server_content["modelTurn"]["parts"]:
                    if "inlineData" in part:
                        audio_data_b64 = part["inlineData"]["data"]
                        # Process or play the base64 encoded audio data
                        # audio_data = base64.b64decode(audio_data_b64)
                        print(f"Received audio data (base64 len: {len(audio_data_b64)})")
            
            # Receiving Text Transcriptions
            if "inputTranscription" in server_content:
                print(f"User: {server_content['inputTranscription']['text']}")
            if "outputTranscription" in server_content:
                print(f"Gemini: {server_content['outputTranscription']['text']}")
        
        # Handling Tool Calls
        if "toolCall" in response:
            await handle_tool_call(websocket, response["toolCall"])

# Example usage:
# await receive_loop(websocket)
```

**Note:** For an example of how to handle the response, see the end-to-end example on [GitHub](https://github.com/google-gemini/gemini-live-api-examples/blob/main/gemini-live-ephemeral-tokens-websocket/frontend/geminilive.js#L22-L75).

### JavaScript Example

```javascript
websocket.onmessage = (event) => {
    const response = JSON.parse(event.data);
    console.log('Received:', response);
    
    if (response.serverContent) {
        const serverContent = response.serverContent;
        
        // Receiving Audio
        if (serverContent.modelTurn?.parts) {
            for (const part of serverContent.modelTurn.parts) {
                if (part.inlineData) {
                    const audioData = part.inlineData.data;  // Base64 encoded string
                    // Process or play audioData
                    console.log(`Received audio data (base64 len: ${audioData.length})`);
                }
            }
        }
        
        // Receiving Text Transcriptions
        if (serverContent.inputTranscription) {
            console.log('User:', serverContent.inputTranscription.text);
        }
        if (serverContent.outputTranscription) {
            console.log('Gemini:', serverContent.outputTranscription.text);
        }
    }
    
    // Handling Tool Calls
    if (response.toolCall) {
        handleToolCall(response.toolCall);
    }
};
```

---

## Handling Tool Calls

When the model requests a tool call, the `LiveSessionResponse` will contain a `toolCall` field. You must execute the function locally and send the result back to the WebSocket using a `LiveSessionRequest` with the `toolResponse` field.

### Python Example

```python
# Placeholder for your tool function
def my_tool_function(args):
    print(f"Executing tool with args: {args}")
    # Implement your tool logic here
    return {"status": "success", "data": "some result"}

async def handle_tool_call(websocket, tool_call):
    function_responses = []
    
    for fc in tool_call["functionCalls"]:
        # 1. Execute the function locally
        try:
            result = my_tool_function(fc.get("args", {}))
            response_data = {"result": result}
        except Exception as e:
            print(f"Error executing tool {fc['name']}: {e}")
            response_data = {"error": str(e)}
        
        # 2. Prepare the response
        function_responses.append({
            "name": fc["name"],
            "id": fc["id"],
            "response": response_data
        })
    
    # 3. Send the tool response back to the session
    tool_response_message = {
        "toolResponse": {
            "functionResponses": function_responses
        }
    }
    await websocket.send(json.dumps(tool_response_message))
    print("Sent tool response")

# This function is called within the receive_loop when a toolCall is detected.
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
- [Clone the example app from GitHub](https://github.com/google-gemini/gemini-live-api-examples/tree/main/gemini-live-ephemeral-tokens-websocket)
- [Use coding agent skills](/gemini-api/docs/coding-agents)

---

*Last updated: 2026-03-09 UTC*
