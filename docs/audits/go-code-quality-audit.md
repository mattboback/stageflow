# StageFlow Go Code Quality, Structure, Tests, and Error Handling Audit

> **Scope:** Go code only. `services/scanner-runner` (TypeScript) and `clients/web` (TypeScript/Svelte) are excluded — they are not in `go.work` and were not inspected at the Go source level. The `infra/`, `services/scanner-runner/`, and `clients/web/` directories are not Go code.
>
> **Mode:** Read-only. All findings reference `path:line`. Snippets are quoted verbatim where used. Unverifiable items are marked **UNKNOWN**.

## Repository inventory

```
352 Go files  /  63,483 LOC  /  157 Go test files  /  29,563 test LOC
```

Distribution by top-level dir (loc, files):

| Dir | Total LOC | Test LOC | Test/non-test ratio |
|---|---:|---:|---:|
| `services/orchestrator/` | 18,914 | 10,469 | ~55% (heavy: Postgres harness + race tests) |
| `services/platform-api/` | 13,167 | 6,283 | ~48% |
| `services/archive-extractor/` | 3,456 | 1,723 | ~50% |
| `clients/cli/` | 12,693 | 4,846 | ~38% (most non-test code is HTTP/jobstream glue) |
| `libs/go/` | 9,109 | 4,854 | ~53% |
| `devtools/` | 1,937 | n/a | n/a |
| `qa/` | 1,000 | 1,000 | 100% (all e2e) |

All four Go services + the CLI + the seven `libs/go/*` shared packages compile against `Go 1.26.3` (pinned in every `go.mod`, `go.work`, `Dockerfile`, and `.golangci.yml:1-3`).

## Go module layout

| Module | Path | Purpose |
|---|---|---|
| `services/platform-api` | `services/platform-api/cmd/server`, `cmd/healthcheck`, `internal/{api,jobstatus,messaging,project,sqlite,status,statussource}` | Public REST API (jobs/projects/scanners/SSE) |
| `services/orchestrator` | `services/orchestrator/cmd/orchestrator`, `cmd/healthcheck`, `internal/{adapters/{repository,runtime,storage},application/jobs,api,domain/jobs,metrics,orchestrator}` | Job FSM, Podman runtime, NATS consumer, admin API |
| `services/archive-extractor` | `services/archive-extractor/cmd/server`, `internal/{discovery,extractor,server}` | ZIP-safe static server + HTML discovery |
| `clients/cli` | `clients/cli` + `internal/{apiclient,diffrender,jobstream,manifesttmpl,projectmode,urlcheck}` | Cobra CLI; submits jobs, watches SSE, renders diffs |
| `libs/go/{bootstrap,config,diff,domain,events,httputil,logging,messaging,models,provenance,scannercatalog,scannerregistry,storage}` | Shared Go libs | Public Go APIs, FSM source, event envelopes, MinIO/NATS adapters, scanner registry |
| `devtools/ops/job-status-cli` | `devtools/ops/job-status-cli/` | UNKNOWN (not read) |
| `devtools/qa/suite-runner` | `devtools/qa/suite-runner/` | UNKNOWN (not read) |
| `qa/e2e` | `qa/e2e/*.go` | End-to-end helpers (1000 LOC, all tests) |
| `libs/contracts/provenance/generated/go` | `libs/contracts/provenance/generated/go/` | UNKNOWN (not read) |
| `libs/contracts/report/generated/go` | `libs/contracts/report/generated/go/` | UNKNOWN (not read; referenced by orchestrator) |
| `libs/contracts/scanner-manifest` | `libs/contracts/scanner-manifest/scanner_manifest.go` | Generated Go types for scanner manifest JSON schema |

---

## Cross-cutting findings

### Strongest areas

1. **Pinned toolchain.** Every Go module declares `go 1.26.3` (verified in all 10 `go.mod` files inspected). The single `Dockerfile` builder base is `golang:1.26.3` for orchestrator and platform-api, `golang:1.26.3-alpine` for archive-extractor. The `.golangci.yml` v2 config selects ~40 linters including `staticcheck`, `govet` (full minus `fieldalignment`), `errcheck`, `gosec`, `gocritic`, `bodyclose`, `rowserrcheck`, `sqlclosecheck`, `noctx`, `exhaustive`, `copyloopvar`, `wsl_v5`, `dupl`, `nestif`, `gocyclo` (min 15), `gocognit` (min 25), `depguard`. `just ci` runs `golangci-lint run --allow-parallel-runners` for every workspace module, then `go test -race ./...`, then `govulncheck ./...`.

2. **Structured logging with context propagation.** `libs/go/logging/logger.go:1-207` provides `slog.NewJSONHandler` plus `WithJobID/WithRequestID/WithRunID/WithScanner/WithComponent` context keys and `Info/Warn/Error/Debug` helpers that pull context values. Used uniformly in all three services. Event handlers in orchestrator explicitly propagate `request_id`/`run_id` from the inbound envelope via `event_trace.go:18-34 backgroundWithCorrelation()` and re-attach to background goroutines.

3. **Bounded sensitive data in audit trail.** `event_trace.go:49-91 redactPayloadForAudit()` strips `auth.content_b64` from `JobCreatedPayload` before persisting job event rows. This is non-trivial: a storage-state credential blob could otherwise leak into Postgres job_events.

4. **FSM as data, not code.** `libs/go/domain/job/state.go:11-47` (not directly read, but referenced from `services/orchestrator/internal/orchestrator/orchestrator.go:230 canTransition` and the application layer) is the only source of truth for transitions. Tests in `service_test.go` (not read) and the orchestrator test suite drive transition coverage through the application service. The `application/jobs/service.go:315-423` `CompleteJob`/`FailJob` flows use `domainjobs.IsTerminalState` and `domainjobs.CanEnterCompleting` guards.

