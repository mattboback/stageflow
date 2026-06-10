# StageFlow Architecture

StageFlow is a self-hosted frontend quality gate built from multiple Go services, a TypeScript/Bun scanner runtime, a SvelteKit 5 web app, and a Go CLI. This document covers the system shape, data flow, and the reasoning behind the key design decisions.

For an exhaustive deep-dive — trust boundaries, failure modes, all event types, deployment topology — see [docs/architecture/system.md](docs/architecture/system.md).

---

## System Topology

```
┌─ Clients ──────────────────────────────────────────────────────────────────┐
│                                                                            │
│   ┌──────────────────────┐       ┌──────────────────────┐                 │
│   │  Web App             │       │  CLI                 │                 │
│   │  SvelteKit 5         │       │  Go                  │                 │
│   │  Scan submission     │       │  Dev loop            │                 │
│   │  Live status (SSE)   │       │  JSON output         │                 │
│   │  Report exploration  │       │  Severity exit codes │                 │
│   └──────────┬───────────┘       └──────────┬───────────┘                 │
│              └──────────────┬───────────────┘                             │
└───────────────────────────  │  ──────────────────────────────────────────┘
                              │  HTTP + SSE
                              ↓
┌─ Platform API (Go) ─────────────────────────────────────────────────────────┐
│                                                                             │
│  POST /v1/scan/url        POST /v1/scan/zip                                 │
│  GET  /v1/jobs/:id/status (SSE)                                             │
│  GET  /v1/jobs/:id/report                                                   │
│  POST /v1/projects        GET /v1/projects/:slug/diff                       │
│                                                                             │
│  Middleware: request IDs, logging, CORS, API key auth, rate limiting        │
│  SSRF guard: blocks private IPs, non-HTTP schemes, metadata endpoints       │
│  Archive guard: file count, total size, nesting depth, ZIP bomb detection   │
│                                                                             │
└──────────────────────────────┬──────────────────────────────────────────────┘
                               │ publishes job.created → NATS JetStream
                               ↓
┌─ NATS JetStream ────────────────────────────────────────────────────────────┐
│  Streams: jobs | extraction | scan                                          │
│  8 event types with typed envelopes and explicit ACK/NAK                    │
│  Durable consumers → Orchestrator can replay events after restart           │
└──────────────────────────────┬──────────────────────────────────────────────┘
                               │ consumes events
                               ↓
┌─ Orchestrator (Go) ─────────────────────────────────────────────────────────┐
│                                                                             │
│  Job FSM: PENDING → EXTRACTING → READY_TO_SCAN →                            │
│           SCANNING → COMPLETING → DONE/FAILED                               │
│  Podman pod lifecycle: launch, monitor, cleanup per job                     │
│  Report aggregation: merges scanner results into unified report             │
│  PostgreSQL: persists job state + event audit trail                         │
│                                                                             │
└──────────────────────────────┬──────────────────────────────────────────────┘
                               │ launches ephemeral rootless pod
                               ↓
┌─ Job Pod (per-job, Podman, rootless) ───────────────────────────────────────┐
│                                                                             │
│  ┌─ Archive Extractor (Go) ──────────────────────────────────────────────┐ │
│  │  Safe ZIP extraction (path traversal checks, size limits, nesting)    │ │
│  │  Provenance generation (URLs, auth mode, archive metadata)            │ │
│  │  Static HTTP server for extracted files                               │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│  ┌─ Scanner Runner (TypeScript / Bun / Playwright) ──────────────────────┐ │
│  │                                                                       │ │
│  │  Plugin discovery → manifest validation → factory loading             │ │
│  │                                                                       │ │
│  │  Axe            accessibility (WCAG 2.1 A/AA/AAA)                    │ │
│  │  Lighthouse     performance, best practices                           │ │
│  │  SEO            markup and meta checks                                │ │
│  │  Link Checker   broken internal and external links                    │ │
│  │  Security Headers  CSP, HSTS, X-Frame-Options, etc.                  │ │
│  │  Open Graph     social preview quality                                │ │
│  │  Spelling/Grammar  content quality                                    │ │
│  │  AI Navigator   natural language quality objectives via LLM           │ │
│  │                                                                       │ │
│  │  Per page: screenshot capture → MinIO                                 │ │
│  │  Per scanner: results.json + report.html → MinIO                      │ │
│  │  Events: scan.completed / scan.page.completed → NATS                  │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘

                    ┌───────────────────────────┐
                    │  Infrastructure           │
                    │  NATS JetStream 2.12      │
                    │  PostgreSQL 17            │
                    │  SQLite (projects)        │
                    │  MinIO (S3-compatible)    │
                    │  Grafana 12 (optional)    │
                    └───────────────────────────┘
```

