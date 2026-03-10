# Tool Use with Live API

**Source:** https://ai.google.dev/gemini-api/docs/live-tools

Tool use allows Live API to go beyond simple conversation by enabling it to perform actions in the real world and pull in external context while maintaining a real-time connection.

## Supported Tools Summary

The following tools are available for the Live API model:

| Tool | `gemini-2.5-flash-native-audio-preview-12-2025` |
|------|------------------------------------------------|
| Search (Google Search) | Yes |
| Function Calling | Yes |
| Google Maps | No |
| Code Execution | No |
| URL Context | No |

## Function Calling

Live API supports function calling, similar to regular content generation requests. Function calling enables the Live API to interact with external data and programs.

### Define Function Declarations

```python
from google.genai import types

# Define function schema
turn_on_the_lights = {
    "name": "turn_on_the_lights",
    "description": "Turn on the smart lights in the room",
    "parameters": {
        "type": "object",
        "properties": {
            "room": {
                "type": "string",
                "description": "The room where lights should be turned on"
            }
        },
        "required": ["room"]
    }
}

turn_off_the_lights = {
    "name": "turn_off_the_lights", 
    "description": "Turn off the smart lights in the room",
    "parameters": {
        "type": "object",
        "properties": {
            "room": {
                "type": "string",
                "description": "The room where lights should be turned off"
            }
        },
        "required": ["room"]
    }
}
```

### Configure Tools in Session

```python
tools = [
    {"google_search": {}},
    {"function_declarations": [turn_on_the_lights, turn_off_the_lights]},
]

config = {"response_modalities": ["AUDIO"], "tools": tools}
```

```javascript
const tools = [
  { googleSearch: {} },
  { functionDeclarations: [turn_on_the_lights, turn_off_the_lights] }
]

const config = {
  responseModalities: [Modality.AUDIO],
  tools: tools
}
```

### Handle Tool Calls

Unlike the `generateContent` API, Live API requires **manual handling** of tool responses.

```python
async for msg in session.receive():
    if msg.server_content.tool_call:
        # Model wants to call a function
        tool_call = msg.server_content.tool_call
        function_name = tool_call.name
        arguments = tool_call.args
        
        # Execute the function
        result = execute_function(function_name, arguments)
        
        # Send result back to the model
        await session.send_tool_response(result)
```

```javascript
// Handle tool call
for (const turn of turns) {
  if (turn.serverContent && turn.serverContent.toolCall) {
    const toolCall = turn.serverContent.toolCall;
    const functionName = toolCall.name;
    const args = toolCall.args;
    
    // Execute the function
    const result = executeFunction(functionName, args);
    
    // Send result back to the model
    session.sendToolResponse({
      id: toolCall.id,
      name: functionName,
      response: result
    });
  }
}
```

## Google Search Grounding

Enable the model to perform Google searches for real-time information:

```python
tools = [
    {"google_search": {}},
]

config = {"response_modalities": ["AUDIO"], "tools": tools}
```

```javascript
const tools = [
  { googleSearch: {} }
]

const config = {
  responseModalities: [Modality.AUDIO],
  tools: tools
}
```

### Example: Search and Act

```python
prompt = """
Hey, I need you to do two things for me.

1. Use Google Search to look up information about the largest earthquake in California the week of Dec 5 2024?
2. Then turn on the lights

Thanks!
"""

tools = [
    {"google_search": {}},
    {"function_declarations": [turn_on_the_lights, turn_off_the_lights]},
]

config = {"response_modalities": ["AUDIO"], "tools": tools}

# ... remaining model call
```

```javascript
const prompt = `Hey, I need you to do two things for me.

1. Use Google Search to look up information about the largest earthquake in California the week of Dec 5 2024?
2. Then turn on the lights

Thanks!
`

const tools = [
  { googleSearch: {} },
  { functionDeclarations: [turn_on_the_lights, turn_off_the_lights] }
]

const config = {
  responseModalities: [Modality.AUDIO],
  tools: tools
}
```

## Tool Response Format

### Send Tool Response

```python
await session.send_tool_response(
    function_responses=[{
        "id": "call_123",
        "name": "function_name",
        "response": {"result": "value"}
    }]
)
```

```javascript
session.sendToolResponse({
  id: "call_123",
  name: "function_name", 
  response: {"result": "value"}
});
```

## Key Differences from generateContent API

| Feature | generateContent API | Live API |
|---------|---------------------|----------|
| Automatic Tool Response | Yes | No |
| Manual Handling Required | No | Yes |
| Real-time Connection | No | Yes |
| Tool Response Method | Part of response | `send_tool_response()` |

## Important Notes

1. **Manual response handling** - Unlike generateContent API, Live API requires you to handle tool calls manually
2. **Real-time connection** - Tools maintain the real-time connection during execution
3. **Function response format** - Must provide responses as FunctionResponse objects
4. **Error handling** - Handle tool call errors gracefully

---

*Generated from Google AI Developer Documentation*