5. **Panic recovery layered on every event handler.** `consumers.go:39` (orchestrator) wraps every subscription handler with a `defer recover()` that records `HandlerStatus=panic` and a stack trace to the job_events table. `services/orchestrator/internal/orchestrator/event_trace.go:107-167 withInboundEvent()` is the second layer with timing/duration metrics. `services/orchestrator/internal/api/middleware.go:86-114 recoverPanic()` does the same for admin HTTP. `services/platform-api/internal/api/middleware.go` does not appear to have an explicit `recoverPanic` middleware; it relies on `http.Server` connection-level recovery. (Worth confirming — not strictly a bug for HTTP since net/http does this, but the structured error body would be different.)

6. **Test infrastructure is real, not faked.** Two real harnesses, both work without docker:
   - **Embedded Postgres**: `services/orchestrator/internal/adapters/repository/postgres_test_harness_test.go:1-85` boots `fergusstrange/embedded-postgres` (V18) in `TestMain`, picks a free port, sets `testDatabaseURL`. The `services/orchestrator/internal/api/postgres_test_harness_test.go:1-` is identical; the `internal/orchestrator/postgres_test_harness_test.go:1-` is identical.
   - **Per-test schema isolation**: `database_test.go:51-137` (`TestInitSchemaDropsLegacyJobEventsJobIDNotNull`) creates a fresh `legacy_events_<nanos>` schema with the old `job_id NOT NULL` shape, then `NewDatabase` migrates it. This is a genuine *legacy-migration* test, not a stub.
   - **No mocking of the DB interface**: handlers, service, and orchestrator all run against a real Postgres. The only mocks are `mockPodmanClient` (`orchestrator_test_helpers_test.go:25-142`), `mockPublisher` (`:144-192`), and `memoryStorage` (`:13-57`) — all for things that genuinely cannot be tested in-process.

7. **MinIO SDK isolation via interface.** `libs/go/storage/minio.go:18-59` defines `minioAPI` so the real `*minio.Client` is decoupled. `EnsureBuckets` (`minio.go:185-270`) uses **30 retries × 2 s** to ride out MinIO container startup races. `DownloadFile` (`:322-337`) explicitly `obj.Stat()`s the response to surface `NoSuchKey` synchronously instead of failing on first read. `FileExists` (`:352-364`) maps `NoSuchKey` to `(false, nil)` and propagates other errors.

8. **Custom `?`→`$N` rebinder with quote awareness.** `sql_bind.go:25-71 rebindQuestionMarks` walks character-by-character, tracks `inSingleQuote` to skip `?` inside string literals, and caches results in `sync.Map`. Not as capable as sqlx, but small and correct for this codebase's query shapes (verified by 388-line `jobs.go` repository using it throughout). One subtle bug to flag: it does not handle PostgreSQL `$$ ... $$` dollar-quoted blocks — none of the queries in this repo use them, so it's a latent issue, not an active one.

9. **SSRF defense in two layers with a private mode.** `services/platform-api/internal/api/security.go:136-301` defines a static blocked-prefix list (`blockedIPPrefixValues`) covering CGNAT (100.64/10), metadata (169.254.169.254), benchmark (198.18/15), link-local, IPv4/IPv6 multicast, IPv4 reserved, IPv6 ULA, IPv4-mapped. `allowedPrivateIPPrefixValues` allow-list contains RFC1918 + loopback for the *private* mode. `classifyTargetIP` returns one of `targetIPDecisionAllow | AllowInPrivateMode | Block` so the validator works without branching around the same CIDR set twice. The DNS resolver interface `ipAddrResolver` is injected (`:19-21`), which is what makes `security_dns_test.go` and `security_resolver_test.go` testable.

10. **Path-traversal defense in two places.** `services/archive-extractor/internal/server/server.go:53-72` `localPathForRequest` calls `path.Clean` then verifies the cleaned path is still inside the workspace root. `services/platform-api/internal/api/object_keys.go:16-37 jobScopedKey` and `object_keys.go:39-67 jobScopedJoin` reject `.`, `..`, and `jobID`-equal keys via `path.Clean` and a strict `jobID/` prefix check.

11. **API key + bearer-token via `subtle.ConstantTimeCompare`.** `services/platform-api/internal/api/middleware.go:351` and `services/orchestrator/internal/api/server.go:146` both use constant-time comparison. The platform-api additionally requires `PLATFORM_API_TOKEN` to be set unless `PLATFORM_API_AUTH_DISABLED=true` (`middleware.go:308-325 ValidateAuthConfig`).

12. **NATS JetStream handler with envelope + delivery metadata context.** `libs/go/messaging/subscribe.go:164-253 SubscribeTyped` unmarshals the envelope *leniently* (allow additive fields), then the payload *strictly* (`DisallowUnknownFields`). It injects `ReceivedEventMeta` into the context with `Deliveries`, `StreamSeq`, `ConsumerSeq`, `StoredAt`, etc. The orchestrator uses this metadata to recover from redeliveries: `service.go:425-434 shouldResumeCompletingJob` returns `meta.Deliveries > 1`. This is a genuinely thoughtful redelivery-aware pattern.

13. **Consumer staleness detection + recreation.** `libs/go/messaging/consumer.go:84-108 consumerStateLooksStale` compares `streamInfo.State.LastSeq` to `consumerInfo.Delivered.Stream`/`AckFloor.Stream` and to the subject's `lastSubjectSeq`. If the consumer's cursor is past stream-tail with zero pending, the consumer is dropped and recreated (`consumer.go:53-79`). This catches the "I added a new event subject, the old consumer cursor is past the new tail, and nothing arrives" failure mode.

14. **Generate-then-test pattern for JSON schemas.** `libs/contracts/scanner-manifest/scanner_manifest.go:1-543` is generated by `go-jsonschema` (banner at line 1). Required fields, length bounds, regex patterns, and enum members are all enforced via custom `UnmarshalJSON` methods on each type. This is the right way to do schema-driven contract types.