---

## Service Responsibilities

### Platform API

The HTTP boundary for all external traffic. It validates and accepts scan requests, enforces SSRF and archive safety, publishes `job.created` to NATS, provides the SSE hub for live job progress, and serves completed reports and baseline diffs. It is intentionally thin: it does not run scanners or manage infrastructure — that belongs to the Orchestrator.

Key files: `services/platform-api/internal/api/`, `services/platform-api/cmd/server/main.go`

### Orchestrator

Consumes NATS events and drives the job state machine. When a `job.created` event arrives, the Orchestrator allocates a Podman pod, coordinates extraction and scanning, collects scanner events as they complete, aggregates results into the unified report, and persists all state transitions to PostgreSQL. The FSM is the heart of this service: all business logic about when a job succeeds, fails, or retries lives here.

Key files: `services/orchestrator/internal/domain/jobs/`, `services/orchestrator/internal/adapters/`

### Archive Extractor

Runs inside a job pod for ZIP-mode scans. It safely extracts archives (enforcing path traversal checks, file count limits, size limits, and nesting depth), generates a provenance record, and starts a simple HTTP server so the Scanner Runner can access the extracted files at `localhost`. Isolation in a separate container means malicious archive content cannot reach the host filesystem.

Key files: `services/archive-extractor/internal/extractor/`, `services/archive-extractor/internal/server/`

### Scanner Runner

The TypeScript/Bun runtime that actually runs the eight scanners. It discovers scanner plugins from manifests on the filesystem, validates configuration schemas, and executes each scanner against the target URLs using Playwright. For every page it captures a screenshot; for every scanner it produces `results.json` (structured data) and `report.html` (human-readable). Both are uploaded to MinIO and announced to NATS.

Key files: `services/scanner-runner/src/worker.ts`, `services/scanner-runner/src/core/plugins/`, `services/scanner-runner/src/scanners/`

### Web App

SvelteKit 5 frontend with three main routes: playground (scan submission), scan status (SSE-driven live view), and report (issue exploration). The report view renders the unified report contract — it contains no scanner-specific branches; adding a new scanner does not require frontend changes.

Key files: `clients/web/src/routes/`, `clients/web/src/lib/stores/`

### CLI

Go binary for terminal-first workflows. Commands: `scan`, `dev` (init/doctor/scan), `project` (create/list/show/update/delete/promote/scan), `auth capture`, `diff`, `report`, `scanners`. Output formats: human text, markdown, JSON. The dev loop (`stageflow dev scan`) automates the full cycle: start dev server → wait for readiness → submit scan → stream results → stop server — all driven by `.stageflow/config.yaml`. Exit codes are machine-readable: `0` (pass), `1` (severity gate), `2` (error).

Key files: `clients/cli/` (Cobra command files, e.g. `project_run.go`), `clients/cli/internal/projectmode/`

---

## Data Flow: URL Scan End-to-End

```
Client                Platform API         NATS         Orchestrator       Job Pod
  │                        │                │                │                │
  │── POST /v1/scan/url ──▶│                │                │                │
  │                        │─ validate ─────│                │                │
  │                        │─ job.created ─▶│                │                │
  │◀── 202 { jobId } ──────│                │─ job.created ─▶│                │
  │                        │                │                │─ launch pod ──▶│
  │── GET /v1/jobs/:id ───▶│ (SSE stream)   │                │                │
  │◀── job:started ─────────────────────────────────────────│                │
  │◀── scan:started ────────────────────────────────────────│◀─ event ───────│
  │◀── scan:page ───────────────────────────────────────────│◀─ event ───────│
  │◀── scan:completed ──────────────────────────────────────│◀─ event ───────│
  │                        │                │                │─ aggregate ────│
  │                        │                │                │─ write report ─│
  │◀── job:done ────────────────────────────────────────────│                │
  │                        │                │                │─ cleanup pod ─▶│
  │── GET /v1/jobs/:id/report ────────────▶ │                │                │
  │◀── unified report (JSON) ──────────────│                │                │
```

---

## Core Design Decisions

### 1. Explicit Job State Machine

