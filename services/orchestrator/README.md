# Orchestrator

The Orchestrator is the event-driven core of StageFlow. It turns scan-job
lifecycle events into real work and durable state.

It is responsible for:

- consuming job lifecycle events from NATS JetStream (`job.created`, `extraction.ready`, scan completion/failure)
- driving the job state machine (`PENDING → EXTRACTING → READY_TO_SCAN → SCANNING → COMPLETING → DONE/FAILED`)
- launching and tearing down a per-job rootless Podman pod (extractor + scanner-runner containers)
- aggregating raw scanner artifacts into a single unified report and storing it in MinIO
- persisting job state and an append-only event audit trail to PostgreSQL
- exposing a small internal admin API (job state, metrics) consumed by the Platform API

It is not responsible for:

- terminating public client traffic (that is the Platform API)
- running scanners itself (the Scanner Runner does, inside the job pod)
- extracting uploaded archives (the Archive Extractor does, inside the job pod)

## Process wiring

Start at `cmd/orchestrator/main.go`. The boot sequence is:

1. load and validate config (`cmd/orchestrator/config.go`)
2. load the scanner registry, applying optional `SCANNER_CONFIG_PATH` overrides
3. connect to MinIO and NATS (ensuring streams exist)
4. open the PostgreSQL database and, if enabled, start the job-events retention pruner
5. create the Podman client
6. construct the `orchestrator.Orchestrator` with all adapters wired in
7. start the NATS consumer and the orchestrator background loops
8. start the internal admin API server
9. block on `SIGINT`/`SIGTERM`, then drain and shut down

## Architecture

The service follows a hexagonal layout:

| Layer | Path | Responsibility |
| --- | --- | --- |
| Domain | `internal/domain/jobs/` | Pure FSM rules: transitions, completion, failure, and extraction-ready policies |
| Application | `internal/application/jobs/` | Use cases (`handle_*` event handlers), ports/interfaces, scanner launch planning |
| Adapters — repository | `internal/adapters/repository/` | PostgreSQL persistence for jobs, scanners, updates, events, and metrics |
| Adapters — runtime | `internal/adapters/runtime/` | Podman client: pods, containers, volumes, job-scoped runtime |
| Adapters — storage | `internal/adapters/storage/` | Report aggregation, rule deduplication, and artifact upload to MinIO |
| Adapters — messaging | `internal/adapters/messaging/` | NATS consumer and publisher |
| Coordination | `internal/orchestrator/` | The long-running loop tying handlers, runtime, deadlines, and cleanup together |
| Admin API | `internal/api/` | Internal HTTP server (job state, metrics) for the Platform API |

Ports are declared as interfaces in `internal/application/jobs/ports.go`, so the
domain and use cases never import Podman, NATS, or SQL directly.

## External dependencies

- **NATS JetStream** — source of lifecycle events and sink for completion/failure events
- **PostgreSQL** — durable job state and event audit trail
- **Podman** (rootless) — per-job pod lifecycle
- **MinIO** — staging input and final report/artifact storage
- **Scanner registry** — which scanners are enabled and how their config is validated

## Local run and test commands

From `services/orchestrator/`:

```bash
go run ./cmd/orchestrator
go test ./...
go test -race ./...
```

From a fresh checkout, run `just generate-contracts` from the repo root before
running Go commands directly inside this module.

Integration tests that touch PostgreSQL use an embedded harness
(`internal/orchtest/postgres.go`); no external database is required.

## Files to inspect first

- `cmd/orchestrator/main.go` — startup sequence and dependency graph
- `internal/orchestrator/orchestrator.go` — the coordination loop
- `internal/domain/jobs/transitions.go` — FSM transition rules
- `internal/application/jobs/handle_job_created.go` — entry point for new jobs
- `internal/adapters/runtime/pods.go` — per-job Podman pod lifecycle
- `internal/adapters/storage/report_aggregator.go` — how scanner output becomes one report