15. **Distributed authentication (form / storage_state / from_env) with no creds in transit.** `services/platform-api/internal/api/handlers_jobs_url_submit.go:411-625` validates a small subset of the PreScanAction schema (no full ajv), decodes a base64 storage-state blob, uploads it to MinIO under `BucketArtifacts/<jobID>/auth/storage-state.json`, then **forwards only `{mode, artifact_key}` in the wire payload**. The base64 bytes never cross the NATS boundary. The orchestrator's defensive `WithAuthUploader` (`application/jobs/service.go:40-46` + `application/jobs/auth_cleanup.go`) catches older producers that still send inline content. The form-recipe `from_env` references are validated against `^[A-Z][A-Z0-9_]*$` (`handlers_jobs_url_submit.go:35`) so only legitimate env-var names reach the orchestrator.

### Critical issues (would block open-sourcing as-is)

**C1. Embedded Postgres tests are non-hermetic and slow on first run.** `services/orchestrator/internal/adapters/repository/postgres_test_harness_test.go:31-49` downloads Postgres 18 binaries to a temp dir and boots a real DB on `TestMain`. Any contributor running `go test ./...` for the first time will silently download ~80 MB, and **all three test packages share the same `testDatabaseURL` global** (`postgres_test_harness_test.go:13`, `internal/api/postgres_test_harness_test.go:1-`, `internal/orchestrator/postgres_test_harness_test.go:1-`). Running the orchestrator package's tests in parallel with `-p 4` is fine because each package has its own binary, but if you ever try `go test ./... -count=1` while another process is using the same dynamic port, races are possible. The package currently has no `_test.go` files that fail without Postgres, so removing it is not a quick option.

**C2. `event_trace.go:36-47 marshalPayload` swallows marshal errors into a string with a typo in the error path.** Quoted verbatim: `return \`{"marshal_error":\` + strconvQuote(err.Error()) + \`}\``. This writes a JSON-shaped string that is *not valid JSON* (no closing brace). Downstream consumers that try to parse the `payload` column as JSON will fail. A safe alternative is `json.Marshal(map[string]string{"marshal_error": err.Error()})`. The redact function (`event_trace.go:49-91`) is also asymmetric: it only redacts `*events.JobCreatedPayload` and `events.JobCreatedPayload` — every other payload type is stored verbatim. For a security-aware audit trail that is correct (auth only lives on JobCreated), but the asymmetry should be commented to prevent future regressions.

**C3. Auth bypass via `PLATFORM_API_AUTH_DISABLED=true` is silent.** `services/platform-api/internal/api/middleware.go:308-325 ValidateAuthConfig` lets operators set `PLATFORM_API_AUTH_DISABLED=true` and warns but proceeds. There is no kill-switch in the code that refuses the request — it just doesn't authenticate. For a portfolio piece this is fine; for a public-facing deployment it's a footgun. Either rename to `PLATFORM_API_AUTH_OPT_IN=true` (require explicit opt-in to disable) or refuse to start.

**C4. `Justfile` deploy recipe is a stub that exits 1 with a "see external control plane" message.** `justfile:21-43 deploy`. This is honest about being a placeholder, but a new contributor reading the file will wonder where deploy commands live. A note in `README.md` or `docs/operations/` pointing to the actual control plane would help.

**C5. `internal/api/handlers_jobs_url_submit.go:42 normalizeHighlightStyle` is referenced but not defined in the read portion** (called from `handlers_jobs_zip_upload.go:271` and `handlers_jobs_url_submit.go:99`). UNKNOWN whether it has its own tests. Search for it shows definition in `services/platform-api/internal/api/handlers_jobs_highlight_style.go` (not read).

**C6. `services/orchestrator/internal/api/metrics.go:13-78 handleMetrics` is unauthenticated by mistake?** Actually no — it's wrapped in `mux.HandleFunc("/metrics", s.requireAuth(s.handleMetrics))` (`server.go:118`). Good. But the platform-api `/metrics` endpoint — does one exist? **UNKNOWN**, not in the routes I read (`router.go:9-22`).

### Polish opportunities (non-blocking but visible)

**P1.** `cobra_root.go:16-66` deprecates `--json` but still accepts it. There's no clear roadmap for removal. Either remove now or document the removal version.

**P2.** `consumers.go:39-` panic recovery in `orchestrator/internal/adapters/messaging/` is the *only* file in that package; combined with the in-orchestrator `withInboundEvent` (which *also* recovers), every event handler has **two** recovery points. The outer recover is the safety net; the inner one is the audit trail. Consider documenting this layering in a comment.

**P3.** `application/jobs/service.go:328-367 beginJobCompletion` has four `return false, publishErr` and two `return false, err` paths that all do effectively the same thing. A small helper `noProceed(reason error)` would clean this up. Functional, but readability suffers.

**P4.** `internal/api/handlers_projects.go:53-91` `handleProjects` does path splitting inline. A `splitSlug` helper would centralize validation. Same code shape in `handlers_jobs_zip_upload.go:218-291` (`parseZipUpload` is large; could be split).

**P5.** `libs/go/provenance/auth.go:91-163 Compact()` builds `map[string]any` for storage_state but is asymmetric with the form-mode wire encoding — `Compact` returns the right shape, but the same logic in `handlers_jobs_url_submit.go:375-388` rebuilds it inline. A shared `auth.CanonicalForStore()` helper would prevent drift.

**P6.** `libs/contracts/scanner-manifest/scanner_manifest.go:517-525` enforces `Id` with regex `^[a-z][a-z0-9-]*$` and length 2..64 in custom `UnmarshalJSON`. The error path (`fmt.Errorf("field %s pattern match: ...")`) is reasonable, but the duplication of "if present, check length" for every optional field is verbose. `go-jsonschema` generates this style by default; a thin wrapper could centralize optional-length checks.

**P7.** `services/orchestrator/internal/api/middleware.go:31-32` rate limiter uses `rateWindow` with `start time.Time; count int`. Stale entries are pruned on every `allow` call, but the eviction sweep is O(N) over `windows`. At 10k entries this is fine; at 1M it isn't. `rateLimiterMaxEntries = 10000` (`:34`) keeps it bounded. Note the platform-api also has its own rate limiter (`middleware.go:42-93`); no shared implementation. Acceptable.

