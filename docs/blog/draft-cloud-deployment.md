---
title: from localhost to cloud run: deploying a live pm plus action worker
published: false
description: how VibeCat's two-service Cloud Run architecture supports a Gemini Live PM, a single-task action worker, and the observability needed to trust a desktop UI navigator
tags: geminiliveagentchallenge, devlog, buildinpublic, go
cover_image:
---

I created this post for the purposes of entering the Gemini Live Agent Challenge.

---

there's a specific kind of confidence you get when something works on your laptop. the logs are clean, the WebSocket connects, the cat sprite blinks at you from the menu bar. then you push it to Cloud Run and spend the next two hours staring at a 503.

this is the story of getting VibeCat — now a macOS desktop UI navigator with a Live PM and a single-task action worker — from `go run .` to two live Cloud Run services in `asia-northeast3`. it covers the deployment script, the observability stack, the CI pipeline, and one specific lesson about health checks that I learned the hard way on a previous project called missless.

source: [github.com/Two-Weeks-Team/vibeCat](https://github.com/Two-Weeks-Team/vibeCat)

---

## the two-service split

VibeCat's backend is deliberately split into two Cloud Run services. this wasn't an aesthetic choice — the challenge rules require using GenAI SDK, ADK, Gemini Live API, and VAD together, and the Live API's WebSocket model doesn't compose cleanly with ADK's agent graph execution model.

**realtime-gateway** handles everything real-time: the WebSocket connection from the macOS client, the Gemini Live API session (voice, VAD, barge-in), JWT auth, and TTS. it needs to stay alive for the duration of a user session.

**adk-orchestrator** handles the slower intelligence lane: contextual analysis, research, memory-adjacent logic, and supporting signals that can enrich the navigator without owning the real-time execution loop.

the gateway calls the orchestrator over HTTP (`POST /analyze`) whenever it needs to analyze a screen capture. the orchestrator is internal-only — no public traffic, IAM-protected.

the deploy script captures this relationship explicitly:

```bash
PROJECT_ID="${GCP_PROJECT:-vibecat-489105}"
REGION="${GCP_REGION:-asia-northeast3}"
REGISTRY="${REGION}-docker.pkg.dev/${PROJECT_ID}/vibecat-images"
GATEWAY_IMAGE="${REGISTRY}/realtime-gateway"
ORCHESTRATOR_IMAGE="${REGISTRY}/adk-orchestrator"
```

orchestrator deploys first, then the gateway gets the orchestrator's URL injected as an environment variable:

```bash
ORCHESTRATOR_URL=$(gcloud run services describe adk-orchestrator \
  --region "${REGION}" \
  --project "${PROJECT_ID}" \
  --format "value(status.url)")

gcloud run deploy realtime-gateway \
  --set-env-vars "ADK_ORCHESTRATOR_URL=${ORCHESTRATOR_URL}" \
  ...
```

this means the gateway never has a hardcoded orchestrator URL. if you redeploy the orchestrator and it gets a new URL (which Cloud Run does sometimes), you just re-run `deploy.sh` and the gateway picks it up.

---

## the secret manager setup

one of the non-negotiables for this project was zero client-side API keys. the Gemini API key lives in GCP Secret Manager as `vibecat-gemini-api-key` and gets injected at deploy time:

```bash
gcloud run deploy adk-orchestrator \
  --no-allow-unauthenticated \
  --set-secrets "GEMINI_API_KEY=vibecat-gemini-api-key:latest" \
  ...

gcloud run deploy realtime-gateway \
  --allow-unauthenticated \
  --set-secrets "GEMINI_API_KEY=vibecat-gemini-api-key:latest,GATEWAY_AUTH_SECRET=vibecat-gateway-auth-secret:latest" \
  ...
```

the gateway is public-facing (clients need to connect to it), but the orchestrator is locked down with `--no-allow-unauthenticated`. the last step of the deploy script grants the gateway's service account the `roles/run.invoker` role on the orchestrator:

```bash
gcloud run services add-iam-policy-binding adk-orchestrator \
  --member="serviceAccount:${COMPUTE_SA}" \
  --role="roles/run.invoker" \
  --region="${REGION}" \
  --project="${PROJECT_ID}"
```

the macOS client never sees an API key. it registers with the gateway, gets a short-lived JWT, and uses that for the WebSocket connection. the gateway handles everything else.

---

## the container

the Dockerfile for the gateway is about as minimal as it gets:

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /realtime-gateway .

FROM gcr.io/distroless/static-debian12
COPY --from=builder /realtime-gateway /realtime-gateway
EXPOSE 8080
ENTRYPOINT ["/realtime-gateway"]
```

two-stage build, distroless final image. `CGO_ENABLED=0` because we're targeting a static binary for a container that has no libc. the final image is around 12MB. the orchestrator Dockerfile follows the same pattern.

one thing worth noting: the gateway deploy uses `--no-use-http2` and `--session-affinity`. WebSocket connections over Cloud Run need HTTP/1.1 (HTTP/2 multiplexing breaks the upgrade handshake in ways that are annoying to debug), and session affinity ensures a client's WebSocket stays on the same instance for the duration of the session.

---

## observability: three layers

this is where it gets interesting. VibeCat uses three separate observability systems, all initialized at startup.

**Cloud Trace** — distributed tracing via OpenTelemetry. both services initialize a trace exporter:

```go
// realtime-gateway/main.go
traceExporter, traceErr := texporter.New(texporter.WithProjectID(projectID))
if traceErr != nil {
    slog.Warn("cloud trace init failed — tracing disabled", "error", traceErr)
} else {
    tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(traceExporter))
    otel.SetTracerProvider(tp)
    defer tp.Shutdown(context.Background())
    slog.Info("cloud trace initialized", "project", projectID)
}
```

the orchestrator creates spans around every analyze request:

```go
// adk-orchestrator/main.go
tracer := otel.Tracer("vibecat/orchestrator")
_, span := tracer.Start(r.Context(), "orchestrator.analyze")
defer span.End()
```

this means you can see the full trace from the gateway's WebSocket handler through to the orchestrator's agent graph execution in Cloud Trace. when something is slow, you can see exactly which agent is the bottleneck.

**Cloud Monitoring** — custom metrics. the orchestrator registers three OTel instruments:

```go
meter := otel.Meter("vibecat/orchestrator")
analyzeCounter, _ := meter.Int64Counter("vibecat.analyze.requests",
    metric.WithDescription("Total analyze requests"),
)
analyzeDurHist, _ := meter.Float64Histogram("vibecat.analyze.duration_ms",
    metric.WithDescription("Analyze request duration in milliseconds"),
)
errorCounter, _ := meter.Int64Counter("vibecat.analyze.errors",
    metric.WithDescription("Total analyze errors"),
)
```

`vibecat.analyze.requests` is a counter — total analyze calls since startup. `vibecat.analyze.duration_ms` is a histogram — you get p50/p95/p99 latency for the full agent graph execution. `vibecat.analyze.errors` counts cases where the agent graph produced no usable result.

the histogram is the one I actually watch. the 9-agent graph runs in three waves (Vision+Memory in parallel, then Mood+Celebration in parallel, then a sequential chain through Mediator→Scheduler→Engagement→Search), and the p95 latency tells you whether the parallel waves are actually helping.

the metric exporter uses a periodic reader:

```go
metricExporter, metricErr := mexporter.New(mexporter.WithProjectID(projectID))
if metricErr != nil {
    slog.Warn("cloud monitoring init failed — metrics disabled", "error", metricErr)
} else {
    mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)))
    otel.SetMeterProvider(mp)
    defer mp.Shutdown(ctx)
}
```

**Cloud Logging** — structured JSON logs via `log/slog`. both services initialize with `slog.NewJSONHandler(os.Stdout, nil)`, which Cloud Run's log collector picks up and forwards to Cloud Logging automatically. the orchestrator also initializes a Cloud Logging client directly for cases where you want to write structured log entries with explicit severity and labels.

**ADK Telemetry** — the orchestrator also initializes ADK's built-in telemetry, which hooks into the same OTel providers:

```go
adkTelemetry, telErr := telemetry.New(ctx,
    telemetry.WithGcpResourceProject(projectID),
)
if telErr != nil {
    slog.Warn("adk telemetry init failed", "error", telErr)
} else {
    adkTelemetry.SetGlobalOtelProviders()
    defer adkTelemetry.Shutdown(ctx)
}
```

this gives you ADK-level spans for free — you can see individual agent invocations, tool calls, and LLM requests in Cloud Trace without instrumenting anything manually.

the pattern across all three is the same: try to initialize, warn and continue if it fails. Cloud Run services should start even if observability is broken. a service that refuses to start because it can't connect to Cloud Monitoring is worse than a service that runs without metrics.

---

## the /readyz lesson

if you read "the websocket cascade from hell" — the post about debugging missless's WebSocket reconnection loop — you know that Cloud Run's health check behavior caused a significant chunk of that incident. the short version: Cloud Run uses `/` as the default health check path if you don't configure one, and if your service returns anything other than 2xx on `/`, Cloud Run marks the instance as unhealthy and kills it. during a deploy, this can cause a cascade where new instances spin up, fail the health check, get killed, and the old instances are already gone.

VibeCat has explicit `/health` and `/readyz` endpoints on both services. the gateway's `/health` includes the active WebSocket connection count:

```go
func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    response := map[string]any{
        "status":      "ok",
        "service":     serviceName,
        "connections": registry.Count(),
    }
    json.NewEncoder(w).Encode(response)
}
```

`/readyz` is separate — it's what Cloud Run uses for the readiness probe. the distinction matters: `/health` tells you if the process is alive, `/readyz` tells you if it's ready to serve traffic. for the gateway, readiness means the Gemini Live manager is initialized. for the orchestrator, it means the ADK runner is built and the agent graph is wired up.

the deploy script doesn't configure the health check path explicitly (Cloud Run defaults to `/` for liveness), but both services return 404 on `/` which... is fine actually, because Cloud Run's default liveness check is TCP-based, not HTTP. the readiness check is what matters, and both services respond 200 on `/readyz` as soon as they're up.

the lesson from missless wasn't "add health checks" — it was "understand what Cloud Run is actually checking and when." the cascade happened because we didn't know Cloud Run was doing HTTP health checks against `/` during rolling deploys. once you know that, the fix is obvious. but you have to know it first.

---

## the CI pipeline

four jobs, all independent, all run in parallel on every push to master and every PR:

```yaml
jobs:
  go-gateway:
    name: Gateway (Go) — Build + Test + Vet
    runs-on: ubuntu-latest
    timeout-minutes: 10
    steps:
      - name: Test with coverage
        run: go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
        working-directory: backend/realtime-gateway

  go-orchestrator:
    name: Orchestrator (Go) — Build + Test + Vet
    runs-on: ubuntu-latest
    timeout-minutes: 10
    steps:
      - name: Test with coverage
        run: go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
        working-directory: backend/adk-orchestrator

  swift:
    name: Client (Swift 6 / macOS) — Build + Test
    runs-on: [self-hosted, macOS, ARM64]
    timeout-minutes: 10

  docker:
    name: Docker — Build images
    runs-on: ubuntu-latest
    timeout-minutes: 15
    steps:
      - name: Build Gateway image
        run: docker build -t vibecat-gateway backend/realtime-gateway/
      - name: Build Orchestrator image
        run: docker build -t vibecat-orchestrator backend/adk-orchestrator/
