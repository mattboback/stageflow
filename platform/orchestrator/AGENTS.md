# AGENTS.md — platform/orchestrator

Go service: core job FSM, Podman pod lifecycle management, cross-scanner report aggregation. Most complex service in the platform.

## STRUCTURE

```
platform/orchestrator/
├── cmd/orchestrator/     # main.go — service bootstrap
├── config/               # Config struct and env loader
├── internal/
│   ├── api/              # Admin HTTP API (jobs, pods, events — port 8081)
│   ├── db/               # PostgreSQL layer (job state, events, scanner results)
│   ├── fsm/              # Job state machine
│   ├── messaging/        # NATS consumers + publishers
│   ├── orchestrator/     # Core logic (largest package)
│   └── podman/           # Podman client (pods, containers, volumes)
└── test/                 # Mock implementations + test harnesses
```

## WHERE TO LOOK

| Task | File |
|------|------|
| Service bootstrap | `cmd/orchestrator/main.go` |
| Job event handlers | `internal/orchestrator/events.go` |
| Pod + extraction start | `internal/orchestrator/extraction.go` |
| Scanner container startup | `internal/orchestrator/scanning.go` |
| Report merge + dedup | `internal/orchestrator/report_aggregator_*.go`, `rule_deduplication.go` |
| Job state transitions | `internal/fsm/state.go` |
| DB schema + helpers | `internal/db/database.go`, `jobs.go`, `job_updates.go` |
| NATS subscriptions | `internal/messaging/consumers.go` |
| NATS publishers | `internal/messaging/publisher.go` |
| Podman abstraction | `internal/podman/client.go` |
| Admin HTTP API | `internal/api/server.go` |
| Mock Podman client | `test/mock_podman_client_test.go` |
| Test helpers + Postgres harness | `test/helpers_test.go`, `test/postgres_test_harness_test.go` |

## CONVENTIONS

- `Publisher` and `PodmanClient` are interfaces — use mocks in tests, never real Podman/NATS.
- Event handlers (`events.go`) call `failJobSafe*` on errors and return `nil` to NATS consumer to prevent redelivery loops.
- Context threading: all DB, Podman, MinIO, and NATS calls receive `context.Context`.
- `backgroundWithCorrelation(ctx)` preserves `logging.RequestID`/`RunID` into goroutines.
- Scanner memory limits from scanner manifests; defaults applied when absent.
- `monitorWG` tracks container monitor goroutines — call `WaitForMonitors()` in tests.
- DB uses pgx stdlib driver via `database/sql`. Schema initialized from embedded `schema.sql`.
- Hand-rolled mocks (no gomock) — `test/mock_podman_client_test.go` is the canonical example.

## KEY TYPES

- `Orchestrator` struct — holds `PodmanClient`, `*db.Database`, publisher, scannerRegistry, timeouts
- `Publisher` interface — mock: `test/mock_publisher_test.go`
- `PodmanClient` interface — mock: `test/mock_podman_client_test.go`

## NOTES

- `extractionTimeout` default 5m; `scanTimeout` default 30m — both in orchestrator Config.
- `shouldIgnoreMissingJob()` silently drops events for missing jobs (no redelivery).
- Admin API runs on port 8081 (separate from public API on 8080).
- Scanner containers share `workspace-{jobID}` volume with extractor container.
- Aggregator reads per-scanner outputs from MinIO; deduplication is cross-scanner (axe/lighthouse/seo overlap).
- Orchestrator mounts Podman socket — intentional, privileged by design. Do not remove.