**P8.** `services/orchestrator/internal/orchestrator/service_adapters.go:262-323 orchestratorRuntime.StartExtractionWorker` uses a `select { case <-resultChan: / case <-timeoutCtx.Done(): }` pattern with a 1-buffered channel. If the goroutine panics, no one will read from the channel. The outer `spawnMonitorContainer` (in `podman_helpers.go:11-27`) has a panic-unaware `go func()` with no recovery. Consider wrapping both in `defer recover()` so a panic doesn't leak.

---

## Per-service findings

### `services/platform-api`

- **Files read**: `cmd/server/main.go`, `cmd/server/config.go`, `cmd/healthcheck/main.go`, `internal/api/server.go`, `router.go`, `middleware.go`, `security.go`, `security_test.go`, `security_dns_test.go` (listed), `security_resolver_test.go` (listed), `handlers_jobs_zip_upload.go`, `handlers_jobs_url_submit.go`, `handlers_jobs_modules.go` (listed), `handlers_jobs_highlight_style.go` (listed), `handlers_jobs_status.go` (listed), `handlers_projects.go`, `handlers_scanners.go` (listed), `handlers_sse.go`, `handlers_health.go`, `handlers_sse_test.go` (listed), `handlers_sse_wire_test.go` (listed), `handlers_test.go` (listed), `handlers_coverage_test.go` (listed), `job_status_screenshots.go` (listed), `job_status_response.go` (listed), `middleware_test.go` (listed), `handlers_jobs_modules_test.go` (listed), `object_keys.go`, `object_keys_test.go` (listed).

- **Strengths**
  - SSRF defense (above) is the single best piece of code in the repo.
  - Multipart upload is `MaxBytesReader`-bounded (`handlers_jobs_zip_upload.go:122`), field-by-field size-limited (`handlers_jobs_zip_upload.go:37-48 readFormValue`), and uses `httputil.NewValidationError` for structured 4xx (`handlers_jobs_zip_upload.go:251-256`).
  - Structured error responses are uniform: `httputil.ErrorDetail` carries `code/message/details/suggestion/field/retry_after/meta` (`libs/go/httputil/errors.go:42-50`). The error taxonomy is enumerated (`errors.go:13-34`).
  - Upload `screenshot` field is a *pointer* (`handlers_jobs_zip_upload.go:110,413-425`) so the handler can distinguish "absent" from "false" — clean.
  - The `ParseModules` flow lists supported modules in the error response (`handlers_jobs_zip_upload.go:260-266`) so users get self-serve guidance.
  - `ValidateAuthConfig` is enforced at startup (`cmd/server/main.go` — **UNKNOWN exact line** but `config.go:31-41` calls `config.ValidateAll`).
  - SSE keepalive every 15 s (`handlers_sse.go:346`) prevents proxies from killing long-lived connections. Initial status is sent before subscription begins, so clients always get an immediate `status` event.

- **Weaknesses**
  - `handlers_jobs_zip_upload.go:115-120 handleJobZipUpload` checks `r.Method != http.MethodPost` and returns plain `http.Error`. Consistent with the orchestrator, but the platform-api *does* use `httputil.RespondError` elsewhere; the inconsistency is small but visible.
  - `handlers_projects.go:137-141` detects UNIQUE constraint violations by string-matching `err.Error()`. A more robust approach is to use the SQLite driver's typed error. UNKNOWN which driver — only saw `internal/sqlite` listed.
  - `object_keys.go:39-67 jobScopedJoin` uses `path.Clean(path.Join(...))`. `path.Clean` is correct for URL paths but not for MinIO keys on Windows. UNKNOWN whether the platform-api ever runs on Windows; the Dockerfiles are Linux-only so this is acceptable.
  - `handlers_jobs_url_submit.go:233-260 normalizeJobURLSubmitAuth` has a deeply nested error-type switch (`errors.Is` against `context.DeadlineExceeded`, `context.Canceled`, `errAuthStorageUnavailable`) inline in the handler. A small `mapError` helper or an `apperror` type would clean this up.
  - `handlers_sse.go:18-27 jobIDFromJobPath` does `strings.Split(path, "/")` and asserts `len(parts) == 2`. If a URL ever has a trailing slash the assertion fails — that's the intent (return invalid) but the 400 message could be more specific.

### `services/orchestrator`

- **Files read**: `cmd/orchestrator/main.go`, `cmd/orchestrator/config.go`, `cmd/healthcheck/main.go`, `internal/orchestrator/{orchestrator.go, events.go, event_trace.go, event_trace_test.go, scanning.go, service_adapters.go, service_adapters_test.go, podman_helpers.go, job_cleanup.go, deadline.go, orchestrator_test_helpers_test.go, memory_storage_test.go, orchestrator_init_test.go, orchestrator_extraction_test.go}` (and the `*_test.go` siblings in the dir listing), `internal/api/{server.go, middleware.go, metrics.go, server_test.go, postgres_test_harness_test.go}`, `internal/adapters/{repository/{database.go, jobs.go, database_test.go, sql_bind.go, postgres_test_harness_test.go}, runtime/{client.go, pods.go, containers.go, job_runtime.go, job_runtime_test.go}, messaging/{consumers.go, publisher.go}, storage/{report_aggregator.go, auth_uploader.go}}` (plus their `_test.go` siblings in dir listing), `internal/application/jobs/{service.go, ports.go, service_test.go, ...}`, `internal/metrics/collector.go`, `internal/domain/jobs/transitions.go`.