The Orchestrator models job lifecycle as a proper FSM with defined states, valid transitions, terminal states, and completion policies. This means:

- Invalid transitions are rejected at the code level, not silently ignored
- Terminal states (`DONE`, `FAILED`) prevent duplicate events from triggering re-processing
- Completion policy (did all required scanners finish? partial success OK?) is a single focused unit of logic
- The FSM is tested independently of Podman, NATS, and PostgreSQL

**Why not a flags-and-timestamps approach?** Implicit state encoded across multiple database columns makes legal state combinations hard to enumerate and easy to get wrong. An explicit FSM surfaces the invariants.

### 2. NATS JetStream for Orchestration Events

The Platform API publishes events; the Orchestrator subscribes. They share no direct call path. This decoupling means:

- The API stays thin and fast — it hands off work immediately after validation
- The Orchestrator can restart and replay missed events from durable streams without any database polling or state reconciliation
- New consumers (future audit service, notifications) can subscribe without touching existing code

**Why not a traditional job queue?** NATS JetStream's durable streams and replay semantics fit event-sourcing-style orchestration better than a queue that discards events on ACK. The Orchestrator's event audit trail in PostgreSQL is a consequence of NATS — not a replacement for it.

### 3. SSE over WebSocket

Job progress is one-directional: server to client. Server-Sent Events require no protocol upgrade, work through most reverse proxies and CDNs, and are natively reconnectable. The SSE hub buffers recent events so a client that reconnects mid-job catches up.

**Why not WebSocket?** WebSocket's bidirectional capability is unnecessary overhead for this use case, and its proxy handling is less reliable in self-hosted environments.

### 4. Per-Job Podman Pods (Rootless)

Every scan runs in an ephemeral pod that is created for the job and destroyed after it completes. Podman runs rootless with `no-new-privileges:true` and per-pod CPU and memory limits.

**Why per-job and not a shared worker pool?** Isolation. A shared long-lived scanner worker means one job's misbehaving scanner can affect other jobs. Per-job pods contain blast radius by design. The startup overhead is acceptable given that scans are not sub-second operations.

**Why Podman instead of Docker?** Rootless Podman requires no daemon and no privilege escalation, which simplifies the security posture for self-hosted deployments.

### 5. Schema-First Contracts

The canonical report shape, scanner manifest format, provenance record, and NATS event envelopes are all defined as JSON Schemas in `libs/contracts/`. Go and TypeScript types are generated from these schemas and committed to the repo.

**Why generate types?** Because hand-written types drift. If the scanner runner emits a field the web UI's TypeScript types don't know about, the TypeScript compiler catches it at build time — not at runtime in production. Adding a new scanner field is a schema change, not a hunt for all places that parse the report.

### 6. Stable Content-Based Issue IDs

Every issue in the unified report is assigned a fingerprint: `sha256(ruleId + context + occurrence)`. This is deterministic — the same violation on the same page produces the same ID every time, regardless of scan order or scanner version.

**Why does this matter?** Baseline diffing requires knowing whether an issue in scan B is the same issue that appeared in scan A (already known), or a new regression. Unstable IDs would make every re-scan look like all issues disappeared and new ones appeared.

### 7. Clean Architecture in the Orchestrator

The Orchestrator uses a layered structure: Domain (FSM rules and policies) → Application (use cases) → Adapters (PostgreSQL, Podman, NATS). Dependency inversion means:

- Domain logic has no imports from infrastructure packages
- Integration tests can exercise the FSM with a fake repository and no real database
- Replacing Podman with a different runtime would touch one adapter, not the domain

### 8. Scanner Plugin System

The Scanner Runner discovers scanners from manifest files at known paths (built-ins, volume mounts, `~/.stageflow/plugins`, and `PLUGIN_PATHS`). Each manifest declares capabilities and a config schema. The runner validates `SCANNER_OPTIONS` against the schema at startup.

Third-party scanners can be added by dropping a manifest and factory into a mounted volume — no changes to the core runner code required.

---

## Data Model: Unified Report

The report schema (`libs/contracts/report/schema/unified-report.v2.schema.json`) defines the contract every scanner must satisfy:

