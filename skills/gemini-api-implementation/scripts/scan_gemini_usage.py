#!/usr/bin/env python3
"""Scan a repo for Gemini SDK usage, model IDs, and Live/tool configuration."""

from __future__ import annotations

import argparse
import json
import re
import sys
from dataclasses import dataclass, asdict
from pathlib import Path

CODE_EXTENSIONS = {
    ".c",
    ".cc",
    ".cpp",
    ".cs",
    ".go",
    ".h",
    ".hpp",
    ".java",
    ".js",
    ".json",
    ".jsx",
    ".kt",
    ".kts",
    ".m",
    ".mjs",
    ".mm",
    ".py",
    ".rb",
    ".rs",
    ".sh",
    ".sql",
    ".swift",
    ".toml",
    ".ts",
    ".tsx",
    ".yaml",
    ".yml",
}

DOC_EXTENSIONS = {
    ".md",
    ".txt",
}

SKIP_DIRS = {
    ".build",
    ".git",
    ".idea",
    ".next",
    ".sisyphus",
    ".swiftpm",
    "Assets",
    "DerivedData",
    "Pods",
    "build",
    "dist",
    "node_modules",
    "out",
    "skills",
    "vendor",
    "voice_samples",
}

IMPORT_PATTERNS = {
    "go-genai": re.compile(r"google\.golang\.org/genai"),
    "go-adk": re.compile(r"google\.golang\.org/adk"),
    "js-genai": re.compile(r"@google/genai"),
    "js-generative-ai-legacy": re.compile(r"@google/generative-ai"),
    "python-genai": re.compile(r"(from\s+google\s+import\s+genai|import\s+google\.genai|from\s+google\.genai)"),
}

MODEL_RE = re.compile(r"\bgemini-[a-z0-9][a-z0-9.\-]*\b")

CATEGORY_PATTERNS = {
    "live_api": re.compile(
        r"(Live\.Connect|\.live\.connect|SendRealtimeInput|BidiGenerateContent|responseModalities|response_modalities|"
        r"AutomaticActivityDetection|automaticActivityDetection|session_resumption|SessionResumption|"
        r"context_window_compression|ContextWindowCompression|audio/pcm;rate=|output_audio_transcription|"
        r"input_audio_transcription|enable_affective_dialog|enableAffectiveDialog|proactive_audio|proactiveAudio)"
    ),
    "built_in_tools": re.compile(
        r"(google_search_retrieval|google_search|GoogleSearch|googleMaps|google_maps|GoogleMaps|"
        r"codeExecution|code_execution|urlContext|url_context|file_search|File Search|computerUse|computer_use)"
    ),
    "auth_and_tokens": re.compile(
        r"(ephemeral|authTokens\.create|auth_tokens\.create|Secret Manager|secret manager|x-goog-api-key|Authorization)"
    ),
}


@dataclass(frozen=True)
class Match:
    path: str
    line: int
    text: str


def should_scan(path: Path, include_docs: bool) -> bool:
    suffix = path.suffix.lower()
    if suffix not in CODE_EXTENSIONS and not (include_docs and suffix in DOC_EXTENSIONS):
        return False
    parts = set(path.parts)
    if not include_docs and "docs" in parts:
        return False
    return True


def iter_files(root: Path, include_docs: bool):
    for path in root.rglob("*"):
        if not path.is_file():
            continue
        if any(part in SKIP_DIRS for part in path.parts):
            continue
        if should_scan(path, include_docs):
            yield path


def add_match(bucket: dict[str, list[Match]], key: str, path: Path, line_number: int, line: str) -> None:
    text = line.strip()
    match = Match(path=str(path), line=line_number, text=text)
    items = bucket.setdefault(key, [])
    if match not in items:
        items.append(match)


def is_probable_model_id(value: str) -> bool:
    return any(ch.isdigit() for ch in value)


def scan(root: Path, include_docs: bool) -> dict[str, object]:
    imports: dict[str, list[Match]] = {}
    models: dict[str, list[Match]] = {}
    categories: dict[str, list[Match]] = {}
    scanned_files = 0

    for path in iter_files(root, include_docs):
        scanned_files += 1
        try:
            content = path.read_text(encoding="utf-8", errors="ignore")
        except OSError:
            continue

        rel_path = path.relative_to(root)
        for line_number, line in enumerate(content.splitlines(), start=1):
            for name, pattern in IMPORT_PATTERNS.items():
                if pattern.search(line):
                    add_match(imports, name, rel_path, line_number, line)

            for model in MODEL_RE.findall(line):
                if is_probable_model_id(model):
                    add_match(models, model, rel_path, line_number, line)

            for name, pattern in CATEGORY_PATTERNS.items():
                if pattern.search(line):
                    add_match(categories, name, rel_path, line_number, line)

    return {
        "root": str(root),
        "scanned_files": scanned_files,
        "imports": {k: [asdict(m) for m in sorted(v, key=lambda m: (m.path, m.line))] for k, v in sorted(imports.items())},
        "models": {k: [asdict(m) for m in sorted(v, key=lambda m: (m.path, m.line))] for k, v in sorted(models.items())},
        "categories": {k: [asdict(m) for m in sorted(v, key=lambda m: (m.path, m.line))] for k, v in sorted(categories.items())},
    }


def print_markdown(data: dict[str, object]) -> None:
    print("# Gemini Usage Scan")
    print()
    print(f"- Root: `{data['root']}`")
    print(f"- Files scanned: `{data['scanned_files']}`")
    print()

    for section_name, title in (
        ("imports", "SDK Imports"),
        ("models", "Model IDs"),
        ("categories", "Config and Feature Signals"),
    ):
        section = data[section_name]
        print(f"## {title}")
        print()
        if not section:
            print("_No matches found._")
            print()
            continue
        for key, matches in section.items():
            print(f"### `{key}`")
            print()
            for match in matches:
                print(f"- `{match['path']}:{match['line']}` {match['text']}")
            print()


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("root", nargs="?", default=".", help="Repo root to scan")
    parser.add_argument("--include-docs", action="store_true", help="Also scan docs/")
    parser.add_argument("--json", action="store_true", help="Emit JSON instead of Markdown")
    args = parser.parse_args()

    root = Path(args.root).resolve()
    if not root.exists() or not root.is_dir():
        print(f"error: not a directory: {root}", file=sys.stderr)
        return 2

    data = scan(root, args.include_docs)
    if args.json:
        json.dump(data, sys.stdout, indent=2)
        sys.stdout.write("\n")
    else:
        print_markdown(data)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
