# StageFlow Architecture

This document explains StageFlow job flow, trust boundaries, and service responsibilities.

If you are still orienting yourself, start with the [repository README](../../README.md) for the product overview and fastest local setup path.

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

- `services/platform-api`: intake, validation, status APIs, SSE.
- `services/orchestrator`: FSM, container lifecycle, aggregation.
- `services/archive-extractor`: secure archive extraction and provenance generation.
- `services/scanner-runner`: plugin discovery and scanner execution runtime.
- `clients/web`: submission UX and live status/report views.
- `clients/cli`: Go CLI — scan submission, SSE streaming, report rendering, project mode.
- `libs/contracts`: JSON Schemas and generated contracts.

---

## Core Services and Responsibilities

### Platform API (`services/platform-api`)

Responsibilities:

- Accept URL and ZIP job submissions.
- Validate request payloads and enforce intake limits.
- Apply SSRF protections for URL jobs.
- Publish job creation events.
- Serve REST status and SSE updates.
- Manage project entities (CRUD, baseline promotion, job-to-project mapping).
- Compute on-demand diffs between a project's baseline and current scan.

Important entry points:

- Router: `services/platform-api/internal/api/router.go`
- URL intake: `services/platform-api/internal/api/handlers_jobs_url_submit.go`
- ZIP intake: `services/platform-api/internal/api/handlers_jobs_zip_upload.go`
- SSE: `services/platform-api/internal/api/handlers_sse.go`
- SSRF validation: `services/platform-api/internal/api/security.go`
- Project CRUD: `services/platform-api/internal/api/handlers_projects.go`
- Project store: `services/platform-api/internal/project/store.go`
- Diff engine: `services/platform-api/internal/api/handlers_diff.go`

### Orchestrator (`services/orchestrator`)

Responsibilities:

- Consume lifecycle events from NATS.
- Drive jobs through legal state transitions.
- Create/manage Podman pods and scanner containers.
- Coordinate extraction for ZIP jobs.
- Aggregate scanner outputs into a unified report.
- Persist job state and event audit trail.

Important entry points:

- Event handlers: `services/orchestrator/internal/orchestrator/events.go`
- Lifecycle workflows: `services/orchestrator/internal/application/jobs`
- Domain policies: `services/orchestrator/internal/domain/jobs`
- Runtime and persistence adapters: `services/orchestrator/internal/adapters`
- Aggregation: `services/orchestrator/internal/adapters/storage`
- Rule deduplication: `services/orchestrator/internal/adapters/storage/rule_deduplication.go`

### Extractor (`services/archive-extractor`)

Responsibilities:

- Download submitted ZIP from staging storage.
- Validate archive safety constraints.
- Extract safely into workspace volume.
- Generate provenance document for scanning.
- Publish extraction completion/failure events.

Important entry point:

- `services/archive-extractor/internal/extractor/extractor.go`

### Scanner Runner (`services/scanner-runner`)

Responsibilities:

- Discover scanner plugins by manifest.
- Validate scanner identity and scanner config schema.
- Run scanner lifecycle over input pages.
- Emit scanner completion/failure events.
- Upload scanner artifacts.

Important entry points:

- Runtime worker: `services/scanner-runner/src/worker.ts`
- Base lifecycle: `services/scanner-runner/src/core/scanner-base.ts`
- Plugin discovery/loader: `services/scanner-runner/src/core/plugins`

### Web App (`clients/web`)

Responsibilities:

- Submit jobs and scanner selections.
- Render job progress and report outputs.
- Subscribe to job SSE stream with reconnect behavior.
- Build to static assets that are served by the frontend container's Caddy runtime.

Important entry points:

- API client: `clients/web/src/lib/api/client.ts`
- Scan status store: `clients/web/src/lib/stores/scan-status.svelte.ts`

### CLI (`clients/cli`)

Responsibilities:

