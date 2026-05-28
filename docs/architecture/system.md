# StageFlow Architecture

This document explains StageFlow's system design, trust boundaries, data flows, service responsibilities, deployment topology, and failure modes.

If you are still orienting yourself, start with the [repository README](../../README.md) for the product overview and fastest local setup path.

---

## Table of Contents

- [System Goals](#system-goals)
- [Platform Shape](#platform-shape)
- [Service Topology](#service-topology)
- [Core Services](#core-services)
- [Data Models](#data-models)
- [Event Model](#event-model)
- [Data Flows](#data-flows)
- [Job State Machine](#job-state-machine)
- [Contract Architecture](#contract-architecture)
- [Unified Report](#unified-report)
- [Scanner Plugin System](#scanner-plugin-system)
- [AI Navigator](#ai-navigator)
- [Security Model](#security-model)
- [Authenticated Scanning](#authenticated-scanning)
- [Storage Model](#storage-model)
- [Database Schemas](#database-schemas)
- [CLI Architecture](#cli-architecture)
- [Web App Architecture](#web-app-architecture)
- [Observability](#observability)
- [Deployment Topology](#deployment-topology)
- [Failure Modes](#failure-modes)
- [Testing Architecture](#testing-architecture)
- [Code Reference Map](#code-reference-map)

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
┌──────────────────────────────────────────────────────────────────────────────┐
│                            StageFlow Platform                                 │
│                                                                              │
│  ┌──────────┐    ┌──────────────────────────────────────────────────────┐    │
│  │  Clients  │    │              Infrastructure Layer                     │    │
│  │          │    │                                                       │    │
│  │ ┌──────┐ │    │  ┌──────┐  ┌──────┐  ┌──────────┐  ┌──────────┐     │    │
│  │ │ Web  │ │    │  │ NATS │  │ MinIO│  │PostgreSQL│  │ Grafana  │     │    │
│  │ │Svelte│ │    │  │JetStr│  │ S3   │  │ 17       │  │ 12       │     │    │
│  │ │Kit   │ │    │  │eam   │  │      │  │          │  │          │     │    │
│  │ └──┬───┘ │    │  └──┬───┘  └──┬───┘  └──────────┘  └──────────┘     │    │
│  │    │     │    │     │          │                                     │    │
│  │ ┌──┴───┐ │    │     ▼          ▼                                     │    │
│  │ │ Go   │ │    │  ┌─────────────────────────────────────────────┐    │    │
│  │ │ CLI  │ │    │  │           Platform API (Go)                  │    │    │
│  │ └──┬───┘ │    │  │  • URL/ZIP intake + SSRF validation         │    │    │
│  │    │     │    │  │  • Job/report APIs + SSE hub                │    │    │
│  │    │     │    │  │  • Project CRUD + baseline promotion        │    │    │
│  │    │     │    │  │  • On-demand diff engine                    │    │    │
│  │    │     │    │  │  • SQLite project store                     │    │    │
│  │    │     │    │  └────────────────┬────────────────────────────┘    │    │
│  └────┼───────────┼─────────────────┼─────────────────────────────────┘    │
│       │           │                 │                                      │
│       │     POST  │           NATS events             SSE stream           │
│       │    /jobs  │        (job.created, etc.)       /jobs/{id}/stream     │
│       ▼           ▼                 ▼                                      │
│              ┌──────────────────────────────────────────────────────┐      │
│              │              NATS JetStream                           │      │
│              │  Streams: jobs | extraction | scan                    │      │
│              │  8 event types, durable consumers, explicit ACK       │      │
│              └──────────────────────┬───────────────────────────────┘      │
│                                     │                                      │
│              ┌──────────────────────┴───────────────────────────────┐      │
│              │              Orchestrator (Go)                        │      │
│              │  • Job FSM (PENDING→DONE/FAILED)                     │      │
│              │  • Podman pod lifecycle management                   │      │
│              │  • Scanner coordination + completion tracking        │      │
│              │  • Report aggregation + deduplication                │      │
│              │  • PostgreSQL job state + event audit trail          │      │
│              │  • Deadline sweeper for stuck jobs                   │      │
│              └──────────────────────┬───────────────────────────────┘      │
│                                     │                                      │
│              ┌──────────────────────┴───────────────────────────────┐      │
│              │              Podman Job Pod (per job)                  │      │
│              │                                                       │      │
│              │  ┌───────────────────┐  ┌─────────────────────────┐  │      │
│              │  │ Archive Extractor │  │    Scanner Runner       │  │      │
│              │  │ (Go, ZIP jobs)    │  │ (TS/Bun/Playwright)     │  │      │
│              │  │ • Safe extraction │  │ • Plugin discovery      │  │      │
│              │  │ • ZIP bomb defense│  │ • Browser automation    │  │      │
│              │  │ • Provenance gen  │  │ • Artifact upload       │  │      │
│              │  │ • Static server   │  │ • NATS publishing       │  │      │
│              │  └───────────────────┘  └─────────────────────────┘  │      │
│              └───────────────────────────────────────────────────────┘      │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Primary Repository Areas

| Directory                    | Responsibility                                                          |
| ---------------------------- | ----------------------------------------------------------------------- |
| `services/platform-api`      | Intake, validation, status APIs, SSE, project management, diff engine   |
| `services/orchestrator`      | FSM, container lifecycle, aggregation, PostgreSQL persistence           |
| `services/archive-extractor` | Secure archive extraction and provenance generation                     |
| `services/scanner-runner`    | Plugin discovery and scanner execution runtime                          |
| `clients/web`                | Submission UX and live status/report views (SvelteKit 5)                |
| `clients/cli`                | Go CLI — scan submission, SSE streaming, report rendering, project mode |
| `libs/contracts`             | JSON Schemas and generated contracts (Go + TypeScript)                  |
| `libs/go/*`                  | 12 shared Go packages (messaging, models, config, etc.)                 |

---

## Service Topology

### Container Services

| Service        | Image                                     | Language          | Ports (local)                      | Resources        |
| -------------- | ----------------------------------------- | ----------------- | ---------------------------------- | ---------------- |
| `platform-api` | `localhost/stageflow/platform-api:latest` | Go 1.26           | 8080                               | 512M RAM, 2 CPU  |
| `orchestrator` | `localhost/stageflow/orchestrator:latest` | Go 1.26           | 8081 (internal)                    | 512M RAM, 4 CPU  |
| `frontend`     | `localhost/stageflow/frontend:latest`     | SvelteKit + Caddy | 3010                               | 64M RAM, 0.5 CPU |
| `nats`         | `nats:2.12.2-alpine`                      | —                 | 4222 (internal), 8222 (monitoring) | 256M RAM, 1 CPU  |
| `minio`        | `minio:RELEASE.2025-09-07`                | —                 | 9000 (internal), 9001 (console)    | 512M RAM, 2 CPU  |
| `postgres`     | `postgres:17-alpine`                      | —                 | internal                           | 512M RAM, 2 CPU  |
| `grafana`      | `grafana:12.2.0`                          | —                 | 3001                               | 256M RAM, 1 CPU  |

### Ephemeral Job Containers

| Container         | Image                                       | Language       | Trigger                        |
| ----------------- | ------------------------------------------- | -------------- | ------------------------------ |
| Archive Extractor | `localhost/stageflow/extractor:latest`      | Go             | ZIP jobs only                  |
| Scanner Runner    | `localhost/stageflow/scanner-runner:latest` | TypeScript/Bun | One per enabled scanner module |

All containers run rootless with `security_opt: no-new-privileges:true` and resource limits.

---

## Core Services

### Platform API (`services/platform-api`)

**Role:** Public-facing HTTP API and entry boundary for the system.

**Architecture:**

```
services/platform-api/
├── cmd/server/main.go              # Entry point
├── cmd/server/config.go            # Configuration loading + validation
└── internal/
    ├── api/                        # HTTP handlers and router
    │   ├── router.go               # Route registration + middleware stack
    │   ├── handlers_jobs_url_submit.go   # URL intake
    │   ├── handlers_jobs_zip_upload.go   # ZIP intake
    │   ├── handlers_sse.go               # SSE stream hub
    │   ├── handlers_projects.go          # Project CRUD, scans, and baseline promotion
    │   ├── handlers_jobs_status.go       # Job status, artifact redirects, and project diffs
    │   ├── scanner_configs.go            # Per-scanner config validation
    │   └── security.go                   # SSRF protection
    ├── jobstatus/                  # Event-sourced pipeline
    │   ├── types.go                # Signal, Change, Subscription types
    │   └── pipeline.go             # StatusPipeline interface
    ├── messaging/                  # NATS publish/subscribe wrapper
    ├── status/                     # Job status model/store used by tests and projections
    │   └── schema.sql              # job_status table
    ├── statussource/               # HTTP client to read orchestrator status
    ├── project/                    # SQLite-backed project store
    │   ├── store.go                # CRUD operations
    │   └── schema.sql              # projects + project_jobs tables
    └── sqlite/                     # SQLite connection management
```

**Middleware Stack:**

```
Request → Logging → CORS → API Key Auth → Rate Limiting → Timeout → Handler
                                                          (except SSE)
```

**API Endpoints:**

| Method | Path                              | Purpose                                         | Auth    |
| ------ | --------------------------------- | ----------------------------------------------- | ------- |
| POST   | `/api/v1/jobs/urls`               | Submit URLs for scanning                        | API key |
| POST   | `/api/v1/jobs/zip`                | Submit ZIP archive for scanning                 | API key |
| GET    | `/api/v1/jobs/{id}`               | Get job status                                  | API key |
| GET    | `/api/v1/jobs/{id}/stream`        | SSE stream for real-time updates                | API key |
| GET    | `/api/v1/jobs/{id}/report`        | Redirect to HTML report artifact                | API key |
| GET    | `/api/v1/jobs/{id}/results`       | Redirect to normalized JSON report artifact     | API key |
| GET    | `/api/v1/jobs/{id}/diff`          | Diff project scan against its promoted baseline | API key |
| GET    | `/api/v1/projects`                | List projects                                   | API key |
| POST   | `/api/v1/projects`                | Create project                                  | API key |
| GET    | `/api/v1/projects/{slug}`         | Show project                                    | API key |
| PATCH  | `/api/v1/projects/{slug}`         | Update project                                  | API key |
| DELETE | `/api/v1/projects/{slug}`         | Delete project                                  | API key |
| POST   | `/api/v1/projects/{slug}/scan`    | Launch a scan from stored project config        | API key |
| POST   | `/api/v1/projects/{slug}/promote` | Promote job as baseline                         | API key |
| GET    | `/api/v1/scanners`                | List available scanners                         | API key |
| GET    | `/healthz`                        | Health check                                    | —       |

**Key Configuration (env vars):**

| Variable                       | Default         | Purpose                                                                       |
| ------------------------------ | --------------- | ----------------------------------------------------------------------------- |
| `PORT`                         | 8080            | HTTP listen port                                                              |
| `NATS_URL`                     | —               | NATS server URL                                                               |
| `MINIO_ENDPOINT`               | —               | MinIO internal endpoint                                                       |
| `MINIO_PUBLIC_ENDPOINT`        | —               | MinIO public endpoint (for presigned URLs)                                    |
| `MINIO_ACCESS_KEY`             | —               | MinIO credentials (falls back to `MINIO_ROOT_USER`)                           |
| `MINIO_SECRET_KEY`             | —               | MinIO credentials (falls back to `MINIO_ROOT_PASSWORD`)                       |
| `ORCHESTRATOR_API_URL`         | —               | Internal orchestrator API URL                                                 |
| `ORCHESTRATOR_API_TOKEN`       | —               | Inter-service auth token                                                      |
| `PLATFORM_API_TOKEN`           | —               | Public API token; required unless auth is explicitly disabled                 |
| `PLATFORM_API_AUTH_DISABLED`   | `false`         | Local-only opt-out for API auth                                               |
| `PLATFORM_API_TRUSTED_PROXIES` | —               | Trusted proxy CIDRs/IPs allowed to supply `X-Forwarded-For` for rate limiting |
| `SCANNER_CONFIG_PATH`          | —               | YAML scanner override file path                                               |
| `PROJECT_DB_PATH`              | `./projects.db` | SQLite project database path                                                  |

---

### Orchestrator (`services/orchestrator`)

**Role:** Central coordinator — manages the job state machine, launches containers, and aggregates results.

**Architecture (Clean Architecture):**

```
services/orchestrator/
├── cmd/orchestrator/
│   ├── main.go                     # Entry point
│   └── config.go                   # Configuration (10+ validated fields)
└── internal/
    ├── orchestrator/               # Core orchestrator
    │   └── events.go               # Event handler registration
    ├── application/jobs/           # Application service layer (ports/interfaces)
    │   └── scanner_launch_planner.go  # AI Navigator env injection
    ├── domain/jobs/                # Domain logic
    │   ├── state_transitions.go    # FSM transition rules
    │   ├── completion_policy.go    # When all scanners are done
    │   └── failure_policy.go       # Failure handling
    ├── adapters/
    │   ├── repository/             # PostgreSQL persistence
    │   │   ├── schema.sql          # jobs + job_events tables
    │   │   └── job_repository.go   # CRUD + event audit
    │   ├── runtime/                # Podman client
    │   │   ├── pod_manager.go      # Pod lifecycle
    │   │   ├── container_manager.go# Container management
    │   │   └── job_runtime.go      # Job-scoped runtime
    │   ├── messaging/              # NATS publisher
    │   └── storage/                # Report aggregation
    │       ├── aggregator.go       # Merge scanner outputs
    │       └── rule_deduplication.go  # Cross-scanner dedup
    └── api/                        # Internal admin HTTP API
        └── router.go               # Token-authenticated endpoints
```

**Internal API Endpoints (token-authenticated):**

| Method | Path                       | Purpose                                |
| ------ | -------------------------- | -------------------------------------- |
| GET    | `/api/v1/jobs`             | List jobs (state filter, pagination)   |
| GET    | `/api/v1/jobs/{id}`        | Get single job                         |
| GET    | `/api/v1/jobs/{id}/events` | Get job events                         |
| GET    | `/api/v1/pods`             | List all pods                          |
| GET    | `/api/v1/pods/{id}`        | Pod details                            |
| GET    | `/api/v1/status`           | System status (job counts, pod counts) |
| GET    | `/healthz`                 | Health check                           |

**Key Configuration (env vars):**

| Variable             | Default                                     | Purpose                                         |
| -------------------- | ------------------------------------------- | ----------------------------------------------- |
| `DATABASE_URL`       | —                                           | PostgreSQL connection string                    |
| `PODMAN_SOCKET`      | `/run/podman/podman.sock`                   | Podman API socket                               |
| `NATS_URL`           | —                                           | NATS server URL                                 |
| `MINIO_ENDPOINT`     | —                                           | MinIO internal endpoint                         |
| `EXTRACTION_IMAGE`   | `localhost/stageflow/extractor:latest`      | Extractor container image                       |
| `SCANNER_IMAGE`      | `localhost/stageflow/scanner-runner:latest` | Scanner container image                         |
| `API_PORT`           | 8081                                        | Internal API listen port                        |
| `POD_NETWORK`        | —                                           | Podman network name                             |
| `POD_NETNS_MODE`     | `bridge`                                    | `bridge` or `host`                              |
| `OPENROUTER_API_KEY` | —                                           | AI Navigator (injected into scanner containers) |

---

### Archive Extractor (`services/archive-extractor`)

**Role:** Extracts uploaded ZIP files, discovers HTML pages, generates provenance, serves the site locally.

```
services/archive-extractor/
├── cmd/server/main.go              # Entry point
└── internal/
    ├── extractor/
    │   └── extractor.go            # ZIP extraction with safety checks
    ├── discovery/
    │   └── discovery.go            # HTML page discovery
    ├── provenance/
    │   └── provenance.go           # provenance.json generation
    └── server/
        └── server.go               # Embedded static HTTP server
```

**Safety Controls:**

| Control                    | Purpose                             |
| -------------------------- | ----------------------------------- |
| Entry-count limits         | Prevent archive-of-archives attacks |
| Per-entry size constraints | Prevent single-file bombs           |
| Total size constraints     | Prevent overall disk exhaustion     |
| Compression-ratio checks   | ZIP bomb defense                    |
| Path traversal prevention  | Prevent writing outside workspace   |

**Lifecycle:**

```
1. Download ZIP from MinIO staging bucket
2. Validate archive safety constraints
3. Extract to /workspace with path traversal protection
4. Discover all HTML pages in extracted site
5. Generate provenance.json (page IDs, paths, URLs)
6. Upload provenance.json to MinIO artifacts
7. Start embedded static HTTP server on :8080
8. Publish extraction.ready or extraction.failed to NATS
```

---

### Scanner Runner (`services/scanner-runner`)

**Role:** Executes individual scanners against a target site using Playwright for browser automation.

```
services/scanner-runner/
├── src/
│   ├── worker.ts                   # Worker mode entry point
│   └── core/
│       ├── scanner-base.ts         # Abstract base class (lifecycle)
│       ├── types.ts                # Core type definitions
│       ├── config-loader.ts        # Environment variable parsing
│       ├── plugins/
│       │   └── plugin-loader.ts    # Plugin discovery and loading
│       ├── event-publisher.ts      # NATS JetStream event publishing
│       ├── storage-provider/       # MinIO storage abstraction
│       ├── browser-manager.ts      # Playwright browser lifecycle
│       └── page-iterator.ts        # Concurrent page iteration
│   ├── scanners/                   # Built-in scanner implementations
│   │   ├── axe/                    # WCAG accessibility
│   │   ├── lighthouse/             # Performance, a11y, SEO, best practices
│   │   ├── seo/                    # SEO analysis
│   │   ├── security-headers/       # HTTP security headers
│   │   ├── link-checker/           # Broken link detection
│   │   ├── open-graph/             # Open Graph validation
│   │   ├── spelling-grammar/       # Rule-based content quality
│   │   └── ai-navigator/           # LLM-powered browser agent
│   └── screenshots/
│       └── axe/                    # Advanced screenshot capture for violations
└── tests/                          # Vitest test suite
```

**Scanner Lifecycle (ScannerBase):**

```
initialize()
    ↓
iteratePages() ── concurrent page processing ──► writeResults()
    ↓                                                    ↓
  per-page scans                              uploadArtifacts()
    ↓                                                    ↓
  scan.page.completed events                   scan.completed event
```

---

## Data Models

### Core Domain Types

```
┌─────────────────────────────────────────────────────────────────┐
│                        Job (libs/go/models)                      │
├─────────────────────────────────────────────────────────────────┤
│  ID: UUID                                                       │
│  State: JobState (PENDING→EXTRACTING→READY_TO_SCAN→...)         │
│  InputType: "zip" | "urls"                                      │
│  InputPath: string (MinIO key for ZIP jobs)                     │
│  URLs: []string                                                 │
│  PodID: string                                                  │
│  Config: JobConfig { Modules, ScannerConfigs, Screenshot, ... } │
│  Timestamps: CreatedAt, UpdatedAt, CompletedAt                  │
│  Error, ErrorDetails                                            │
│  TotalPages, CurrentPage, TotalViolations                       │
│  ArtifactKeys: ReportJSONKey, ReportKey, ProvenancePath, ...    │
│  ScannerTracking: ExpectedScanners, CompletedScanners           │
│  ScannerResults: map[string]*ScannerResult                      │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                     JobState (enum)                              │
├─────────────────────────────────────────────────────────────────┤
│  PENDING → EXTRACTING → READY_TO_SCAN → SCANNING → COMPLETING   │
│                                                              → DONE
│  Any state → FAILED (terminal)                                  │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                   ScannerResult (per-scanner)                    │
├─────────────────────────────────────────────────────────────────┤
│  ScannerType: string                                            │
│  ResultsPath, ReportPath: string (MinIO keys)                   │
│  Success: bool                                                  │
│  Error: string                                                  │
│  PagesScanned: int                                              │
│  SeverityCounts: map[string]int (critical, serious, ...)        │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                    ArtifactLocation (API response)               │
├─────────────────────────────────────────────────────────────────┤
│  ReportJSON, ReportHTML: string (presigned URLs)                │
│  ScanStageLog, ScanRecipe: string                               │
│  ProvenanceJSON: string                                         │
│  Screenshots: []ScreenshotArtifact                              │
│  ScannerArtifacts: map[string]*ScannerArtifacts                 │
└─────────────────────────────────────────────────────────────────┘
```

### Event Payload Types (`libs/go/events/`)

| Type                       | Key Fields                                                   |
| -------------------------- | ------------------------------------------------------------ |
| `JobCreatedPayload`        | JobID, InputType, InputPath, URLs, Config                    |
| `ExtractionReadyPayload`   | JobID, ProvenancePath, BaseURL, TotalPages                   |
| `ExtractionFailedPayload`  | JobID, Error, ErrorDetails                                   |
| `ScanPageCompletedPayload` | JobID, ScannerType, PageID, PageIndex, TotalPages            |
| `ScanCompletedPayload`     | JobID, ScannerType, ResultsPath, ReportPath, Summary, Timing |
| `ScanFailedPayload`        | JobID, ScannerType, Error, ErrorDetails                      |
| `JobCompletedPayload`      | JobID, Artifacts, ScannerArtifacts                           |
| `JobFailedPayload`         | JobID, Stage (extraction/scanning/reporting), Error          |

### Scanner Registry Types (`libs/go/scannerregistry/`)

```
Definition (internal)
├── ID, Name, Version, Description
├── Categories: []string
├── Aliases: []string
├── Image: string
├── Enabled: bool
├── BuiltIn: bool
├── Config: map[string]any
├── Capabilities: { OutputFormats, SupportsScreenshots, SupportsConcurrency, ... }
└── Requirements: { Browser, NodeVersion, MaxMemoryMB, MaxTimeoutMs }

Info (public API projection — strips Image, Aliases, Config, Requirements)
```

---

## Event Model

### NATS JetStream Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        NATS JetStream                                │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │  Stream: JOBS (max age: 72h)                                │    │
│  │  Subjects:                                                  │    │
│  │    jobs.events.created    ← platform-api → orchestrator     │    │
│  │    jobs.events.completed  ← orchestrator → platform-api     │    │
│  │    jobs.events.failed     ← orchestrator → platform-api     │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │  Stream: EXTRACTION (max age: 72h)                          │    │
│  │  Subjects:                                                  │    │
│  │    extraction.events.ready   ← extractor → orchestrator     │    │
│  │    extraction.events.failed  ← extractor → orchestrator     │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │  Stream: SCAN (max age: 72h)                                │    │
│  │  Subjects:                                                  │    │
│  │    scan.events.page.completed  ← scanner-runner → all       │    │
│  │    scan.events.completed       ← scanner-runner → all       │    │
│  │    scan.events.failed          ← scanner-runner → all       │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  Consumer Properties:                                                │
│    • Durable consumers with explicit ACK                            │
│    • Max 10 deliveries                                              │
│    • 10-minute ACK wait                                             │
│    • 5-second NAK delay on failure                                  │
│    • Strict payload validation (DisallowUnknownFields)              │
│    • Lenient envelope parsing (forward-compatible)                  │
└─────────────────────────────────────────────────────────────────────┘
```

### Event Envelope

```json
{
	"event": "scan.completed",
	"job_id": "550e8400-e29b-41d4-a716-446655440000",
	"request_id": "optional-correlation-id",
	"run_id": "optional-run-id",
	"timestamp": "2026-04-04T00:00:00Z",
	"producer": "scanner-runner",
	"payload": {
		"job_id": "550e8400-e29b-41d4-a716-446655440000",
		"scanner_type": "axe",
		"results_path": "550e8400.../axe/results.json",
		"report_path": "550e8400.../axe/report.html",
		"total_pages_scanned": 5,
		"summary": {
			"total_violations": 12,
			"by_severity": { "critical": 2, "serious": 5, "moderate": 3, "minor": 2 },
			"pages_scanned": 5
		},
		"timing": {
			"total_ms": 45000,
			"page_iteration_ms": 30000,
			"write_results_ms": 5000,
			"upload_artifacts_ms": 8000,
			"publish_completed_ms": 1000,
			"finalization_ms": 1000
		}
	}
}
```

### Messaging Implementation (`libs/go/messaging/`)

The Go messaging library provides:

- **`Client`** — Wraps `nats.Conn` + `jetstream.JetStream`
- **`SubscribeTyped[T any]`** — Generic typed subscription that:
  1. Creates/updates a durable consumer
  2. Parses the envelope (lenient, for forward compatibility)
  3. Parses the payload with `DisallowUnknownFields()` (strict)
  4. Attaches `ReceivedEventMeta` to context (event, jobID, stream, consumer, seq numbers, delivery count)
  5. Attaches jobID/requestID/runID to logging context
  6. Calls the typed handler
- **`PublishEvent`** — Requires payloads to implement `Validate() error` before sending
- **Auto-retry** — NAK with 5-second delay on handler failure

---

## Data Flows

### URL Job Flow (Complete)

```
┌──────────┐
│  Client  │
│ (Web/CLI)│
└────┬─────┘
     │ POST /api/v1/jobs/urls
     │ { urls: [...], modules: ["axe","seo"], screenshot: true }
     ▼
┌──────────────────────────────────────────────────────────────┐
│                    Platform API                               │
│                                                              │
│  1. Validate request body (size, shape, URL count ≤ 100)    │
│  2. Validate each URL (scheme, format, SSRF check)          │
│  3. Normalize modules against scanner registry              │
│  4. Validate scanner configs against manifests              │
│  5. Generate jobID (UUID v4), runID                         │
│  6. Publish events.Envelope{event: "job.created"} to NATS   │
│  7. Seed job status projection (PENDING) in SQLite          │
│  8. Return { job_id, status: "PENDING" }                    │
└──────────────────────┬───────────────────────────────────────┘
                       │ job.created (NATS JetStream)
                       ▼
┌──────────────────────────────────────────────────────────────┐
│                    Orchestrator                               │
│                                                              │
│  1. Consume job.created event                               │
│  2. Validate payload (JobCreatedPayload.Validate())         │
│  3. Transition PENDING → READY_TO_SCAN (URL jobs skip ext.) │
│  4. Create Podman pod for this job                          │
│  5. Launch scanner-runner containers (one per module)       │
│  6. Track expected scanners in PostgreSQL                   │
└──────────────────────┬───────────────────────────────────────┘
                       │
        ┌──────────────┼──────────────┐
        ▼              ▼              ▼
┌───────────┐  ┌───────────┐  ┌───────────┐
│ Scanner:  │  │ Scanner:  │  │ Scanner:  │
│   axe     │  │    seo    │  │ lighthouse│
└─────┬─────┘  └─────┬─────┘  └─────┬─────┘
      │              │              │
      │ scan.page.completed (per page, per scanner)
      │ scan.completed (when all pages done)
      │ scan.failed (on error)
      ▼              ▼              ▼
┌──────────────────────────────────────────────────────────────┐
│                    Orchestrator                               │
│                                                              │
│  1. Consume scan.page.completed → update progress in DB     │
│  2. Consume scan.completed → mark scanner done              │
│  3. When all expected scanners complete:                    │
│     a. Transition SCANNING → COMPLETING                     │
│     b. Download all scanner results from MinIO              │
│     c. Merge pages and issues                               │
│     d. Normalize severities/categories                      │
│     e. Deduplicate equivalent findings                      │
│     f. Recalculate aggregate summaries and scores           │
│     g. Upload unified report to MinIO                       │
│     h. Transition COMPLETING → DONE                         │
│     i. Publish job.completed with artifact locations        │
└──────────────────────┬───────────────────────────────────────┘
                       │ job.completed (NATS JetStream)
                       ▼
┌──────────────────────────────────────────────────────────────┐
│                    Platform API                               │
│                                                              │
│  1. Consume job.completed event                             │
│  2. Update job status projection (DONE) in SQLite           │
│  3. Store artifact locations (presigned URLs)               │
│  4. Push SSE update to connected clients                    │
└──────────────────────────────────────────────────────────────┘
```

### ZIP Job Flow (Differences from URL)

```
Client ──POST /api/v1/jobs/zip──► Platform API
                                     │
                                     ├─ Validate multipart form
                                     ├─ Store ZIP in MinIO staging bucket
                                     └─ Publish job.created
                                             │
                                             ▼
                                     Orchestrator
                                     ├─ Create Podman job pod
                                     ├─ Start archive-extractor container
                                     │     ├─ Download ZIP from MinIO
                                     │     ├─ Validate: entry count, size, compression ratio
                                     │     ├─ Extract with path traversal protection
                                     │     ├─ Discover HTML pages
                                     │     ├─ Generate provenance.json
                                     │     ├─ Upload provenance to MinIO
                                     │     └─ Start static HTTP server
                                     ├─ Consume extraction.ready
                                     ├─ Set TotalPages from provenance
                                     └─ Continue as URL flow (READY_TO_SCAN → SCANNING → ...)
```

### CLI Scan Flow

```
stageflow scan https://example.com --scanners axe,seo --format json --fail-on serious
     │
     ├─ 1. Normalize URLs, validate private targets
     │     (auto-enable --allow-private-targets if localhost detected)
     │
     ├─ 2. POST /api/v1/jobs/urls { urls, modules }
     │     → Receive { job_id, status }
     │
     ├─ 3. Open SSE stream: GET /api/v1/jobs/{id}/stream
     │     → Parse event:/data: lines
     │     → Print progress to stderr (scanner completions, page counts)
     │     → Track seen scanners to avoid duplicate output
     │     → Handle reconnect (buffering for missed events)
     │
     ├─ 4. On terminal state (DONE/FAILED):
     │     → GET /api/v1/jobs/{id}/results
     │     → Download report.json (UnifiedReportV2)
     │     → Wrap in CLI envelope (stageflow-cli/report@v1)
     │     → Filter issues (severity, category, max_issues)
     │     → Sort by severity desc, scanner asc, rule asc
     │     → Render to stdout (text/markdown/json)
     │
     └─ 5. Check --fail-on threshold
           → Exit 0 if no issues at/above threshold
           → Exit 1 if issues meet/exceed threshold
```

### CLI Project Mode Flow

```
stageflow project [path]
     │
     ├─ 1. Resolve project root (git repo root)
     ├─ 2. Load or bootstrap .stageflow/config.yaml
     │     └─ Auto-detect dev commands from Justfile + package.json
     │
     ├─ 3. Start dev server lifecycle:
     │     a. Run dev.up command steps (e.g., docker compose up)
     │     b. Start dev.start.cmd as subprocess (process group on Unix)
     │     c. Poll dev.ready.url until HTTP 2xx/3xx or timeout
     │
     ├─ 4. Submit scan job (same as scan flow, within remaining timeout)
     ├─ 5. Stream progress and render report
     │
     └─ 6. Stop dev server:
           a. Send SIGINT (configurable signal)
           b. Wait up to timeout, then SIGKILL
           c. Run dev.down command steps
```

### Remote Project Scan Flow

```
stageflow scan --project my-app --format json
     │
     ├─ 1. GET /api/v1/projects/my-app
     │     → Resolve: URLs, scanners, baseline_job_id
     │
     ├─ 2. POST /api/v1/jobs/urls { urls, modules, project_slug }
     │
     ├─ 3. Open SSE stream, wait for completion
     │
     ├─ 4. On DONE:
     │     a. GET /api/v1/jobs/{id}/results → render report
     │     b. If project has baseline_job_id:
     │          → GET /api/v1/projects/my-app/diff?job_id={id}
     │          → Render diff envelope (separated by blank line)
     │          → Exit 1 if newIssues > 0
     │
     └─ 5. Exit 0 if no regressions
```

---

## Job State Machine

### Transition Matrix

```
┌──────────────┬────────────────────────────────────────────────────────────┐
│ From State   │ Allowed Transitions                                      │
├──────────────┼────────────────────────────────────────────────────────────┤
│ PENDING      │ EXTRACTING, READY_TO_SCAN, FAILED                         │
│ EXTRACTING   │ READY_TO_SCAN, FAILED                                     │
│ READY_TO_SCAN│ SCANNING, FAILED                                          │
│ SCANNING     │ COMPLETING, FAILED                                        │
│ COMPLETING   │ DONE, FAILED                                              │
│ DONE         │ (terminal)                                                │
│ FAILED       │ (terminal)                                                │
└──────────────┴────────────────────────────────────────────────────────────┘
```

### State Intent

| State           | Meaning                               | Triggered By                                  |
| --------------- | ------------------------------------- | --------------------------------------------- |
| `PENDING`       | Accepted and queued                   | API job submission                            |
| `EXTRACTING`    | ZIP processing in extractor container | Orchestrator starts extractor (ZIP jobs only) |
| `READY_TO_SCAN` | Scanner inputs are ready              | `extraction.ready` event or URL job skip      |
| `SCANNING`      | One or more scanner containers active | Orchestrator launches scanners                |
| `COMPLETING`    | Output merge/dedup/report publish     | All scanners complete                         |
| `DONE`          | Successful terminal state             | Report aggregated and uploaded                |
| `FAILED`        | Terminal failure state                | Any unrecoverable error                       |

### Implementation

The state machine is defined in `libs/go/domain/job/state.go` as the single source of truth:

```go
var allowedTransitions = map[JobState][]JobState{
    PENDING:      {EXTRACTING, READY_TO_SCAN, FAILED},
    EXTRACTING:   {READY_TO_SCAN, FAILED},
    READY_TO_SCAN: {SCANNING, FAILED},
    SCANNING:     {COMPLETING, FAILED},
    COMPLETING:   {DONE, FAILED},
    DONE:         {},
    FAILED:       {},
}
```

Database queries use `StateRankSQL()` CASE expression to prevent state regressions at the SQL level.

---

## Contract Architecture

StageFlow uses a **schema-first, contract-driven** approach. JSON Schema files in `libs/contracts/` are the single source of truth, with generated code in both Go and TypeScript.

### Contract Families

```
libs/contracts/
├── report/
│   ├── schema/
│   │   └── unified-report.v2.schema.json    # 356 lines — the canonical output format
│   └── generated/
│       ├── go/
│       │   └── report_schema.go             # 1208 lines — atombender/go-jsonschema
│       │                                     # Custom UnmarshalJSON enforces all constraints
│       └── typescript/
│           ├── unified-report.v2.ts         # 276 lines — json-schema-to-typescript
│           └── validator.ts                 # 345 lines — Ajv + data integrity checks
│
├── scanner-manifest/
│   ├── schema/
│   │   └── scanner-manifest.schema.json     # 294 lines — plugin descriptor format
│   └── generated/
│       ├── go/
│       │   └── scanner_manifest.go          # 534 lines — atombender/go-jsonschema
│       └── typescript/
│           └── scanner-manifest.ts          # 163 lines — json-schema-to-typescript
│   └── validator.go                         # 125 lines — santhosh-tekuri/jsonschema
│
└── events/
    ├── schema/
    │   ├── scan.completed.schema.json
    │   ├── scan.failed.schema.json
    │   └── scan.page.completed.schema.json
    └── fixtures/                            # JSON examples for each event type
```

### Code Generation Pipeline

```
JSON Schema (source of truth)
    │
    ├──► atombender/go-jsonschema ──► Go types with custom UnmarshalJSON
    │                                  • Required field enforcement
    │                                  • minLength, min/max validation
    │                                  • Regex pattern matching
    │                                  • Enum validation
    │
    └──► json-schema-to-typescript ──► TypeScript types
                                       • Compile-time type safety
```

### Cross-Language Consistency

The following enums appear identically across all contract families:

| Enum                 | Values                                                     | Appears In                                                       |
| -------------------- | ---------------------------------------------------------- | ---------------------------------------------------------------- |
| `IssueSeverity`      | critical, serious, moderate, minor, info                   | Report contract, scanner-manifest, Go events, Go types, TS types |
| `ScannerStatus`      | success, failed, skipped                                   | Report contract, Go types, TS types                              |
| `ScannerCategory`    | accessibility, performance, security, seo, quality, custom | Scanner-manifest contract                                        |
| `OutputFormat`       | json, html, csv                                            | Scanner-manifest contract                                        |
| `ErrorScope`         | scanner, page, global                                      | Report contract                                                  |
| `UserGroup`          | blind, low-vision, motor, cognitive, deaf, vestibular, all | Report contract                                                  |
| `UserImpactSeverity` | blocking, degraded, inconvenient                           | Report contract                                                  |

### Strict vs. Lenient Decoding

| Operation                  | Strictness                                 | Rationale                            |
| -------------------------- | ------------------------------------------ | ------------------------------------ |
| **Publishing**             | Strict — `Validate()` required on envelope | Reject invalid events before sending |
| **Subscribing (envelope)** | Lenient — unknown fields allowed           | Forward-compatible event evolution   |
| **Subscribing (payload)**  | Strict — `DisallowUnknownFields()`         | Catch schema drift in payloads       |

### Data Integrity Beyond Schema

The TypeScript report validator (`libs/contracts/report/generated/typescript/validator.ts`) checks business logic invariants that JSON Schema cannot express:

- `summary.totalIssues` matches `issues.length`
- Severity counts match actual issue severities
- `pagesScanned` matches `pages.length`
- Scanner IDs in issues exist in scanners array
- Page IDs in issues exist in pages array
- Artifact IDs in occurrences exist in artifacts array

Go-side validation:

- `ScanTiming.Validate()` — component sum must not exceed total
- `ScanSummary.Validate()` — `bySeverity` map must be non-nil with non-negative values
- `Provenance.Validate()` — pages non-empty, paths must start with `/`

---

## Unified Report

### Report Schema (`UnifiedReportV2`)

```
UnifiedReportV2
├── version: string (semver ^2\.\d+\.\d+$)
├── meta: ReportMeta
│   ├── jobId: string (required)
│   ├── baseUrl: string
│   ├── scannedAt: string (ISO8601)
│   ├── completedAt: string (ISO8601)
│   └── durationMs: number
├── summary: ReportSummary
│   ├── totalIssues: number (required)
│   ├── bySeverity: SeverityCounts { critical, serious, moderate, minor, info? }
│   ├── pagesScanned: number
│   ├── pagesWithIssues: number
│   ├── score?: number (0-100)
│   ├── scoreGrade?: string (pattern ^[A-F][+-]?$)
│   ├── byScanner?: map[string]number
│   └── lighthouseCategories?: LighthouseCategorySummary[]
├── scanners: ScannerSummary[]
│   ├── id: string (required)
│   ├── status: enum (success|failed|skipped)
│   ├── name?: string
│   ├── error?: string
│   ├── issueCount: number
│   ├── severity: string
│   ├── toolVersion?: string
│   ├── resultsPath?: string
│   └── reportPath?: string
├── pages: PageSummary[]
│   ├── id: string (required)
│   ├── url: string (required)
│   ├── issueCount: number (required)
│   ├── durationMs: number (required)
│   ├── path?: string
│   ├── bySeverity?: SeverityCounts
│   └── pageOverview?: PageOverview
│       ├── screenshotFilename: string
│       ├── pageWidth, pageHeight: number
│       └── elements: PageOverviewElement[]
│           ├── issueId, ruleId, severity: string
│           ├── selector: string
│           ├── nodeIndex: number
│           ├── x, y, width, height: number (absolute)
│           └── xPercent, yPercent, widthPercent, heightPercent: number (percentage)
├── issues: IssueDetail[]
│   ├── id: string (required) — stable content hash
│   ├── scanner: string (required)
│   ├── ruleId: string (required)
│   ├── severity: enum (required)
│   ├── title: string (required)
│   ├── description: string (required)
│   ├── pageId: string (required)
│   ├── pageUrl: string (required)
│   ├── elementCount: number (required)
│   ├── severityRaw?: string
│   ├── helpUrl?: string
│   ├── wcagTags?: string[]
│   ├── occurrences?: IssueOccurrence[]
│   │   ├── target: string[]
│   │   ├── selector: string
│   │   ├── html: string
│   │   ├── contextHtml?: string
│   │   ├── ancestorPath?: string[]
│   │   ├── failureSummary?: string
│   │   ├── textSnippet?: string
│   │   ├── boundingBox?: BoundingBox { x, y, width, height }
│   │   └── artifactIds?: string[]
│   ├── category?: string
│   ├── friendlyNode?: FriendlyNodeInfo
│   ├── locationInfo?: LocationInfo
│   ├── userImpact?: UserImpact
│   │   ├── statement: string
│   │   ├── affectedGroups: UserGroup[]
│   │   ├── severity: UserImpactSeverity
│   │   └── userStory: string
│   ├── howToFix?: string
│   └── scannerData?: object (open schema)
├── artifacts?: ReportArtifact[]
│   ├── id: string (required)
│   ├── type: string (required)
│   ├── path?: string
│   ├── mime?: string
│   └── dataUri?: string
└── errors?: ReportError[]
    ├── scope: enum (scanner|page|global)
    ├── code: string
    ├── message: string
    └── retryable: bool (required)
```

### Issue Identity

Each issue has a **stable content-based hash** (`id` field) derived from its rule, page, and element context. The same violation on the same page produces the same `id` across runs, enabling:

- **Baseline diffing** — compare reports by matching issue IDs
- **Regression detection** — new IDs = regressions, missing IDs = fixes
- **Deduplication** — equivalent findings across scanners are merged

### Rule Deduplication

The orchestrator's `rule_deduplication.go` handles cross-scanner deduplication during report aggregation, merging equivalent findings from different scanners into single issue entries.

---

## Scanner Plugin System

### Discovery Order

```
1. Built-ins (libs/go/scannercatalog/manifests/*/)
   └── Embedded at compile time via //go:embed
   └── Validated against scanner-manifest schema at load time
   └── 8 scanners: axe, lighthouse, seo, security-headers,
       link-checker, open-graph, spelling-grammar, ai-navigator

2. Mounted /plugins volume
   └── Podman/compose volume mount at runtime

3. ~/.stageflow/plugins/
   └── User-local plugin directory

4. PLUGIN_PATHS env var
   └── Colon-separated additional paths
```

### Plugin Manifest Contract

Each plugin supplies a `manifest.json` conforming to the scanner-manifest schema:

```json
{
  "id": "axe",
  "name": "Axe Accessibility",
  "version": "1.0.0",
  "capabilities": {
    "categories": ["accessibility"],
    "outputFormats": ["json"],
    "supportsScreenshots": true,
    "supportsConcurrency": true,
    "requiresBrowser": true,
    "maxConcurrency": 5,
    "estimatedTimePerPage": 3000
  },
  "configSchema": { ... },
  "requirements": {
    "browser": { "type": "chromium", "headless": true },
    "nodeVersion": ">=18.0.0"
  },
  "entry": {
    "module": "./scanners/axe/index.js",
    "exportName": "AxeScanner"
  },
  "output": {
    "severityMapping": { "critical": "critical", ... },
    "categoryPrefix": "a11y"
  }
}
```

### Scanner Registry (`libs/go/scannerregistry/`)

The Go scanner registry provides runtime scanner resolution:

| Operation              | Description                                           |
| ---------------------- | ----------------------------------------------------- |
| `Register`             | Add a scanner definition                              |
| `Unregister`           | Remove a scanner                                      |
| `Get`                  | Get scanner by ID                                     |
| `Resolve`              | Resolve by ID or alias (lenient)                      |
| `List`                 | List all scanners                                     |
| `ListEnabled`          | List enabled scanners only                            |
| `ListByCategory`       | Filter by category                                    |
| `ResolveModules`       | Resolve module tokens (lenient, pass through unknown) |
| `ResolveModulesStrict` | Resolve module tokens (error on unknown)              |

Default module: `axe`.

---

## AI Navigator

The AI Navigator (`ai-navigator`) is a vision-model-powered browser automation agent that uses LLMs to understand and navigate web pages based on user-defined goals.

### Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                     AI Navigator Agent Loop                          │
│                                                                      │
│  Browser loads target URL                                           │
│       │                                                             │
│       ▼                                                             │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  Loop (until goal met or limits exceeded):                  │   │
│  │                                                              │   │
│  │  1. Screenshot current page                                 │   │
│  │  2. Extract interactive elements (links, buttons, inputs)   │   │
│  │  3. Page Analyzer:                                          │   │
│  │     ├─ Send screenshot to vision model                     │   │
│  │     └─ Returns: page type, description, suggested actions  │   │
│  │  4. Action Decider:                                         │   │
│  │     ├─ Send screenshot + goal + history to vision model    │   │
│  │     └─ Returns: next action with reasoning and confidence  │   │
│  │  5. Execute browser action (click, fill, scroll, etc.)     │   │
│  │  6. Goal Checker: evaluate success criteria                │   │
│  │  7. Loop Detector: check for navigation loops (3+ same URL│   │
│  │     in last 6 steps)                                       │   │
│  │  8. Record step in trace                                   │   │
│  └─────────────────────────────────────────────────────────────┘   │
│       │                                                             │
│       ▼                                                             │
│  AgentResult { success, stepTrace[], screenshots[] }               │
└─────────────────────────────────────────────────────────────────────┘
```

### Core Modules

| Module         | File                        | Responsibility                                                                        |
| -------------- | --------------------------- | ------------------------------------------------------------------------------------- |
| Vision Client  | `vision-client.ts`          | OpenRouter API, image compression (Sharp), semaphore concurrency, exponential backoff |
| Page Analyzer  | `page-analyzer.ts`          | Extract interactive elements, classify page, suggest actions                          |
| Action Decider | `action-decider.ts`         | Determine next browser action from goal + history + screenshot                        |
| Goal Checker   | `goal-checker.ts`           | Evaluate: `url-contains`, `url-matches`, `element-visible`, `text-visible`, `custom`  |
| Action Parser  | `action-decision-parser.ts` | Parse vision model JSON into executable actions                                       |
| Loop Detector  | `loop-detector.ts`          | Detect navigation loops (same URL 3+ times in last 6 steps)                           |
| Agent          | `agent.ts`                  | Main execution loop, respects max steps/wall time/token budgets                       |
| Options        | `options.ts`                | Validate and parse agent configuration                                                |

### Backend Integration

- **API validation** (`scanner_configs.go`): Validates `goal.objective` is set and `vision.model` is specified (or falls back to `AI_NAVIGATOR_DEFAULT_MODEL`)
- **Orchestrator** (`scanner_launch_planner.go`): Injects `OPENROUTER_API_KEY`, `OPENROUTER_APP_TITLE`, `OPENROUTER_APP_REFERER` into scanner container env vars
- **No direct AI SDK** — raw HTTP calls to OpenRouter (`https://openrouter.ai/api/v1/chat/completions`)

### Environment Variables

| Variable                          | Required                   | Default              | Purpose                       |
| --------------------------------- | -------------------------- | -------------------- | ----------------------------- |
| `OPENROUTER_API_KEY`              | Yes (if AI Navigator used) | —                    | OpenRouter API authentication |
| `OPENROUTER_APP_TITLE`            | No                         | StageFlow            | Request attribution metadata  |
| `OPENROUTER_APP_REFERER`          | No                         | —                    | Request referer tracking      |
| `AI_NAVIGATOR_DEFAULT_MODEL`      | No                         | `openai/gpt-4o-mini` | Backend fallback model        |
| `VITE_AI_NAVIGATOR_DEFAULT_MODEL` | No                         | `openai/gpt-4o-mini` | Web App default model         |

---

## Security Model

### Four-Layer Security

```
┌──────────────────────────────────────────────────────────────────────┐
│                         Trust Boundaries                              │
│                                                                       │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │  Layer 4: Edge Proxy                                          │  │
│  │  ┌──────────────────────────────────────────────────────────┐ │  │
│  │  │  Caddy reverse proxy (infra/caddy/Caddyfile)             │ │  │
│  │  │  • TLS termination                                      │ │  │
│  │  │  • Rate limiting                                        │ │  │
│  │  │  • WAF policy (deployment concern)                      │ │  │
│  │  │  • Route: /api/* → platform-api:8100                    │ │  │
│  │  │  • Route: /scanner-artifacts/* → MinIO:9100             │ │  │
│  │  │  • Route: /monitoring* → Grafana:3101                   │ │  │
│  │  │  • Route: /* → frontend:3100                            │ │  │
│  │  └──────────────────────────────────────────────────────────┘ │  │
│  └────────────────────────────────────────────────────────────────┘  │
│                                ▲                                      │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │  Layer 3: Scanner Runtime                                     │  │
│  │  ┌──────────────────────────────────────────────────────────┐ │  │
│  │  │  Per-job Podman pod isolation                            │ │  │
│  │  │  • Rootless containers (no daemon, no privilege escalation│ │  │
│  │  │  • security_opt: no-new-privileges:true                  │ │  │
│  │  │  • Resource limits (memory, CPU)                         │ │  │
│  │  │  • Scanner identity validation against manifest          │ │  │
│  │  │  • SCANNER_OPTIONS schema validation                     │ │  │
│  │  │  • Artifact upload through storage interfaces only       │ │  │
│  │  └──────────────────────────────────────────────────────────┘ │  │
│  └────────────────────────────────────────────────────────────────┘  │
│                                ▲                                      │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │  Layer 2: Archive Extraction                                  │  │
│  │  ┌──────────────────────────────────────────────────────────┐ │  │
│  │  │  ZIP safety controls                                    │ │  │
│  │  │  • Entry-count limits                                   │ │  │
│  │  │  • Per-entry and total size constraints                 │ │  │
│  │  │  • Compression-ratio checks (ZIP bomb defense)          │ │  │
│  │  │  • Path traversal prevention                            │ │  │
│  │  │  • Workspace isolation                                  │ │  │
│  │  └──────────────────────────────────────────────────────────┘ │  │
│  └────────────────────────────────────────────────────────────────┘  │
│                                ▲                                      │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │  Layer 1: API Intake                                          │  │
│  │  ┌──────────────────────────────────────────────────────────┐ │  │
│  │  │  URL/ZIP validation                                     │ │  │
│  │  │  • URL scheme validation (http/https only)              │ │  │
│  │  │  • SSRF: IP classification, DNS resolution              │ │  │
│  │  │  • Blocked ranges: 0.0.0.0/8, 100.64.0.0/10, etc.      │ │  │
│  │  │  • Metadata service: 169.254.169.254 always blocked     │ │  │
│  │  │  • Request size limits: 2MB URL, 100MB ZIP              │ │  │
│  │  │  • URL count ≤ 100, length ≤ 2048 chars                 │ │  │
│  │  │  • API key middleware                                   │ │  │
│  │  │  • Rate limiting                                        │ │  │
│  │  │  • Request timeout (except SSE)                         │ │  │
│  │  └──────────────────────────────────────────────────────────┘ │  │
│  └────────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────────┘
```

### SSRF Protection Details

The Platform API (`security.go`) implements comprehensive SSRF protection:

**Three-tier IP classification:**

| Decision                  | IP Ranges                                                                                                                                                                                 | Context                             |
| ------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------- |
| **Block** (always)        | `0.0.0.0/8`, `100.64.0.0/10`, `169.254.0.0/16`, `192.0.0.0/24`, `192.0.2.0/24`, `198.18.0.0/15`, `198.51.100.0/24`, `203.0.113.0/24`, `224.0.0.0/4`, `240.0.0.0/4`, plus IPv6 equivalents | Never allowed                       |
| **Allow in private mode** | `10.0.0.0/8`, `127.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `::1/128`                                                                                                                 | Only with `--allow-private-targets` |
| **Allow**                 | All other public IPs                                                                                                                                                                      | Always allowed                      |

**Validation process:**

1. Parse URL, validate scheme (http/https)
2. Resolve hostname to IP addresses via DNS
3. Check each resolved IP against classification
4. Reject if any IP is in the Block list
5. Reject if any IP is in AllowInPrivateMode and private targets not enabled

**Config validation at startup:** `ValidateSecurityConfig()` parses all CIDR ranges at startup, failing fast on invalid entries.

Scanner Runner applies the same target policy at browser runtime for URL jobs: initial targets, redirects, final URLs, and HTTP(S) subresources are validated when `SCAN_URLS` is set. Hosted and self-hosted public deployments should also enforce a container or host egress policy; see [infra/security/egress-policy.example.md](../../infra/security/egress-policy.example.md).

### Additional Security Measures

| Measure                       | Implementation                                                                                              |
| ----------------------------- | ----------------------------------------------------------------------------------------------------------- |
| **Dependency scanning**       | `gitleaks` on every commit + CI, `govulncheck` across all Go modules, `bun audit --audit-level=high`        |
| **Container security**        | Rootless Podman job pods, `no-new-privileges:true`, resource limits, logging limits (`max-size`/`max-file`) |
| **Credential aliasing**       | `MINIO_ACCESS_KEY` ↔ `MINIO_ROOT_USER`, `MINIO_SECRET_KEY` ↔ `MINIO_ROOT_PASSWORD`                          |
| **VPS deployment guardrails** | Protected hostname check prevents accidental production disruption                                          |
| **API key auth**              | `X-Api-Key` header on all API endpoints (except health/scanners)                                            |
| **Request timeouts**          | All endpoints have timeout middleware except SSE stream                                                     |

---

## Authenticated Scanning

StageFlow can scan beyond a marketing landing page or login redirect by
attaching a session-aware authentication block to a job's Provenance. This
section is the source of truth for the contract, the trust boundaries, and
the storage-state retention rules.

### The contract

`Provenance.auth` is an optional, discriminated union that flows from the CLI
through the platform-api and orchestrator into the scanner-runner. When it is
absent the runtime is byte-identical to the pre-auth shape on disk and on the
wire.

```jsonc
// libs/contracts/provenance/schema/provenance.schema.json
{
  "auth": {
    "mode": "form",                               // "form" | "storage_state"
    "login_url": "https://app.example.com/login",
    "steps": [
      { "type": "fill", "selector": "input[name=email]",    "value": { "from_env": "STAGEFLOW_AUTH_USER" } },
      { "type": "fill", "selector": "input[name=password]", "value": { "from_env": "STAGEFLOW_AUTH_PASSWORD" } },
      { "type": "click", "selector": "button[type=submit]" }
    ],
    "success": { "type": "selector", "selector": "[data-test=signed-in]" }
  }
}

// or

{
  "auth": {
    "mode": "storage_state",
    "artifact_key": "<jobID>/auth/storage-state.json"
  }
}
```

Two important invariants:

- `form` step values are either literal strings or `{from_env: NAME}`
  references resolved at scanner-runner execution time against an explicit
  allow-list derived from the recipe. Resolved credential values never appear
  in stored Provenance, the unified report, the scan stage log, or any NATS
  event.
- `storage_state` carries an `artifact_key` only. The captured Playwright JSON
  lives at that key under the job's MinIO prefix; bytes are never persisted
  to Postgres or re-emitted on subsequent NATS events.

### Trust boundaries

```
┌────────────────────────────────────────────────────────────────────────────┐
│                       Authenticated Scanning Boundaries                    │
│                                                                            │
│  Developer machine                                                         │
│   • `stageflow auth capture` is the only flow that ever sees a real        │
│     password. It launches non-headless Chromium via Playwright and writes  │
│     storageState JSON locally with file mode 0600.                         │
│   • `stageflow scan --auth-state <path>` base64-encodes the captured file  │
│     and ships it inline with the URL submission. `--auth-recipe <path>`    │
│     loads a YAML/JSON recipe whose values reference env vars by name only. │
│        │                                                                   │
│        ▼                                                                   │
│  Platform API                                                              │
│   • Validates auth.mode, auth.form schema, and the storage-state byte      │
│     limit (1 MiB). Never sees a password. Forwards JobConfig.Auth on the   │
│     job.created event without resolving any from_env reference.            │
│        │                                                                   │
│        ▼                                                                   │
│  Orchestrator                                                              │
│   • For storage_state with inline content: uploads to                      │
│     scanner-artifacts/<jobID>/auth/storage-state.json and rewrites Auth    │
│     to `{mode: storage_state, artifact_key}` before persisting Job.Config  │
│     to Postgres. Bytes never appear in the database.                       │
│   • Walks Provenance.auth.form.steps for {from_env: NAME} references and  │
│     forwards exactly those env-var names from the orchestrator host into  │
│     the scanner-runner pod via the launch plan. Anything else from the    │
│     host environment stays out of the pod. Unresolved references fail     │
│     fast with a structured error before the pod starts.                   │
│        │                                                                   │
│        ▼                                                                   │
│  Scanner-runner pod                                                        │
│   • Reads PROVENANCE_AUTH_JSON (form recipe with from_env refs, or         │
│     storage_state with artifact_key) and merges it into the Provenance.    │
│   • PageIterator hydrates auth once per scanner before iterating pages:    │
│     storage_state mode downloads the artifact and feeds it into            │
│     Playwright's `storageState`; form mode replays the recipe through the  │
│     existing PreScanAction executor with a SecretsResolver bound to the   │
│     recipe's allow-list. Resolved values stay in process memory.          │
│   • SSRF policy applies to login_url and to every subsequent navigation.   │
│   • Hydration failure surfaces an `auth-hydration-failed` issue at         │
│     severity `critical` and skips downstream pages for that scanner.       │
└────────────────────────────────────────────────────────────────────────────┘
```

### Storage-state retention

A captured storage-state file is treated as a job-scoped credential:

- It is uploaded once to `scanner-artifacts/<jobID>/auth/storage-state.json`
  by the orchestrator during job-created handling.
- It is subject to the existing scan-artifact retention policy and is removed
  at the same time as the rest of the job's MinIO objects.
- The Web UI never receives a signed URL for the storage-state object — it is
  not exposed via the public artifact surface that other scan outputs use.
- The scanner-runner downloads it to the job workspace, sets file mode 0600,
  and deletes it during `cleanup()` at end-of-run.

If a storage-state file expires (cookies invalidated, session terminated by
the target), the scan fails with `auth-hydration-failed`. Capture a new one
with `stageflow auth capture` and re-submit.

### Where to read more

- The PR-1 runtime implementation: `services/scanner-runner/src/core/`
  (`auth-hydrator.ts`, `secrets-resolver.ts`, `page-iterator.ts`).
- The PR-2 user-facing surfaces: `clients/cli/cobra_auth.go`,
  `clients/cli/cobra_scan.go`, `clients/cli/auth_intake.go`, and the
  orchestrator launch planner at
  `services/orchestrator/internal/application/jobs/scanner_launch_planner.go`.
- The shared from_env walker: `libs/go/provenance/auth.go` (Go) is a direct
  port of `services/scanner-runner/src/core/secrets-resolver.ts`
  (`collectFromEnvReferences`); fixture-driven tests keep the two in sync.

---

## Storage Model

### MinIO Object Storage

**Interface** (`libs/go/storage/client.go`):

```go
type Client interface {
    Uploader
    Downloader
    Deleter
    Presigner
    FileExists(ctx, bucket, key) (bool, error)
}
```

**Buckets:**

| Bucket              | Purpose                                  |
| ------------------- | ---------------------------------------- |
| `scanner-staging`   | Temporary storage for uploaded ZIP files |
| `scanner-artifacts` | Permanent storage for all scan outputs   |

**Object Key Patterns:**

```
staging/{jobID}/{filename}              Uploaded ZIP files
{jobID}/report.json                     Aggregated unified report
{jobID}/report.html                     HTML report
{jobID}/provenance.json                 Page provenance document
{jobID}/stage.log                       Stage execution log
{jobID}/recipe.json                     Stage recipe
{jobID}/{scannerType}/results.json      Per-scanner results
{jobID}/{scannerType}/report.html       Per-scanner HTML report
{jobID}/{scannerType}/screenshots/...   Per-scanner screenshots
{jobID}/stage-logs/...                  Stage logs
```

**URL Generation Modes:**

| Mode           | Configuration               | Use Case                                         |
| -------------- | --------------------------- | ------------------------------------------------ |
| Presigned URLs | Default                     | Direct MinIO access with time-limited signatures |
| Proxy URLs     | `MINIO_USE_PROXY_URLS=true` | Routes through Caddy reverse proxy               |

### Storage Flow

```
┌─────────────┐     ┌──────────────┐     ┌──────────────┐
│  Platform   │     │ Orchestrator │     │   Scanner    │
│    API      │     │              │     │   Runner     │
└──────┬──────┘     └──────┬───────┘     └──────┬───────┘
       │                   │                    │
       │ Upload ZIP        │                    │
       │ to staging        │                    │
       ▼                   │                    │
  ┌─────────────┐          │                    │
  │   MinIO     │          │                    │
  │  staging    │          │ Download ZIP       │
  │  bucket     │─────────►│ for extraction     │
  └─────────────┘          │                    │
                           │                    │
                           │ Download provenance│
                           │ from artifacts     │
                           │                    │
                           │                    │ Upload results
                           │                    │ to artifacts
                           │                    ▼
                           │              ┌─────────────┐
                           │              │   MinIO     │
                           │              │  artifacts  │
                           │              │   bucket    │
                           │              └──────┬──────┘
                           │                     │
                           │ Download scanner    │
                           │ results for         │
                           │ aggregation         │
                           │                     │
                           │ Upload unified      │
                           │ report              │
                           ▼                     │
                      ┌─────────────┐            │
                      │   MinIO     │◄───────────┘
                      │  artifacts  │
                      │   bucket    │
                      └──────┬──────┘
                             │
                      Platform API
                      generates presigned
                      URLs for clients
```

---

## Database Schemas

### PostgreSQL (Orchestrator)

**Schema:** `services/orchestrator/internal/adapters/repository/schema.sql`

**`jobs` table:**

| Column                     | Type        | Purpose                                       |
| -------------------------- | ----------- | --------------------------------------------- |
| `id`                       | UUID (PK)   | Job identifier                                |
| `state`                    | VARCHAR     | Current FSM state                             |
| `input_type`               | VARCHAR     | "zip" or "urls"                               |
| `input_path`               | VARCHAR     | MinIO key for ZIP jobs                        |
| `urls`                     | JSONB       | URL list for URL jobs                         |
| `config_json`              | JSONB       | JobConfig (modules, scanner_configs, options) |
| `pod_id`                   | VARCHAR     | Podman pod ID                                 |
| `total_pages`              | INT         | Total pages to scan                           |
| `current_page`             | INT         | Current progress                              |
| `total_violations`         | INT         | Aggregate violation count                     |
| `expected_scanners`        | JSONB       | List of expected scanner types                |
| `completed_scanners`       | JSONB       | List of completed scanner types               |
| `scanner_results`          | JSONB       | Per-scanner metrics map                       |
| `report_json_key`          | VARCHAR     | MinIO key for unified report                  |
| `report_key`               | VARCHAR     | MinIO key for HTML report                     |
| `provenance_path`          | VARCHAR     | MinIO key for provenance                      |
| `scan_stage_log_key`       | VARCHAR     | MinIO key for stage log                       |
| `scan_recipe_key`          | VARCHAR     | MinIO key for recipe                          |
| `extraction_stage_log_key` | VARCHAR     | MinIO key for extraction log                  |
| `extraction_recipe_key`    | VARCHAR     | MinIO key for extraction recipe               |
| `error`                    | TEXT        | Error message                                 |
| `error_details`            | JSONB       | Structured error details                      |
| `last_stage`               | VARCHAR     | Last execution stage                          |
| `timing_extraction_ms`     | BIGINT      | Extraction duration                           |
| `timing_scanning_ms`       | BIGINT      | Scanning duration                             |
| `timing_reporting_ms`      | BIGINT      | Reporting duration                            |
| `created_at`               | TIMESTAMPTZ | Creation time                                 |
| `updated_at`               | TIMESTAMPTZ | Last update time                              |
| `completed_at`             | TIMESTAMPTZ | Completion time                               |

**`job_events` table (append-only audit log):**

| Column                | Type           | Purpose                                        |
| --------------------- | -------------- | ---------------------------------------------- |
| `id`                  | BIGSERIAL (PK) | Event sequence                                 |
| `job_id`              | UUID (FK)      | Parent job                                     |
| `event_type`          | VARCHAR        | Event type (job.created, scan.completed, etc.) |
| `nats_subject`        | VARCHAR        | NATS subject                                   |
| `nats_stream`         | VARCHAR        | NATS stream name                               |
| `nats_consumer`       | VARCHAR        | Consumer name                                  |
| `nats_stream_seq`     | BIGINT         | Stream sequence number                         |
| `nats_consumer_seq`   | BIGINT         | Consumer sequence number                       |
| `nats_deliveries`     | INT            | Delivery attempt count                         |
| `nats_stored_at`      | TIMESTAMPTZ    | NATS storage timestamp                         |
| `handler_status`      | VARCHAR        | Success/failed                                 |
| `handler_duration_ms` | BIGINT         | Handler execution time                         |
| `created_at`          | TIMESTAMPTZ    | Event timestamp                                |

### SQLite (Platform API)

**Job Status Projection** (`services/platform-api/internal/status/schema.sql`):

| Column                                            | Type        | Purpose                  |
| ------------------------------------------------- | ----------- | ------------------------ |
| `job_id`                                          | TEXT (PK)   | Job identifier           |
| `state`                                           | TEXT        | Current state            |
| `input_type`                                      | TEXT        | "zip" or "urls"          |
| `created_at`, `updated_at`, `completed_at`        | TEXT        | Timestamps               |
| `error`                                           | TEXT        | Error message            |
| `total_pages`, `current_page`, `total_violations` | INT         | Progress metrics         |
| Various artifact key columns                      | TEXT        | MinIO object keys        |
| `expected_scanners`, `completed_scanners`         | TEXT (JSON) | Scanner tracking         |
| `scanner_artifacts`                               | TEXT (JSON) | Per-scanner artifact map |

**Projects** (`services/platform-api/internal/project/schema.sql`):

| Table          | Columns                                                                                             | Purpose                                        |
| -------------- | --------------------------------------------------------------------------------------------------- | ---------------------------------------------- |
| `projects`     | `id` (PK), `slug` (unique), `name`, `urls` (JSON), `scanners` (JSON), `baseline_job_id`, timestamps | Registered scan targets with baseline tracking |
| `project_jobs` | `project_id` (FK), `job_id` (FK), cascade delete                                                    | Junction table linking projects to jobs        |

---

## CLI Architecture

### Command Structure

```
clients/cli/
├── main.go                         # Entry point → run()
├── cobra_root.go                   # Root command + persistent flags
├── cobra_scan.go                   # scan command
├── cobra_diff.go                   # diff command
├── cobra_ai.go                     # ai command
├── cobra_project.go                # project + init + doctor (local mode)
├── cobra_project_remote.go         # project create/list/show/update/delete/promote
├── cobra_project_update.go         # project update (partial)
├── cobra_report.go                 # report command
├── cobra_scanners.go               # scanners command
├── cobra_version.go                # version command
├── cobra_completion.go             # completion command
├── cobra_docs.go                   # docs command
├── client.go                       # HTTP API client
├── client_projects.go              # Remote project API client
├── types.go                        # API request/response types
├── sse.go                          # SSE streaming client
├── report_output.go                # Text report rendering
├── report_output_markdown.go       # Markdown report rendering
├── report_flags.go                 # Report render options
├── filter.go                       # Issue filtering and sorting
├── project_config.go               # .stageflow/config.yaml types
├── dev_stack.go                    # Dev server lifecycle management
├── local_targets.go                # Private/loopback target validation
└── cli_errors.go                   # exitCodeError type
```

### Testable Design

```go
func run(args []string, getenv func(string) string, stdout, stderr io.Writer) error
```

The `run()` function accepts injected dependencies — no global state, fully testable.

### Flag Precedence

```
CLI flags > Project config (.stageflow/config.yaml) > Env vars > Defaults
```

The `cobraFlagChanged()` helper checks whether a flag was explicitly set on the command line to determine precedence.

### Dev Server Lifecycle (`dev_stack.go`)

```
1. Run dev.up command steps (e.g., docker compose up)
2. Start dev.start.cmd as subprocess (Setpgid: true on Unix for process group)
3. Poll dev.ready.url until HTTP 2xx/3xx or timeout
4. Run the scan (within remaining timeout budget)
5. Stop: send SIGINT (configurable), wait up to timeout, then SIGKILL
6. Run dev.down command steps
```

### Auto-Detection (`project init`)

The `project init` command intelligently detects:

- **Dev commands** by inspecting Justfile recipes (`stageflow-dev`, `dev`, `dev-web`, `run`) and `package.json` scripts (`stageflow:dev`, `dev`, `start`)
- **Package manager** via lockfiles (bun → `bun.lock`, pnpm → `pnpm-lock.yaml`, yarn → `yarn.lock`, npm → `package-lock.json`)
- **Dev URL** — guesses Vite default 5173, fallback 3000

---

## Web App Architecture

### Technology Stack

- **Framework:** SvelteKit 5
- **Styling:** Tailwind CSS
- **Testing:** Vitest component/unit tests, Storybook test-runner, and Storybook a11y checks
- **Build:** Static assets served by Caddy in the frontend container

### Key Components

```
clients/web/
├── src/
│   ├── routes/                     # Landing, playground, live scan, report routes
│   └── lib/
│       ├── api/                    # Platform API client, URL helpers, SSE plumbing
│       ├── components/playground/  # URL/ZIP submission, scanner selection, AI/auth config
│       ├── components/report/      # Report shell, issue detail modal, artifacts, visual review
│       ├── components/scan-status/ # Live status, terminal, artifacts sidebar
│       ├── components/ui/          # Design-system primitives and Storybook stories
│       ├── config/                 # VITE_* env var normalization
│       ├── domain/scanners/        # Scanner presets and product grouping
│       ├── report/                 # Filtering, grouping, sorting, screenshots, severity helpers
│       ├── stores/                 # Svelte stores for scan status, monitor, report, history
│       └── types/                  # TypeScript report/scan types
├── static/                         # Static assets served by SvelteKit/Caddy
├── tests/unit/                     # Vitest API, store, utility, and component tests
└── .storybook/                     # Storybook configuration and test harnesses
```

### Client-Side Types

| Type / module                     | Purpose                                                        |
| --------------------------------- | -------------------------------------------------------------- |
| `src/lib/types/scan.ts`           | Job status, scan status strings, progress, artifacts, scanners |
| `src/lib/types/unified-report.ts` | Frontend-facing aliases for the canonical report contract      |
| `ScannerDefinition`               | Browser/API representation of `scannerregistry.Info`           |
| `ScannerSelection`                | `{id, enabled, config?}` selection state for scan submission   |
| `ScreenshotArtifact`              | Discriminated screenshot/link metadata rendered in reports     |

### SSE Streaming

The web app subscribes to `/api/v1/jobs/{id}/stream` with:

- Auto-reconnect on connection failure
- Event buffering for missed updates
- Real-time progress bar updates
- Scanner completion notifications

---

## Observability

### Primary Signals

| Signal                     | Source                        | Access                                  |
| -------------------------- | ----------------------------- | --------------------------------------- |
| Job state transitions      | Platform API SSE              | `/api/v1/jobs/{id}/stream`              |
| Orchestrator event history | PostgreSQL `job_events` table | Internal API `/api/v1/jobs/{id}/events` |
| Scanner artifacts          | MinIO object storage          | Presigned URLs via API                  |
| Service logs               | Container stdout/stderr       | `just dev logs`                         |

### Grafana Dashboards

**Provisioned via** `infra/grafana/provisioning/`:

| Dashboard                    | Data Source | Purpose                                                  |
| ---------------------------- | ----------- | -------------------------------------------------------- |
| `job-overview.json`          | PostgreSQL  | Job counts by state, completion rates, timing breakdowns |
| `provenance-validation.json` | PostgreSQL  | Extraction success rates, page discovery metrics         |

### Debugging Tools

| Tool             | Location                       | Purpose                                                    |
| ---------------- | ------------------------------ | ---------------------------------------------------------- |
| `stageflow` CLI  | `clients/cli/`                 | Submit scans, render reports, severity gating              |
| `job-status-cli` | `devtools/ops/job-status-cli/` | Inspect jobs/events/pods/status via orchestrator admin API |
| `suite-runner`   | `devtools/qa/suite-runner/`    | Threshold-based multi-domain validation                    |

### Recommended Debug Path

1. Check job terminal state and latest event
2. Inspect orchestrator timeline for stage boundary failure
3. Pull scanner artifacts and verify payload validity
4. Correlate service logs with event timestamps

---

## Deployment Topology

### Local Development

```
┌─────────────────────────────────────────────────────────────────┐
│                    Local Machine                                 │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │              Podman Compose (stageflow project)           │   │
│  │                                                          │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐   │   │
│  │  │platform- │ │orchestra-│ │ frontend │ │  nats    │   │   │
│  │  │  api     │ │  tor     │ │(Caddy+   │ │          │   │   │
│  │  │:8080     │ │:8081     │ │ SvelteKit│ │:4222     │   │   │
│  │  └──────────┘ └──────────┘ │ :3010)   │ └──────────┘   │   │
│  │                            └──────────┘                 │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐                 │   │
│  │  │  minio   │ │ postgres │ │ grafana  │                 │   │
│  │  │:9000/9001│ │  :5432   │ │ :3001    │                 │   │
│  │  └──────────┘ └──────────┘ └──────────┘                 │   │
│  │                                                          │   │
│  │  Network: stageflow_net                                  │   │
│  │  POD_NETNS_MODE: host                                    │   │
│  │  Private targets: enabled                                │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  External: stageflow CLI (go run or installed binary)           │
│  External: Browser at http://localhost:3010                     │
└─────────────────────────────────────────────────────────────────┘
```

### Staging

```
┌─────────────────────────────────────────────────────────────────┐
│                    Staging Server                                 │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │         Podman Compose (stageflow-staging project)        │   │
│  │                                                          │   │
│  │  Same services as local, different port ranges:          │   │
│  │  • Frontend: 3300                                        │   │
│  │  • API: 8300                                             │   │
│  │  • MinIO: 9300/9301                                      │   │
│  │  • Grafana: 3301                                         │   │
│  │                                                          │   │
│  │  Network: stageflow_staging_net                          │   │
│  │  POD_NETNS_MODE: bridge                                  │   │
│  │  Private targets: disabled                               │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  External Caddy (not in repo)                            │   │
│  │  staging.stageflow.org ──► localhost:3300 (frontend)     │   │
│  │                    ──► localhost:8300 (API)              │   │
│  │                    ──► localhost:9300 (MinIO)            │   │
│  │                    ──► localhost:3301 (Grafana)          │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Production (stageflow.org)

```
┌─────────────────────────────────────────────────────────────────┐
│                    Production VPS                                 │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │         Podman Compose (stageflow project)                │   │
│  │                                                          │   │
│  │  Same services, NO host ports exposed by compose         │   │
│  │  Internal ports: API 8100, MinIO 9100, Grafana 3101,     │   │
│  │                  Frontend 3100                           │   │
│  │                                                          │   │
│  │  Network: stageflow_net                                  │   │
│  │  POD_NETNS_MODE: bridge                                  │   │
│  │  Private targets: disabled                               │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  Production Caddy (infra/caddy/Caddyfile)                │   │
│  │  stageflow.org ──► /api/* → platform-api:8100            │   │
│  │                ──► /scanner-artifacts/* → MinIO:9100     │   │
│  │                ──► /monitoring* → Grafana:3101           │   │
│  │                ──► /* → frontend:3100                    │   │
│  │                                                          │   │
│  │  TLS via Let's Encrypt                                   │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  Production operations via shared root control plane:           │
│  cd $STAGEFLOW_PROD_DEPLOY_DIR && just_deploy stageflow         │
└─────────────────────────────────────────────────────────────────┘
```

### Environment Comparison

| Aspect              | Local Dev                  | Staging                 | Production                        |
| ------------------- | -------------------------- | ----------------------- | --------------------------------- |
| **Domain**          | `localhost`                | `staging.stageflow.org` | `stageflow.org`                   |
| **Compose project** | `stageflow`                | `stageflow-staging`     | `stageflow`                       |
| **Frontend port**   | 3010                       | 3300                    | 3100 (Caddy proxy)                |
| **API port**        | 8080                       | 8300                    | 8100 (Caddy proxy)                |
| **MinIO ports**     | 9000, 9001                 | 9300, 9301              | 9100 (Caddy proxy)                |
| **Grafana port**    | 3001                       | 3301                    | 3101 (Caddy proxy)                |
| **NATS port**       | 4222 (exposed)             | Not exposed             | Not exposed                       |
| **Private targets** | Enabled                    | Disabled                | Disabled                          |
| **MinIO SSL**       | `false`                    | `true` (default)        | `true`                            |
| **Pod netns mode**  | `host`                     | `bridge`                | `bridge`                          |
| **CORS origins**    | `localhost:3010,3000,8080` | `staging.stageflow.org` | `stageflow.org,www.stageflow.org` |
| **Edge proxy**      | None                       | External Caddy          | Production Caddy                  |

---

## Failure Modes

### Intake Failures

| Failure                      | Detection                          | Outcome                              |
| ---------------------------- | ---------------------------------- | ------------------------------------ |
| Invalid URL format/scheme    | API validation                     | Rejected with 400 + structured error |
| SSRF policy violation        | IP classification + DNS resolution | Rejected with 403                    |
| Payload size/shape violation | Request body limits                | Rejected with 413/400                |
| Invalid module names         | Scanner registry resolution        | Rejected with 400                    |
| Invalid scanner config       | Manifest configSchema validation   | Rejected with 400                    |

### Extraction Failures

| Failure                     | Detection                                   | Outcome                                  |
| --------------------------- | ------------------------------------------- | ---------------------------------------- |
| Malicious/malformed archive | Entry count, size, compression ratio checks | Job → FAILED with extraction-stage error |
| Path traversal attempt      | Path normalization + prefix check           | Job → FAILED                             |
| Archive limits exceeded     | Per-entry and total size constraints        | Job → FAILED                             |

### Scanner Failures

| Failure                           | Detection                        | Outcome                                    |
| --------------------------------- | -------------------------------- | ------------------------------------------ |
| Scanner Runner crash              | Container exit code              | Job → FAILED or partial handling           |
| Plugin load/validation mismatch   | Manifest validation at load time | Job → FAILED                               |
| External page/runtime instability | Scanner error handling           | scan.failed event → job failure or partial |
| Browser automation timeout        | Playwright timeout               | scan.failed event                          |

### Aggregation/Report Failures

| Failure                          | Detection              | Outcome      |
| -------------------------------- | ---------------------- | ------------ |
| Missing/invalid scanner artifact | MinIO download failure | Job → FAILED |
| Contract/merge incompatibility   | Schema validation      | Job → FAILED |
| Deduplication error              | Rule dedup logic       | Job → FAILED |

### Infrastructure Failures

| Failure                   | Detection                                    | Recovery                           |
| ------------------------- | -------------------------------------------- | ---------------------------------- |
| NATS unavailable          | Connection failure on startup                | Service fails fast (not resilient) |
| PostgreSQL unavailable    | Connection failure on startup                | Orchestrator fails fast            |
| MinIO unavailable         | Bucket ensure failure (30 retries, 2s delay) | Service fails after retries        |
| Podman socket unavailable | Connection failure                           | Orchestrator fails fast            |
| Stuck jobs                | Deadline sweeper (configurable interval)     | Jobs transitioned to FAILED        |

---

## Testing Architecture

### Test Pyramid

```
                         ┌─────────────────────┐
                         │   Golden Regression  │
                         │   Full pipeline:     │
                         │   scan → baseline →  │
                         │   diff → compare     │
                         ├─────────────────────┤
                         │   E2E Acceptance     │
                         │   Live API URL/ZIP   │
                         │   scans (RUN_E2E=1)  │
                         ├─────────────────────┤
                         │   Integration        │
                         │   Orchestrator E2E   │
                         │   with mock Podman   │
                         ├─────────────────────┤
                         │   Unit               │
                         │   Go race tests      │
                         │   Vitest             │
                         │   Storybook          │
                         └─────────────────────┘
```

### Test Frameworks

| Language       | Framework                          | Location                              |
| -------------- | ---------------------------------- | ------------------------------------- |
| **Go**         | `testing` (stdlib) + `-race`       | All Go packages, qa/e2e, devtools     |
| **TypeScript** | Vitest with `v8` coverage          | services/scanner-runner/tests/        |
| **TypeScript** | Playwright + Storybook test-runner | clients/web/                          |
| **Shell**      | Bash with `set -Eeuo pipefail`     | devtools/scripts/tests/, qa/e2e/\*.sh |

### E2E Gating Pattern

```go
func requireE2E(t *testing.T) {
    if testing.Short() { t.Skip("...") }
    if os.Getenv("RUN_E2E") == "" { t.Skip("...") }
}
```

Double-gated: `go test ./...` runs unit tests only; `RUN_E2E=1 go test ./...` runs the full suite.

### Golden Regression Flow

```
qa/e2e/project-scan-golden.sh (316 lines)

1. Create project → baseline scan (clean HTML, 0 issues)
2. Promote baseline (stageflow project promote)
3. Update project URL → regression fixture (image-alt violation)
4. Regression scan (expect exit code 1)
5. Normalize output (jq: strip timestamps, IDs, durations, paths)
6. diff -u against golden fixtures in qa/fixtures/project-golden/
7. Structural assertions: 1 new issue, ruleId=image-alt, severity=critical
8. Cleanup (delete test project)
```

Golden fixtures fail when missing or different. Regenerate intentionally with `UPDATE_GOLDENS=1 qa/e2e/project-scan-golden.sh`.

### CI/CD Workflows

| Workflow              | Trigger                  | Duration | Key Gates                                                                |
| --------------------- | ------------------------ | -------- | ------------------------------------------------------------------------ |
| **CI**                | Push/PR to `main`        | ~30m     | workflow_lint → secrets → Go → web → Storybook → scanner-runner → images |
| **Golden Regression** | Manual + daily 08:23 UTC | ~90m     | Full stack bootstrap → golden test → teardown                            |
| **CLI Release**       | Tags `clients/cli/v*`    | ~15m     | Matrix build (5 OS/arch) → GitHub Release                                |

### Local Quality Gate

```bash
just ci
# Stale vocabulary check (naming drift guard)
# Go build, lint, test (-race), vulncheck
# CLI docs regression (git diff --exit-code)
# Shell regression tests
# Frontend CI, Storybook tests, audit
# Scanner-runner CI, audit
```

### Go Linting

30+ linters organized into categories:

| Category        | Linters                                                     |
| --------------- | ----------------------------------------------------------- |
| **Correctness** | errcheck, govet, staticcheck, nilerr, bodyclose, exhaustive |
| **Security**    | gosec, asciicheck, bidichk, depguard                        |
| **Quality**     | gocritic, revive, gocyclo, gocognit, dupl                   |
| **Tests**       | testifylint, thelper, tparallel                             |

Test file exclusions: bodyclose, dupl, errcheck, gosec, noctx are relaxed for `_test.go`.

---

## Code Reference Map

### Platform API

| Area                              | File                                                             |
| --------------------------------- | ---------------------------------------------------------------- |
| Entry point                       | `services/platform-api/cmd/server/main.go`                       |
| Configuration                     | `services/platform-api/cmd/server/config.go`                     |
| Route registration                | `services/platform-api/internal/api/router.go`                   |
| URL intake validation             | `services/platform-api/internal/api/handlers_jobs_url_submit.go` |
| ZIP intake handler                | `services/platform-api/internal/api/handlers_jobs_zip_upload.go` |
| SSRF checks                       | `services/platform-api/internal/api/security.go`                 |
| SSE stream handler                | `services/platform-api/internal/api/handlers_sse.go`             |
| Scanner config validation         | `services/platform-api/internal/api/scanner_configs.go`          |
| Project CRUD/scan handlers        | `services/platform-api/internal/api/handlers_projects.go`        |
| Job status/artifact/diff handlers | `services/platform-api/internal/api/handlers_jobs_status.go`     |
| Job status pipeline               | `services/platform-api/internal/jobstatus/`                      |
| Project store (SQLite)            | `services/platform-api/internal/project/store.go`                |
| Project store schema              | `services/platform-api/internal/project/schema.sql`              |
| Job status schema                 | `services/platform-api/internal/status/schema.sql`               |

### Orchestrator

| Area                     | File                                                                        |
| ------------------------ | --------------------------------------------------------------------------- |
| Entry point              | `services/orchestrator/cmd/orchestrator/main.go`                            |
| Configuration            | `services/orchestrator/cmd/orchestrator/config.go`                          |
| Event handlers           | `services/orchestrator/internal/orchestrator/events.go`                     |
| Lifecycle workflows      | `services/orchestrator/internal/application/jobs/`                          |
| Scanner launch planner   | `services/orchestrator/internal/application/jobs/scanner_launch_planner.go` |
| Domain policies          | `services/orchestrator/internal/domain/jobs/`                               |
| Repository adapter       | `services/orchestrator/internal/adapters/repository/`                       |
| Repository schema        | `services/orchestrator/internal/adapters/repository/schema.sql`             |
| Runtime adapter (Podman) | `services/orchestrator/internal/adapters/runtime/`                          |
| Messaging adapter        | `services/orchestrator/internal/adapters/messaging/`                        |
| Report aggregation       | `services/orchestrator/internal/adapters/storage/`                          |
| Rule deduplication       | `services/orchestrator/internal/adapters/storage/rule_deduplication.go`     |
| Internal API             | `services/orchestrator/internal/api/`                                       |

### Archive Extractor

| Area                  | File                                                         |
| --------------------- | ------------------------------------------------------------ |
| Entry point           | `services/archive-extractor/cmd/server/main.go`              |
| ZIP extraction        | `services/archive-extractor/internal/extractor/extractor.go` |
| Page discovery        | `services/archive-extractor/internal/discovery/`             |
| Provenance generation | `services/archive-extractor/internal/provenance/`            |
| Static HTTP server    | `services/archive-extractor/internal/server/`                |

### Scanner Runner

| Area                        | File                                                                  |
| --------------------------- | --------------------------------------------------------------------- |
| Worker entry point          | `services/scanner-runner/src/worker.ts`                               |
| Scanner base lifecycle      | `services/scanner-runner/src/core/scanner-base.ts`                    |
| Core types                  | `services/scanner-runner/src/core/types.ts`                           |
| Config loader               | `services/scanner-runner/src/core/config-loader.ts`                   |
| Plugin loader               | `services/scanner-runner/src/core/plugins/`                           |
| Event publisher             | `services/scanner-runner/src/core/event-publisher.ts`                 |
| Storage provider            | `services/scanner-runner/src/core/storage-provider/`                  |
| Browser manager             | `services/scanner-runner/src/core/browser-manager.ts`                 |
| Page iterator               | `services/scanner-runner/src/core/page-iterator.ts`                   |
| AI Navigator agent          | `services/scanner-runner/src/scanners/ai-navigator/agent.ts`          |
| AI Navigator vision client  | `services/scanner-runner/src/scanners/ai-navigator/vision-client.ts`  |
| AI Navigator page analyzer  | `services/scanner-runner/src/scanners/ai-navigator/page-analyzer.ts`  |
| AI Navigator action decider | `services/scanner-runner/src/scanners/ai-navigator/action-decider.ts` |

### CLI

| Area                      | File                                    |
| ------------------------- | --------------------------------------- |
| Entry point               | `clients/cli/main.go`                   |
| Root command              | `clients/cli/cobra_root.go`             |
| Scan command              | `clients/cli/cobra_scan.go`             |
| Diff command              | `clients/cli/cobra_diff.go`             |
| AI command                | `clients/cli/cobra_ai.go`               |
| Project (local)           | `clients/cli/cobra_project.go`          |
| Remote project commands   | `clients/cli/cobra_project_remote.go`   |
| Report rendering          | `clients/cli/report_output.go`          |
| Markdown rendering        | `clients/cli/report_output_markdown.go` |
| SSE streaming             | `clients/cli/sse.go`                    |
| Issue filtering           | `clients/cli/filter.go`                 |
| Dev server lifecycle      | `clients/cli/dev_stack.go`              |
| Project config            | `clients/cli/project_config.go`         |
| API client                | `clients/cli/client.go`                 |
| Remote project API client | `clients/cli/client_projects.go`        |
| Types                     | `clients/cli/types.go`                  |
| Local target validation   | `clients/cli/local_targets.go`          |

### Shared Libraries

| Area                | File                          |
| ------------------- | ----------------------------- |
| State machine       | `libs/go/domain/job/state.go` |
| Event types         | `libs/go/events/types.go`     |
| NATS client         | `libs/go/messaging/nats.go`   |
| Core models         | `libs/go/models/job.go`       |
| Scanner registry    | `libs/go/scannerregistry/`    |
| Scanner catalog     | `libs/go/scannercatalog/`     |
| Diff engine         | `libs/go/diff/diff.go`        |
| MinIO client        | `libs/go/storage/`            |
| Config loading      | `libs/go/config/`             |
| Structured logging  | `libs/go/logging/`            |
| Bootstrap utilities | `libs/go/bootstrap/`          |
| HTTP utilities      | `libs/go/httputil/`           |

### Contracts

| Area                      | File                                                                  |
| ------------------------- | --------------------------------------------------------------------- |
| Report schema             | `libs/contracts/report/schema/unified-report.v2.schema.json`          |
| Report Go types           | `libs/contracts/report/generated/go/report_schema.go`                 |
| Report TS types           | `libs/contracts/report/generated/typescript/unified-report.v2.ts`     |
| Report TS validator       | `libs/contracts/report/generated/typescript/validator.ts`             |
| Scanner manifest schema   | `libs/contracts/scanner-manifest/schema/scanner-manifest.schema.json` |
| Scanner manifest Go types | `libs/contracts/scanner-manifest/scanner_manifest.go`                 |
| Event schemas             | `libs/contracts/events/schema/`                                       |

### Infrastructure

| Area                 | File                                       |
| -------------------- | ------------------------------------------ |
| Base compose         | `infra/compose/podman-compose.yml`         |
| Local overlay        | `infra/compose/podman-compose.local.yml`   |
| Test overlay         | `infra/compose/podman-compose.test.yml`    |
| Staging overlay      | `infra/compose/podman-compose.staging.yml` |
| Caddy config         | `infra/caddy/Caddyfile`                    |
| Scanner config       | `infra/scanners/scanners.yaml`             |
| MinIO bucket init    | `infra/minio/init-buckets.sh`              |
| Grafana provisioning | `infra/grafana/provisioning/`              |

### Testing

| Area              | File                            |
| ----------------- | ------------------------------- |
| E2E URL scan      | `qa/e2e/url_scan_test.go`       |
| E2E ZIP scan      | `qa/e2e/zip_scan_test.go`       |
| Golden regression | `qa/e2e/project-scan-golden.sh` |
| Golden fixtures   | `qa/fixtures/project-golden/`   |
| Test fixtures     | `qa/fixtures/`                  |
| Suite runner      | `devtools/qa/suite-runner/`     |
| Job status CLI    | `devtools/ops/job-status-cli/`  |
| Shell tests       | `devtools/scripts/tests/`       |
| Pre-commit hooks  | `.pre-commit-config.yaml`       |
| Go lint config    | `.golangci.yml`                 |

### CI/CD

| Workflow          | File                                          |
| ----------------- | --------------------------------------------- |
| Main CI           | `.github/workflows/ci.yml`                    |
| Golden regression | `.github/workflows/golden-regression.yml`     |
| CLI release       | `.github/workflows/release-stageflow-cli.yml` |
| Dependabot        | `.github/dependabot.yml`                      |

### Task Runner

| Area              | File                                  |
| ----------------- | ------------------------------------- |
| Justfile          | `justfile`                            |
| Diagnose script   | `infra/scripts/diagnose-local-env.sh` |
| Go workspace dirs | `devtools/scripts/go/go-work-dirs.sh` |
