# Get Started with Live API

**Source:** https://ai.google.dev/gemini-api/docs/live

The Live API enables low-latency, real-time voice and video interactions with Gemini. It processes continuous streams of audio, video, or text to deliver immediate, human-like spoken responses, creating a natural conversational experience for your users.

## Features

Live API offers a comprehensive set of features:

- **Voice Activity Detection (VAD)** - Automatically detects when users start and stop speaking
- **Tool Use and Function Calling** - Enable the model to interact with external tools and APIs
- **Session Management** - Manage long-running conversations with session resumption
- **Ephemeral Tokens** - Secure client-side authentication

## Choose an Implementation Approach

When integrating with Live API, you can choose from the following implementation approaches:

### 1. Server-to-Server

Your backend connects to the Live API using WebSockets. Typically, your client sends stream data (audio, video, text) to your server, which then forwards it to the Live API.

**Pros:**
- API key stays on server (secure)
- Full control over data flow
- Easier authentication

**Cons:**
- Higher latency (data through your server)
- More server infrastructure

### 2. Client-to-Server

Your frontend code connects directly to the Live API using WebSockets, bypassing your backend. Requires ephemeral tokens for security.

**Pros:**
- Lower latency (direct connection)
- Less server infrastructure
- Better user experience

**Cons:**
- Requires ephemeral token setup
- More complex security considerations

## Quick Start

### Step 1: Get an API Key

1. Go to [Google AI Studio](https://aistudio.google.com/app/apikey)
2. Create a new API key
3. Copy the key (you'll need it for authentication)

### Step 2: Install the SDK

**Python:**
```bash
pip install google-genai
```

**JavaScript/Node.js:**
```bash
npm install @google/genai
```

### Step 3: Try the Live API

Try the Live API in [Google AI Studio](https://aistudio.google.com/live).

## Code Examples

### Python Quick Start

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

### JavaScript Quick Start

```javascript
import { GoogleGenAI, Modality } from '@google/genai';

const ai = new GoogleGenAI({});
const model = 'gemini-2.5-flash-native-audio-preview-12-2025';

const session = await ai.live.connect({
  model: model,
  config: { responseModalities: [Modality.AUDIO] },
  callbacks: {
    onopen: () => console.log('Connected'),
    onmessage: (msg) => console.log('Received:', msg),
    onerror: (e) => console.error('Error:', e),
    onclose: () => console.log('Closed')
  }
});
```

## Available Models

Use the native audio model for best results:

- `gemini-2.5-flash-native-audio-preview-12-2025` - Latest native audio model

## Next Steps

- [Learn about capabilities](capabilities.md)
- [Build with Live API](build-with-live-api.md)
- [Session management](session-management.md)
- [Use tools](tools.md)
- [Secure with ephemeral tokens](ephemeral-tokens.md)

---

*Generated from Google AI Developer Documentation*