- Submit URL scan jobs via the Platform API.
- Stream live job progress over SSE with reconnect.
- Fetch and render unified reports in text, markdown, and JSON formats.
- Filter, sort, and truncate issues client-side (by severity, category, max count).
- Enforce severity-based exit codes for CI/automation gating (`--fail-on`).
- Manage local Project Mode lifecycle: start dev server, poll readiness, submit scan, stop server.
- Manage remote projects: create, list, show, update, delete, promote baseline.
- Scan against remote projects with automatic baseline diffing (`--project`).
- Discover available scanners from the API.

Important entry points:

- Command surface: `clients/cli/cobra_scan.go`, `cobra_project.go`, `cobra_report.go`
- Remote project commands: `clients/cli/cobra_project_remote.go`, `cobra_project_update.go`
- Remote project API client: `clients/cli/client_projects.go`
- Diff rendering: `clients/cli/cobra_diff.go`
- SSE streaming: `clients/cli/sse.go`
- Report rendering: `clients/cli/report_output.go`, `report_output_markdown.go`
- Issue filtering/sorting: `clients/cli/filter.go`, `report_output.go:selectIssues`
- Project config: `clients/cli/project_config.go`
- Dev server lifecycle: `clients/cli/dev_stack.go`

---

## CLI Report Contract

The CLI wraps the unified report in a versioned envelope (`stageflow-cli/report@v1`) designed for both human review and machine consumption.

### Envelope Structure

```text
reportEnvelope
├── schema          "stageflow-cli/report@v1"
├── cli             { version, commit, date }
├── api             { base_url }
├── job             { id, state, created_at, updated_at }
├── links           { job, results }          — API URLs for the raw job and results
├── urls            [scanned URLs]
├── filters         { max_issues, issues_returned, issues_total, truncated, sort }
└── report          UnifiedReportV2
    ├── summary     { score, scoreGrade, totalIssues, bySeverity, byScanner }
    ├── issues[]    IssueDetail (sorted by severity desc, scanner asc, rule asc)
    ├── scanners[]  per-scanner status, timing, severity counts
    ├── pages[]     per-page issue counts and timing
    └── meta        { jobId, scannedAt, completedAt, durationMs }
```

### Issue Identity

Each issue has an `id` field that is a content-based hash derived from the rule, page, and element context. The same violation on the same page produces the same `id` across runs, making it suitable for diffing scan results to detect regressions.

### Issue Detail Fields

Every issue carries enough context for automated remediation:

| Field | Purpose |
| --- | --- |
| `id` | Stable content hash — usable for deduplication and diffing |
| `ruleId` | Scanner-specific rule identifier (e.g., `landmark-one-main`) |
| `scanner` | Which scanner produced the finding |
| `severity` | Normalized: `critical`, `serious`, `moderate`, `minor`, `info` |
| `title` | Human-readable summary |
| `description` | What the rule checks |
| `howToFix` | Remediation guidance |
| `wcagTags` | WCAG references (axe issues) |
| `helpUrl` | Link to full rule documentation |
| `occurrences[].selector` | CSS selector to locate the element |
| `occurrences[].html` | HTML snippet of the violating element |
| `occurrences[].target` | DOM target path array |
| `occurrences[].contextHtml` | Surrounding HTML for context |
| `scannerData` | Scanner-specific structured data (e.g., SEO word counts, missing OG tags) |

### Exit Code Contract

| Exit code | Meaning |
| --- | --- |
| 0 | Scan completed successfully, no issues at or above `--fail-on` threshold |
| 1 | Scan completed but issues meet or exceed `--fail-on` severity threshold |
| 2 | CLI or API error (network failure, invalid arguments, malformed response) |

### Output Formats

| Format | Flag | Use case |
| --- | --- | --- |
| Text | `--format text` (default) | Human review in terminal |
| Markdown | `--format markdown` | Structured sections with headings, suitable for PR comments or agent parsing |
| JSON | `--format json` | Machine consumption — full envelope with all metadata |

### Filtering Flags

