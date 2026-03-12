# E2E Test Guide

## OVERVIEW

`tests/e2e/` is a separate Go module for smoke and live-path tests against deployed VibeCat services.

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Baseline gateway smoke | `tests/e2e/e2e_test.go` | `/readyz`, auth, websocket, setup |
| Search and memory paths | `tests/e2e/grounding_and_memory_test.go` | orchestrator + gateway grounded flows |
| Voice interruption path | `tests/e2e/voice_barge_in_test.go` | macOS voice barge-in smoke |
| Legacy companion flow | `tests/e2e/companion_intelligence_test.go` | retained filename; still exercises orchestrator session paths |
| Module boundary | `tests/e2e/go.mod` | standalone `vibecat/e2e` module |

## CONVENTIONS

- Tests are env-driven and skip when required URLs or auth context are missing.
- Default target is deployed infrastructure, not hermetic local mocks.
- Gateway tests use `GATEWAY_URL`; orchestrator tests obtain identity tokens and hit deployed endpoints.
- Orchestrator-real coverage needs `ORCHESTRATOR_URL` plus a working `gcloud auth print-identity-token` path.
- Keep happy-path smoke coverage separate from long-running live-path checks.

## ANTI-PATTERNS

- hardcoding secrets or tokens into tests
- rewriting live tests into pure mocks just to make CI greener
- assuming local `localhost` behavior matches deployed Cloud Run behavior
- removing skip guards around environment-dependent tests

## COMMANDS

```bash
cd tests/e2e && go test -v -count=1 ./...
cd tests/e2e && GATEWAY_URL=http://localhost:8080 go test -v -count=1 ./...
cd tests/e2e && GATEWAY_URL=https://realtime-gateway-....run.app go test -v -count=1 ./...
```
