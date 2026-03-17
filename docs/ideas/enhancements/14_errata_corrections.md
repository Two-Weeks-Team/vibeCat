# 14: Validation Ledger and Resolved Errata

## Purpose

This file is no longer a loose correction dump. It is the validation ledger for the enhancement set after source review on 2026-03-17.

## Resolved Corrections

### 01 Thinking

- Corrected: the current Live model does support thinking
- Updated instruction: implement Live thinking budget first; gate thought-summary UI

### 02 Context Caching

- Corrected request field: `GenerateContentConfig.CachedContent`
- Corrected threshold assumption: use token-count precheck; do not assume 2,048+
- Corrected scope: batch orchestrator only

### 03 Forced FC

- Confirmed: valid only for direct batch `GenerateContent` routes
- Clarified: not part of `LiveConnectConfig`

### 04 Parallel FC

- Reframed from optimization to correctness fix
- Current priority is serializing multiple Live calls safely

### 05 Safety

- Corrected: safety flow already exists in backend and client
- Current work is classifier hardening and UX improvement

### 06 Heartbeat

- Corrected: keepalive already exists in gateway handler
- Current work is session-owned refactor and idle-aware tuning

### 07 Controlled Generation

- Updated recommendation: prefer `application/json` plus schema-based validation on direct calls
- Avoid depending on older enum-only patterns as the primary path

### 08 Computer Use

- Confirmed: browser-only and still deferred

### 09 Always-On Memory

- Corrected: Cloud Run timer pattern is invalid
- Required architecture: Scheduler + Tasks + HTTP consolidation job

### 10 MCP

- Corrected: production Cloud Run cannot rely on `npx`
- Added boundary: filesystem MCP in cloud cannot access user-local files

### 11 RAG

- Corrected: backend cannot directly walk arbitrary local desktop workspaces
- Required architecture: upload/sync for local docs, Firestore vector search for retrieval

### 12 Navigator Tools

- Corrected gap statement: Swift already supports several missing actions
- Actual gap is FC exposure plus parser/data-contract completeness

### 13 Progress Communication

- Confirmed: most of the architecture already exists
- Remaining work is protocol completeness, especially total-step metadata and improved confirmation UI

## Current Truth Source

When any feature doc conflicts with current implementation reality, trust:

1. the repo code
2. official docs
3. this ledger
4. older draft language last

## Primary Sources

- [Gemini Live API guide](https://ai.google.dev/gemini-api/docs/live-guide)
- [Gemini thinking guide](https://ai.google.dev/gemini-api/docs/thinking)
- [Gemini function calling guide](https://ai.google.dev/gemini-api/docs/function-calling)
- [Gemini caching guide](https://ai.google.dev/gemini-api/docs/caching)
- [Gemini structured output guide](https://ai.google.dev/gemini-api/docs/structured-output)
- [Gemini computer use guide](https://ai.google.dev/gemini-api/docs/computer-use)
- [Gemini deprecations](https://ai.google.dev/gemini-api/docs/deprecations)
- [Firestore vector search](https://cloud.google.com/firestore/native/docs/vector-search)
