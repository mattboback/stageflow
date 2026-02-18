# AGENTS.md — platform/api

Go service: public REST API + SSE streaming. Intake for URL/ZIP jobs, SSRF validation, broadcasts lifecycle events to frontend clients.

## STRUCTURE

```
platform/api/
├── cmd/
│   ├── server/         # main.go + config.go
│   └── healthcheck/    # Small health probe CLI
├── internal/
│   ├── api/            # HTTP handlers, router, security (largest package)
│   ├── messaging/      # NATS service + SSE broadcast wiring
│   ├── sse/            # SSE Hub (fan-out to connected clients)
│   ├── status/         # Job status store + DB queries
│   └── statussource/   # Client to orchestrator admin API (port 8081)
└── tests/integration/
```

## WHERE TO LOOK

| Task | File |
|------|------|
| Route registration | `internal/api/router.go` |
| URL job intake | `internal/api/handlers_jobs_url_submit.go` |
| ZIP upload intake | `internal/api/handlers_jobs_zip_upload.go` |
| SSE stream handler | `internal/api/handlers_sse.go` |
| SSRF/URL validation | `internal/api/security.go` |
| SSE hub (broadcast) | `internal/sse/hub.go` |
| NATS subscription + SSE | `internal/messaging/service.go` |
| Job status queries | `internal/status/store.go` |
| Service bootstrap | `cmd/server/main.go` |

## CONVENTIONS

- `JobPublisher` and `JobStatusReader` are interfaces — inject mocks in tests.
- `WriteTimeout = 0` on HTTP server is **intentional** for SSE — per-handler timeouts via middleware only.
- URL validation limits: 2 MB body, max 100 URLs, `http`/`https` only, SSRF policy enforced.
- ZIP upload limit: 100 MB max. Files staged to MinIO; orchestrator notified via NATS.
- All error responses via `httputil.RespondStructuredError` (shared-go/httputil).

## NOTES

- **SSRF policy** blocks: loopback (127.x, ::1), private (10/8, 172.16/12, 192.168/16), link-local (169.254/16), cloud metadata (169.254.169.254).
- API publishes `JobCreated` to NATS; orchestrator consumes and drives lifecycle.
- SSE hub fan-outs all lifecycle events received from NATS to watching clients.
- Status reads proxy to orchestrator admin API via `statussource` client.
- Runs on port 8080. Admin API (orchestrator) is on 8081 — do not conflate.