- **Strengths**
  - **In-process metrics with no third-party client.** `internal/metrics/collector.go:30-47` is a tiny thread-safe `Collector` with three metric families: event counts, HTTP status, and a duration histogram (`durationBucketsMs` 5ms..60s, `:18-20`). Output is hand-rolled Prometheus text format (`:99-173`). Wired into `/metrics` via `metrics.go:70 WriteProm`. This is impressive: a real portfolio-worthy alternative to pulling in `prometheus/client_golang`.
  - **Per-event delivery metadata → recovery decisions.** `application/jobs/service.go:425-434 shouldResumeCompletingJob` uses `meta.Deliveries > 1` to decide whether a `COMPLETING`-state event handler should resume or yield. The redelivery signal is the *only* signal you have for "the previous owner died." Using it correctly here is a sign of careful NATS usage.
  - **Reclaimable container monitor goroutines.** `podman_helpers.go:11-27 spawnMonitorContainer` increments `monitorWG` so `WaitForMonitors` (orchestrator.go:247-253) can block tests. The cleanup ordering is right: `t.Cleanup(orchestrator.WaitForMonitors)` (test_helpers:277) is registered *before* the database cleanup because `t.Cleanup` runs LIFO. This is the kind of test plumbing that usually breaks; here it works.
  - **Deadline sweeper runs as a single goroutine guarded by `sync.Once`.** `deadline.go:14-29 startDeadlineSweeper` and `orchestrator.go:240-243 Start` use `deadlineSweepOnce.Do`. No way to start two sweepers, even on `Start` re-entry.
  - **Two failure-recording paths: `failJob` (returns error) and `failJobSafe` (swallows).** `job_cleanup.go:62-69`. The signature makes intent obvious at call sites.

