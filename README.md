# StageFlow

[![CI](https://github.com/mattboback/stageflow/actions/workflows/ci.yml/badge.svg)](https://github.com/mattboback/stageflow/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

**Live demo:** [stageflow.org](https://stageflow.org) | **Run locally:** `cp .env.example .env && just diagnose && just demo`

Podman-native web accessibility and quality scanning platform.

StageFlow runs multi-scanner audits against live URLs or static-site ZIP archives, streams job progress in real time, and merges heterogeneous scanner outputs into one normalized report. Everything needed to run StageFlow is in this repo — the hosted demo and a self-hosted instance use identical code.

This repository is set up for self-hosting and code review from the same source tree. The live `stageflow.org` deployment uses this codebase, but local development and self-hosted installs should start from `.env.example`, replace all `change-me` values before exposing anything outside localhost, and treat `stageflow.org` references as project examples rather than defaults to reuse unchanged.

![StageFlow — live scan pipeline dashboard](docs/images/hero.png)

---

## Table of Contents

- [What StageFlow Does](#what-stageflow-does)
- [Engineering Highlights](#engineering-highlights)
- [Architecture at a Glance](#architecture-at-a-glance)
- [Quick Start](#quick-start)
- [CLI Reference](#cli-reference)
- [Built-in Scanners](#built-in-scanners)
- [Job Lifecycle](#job-lifecycle)
- [Event Model](#event-model)
- [Security Model](#security-model)
- [Repo Map](#repo-map)
- [Testing Strategy](#testing-strategy)
- [Configuration](#configuration)
- [Screenshots](#screenshots)
- [For Reviewers](#for-reviewers)
- [Docs](#docs)
- [Support](#support)
- [License](#license)

---

## What StageFlow Does

StageFlow solves the problem of **running multiple web quality scanners against a website and unifying their disparate outputs into a single actionable report**. It accepts two input types:

1. **URL scans** — submit one or more URLs, StageFlow crawls each page with every enabled scanner
2. **ZIP scans** — upload a static-site archive, StageFlow safely extracts it, discovers all HTML pages, and scans them locally

The system runs **eight heterogeneous scanners** (accessibility, performance, SEO, security, link quality, spelling/grammar, social metadata, and AI-driven navigation), then **normalizes their outputs** into one `UnifiedReportV2` contract with severity scoring, deduplication, annotated screenshots, and regression detection.

### Two Client Surfaces

| Surface                    | Purpose                | Key Features                                                                                                                                                                      |
| -------------------------- | ---------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Web UI** (`clients/web`) | Visual scan management | Job submission, live SSE progress streaming, report visualization with annotated screenshots, scanner selection, AI Navigator playground                                          |
| **CLI** (`clients/cli`)    | Terminal/CI automation | Scan submission, SSE streaming, report rendering (text/markdown/JSON), `--fail-on` severity gating, Project Mode (local dev server lifecycle), remote project CRUD, baseline diff |

Both surfaces talk to the same Platform API — one backend, two very different client needs.

---

## Engineering Highlights

- **Multi-service Go/TypeScript architecture** coordinated through NATS JetStream with durable streams, replay, and explicit ACK
- **Real-time SSE streaming** from orchestrator through API to browser and CLI — unidirectional, proxy-friendly, no WebSocket complexity
- **Per-job Podman pod isolation** — each scan gets its own rootless pod with hard resource and network boundaries; scanner containers cannot affect the host even if compromised
- **Contract-driven development** — JSON Schema as the single source of truth for report format, scanner manifests, and event envelopes; generated Go + TypeScript types used by all consumers
- **Dual-surface API** — same backend drives both a SvelteKit web UI and a Go CLI with CI-ready exit codes
- **Stable content-based issue IDs** — deterministic hashes from rule + page + element context enable baseline diffing and regression detection
- **Scanner plugin system** — runtime-discovered plugins with manifest-defined capabilities; built-ins, mounted volumes, and user plugin directories
- **Go workspace monorepo** — `go.work` coordinates 21 Go modules across services, libs, clients, devtools, and QA with clean module boundaries

---

## Architecture at a Glance

### System Topology

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           StageFlow Platform                            │
│                                                                         │
│  ┌──────────┐    ┌──────────────────────────────────────────────────┐   │
│  │  Clients  │    │              Infrastructure                      │   │
│  │          │    │                                                   │   │
│  │ ┌──────┐ │    │  ┌──────┐  ┌──────┐  ┌──────────┐  ┌──────────┐ │   │
│  │ │ Web  │ │    │  │ NATS │  │ MinIO│  │PostgreSQL│  │ Grafana  │ │   │
│  │ │Svelte│ │    │  │JetStr│  │ S3   │  │ 17       │  │ 12       │ │   │
│  │ │Kit   │ │    │  │eam   │  │      │  │          │  │          │ │   │
│  │ └──┬───┘ │    │  └──┬───┘  └──┬───┘  └──────────┘  └──────────┘ │   │
│  │    │     │    │     │          │                                  │   │
│  │ ┌──┴───┐ │    │     │          │                                  │   │
│  │ │ Go   │ │    │     ▼          ▼                                  │   │
│  │ │ CLI  │ │    │  ┌──────────────────────────────────────────┐    │   │
│  │ └──┬───┘ │    │  │           Platform API (Go)              │    │   │
│  │    │     │    │  │  • URL/ZIP intake + SSRF validation      │    │   │
│  │    │     │    │  │  • Job/report APIs + SSE hub             │    │   │
│  │    │     │    │  │  • Project CRUD + baseline promotion     │    │   │
│  │    │     │    │  │  • On-demand diff engine                 │    │   │
│  │    │     │    │  │  • SQLite project store                  │    │   │
│  │    │     │    │  └────────────────┬─────────────────────────┘    │   │
│  └────┼───────────┼─────────────────┼──────────────────────────────┘   │
│       │           │                 │                                  │
│       │     POST  │           NATS events         SSE stream           │
│       │    /jobs  │        (job.created, etc.)    /jobs/{id}/stream    │
│       ▼           ▼                 ▼                                  │
│              ┌──────────────────────────────────────────────────┐      │
│              │              NATS JetStream                       │      │
│              │  Streams: jobs | extraction | scan                │      │
│              │  8 event types, durable consumers, explicit ACK   │      │
│              └──────────────────────┬───────────────────────────┘      │
│                                     │                                  │
│              ┌──────────────────────┴───────────────────────────┐      │
│              │              Orchestrator (Go)                     │      │
│              │  • Job FSM (PENDING→DONE/FAILED)                  │      │
│              │  • Podman pod lifecycle management                │      │
│              │  • Scanner coordination + completion tracking     │      │
│              │  • Report aggregation + deduplication             │      │
│              │  • PostgreSQL job state + event audit trail       │      │
│              │  • Deadline sweeper for stuck jobs                │      │
│              └──────────────────────┬───────────────────────────┘      │
│                                     │                                  │
│              ┌──────────────────────┴───────────────────────────┐      │
│              │              Podman Job Pod (per job)              │      │
│              │                                                   │      │
│              │  ┌───────────────────┐  ┌─────────────────────┐  │      │
│              │  │ Archive Extractor │  │   Scanner Runner    │  │      │
│              │  │ (Go, ZIP jobs)    │  │ (TS/Bun/Playwright) │  │      │
│              │  │ • Safe extraction │  │ • Plugin discovery  │  │      │
│              │  │ • ZIP bomb defense│  │ • Browser automation│  │      │
│              │  │ • Provenance gen  │  │ • Artifact upload   │  │      │
│              │  │ • Static server   │  │ • NATS publishing   │  │      │
│              │  └───────────────────┘  └─────────────────────┘  │      │
│              └───────────────────────────────────────────────────┘      │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### Technology Stack

| Layer                 | Technology                                                                                |
| --------------------- | ----------------------------------------------------------------------------------------- |
| **Backend services**  | Go 1.26.1 (platform-api, orchestrator, archive-extractor, CLI)                            |
| **Scanner runtime**   | TypeScript/Bun + Playwright (axe-core, Lighthouse)                                        |
| **Web frontend**      | SvelteKit 5 + Tailwind CSS + Storybook                                                    |
| **Message bus**       | NATS JetStream (durable streams, replay, explicit ACK)                                    |
| **Object storage**    | MinIO (artifacts, ZIP staging, reports)                                                   |
| **Relational DB**     | PostgreSQL 17 (orchestrator job state + event audit trail)                                |
| **Embedded DB**       | SQLite (platform-api project management, baseline tracking)                               |
| **Container runtime** | Rootless Podman (per-job pod isolation, no daemon, no privilege escalation)               |
| **Web server**        | Caddy (frontend container serves static SvelteKit build)                                  |
| **Observability**     | Grafana 12 (provisioned dashboards)                                                       |
| **Task runner**       | `just`                                                                                    |
| **CI/CD**             | GitHub Actions (workflow lint, secrets scan, Go quality, web CI, Storybook, image builds) |
| **AI**                | OpenRouter API (vision model for AI Navigator scanner)                                    |

---

## Quick Start

### Prerequisites

- [Go 1.26.1](https://go.dev/dl/)
- [Bun](https://bun.sh/)
- [Podman](https://podman.io/) with `podman compose`
- [just](https://github.com/casey/just)
- [golangci-lint v2](https://golangci-lint.run/)

### Fastest Local Smoke Test

```bash
git clone https://github.com/mattboback/stageflow.git
cd stageflow
cp .env.example .env
just diagnose
just demo
```

`just demo` runs the full bootstrap: prerequisite checks → image builds → stack restart → health endpoint waits → MinIO initialization → prints next scan commands.

### Manual Bootstrap

```bash
just setup          # Podman network, Go sync, Bun install
just images         # Build all container images
just dev up         # Start the local stack
just dev init       # Initialize MinIO buckets
```

| Service                | `dev` mode              | `local` overlay         |
| ---------------------- | ----------------------- | ----------------------- |
| Web App                | `http://localhost:3000` | `http://localhost:3010` |
| Platform API           | `http://localhost:8080` | `http://localhost:8080` |
| Orchestrator Admin API | `http://localhost:8081` | `http://localhost:8081` |

### Run a First Scan

```bash
just cli-install
stageflow scan https://example.com
```

To scan `localhost` or private targets, use the local overlay:

```bash
just setup && just images && just dev up local && just dev init local
```

Then dogfood StageFlow against the web client:

```bash
stageflow project init && stageflow project doctor . && stageflow project .
```

### Self-Hosting Notes

- Start from `.env.example`; never commit `.env`, `.env.staging`, or real credentials
- Replace every `change-me` value before using a public domain or shared environment
- Set `STAGEFLOW_PUBLIC_DOMAIN`, `PLATFORM_API_CORS_ALLOW_ORIGINS`, `GF_SERVER_ROOT_URL`, and the frontend `VITE_*` URLs for your deployment
- The frontend container serves the built SvelteKit app with Caddy on port `3000`; `infra/caddy/Caddyfile` is the separate optional edge proxy for public domains and TLS
- The repository includes `stageflow.org` references for the hosted demo, docs, and regression fixtures; treat them as project examples, not values to reuse unchanged

---

## CLI Reference

The `stageflow` CLI submits scan jobs, streams live progress, and renders unified reports in text, markdown, or JSON.

### Command Hierarchy

```
stageflow                              (root)
  ├── scan [url...]                    Submit URLs, wait, render results
  ├── diff <baseline> <current|url>    Compare baseline vs current report
  ├── ai <url> <objective>             Run AI Navigator scanner
  ├── project [path]                   Local project-mode scan (full lifecycle)
  │     ├── init [path]                Scaffold .stageflow/config.yaml
  │     ├── doctor [path]              Validate config + dev readiness
  │     ├── create <slug>              Create remote project
  │     ├── list                       List remote projects
  │     ├── show <slug>                Show remote project details
  │     ├── update <slug>              Update remote project (partial)
  │     ├── delete <slug>              Delete remote project
  │     └── promote <slug>             Promote job as project baseline
  ├── report <job-id>                  Fetch/display results for existing job
  ├── scanners                         List available scanners on API
  ├── version                          Print version/commit/date
  ├── completion [bash|zsh|fish|ps]    Generate shell completion scripts
  └── docs                             Generate Markdown CLI documentation
```

### Persistent Flags

| Flag              | Env Var             | Default                 | Description                               |
| ----------------- | ------------------- | ----------------------- | ----------------------------------------- |
| `--api <url>`     | `STAGEFLOW_API_URL` | `http://localhost:8080` | API base URL                              |
| `--api-key <key>` | `STAGEFLOW_API_KEY` | —                       | API key for authentication                |
| `--format <fmt>`  | —                   | `text`                  | Output format: `text`, `markdown`, `json` |

### Key Commands

**Submit a scan:**

```bash
stageflow scan https://example.com
stageflow scan https://example.com --scanners axe,seo --format json
stageflow scan https://example.com --fail-on serious   # exit 1 on regressions
stageflow scan https://example.com --api https://stageflow.org
```

**Run Project Mode (local dev lifecycle):**

```bash
stageflow project init          # Auto-detect dev commands from Justfile + package.json
stageflow project doctor .      # Validate config without scanning
stageflow project .             # Start dev server → scan → stop server
```

**Compare reports:**

```bash
stageflow diff baseline.json current.json
stageflow diff baseline.json https://example.com --fail-on-regression
```

**AI Navigator:**

```bash
stageflow ai https://example.com "Add an item to the cart and checkout"
```

### Exit Codes

| Code | Meaning                                                                   |
| ---- | ------------------------------------------------------------------------- |
| 0    | Scan completed, no issues at or above `--fail-on` threshold               |
| 1    | Scan completed but issues meet or exceed `--fail-on` severity             |
| 2    | CLI or API error (network failure, invalid arguments, malformed response) |

### Output Formats

| Format   | Flag                      | Use Case                                          |
| -------- | ------------------------- | ------------------------------------------------- |
| Text     | `--format text` (default) | Human review in terminal                          |
| Markdown | `--format markdown`       | PR comments, agent parsing                        |
| JSON     | `--format json`           | Machine consumption — full envelope with metadata |

### Report Filtering Flags

| Flag                    | Effect                                                     |
| ----------------------- | ---------------------------------------------------------- |
| `--fail-on <severity>`  | Exit 1 if any displayed issue is at or above this severity |
| `--severity <csv>`      | Only display issues matching these severities              |
| `--category <csv>`      | Only display issues matching these categories              |
| `--max-issues <n>`      | Cap returned issues (default 200, 0 = unlimited)           |
| `--max-occurrences <n>` | Cap occurrences per issue (default 3, 0 = unlimited)       |
| `--summary-only`        | Print summary counts only, skip individual findings        |
| `--group-by <mode>`     | Group findings by `category`, `scanner`, or `none`         |

### Severity Hierarchy

`critical` (4) > `serious` (3) > `moderate` (2) > `minor` (1) > `info` (0)

### CLI Report Envelope

The CLI wraps the unified report in a versioned envelope (`stageflow-cli/report@v1`):

```
reportEnvelope
├── schema          "stageflow-cli/report@v1"
├── cli             { version, commit, date }
├── api             { base_url }
├── job             { id, state, created_at, updated_at }
├── links           { job, results }          — API URLs
├── urls            [scanned URLs]
├── filters         { max_issues, issues_returned, issues_total, truncated, sort }
└── report          UnifiedReportV2
    ├── summary     { score, scoreGrade, totalIssues, bySeverity, byScanner }
    ├── issues[]    IssueDetail (sorted by severity desc, scanner asc, rule asc)
    ├── scanners[]  per-scanner status, timing, severity counts
    ├── pages[]     per-page issue counts and timing
    └── meta        { jobId, scannedAt, completedAt, durationMs }
```

---

## Built-in Scanners

StageFlow ships with **eight scanners**, each discovered at runtime via a manifest-based plugin system.

| Scanner            | Categories                               | Focus                                                                     |
| ------------------ | ---------------------------------------- | ------------------------------------------------------------------------- |
| `axe`              | accessibility                            | WCAG violations — landmarks, ARIA, color contrast, alt text, keyboard nav |
| `lighthouse`       | performance, accessibility, seo, quality | Google Lighthouse audits — Core Web Vitals, best practices, scores        |
| `seo`              | seo                                      | Meta tags, canonical URLs, structured data, content depth, title length   |
| `security-headers` | security                                 | HTTP header posture — CSP, HSTS, X-Frame-Options, Permissions-Policy      |
| `link-checker`     | quality                                  | Broken links, redirect chains, link quality                               |
| `open-graph`       | seo                                      | Open Graph and Twitter Card metadata validation                           |
| `spelling-grammar` | quality                                  | AI-assisted spelling and grammar analysis                                 |
| `ai-navigator`     | custom                                   | LLM-powered Playwright agent — goal-driven browser flow evaluation        |

### Scanner Plugin Architecture

```
Scanner Discovery Order:
  1. Built-ins (libs/go/scannercatalog/manifests/*/)
  2. Mounted /plugins volume
  3. ~/.stageflow/plugins/
  4. PLUGIN_PATHS env var

Each plugin provides:
  ├── manifest.json          — scanner-manifest schema contract
  ├── entrypoint module      — executable scanner code
  └── configSchema           — JSON Schema for SCANNER_OPTIONS validation

Scanner Lifecycle (ScannerBase):
  initialize() → iteratePages() → writeResults() → uploadArtifacts()
```

### Scanner Configuration

Scanner overrides are managed via YAML at `infra/scanners/scanners.yaml`:

```yaml
scanners:
  lighthouse:
    enabled: true
    capabilities:
      maxMemoryMB: 2048
      maxTimeoutMs: 60000
  ai-navigator:
    enabled: false # disabled by default (requires OpenRouter API key)
```

---

## Job Lifecycle

StageFlow uses a **strict finite state machine** to keep transitions explicit and debuggable.

### State Machine

```
                    ┌─────────────────────────────────────────────┐
                    │                                             │
                    ▼                                             │
  ┌─────────┐   ┌────────────┐   ┌──────────────┐   ┌──────────┐ │
  │ PENDING  │──►│ EXTRACTING │──►│ READY_TO_SCAN│──►│ SCANNING │ │
  └────┬─────┘   └─────┬──────┘   └──────┬───────┘   └────┬─────┘ │
       │               │                 │                │       │
       │     (URL jobs)│                 │                │       │
       └───────────────┘                 │                │       │
                                         │                ▼       │
                                         │          ┌──────────┐  │
                                         │          │COMPLETING│  │
                                         │          └────┬─────┘  │
                                         │               │        │
                                         │               ▼        │
                                         │          ┌──────────┐  │
                                         │          │   DONE   │  │
                                         │          └──────────┘  │
                                         │                        │
                                         ▼                        │
                                    ┌──────────┐                  │
                                    │  FAILED  │◄─────────────────┘
                                    └──────────┘
                                    (terminal, from any state)
```

### State Transitions

| From            | To              | Trigger                                           |
| --------------- | --------------- | ------------------------------------------------- |
| `PENDING`       | `EXTRACTING`    | ZIP job — orchestrator starts extractor container |
| `PENDING`       | `READY_TO_SCAN` | URL job — no extraction needed                    |
| `PENDING`       | `FAILED`        | Intake validation failure                         |
| `EXTRACTING`    | `READY_TO_SCAN` | `extraction.ready` event received                 |
| `EXTRACTING`    | `FAILED`        | `extraction.failed` event received                |
| `READY_TO_SCAN` | `SCANNING`      | Orchestrator launches scanner containers          |
| `SCANNING`      | `COMPLETING`    | All scanners complete                             |
| `COMPLETING`    | `DONE`          | Report aggregated and uploaded                    |
| _any_           | `FAILED`        | Any unrecoverable error                           |

### Two Input Flows

**URL Job Flow:**

```
Client ──POST /api/v1/jobs/urls──► Platform API
                                      │
                                      ├─ Validate URLs, modules, SSRF check
                                      ├─ Generate jobID (UUID), runID
                                      ├─ Publish job.created to NATS
                                      └─ Seed job status projection (PENDING)
                                              │
                                              ▼
                                      NATS JetStream (job.created)
                                              │
                                              ▼
                                      Orchestrator
                                      ├─ Transition PENDING → READY_TO_SCAN
                                      ├─ Launch scanner-runner containers
                                      ├─ Track scanner completions
                                      ├─ Aggregate report when all done
                                      └─ Publish job.completed
```

**ZIP Job Flow:**

```
Client ──POST /api/v1/jobs/zip──► Platform API
                                     │
                                     ├─ Validate ZIP, store in MinIO staging
                                     ├─ Publish job.created to NATS
                                     └─ Seed job status projection (PENDING)
                                             │
                                             ▼
                                     NATS JetStream (job.created)
                                             │
                                             ▼
                                     Orchestrator
                                     ├─ Create Podman job pod
                                     ├─ Start archive-extractor container
                                     │     ├─ Download ZIP from MinIO
                                     │     ├─ Validate archive safety
                                     │     ├─ Extract to workspace
                                     │     ├─ Discover HTML pages
                                     │     ├─ Generate provenance.json
                                     │     └─ Start static HTTP server
                                     ├─ Publish extraction.ready
                                     ├─ Transition to READY_TO_SCAN
                                     └─ Continue as URL flow from here
```

---

## Event Model

All services communicate through **NATS JetStream** with **3 streams** and **8 event types**.

### Streams

| Stream       | Subjects                                                                    | Max Age  |
| ------------ | --------------------------------------------------------------------------- | -------- |
| `jobs`       | `jobs.events.created`, `jobs.events.completed`, `jobs.events.failed`        | 72 hours |
| `extraction` | `extraction.events.ready`, `extraction.events.failed`                       | 72 hours |
| `scan`       | `scan.events.page.completed`, `scan.events.completed`, `scan.events.failed` | 72 hours |

### Event Types

| Event                 | Producer          | Consumers                  | Key Payload Fields                                           |
| --------------------- | ----------------- | -------------------------- | ------------------------------------------------------------ |
| `job.created`         | platform-api      | orchestrator               | jobID, inputType, inputPath, URLs, config                    |
| `extraction.ready`    | archive-extractor | orchestrator, platform-api | jobID, provenancePath, baseURL, totalPages                   |
| `extraction.failed`   | archive-extractor | orchestrator, platform-api | jobID, error, errorDetails                                   |
| `scan.page.completed` | scanner-runner    | orchestrator, platform-api | jobID, scannerType, pageID, pageIndex, totalPages            |
| `scan.completed`      | scanner-runner    | orchestrator, platform-api | jobID, scannerType, resultsPath, reportPath, summary, timing |
| `scan.failed`         | scanner-runner    | orchestrator, platform-api | jobID, scannerType, error, errorDetails                      |
| `job.completed`       | orchestrator      | platform-api               | jobID, artifacts, scannerArtifacts                           |
| `job.failed`          | orchestrator      | platform-api               | jobID, stage, error, errorDetails                            |

### Event Envelope

```json
{
  "event": "job.created",
  "job_id": "uuid",
  "request_id": "optional-correlation-id",
  "run_id": "optional-run-id",
  "timestamp": "2026-04-04T00:00:00Z",
  "producer": "platform-api",
  "payload": { ... }
}
```

### Messaging Properties

- **Durable consumers** with explicit ACK
- **Max 10 deliveries**, 10-minute ACK wait, 5-second NAK delay on failure
- **Strict payload validation** — `DisallowUnknownFields()` catches schema drift
- **Lenient envelope parsing** — forward-compatible event evolution
- **Correlation IDs** — `request_id` and `run_id` propagated through entire pipeline

---

## Security Model

StageFlow implements a **four-layer security model** with explicit trust boundaries.

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Security Layers                              │
│                                                                     │
│  Layer 4: Edge Proxy                                                │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  Caddy reverse proxy: rate limiting, TLS, WAF               │   │
│  │  File: infra/caddy/Caddyfile                                │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                              ▲                                      │
│  Layer 3: Scanner Runtime                                           │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  Per-job Podman pod isolation (rootless, no-new-privileges) │   │
│  │  Scanner identity validation against manifest               │   │
│  │  SCANNER_OPTIONS schema validation                          │   │
│  │  Resource limits (memory, CPU, timeout)                     │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                              ▲                                      │
│  Layer 2: Archive Extraction                                        │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  Entry-count limits                                         │   │
│  │  Per-entry and total size constraints                       │   │
│  │  Compression-ratio checks (ZIP bomb defense)                │   │
│  │  Path traversal prevention                                  │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                              ▲                                      │
│  Layer 1: API Intake                                                │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  URL scheme validation (http/https only)                    │   │
│  │  SSRF protection: IP classification, DNS resolution         │   │
│  │  Blocked: 0.0.0.0/8, 100.64.0.0/10, 169.254.0.0/16, etc.  │   │
│  │  Request size limits (2MB URL, 100MB ZIP)                   │   │
│  │  URL count limit (100), length limit (2048 chars)           │   │
│  │  API key middleware                                         │   │
│  └─────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
```

### SSRF Protection Details

The Platform API implements comprehensive SSRF protection for URL-based scan targets:

- **Three-tier IP classification**: `Allow`, `AllowInPrivateMode`, `Block`
- **Blocked ranges** (always): `0.0.0.0/8`, `100.64.0.0/10`, `169.254.0.0/16`, `192.0.0.0/24`, `192.0.2.0/24`, `198.18.0.0/15`, `198.51.100.0/24`, `203.0.113.0/24`, `224.0.0.0/4`, `240.0.0.0/4`, plus IPv6 equivalents
- **Metadata service**: `169.254.169.254` always blocked
- **Private ranges** (allowed only with `--allow-private-targets`): `10.0.0.0/8`, `127.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `::1/128`
- **DNS resolution validation**: Resolves hostnames and checks all resolved IPs against policy

### Additional Security Measures

- **Dependency scanning**: `gitleaks` on every commit + CI, `govulncheck` across all Go modules, `bun audit --audit-level=high` for TypeScript
- **Container security**: `security_opt: no-new-privileges:true` on all services, resource limits on every container, logging limits to prevent disk exhaustion
- **VPS deployment guardrails**: Protected hostname check prevents accidental production disruption from local commands

---

## Repo Map

| Directory                    | Contents                                                                                                                                       | Language    |
| ---------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- | ----------- |
| `clients/web`                | SvelteKit 5 web app with Tailwind CSS, Storybook                                                                                               | TypeScript  |
| `clients/cli`                | `stageflow` CLI — scan, project, diff, report commands                                                                                         | Go          |
| `services/platform-api`      | Intake API, SSRF validation, SSE hub, project CRUD, diff engine                                                                                | Go          |
| `services/orchestrator`      | Job FSM, Podman pod lifecycle, report aggregation, PostgreSQL persistence                                                                      | Go          |
| `services/archive-extractor` | Safe ZIP extraction, provenance generation, static HTTP server                                                                                 | Go          |
| `services/scanner-runner`    | Scanner plugin runtime, Playwright browser automation, NATS publishing                                                                         | TypeScript  |
| `libs/go/*`                  | 12 shared Go packages: bootstrap, config, diff, domain, events, httputil, logging, messaging, models, scannercatalog, scannerregistry, storage | Go          |
| `libs/contracts`             | JSON Schema contracts + generated Go/TS types for reports, scanner manifests, events                                                           | JSON Schema |
| `infra/`                     | Compose files (base, local, test, staging), Caddy, Grafana provisioning, MinIO init, scanner config                                            | YAML        |
| `devtools/`                  | Ops tools (job-status-cli), QA suite runner, shell scripts, pre-commit hooks                                                                   | Go, Shell   |
| `qa/`                        | E2E tests (Go), golden regression shell test, fixtures (baseline/regression HTML, golden JSON, test ZIP)                                       | Go, Shell   |
| `spec/`                      | Formal CI/CD specifications with requirements matrices and quality gates                                                                       | Markdown    |
| `docs/`                      | Architecture, evaluator guide, project mode docs, CLI reference, configuration reference                                                       | Markdown    |

### Go Workspace

`go.work` coordinates **21 Go modules** with clean boundaries:

```
go.work
├── clients/cli/
├── services/platform-api/
├── services/orchestrator/
├── services/archive-extractor/
├── devtools/ops/job-status-cli/
├── devtools/qa/suite-runner/
├── qa/e2e/
└── libs/go/
    ├── bootstrap/
    ├── config/
    ├── diff/
    ├── domain/job/
    ├── events/
    ├── httputil/
    ├── logging/
    ├── messaging/
    ├── models/
    ├── scannercatalog/
    ├── scannerregistry/
    └── storage/
```

---

## Testing Strategy

StageFlow uses a **multi-layered testing strategy** spanning unit, integration, E2E, golden regression, and CI quality gates.

### Test Pyramid

```
                    ┌─────────────┐
                    │   Golden    │  Full pipeline: scan → baseline → diff
                    │  Regression │  (qa/e2e/project-scan-golden.sh)
                    ├─────────────┤
                    │    E2E      │  Live API URL/ZIP scans (RUN_E2E=1)
                    ├─────────────┤
                    │ Integration │  Orchestrator E2E with mock Podman
                    ├─────────────┤
                    │    Unit     │  Go race tests, Vitest, Storybook
                    └─────────────┘
```

### Test Frameworks by Language

| Language       | Framework                          | Where Used                            |
| -------------- | ---------------------------------- | ------------------------------------- |
| **Go**         | `testing` (stdlib) + `-race` flag  | All Go packages, qa/e2e, devtools     |
| **TypeScript** | Vitest with `v8` coverage          | services/scanner-runner/tests/        |
| **TypeScript** | Playwright + Storybook test-runner | clients/web/                          |
| **Shell**      | Bash with `set -Eeuo pipefail`     | devtools/scripts/tests/, qa/e2e/\*.sh |

### Key Test Types

**Go Race Tests** — Run across all 21 workspace modules with `-race` flag, `golangci-lint` (30+ linters), and `govulncheck`.

**E2E Acceptance Tests** (`qa/e2e/`) — Double-gated behind `RUN_E2E` env var and `testing.Short()`:

- `TestE2E_URLScan` — Submits URL scan, validates `UnifiedReportV2` structure, summary consistency, per-scanner results
- `TestE2E_ZipScan` — Uploads test archive, verifies report and artifact endpoints (including MinIO signed URL rewriting)

**Golden Regression** (`qa/e2e/project-scan-golden.sh`) — 316-line script implementing complete baseline-then-regression flow:

1. Create project → baseline scan (0 issues) → promote baseline
2. Update URL to regression fixture (intentional `image-alt` violation)
3. Regression scan (expect exit 1)
4. Normalize output (strip timestamps, IDs, durations) → `diff -u` against golden fixtures
5. Structural assertions: 1 new issue, `ruleId=image-alt`, `severity=critical`, 0 fixed

**Suite Runner** (`devtools/qa/suite-runner/`) — Multi-domain accessibility regression testing:

- Loads YAML suite definition (domains, modules, thresholds)
- Submits scans, streams SSE with auto-reconnect
- Evaluates against violation thresholds (`max_critical`, `max_serious`, `max_total`)
- Prints tabular PASS/FAIL summary

**Storybook Tests** — Playwright interaction tests + axe-based accessibility tests for the web UI.

### CI/CD Pipeline

Three GitHub Actions workflows:

| Workflow                                                          | Trigger                 | Key Jobs                                                                                                                   |
| ----------------------------------------------------------------- | ----------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| **CI** (`.github/workflows/ci.yml`)                               | Push/PR to `main`       | workflow_lint, secrets (gitleaks), Go (build+lint+race tests+govulncheck), web CI, Storybook, scanner-runner, image builds |
| **Golden Regression** (`.github/workflows/golden-regression.yml`) | Manual + daily 6 AM UTC | Full stack bootstrap → golden test → teardown                                                                              |
| **CLI Release** (`.github/workflows/release-stageflow-cli.yml`)   | Tags `clients/cli/v*`   | Matrix build: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64                                          |

### Local Quality Gate

```bash
just ci    # Runs the full local CI pipeline:
           #   Stale vocabulary check (naming drift guard)
           #   Go build, lint, test (-race), vulncheck
           #   CLI docs regression (git diff --exit-code)
           #   Shell regression tests
           #   Frontend CI, Storybook tests, audit
           #   Scanner-runner CI, audit
```

---

## Configuration

### Environment Files

| File                         | Purpose                                                      |
| ---------------------------- | ------------------------------------------------------------ |
| `.env.example`               | Local development baseline with `change-me` placeholders     |
| `.env`                       | Production/local env (gitignored, contains real credentials) |
| `.env.staging`               | Staging environment (gitignored)                             |
| `infra/.env.staging.example` | Staging deployment template                                  |

### Configuration Loading

Go services use `libs/go/config/` for type-safe environment variable parsing:

- `GetEnv(key, default)` — String with fallback
- `GetEnvInt(key, default)` — Int with fallback on parse failure
- `GetEnvBool(key, default)` — Strict allowlist: `1/true/TRUE/True` = true
- `GetEnvDuration(key, default)` — `time.ParseDuration` format (e.g., "5m", "30s")
- **Credential aliasing**: `MINIO_ACCESS_KEY` falls back to `MINIO_ROOT_USER`, same for secret key

### Compose Overlays

| Overlay                      | Purpose                             | Key Differences                                                          |
| ---------------------------- | ----------------------------------- | ------------------------------------------------------------------------ |
| `podman-compose.yml` (base)  | All services, no host ports exposed | `${VAR:?error}` for required env vars                                    |
| `podman-compose.local.yml`   | Local dev                           | Binds ports to localhost, enables private targets, `POD_NETNS_MODE=host` |
| `podman-compose.test.yml`    | Test overlay                        | Binds ports for testing, enables orchestrator port                       |
| `podman-compose.staging.yml` | Staging                             | Different port ranges (8300, 9300, 3301) to coexist with local dev       |

### Environment Comparison

| Aspect          | Local Dev       | Staging                 |
| --------------- | --------------- | ----------------------- |
| Domain          | `localhost`     | `staging.stageflow.org` |
| Frontend port   | 3010            | 3300                    |
| API port        | 8080            | 8300                    |
| MinIO ports     | 9000, 9001      | 9300, 9301              |
| Private targets | Enabled         | Disabled                |
| Pod netns mode  | `host`          | `bridge`                |
| Network         | `stageflow_net` | `stageflow_staging_net` |

---

## Screenshots

### Scanner Selection

![Eight scanners — Axe, Lighthouse, SEO, Security Headers, Link Checker, AI Navigator, Open Graph, Spelling & Grammar](docs/images/landing-scanners.png)

### Live Scan Execution

<table>
  <tr>
    <td><img src="docs/images/scan-in-progress.png" alt="Live scan in progress — SCANNING state with progress pipeline and scanner activity" /></td>
    <td><img src="docs/images/scan-complete.png" alt="Scan complete — all pipeline stages green with report and artifact links" /></td>
  </tr>
  <tr>
    <td align="center"><em>Live progress stream during execution</em></td>
    <td align="center"><em>Completed scan with report and artifact links</em></td>
  </tr>
</table>

### Unified Report

<table>
  <tr>
    <td><img src="docs/images/report-overview.png" alt="Report overview — risk snapshot, severity breakdown, Lighthouse scores, scanner status" /></td>
    <td><img src="docs/images/report-issues.png" alt="Issues tab — grouped findings with severity and remediation" /></td>
  </tr>
  <tr>
    <td align="center"><em>Overview with severity, scores, and scanner status</em></td>
    <td align="center"><em>Issues tab with grouped findings and remediation detail</em></td>
  </tr>
</table>

### Page-Level Evidence

<table>
  <tr>
    <td><img src="docs/images/report-pages.png" alt="Pages tab — annotated screenshot with bounding box overlays" /></td>
    <td><img src="docs/images/report-pages-detail.png" alt="Issue detail — evidence crop, selector, and fix guidance" /></td>
  </tr>
  <tr>
    <td align="center"><em>Annotated screenshots for page-level evidence</em></td>
    <td align="center"><em>Issue detail with evidence crop and fix guidance</em></td>
  </tr>
</table>

### CLI

<table>
  <tr>
    <td><img src="docs/images/cli-help.png" alt="stageflow --help output" /></td>
    <td><img src="docs/images/cli-scanners.png" alt="stageflow scanners output" /></td>
  </tr>
  <tr>
    <td align="center"><em>CLI command surface</em></td>
    <td align="center"><em>Scanner discovery from the terminal</em></td>
  </tr>
</table>

---

## For Reviewers

StageFlow is a solo-built portfolio project designed to be runnable as open source while still making the engineering depth easy to evaluate.

- **Distributed system design** — multi-service Go/TypeScript architecture coordinated through NATS JetStream, with explicit service boundaries, a documented job FSM, and Podman pod isolation
- **Contract-driven development** — JSON Schema as the single source of truth for the report format, with generated TypeScript and Go types used by all consumers
- **Testing at every layer** — Go race tests and `golangci-lint` across modules; Vitest unit tests; Storybook interaction and axe-based accessibility tests; orchestrator E2E with a mock Podman adapter; and a golden shell test for the full project scan → baseline → diff pipeline
- **Developer experience** — a Go CLI with streaming SSE, Project Mode, JSON output, and `--fail-on` severity gating; a `just`-based task runner; pre-commit-based repo hooks; and generated CLI reference docs that stay in sync with the code
- **Security and operational discipline** — SSRF guardrails, archive extraction limits, API key middleware, `govulncheck` in CI, and clear separation of credentials per environment

See the [Evaluator guide](docs/evaluators-guide.md) for a structured path through the codebase aimed at reviewers and hiring managers.

---

## Docs

| Document                                                   | Description                                                                         |
| ---------------------------------------------------------- | ----------------------------------------------------------------------------------- |
| [Architecture deep-dive](docs/architecture/system.md)      | Full system design: trust boundaries, data flows, failure modes, code reference map |
| [Configuration reference](docs/reference/configuration.md) | All environment variables, compose overlays, scanner config                         |
| [CLI README](clients/cli/README.md)                        | Full CLI command reference                                                          |
| [Project Mode](docs/PROJECT_MODE.md)                       | Local dev server lifecycle management                                               |
| [Evaluator guide](docs/evaluators-guide.md)                | Structured path through the codebase for reviewers                                  |
| [Devtools](docs/operations/devtools.md)                    | Internal ops and QA tooling                                                         |
| [CLI cheatsheet](docs/operations/cli_cheatsheet.md)        | Quick reference for common CLI operations                                           |

---

## Support

- Use the repository issue templates for bug reports and setup questions
- Use [SECURITY.md](SECURITY.md) for private vulnerability reporting

---

## License

[MIT](LICENSE)
