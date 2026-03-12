# StageFlow Architecture

This document explains StageFlow job flow, trust boundaries, and service responsibilities.

## Table of Contents

1. [System Goals](#system-goals)
2. [Platform Shape](#platform-shape)
3. [Core Services and Responsibilities](#core-services-and-responsibilities)
4. [Security and Trust Boundaries](#security-and-trust-boundaries)
5. [Job Lifecycle and State Machine](#job-lifecycle-and-state-machine)
6. [Data Flows](#data-flows)
7. [Eventing Model (NATS JetStream)](#eventing-model-nats-jetstream)
8. [Scanner Plugin System](#scanner-plugin-system)
9. [Unified Report Aggregation](#unified-report-aggregation)
10. [Storage Model](#storage-model)
11. [Runtime Operations and Deployment](#runtime-operations-and-deployment)
12. [Failure Modes and Recovery](#failure-modes-and-recovery)
13. [Observability and Debugging](#observability-and-debugging)
14. [Code Reference Map](#code-reference-map)

---

## System Goals

StageFlow is designed around four goals:

1. **Safe intake**: Treat all submitted URLs and archives as untrusted.
2. **Isolation**: Run extraction and scanners in scoped job containers.
3. **Deterministic orchestration**: Drive jobs through an explicit state machine.
4. **Actionable output**: Normalize multiple scanner outputs into one report contract.

Non-goals:

- It is not a hosted multi-tenant control plane by default.
- It is not a replacement for edge rate limiting or perimeter security controls.

---

## Platform Shape

```text
Client (UI/API)
  -> Platform API (Go)
    -> NATS JetStream
      -> Orchestrator (Go)
        -> Podman job pod
             |- Extractor (Go, ZIP jobs only)
             `- Scanner runner containers (TS/Bun/Playwright)

Artifacts/status
  -> MinIO (objects) + PostgreSQL (job state/events)
  -> API job endpoints + SSE stream
```

Primary repository areas:

- `platform/api`: intake, validation, status APIs, SSE.
- `platform/orchestrator`: FSM, container lifecycle, aggregation.
- `platform/extractor`: secure archive extraction and provenance generation.
- `platform/scanner-runner`: plugin discovery and scanner execution runtime.
- `frontend`: submission UX and live status/report views.
- `packages/contracts`: JSON Schemas and generated contracts.

---

## Core Services and Responsibilities

### Platform API (`platform/api`)

Responsibilities:

- Accept URL and ZIP job submissions.
- Validate request payloads and enforce intake limits.
- Apply SSRF protections for URL jobs.
- Publish job creation events.
- Serve REST status and SSE updates.

Important entry points:

- Router: `platform/api/internal/api/router.go`
- URL intake: `platform/api/internal/api/handlers_jobs_url_submit.go`
- ZIP intake: `platform/api/internal/api/handlers_jobs_zip_upload.go`
- SSE: `platform/api/internal/api/handlers_sse.go`
- SSRF validation: `platform/api/internal/api/security.go`

### Orchestrator (`platform/orchestrator`)

Responsibilities:

- Consume lifecycle events from NATS.
- Drive jobs through legal state transitions.
- Create/manage Podman pods and scanner containers.
- Coordinate extraction for ZIP jobs.
- Aggregate scanner outputs into a unified report.
- Persist job state and event audit trail.

Important entry points:

- Event handlers: `platform/orchestrator/internal/orchestrator/events.go`
- Lifecycle workflows: `platform/orchestrator/internal/application/jobs`
- Domain policies: `platform/orchestrator/internal/domain/jobs`
- Runtime and persistence adapters: `platform/orchestrator/internal/adapters`
- Aggregation: `platform/orchestrator/internal/adapters/storage`
- Rule deduplication: `platform/orchestrator/internal/adapters/storage/rule_deduplication.go`

### Extractor (`platform/extractor`)

Responsibilities:

- Download submitted ZIP from staging storage.
- Validate archive safety constraints.
- Extract safely into workspace volume.
- Generate provenance document for scanning.
- Publish extraction completion/failure events.

Important entry point:

- `platform/extractor/internal/extractor/extractor.go`

### Scanner Runner (`platform/scanner-runner`)

Responsibilities:

- Discover scanner plugins by manifest.
- Validate scanner identity and scanner config schema.
- Run scanner lifecycle over input pages.
- Emit scanner completion/failure events.
- Upload scanner artifacts.

Important entry points:

- Runtime worker: `platform/scanner-runner/src/worker.ts`
- Base lifecycle: `platform/scanner-runner/src/core/scanner-base.ts`
- Plugin discovery/loader: `platform/scanner-runner/src/core/plugins`

### Frontend (`frontend`)

Responsibilities:

- Submit jobs and scanner selections.
- Render job progress and report outputs.
- Subscribe to job SSE stream with reconnect behavior.

Important entry points:

- API client: `frontend/src/lib/api/client.ts`
- Scan status store: `frontend/src/lib/stores/scan-status.svelte.ts`

---

## Security and Trust Boundaries

### Boundary 1: Untrusted submission -> API

Submitted URLs and archives are untrusted input.

Controls include:

- Request size and shape limits.
- URL count and URL format validation.
- Explicit scheme checks (`http` / `https`).
- Hostname resolution and blocked IP range checks.

### Boundary 2: ZIP archive -> extractor workspace

Extractor enforces archive safety before scanner runtime sees extracted content.

Controls include:

- Entry-count limits.
- Per-entry and total size constraints.
- Compression-ratio checks (ZIP bomb defense).
- Path sanitation and traversal prevention.

### Boundary 3: scanner runtime -> shared infrastructure

Scanners run in orchestrator-managed containers with bounded resources and scoped volumes.

Controls include:

- Per-job pod boundaries.
- Scanner identity validation against manifest.
- `SCANNER_OPTIONS` schema validation.
- Artifact upload boundary through storage interfaces.

### Boundary 4: edge proxy -> public access

Rate limiting, WAF policy, and perimeter controls are deployment concerns at edge/proxy/CDN layers.

---

## Job Lifecycle and State Machine

StageFlow uses a strict FSM to keep transitions explicit and debuggable.

```text
PENDING
  -> EXTRACTING (ZIP jobs)
  -> READY_TO_SCAN
  -> SCANNING
  -> COMPLETING
  -> DONE

Any state -> FAILED (terminal)
```

State intent:

- `PENDING`: accepted and queued.
- `EXTRACTING`: ZIP processing in extractor container.
- `READY_TO_SCAN`: scanner inputs are ready.
- `SCANNING`: one or more scanner containers active.
- `COMPLETING`: output merge/dedup/report publish.
- `DONE`: successful terminal state.
- `FAILED`: terminal failure state.

Transition and outcome policy location: `platform/orchestrator/internal/domain/jobs`.

---

## Data Flows

### URL Job Flow

```text
Client -> API /api/v1/jobs/urls
  -> API validates + publishes job.created
  -> Orchestrator consumes event
  -> Orchestrator transitions READY_TO_SCAN -> SCANNING
  -> Scanner containers run and upload artifacts
  -> Orchestrator aggregates report -> DONE
  -> API serves status/SSE updates to client
```

### ZIP Job Flow

```text
Client -> API /api/v1/jobs/zip
  -> API stores upload in staging bucket + publishes job.created
  -> Orchestrator creates job pod and starts extractor
  -> Extractor validates/unpacks and emits extraction.ready
  -> Orchestrator starts scanner containers
  -> Scanner artifacts uploaded
  -> Orchestrator aggregates report -> DONE
  -> API serves status/SSE updates to client
```

### SSE Progress Flow

```text
Client opens /api/v1/jobs/{id}/stream
  -> API sends current status immediately
  -> API pushes updates on job state/page progress changes
  -> Terminal state emits done event
```

SSE handler: `platform/api/internal/api/handlers_sse.go`.

---

## Eventing Model (NATS JetStream)

Services communicate through event subjects and durable consumers.

Typical lifecycle subjects:

- Job lifecycle events (creation/completion/failure).
- Extraction readiness/failure events.
- Scanner progress/completion/failure events.

Operational properties:

- Durable stream persistence.
- Explicit acknowledgement behavior.
- Retry/redelivery when handlers fail.
- Decoupling between intake, orchestration, and scanner runtimes.

Shared event type definitions: `packages/shared-go/events/types.go`.

---

## Scanner Plugin System

Scanners are runtime-discovered plugins with manifest-defined capabilities and validation.

### Discovery Order

1. Built-ins in `platform/scanner-runner/src/scanners`
2. Mounted `/plugins`
3. `~/.stageflow/plugins`
4. Additional `PLUGIN_PATHS`

### Plugin Contract

Each plugin supplies:

- `manifest.json` matching scanner-manifest schema.
- Executable module entrypoint.
- Scanner implementation compatible with runner lifecycle.

Manifest schema references:

- `packages/contracts/scanner-manifest/schema/scanner-manifest.schema.json`
- `packages/contracts/scanner-manifest/schema/README.md`

### Runtime Validation

Before execution, runner validates:

- Scanner identity consistency (manifest vs runtime metadata).
- Scanner options (`SCANNER_OPTIONS`) against manifest `configSchema`.
- Plugin loadability and entrypoint compatibility.

### Built-In Modules

- `axe`
- `lighthouse`
- `seo`
- `security-headers`
- `link-checker`
- `ai-navigator`
- `open-graph` (Validates social sharing metadata and rich previews)
- `spelling-grammar` (AI-powered content quality and grammar checks)

### AI Navigator (`ai-navigator`)

The AI Navigator is a vision-model-powered browser automation agent that uses LLMs to understand and navigate web pages based on user-defined goals.

#### Provider

All AI requests route through **OpenRouter** (`https://openrouter.ai/api/v1/chat/completions`). No direct AI SDK is used — the scanner makes raw HTTP calls. The API key is injected into scanner containers via environment variables by the orchestrator (`scanner_launch_planner.go`).

#### Core Modules

| Module | File | Responsibility |
| --- | --- | --- |
| Vision Client | `vision-client.ts` | OpenRouter API communication, image compression (Sharp), semaphore concurrency control, exponential backoff retry |
| Page Analyzer | `page-analyzer.ts` | Extracts interactive elements, sends screenshot to vision model, returns page classification, description, suggested actions, and goal relevance score |
| Action Decider | `action-decider.ts` | Determines next browser action from goal + history + screenshot; returns action with reasoning and confidence |
| Goal Checker | `goal-checker.ts` | Evaluates success criteria: `url-contains`, `url-matches`, `element-visible`, `text-visible`, `custom` |
| Decision Prompt | `decision-prompt.ts` | Constructs system prompts with goal, step history, available elements, and input constraints |
| Action Parser | `action-decision-parser.ts` | Parses vision model JSON responses into executable actions (click, hover, scroll, keyboard, wait, fill, select, done, stuck) |
| Loop Detector | `loop-detector.ts` | Detects navigation loops by checking if the same URL appears 3+ times in the last 6 steps |
| Agent | `agent.ts` | Main execution loop: perception → decision → action → repeat; respects max steps, wall time, and token budgets; captures screenshots and generates trace output |
| Options | `options.ts` | Validates and parses agent configuration (goal, vision settings, constraints) |

All modules live under `platform/scanner-runner/src/scanners/ai-navigator/`.

#### Agent Execution Flow

```text
Browser loads target URL
  -> Loop (until goal met or limits exceeded):
       1. Screenshot current page
       2. Extract interactive elements (links, buttons, inputs, forms)
       3. Send screenshot to vision model (Page Analyzer)
          <- Returns: page type, description, suggested actions, goal relevance
       4. Send screenshot + goal + history to vision model (Action Decider)
          <- Returns: next action with reasoning and confidence
       5. Execute browser action (click, fill, scroll, etc.)
       6. Check success criteria (Goal Checker)
       7. Check for navigation loops (Loop Detector)
       8. Record step in trace
  -> Output: AgentResult with success status, step trace, and screenshots
```

#### Backend Integration

- **API validation** (`platform/api/internal/api/scanner_configs.go`): Validates AI Navigator config on job submission. Enforces that `goal.objective` is set and `vision.model` is specified (or falls back to `AI_NAVIGATOR_DEFAULT_MODEL`).
- **Orchestrator** (`platform/orchestrator/internal/application/jobs/scanner_launch_planner.go`): Injects `OPENROUTER_API_KEY`, `OPENROUTER_APP_TITLE`, and `OPENROUTER_APP_REFERER` into scanner container environment. API key is restricted to environment variables only and cannot be set in scanner options.

#### Frontend

- **`PlaygroundAiConfig.svelte`**: Goal objective input, model dropdown, input values form (key-value pairs for form fills), advanced settings (max steps, timeout, success criteria).
- **`playground-utils.ts`**: Builds scanner config JSON from form state.

#### Environment Variables

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `OPENROUTER_API_KEY` | Yes (if AI Navigator used) | — | OpenRouter API authentication |
| `OPENROUTER_APP_TITLE` | No | StageFlow | Request attribution metadata |
| `OPENROUTER_APP_REFERER` | No | — | Request referer tracking |
| `AI_NAVIGATOR_DEFAULT_MODEL` | No | `openai/gpt-4o-mini` | Backend fallback model |
| `VITE_AI_NAVIGATOR_DEFAULT_MODEL` | No | `openai/gpt-4o-mini` | Frontend default model selection |

---

## Unified Report Aggregation

The orchestrator merges scanner outputs into one unified contract.

Core steps:

1. Load scanner artifacts.
2. Merge pages and issue collections.
3. Normalize severities/categories.
4. Deduplicate equivalent findings across scanners.
5. Recalculate aggregate summaries and score fields.
6. Upload final report artifacts.

Deduplication logic source:

- `platform/orchestrator/internal/adapters/storage/rule_deduplication.go`

Report schema source:

- `packages/contracts/report/schema/unified-report.v2.schema.json`

---

## Storage Model

### PostgreSQL

Used by orchestrator for durable state and event history.

Primary persisted entities:

- Jobs (state, timing, scanner progress, completion details).
- Job events (audit timeline and processing metadata).

Schema source:

- `platform/orchestrator/internal/adapters/repository/schema.sql`

### MinIO

Used for ZIP staging and scan artifacts.

Common artifact classes:

- Uploaded input archives.
- Provenance files.
- Per-scanner result payloads.
- Final unified report output.

---

## Runtime Operations and Deployment

### Local and Staging (Compose)

Command surface:

- `just dev up|down|restart|logs|init`
- `just staging up|down|restart|logs|init|ps`

Compose definitions live under `infra/compose`.

### Production Deployment

This repo does not own standalone production deployment on the shared VPS.

Use the shared root control plane (configured via `STAGEFLOW_PROD_DEPLOY_DIR`, defaults to `/home/matt/Deployment`) for live
production operations:

```bash
cd ${STAGEFLOW_PROD_DEPLOY_DIR:-/home/matt/Deployment}
just stageflow-deploy
just stageflow-restart
just stageflow-logs
just stageflow-health
```

The repo-local `just prod ...` and `just deploy ...` commands intentionally
stop and point back to that shared root control plane.

### Build and Quality Gates

- `just build`: service and frontend builds.
- `just images`: container images.
- `just ci`: local CI pipeline (build/lint/test/typecheck/audit).

---

## Failure Modes and Recovery

### Intake failures

- Invalid URL format/scheme.
- SSRF policy violations.
- Payload size or shape violations.

Outcome: rejected at API boundary, no orchestration run.

### Extraction failures

- Malicious or malformed archive.
- Traversal/path safety violations.
- Archive limits exceeded.

Outcome: job transitions to `FAILED` with extraction-stage error context.

### Scanner failures

- Scanner runtime crash or timeout.
- Plugin load/validation mismatch.
- External page/runtime instability.

Outcome: job failure or partial scanner failure handling, depending on orchestration policy and stage.

### Aggregation/report failures

- Missing/invalid scanner artifact payload.
- Contract/merge incompatibilities.

Outcome: transition to `FAILED`; event timeline records failure reason.

---

## Observability and Debugging

### Primary operators' signals

- Job state transitions via API/SSE.
- Orchestrator event history.
- Scanner artifacts in object storage.
- Service logs (`just dev logs` or root prod logs via `STAGEFLOW_PROD_DEPLOY_DIR`).

### Useful tools

- `tools/stageflow-cli`: submit URL scan jobs and fetch unified reports.
- `tools/job-status-cli`: inspect jobs/events/pods/status.
- `tools/suite-runner`: run threshold-based multi-domain validation.

Tooling docs: `docs/TOOLS.md` and `tools/README.md`.

### Recommended debug path

1. Check job terminal state and latest event.
2. Inspect orchestrator timeline for stage boundary failure.
3. Pull scanner artifacts and verify payload validity.
4. Correlate service logs with event timestamps.

---

## Code Reference Map

| Area | File |
| --- | --- |
| API route registration | `platform/api/internal/api/router.go` |
| URL intake validation | `platform/api/internal/api/handlers_jobs_url_submit.go` |
| ZIP intake handler | `platform/api/internal/api/handlers_jobs_zip_upload.go` |
| SSRF checks | `platform/api/internal/api/security.go` |
| SSE stream handler | `platform/api/internal/api/handlers_sse.go` |
| Orchestrator events | `platform/orchestrator/internal/orchestrator/events.go` |
| Lifecycle workflows | `platform/orchestrator/internal/application/jobs` |
| Domain policies | `platform/orchestrator/internal/domain/jobs` |
| Repository adapter | `platform/orchestrator/internal/adapters/repository` |
| Runtime adapter | `platform/orchestrator/internal/adapters/runtime` |
| Messaging adapter | `platform/orchestrator/internal/adapters/messaging` |
| Report aggregation | `platform/orchestrator/internal/adapters/storage` |
| Rule deduplication | `platform/orchestrator/internal/adapters/storage/rule_deduplication.go` |
| Extractor safety logic | `platform/extractor/internal/extractor/extractor.go` |
| Scanner runner entry | `platform/scanner-runner/src/worker.ts` |
| Scanner base lifecycle | `platform/scanner-runner/src/core/scanner-base.ts` |
| AI Navigator agent loop | `platform/scanner-runner/src/scanners/ai-navigator/agent.ts` |
| AI Navigator vision client | `platform/scanner-runner/src/scanners/ai-navigator/vision-client.ts` |
| AI Navigator page analyzer | `platform/scanner-runner/src/scanners/ai-navigator/page-analyzer.ts` |
| AI Navigator action decider | `platform/scanner-runner/src/scanners/ai-navigator/action-decider.ts` |
| AI Navigator config validation | `platform/api/internal/api/scanner_configs.go` |
| Scanner manifests | `packages/shared-go/scannercatalog/manifests/*/manifest.json` |
| Event type contracts | `packages/shared-go/events/types.go` |
| Report schema | `packages/contracts/report/schema/unified-report.v2.schema.json` |
| Scanner manifest schema | `packages/contracts/scanner-manifest/schema/scanner-manifest.schema.json` |
| Just command surface | `justfile` |