```

the Go jobs run with `-race` flag. the race detector has caught two actual bugs during development — both in the WebSocket registry's connection map. the Swift job runs on a self-hosted macOS ARM64 runner because GitHub's hosted macOS runners are slow and expensive for a hackathon project.

the Docker job doesn't push to Artifact Registry — it just verifies the images build. actual deployment is manual via `./infra/deploy.sh`. for a hackathon, that's the right call. automated deploys on every push to master would be nice but it's not worth the Cloud Build cost or the complexity of managing GCP credentials in GitHub Actions secrets.

coverage artifacts get uploaded on every run, even if tests fail (`if: always()`). this means you can look at coverage even when a test is broken, which is useful when you're trying to figure out whether a failing test is actually testing the thing you think it's testing.

---

## the ADK runner setup

the orchestrator's ADK setup is worth looking at in detail because it uses a few features that aren't obvious from the docs:

```go
sessService := session.InMemoryService()
memService := memory.InMemoryService()
retryPlugin := retryandreflect.MustNew(
    retryandreflect.WithMaxRetries(3),
    retryandreflect.WithTrackingScope(retryandreflect.Invocation),
)
r, err := runner.New(runner.Config{
    AppName:        "vibecat",
    Agent:          agentGraph,
    SessionService: sessService,
    MemoryService:  memService,
    PluginConfig: runner.PluginConfig{
        Plugins: []*plugin.Plugin{retryPlugin},
    },
})
```

`retryandreflect` is an ADK plugin that automatically retries failed agent invocations and reflects on why they failed. `WithTrackingScope(retryandreflect.Invocation)` means it tracks retries at the invocation level — if the VisionAgent fails, it retries VisionAgent specifically, not the entire graph. `WithMaxRetries(3)` means it'll try three times before giving up and returning an error.

this matters because Gemini API calls can fail transiently. without retry logic, a single 429 or 503 from the API would cause the entire analyze request to fail. with `retryandreflect`, transient failures are handled automatically.

the session service is in-memory for now. the MemoryAgent writes cross-session context to Firestore directly, but the ADK session state (which tracks things like `activity_minutes` and `language` within a single analyze call) lives in memory. for a Cloud Run service with `--min-instances 0`, this means session state doesn't survive instance restarts — but that's acceptable because each analyze call is stateless from the orchestrator's perspective. the gateway maintains the actual session continuity.

---

## current state

gateway is on revision `00010-m9p`, orchestrator on `00011-qj4`. both are running in `asia-northeast3` with `--min-instances 0` (cold starts are acceptable for a hackathon) and `--max-instances 3`.

the full deploy takes about 4 minutes: two Cloud Build jobs running sequentially (gateway then orchestrator), then two `gcloud run deploy` calls. it's not fast, but it's reliable. `set -euo pipefail` at the top of the deploy script means any failure stops the whole thing — no partial deploys where the gateway is updated but the orchestrator isn't.

the thing I'm most happy with is the observability setup. having Cloud Trace, Cloud Monitoring, and Cloud Logging all initialized from the first line of `main()` means that when something goes wrong in production, I have actual data to look at. the histogram for `vibecat.analyze.duration_ms` has already told me that the parallel wave execution (Vision+Memory running concurrently) is saving about 800ms per analyze call compared to running them sequentially. that's the kind of thing you can only know if you're measuring it.

---

*VibeCat is built for the Gemini Live Agent Challenge 2026. source at [github.com/Two-Weeks-Team/vibeCat](https://github.com/Two-Weeks-Team/vibeCat).*
