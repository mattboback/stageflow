# StageFlow Architecture

This document is the canonical description of StageFlow's system design, trust boundaries, data flows, service responsibilities, and failure modes.

Start with the [repository README](../README.md) for the product overview and fastest local setup path. Deployment instructions live in the [self-hosting guide](self-hosting.md).

---

## Table of Contents

- [System Goals](#system-goals)
- [Platform Shape](#platform-shape)
- [Core Services](#core-services)
- [Event Model](#event-model)
- [Data Flows](#data-flows)
- [Job State Machine](#job-state-machine)
- [Contract Architecture](#contract-architecture)
- [Unified Report](#unified-report)
- [Scanner Plugin System](#scanner-plugin-system)
- [Security Model](#security-model)
- [Authenticated Scanning](#authenticated-scanning)
- [Storage Model](#storage-model)
- [Persistence](#persistence)
- [Clients](#clients)
- [Observability](#observability)
- [Deployment Topology](#deployment-topology)
- [Known Limitations](#known-limitations)
- [Failure Modes](#failure-modes)
- [Testing Architecture](#testing-architecture)

---

## System Goals

StageFlow is designed around four goals:

1. **Safe intake** — Treat all submitted URLs and archives as untrusted.
2. **Isolation** — Run extraction and scanners in scoped job containers.
3. **Deterministic orchestration** — Drive jobs through an explicit state machine.
4. **Actionable output** — Normalize multiple scanner outputs into one report contract.

### Non-Goals

- Not a hosted multi-tenant control plane by default.
- Not a replacement for edge rate limiting or perimeter security controls.

---

## Platform Shape

```
  Clients                      Infrastructure
┌──────────┐   ┌──────────────────────────────────────────────┐
│ Web      │   │  NATS JetStream · MinIO · PostgreSQL · Grafana│
│ (React   │   └──────────────────────────────────────────────┘
│  Router) │
│          │   ┌──────────────────────────────────────────────┐
│ Go CLI   ├──►│ Platform API (Go)                            │
└──────────┘   │  URL/ZIP intake + SSRF validation            │
               │  Job/report APIs + SSE hub                   │
               │  Project CRUD + baseline promotion + diffs   │
               └──────────────────┬───────────────────────────┘
                                  │ NATS events (jobs/extraction/scan)
               ┌──────────────────▼───────────────────────────┐
               │ Orchestrator (Go)                            │
               │  Job FSM (PENDING → DONE/FAILED)             │
               │  Podman pod lifecycle per job                │
               │  Report aggregation + deduplication          │
               │  PostgreSQL job state + event audit trail    │
               │  Deadline sweeper for stuck jobs             │
               └──────────────────┬───────────────────────────┘
                                  │ one rootless pod per job
               ┌──────────────────▼───────────────────────────┐
               │ Podman Job Pod                               │
               │  Archive Extractor (Go, ZIP jobs)            │
               │  Scanner Runner ×N (TS/Bun/Playwright)       │
               └──────────────────────────────────────────────┘
```

| Directory                    | Responsibility                                                          |
| ---------------------------- | ----------------------------------------------------------------------- |
| `services/platform-api`      | Intake, validation, status APIs, SSE, project management, diff engine   |
| `services/orchestrator`      | FSM, container lifecycle, aggregation, PostgreSQL persistence           |
| `services/archive-extractor` | Secure archive extraction and provenance generation                     |
| `services/scanner-runner`    | Plugin discovery and scanner execution runtime                          |
| `clients/web`                | Submission UX and live status/report views (React Router v7 SPA)        |
| `clients/cli`                | Go CLI — scan submission, SSE streaming, report rendering, project mode |
| `libs/contracts`             | JSON Schemas and generated contracts (Go + TypeScript)                  |
| `libs/go/*`                  | 13 shared Go packages (messaging, models, config, etc.)                 |

All long-running containers and per-job pods run rootless with `no-new-privileges:true` and resource limits. Environment variables for every service are documented in the [configuration reference](reference/configuration.md).

---

## Core Services

### Platform API (`services/platform-api`)

The public HTTP boundary. Every request passes a middleware stack of logging → CORS → API-key auth → rate limiting → timeout (SSE excepted). It validates intake (request shape, URL count ≤ 100, SSRF classification, scanner configs against manifests), publishes `job.created` to NATS, and answers status queries from an event-sourced SQLite projection updated by job events. It also owns project CRUD, baseline promotion, and the on-demand diff engine (`libs/go/diff`). Key surfaces:

- `POST /api/v1/jobs/urls`, `POST /api/v1/jobs/urls/anonymous`, `POST /api/v1/jobs/urls/browser-auth`, and `POST /api/v1/jobs/zip` — intake; the browser-auth route is the deliberately narrow public form-login flow, while storage state and environment references require a caller API key
- `GET /api/v1/jobs/{id}` / `…/stream` (SSE) / `…/report` / `…/results` / `…/diff`
- `GET|POST|PATCH|DELETE /api/v1/projects…`, `POST …/{slug}/scan`, `POST …/{slug}/promote`
- `GET /api/v1/scanners`, unauthenticated `GET /healthz`

SSRF checks live in `internal/api/security.go`; routing and middleware in `internal/api/router.go`.

### Orchestrator (`services/orchestrator`)

The central coordinator, structured as clean architecture: domain policies (`internal/domain/jobs` — FSM transitions, completion policy, failure policy), application services, and adapters for PostgreSQL, Podman, NATS, and MinIO. It consumes `job.created`, drives the state machine, creates one Podman pod per job, launches one scanner-runner container per enabled scanner, tracks expected vs. completed scanners in PostgreSQL, and — when all scanners finish — downloads their results, merges/normalizes/deduplicates them into the unified report, and publishes `job.completed`. A token-authenticated internal API exposes jobs, events, pods, and system status for operations; a deadline sweeper fails stuck jobs.

### Archive Extractor (`services/archive-extractor`)

Runs only for ZIP jobs, inside the job pod. Lifecycle: download the ZIP from the MinIO staging bucket → validate safety constraints (entry-count limits, per-entry and total size caps, compression-ratio checks against ZIP bombs, path-traversal prevention) → extract into an isolated workspace → discover HTML pages → generate and upload `provenance.json` → serve the extracted site from an embedded static HTTP server → publish `extraction.ready` or `extraction.failed`.

### Scanner Runner (`services/scanner-runner`)

TypeScript/Bun runtime that executes one scanner per container using Playwright. `ScannerBase` defines the lifecycle: `initialize()` → concurrent page iteration (publishing `scan.page.completed` per page) → `writeResults()` → `uploadArtifacts()` → `scan.completed`. Plugins are discovered from embedded built-ins and mounted plugin paths (see [Scanner Plugin System](#scanner-plugin-system)). Seven built-in scanners: axe, lighthouse, seo, security-headers, link-checker, open-graph, spelling-grammar.

---

## Event Model

NATS JetStream carries all inter-service communication across three streams (each with 72h max age):

| Stream       | Subjects                                                                    | Producer → Consumer                 |
| ------------ | --------------------------------------------------------------------------- | ----------------------------------- |
| `JOBS`       | `jobs.events.created`, `jobs.events.completed`, `jobs.events.failed`        | platform-api ↔ orchestrator         |
| `EXTRACTION` | `extraction.events.ready`, `extraction.events.failed`                       | extractor → orchestrator            |
| `SCAN`       | `scan.events.page.completed`, `scan.events.completed`, `scan.events.failed` | scanner-runner → orchestrator + API |

Consumers are durable with explicit ACK, max 10 deliveries, 10-minute ACK wait, and a 5-second NAK delay on handler failure.

Every message is an envelope — `event`, `job_id`, optional `request_id`/`run_id`, `timestamp`, `producer`, `payload` — with a deliberate strictness asymmetry:

| Operation                  | Strictness                             | Rationale                            |
| -------------------------- | -------------------------------------- | ------------------------------------ |
| **Publishing**             | Strict — payload `Validate()` required | Reject invalid events before sending |
| **Subscribing (envelope)** | Lenient — unknown fields allowed       | Forward-compatible event evolution   |
| **Subscribing (payload)**  | Strict — `DisallowUnknownFields()`     | Catch schema drift in payloads       |

`libs/go/messaging` wraps this in a generic `SubscribeTyped[T]` that manages durable consumers, parses envelope + payload, attaches event metadata and job/request/run IDs to the logging context, and NAKs on handler failure.

---

## Data Flows

### URL Job Flow

```
Client ── POST /api/v1/jobs/urls[/anonymous|/browser-auth] {urls, modules} ──► Platform API
  │  validate shape → SSRF-check each URL → normalize modules
  │  → validate scanner configs → publish job.created
  │  → seed PENDING status projection → return {job_id}
  ▼
Orchestrator (consumes job.created)
  │  PENDING → READY_TO_SCAN (URL jobs skip extraction)
  │  create Podman pod, launch one scanner container per module,
  │  record expected scanners in PostgreSQL
  ▼
Scanner containers (axe, seo, lighthouse, …) run in parallel
  │  scan.page.completed per page → orchestrator updates progress
  │  scan.completed per scanner  → orchestrator marks scanner done
  ▼
Orchestrator (all expected scanners complete)
  │  SCANNING → COMPLETING: download all results from MinIO,
  │  merge pages/issues, normalize severities, deduplicate,
  │  recompute summaries/scores, upload unified report
  │  COMPLETING → DONE, publish job.completed with artifact keys
  ▼
Platform API (consumes job.completed)
     update projection → presign artifact URLs → push SSE update
```

### ZIP Job Flow (differences)

The ZIP is stored in the MinIO staging bucket at intake. The orchestrator first starts the archive-extractor container in the job pod (`PENDING → EXTRACTING`); on `extraction.ready` it sets `TotalPages` from the provenance document and continues exactly as the URL flow, with scanners pointed at the extractor's in-pod static server.

### CLI Flows

`stageflow scan` submits the job, streams SSE progress to stderr (with reconnect buffering), downloads the unified report on the terminal state, wraps it in a versioned CLI envelope, filters/sorts issues, renders text/markdown/JSON, and exits 0/1 by `--fail-on` threshold (2 on CLI/API error). `stageflow dev scan` additionally manages a local dev-server lifecycle around the scan (start → poll ready URL → scan → SIGINT/SIGKILL teardown). `stageflow project scan` resolves a stored project, scans it, and — if a baseline is promoted — fetches the server-side diff and exits 1 on any new issue, which is the CI regression gate.

---

## Job State Machine

```
PENDING → EXTRACTING → READY_TO_SCAN → SCANNING → COMPLETING → DONE
   │           │             │             │           │
   └───────────┴─────────────┴─────────────┴───────────┴──► FAILED
                     (URL jobs skip EXTRACTING)
```

| State           | Meaning                               | Triggered By                                  |
| --------------- | ------------------------------------- | --------------------------------------------- |
| `PENDING`       | Accepted and queued                   | API job submission                            |
| `EXTRACTING`    | ZIP processing in extractor container | Orchestrator starts extractor (ZIP jobs only) |
| `READY_TO_SCAN` | Scanner inputs are ready              | `extraction.ready` event or URL job skip      |
| `SCANNING`      | One or more scanner containers active | Orchestrator launches scanners                |
| `COMPLETING`    | Output merge/dedup/report publish     | All scanners complete                         |
| `DONE`          | Successful terminal state             | Report aggregated and uploaded                |
| `FAILED`        | Terminal failure state                | Any unrecoverable error                       |

The transition table is defined once, in `libs/go/domain/job/state.go`, and shared by every service. Database updates additionally guard against state regressions at the SQL level via a `StateRankSQL()` CASE expression, so a delayed or replayed event can never move a job backwards.

---

## Contract Architecture

StageFlow is **schema-first**: JSON Schema files in `libs/contracts/` are the single source of truth, with code generated for both languages.

```
JSON Schema (libs/contracts/*/schema/)
    ├──► atombender/go-jsonschema ──► Go types with custom UnmarshalJSON
    │                                  (required fields, min/max, patterns, enums)
    └──► json-schema-to-typescript ──► TypeScript types
```

Three contract families:

- **`report/`** — `unified-report.v2.schema.json`, the canonical scan output every scanner feeds and every client renders.
- **`scanner-manifest/`** — the plugin descriptor format, validated at plugin load time.
- **`events/`** — schemas and fixtures for the scan event payloads.

Shared enums (`IssueSeverity`, `ScannerStatus`, `ScannerCategory`, `ErrorScope`, `UserGroup`, …) appear identically across families and languages, so a severity means the same thing in a Go event, a TypeScript scanner, and the web UI. Generated code is a build artifact: run `just generate-contracts` before building packages that import it. Go-side payloads add semantic validation beyond the schema (e.g. `ScanTiming` component sums must not exceed the total; provenance paths must start with `/`).

---

## Unified Report

`UnifiedReportV2` is one document per job: `meta` (job, base URL, timings), `summary` (issue counts by severity/scanner, pages scanned, optional 0–100 score with grade), per-`scanner` status, per-`page` summaries (including screenshot overlays with per-issue bounding boxes), flat `issues`, optional `artifacts`, and structured `errors` with scanner/page/global scope and a `retryable` flag. The full field reference is the schema itself: `libs/contracts/report/schema/unified-report.v2.schema.json`; a complete committed fixture (`report/fixtures/unified-report.v2.all-scans.json`) doubles as executable documentation and drives the web client's tests.

Two properties make the report more than a log of findings:

- **Stable issue identity** — each issue's `id` is a content hash of its rule, page, and element context, so the same violation yields the same ID across runs. Baseline diffing, regression detection (new IDs = regressions, missing IDs = fixes), and deduplication all key off it.
- **Cross-scanner deduplication** — during aggregation the orchestrator merges equivalent findings reported by different scanners (`rule_deduplication.go`) so one underlying defect is one issue.

---

## Scanner Plugin System

Scanners are plugins described by a `manifest.json` conforming to the scanner-manifest schema: identity, categories, capabilities (browser requirement, concurrency, screenshots), a `configSchema` for per-scanner options, an entry module, and severity/category output mapping. Discovery order:

1. **Built-ins** — embedded at compile time via `//go:embed` from `libs/go/scannercatalog/manifests/`, validated against the schema at load.
2. Mounted `/plugins` volume.
3. `~/.stageflow/plugins/`.
4. `PLUGIN_PATHS` (colon-separated).

`libs/go/scannerregistry` provides runtime resolution — by ID or alias, lenient (`ResolveModules`, unknown tokens pass through) or strict (`ResolveModulesStrict`) — and a public `Info` projection that strips internal fields (image, aliases, config, requirements) from API responses. The default module is `axe`.

---

## Security Model

Four nested trust boundaries, outermost first:

1. **Edge proxy** — Caddy (`infra/caddy/Caddyfile`): TLS termination, security headers, server-side credential injection for an exact allowlist of browser submission, job/report/SSE, and scanner-catalog routes, plus routing (`/api/*` → platform-api, `/scanner-artifacts/*` → MinIO, `/monitoring*` → Grafana, `/*` → frontend). Project, baseline, diff, caller-authenticated URL submission, and all unmatched API routes require a caller-provided API key. The anonymous endpoint rejects every auth recipe; the browser-auth endpoint accepts only literal form steps and rejects storage state, environment references, and private targets. Both share the Platform API's trusted-proxy-aware public-submission limiter. Operators may add a shared edge limiter for multi-instance or higher-risk deployments.
2. **API intake** — scheme validation (http/https only), SSRF classification (below), request size limits (2MB URL submissions, 100MB ZIP), URL count ≤ 100 and length ≤ 2048, API-key middleware, rate limiting, timeouts.
3. **Archive extraction** — the ZIP safety controls listed under [Archive Extractor](#archive-extractor-servicesarchive-extractor), in an isolated workspace.
4. **Scanner runtime** — per-job rootless Podman pods, `no-new-privileges:true`, resource limits, scanner identity validated against its manifest, `SCANNER_OPTIONS` schema-validated, artifact upload only through storage interfaces.

### SSRF Protection

`services/platform-api/internal/api/security.go` classifies every resolved IP of every submitted URL:

| Decision                  | Ranges                                                                                                                                     |
| ------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| **Block (always)**        | `0.0.0.0/8`, `100.64.0.0/10`, `169.254.0.0/16` (cloud metadata), `192.0.0.0/24`, doc/benchmark/multicast/reserved ranges, IPv6 equivalents |
| **Allow in private mode** | `10.0.0.0/8`, `127.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `::1/128` — only with `--allow-private-targets`                            |
| **Allow**                 | All other public IPs                                                                                                                       |

Validation resolves the hostname via DNS and rejects if _any_ resolved IP is blocked (or private outside private mode). All CIDR ranges are parsed at startup by `ValidateSecurityConfig()`, failing fast on invalid entries. The scanner-runner re-applies the same target policy at browser runtime — initial targets, redirects, final URLs, and HTTP(S) subresources. This is not connection-level DNS pinning; [SECURITY.md](../SECURITY.md) documents the residual DNS-rebinding risk, and public deployments should add a container/host egress policy ([infra/security/egress-policy.example.md](../infra/security/egress-policy.example.md)).

### Additional Measures

`gitleaks` on every commit and in CI, `govulncheck` across all Go modules, a moderate-level scanner dependency audit with the exact documented OpenTelemetry compatibility exception, and a high-level web dependency audit; container log rotation limits; MinIO credential aliasing; request timeouts on every endpoint except SSE.

---

## Authenticated Scanning

StageFlow can scan behind a login by attaching an optional `auth` block to a job's Provenance — a discriminated union with two modes:

- **`form`** — a login URL plus recorded steps (`fill`/`click`) and a success condition. Caller-authenticated API and CLI workflows should use `{from_env: NAME}` references, resolved only inside the scanner-runner against an allow-list derived from the recipe. The public web browser-auth endpoint accepts literal values for throwaway demo accounts; it rejects environment references, storage state, and private targets.
- **`storage_state`** — an `artifact_key` pointing at captured Playwright storage-state JSON under the job's MinIO prefix.

The credential-handling invariants are the point of the design:

- `stageflow auth capture` writes storage state locally with mode 0600, keeping the password itself out of the CLI and API. The optional public web form-login flow is a separate, explicitly warned path that does receive literal credentials.
- The Platform API validates shape and size (1 MiB cap), uploads storage-state bytes to MinIO, and publishes only the artifact key — it never resolves a credential.
- The orchestrator walks `form` steps for `from_env` references and forwards **exactly those** env-var names into the scanner pod; everything else in the host environment stays out. Unresolved references fail fast before the pod starts. Event audit rows redact any inline content.
- The scanner-runner hydrates auth once per scanner (downloading storage-state to mode-0600 workspace files, or replaying the form recipe with the allow-listed secrets resolver), keeps resolved values in process memory only, applies the SSRF policy to the login URL and every subsequent navigation, and deletes credential files in `cleanup()`.
- Resolved credentials never appear in public Provenance, the unified report, stage logs, screenshots of sensitive controls, or terminal database event records. The public form-login recipe necessarily crosses the file-backed `job.created` JetStream message (configured with a 72-hour maximum age) and stays in the live job configuration until terminal cleanup scrubs it. The storage-state object is deleted when the job reaches `DONE`/`FAILED` and is never exposed through the public artifact surface.
- Hydration failure surfaces a `critical` `auth-hydration-failed` issue and skips that scanner's pages rather than scanning the logged-out surface silently.

Implementation: `services/scanner-runner/src/core/{auth-hydrator,secrets-resolver,page-iterator}.ts`, `clients/cli/cobra_auth.go` + `clients/cli/internal/authintake/`, the orchestrator's `scanner_launch_planner.go`, and the shared `from_env` walker `libs/go/provenance/auth.go` (a direct port of the TS resolver, kept in sync by fixture-driven tests).

---

## Storage Model

MinIO uses `scanner-staging` for transient ZIP uploads, `scanner-artifacts` for expiring scan outputs, and lifecycle-exempt `scanner-baselines` for reports explicitly promoted as project baselines. Ordinary object keys are job-prefixed; baseline keys include both project and job identity:

```
staging/{jobID}/{filename}              Uploaded ZIP
{jobID}/report.json | report.html       Unified report
{jobID}/provenance.json                 Page provenance
{jobID}/stage.log | recipe.json         Execution logs/recipes
{jobID}/{scanner}/results.json          Per-scanner results
{jobID}/{scanner}/screenshots/…         Per-scanner screenshots
{projectID}/{jobID}/report.json         Persistent promoted baseline
```

Services access storage only through the `libs/go/storage` client interface (upload/download/delete/presign/exists). Clients never touch MinIO directly: the Platform API hands out time-limited presigned URLs (generated against `MINIO_PUBLIC_ENDPOINT` when the public Caddy route is configured).

---

## Persistence

- **PostgreSQL (orchestrator)** — the authoritative `jobs` table (state, config, scanner tracking, artifact keys, per-stage timings) plus an append-only `job_events` audit table recording every consumed NATS message with its stream/consumer sequence numbers, delivery count, and handler outcome — a complete replayable timeline per job. Schema: `services/orchestrator/internal/adapters/repository/schema.sql`.
- **SQLite (platform-api)** — two small databases: a job-status projection (the read model behind status queries and SSE), and the project store (`projects` with slug/URLs/scanners/`baseline_job_id`, and a `project_jobs` junction). Schemas: `services/platform-api/internal/{status,project}/schema.sql`.

---

## Clients

### CLI (`clients/cli`)

A Cobra-based Go binary with a minimal `main.go` and a testable
`internal/command.Run(args, getenv, stdout, stderr)` entry point — all
dependencies injected, no globals. Flag precedence is
`CLI flags > .stageflow/config.yaml > env vars > defaults`. `stageflow dev init`
bootstraps config by auto-detecting dev commands from Justfile recipes and
`package.json` scripts, the package manager from lockfiles, and the dev URL.
Reports render as text, markdown, or versioned JSON envelopes; exit codes are
the CI interface.

### Web App (`clients/web`)

React Router v7 SPA, styled with repo-owned CSS modules and global design tokens (no CSS framework), built to static assets served by nginx. `app/lib/` separates the API client, scanner domain presets, report logic (filtering/grouping/severity), and types that alias the canonical report contract. Live scans subscribe to the SSE stream with auto-reconnect and event buffering.

---

## Observability

| Signal                     | Source                    | Access                                  |
| -------------------------- | ------------------------- | --------------------------------------- |
| Job state transitions      | Platform API SSE          | `/api/v1/jobs/{id}/stream`              |
| Orchestrator event history | PostgreSQL `job_events`   | Internal API `/api/v1/jobs/{id}/events` |
| Prometheus metrics         | Orchestrator (in-process) | Authenticated `/metrics`                |
| Scanner artifacts          | MinIO                     | Presigned URLs via API                  |
| Service logs               | Container stdout/stderr   | `just dev logs`                         |

The orchestrator exposes job-state and pod gauges plus event-handler counters and a latency histogram, collected in-process without a metrics-client dependency. Grafana dashboards (`infra/grafana/provisioning/`) chart job counts, completion rates, timing breakdowns, and extraction success. `devtools/ops/job-status-cli` inspects jobs/events/pods via the admin API; `devtools/qa/suite-runner` runs threshold-based multi-domain validation.

Debug path: check the job's terminal state and latest event → inspect the orchestrator timeline for the failing stage boundary → pull scanner artifacts → correlate service logs by event timestamps.

---

## Deployment Topology

The repository supports a normal local demo, a private-target local overlay, and a compose-based self-hosting example. The hosted `stageflow.org` service uses the same application code but a separately managed gateway and Quadlet deployment; it does not mirror the checked-in compose topology. See [Self-hosting](self-hosting.md) for the supported layouts and security checklist.

### Horizontal Scaling Boundary

The topology is intentionally single-instance for the Platform API and Orchestrator. PostgreSQL, NATS, and MinIO are shared infrastructure, but SSE fanout is process-local, admin rate limiting is in-memory, and Podman job ownership assumes one orchestrator process. Running replicas would require a shared status/pub-sub fanout for SSE, a lease-based runtime owner for job pods, and shared rate limiting.

---

## Known Limitations

- DNS and browser-request validation reduce SSRF risk but do not provide connection-level DNS pinning. Public operators still need egress policy that blocks private, metadata, and link-local destinations.
- The reference public edge uses one server-side Platform API credential. It is a gateway trust boundary, not per-user identity or authorization.
- SSE fanout, request limiting, and job-pod ownership assume one Platform API and one Orchestrator instance. Horizontal scaling needs shared coordination as described above.
- The hosted demo retains anonymous uploads and generated artifacts for 24 hours; it is not intended for confidential builds or sensitive authenticated targets. See [Privacy](privacy.md).
- Submitted URLs and job configuration remain in durable job records in this release; event-audit records default to 30-day retention. Operators that need stricter metadata deletion must add database pruning or anonymization.
- Production gateway, rollout, monitoring, and rollback automation for `stageflow.org` live outside this application repository. The checked-in Caddy and compose files are self-hosting references, not a reproduction of that host.

---

## Failure Modes

| Stage              | Representative failures                                                                      | Outcome                                         |
| ------------------ | -------------------------------------------------------------------------------------------- | ----------------------------------------------- |
| **Intake**         | Invalid URL/scheme, SSRF violation, oversize payload, unknown module, invalid scanner config | Rejected with a structured 4xx — no job created |
| **Extraction**     | ZIP bomb, path traversal, archive limits exceeded                                            | Job → FAILED with extraction-stage error        |
| **Scanning**       | Scanner crash, manifest mismatch, browser timeout, page instability                          | `scan.failed` event → job failure or partial    |
| **Aggregation**    | Missing artifact, contract/merge incompatibility                                             | Job → FAILED                                    |
| **Infrastructure** | NATS/Postgres/Podman socket unavailable                                                      | Services fail fast at startup                   |
| **Stuck jobs**     | Scanner never reports; pod dies silently                                                     | Deadline sweeper transitions the job to FAILED  |

MinIO is the one dependency retried at startup (bucket ensure: 30 retries, 2s delay); everything else fails fast so the supervisor restarts a well-defined world.

---

## Testing Architecture

```
Golden Regression   full pipeline: scan → baseline → diff → compare
E2E Acceptance      live API URL/ZIP scans (RUN_E2E=1)
Integration         orchestrator end-to-end with mock Podman
Unit                Go tests with -race · Vitest · frontend typecheck/build
```

E2E tests are double-gated (`testing.Short()` plus a `RUN_E2E` env check), so `go test ./...` stays fast by default.

The **golden regression** (`qa/e2e/project-scan-golden.sh`, run in CI daily) exercises the product's core promise end-to-end: create a project → baseline scan of a clean fixture page → promote → point the project at a fixture with a known `image-alt` violation → regression scan must exit 1 → normalize the JSON output (strip timestamps/IDs/durations) → `diff -u` against committed golden fixtures in `qa/fixtures/project-golden/`. Regenerate intentionally with `UPDATE_GOLDENS=1`.

CI (`.github/workflows/`): the main pipeline gates workflow lint → secrets scan → Go build/lint/test/vulncheck → web CI and Playwright/axe E2E → scanner-runner CI → dead-code check → image builds and security scans; the CLI release workflow cross-builds 5 OS/arch targets on `clients/cli/v*` tags. `just ci` is the fast local quality gate; hosted CI adds secret scanning, browser E2E, image builds, SBOMs, and Trivy. Go linting runs 30+ linters via `.golangci.yml`, spanning correctness (errcheck, staticcheck, nilerr), security (gosec, depguard), and quality (gocritic, revive, gocyclo).