| Flag | Effect |
| --- | --- |
| `--fail-on <severity>` | Exit 1 if any displayed issue is at or above this severity |
| `--severity <csv>` | Only display issues matching these severities |
| `--category <csv>` | Only display issues matching these categories |
| `--max-issues <n>` | Cap returned issues (default 200, 0 = unlimited) |
| `--summary-only` | Print summary counts only, skip individual findings |
| `--group-by <mode>` | Group findings by `category`, `scanner`, or `none` |

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

For self-hosting, this outer boundary is represented by `infra/caddy/Caddyfile`. It is separate from the frontend container, which also uses Caddy internally to serve the static web build on port `3000`.

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

Transition and outcome policy location: `services/orchestrator/internal/domain/jobs`.

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

### CLI Scan Flow

```text
stageflow scan <url> --scanners axe,seo --format json --fail-on serious
  -> POST /api/v1/jobs/urls { urls, modules }
  -> CLI opens SSE stream /api/v1/jobs/{id}/stream
  -> Prints scanner progress to stderr as events arrive
  -> On terminal state (DONE/FAILED):
       -> GET /api/v1/jobs/{id}/results
       -> Filter, sort, truncate issues client-side
       -> Render report envelope to stdout
       -> Check --fail-on threshold: exit 0 (pass) or 1 (fail)
```

### CLI Project Mode Flow

```text
stageflow project [path]
  -> Read .stageflow/config.yaml
  -> Start dev server (dev.start.cmd)
  -> Poll dev.ready.url until HTTP 2xx/3xx
  -> Submit scan job to configured API
  -> Stream progress and render report (same as scan flow)
  -> Send interrupt to dev server, wait for graceful shutdown
```

### Remote Project Scan Flow

```text
stageflow scan --project my-app --format json
  -> GET /api/v1/projects/my-app  (resolve URLs, scanners, baseline job ID)
  -> POST /api/v1/jobs/urls { urls, modules, project_slug }
  -> CLI opens SSE stream /api/v1/jobs/{id}/stream
  -> On terminal state (DONE):
       -> GET /api/v1/jobs/{id}/results  (scan report)
       -> Render report envelope to stdout
       -> If project has a baseline:
            -> GET /api/v1/projects/my-app/diff?job_id={id}
            -> Render diff envelope to stdout (separated by blank line)
            -> Exit 1 if regressions detected (newIssues > 0)
```

### Baseline Diff

The diff endpoint compares two scan reports issue-by-issue using the stable content-hash `id` on each issue. It returns:

- `delta` — counts: `newIssues`, `fixedIssues`, `unchangedIssues`
- `new[]` — issues in the current scan not present in the baseline
- `fixed[]` — issues in the baseline not present in the current scan
- `baseline` / `current` — metadata (job ID, scanned-at timestamp)

The diff is computed on-demand by the Platform API, not stored. The project store tracks only which job ID is the current baseline.

Project store: SQLite at `./projects.db` in the platform-api working directory. Schema covers projects, project-to-job mappings, scanner lists, and baseline pointers. Tests in `services/platform-api/internal/project/store_test.go`.

### SSE Progress Flow

```text
Client opens /api/v1/jobs/{id}/stream
  -> API sends current status immediately
  -> API pushes updates on job state/page progress changes
  -> Terminal state emits done event
```

SSE handler: `services/platform-api/internal/api/handlers_sse.go`.

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

Shared event type definitions: `libs/go/events/types.go`.

---

## Scanner Plugin System

Scanners are runtime-discovered plugins with manifest-defined capabilities and validation.

### Discovery Order

1. Built-ins in `services/scanner-runner/src/scanners`
2. Mounted `/plugins`
3. `~/.stageflow/plugins`
4. Additional `PLUGIN_PATHS`

### Plugin Contract

Each plugin supplies:

- `manifest.json` matching scanner-manifest schema.
- Executable module entrypoint.
- Scanner implementation compatible with runner lifecycle.

Manifest schema references:

- `libs/contracts/scanner-manifest/schema/scanner-manifest.schema.json`
- `libs/contracts/scanner-manifest/schema/README.md`

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

