# Ephemeral Tokens

**Source:** https://ai.google.dev/gemini-api/docs/ephemeral-tokens

Ephemeral tokens are temporary authentication credentials that can be used instead of API keys when connecting to the Live API.

## Overview

Once you have an ephemeral token, you use it as if it were an API key, but remember that it only works for the Live API and only with the v1alpha version of the API.

The use of ephemeral tokens only adds value when deploying applications that follow a client-to-server implementation approach, where a backend server generates tokens for client applications to use.

## How Ephemeral Tokens Work

1. **Client authenticates with your backend** - Your client (e.g., web app) authenticates with your backend
2. **Backend requests token** - Your backend requests an ephemeral token from Gemini API's provisioning service
3. **Token issued** - The service issues a short-lived token
4. **Backend sends token to client** - Your backend sends this token to the client
5. **Client connects to Live API** - The client uses this token as if it were an API key

This process significantly enhances security because even if the token is extracted, its short lifespan limits potential damage.

## Benefits

- **Enhanced security** - Short-lived tokens limit damage if extracted
- **Improved latency** - Client sends data directly to Gemini, no proxy needed
- **No backend proxy** - Eliminates need for backend to proxy real-time data
- **Usage limits** - Can configure maximum uses per token

## Create Ephemeral Token

### Endpoint

```
POST https://generativelanguage.googleapis.com/v1alpha/auth_tokens:create
```

### Request Body

```json
{
  "config": {
    "uses": 1,
    "expire_time": "2025-05-17T00:30:00Z",
    "new_session_expire_time": "2025-05-17T00:01:00Z",
    "live_connect_constraints": {
      "model": "gemini-2.5-flash-native-audio-preview-12-2025",
      "config": {
        "session_resumption": {},
        "temperature": 0.7,
        "response_modalities": ["AUDIO"]
      }
    }
  }
}
```

### Configuration Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `uses` | integer | Maximum number of new sessions that can be started with this token. Default is 1. |
| `expire_time` | string | Absolute time at which the token will expire (ISO 8601). Default is 30 minutes from creation. |
| `new_session_expire_time` | string | Absolute time until which this token can be used to start new Live API sessions (ISO 8601). Default is 1 minute from creation. |
| `live_connect_constraints` | object | Constraints to lock the token to specific Live API configurations |
| `live_connect_constraints.model` | string | Specific model to use (required if constraints present) |
| `live_connect_constraints.config` | object | Model configuration constraints |
| `live_connect_constraints.config.session_resumption` | object | Empty object to allow session resumption |
| `live_connect_constraints.config.temperature` | number | Temperature setting for the model |
| `live_connect_constraints.config.response_modalities` | array | Desired response modalities (e.g., ["AUDIO"]) |

### Response

```json
{
  "name": "ephemeral-token-string-12345",
  "expireTime": "2025-05-17T00:30:00Z",
  "newSessionExpireTime": "2025-05-17T00:01:00Z"
}
```

### Python Example

```python
from google import genai

client = genai.Client()

response = client.auth_tokens.create(
    config={
        "uses": 1,
        "expire_time": "2025-05-17T00:30:00Z",
        "new_session_expire_time": "2025-05-17T00:01:00Z",
        "live_connect_constraints": {
            "model": "gemini-2.5-flash-native-audio-preview-12-2025",
            "config": {
                "session_resumption": {},
                "temperature": 0.7,
                "response_modalities": ["AUDIO"]
            }
        }
    }
)

ephemeral_token = response.name
print(f"Ephemeral token: {ephemeral_token}")
```

### cURL Example

```bash
curl -X POST "https://generativelanguage.googleapis.com/v1alpha/auth_tokens:create" \
-H "Content-Type: application/json" \
-H "x-goog-api-key: YOUR_API_KEY" \
-d '{
  "config": {
    "uses": 1,
    "expire_time": "2025-05-17T00:30:00Z",
    "new_session_expire_time": "2025-05-17T00:01:00Z",
    "live_connect_constraints": {
      "model": "gemini-2.5-flash-native-audio-preview-12-2025",
      "config": {
        "session_resumption": {},
        "temperature": 0.7,
        "response_modalities": ["AUDIO"]
      }
    }
  }
}'
```

## Use Ephemeral Token

### Connect with Token

```
wss://generativelanguage.googleapis.com/v1beta/models/{model}:bidiGenerateContent?token=EPHEMERAL_TOKEN
```

### Example Connection

```javascript
// Client-side WebSocket connection
const ws = new WebSocket(
  'wss://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash-native-audio-preview-12-2025:bidiGenerateContent?token=' + ephemeralToken
);
```

## Security Best Practices

1. **Never expose API keys in client code** - Use ephemeral tokens instead
2. **Set appropriate expiration times** - Use shortest practical duration
3. **Limit uses** - Set `uses` to minimum needed
4. **Lock to specific configurations** - Use `live_connect_constraints` to restrict token usage
5. **Validate tokens server-side** - Verify token validity before using

## When to Use Ephemeral Tokens

Use ephemeral tokens when:
- Building client-to-server applications
- Client connects directly to Live API
- Want to avoid exposing API keys in client code
- Need improved latency (no proxy needed)

Do NOT use ephemeral tokens when:
- Building server-to-server applications
- Backend proxies all Live API traffic
- Server-side API key management is acceptable

---

*Generated from Google AI Developer Documentation*
