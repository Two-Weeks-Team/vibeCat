#!/usr/bin/env python3
"""
VibeCat Demo Video TTS Generator
Generates voice-over WAV files using Gemini TTS API.
Reads dubbing-script.json and produces individual WAV files per line.
"""

import json
import os
import sys
import struct
import time
import base64
import requests

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
DUBBING_SCRIPT = os.path.join(SCRIPT_DIR, "dubbing-script.json")
OUTPUT_DIR = os.path.join(SCRIPT_DIR, "..", "docs", "video", "tts")

# Gemini TTS API
API_URL = (
    "https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent"
)
SAMPLE_RATE = 24000  # Gemini TTS outputs 24kHz LINEAR16
CHANNELS = 1
SAMPLE_WIDTH = 2  # 16-bit


def load_api_key():
    """Load Gemini API key from .env.test"""
    env_file = os.path.join(SCRIPT_DIR, "..", ".env.test")
    if not os.path.exists(env_file):
        env_file = os.path.join(SCRIPT_DIR, "..", ".env")
    if not os.path.exists(env_file):
        print("ERROR: No .env.test or .env found")
        sys.exit(1)

    with open(env_file) as f:
        for line in f:
            line = line.strip()
            if line.startswith("GEMINI_API_KEY="):
                return line.split("=", 1)[1].strip().strip('"').strip("'")

    print("ERROR: GEMINI_API_KEY not found in env file")
    sys.exit(1)


def write_wav(filename, pcm_data):
    """Write PCM data as WAV file."""
    data_size = len(pcm_data)
    with open(filename, "wb") as f:
        # RIFF header
        f.write(b"RIFF")
        f.write(struct.pack("<I", 36 + data_size))
        f.write(b"WAVE")
        # fmt chunk
        f.write(b"fmt ")
        f.write(struct.pack("<I", 16))  # chunk size
        f.write(struct.pack("<H", 1))  # PCM format
        f.write(struct.pack("<H", CHANNELS))
        f.write(struct.pack("<I", SAMPLE_RATE))
        f.write(struct.pack("<I", SAMPLE_RATE * CHANNELS * SAMPLE_WIDTH))
        f.write(struct.pack("<H", CHANNELS * SAMPLE_WIDTH))
        f.write(struct.pack("<H", SAMPLE_WIDTH * 8))
        # data chunk
        f.write(b"data")
        f.write(struct.pack("<I", data_size))
        f.write(pcm_data)


def generate_tts(api_key, text, voice_name, model):
    """Generate TTS audio using Gemini API. Returns raw PCM bytes."""
    url = API_URL.format(model=model)
    payload = {
        "contents": [{"parts": [{"text": text}]}],
        "generationConfig": {
            "responseModalities": ["AUDIO"],
            "speechConfig": {
                "voiceConfig": {"prebuiltVoiceConfig": {"voiceName": voice_name}}
            },
        },
    }

    resp = requests.post(
        f"{url}?key={api_key}",
        headers={"Content-Type": "application/json"},
        json=payload,
        timeout=30,
    )

    if resp.status_code != 200:
        print(f"  ERROR: API returned {resp.status_code}: {resp.text[:200]}")
        return None

    data = resp.json()
    candidates = data.get("candidates", [])
    if not candidates:
        print("  ERROR: No candidates in response")
        return None

    content = candidates[0].get("content", {})
    parts = content.get("parts", [])

    pcm_chunks = []
    for part in parts:
        inline = part.get("inlineData")
        if inline and inline.get("data"):
            pcm_chunks.append(base64.b64decode(inline["data"]))

    if not pcm_chunks:
        print("  ERROR: No audio data in response")
        return None

    return b"".join(pcm_chunks)


def main():
    api_key = load_api_key()
    print(f"API key loaded: {api_key[:10]}...")

    with open(DUBBING_SCRIPT) as f:
        script = json.load(f)

    voices = script["voices"]
    lines = script["lines"]

    os.makedirs(OUTPUT_DIR, exist_ok=True)
    print(f"Output directory: {OUTPUT_DIR}")
    print(f"Total lines to generate: {len(lines)}")
    print()

    results = []
    for i, line in enumerate(lines):
        line_id = line["id"]
        voice_key = line["voice"]
        voice_cfg = voices[voice_key]
        voice_name = voice_cfg["name"]
        model = voice_cfg["model"]
        text = line["text"]

        out_file = os.path.join(OUTPUT_DIR, f"{line_id}.wav")

        # Skip if already generated
        if os.path.exists(out_file) and os.path.getsize(out_file) > 100:
            dur = get_wav_duration(out_file)
            print(
                f"[{i + 1}/{len(lines)}] SKIP {line_id} ({dur:.1f}s) — already exists"
            )
            results.append(
                {"id": line_id, "file": out_file, "duration_ms": int(dur * 1000)}
            )
            continue

        print(
            f"[{i + 1}/{len(lines)}] Generating {line_id} [{voice_name}]: {text[:50]}..."
        )

        pcm_data = generate_tts(api_key, text, voice_name, model)
        if pcm_data is None:
            print(f"  FAILED — skipping")
            continue

        write_wav(out_file, pcm_data)
        dur = len(pcm_data) / (SAMPLE_RATE * SAMPLE_WIDTH * CHANNELS)
        print(f"  OK — {dur:.1f}s, {len(pcm_data)} bytes")

        results.append(
            {"id": line_id, "file": out_file, "duration_ms": int(dur * 1000)}
        )

        if i < len(lines) - 1:
            time.sleep(3)

    # Write results manifest
    manifest_path = os.path.join(OUTPUT_DIR, "manifest.json")
    with open(manifest_path, "w") as f:
        json.dump(results, f, indent=2)
    print(f"\nManifest written: {manifest_path}")
    print(f"Generated {len(results)}/{len(lines)} audio files")


def get_wav_duration(path):
    """Get duration of a WAV file in seconds."""
    try:
        with open(path, "rb") as f:
            f.read(4)  # RIFF
            f.read(4)  # size
            f.read(4)  # WAVE
            f.read(4)  # fmt
            chunk_size = struct.unpack("<I", f.read(4))[0]
            f.read(chunk_size)  # skip fmt data
            f.read(4)  # data
            data_size = struct.unpack("<I", f.read(4))[0]
            return data_size / (SAMPLE_RATE * SAMPLE_WIDTH * CHANNELS)
    except Exception:
        return 0.0


if __name__ == "__main__":
    main()