All modules live under `services/scanner-runner/src/scanners/ai-navigator/`.

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

- **API validation** (`services/platform-api/internal/api/scanner_configs.go`): Validates AI Navigator config on job submission. Enforces that `goal.objective` is set and `vision.model` is specified (or falls back to `AI_NAVIGATOR_DEFAULT_MODEL`).
- **Orchestrator** (`services/orchestrator/internal/application/jobs/scanner_launch_planner.go`): Injects `OPENROUTER_API_KEY`, `OPENROUTER_APP_TITLE`, and `OPENROUTER_APP_REFERER` into scanner container environment. API key is restricted to environment variables only and cannot be set in scanner options.

#### Web App

- **`PlaygroundAiConfig.svelte`**: Goal objective input, model dropdown, input values form (key-value pairs for form fills), advanced settings (max steps, timeout, success criteria).
- **`playground-utils.ts`**: Builds scanner config JSON from form state.

#### Environment Variables

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `OPENROUTER_API_KEY` | Yes (if AI Navigator used) | — | OpenRouter API authentication |
| `OPENROUTER_APP_TITLE` | No | StageFlow | Request attribution metadata |
| `OPENROUTER_APP_REFERER` | No | — | Request referer tracking |
| `AI_NAVIGATOR_DEFAULT_MODEL` | No | `openai/gpt-4o-mini` | Backend fallback model |
| `VITE_AI_NAVIGATOR_DEFAULT_MODEL` | No | `openai/gpt-4o-mini` | Web App default model selection |

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

- `services/orchestrator/internal/adapters/storage/rule_deduplication.go`

Report schema source:

- `libs/contracts/report/schema/unified-report.v2.schema.json`

---

## Storage Model

### PostgreSQL

Used by orchestrator for durable state and event history.

Primary persisted entities:

- Jobs (state, timing, scanner progress, completion details).
- Job events (audit timeline and processing metadata).

Schema source:

- `services/orchestrator/internal/adapters/repository/schema.sql`

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

Use the shared root control plane (set `STAGEFLOW_PROD_DEPLOY_DIR` to your deployment workspace root) for live
production operations:

```bash
cd $STAGEFLOW_PROD_DEPLOY_DIR
just deploy stageflow
just restart stageflow
just logs stageflow
just health
```

The repo-local `just prod ...` and `just deploy ...` commands intentionally
stop and point back to that shared root control plane.

### Build and Quality Gates

- `just build`: service and web app builds.
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

- Scanner Runner crash or timeout.
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

- `clients/cli`: submit scans, render reports, enforce severity gates via `--fail-on`.
- `devtools/ops/job-status-cli`: inspect jobs/events/pods/status.
- `devtools/qa/suite-runner`: run threshold-based multi-domain validation.

Tooling docs: `docs/operations/devtools.md`.

### Recommended debug path

1. Check job terminal state and latest event.
2. Inspect orchestrator timeline for stage boundary failure.
3. Pull scanner artifacts and verify payload validity.
4. Correlate service logs with event timestamps.

---

## End-to-End Testing

### Golden E2E Test (`qa/e2e/project-scan-golden.sh`)

The golden test exercises the full project scan → diff pipeline against a live stack:

1. Create a project pointing at a clean HTML fixture (`/qa/baseline.html`, 0 axe violations).
2. Run a scan with `--scanner axe`.
3. Promote the scan as the project baseline.
4. Update the project URL to a regression fixture (`/qa/regression.html`, 1 `image-alt` violation).
5. Rescan — expect exit code 1 (regressions detected).
6. Normalize non-deterministic fields (job IDs, timestamps, durations, paths) via jq.
7. Compare normalized JSON against committed golden files with `diff -u`.
8. Assert structural properties: 1 new issue, `ruleId=image-alt`, `severity=critical`, 0 fixed.
9. Clean up the project.

Fixture pages are static HTML served from `clients/web/static/qa/`. Golden files live in `qa/fixtures/project-golden/` and are auto-created on first run.

