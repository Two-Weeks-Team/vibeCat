# Audio Understanding

**Source:** https://ai.google.dev/gemini-api/docs/audio  
**Extracted:** 2026-03-10

---

## Overview

Gemini can analyze and understand audio input and generate text responses to it, enabling use cases like:

- Describe, summarize, or answer questions about audio content.
- Provide a transcription and translation of the audio (speech to text).
- Detect emotion in speech and music.
- Analyze specific segments of the audio, and provide timestamps.

**Note:** As of now the Gemini API doesn't support real-time transcription use cases. For real-time voice and video interactions refer to the [Live API](/gemini-api/docs/live). For dedicated speech to text models with support for real-time transcription, use the [Google Cloud Speech-to-Text API](https://cloud.google.com/speech-to-text).

---

## Quick Start

### Python

```python
from google import genai

client = genai.Client()
myfile = client.files.upload(file="path/to/sample.mp3")

response = client.models.generate_content(
    model="gemini-3-flash-preview",
    contents=["Describe this audio clip", myfile]
)

print(response.text)
```

---

## Transcribe Speech to Text

This example application shows how to prompt the Gemini API to transcribe, translate, and summarize speech, including timestamps and emotion detection using [structured outputs](/gemini-api/docs/structured-output).

### Python Example

```python
from google import genai
from google.genai import types

client = genai.Client()
YOUTUBE_URL = "https://www.youtube.com/watch?v=ku-N-eS1lgM"

def main():
    prompt = """
    Process the audio file and generate a detailed transcription.
    
    Requirements:
    1. Provide accurate timestamps for each segment (Format: MM:SS).
    2. Detect the primary language of each segment.
    3. If the segment is in a language different than English, also provide the English translation.
    4. Identify the primary emotion of the speaker in this segment. You MUST choose exactly one of the following: Happy, Sad, Angry, Neutral.
    5. Provide a brief summary of the entire audio at the beginning.
    """
    
    response = client.models.generate_content(
        model="gemini-3-flash-preview",
        contents=[
            types.Content(
                parts=[
                    types.Part(
                        file_data=types.FileData(
                            file_uri=YOUTUBE_URL
                        )
                    ),
                    types.Part(text=prompt)
                ]
            )
        ],
        config=types.GenerateContentConfig(
            response_mime_type="application/json",
            response_schema=types.Schema(
                type=types.Type.OBJECT,
                properties={
                    "summary": types.Schema(
                        type=types.Type.STRING,
                        description="A concise summary of the audio content.",
                    ),
                    "segments": types.Schema(
                        type=types.Type.ARRAY,
                        description="List of transcribed segments with timestamp.",
                        items=types.Schema(
                            type=types.Type.OBJECT,
                            properties={
                                "timestamp": types.Schema(type=types.Type.STRING),
                                "content": types.Schema(type=types.Type.STRING),
                                "language": types.Schema(type=types.Type.STRING),
                                "language_code": types.Schema(type=types.Type.STRING),
                                "translation": types.Schema(type=types.Type.STRING),
                                "emotion": types.Schema(
                                    type=types.Type.STRING,
                                    enum=["happy", "sad", "angry", "neutral"]
                                ),
                            },
                            required=["timestamp", "content", "language", "language_code", "emotion"],
                        ),
                    ),
                },
                required=["summary", "segments"],
            ),
        ),
    )
    
    print(response.text)

if __name__ == "__main__":
    main()
```

---

## Input Audio

You can provide audio data to Gemini in the following ways:

1. [Upload an audio file](#upload-an-audio-file) before making a request to `generateContent`.
2. [Pass inline audio data](#pass-audio-data-inline) with the request to `generateContent`.

To learn about other file input methods, see the [File input methods](/gemini-api/docs/file-input-methods) guide.

---

## Upload an Audio File

You can use the [Files API](/gemini-api/docs/files) to upload an audio file. Always use the Files API when the total request size (including the files, text prompt, system instructions, etc.) is larger than 20 MB.

The following code uploads an audio file and then uses the file in a call to `generateContent`:

### Python

```python
from google import genai

client = genai.Client()
myfile = client.files.upload(file="path/to/sample.mp3")

response = client.models.generate_content(
    model="gemini-3-flash-preview",
    contents=["Describe this audio clip", myfile]
)

print(response.text)
```

To learn more about working with media files, see [Files API](/gemini-api/docs/files).

---

## Pass Audio Data Inline

Instead of uploading an audio file, you can pass inline audio data in the request to `generateContent`:

### Python

```python
from google import genai
from google.genai import types

with open('path/to/small-sample.mp3', 'rb') as f:
    audio_bytes = f.read()

client = genai.Client()

response = client.models.generate_content(
    model='gemini-3-flash-preview',
    contents=[
        'Describe this audio clip',
        types.Part.from_bytes(
            data=audio_bytes,
            mime_type='audio/mp3',
        )
    ]
)

print(response.text)
```

**Important notes about inline audio data:**

- The maximum request size is 20 MB, which includes text prompts, system instructions, and files provided inline. If your file's size will make the **total request size** exceed 20 MB, then use the Files API to [upload an audio file](#upload-an-audio-file) for use in the request.
- If you're using an audio sample multiple times, it's more efficient to [upload an audio file](#upload-an-audio-file).

---

## Get a Transcript

To get a transcript of audio data, just ask for it in the prompt:

### Python

```python
from google import genai

client = genai.Client()
myfile = client.files.upload(file='path/to/sample.mp3')

prompt = 'Generate a transcript of the speech.'

response = client.models.generate_content(
    model='gemini-3-flash-preview',
    contents=[prompt, myfile]
)

print(response.text)
```

---

## Refer to Timestamps

You can refer to specific sections of an audio file using timestamps of the form `MM:SS`. For example, the following prompt requests a transcript that:

- Starts at 2 minutes 30 seconds from the beginning of the file.
- Ends at 3 minutes 29 seconds from the beginning of the file.

### Python

```python
# Create a prompt containing timestamps.
prompt = "Provide a transcript of the speech from 02:30 to 03:29."
```

---

## Count Tokens

Call the `countTokens` method to get a count of the number of tokens in an audio file:

### Python

```python
from google import genai

client = genai.Client()

response = client.models.count_tokens(
    model='gemini-3-flash-preview',
    contents=[myfile]
)

print(response)
```

---

## Supported Audio Formats

Gemini supports the following audio format MIME types:

| Format | MIME Type |
|--------|-----------|
| WAV | `audio/wav` |
| MP3 | `audio/mp3` |
| AIFF | `audio/aiff` |
| AAC | `audio/aac` |
| OGG Vorbis | `audio/ogg` |
| FLAC | `audio/flac` |

---

## Technical Details About Audio

- Gemini represents each second of audio as 32 tokens; for example, one minute of audio is represented as 1,920 tokens.
- Gemini can "understand" non-speech components, such as birdsong or sirens.
- The maximum supported length of audio data in a single prompt is 9.5 hours. Gemini doesn't limit the **number** of audio files in a single prompt; however, the total combined length of all audio files in a single prompt can't exceed 9.5 hours.
- Gemini downsamples audio files to a 16 Kbps data resolution.
- If the audio source contains multiple channels, Gemini combines those channels into a single channel.

---

## What's Next

- [File prompting strategies](/gemini-api/docs/files#prompt-guide): The Gemini API supports prompting with text, image, audio, and video data, also known as multimodal prompting.
- [System instructions](/gemini-api/docs/text-generation#system-instructions): System instructions let you steer the behavior of the model based on your specific needs and use cases.
- [Safety guidance](/gemini-api/docs/safety-guidance): Sometimes generative AI models produce unexpected outputs, such as outputs that are inaccurate, biased, or offensive. Post-processing and human evaluation are essential to limit the risk of harm from such outputs.

---

*Last updated: 2026-03-03 UTC*