- **Weaknesses**
  - **`event_trace.go:36-47 marshalPayload` bug** (C2 above).
  - **No `-race` discipline evident in the message handlers.** `consumers.go:39-` recovers panics but the actual handler closures capture shared mutable state (e.g., `slog` defaults, `metrics` collector). The collector is `sync.Mutex`-guarded; `slog` is concurrent-safe; but the test helper `mockPodmanClient.createContainerFunc` etc. let tests inject racy callbacks. UNKNOWN whether the existing tests catch this.
  - **`internal/adapters/storage/report_aggregator.go:29-70` `BuildAggregatedReport`** does no partial-success tolerance: if any scanner succeeded it builds the report, but if the report JSON marshalling fails the error is opaque. UNKNOWN whether there's a test for that path. The aggregator has `report_aggregator_test.go` (listed but not read).
  - **The orchestrator does not have an explicit "service must be healthy" probe for the scanner-runner image.** It assumes the image exists locally (`localhost/stageflow/scanner-runner:latest`). If the pod fails to start because of a missing image, the monitor detects it via exit code (`podman_helpers.go:60-116`) and marks the job failed. Correct, but slow.
  - **`orchestrator.go:170-185` `NewOrchestrator` falls back to an empty registry if `DefaultConfig()` fails.** This is defensive in the right way (don't crash on missing default config), but the fallback means jobs fail later with "scanner not found" instead of at startup. A `slog.Error` is logged (`orchestrator.go:181`), which is good — but the orchestrator proceeds. For a portfolio piece this is fine; for prod you'd want a fatal.

### `services/archive-extractor`

- **Files read**: `cmd/server/main.go`, `cmd/server/stage_logger.go`, `internal/extractor/extractor.go`, `internal/server/server.go`, `internal/discovery/discovery.go`, `Dockerfile`.

- **Strengths**
  - **ZIP-bomb defense is explicit and well-named.** `internal/extractor/extractor.go:22-28` defines 6 constants: `maxZipEntries=5000`, `maxZipExpansionRatio=100`, `maxZipCompressedSize=100MB`, `maxZipUncompressedSize=1GiB`, `maxZipEntryUncompressedSize=250MiB`, `maxZipNameLen=4096`. The *combination* of these is what stops zip bombs; any one alone is insufficient. Good design.
  - **Static server has its own path-traversal guard.** `internal/server/server.go:53-72` `localPathForRequest` calls `path.Clean`, strips leading `/`, and verifies the result is still rooted in the workspace directory. `containsDotPathSegment` (`:53-58`) blocks `..` segments explicitly. UNKNOWN whether there's a test for this (listed in dir but not read).
  - **Symlink-skipping on HTML discovery.** `internal/discovery/discovery.go:38-44` checks `d.Type()&fs.ModeSymlink` and skips. This is correct — a symlink in the workspace is an obvious escape route.
  - **Single-binary alpine image** (`Dockerfile:1-32`), non-root user, `HEALTHCHECK` not set (relies on the host's pod healthcheck).
  - **Run-with-context, signal-handled main.** `cmd/server/main.go` (line range not re-read; **UNKNOWN exact lines** but the pattern of `runExtraction` with `must*` helpers is a deliberate testability wrapper).

- **Weaknesses**
  - **Limited test surface visible in what I read.** I saw the integration test file listed (`integration_test.go`) but didn't read it. UNKNOWN coverage of the ZIP-bomb paths.
  - **`internal/discovery/discovery.go`** — the symlink-skipping is *defense in depth*, but if the symlink target is a directory the current `WalkDir` callback still recurses via `WalkDir`'s own descent. The `if d.Type()&fs.ModeSymlink { return nil }` (`:38-44`) prevents the *read* but not the *listing*. Re-checking the snippet: yes, it returns `nil` *before* `d.IsDir()` so symlinks are entirely skipped. Correct.
  - **`internal/server/server.go:53-58 containsDotPathSegment`** checks for segments that *equal* `.` or `..` (or start with `..`), but doesn't check for Windows-style separators on a Linux-only build. Acceptable for the runtime.

### `clients/cli`

- **Files read (top-level)**: `main.go`, `run.go`, `cobra_root.go`, `cobra_scan.go`, `scan_job.go`, `scanners_output.go`, `scan_output.go`, `time_helpers.go`, `version.go` + `test_helpers_test.go`, `version_test.go`, `scanners_output_test.go`.

- **Files read (internal/)**: `internal/urlcheck/urlcheck.go`, `internal/apiclient/client.go`, `internal/jobstream/sse.go`, `internal/diffrender/diffrender.go`, `internal/manifesttmpl/manifesttmpl.go`, `internal/projectmode/project_root.go` (plus all `_test.go` files in those dirs, listed).

- **Strengths**
  - **Testable `run(args, getenv, stdout, stderr)` pattern.** `run.go:12-33` takes all four as parameters so a test can drive the whole CLI without capturing stdin/stdout. This is the right shape for a Cobra-rooted CLI — most projects skip it.
  - **`urlcheck` mirrors the platform-api SSRF defense.** `internal/urlcheck/urlcheck.go:127-181 ContainsPrivateTargets` / `isPrivateLiteralIP` use `netip.AddrFromSlice` + `IsLoopback`/`IsPrivate` — same primitives the server uses, so the two layers cannot disagree. `ValidateLocalTargets` (`:75-93`) refuses to send private targets to a non-loopback API URL. That's a thoughtful UX safety.
  - **SSE parser in `internal/jobstream/sse.go`** (466 LOC) handles keepalive lines (`writeSSEKeepalive` in platform-api sends `:keepalive\n\n`). UNKNOWN coverage without reading the test file.
  - **`diffrender` is its own package.** `internal/diffrender/diffrender.go` is 372 LOC and has `diffrender_test.go` (182 LOC). This is the right factoring — diffing + coloring is independent of API + SSE.

- **Weaknesses**
  - **`run.go:12-33 run(args, getenv, stdout, stderr)` is called from `main.go`** (line range not re-read). UNKNOWN if it surfaces cobra's `--help` exit codes correctly. The `exitCodeError` wrapper (`run.go:1-10`, not read in detail) suggests yes.
  - **`scanners_output.go:1-105` and `scan_output.go:1-112`** split rendering across files. UNKNOWN overlap. The test file `scanners_output_test.go:1-89` exists.
  - **`cobra_root.go:16-66`** has the `--json` deprecation. UNKNOWN test coverage.
  - **`internal/manifesttmpl/manifesttmpl.go` is 144 LOC with 57 LOC test.** A `manifesttmpl` package is for rendering scanner manifests. Reasonable isolation.
  - **CLI uses `cobra` (per `cobra_root.go:1-15`) but I did not see a typed command tree test.** The `test_helpers_test.go:1-89` is referenced but not read.

---

## Shared library (`libs/go`)

### `libs/go/messaging`

- **Files read**: `client.go`, `publish.go`, `subscribe.go`, `envelope.go`, `consumer.go`, `streams.go` + `nats_client_test.go` (623 LOC), `nats_test.go` (301 LOC), `nats_consumer_state_test.go` (72 LOC).
- **Strengths**: above (#12, #13). The `consumerStateLooksStale` check is the standout.
- **Weaknesses**:
  - `client.go:172-187 trackConsumeContext` and `:189-201 untrackConsumeContext` are fine but the `consumeContexts` map is *not* bounded. A consumer re-create storm could grow it. Acceptable in practice.
  - `subscribe.go:255-273 unmarshalStrict` uses `DisallowUnknownFields` which is great, but the `// Ensure exactly one JSON value` check (`dec.Decode(&struct{}{}); err != io.EOF`) is duplicated in `unmarshalLenient` (`:275-291`). A small `assertSingleValue(dec)` helper would dedupe.
  - `streams.go:35-91 EnsureStreams` hard-codes the stream → subjects mapping. The stream/consumer names (`:12-15`, `:25-32`) are also hard-coded constants. Acceptable; the test file (`nats_client_test.go:623 LOC`) covers a lot of this.

### `libs/go/events`

- **Files read**: `envelope.go`, `types.go`.
- **Strengths**: every payload type has a `Validate()` method that returns structured errors with consistent prefixes (`events: <Type>.<field> is required`). `types.go:1-460` defines 11 payload types — comprehensive. `JobFailedPayload.Stage` is an enum check (`types.go:447-453`) over the three known stage values.
- **Weaknesses**:
  - `JobCreatedPayload.Config.Modules` is required to be non-empty (`types.go:71-73`). The platform-api enforces this at the handler level. Good cross-layer discipline.
  - `ExtractionReadyPayload.ProvenancePath` and `BaseURL` are required (`types.go:97-102`). The orchestrator never re-validates payloads it produced — it only validates payloads at the consume boundary. UNKNOWN whether the orchestrator's handler ignores invalid envelopes or treats them as fatal.

### `libs/go/models`

- **Files read**: `job.go`, `job_test.go`.
- **Strengths**:
  - `JobState` is a typed string with `IsValid()` and `IsTerminal()` methods (`job.go:40-57`).
  - `JobConfig.Auth` is `json.RawMessage` (`job.go:122`) with a long comment explaining the storage_state / form / from_env invariant. This is the right type for "we pass opaque bytes through."
- **Weaknesses**:
  - `Job` has 28 fields (`job.go:60-88`) with no constructor or `Validate()`. The orchestrator's `CreateJob` (`internal/adapters/repository/jobs.go:28-59`) only writes a subset. UNKNOWN whether other writers enforce shape consistency.
  - `job_test.go:1-92` covers `JobState_ValuesAndHelpers` and `Job_JSONTagsAndOmitEmpty`. Not enough to catch field-level bugs but adequate for invariants.

### `libs/go/storage`

- **Files read**: `minio.go` (364 LOC).
- **Strengths**: covered above (#7). The 30×2s retry is opinionated and the rationale is clear in the comment.
- **Weaknesses**:
  - `minio.go:144-153 NewMinIOClient` requires `cfg` to be non-nil. The constructor doesn't validate individual fields; a bad `Endpoint` will surface at first use. The caller is expected to validate via `libs/go/config/loaders.go` first.

### `libs/go/config`

- **Files read**: `env.go`, `loaders.go`, `validation.go` (via summary).
- **Strengths**: typed env helpers (`GetEnv`, `GetEnvInt`, `GetEnvBool`, `GetEnvDuration`) with `MustGetEnvInt` for required values. The strict `GetEnvBool` allow-list `["1", "true", "TRUE", "True"]` prevents "Yes" / "On" silent acceptance.
- **Weaknesses**: UNKNOWN test coverage; tests for `env.go` were not listed in the `ls` output.

### `libs/go/scannerregistry`

- **Files read**: `registry.go` (24 LOC), `types.go` (80 LOC), `registry_modules.go` (101 LOC), `registry_query.go` (149 LOC) + `config.go` (286 LOC, not read in detail).
- **Strengths**:
  - `Registry` is `sync.RWMutex`-guarded (`:9-14`) and all public methods are read-locked (`registry_query.go:11` etc.).
  - `ResolveModules` is lenient (unknown tokens pass through) for backwards compat with custom scanners (`registry_modules.go:46-50`); `ResolveModulesStrict` is strict.
  - `Registry` default image fallback (`registry_query.go:116-125 GetImage`) means missing-image scanners fall back to the configured default.
- **Weaknesses**:
  - No port from `config.go` (286 LOC) reviewed. UNKNOWN whether scanner config is hot-reloadable.

### `libs/go/provenance`

- **Files read**: `auth.go` (200/426 LOC), `auth_test.go` (241 LOC).
- **Strengths**:
  - `auth.go:1-12` is a long package comment explicitly stating: this is a Go-side mirror of the TypeScript `secrets-resolver.ts`, both implementations must produce identical output, both kept in sync by shared fixtures. The fixture-driven test pattern is the right way to keep two implementations aligned.
  - `CollectFromEnvReferences` (`auth.go:177-200+`) walks form steps and only emits `from_env` names for `fill` / `select` step types. Sorted, deduplicated, allow-list for the orchestrator's pod env.

### `libs/go/diff`

- **Files read**: `diff.go` (105 LOC), `diff_test.go` (164 LOC).
- **Strengths**: `ComputeDiff` is a clean 65-line function with deterministic sorted output (`diff.go:74-75`). The Result type is the wire shape, so the same struct can be marshalled and consumed by the CLI's `diffrender` package.
- **Weaknesses**: `diff.go:84-105` returns a `Result` that includes `ScoreDelta` only if both sides have a score. The contract "delta is nil if either is nil" is implicit; a comment would help.

### `libs/go/httputil`

- **Files read**: `response.go` (39 LOC), `errors.go` (169 LOC) + `response_test.go` and `errors_test.go` (listed).
- **Strengths**:
  - `errors.go:42-50 ErrorDetail` is a thoughtful shape: `code` is the machine-readable enum, `suggestion` is for client UX, `retry_after` only on rate limits, `meta` is open-ended. Self-documenting.
  - `NewUnsupportedModuleError` (`errors.go:120-130`) lists supported modules in the suggestion — same pattern as the platform-api inline.
- **Weaknesses**: `errors.go:152-160 NewDatabaseError` and `errors.go:162-169 NewStorageError` are unused in the read code paths. The platform-api uses `NewValidationError`, `NewNotFoundError`, `NewRateLimitError`, `NewPayloadTooLargeError`, `NewUnsupportedModuleError` — but no handler emits `DATABASE_ERROR` / `STORAGE_ERROR` codes. UNKNOWN whether they're used in code I didn't read.

### `libs/go/logging`

- **Files read**: `logger.go` (207 LOC), `logger_test.go` (232 LOC).
- **Strengths**:
  - Context-key types are *string* (`logger.go:12-20`) but stored via `contextKey` unexported type to avoid collisions. Idiomatic.
  - `L(ctx)` (`:173-187`) returns a logger with all context values pre-bound. Convenient and avoids passing 5 keys manually.
- **Weaknesses**: UNKNOWN coverage of the `slog` JSON output schema. Tests are 232 LOC; probably adequate.

### `libs/go/domain` and `libs/go/bootstrap`

- `libs/go/domain/job/state.go` is the FSM source. The orchestrator's `application/jobs/service.go` is structured to delegate every state-check to `domainjobs.*` (e.g., `:104 DecideExtractionReady`, `:134 CanTransitionTo`, `:195 ShouldIgnoreTerminalEvent`, `:232 DecideScanFailureCompletion`, `:332 IsTerminalState`). This is clean hexagonal architecture.
- `libs/go/bootstrap/bootstrap.go` provides `SetupLogging`, `NewNATSClient`, `NewMinIOClient`. Each takes an options struct so callers can opt into `EnsureStreams` / `EnsureBuckets` and `IgnoreEnsureFailure` (so a startup race doesn't fail the boot).

---

## Shared contracts (`libs/contracts/`)

- **Read**: `libs/contracts/scanner-manifest/scanner_manifest.go` (543 LOC, all generated).
- **Not read**: `libs/contracts/provenance/`, `libs/contracts/report/`, `libs/contracts/events/` (TypeScript workspaces).
- **Strengths**: scanner-manifest has hand-rolled required-field + length + pattern checks in custom `UnmarshalJSON` (`:12-543`). The `ManifestConfigSchema` and `ScannerManifestConfigSchema` types are *intentionally* `json.RawMessage` with comments explaining "Go does not introspect this; scanner-runner validates at runtime via Ajv" (`:218-226`, `:477-480`). Correct.
- **Weaknesses**: UNKNOWN. The TS contracts drive the contract evolution, and the Go side is generated/hand-rolled to match. This is the right shape.

---

## Test coverage assessment

### Where coverage is real
- **Orchestrator end-to-end** (`orchestrator_extraction_test.go:1-132`): Real Postgres, real orchestrator, real Podman mock. The full flow from `HandleExtractionReady` → state transition → `SCANNING` → `RecordScanStart` is asserted against the database.
- **API server** (`services/orchestrator/internal/api/server_test.go:1-533`): Real Postgres, fake podman, full router. Tests pagination, filtering, auth, rate-limit, recovery middleware, and `/metrics`.
- **Platform-api security** (`security_test.go:1-322`): A thorough SSRF matrix — public, private, link-local, metadata, CGNAT, benchmark, multicast, IPv6 ULA, invalid schemes, empty/whitespace, port handling, private-mode differential.
- **MinIO storage** (`libs/go/storage/minio_client_test.go` — listed, not read): Likely exercises the bucket retry + `NoSuchKey` mapping.
- **Event envelopes** (`libs/go/events/types_test.go`, `envelope.go` companions, `events_test.go`, `contracts_test.go`): Validation paths.
- **Auth port** (`libs/go/provenance/auth_test.go:1-241`): Fixture-driven, mirrors TS implementation. (Confirmed by `auth.go:1-12` package comment.)
- **CLI HTTP client** (`clients/cli/internal/apiclient/client_test.go:1-145`): URL building + JSON encode/decode round-trips.
- **CLI URL check** (`urlcheck_test.go`): 203 LOC, parallel to platform-api SSRF defense.
- **SSE parser** (`sse_test.go`): 189 LOC.

### Where coverage is missing or thin
- **Archive-extractor**: only `integration_test.go` (1723 LOC of test) per the dir listing. The ZIP-bomb defense (5000 entries, 100x ratio, 1GiB uncompressed) is **UNVERIFIED** in the read portion.
- **Domain transitions** (`libs/go/domain/job/state.go`): The state machine has no test in `libs/go/domain/`. All transition coverage flows through the orchestrator's `application/jobs/service_test.go`. A direct unit test of `CanTransitionTo` would be the right shape for a state-machine module.
- **`libs/go/httputil/response.go:39 RespondNotFound`** is the most-deprecated helper. UNKNOWN usage.
- **`consumers.go` panic-recovery**: UNKNOWN whether there's a test that triggers a panic in a subscription handler and asserts the recovery writes a job_event row.
- **`marshalled payload redaction** (`event_trace.go:49-91`): UNKNOWN test coverage of the `redactAuthForAudit` path.
- **CLI `cobra_root.go` deprecation**: UNKNOWN test asserts the `--json` flag still works.

---

## Error handling patterns

| Pattern | Used in | Notes |
|---|---|---|
| `fmt.Errorf("...: %w", err)` wrap with `%w` | universal | Idiomatic, preserves chain for `errors.Is/As`. |
| `sentinel errors` (`ErrNilClient`, `ErrJobNotFound`, `ErrAuthStorageUnavailable`) | `libs/go/messaging/client.go:27-33`, `services/platform-api/internal/status`, `services/platform-api/internal/api/handlers_jobs_url_submit.go:37` | Good — only some packages use them. `internal/adapters/storage/` and `internal/orchestrator/` mostly return anonymous errors. |
| `errors.Is(err, context.DeadlineExceeded)` | `services/platform-api/internal/api/handlers_jobs_url_submit.go:236-244`, `handlers_jobs_zip_upload.go:131-151` | Used to translate deadlines to `503` instead of `500`. Consistent. |
| `errors.As` for typed errors | `services/platform-api/internal/api/handlers_jobs_zip_upload.go:51-58` (`formFieldTooLargeError`), `handlers_projects.go:480-490` (`*http.MaxBytesError`) | Idiomatic. |
| `defer recover()` with stack capture | `consumers.go:39-`, `event_trace.go:107-167`, `services/orchestrator/internal/api/middleware.go:86-114` | Layered, audited, metrics-instrumented. |
| `http.MaxBytesReader` + size-checked form fields | `handlers_jobs_zip_upload.go:122`, `handlers_jobs_url_submit.go:155`, `handlers_projects.go:475` | Bounded. |
| `subtle.ConstantTimeCompare` for API keys | `services/platform-api/internal/api/middleware.go:351`, `services/orchestrator/internal/api/server.go:146` | Constant-time. |
| Context cancellation propagation | universal | All async operations respect `ctx.Done()`. |
| `sync.Once` for one-shot background workers | `services/orchestrator/internal/orchestrator/deadline.go:240-243` (`deadlineSweepOnce`), `services/platform-api/internal/api/middleware.go:127-132` (`trustedProxiesOnce`) | Standard. |
| Context-keyed logger | `libs/go/logging/logger.go:63-141` | Custom type avoids collisions. |
| `slog.Default` for ambient context | all services | JSON output, no further config needed. |

---

## Summary verdict

This is a **portfolio-quality Go codebase**. The strengths are real and observable in the code: a real embedded-Postgres test harness, a real SSRF defense that is tested in two layers, a genuine redelivery-aware NATS consumer with staleness detection, a custom in-process metrics collector that doesn't pull in `prometheus/client_golang`, a small but principled in-memory rate limiter, careful panic recovery at three layers, and tight path-traversal guards. The cross-service boundary between `platform-api` and `orchestrator` is well-typed via `events.Envelope` + per-payload `Validate()`.

The weaknesses are all polish, not architecture:
- One bug in `event_trace.go marshalPayload` produces non-JSON output for marshal errors.
- The CLI could use one more end-to-end test that drives `cobra` itself (not just `run(args, getenv, stdout, stderr)`).
- The deploy justfile is a documented stub.
- A few duplicate inline patterns (`error.Is` ladders, `slog.Error` + `httputil.RespondError` couples) could be cleaned up.

The biggest *open-source readiness* concern is the **embedded Postgres** in tests: it's a one-time ~80 MB download and the `testDatabaseURL` global is process-scoped. If a contributor is on a metered connection, `go test ./...` will fail or be slow. Add a `just test-fast` recipe that runs only the non-orchestrator packages, and document the slow path in `README.md` or `CONTRIBUTING.md`.

**Recommended command to run before tagging a release:**

```bash
just ci
```

This runs, in order: stale-vocab grep, `golangci-lint run --allow-parallel-runners` for every module, `go test -race ./...` for every module, `govulncheck ./...` for every module, CLI docs regen with `git diff --exit-code`, shell regression tests, then the web and scanner-runner TypeScript CI. UNKNOWN whether the orchestrator's `golangci-lint` and `go test -race` complete in a reasonable time on a cold cache — not measured in this audit.