The test requires `stageflow`, `jq`, `python3`, and `curl` on PATH, plus a reachable API at `STAGEFLOW_API_URL` (defaults to `http://localhost:8080`) and fixture pages at `STAGEFLOW_FIXTURE_BASE_URL` (defaults to `http://localhost:3010` when the API is local). The intended repo-local path is `just setup && just images && just dev up local && just dev init local`.

---

## Code Reference Map

| Area | File |
| --- | --- |
| API route registration | `services/platform-api/internal/api/router.go` |
| URL intake validation | `services/platform-api/internal/api/handlers_jobs_url_submit.go` |
| ZIP intake handler | `services/platform-api/internal/api/handlers_jobs_zip_upload.go` |
| SSRF checks | `services/platform-api/internal/api/security.go` |
| SSE stream handler | `services/platform-api/internal/api/handlers_sse.go` |
| Orchestrator events | `services/orchestrator/internal/orchestrator/events.go` |
| Lifecycle workflows | `services/orchestrator/internal/application/jobs` |
| Domain policies | `services/orchestrator/internal/domain/jobs` |
| Repository adapter | `services/orchestrator/internal/adapters/repository` |
| Runtime adapter | `services/orchestrator/internal/adapters/runtime` |
| Messaging adapter | `services/orchestrator/internal/adapters/messaging` |
| Report aggregation | `services/orchestrator/internal/adapters/storage` |
| Rule deduplication | `services/orchestrator/internal/adapters/storage/rule_deduplication.go` |
| Extractor safety logic | `services/archive-extractor/internal/extractor/extractor.go` |
| Scanner runner entry | `services/scanner-runner/src/worker.ts` |
| Scanner base lifecycle | `services/scanner-runner/src/core/scanner-base.ts` |
| AI Navigator agent loop | `services/scanner-runner/src/scanners/ai-navigator/agent.ts` |
| AI Navigator vision client | `services/scanner-runner/src/scanners/ai-navigator/vision-client.ts` |
| AI Navigator page analyzer | `services/scanner-runner/src/scanners/ai-navigator/page-analyzer.ts` |
| AI Navigator action decider | `services/scanner-runner/src/scanners/ai-navigator/action-decider.ts` |
| AI Navigator config validation | `services/platform-api/internal/api/scanner_configs.go` |
| Scanner manifests | `libs/go/scannercatalog/manifests/*/manifest.json` |
| Event type contracts | `libs/go/events/types.go` |
| Report schema | `libs/contracts/report/schema/unified-report.v2.schema.json` |
| Scanner manifest schema | `libs/contracts/scanner-manifest/schema/scanner-manifest.schema.json` |
| Project CRUD handlers | `services/platform-api/internal/api/handlers_projects.go` |
| Project store (SQLite) | `services/platform-api/internal/project/store.go` |
| Project store tests | `services/platform-api/internal/project/store_test.go` |
| Diff endpoint | `services/platform-api/internal/api/handlers_diff.go` |
| CLI scan command | `clients/cli/cobra_scan.go` |
| CLI local project mode | `clients/cli/cobra_project.go` |
| CLI remote project commands | `clients/cli/cobra_project_remote.go` |
| CLI project update | `clients/cli/cobra_project_update.go` |
| CLI project API client | `clients/cli/client_projects.go` |
| CLI diff rendering | `clients/cli/cobra_diff.go` |
| CLI report rendering | `clients/cli/report_output.go` |
| CLI SSE streaming | `clients/cli/sse.go` |
| CLI issue filtering | `clients/cli/filter.go` |
| CLI dev server lifecycle | `clients/cli/dev_stack.go` |
| CLI project config | `clients/cli/project_config.go` |
| CLI types (API contracts) | `clients/cli/types.go` |
| Golden E2E test | `qa/e2e/project-scan-golden.sh` |
| QA fixture pages | `clients/web/static/qa/` |
| Golden test fixtures | `qa/fixtures/project-golden/` |
| Just command surface | `justfile` |