```
UnifiedReport
├── meta         (jobId, scanType, timestamp, scannerVersions)
├── summary
│   ├── score    (0–100)
│   ├── grade    (A–F)
│   ├── totalIssues
│   ├── bySeverity  { critical, serious, moderate, minor, info }
│   ├── byScanner   { axe: N, lighthouse: N, ... }
│   └── lighthouseCategories  { performance, accessibility, seo, bestPractices }
├── scanners[]   (per-scanner summaries and metadata)
├── pages[]
│   ├── url
│   ├── screenshot  (MinIO presigned URL)
│   └── issues[]  (page-scoped subset)
└── issues[]
    ├── id          (stable content hash)
    ├── ruleId
    ├── scanner
    ├── severity    (critical | serious | moderate | minor | info)
    ├── title
    ├── description
    └── occurrences[]  (selector, context, evidence)
```

The web app's report view, the CLI's formatter, and the diff engine all work from this shape. None contain scanner-specific logic.

---

## Storage Boundaries

| Store | What lives there | Why |
|-------|-----------------|-----|
| **PostgreSQL** | Job rows, job_events audit trail | Durable state for Orchestrator; supports concurrent writes |
| **SQLite** | Projects, baselines, promoted reports | Self-hosting friendly; project data is single-writer |
| **MinIO (S3)** | Reports (JSON), screenshots, HTML reports, state files | Object storage for large/binary artifacts; presigned URLs keep clients away from the bucket |
| **NATS** | In-flight events, durable streams | Coordination only; not the source of truth for job state |

---

## Security Posture

**URL intake (SSRF)** — StageFlow blocks non-HTTP schemes, RFC 1918 private ranges, link-local addresses, and cloud metadata endpoints unless `ALLOW_PRIVATE_TARGETS=true`. Hostname resolution is re-validated against policy before connecting.

**ZIP intake** — File count, total uncompressed size, and nesting depth are enforced. ZIP bombs are detected by comparing reported vs actual sizes. Extraction runs in an isolated container so path traversal attempts cannot reach the host.

**Execution** — Rootless Podman pods with `no-new-privileges:true` and CPU/memory limits. Per-job isolation contains any scanner misbehavior.

**API boundary** — Middleware stack handles request IDs, structured logging, panic recovery, CORS, optional API key auth, and rate limiting (by IP or `X-Forwarded-For` from trusted proxies).

**Artifacts** — Presigned MinIO URLs are scoped to specific job object keys. Clients never have direct bucket access.

**CI** — `govulncheck` (Go CVEs), Trivy (container CVEs), `bun audit` (Node CVEs), gitleaks (secrets scanning).

---

## Testing Strategy

```
Layer                  Tooling                    What it validates
─────────────────────────────────────────────────────────────────────────────
Go unit                go test -race              FSM transitions, domain rules,
                                                  utilities — no infra required
Go lint/vuln           golangci-lint,             Code quality, known CVEs
                       govulncheck
Web unit               Vitest, Testing Library    Components, SSE stores,
                                                  report logic, scoring
Storybook CI           Storybook test runner      Component interaction,
                                                  accessibility (axe audit)
Scanner runner         Vitest                     Plugin loading, output
                                                  validation, config schemas
E2E golden flow        qa/e2e/ shell script       Baseline → promote → regression
                                                  diff → exit code assertions
Container security     Trivy, bun audit,          Image CVEs, dependency audit,
                       gitleaks                   secrets in source
```

The E2E golden flow (`just project-golden`) is the most valuable integration check: it exercises the full remote project loop (`stageflow project scan`), promotes a baseline, makes a breaking change, and asserts that the CLI exits `1` with the right regression output.

---

## Code Reference Map

| Question | Where to look |
|----------|--------------|
| How does a job move between states? | `services/orchestrator/internal/domain/jobs/transitions.go` |
| When is a job considered done? | `services/orchestrator/internal/domain/jobs/completion_policy.go` |
| How does the API validate URLs for SSRF? | `services/platform-api/internal/api/security.go` |
| How does the SSE hub buffer and replay events? | `services/platform-api/internal/jobstatus/` |
| How are scanner plugins discovered and loaded? | `services/scanner-runner/src/core/plugins/` |
| What is the full report shape? | `libs/contracts/report/schema/unified-report.v2.schema.json` |
| How does the CLI drive the dev loop? | `clients/cli/project_run.go`, `clients/cli/internal/projectmode/` |
| How are Podman pods launched and cleaned up? | `services/orchestrator/internal/adapters/runtime/` |
| How does the diff engine compare baselines? | `libs/go/diff/` |
| How are NATS streams created and consumed? | `libs/go/messaging/` |
