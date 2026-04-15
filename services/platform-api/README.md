# Platform API

The Platform API is the public HTTP boundary for StageFlow.

It is responsible for:

- accepting URL and ZIP scan submissions
- validating targets, scanner selections, and scanner config payloads
- exposing job status, report, results, and diff endpoints
- streaming live job updates over SSE
- serving project CRUD and baseline-promotion workflows
- listing scanner capabilities to web and CLI clients

It is not responsible for:

- running scanners directly
- managing Podman job pods
- aggregating raw scanner artifacts into final job state
- storing long-term orchestration state in PostgreSQL

Those responsibilities belong to the Orchestrator, Archive Extractor, Scanner Runner, and shared infrastructure services.

## Process wiring

Start at `cmd/server/main.go`.

The process boot sequence is:

1. load and validate config from `cmd/server/config.go`
2. validate the hardcoded SSRF/security policy in `internal/api/security.go`
3. connect to NATS and ensure streams exist
4. connect to MinIO and ensure buckets exist
5. build the orchestrator status-source client
6. open the SQLite project store
7. load the scanner registry and optional overrides
8. construct `api.Server`
9. subscribe the in-memory job-status pipeline to lifecycle events
10. start the HTTP server with long-lived SSE support

## Public routes

`internal/api/router.go` registers the public surface:

| Route                                     | Purpose                                           |
| ----------------------------------------- | ------------------------------------------------- |
| `POST /api/v1/jobs/urls`                  | Submit a URL-based scan job                       |
| `POST /api/v1/jobs/zip`                   | Submit a ZIP upload job                           |
| `GET /api/v1/jobs/:id`                    | Current job status snapshot                       |
| `GET /api/v1/jobs/:id/stream`             | SSE stream for live updates                       |
| `GET /api/v1/jobs/:id/report`             | Redirect to the HTML report artifact              |
| `GET /api/v1/jobs/:id/results`            | Redirect to the normalized JSON report artifact   |
| `GET /api/v1/jobs/:id/diff`               | Diff a project scan against its promoted baseline |
| `GET/POST /api/v1/projects`               | List or create named projects                     |
| `GET/PATCH/DELETE /api/v1/projects/:slug` | Inspect, update, or delete a project              |
| `POST /api/v1/projects/:slug/scan`        | Launch a scan from stored project config          |
| `POST /api/v1/projects/:slug/promote`     | Promote a completed scan to the project baseline  |
| `GET /api/v1/scanners`                    | List enabled scanners and capabilities            |
| `GET /healthz`                            | Liveness check                                    |

## Middleware and boundary rules

Most request paths run through:

1. request logging
2. CORS
3. API key auth
4. rate limiting
5. per-request timeout

Uploads use a longer timeout. SSE intentionally skips the write-timeout wrapper so long-lived streams can stay open.

Security-sensitive behavior worth inspecting:

- `internal/api/security.go` blocks non-HTTP schemes and private/metadata IP ranges unless private-target mode is explicitly enabled
- `internal/api/handlers_jobs_url_submit.go` limits body size, URL count, and URL length before publishing a job-created event
- `internal/api/handlers_jobs_zip_upload.go` constrains multipart upload size and validates required form parts before enqueueing work
- `internal/api/scanner_configs.go` validates per-scanner config payloads against the registry
- `internal/api/object_keys.go` and the job artifact handlers only presign job-scoped object keys

## Key internal packages

| Path                     | Responsibility                                                  |
| ------------------------ | --------------------------------------------------------------- |
| `internal/api/`          | HTTP handlers, middleware, SSRF checks, SSE, artifact redirects |
| `internal/jobstatus/`    | In-memory projection pipeline fed from lifecycle events         |
| `internal/project/`      | SQLite-backed project records and baseline metadata             |
| `internal/status/`       | Persistent status schema and storage helpers                    |
| `internal/statussource/` | Client used to read orchestrator job state                      |
| `internal/messaging/`    | NATS publishing/subscription service wiring                     |

## External dependencies

The service sits between clients and backend infrastructure:

- **Web UI / CLI** call this service directly
- **NATS JetStream** receives `job.created` and other lifecycle events
- **Orchestrator Admin API** provides current job state used by the status projection
- **MinIO** stores report and screenshot artifacts that this API presigns
- **SQLite** stores named projects and promoted baselines
- **Scanner registry** defines which scanners/modules are allowed and how config is validated

## Local run and test commands

From `services/platform-api/`:

```bash
go run ./cmd/server
```

Typical verification commands:

```bash
go test ./...
go test -race ./...
```

When working from the repo root, the shared commands used in CI are:

```bash
bash devtools/scripts/go/run-in-work-dirs.sh Building go build ./...
bash devtools/scripts/go/run-in-work-dirs.sh Testing go test -race ./...
```

## Tests worth inspecting

- `internal/api/handlers_sse_test.go` and `handlers_sse_wire_test.go` — stream behavior, terminal events, and wire format
- `internal/api/security_test.go`, `security_dns_test.go`, `security_resolver_test.go` — SSRF and hostname resolution policy
- `internal/api/handlers_jobs_modules_test.go` — module normalization and validation paths
- `internal/api/middleware_test.go` — middleware ordering and behavior
- `internal/jobstatus/pipeline_test.go` — event projection logic
- `tests/integration/messaging_nats_test.go` — integration coverage for NATS-driven lifecycle updates

## Files to inspect first

If you want the shortest path to understanding the service, start here:

- `cmd/server/main.go` — startup sequence and dependency graph
- `internal/api/router.go` — public route map
- `internal/api/handlers_jobs_url_submit.go` — URL intake contract and event publication
- `internal/api/handlers_jobs_zip_upload.go` — multipart ZIP intake path
- `internal/api/security.go` — SSRF policy implementation
- `internal/api/handlers_sse.go` — browser/CLI live stream behavior
- `internal/project/store.go` — project persistence and baseline support
