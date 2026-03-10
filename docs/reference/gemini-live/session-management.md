# Session Management with Live API

**Source:** https://ai.google.dev/gemini-api/docs/live-session

In the Live API, a session refers to a persistent connection where input and output are streamed continuously over the same connection. This unique session design enables low latency and supports unique features.

## Session Lifetime

### Connection Limits

The lifetime of a connection is limited to around 10 minutes. When the connection terminates, the session terminates as well. You can configure a single session to stay active over multiple connections using session resumption.

You'll receive a `GoAway` message before the connection ends, allowing you to take further actions.

### Long-Running Sessions

The Live API model now supports enhanced session management capabilities that allow you to maintain continuous interactions even during temporary network disruptions. Sessions can be kept alive for up to 24 hours through server-side session state storage.

## Session Resumption

To prevent session termination when the server periodically resets the WebSocket connection, configure the `sessionResumption` field within the setup configuration.

Passing this configuration causes the server to send `SessionResumptionUpdate` messages, which can be used to resume the session by passing the last resumption token as the `SessionResumptionConfig.handle` of the subsequent connection.

**Important:** Resumption tokens are valid for 2 hours after the last session termination.

### Configure Session Resumption (Python)

```python
import asyncio
from google import genai
from google.genai import types

client = genai.Client()
model = "gemini-2.5-flash-native-audio-preview-12-2025"

async def main():
    # Start with a new session (no previous handle)
    previous_session_handle = None
    
    print(f"Connecting to the service with handle {previous_session_handle}...")
    async with client.aio.live.connect(
        model=model,
        config=types.LiveConnectConfig(
            response_modalities=["AUDIO"],
            session_resumption=types.SessionResumptionConfig(
                # The handle of the session to resume is passed here,
                # or else None to start a new session.
                handle=previous_session_handle
            ),
        ),
    ) as session:
        while True:
            await session.send_client_content(
                turns=types.Content(
                    role="user", parts=[types.Part(text="Hello world!")]
                )
            )
            async for message in session.receive():
                # Periodically, the server will send update messages that may
                # contain a handle for the current state of the session.
                if message.session_resumption_update:
                    update = message.session_resumption_update
                    if update.resumable and update.new_handle:
                        # The handle should be retained and linked to the session.
                        print(f"Received new session handle: {update.new_handle}")
                        return update.new_handle

                # For the purposes of this example, placeholder input is continually fed
                # to the model. In non-sample code, the model inputs would come from
                # the user.
                if message.server_content and message.server_content.turn_complete:
                    break

if __name__ == "__main__":
    asyncio.run(main())
```

### Configure Session Resumption (JavaScript)

```javascript
import { GoogleGenAI, Modality } from '@google/genai';

const ai = new GoogleGenAI({});
const model = 'gemini-2.5-flash-native-audio-preview-12-2025';

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

  // Previous session handle (or null for new session)
  const previousSessionHandle = null;

  console.debug('Connecting to the service with handle %s...', previousSessionHandle)
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
      },
    },
    config: {
      responseModalities: [Modality.AUDIO],
      sessionResumption: { handle: previousSessionHandle }
    }
  });

  const inputTurns = 'Hello how are you?';
  session.sendClientContent({ turns: inputTurns });

  const turns = await handleTurn();
  for (const turn of turns) {
    if (turn.sessionResumptionUpdate) {
      if (turn.sessionResumptionUpdate.resumable && turn.sessionResumptionUpdate.newHandle) {
        let newHandle = turn.sessionResumptionUpdate.newHandle
        console.log('Received new session handle:', newHandle)
        // Store newHandle and use it to resume the session later
      }
    }
  }

  session.close();
}

async function main() {
  await live().catch((e) => console.error('got error', e));
}

main();
```

### Resume a Session

To resume a previous session:

1. Store the `new_handle` from `SessionResumptionUpdate` messages
2. When reconnecting, pass this handle in the configuration:

```python
previous_session_handle = "stored-handle-from-previous-session"

async with client.aio.live.connect(
    model=model,
    config=types.LiveConnectConfig(
        response_modalities=["AUDIO"],
        session_resumption=types.SessionResumptionConfig(
            handle=previous_session_handle
        ),
    ),
) as session:
    # Session is resumed from previous state
    pass
```

```javascript
const previousSessionHandle = "stored-handle-from-previous-session";

const session = await ai.live.connect({
  model: model,
  config: {
    responseModalities: [Modality.AUDIO],
    sessionResumption: { handle: previousSessionHandle }
  },
  callbacks: {...}
});
```

## Handling Disconnection

### GoAway Message

The server sends a `GoAway` message before the connection ends. Handle this message to gracefully reconnect or resume the session:

```python
async for message in session.receive():
    if message.go_away:
        # Connection will close soon
        # Prepare to reconnect
        print(f"Received GoAway: {message.go_away}")
```

### Automatic Reconnection

Implement automatic reconnection with session resumption:

```python
async def maintain_session(model, session_handle=None):
    while True:
        try:
            async with client.aio.live.connect(
                model=model,
                config=types.LiveConnectConfig(
                    response_modalities=["AUDIO"],
                    session_resumption=types.SessionResumptionConfig(
                        handle=session_handle
                    ),
                ),
            ) as session:
                # Update handle when new one is received
                async for message in session.receive():
                    if message.session_resumption_update:
                        if message.session_resumption_update.new_handle:
                            session_handle = message.session_resumption_update.new_handle
                    
                    if message.go_away:
                        # Reconnect immediately
                        break
                        
        except Exception as e:
            print(f"Connection error: {e}")
            await asyncio.sleep(1)
```

## Best Practices

1. **Store session handles persistently** - Save handles to database or file system
2. **Implement reconnection logic** - Handle connection drops gracefully
3. **Monitor token expiration** - Handles expire after 2 hours
4. **Handle GoAway messages** - Prepare for reconnection before connection drops
5. **Test edge cases** - Network failures, token expiration, etc.

---

*Generated from Google AI Developer Documentation*
