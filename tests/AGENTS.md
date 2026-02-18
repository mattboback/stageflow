# AGENTS.md — tests

Go e2e integration tests against a running StageFlow stack. Not unit tests.

## STRUCTURE

```
tests/
├── e2e/
│   ├── url_scan_test.go   # Full URL scan: submit → SSE stream → result validation
│   ├── zip_scan_test.go   # Full ZIP upload scan job
│   └── helpers_test.go    # Shared setup + API/SSE client helpers
└── fixtures/
    └── simple-site/       # Static HTML site used by zip_scan_test.go
```

## CONVENTIONS

- Tests require a **running stack**: `just dev up` + `just dev init` + `just images`.
- Configure via env: `PLATFORM_API_BASE_URL` (default `http://localhost:8080`).
- Go `testing` package; run with `go test -race ./...` from `tests/e2e/`.
- `go.work` participant — `just ci` runs them automatically.
- Tests stream SSE to verify job completion (not just polling status).

## NOTES

- Failures here usually indicate infrastructure issues (NATS, MinIO, Podman, DB), not unit logic.
- `tools/suite-runner` is the more flexible integration runner for regression suites (YAML + threshold evaluation).
- `zip_scan_test.go` creates a ZIP from `fixtures/simple-site/` at test time.
