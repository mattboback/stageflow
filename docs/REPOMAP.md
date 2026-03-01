# StageFlow Repo Map

This map is a practical index for contributors working in the StageFlow monorepo.
It focuses on service boundaries, directory responsibilities, key file ownership,
API/event surfaces, and where to make changes safely.

Scope note: infra/deployment deep detail is intentionally minimized in this edition.

## Quick Index

- `platform/api` - public intake API, SSRF validation, SSE streaming, scanner registry API
- `platform/orchestrator` - job FSM, Podman pod lifecycle, aggregation, job/admin control plane
- `platform/extractor` - secure ZIP extraction, page discovery, provenance generation
- `platform/scanner-runner` - scanner worker runtime, plugin loading, per-page scans
- `frontend` - SvelteKit 5 UI (playground, live status, report)
- `packages/contracts` - report + scanner-manifest schemas and generated Go/TS types
- `packages/shared-go` - shared models/events/messaging/storage/logging/config utilities

---

## 1) `platform/api`

### Responsibility

Public API boundary for job intake and read APIs.

- Accept URL and ZIP jobs
- Enforce URL security policy (SSRF guardrails)
- Publish `job.created` events to NATS
- Serve job status/results/report links
- Stream live updates over SSE
- Expose scanner catalog to frontend

### Directory Responsibility

- `platform/api/cmd/server` - service bootstrap, env config, dependency wiring
- `platform/api/internal/api` - router, handlers, middleware, security, response composition
- `platform/api/internal/messaging` - NATS publish/subscribe glue + SSE broadcast adapter
- `platform/api/internal/sse` - in-memory fanout hub for SSE clients
- `platform/api/internal/status` - local status projection model/store helpers
- `platform/api/internal/statussource` - status read client against orchestrator admin API

### Key File Responsibility

- `platform/api/internal/api/router.go` - route registration + middleware stacks
- `platform/api/internal/api/handlers_jobs_url_submit.go` - URL job intake + validation + publish
- `platform/api/internal/api/handlers_jobs_zip_upload.go` - ZIP upload intake + staging upload
- `platform/api/internal/api/handlers_sse.go` - `/stream` endpoint lifecycle + keepalive + terminal close
- `platform/api/internal/api/security.go` - host/IP classification, DNS resolve, CIDR blocking rules
- `platform/api/internal/api/handlers_jobs_status.go` - `/jobs/{id}`, `/report`, `/results`
- `platform/api/internal/api/handlers_scanners.go` - scanner list endpoint
- `platform/api/internal/messaging/service.go` - event publish + typed subscriptions
- `platform/api/internal/sse/hub.go` - per-job client pub/sub + backpressure strategy
- `platform/api/cmd/server/main.go` - startup flow + graceful shutdown

### HTTP API Surface

- `POST /api/v1/jobs/urls` - submit URL scan job
- `POST /api/v1/jobs/zip` - submit ZIP scan job
- `GET /api/v1/jobs/{id}` - job status model
- `GET /api/v1/jobs/{id}/report` - report URL/redirect behavior
- `GET /api/v1/jobs/{id}/results` - JSON results URL/redirect behavior
- `GET /api/v1/jobs/{id}/stream` - SSE updates for a job
- `GET /api/v1/scanners` - scanner registry list
- `GET /healthz` - health check

### Events

- Publishes: `job.created`
- Consumes for SSE/status fanout:
  - `extraction.ready`
  - `extraction.failed`
  - `scan.page.completed`
  - `scan.completed`
  - `scan.failed`
  - `job.completed`
  - `job.failed`

---

## 2) `platform/orchestrator`

### Responsibility

Core execution control plane.

- Owns job state machine and transitions
- Creates/manages Podman pods and containers for each job
- Handles extraction and scanning lifecycle events
- Aggregates scanner outputs into unified report
- Publishes terminal job events
- Exposes admin/status APIs for control plane visibility

### Directory Responsibility

- `platform/orchestrator/cmd/orchestrator` - service bootstrap/config
- `platform/orchestrator/internal/fsm` - state machine wrapper
- `platform/orchestrator/internal/orchestrator` - lifecycle/event handlers, scanning, aggregation, cleanup
- `platform/orchestrator/internal/podman` - Podman HTTP API client (pods/containers/volumes)
- `platform/orchestrator/internal/messaging` - NATS consumers + publisher
- `platform/orchestrator/internal/db` - Postgres persistence + event log + metrics
- `platform/orchestrator/internal/api` - admin HTTP endpoints

### Key File Responsibility

- `platform/orchestrator/internal/fsm/state.go` - state transitions delegate + guards
- `platform/orchestrator/internal/orchestrator/events.go` - event-driven orchestration logic
- `platform/orchestrator/internal/orchestrator/scanning.go` - scanner container startup and execution
- `platform/orchestrator/internal/orchestrator/extraction.go` - extraction worker orchestration
- `platform/orchestrator/internal/orchestrator/completion.go` - completion + publish paths
- `platform/orchestrator/internal/orchestrator/report_aggregator.go` - merged report assembly entry
- `platform/orchestrator/internal/orchestrator/rule_deduplication.go` - cross-scanner dedup policy
- `platform/orchestrator/internal/orchestrator/job_cleanup.go` - pod/volume/staging cleanup on terminal states
- `platform/orchestrator/internal/messaging/consumers.go` - NATS subscription wiring
- `platform/orchestrator/internal/api/server.go` - admin API handlers

### FSM Surface

- States:
  - `PENDING`
  - `EXTRACTING`
  - `READY_TO_SCAN`
  - `SCANNING`
  - `COMPLETING`
  - `DONE`
  - `FAILED`
- Typical lifecycle:
  - URL job: `PENDING -> READY_TO_SCAN -> SCANNING -> COMPLETING -> DONE`
  - ZIP job: `PENDING -> EXTRACTING -> READY_TO_SCAN -> SCANNING -> COMPLETING -> DONE`
- Error path:
  - any active state can transition to `FAILED`

### Admin API Surface

- `GET /api/v1/jobs` - list jobs (filters/pagination)
- `GET /api/v1/jobs/{id}` - get job record
- `GET /api/v1/jobs/{id}/events` - event trace for a job
- `POST /api/v1/test/jobs` - test job creation endpoint
- `GET /api/v1/pods` - pod listing and associations
- `GET /api/v1/pods/{id}` - pod detail
- `GET /api/v1/status` - orchestrator/system status summary
- `GET /healthz` - health

### Events

- Consumes:
  - `job.created`
  - `extraction.ready`
  - `extraction.failed`
  - `scan.page.completed`
  - `scan.completed`
  - `scan.failed`
- Publishes:
  - `job.completed`
  - `job.failed`

---

## 3) `platform/extractor`

### Responsibility

Securely process ZIP jobs inside the job pod.

- Download uploaded ZIP from staging
- Validate archive safety constraints
- Extract to workspace safely
- Discover HTML pages
- Generate and upload provenance
- Serve extracted site locally for scanners
- Publish extraction success/failure event

### Directory Responsibility

- `platform/extractor/cmd/server` - runtime orchestration and startup
- `platform/extractor/internal/extractor` - ZIP validation and extraction safety logic
- `platform/extractor/internal/discovery` - HTML page discovery
- `platform/extractor/internal/provenance` - provenance manifest generation
- `platform/extractor/internal/server` - embedded static file server

### Key File Responsibility

- `platform/extractor/internal/extractor/extractor.go` - full validation + extraction core
- `platform/extractor/internal/discovery/discovery.go` - deterministic HTML discovery and IDs
- `platform/extractor/internal/provenance/provenance.go` - provenance JSON generation/writes
- `platform/extractor/internal/server/server.go` - local static host lifecycle
- `platform/extractor/cmd/server/main.go` - stage pipeline + event publishing + shutdown

### Safety Boundaries

- ZIP entry count cap
- per-entry size cap
- total uncompressed size cap
- compression-ratio checks (ZIP bomb mitigation)
- path traversal/absolute path/drive-letter blocking
- base-dir confinement checks before writes
- restrictive file permissions after extraction

### Event Surface

- Publishes:
  - `extraction.ready`
  - `extraction.failed`

---

## 4) `platform/scanner-runner`

### Responsibility

Worker runtime that executes selected scanners over page provenance.

- Resolve scanner plugin by module/manifest
- Validate scanner options against manifest schema
- Execute shared scanner lifecycle
- Process pages and generate issue artifacts
- Upload results/logs/artifacts
- Publish scan progress and terminal events

### Directory Responsibility

- `platform/scanner-runner/src/worker.ts` - scanner selection + execution dispatch
- `platform/scanner-runner/src/core` - base lifecycle, types, browser/page/event/storage abstractions
- `platform/scanner-runner/src/core/plugins` - plugin discovery/load/validation
- `platform/scanner-runner/src/core/manifest` - manifest and config-schema validation
- `platform/scanner-runner/src/scanners` - built-in scanner implementations
- `platform/scanner-runner/src/worker` - worker-specific validation helpers

### Key File Responsibility

- `platform/scanner-runner/src/worker.ts` - module resolution and run orchestration
- `platform/scanner-runner/src/core/scanner-base.ts` - standard scanner lifecycle template
- `platform/scanner-runner/src/core/plugins/plugin-discovery.ts` - manifest path discovery
- `platform/scanner-runner/src/core/plugins/plugin-load.ts` - dynamic module loading and factory resolution
- `platform/scanner-runner/src/core/manifest/index.ts` - Ajv manifest/config validation
- `platform/scanner-runner/src/worker/worker-validation.ts` - env option parsing and enforcement

### Scanner Base Lifecycle

- initialize runtime dependencies
- load provenance/pages
- iterate pages with bounded concurrency/retries
- call scanner-specific `scanPage()`
- aggregate/build output model
- write results + upload artifacts
- publish completion/failure events
- cleanup browser/runtime resources

### Built-in Scanners

- `axe` - WCAG rule violations via axe-core
- `lighthouse` - Lighthouse categories/audits
- `seo` - content/meta/structure SEO checks
- `security-headers` - header posture + policy checks
- `link-checker` - broken links, redirect/latency issues
- `ai-navigator` - goal-oriented browser flow evaluation

---

## 5) `frontend`

### Responsibility

Primary operator and demo interface.

- Landing and product narrative
- Playground job submission UX
- Live scan status stream UI
- Unified report visualization and filtering
- Artifact access and issue triage views

### Directory Responsibility

- `frontend/src/routes` - page routes
- `frontend/src/lib/components` - UI/report/playground/status components
- `frontend/src/lib/stores` - rune-based app stores/factories
- `frontend/src/lib/api` - HTTP and SSE client layer
- `frontend/src/lib/report` - report transforms/selectors/derived helpers
- `frontend/src/app.css` - Tailwind v4 theme tokens + global styles

### Route Surface

- `/` - marketing landing page
- `/playground` - scanner selection and job submission
- `/scan/[id]` - live scan progress status
- `/scan/[id]/report` - report shell with tabs/views

### Key File Responsibility

- `frontend/src/lib/stores/scan-status.svelte.ts` - live status state machine and stream/poll behavior
- `frontend/src/lib/stores/scan-report.svelte.ts` - report loading + UI state
- `frontend/src/lib/stores/scan-history.svelte.ts` - persisted run history
- `frontend/src/lib/api/client.ts` - submit jobs + fetch scanners
- `frontend/src/lib/api/sse.ts` - SSE connection/reconnect/event parsing
- `frontend/src/lib/components/report/ReportShell.svelte` - report tabs, URL-driven state, audience modes

### Frontend API Surface

Consumed endpoints:

- `POST /api/v1/jobs/urls`
- `POST /api/v1/jobs/zip`
- `GET /api/v1/scanners`
- `GET /api/v1/jobs/{id}`
- `GET /api/v1/jobs/{id}/stream`
- `GET /api/v1/jobs/{id}/results`
- `GET /api/v1/jobs/{id}/report`

---

## 6) `packages/contracts`

### Responsibility

Schema-first contracts shared between Go and TypeScript services.

- Define JSON schemas
- Generate strict Go and TS types
- Provide validators around schema output

### Directory Responsibility

- `packages/contracts/report` - unified report schema and generated bindings
- `packages/contracts/scanner-manifest` - scanner manifest schema and generated bindings
- `packages/contracts/events` - event fixtures/examples

### Key Artifacts

- `packages/contracts/report/schema/unified-report.v2.schema.json`
- `packages/contracts/report/generated/go/report_schema.go`
- `packages/contracts/report/generated/typescript/unified-report.v2.ts`
- `packages/contracts/report/generated/typescript/validator.ts`
- `packages/contracts/scanner-manifest/schema/scanner-manifest.schema.json`
- `packages/contracts/scanner-manifest/scanner_manifest.go`
- `packages/contracts/scanner-manifest/validator.go`
- `packages/contracts/scanner-manifest/generated/typescript/scanner-manifest.ts`

---

## 7) `packages/shared-go`

### Responsibility

Shared runtime primitives consumed by API, orchestrator, extractor, and tooling.

### Directory Responsibility

- `packages/shared-go/models` - domain models and job/provenance types
- `packages/shared-go/events` - event type constants + payload structs + envelope
- `packages/shared-go/messaging` - NATS client wrapper + streams/subjects helpers
- `packages/shared-go/storage` - MinIO client, presigned URLs, bucket helpers
- `packages/shared-go/httputil` - structured error and JSON response helpers
- `packages/shared-go/logging` - context-aware structured logging
- `packages/shared-go/config` - env/config loaders and validators
- `packages/shared-go/bootstrap` - common startup wiring helpers
- `packages/shared-go/domain/job` - job transition/order utilities
- `packages/shared-go/scannercatalog` - embedded scanner manifests
- `packages/shared-go/scannerregistry` - registry, aliases, modules/categories resolution

### Event Type Reference

Defined in `packages/shared-go/events/types.go`:

- `job.created`
- `job.completed`
- `job.failed`
- `extraction.ready`
- `extraction.failed`
- `scan.page.completed`
- `scan.completed`
- `scan.failed`

---

## 8) End-to-End Responsibility Map

### Runtime Flow

1. Frontend submits job -> API (`/api/v1/jobs/urls` or `/api/v1/jobs/zip`)
2. API validates input/security -> publishes `job.created`
3. Orchestrator consumes `job.created` -> creates pod/workspace/runtime
4. ZIP jobs run extractor -> emits `extraction.ready`/`extraction.failed`
5. Orchestrator schedules scanner-runner containers
6. Scanner runners emit `scan.page.completed`, then `scan.completed`/`scan.failed`
7. Orchestrator aggregates merged report -> emits `job.completed`/`job.failed`
8. API status source and SSE reflect updates to frontend
9. Frontend status page transitions to report view with artifact/report links

### Ownership by Concern

- Input validation/security - `platform/api/internal/api/security.go`
- Job lifecycle and transitions - `platform/orchestrator/internal/fsm/state.go`
- Container orchestration - `platform/orchestrator/internal/podman/*`
- ZIP safety - `platform/extractor/internal/extractor/extractor.go`
- Scanner execution abstraction - `platform/scanner-runner/src/core/scanner-base.ts`
- Unified report shape - `packages/contracts/report/schema/unified-report.v2.schema.json`
- Live UI streaming state - `frontend/src/lib/stores/scan-status.svelte.ts`

---

## 9) Contributor "Where Do I Change X?" Index

- Add/modify public API endpoint -> `platform/api/internal/api/router.go` + handlers
- Tighten URL security/SSRF policy -> `platform/api/internal/api/security.go`
- Add new scanner module behavior to orchestration -> `platform/orchestrator/internal/orchestrator/scanning.go`
- Change job transition logic -> `packages/shared-go/domain/job/state.go`
- Add scanner implementation -> `platform/scanner-runner/src/scanners/<scanner>/index.ts`
- Add scanner manifest fields -> `packages/contracts/scanner-manifest/schema/scanner-manifest.schema.json`
- Update report model fields -> `packages/contracts/report/schema/unified-report.v2.schema.json`
- Change report tab behavior -> `frontend/src/lib/components/report/ReportShell.svelte`
- Adjust live status UX/state -> `frontend/src/lib/stores/scan-status.svelte.ts`

---

## 10) Reference Paths

### Core Service Entrypoints

- `platform/api/cmd/server/main.go`
- `platform/orchestrator/cmd/orchestrator/main.go`
- `platform/extractor/cmd/server/main.go`
- `platform/scanner-runner/src/index.ts`
- `frontend/src/routes/+layout.svelte`

### API and Streaming

- `platform/api/internal/api/router.go`
- `platform/api/internal/api/handlers_jobs_url_submit.go`
- `platform/api/internal/api/handlers_jobs_zip_upload.go`
- `platform/api/internal/api/handlers_jobs_status.go`
- `platform/api/internal/api/handlers_sse.go`
- `frontend/src/lib/api/client.ts`
- `frontend/src/lib/api/sse.ts`

### Orchestration and Aggregation

- `platform/orchestrator/internal/orchestrator/events.go`
- `platform/orchestrator/internal/orchestrator/scanning.go`
- `platform/orchestrator/internal/orchestrator/completion.go`
- `platform/orchestrator/internal/orchestrator/report_aggregator.go`
- `platform/orchestrator/internal/orchestrator/rule_deduplication.go`

### Scanner Runtime and Plugins

- `platform/scanner-runner/src/worker.ts`
- `platform/scanner-runner/src/core/scanner-base.ts`
- `platform/scanner-runner/src/core/plugins/plugin-discovery.ts`
- `platform/scanner-runner/src/core/plugins/plugin-load.ts`
- `platform/scanner-runner/src/core/manifest/index.ts`

### Shared Contracts and Models

- `packages/contracts/report/schema/unified-report.v2.schema.json`
- `packages/contracts/scanner-manifest/schema/scanner-manifest.schema.json`
- `packages/shared-go/events/types.go`
- `packages/shared-go/messaging/nats.go`
- `packages/shared-go/models/job.go`
- `packages/shared-go/scannerregistry/registry.go`

---

## Notes for Maintainers

- Keep this map updated whenever you add/remove:
  - routes/endpoints
  - event types or subjects
  - scanner manifests or scanner modules
  - report schema major fields
- Recommended update points:
  - after any cross-service architecture change
  - before public release notes/documentation refresh

---

## 11) Tracked Repository Tree

This tree is intentionally limited to versioned source and documentation paths.
Generated build output, local environment files, editor caches, and scratch tooling
artifacts are omitted so this section stays accurate for a clean clone.

```text
stageflow/
├── .github/  # GitHub automation, templates, and workflows
├── docs/  # project documentation and screenshot assets
│   ├── images/  # README and doc screenshots
│   ├── testing/  # testing strategy documentation
│   ├── ARCHITECTURE.md  # architecture guide
│   ├── CONFIGURATION.md  # configuration guide
│   ├── TOOLS.md  # CLI tooling guide
│   └── REPOMAP.md  # contributor repo map
├── frontend/  # SvelteKit frontend application
│   ├── .storybook/  # Storybook configuration
│   ├── scripts/  # Storybook and frontend helper scripts
│   ├── src/  # application source
│   │   ├── lib/  # shared frontend modules, stores, and components
│   │   ├── routes/  # SvelteKit routes
│   │   └── stories/  # Storybook docs content
│   ├── static/  # static public assets
│   ├── tests/  # frontend unit and component tests
│   ├── package.json  # frontend package manifest
│   └── bun.lock  # frontend dependency lockfile
├── infra/  # deployment, compose, proxy, and observability config
├── packages/  # shared schemas and libraries
│   ├── contracts/  # JSON Schema contracts and generated types
│   └── shared-go/  # shared Go models and utilities
├── platform/  # core services
│   ├── api/  # public intake API and SSE service
│   ├── extractor/  # ZIP extraction service
│   ├── orchestrator/  # job lifecycle and aggregation service
│   └── scanner-runner/  # scanner worker runtime
├── scripts/  # project automation scripts
├── tests/  # end-to-end tests and fixtures
│   ├── e2e/  # service integration coverage
│   └── fixtures/  # sample sites and archives
├── tools/  # developer and operations CLIs
│   ├── stageflow-cli/  # API client for submitting URL scan jobs
│   ├── job-status-cli/  # job inspection CLI source
│   ├── suite-runner/  # suite execution CLI source
│   └── README.md  # tooling overview and usage
├── AGENTS.md  # repository-specific agent instructions
├── go.work  # Go workspace definition
├── justfile  # primary task runner entrypoint
├── README.md  # project overview and quickstart
└── tsconfig.strict.json  # shared TypeScript strictness baseline
```
│   │   │   │       ├── alert.test.ts  # test file
│   │   │   │       ├── badge.test.ts  # test file
│   │   │   │       ├── button.test.ts  # test file
│   │   │   │       ├── chip-variants.test.ts  # test file
│   │   │   │       ├── chip.component.test.ts  # test file
│   │   │   │       ├── modal-variants.test.ts  # test file
│   │   │   │       ├── modal.component.test.ts  # test file
│   │   │   │       ├── PageSection.test.ts  # test file
│   │   │   │       ├── panel-variants.test.ts  # test file
│   │   │   │       ├── panel.component.test.ts  # test file
│   │   │   │       ├── select.test.ts  # test file
│   │   │   │       ├── SelectField.test.ts  # test file
│   │   │   │       └── TerminalCardHeader.test.ts  # test file
│   │   │   ├── report/  # directory
│   │   │   │   ├── filters.test.ts  # test file
│   │   │   │   ├── focus.test.ts  # test file
│   │   │   │   ├── occurrence-mode.test.ts  # test file
│   │   │   │   ├── scanner-summary.test.ts  # test file
│   │   │   │   ├── screenshots.test.ts  # test file
│   │   │   │   ├── severity.test.ts  # test file
│   │   │   │   └── virtualization.test.ts  # test file
│   │   │   ├── stores/  # directory
│   │   │   │   ├── scan-status/  # directory
│   │   │   │   │   ├── constants.test.ts  # test file
│   │   │   │   │   └── log-messages.test.ts  # test file
│   │   │   │   ├── scan-report.test.ts  # test file
│   │   │   │   └── scan-status.test.ts  # test file
│   │   │   └── utils/  # directory
│   │   │       ├── cn.test.ts  # test file
│   │   │       ├── date.test.ts  # test file
│   │   │       ├── failure-summary.test.ts  # test file
│   │   │       ├── fix-examples.test.ts  # test file
│   │   │       └── wcag.test.ts  # test file
│   │   └── setup.ts  # TypeScript source file
│   ├── .gitignore  # git ignore patterns
│   ├── .npmrc  # file
│   ├── .prettierignore  # file
│   ├── .prettierrc  # file
│   ├── AGENTS.md  # agent instructions and repository conventions
│   ├── bun.lock  # dependency lockfile
│   ├── Dockerfile  # container build definition
│   ├── eslint.config.js  # JavaScript source file
│   ├── junit.xml  # file
│   ├── knip.json  # JSON data or configuration
│   ├── nginx.conf  # file
│   ├── package.json  # Node package manifest
│   ├── svelte.config.js  # JavaScript source file
│   ├── tsconfig.eslint.json  # JSON data or configuration
│   ├── tsconfig.json  # JSON data or configuration
│   ├── vite.config.ts  # TypeScript source file
│   └── vitest.config.ts  # TypeScript source file
├── infra/  # deployment and runtime infrastructure
│   ├── caddy/  # Caddy reverse proxy configs
│   │   ├── Caddyfile  # Caddy reverse proxy configuration
│   │   └── Caddyfile.staging.example  # Caddy reverse proxy configuration
│   ├── compose/  # compose stacks for environments
│   │   ├── podman-compose.local.yml  # YAML configuration
│   │   ├── podman-compose.staging.yml  # YAML configuration
│   │   ├── podman-compose.test.yml  # test file
│   │   └── podman-compose.yml  # YAML configuration
│   ├── grafana/  # Grafana provisioning and dashboards
│   │   └── provisioning/  # directory
│   │       ├── dashboards/  # directory
│   │       │   ├── dashboards.yml  # YAML configuration
│   │       │   ├── job-overview.json  # JSON data or configuration
│   │       │   └── provenance-validation.json  # JSON data or configuration
│   │       └── datasources/  # directory
│   │           └── orchestrator.yml  # YAML configuration
│   ├── minio/  # MinIO initialization assets
│   │   └── init-buckets.sh  # shell script
│   ├── quadlets/  # systemd/Quadlet templates
│   │   └── templates/  # directory
│   │       ├── stageflow-caddy.container.in  # file
│   │       ├── stageflow-frontend.container.in  # file
│   │       ├── stageflow-grafana.container.in  # file
│   │       ├── stageflow-minio.container.in  # file
│   │       ├── stageflow-nats.container.in  # file
│   │       ├── stageflow-orchestrator.container.in  # file
│   │       ├── stageflow-platform-api.container.in  # file
│   │       ├── stageflow-postgres.container.in  # file
│   │       ├── stageflow.pod.in  # file
│   │       └── stageflow.target.in  # file
│   └── scanners/  # scanner runtime configuration
│       ├── scanners.example.yaml  # YAML configuration
│       └── scanners.yaml  # YAML configuration
├── packages/  # shared contracts and libraries
│   ├── contracts/  # schema-first cross-language contracts
│   │   ├── events/  # event fixture payloads
│   │   │   └── fixtures/  # directory
│   │   │       ├── scan.completed.json  # JSON data or configuration
│   │   │       ├── scan.failed.json  # JSON data or configuration
│   │   │       └── scan.page.completed.json  # JSON data or configuration
│   │   ├── report/  # unified report schema and generated types
│   │   │   ├── fixtures/  # directory
│   │   │   │   ├── unified-report.v2.all-scans.json  # JSON data or configuration
│   │   │   │   └── unified-report.v2.json  # JSON data or configuration
│   │   │   ├── generated/  # directory
│   │   │   │   ├── go/  # directory
│   │   │   │   │   ├── go.mod  # Go module definition
│   │   │   │   │   └── report_schema.go  # Go source file
│   │   │   │   └── typescript/  # directory
│   │   │   │       ├── index.d.ts  # TypeScript source file
│   │   │   │       ├── index.d.ts.map  # file
│   │   │   │       ├── index.js  # JavaScript source file
│   │   │   │       ├── index.ts  # TypeScript source file
│   │   │   │       ├── unified-report.v2.d.ts  # TypeScript source file
│   │   │   │       ├── unified-report.v2.d.ts.map  # file
│   │   │   │       ├── unified-report.v2.js  # JavaScript source file
│   │   │   │       ├── unified-report.v2.ts  # TypeScript source file
│   │   │   │       ├── validator.d.ts  # TypeScript source file
│   │   │   │       ├── validator.d.ts.map  # file
│   │   │   │       ├── validator.js  # JavaScript source file
│   │   │   │       └── validator.ts  # TypeScript source file
│   │   │   ├── schema/  # directory
│   │   │   │   ├── README.md  # project overview and quickstart
│   │   │   │   ├── unified-report.v2.schema.json  # JSON data or configuration
│   │   │   │   └── validate.js  # JavaScript source file
│   │   │   ├── scripts/  # directory
│   │   │   │   └── pre-commit-check.sh  # shell script
│   │   │   ├── bun.lock  # dependency lockfile
│   │   │   ├── Makefile  # file
│   │   │   ├── MIGRATION.md  # documentation
│   │   │   ├── package.json  # Node package manifest
│   │   │   └── tsconfig.json  # JSON data or configuration
│   │   ├── scanner-manifest/  # scanner manifest schema and generated types
│   │   │   ├── fixtures/  # directory
│   │   │   │   ├── scanner-manifest.full.json  # JSON data or configuration
│   │   │   │   └── scanner-manifest.min.json  # JSON data or configuration
│   │   │   ├── generated/  # directory
│   │   │   │   └── typescript/  # directory
│   │   │   │       ├── index.ts  # TypeScript source file
│   │   │   │       └── scanner-manifest.ts  # TypeScript source file
│   │   │   ├── schema/  # directory
│   │   │   │   ├── README.md  # project overview and quickstart
│   │   │   │   ├── scanner-manifest.schema.json  # JSON data or configuration
│   │   │   │   └── validate.js  # JavaScript source file
│   │   │   ├── go.mod  # Go module definition
│   │   │   ├── go.sum  # Go dependency checksums
│   │   │   ├── Makefile  # file
│   │   │   ├── package.json  # Node package manifest
│   │   │   ├── scanner_manifest.go  # Go source file
│   │   │   ├── tsconfig.json  # JSON data or configuration
│   │   │   └── validator.go  # Go source file
│   │   └── AGENTS.md  # agent instructions and repository conventions
│   └── shared-go/  # shared Go packages for platform services
│       ├── bootstrap/  # startup bootstrap helpers
│       │   └── bootstrap.go  # Go source file
│       ├── config/  # env loading and validation helpers
│       │   ├── env.go  # Go source file
│       │   ├── env_test.go  # Go test file
│       │   ├── loaders.go  # Go source file
│       │   ├── loaders_test.go  # Go test file
│       │   ├── validation.go  # Go source file
│       │   └── validation_test.go  # Go test file
│       ├── domain/  # shared domain logic
│       │   └── job/  # job state transition logic
│       │       ├── state.go  # Go source file
│       │       └── state_test.go  # Go test file
│       ├── events/  # event constants and payload types
│       │   ├── contracts_test.go  # Go test file
│       │   ├── envelope.go  # Go source file
│       │   ├── events_test.go  # Go test file
│       │   ├── types.go  # Go source file
│       │   └── types_test.go  # Go test file
│       ├── httputil/  # HTTP response and error helpers
│       │   ├── errors.go  # Go source file
│       │   ├── errors_test.go  # Go test file
│       │   ├── response.go  # Go source file
│       │   └── response_test.go  # Go test file
│       ├── logging/  # structured logging helpers
│       │   ├── logger.go  # Go source file
│       │   └── logger_test.go  # Go test file
│       ├── messaging/  # NATS/JetStream wrappers
│       │   ├── nats.go  # Go source file
│       │   ├── nats_client_test.go  # Go test file
│       │   └── nats_test.go  # Go test file
│       ├── models/  # shared domain models
│       │   ├── contracts_test.go  # Go test file
│       │   ├── job.go  # Go source file
│       │   ├── job_test.go  # Go test file
│       │   ├── provenance.go  # Go source file
│       │   ├── provenance_test.go  # Go test file
│       │   └── results_test.go  # Go test file
│       ├── scannercatalog/  # embedded scanner manifests
│       │   ├── manifests/  # directory
│       │   │   ├── ai-navigator/  # directory
│       │   │   │   └── manifest.json  # JSON data or configuration
│       │   │   ├── axe/  # directory
│       │   │   │   └── manifest.json  # JSON data or configuration
│       │   │   ├── lighthouse/  # directory
│       │   │   │   └── manifest.json  # JSON data or configuration
│       │   │   ├── link-checker/  # directory
│       │   │   │   └── manifest.json  # JSON data or configuration
│       │   │   ├── security-headers/  # directory
│       │   │   │   └── manifest.json  # JSON data or configuration
│       │   │   └── seo/  # directory
│       │   │       └── manifest.json  # JSON data or configuration
│       │   ├── catalog.go  # Go source file
│       │   └── catalog_test.go  # Go test file
│       ├── scannerregistry/  # scanner registry and module resolution
│       │   ├── config.go  # Go source file
│       │   ├── config_integration_test.go  # Go test file
│       │   ├── registry.go  # Go source file
│       │   ├── registry_module_tokens.go  # Go source file
│       │   ├── registry_modules.go  # Go source file
│       │   ├── registry_modules_test.go  # Go test file
│       │   ├── registry_query.go  # Go source file
│       │   ├── registry_registration.go  # Go source file
│       │   ├── registry_test.go  # Go test file
│       │   ├── types.go  # Go source file
│       │   └── types_test.go  # Go test file
│       ├── storage/  # directory
│       │   ├── client.go  # Go source file
│       │   ├── minio.go  # Go source file
│       │   ├── minio_client_test.go  # Go test file
│       │   └── minio_test.go  # Go test file
│       ├── AGENTS.md  # agent instructions and repository conventions
│       ├── go.mod  # Go module definition
│       └── go.sum  # Go dependency checksums
├── platform/  # backend services and workers
│   ├── api/  # public API service
│   │   ├── cmd/  # API service entrypoints
│   │   │   ├── healthcheck/  # healthcheck binary source
│   │   │   │   └── main.go  # Go source file
│   │   │   └── server/  # API server startup code
│   │   │       ├── config.go  # Go source file
│   │   │       ├── main.go  # Go source file
│   │   │       └── main_test.go  # Go test file
│   │   ├── internal/  # API internal packages
│   │   │   ├── api/  # API routing, handlers, middleware, security
│   │   │   │   ├── handlers_coverage_test.go  # Go test file
│   │   │   │   ├── handlers_health.go  # Go source file
│   │   │   │   ├── handlers_jobs_highlight_style.go  # Go source file
│   │   │   │   ├── handlers_jobs_modules.go  # Go source file
│   │   │   │   ├── handlers_jobs_modules_test.go  # Go test file
│   │   │   │   ├── handlers_jobs_status.go  # Go source file
│   │   │   │   ├── handlers_jobs_url_submit.go  # Go source file
│   │   │   │   ├── handlers_jobs_zip_upload.go  # Go source file
│   │   │   │   ├── handlers_scanners.go  # Go source file
│   │   │   │   ├── handlers_sse.go  # Go source file
│   │   │   │   ├── handlers_sse_test.go  # Go test file
│   │   │   │   ├── handlers_test.go  # Go test file
│   │   │   │   ├── job_status_reader.go  # Go source file
│   │   │   │   ├── job_status_reader_test.go  # Go test file
│   │   │   │   ├── job_status_response.go  # Go source file
│   │   │   │   ├── job_status_screenshots.go  # Go source file
│   │   │   │   ├── job_status_screenshots_test.go  # Go test file
│   │   │   │   ├── middleware.go  # Go source file
│   │   │   │   ├── middleware_test.go  # Go test file
│   │   │   │   ├── object_keys.go  # Go source file
│   │   │   │   ├── object_keys_test.go  # Go test file
│   │   │   │   ├── router.go  # Go source file
│   │   │   │   ├── scanner_configs.go  # Go source file
│   │   │   │   ├── security.go  # Go source file
│   │   │   │   ├── security_dns_test.go  # Go test file
│   │   │   │   ├── security_resolver_test.go  # Go test file
│   │   │   │   ├── security_test.go  # Go test file
│   │   │   │   └── server.go  # Go source file
│   │   │   ├── messaging/  # API messaging integrations
│   │   │   │   ├── service.go  # Go source file
│   │   │   │   └── service_test.go  # Go test file
│   │   │   ├── sse/  # SSE hub implementation
│   │   │   │   ├── hub.go  # Go source file
│   │   │   │   └── hub_test.go  # Go test file
│   │   │   ├── status/  # API-side status projection models/store
│   │   │   │   ├── model.go  # Go source file
│   │   │   │   ├── schema.sql  # SQL schema or query file
│   │   │   │   ├── store.go  # Go source file
│   │   │   │   ├── store_handlers.go  # Go source file
│   │   │   │   ├── store_queries.go  # Go source file
│   │   │   │   ├── store_schema.go  # Go source file
│   │   │   │   └── store_test.go  # Go test file
│   │   │   └── statussource/  # orchestrator status source client
│   │   │       ├── client.go  # Go source file
│   │   │       └── client_test.go  # Go test file
│   │   ├── tests/  # API integration tests
│   │   │   └── integration/  # directory
│   │   │       ├── doc.go  # Go source file
│   │   │       └── messaging_nats_test.go  # Go test file
│   │   ├── AGENTS.md  # agent instructions and repository conventions
│   │   ├── Dockerfile  # container build definition
│   │   ├── go.mod  # Go module definition
│   │   └── go.sum  # Go dependency checksums
│   ├── extractor/  # ZIP extraction service
│   │   ├── cmd/  # extractor entrypoints
│   │   │   └── server/  # directory
│   │   │       ├── main.go  # Go source file
│   │   │       └── stage_logger.go  # Go source file
│   │   ├── internal/  # extractor internal packages
│   │   │   ├── discovery/  # extracted page discovery
│   │   │   │   ├── discovery.go  # Go source file
│   │   │   │   └── discovery_test.go  # Go test file
│   │   │   ├── extractor/  # secure ZIP extraction logic
│   │   │   │   ├── extract_file_test.go  # Go test file
│   │   │   │   ├── extract_zip_test.go  # Go test file
│   │   │   │   ├── extraction_skipped_test.go  # Go test file
│   │   │   │   ├── extractor.go  # Go source file
│   │   │   │   ├── test_helpers_test.go  # Go test file
│   │   │   │   └── validate_zip_test.go  # Go test file
│   │   │   ├── provenance/  # provenance generation logic
│   │   │   │   ├── generator_benchmark_test.go  # Go test file
│   │   │   │   ├── generator_generate_test.go  # Go test file
│   │   │   │   ├── generator_integration_test.go  # Go test file
│   │   │   │   ├── generator_write_test.go  # Go test file
│   │   │   │   ├── provenance.go  # Go source file
│   │   │   │   └── test_helpers_test.go  # Go test file
│   │   │   └── server/  # embedded static file server
│   │   │       ├── server.go  # Go source file
│   │   │       └── server_test.go  # Go test file
│   │   ├── testdata/  # directory
│   │   │   ├── html/  # directory
│   │   │   │   ├── empty/  # directory
│   │   │   │   │   └── .gitkeep  # file
│   │   │   │   ├── malicious/  # directory
│   │   │   │   │   └── evil.html  # HTML document
│   │   │   │   ├── nested/  # directory
│   │   │   │   │   ├── subdir/  # directory
│   │   │   │   │   │   └── deep.html  # HTML document
│   │   │   │   │   └── page.html  # HTML document
│   │   │   │   └── simple/  # directory
│   │   │   │       ├── about.htm  # file
│   │   │   │       ├── index.html  # HTML document
│   │   │   │       └── readme.txt  # file
│   │   │   ├── absolute-path.zip  # ZIP archive fixture
│   │   │   ├── nested-site.zip  # ZIP archive fixture
│   │   │   └── path-traversal.zip  # ZIP archive fixture
│   │   ├── AGENTS.md  # agent instructions and repository conventions
│   │   ├── Dockerfile  # container build definition
│   │   ├── go.mod  # Go module definition
│   │   ├── go.sum  # Go dependency checksums
│   │   └── integration_test.go  # Go test file
│   ├── orchestrator/  # orchestration control-plane service
│   │   ├── cmd/  # orchestrator entrypoints
│   │   │   └── orchestrator/  # directory
│   │   │       ├── config.go  # Go source file
│   │   │       ├── config_test.go  # Go test file
│   │   │       └── main.go  # Go source file
│   │   ├── config/  # directory
│   │   │   └── scanners.yaml  # YAML configuration
│   │   ├── internal/  # orchestrator internal packages
│   │   │   ├── api/  # orchestrator admin API
│   │   │   │   ├── postgres_test_harness_test.go  # Go test file
│   │   │   │   ├── server.go  # Go source file
│   │   │   │   ├── server_test.go  # Go test file
│   │   │   │   └── test_jobs.go  # Go test file
│   │   │   ├── db/  # orchestrator Postgres data access
│   │   │   │   ├── context_propagation_test.go  # Go test file
│   │   │   │   ├── database.go  # Go source file
│   │   │   │   ├── database_test.go  # Go test file
│   │   │   │   ├── job_events.go  # Go source file
│   │   │   │   ├── job_events_retention.go  # Go source file
│   │   │   │   ├── job_events_retention_test.go  # Go test file
│   │   │   │   ├── job_metrics.go  # Go source file
│   │   │   │   ├── job_scanners.go  # Go source file
│   │   │   │   ├── job_updates.go  # Go source file
│   │   │   │   ├── jobs.go  # Go source file
│   │   │   │   ├── jobs_create_test.go  # Go test file
│   │   │   │   ├── jobs_events_test.go  # Go test file
│   │   │   │   ├── jobs_scanners_test.go  # Go test file
│   │   │   │   ├── jobs_test_helpers_test.go  # Go test file
│   │   │   │   ├── jobs_update_test.go  # Go test file
│   │   │   │   ├── postgres_test_harness_test.go  # Go test file
│   │   │   │   ├── schema.sql  # SQL schema or query file
│   │   │   │   └── sql_bind.go  # Go source file
│   │   │   ├── fsm/  # orchestrator FSM wrappers
│   │   │   │   ├── state.go  # Go source file
│   │   │   │   └── state_test.go  # Go test file
│   │   │   ├── messaging/  # orchestrator NATS consumers/publisher
│   │   │   │   ├── consumers.go  # Go source file
│   │   │   │   ├── consumers_test.go  # Go test file
│   │   │   │   ├── publisher.go  # Go source file
│   │   │   │   └── publisher_test.go  # Go test file
│   │   │   ├── orchestrator/  # orchestration runtime logic
│   │   │   │   ├── completion.go  # Go source file
│   │   │   │   ├── completion_test.go  # Go test file
│   │   │   │   ├── deadline.go  # Go source file
│   │   │   │   ├── event_trace.go  # Go source file
│   │   │   │   ├── events.go  # Go source file
│   │   │   │   ├── extraction.go  # Go source file
│   │   │   │   ├── job_cleanup.go  # Go source file
│   │   │   │   ├── memory_storage_test.go  # Go test file
│   │   │   │   ├── orchestrator.go  # Go source file
│   │   │   │   ├── orchestrator_cleanup_test.go  # Go test file
│   │   │   │   ├── orchestrator_deadline_test.go  # Go test file
│   │   │   │   ├── orchestrator_extraction_test.go  # Go test file
│   │   │   │   ├── orchestrator_init_test.go  # Go test file
│   │   │   │   ├── orchestrator_job_created_test.go  # Go test file
│   │   │   │   ├── orchestrator_scanner_start_test.go  # Go test file
│   │   │   │   ├── orchestrator_scanning_test.go  # Go test file
│   │   │   │   ├── orchestrator_test_helpers_test.go  # Go test file
│   │   │   │   ├── podman_helpers.go  # Go source file
│   │   │   │   ├── postgres_test_harness_test.go  # Go test file
│   │   │   │   ├── report_aggregator.go  # Go source file
│   │   │   │   ├── report_aggregator_aggregate.go  # Go source file
│   │   │   │   ├── report_aggregator_helpers_test.go  # Go test file
│   │   │   │   ├── report_aggregator_pages.go  # Go source file
│   │   │   │   ├── report_aggregator_storage.go  # Go source file
│   │   │   │   ├── report_aggregator_test.go  # Go test file
│   │   │   │   ├── report_aggregator_utils.go  # Go source file
│   │   │   │   ├── rule_deduplication.go  # Go source file
│   │   │   │   ├── rule_deduplication_test.go  # Go test file
│   │   │   │   ├── scanning.go  # Go source file
│   │   │   │   ├── scanning_test.go  # Go test file
│   │   │   │   ├── url_jobs.go  # Go source file
│   │   │   │   └── url_jobs_test.go  # Go test file
│   │   │   └── podman/  # Podman API client wrappers
│   │   │       ├── client.go  # Go source file
│   │   │       ├── client_test.go  # Go test file
│   │   │       ├── containers.go  # Go source file
│   │   │       ├── containers_test.go  # Go test file
│   │   │       ├── live_contract_test.go  # Go test file
│   │   │       ├── live_pod_network_hosts_contract_test.go  # Go test file
│   │   │       ├── live_resource_limits_contract_test.go  # Go test file
│   │   │       ├── live_test_helpers_test.go  # Go test file
│   │   │       ├── pods.go  # Go source file
│   │   │       ├── pods_test.go  # Go test file
│   │   │       ├── volumes.go  # Go source file
│   │   │       └── volumes_test.go  # Go test file
│   │   ├── test/  # directory
│   │   │   ├── e2e_concurrency_test.go  # Go test file
│   │   │   ├── e2e_extraction_failure_test.go  # Go test file
│   │   │   ├── e2e_job_events_logging_test.go  # Go test file
│   │   │   ├── e2e_scan_failure_test.go  # Go test file
│   │   │   ├── e2e_success_test.go  # Go test file
│   │   │   ├── helpers_test.go  # Go test file
│   │   │   ├── memory_storage_test.go  # Go test file
│   │   │   ├── mock_podman_client_test.go  # Go test file
│   │   │   ├── mock_publisher_test.go  # Go test file
│   │   │   └── postgres_test_harness_test.go  # Go test file
│   │   ├── AGENTS.md  # agent instructions and repository conventions
│   │   ├── Dockerfile  # container build definition
│   │   ├── go.mod  # Go module definition
│   │   └── go.sum  # Go dependency checksums
│   └── scanner-runner/  # scanner worker runtime
│       ├── scripts/  # scanner-runner build helpers
│       │   ├── copy-builtin-manifests.mjs  # JavaScript module
│       │   ├── prepare-contracts-report-types.mjs  # JavaScript module
│       │   └── prepare-contracts-scanner-manifest.mjs  # JavaScript module
│       ├── src/  # scanner-runner source
│       │   ├── ai/  # AI navigation logic
│       │   │   ├── action-decider.ts  # TypeScript source file
│       │   │   ├── action-decision-parser.ts  # TypeScript source file
│       │   │   ├── decision-prompt.ts  # TypeScript source file
│       │   │   ├── goal-checker.ts  # TypeScript source file
│       │   │   ├── index.ts  # TypeScript source file
│       │   │   ├── json.ts  # TypeScript source file
│       │   │   ├── loop-detector.ts  # TypeScript source file
│       │   │   ├── page-analyzer.ts  # TypeScript source file
│       │   │   ├── types.ts  # TypeScript source file
│       │   │   └── vision-client.ts  # TypeScript source file
│       │   ├── config/  # scanner behavior config maps
│       │   │   ├── rule-behaviors.ts  # TypeScript source file
│       │   │   ├── rule-titles.ts  # TypeScript source file
│       │   │   └── user-impact.ts  # TypeScript source file
│       │   ├── core/  # scanner runtime core abstractions
│       │   │   ├── manifest/  # manifest schema validation
│       │   │   │   └── index.ts  # TypeScript source file
│       │   │   ├── plugins/  # plugin discovery and loading
│       │   │   │   ├── index.ts  # TypeScript source file
│       │   │   │   ├── loader.ts  # TypeScript source file
│       │   │   │   ├── plugin-discovery.ts  # TypeScript source file
│       │   │   │   ├── plugin-load.ts  # TypeScript source file
│       │   │   │   ├── plugin-loader-types.ts  # TypeScript source file
│       │   │   │   └── plugin-loader.ts  # TypeScript source file
│       │   │   ├── storage-provider/  # artifact storage abstractions
│       │   │   │   ├── async.ts  # TypeScript source file
│       │   │   │   ├── content-type.ts  # TypeScript source file
│       │   │   │   ├── endpoint.ts  # TypeScript source file
│       │   │   │   ├── files.ts  # TypeScript source file
│       │   │   │   ├── minio-errors.ts  # TypeScript source file
│       │   │   │   └── minio-storage-provider.ts  # TypeScript source file
│       │   │   ├── artifact-paths.ts  # TypeScript source file
│       │   │   ├── browser-manager.ts  # TypeScript source file
│       │   │   ├── config-loader.ts  # TypeScript source file
│       │   │   ├── event-publisher.ts  # TypeScript source file
│       │   │   ├── index.ts  # TypeScript source file
│       │   │   ├── page-iterator.ts  # TypeScript source file
│       │   │   ├── scan-stage-logger.ts  # TypeScript source file
│       │   │   ├── scanner-base.ts  # TypeScript source file
│       │   │   ├── screenshots.ts  # TypeScript source file
│       │   │   ├── storage-provider.ts  # TypeScript source file
│       │   │   ├── target-validation.ts  # TypeScript source file
│       │   │   ├── types.ts  # TypeScript source file
│       │   │   └── web-server-formatter.ts  # TypeScript source file
│       │   ├── scanners/  # built-in scanner implementations
│       │   │   ├── ai-navigator/  # directory
│       │   │   │   ├── agent.ts  # TypeScript source file
│       │   │   │   ├── index.ts  # TypeScript source file
│       │   │   │   ├── options.ts  # TypeScript source file
│       │   │   │   └── trace-uploader.ts  # TypeScript source file
│       │   │   ├── axe/  # directory
│       │   │   │   └── index.ts  # TypeScript source file
│       │   │   ├── lighthouse/  # directory
│       │   │   │   └── index.ts  # TypeScript source file
│       │   │   ├── link-checker/  # directory
│       │   │   │   ├── index.ts  # TypeScript source file
│       │   │   │   ├── types.ts  # TypeScript source file
│       │   │   │   └── validation.ts  # TypeScript source file
│       │   │   ├── security-headers/  # directory
│       │   │   │   └── index.ts  # TypeScript source file
│       │   │   ├── seo/  # directory
│       │   │   │   ├── checks/  # directory
│       │   │   │   │   ├── content.ts  # TypeScript source file
│       │   │   │   │   ├── headings.ts  # TypeScript source file
│       │   │   │   │   ├── images.ts  # TypeScript source file
│       │   │   │   │   ├── index.ts  # TypeScript source file
│       │   │   │   │   ├── meta.ts  # TypeScript source file
│       │   │   │   │   ├── social.ts  # TypeScript source file
│       │   │   │   │   └── technical.ts  # TypeScript source file
│       │   │   │   ├── extract.ts  # TypeScript source file
│       │   │   │   ├── index.ts  # TypeScript source file
│       │   │   │   └── types.ts  # TypeScript source file
│       │   │   └── index.ts  # TypeScript source file
│       │   ├── screenshots/  # screenshot capture services
│       │   │   ├── axe/  # directory
│       │   │   │   ├── clip.ts  # TypeScript source file
│       │   │   │   ├── config.ts  # TypeScript source file
│       │   │   │   ├── context-snippet.ts  # TypeScript source file
│       │   │   │   ├── friendly-node.ts  # TypeScript source file
│       │   │   │   ├── highlight-css.ts  # TypeScript source file
│       │   │   │   ├── image.ts  # TypeScript source file
│       │   │   │   ├── location.ts  # TypeScript source file
│       │   │   │   ├── page-overview-overlay.ts  # TypeScript source file
│       │   │   │   ├── page-overview.ts  # TypeScript source file
│       │   │   │   ├── semantic-overlay.ts  # TypeScript source file
│       │   │   │   ├── targets.ts  # TypeScript source file
│       │   │   │   ├── types.ts  # TypeScript source file
│       │   │   │   └── violation-capture.ts  # TypeScript source file
│       │   │   └── AxeScreenshotService.ts  # TypeScript source file
│       │   ├── types/  # directory
│       │   │   └── lighthouse.d.ts  # TypeScript source file
│       │   ├── utils/  # scanner-runner utility helpers
│       │   │   ├── env.ts  # TypeScript source file
│       │   │   ├── html.ts  # TypeScript source file
│       │   │   ├── logger.ts  # TypeScript source file
│       │   │   ├── playwright.ts  # TypeScript source file
│       │   │   └── severity.ts  # TypeScript source file
│       │   ├── worker/  # worker bootstrapping helpers
│       │   │   └── worker-validation.ts  # TypeScript source file
│       │   ├── index.ts  # TypeScript source file
│       │   ├── lib.ts  # TypeScript source file
│       │   └── worker.ts  # TypeScript source file
│       ├── tests/  # directory
│       │   ├── ai/  # directory
│       │   │   ├── action-decision-parser.test.ts  # test file
│       │   │   ├── decision-prompt.test.ts  # test file
│       │   │   ├── goal-checker.test.ts  # test file
│       │   │   ├── json.test.ts  # test file
│       │   │   └── loop-detector.test.ts  # test file
│       │   ├── config/  # directory
│       │   │   ├── rule-behaviors.test.ts  # test file
│       │   │   ├── rule-titles.test.ts  # test file
│       │   │   └── user-impact.test.ts  # test file
│       │   ├── core/  # directory
│       │   │   ├── plugins/  # directory
│       │   │   │   ├── builtin-manifests.test.ts  # test file
│       │   │   │   ├── loader.test.ts  # test file
│       │   │   │   ├── plugin-discovery.test.ts  # test file
│       │   │   │   ├── plugin-load.test.ts  # test file
│       │   │   │   └── plugin-loader.test.ts  # test file
│       │   │   ├── storage-provider/  # directory
│       │   │   │   ├── content-type.test.ts  # test file
│       │   │   │   └── files.test.ts  # test file
│       │   │   ├── browser-manager.test.ts  # test file
│       │   │   ├── config-loader.test.ts  # test file
│       │   │   ├── event-publisher.contract.test.ts  # test file
│       │   │   ├── event-publisher.test.ts  # test file
│       │   │   ├── page-iterator.test.ts  # test file
│       │   │   ├── scanner-base.test.ts  # test file
│       │   │   ├── screenshots.integration.test.ts  # test file
│       │   │   ├── screenshots.test.ts  # test file
│       │   │   ├── storage-provider.test.ts  # test file
│       │   │   ├── target-validation.test.ts  # test file
│       │   │   └── web-server-formatter.test.ts  # test file
│       │   ├── scanners/  # directory
│       │   │   ├── ai-navigator/  # directory
│       │   │   │   └── options.test.ts  # test file
│       │   │   ├── lighthouse/  # directory
│       │   │   │   └── index.test.ts  # test file
│       │   │   ├── link-checker/  # directory
│       │   │   │   ├── index.test.ts  # test file
│       │   │   │   └── scanPage.test.ts  # test file
│       │   │   ├── security-headers/  # directory
│       │   │   │   └── index.test.ts  # test file
│       │   │   └── seo/  # directory
│       │   │       └── checks.test.ts  # test file
│       │   ├── screenshots/  # directory
│       │   │   ├── axe/  # directory
│       │   │   │   ├── clip.test.ts  # test file
│       │   │   │   ├── config.test.ts  # test file
│       │   │   │   ├── context-snippet.test.ts  # test file
│       │   │   │   ├── friendly-node.test.ts  # test file
│       │   │   │   ├── highlight-css.test.ts  # test file
│       │   │   │   ├── image.test.ts  # test file
│       │   │   │   ├── location.test.ts  # test file
│       │   │   │   ├── page-overview-overlay.test.ts  # test file
│       │   │   │   ├── page-overview.integration.test.ts  # test file
│       │   │   │   ├── page-overview.test.ts  # test file
│       │   │   │   ├── semantic-overlay.test.ts  # test file
│       │   │   │   ├── targets.test.ts  # test file
│       │   │   │   └── violation-capture.test.ts  # test file
│       │   │   └── AxeScreenshotService.test.ts  # test file
│       │   ├── utils/  # directory
│       │   │   ├── env.test.ts  # test file
│       │   │   ├── html.test.ts  # test file
│       │   │   ├── playwright.test.ts  # test file
│       │   │   └── severity.test.ts  # test file
│       │   ├── worker/  # directory
│       │   │   └── worker-validation.test.ts  # test file
│       │   └── tsconfig.json  # JSON data or configuration
│       ├── .dockerignore  # docker build ignore patterns
│       ├── AGENTS.md  # agent instructions and repository conventions
│       ├── bun.lock  # dependency lockfile
│       ├── Dockerfile  # container build definition
│       ├── eslint.config.mjs  # JavaScript module
│       ├── knip.json  # JSON data or configuration
│       ├── package.json  # Node package manifest
│       ├── prettier.config.mjs  # JavaScript module
│       ├── README.md  # project overview and quickstart
│       ├── tsconfig.build.json  # JSON data or configuration
│       ├── tsconfig.json  # JSON data or configuration
│       └── vitest.config.ts  # TypeScript source file
├── scripts/  # project automation scripts
│   ├── a11y/  # accessibility automation scripts
│   │   └── test-axe-local.js  # JavaScript source file
│   ├── tests/  # script-level test helpers
│   │   └── verify-justfile.test.sh  # test file
│   ├── build-images.sh  # shell script
│   └── quadlet-install.sh  # shell script
├── tests/  # cross-service end-to-end tests
│   ├── e2e/  # end-to-end test cases
│   │   ├── config_test.go  # Go test file
│   │   ├── go.mod  # Go module definition
│   │   ├── url_scan_helpers_test.go  # Go test file
│   │   ├── url_scan_test.go  # Go test file
│   │   ├── zip_scan_helpers_test.go  # Go test file
│   │   └── zip_scan_test.go  # Go test file
│   ├── fixtures/  # test fixtures and sample sites
│   │   ├── simple-site/  # directory
│   │   │   └── index.html  # HTML document
│   │   └── test-site.zip  # ZIP archive fixture
│   └── AGENTS.md  # agent instructions and repository conventions
├── tools/  # developer and operations CLI tools
│   ├── stageflow-cli/  # CLI client for submitting URL scan jobs
│   │   ├── client.go  # Go source file
│   │   ├── cmd_report.go  # Go source file
│   │   ├── cmd_report_test.go  # Go test file
│   │   ├── cmd_run.go  # Go source file
│   │   ├── cmd_run_test.go  # Go test file
│   │   ├── cmd_scanners.go  # Go source file
│   │   ├── cmd_scanners_test.go  # Go test file
│   │   ├── constants.go  # Go source file
│   │   ├── dev_stack.go  # Go source file
│   │   ├── filter.go  # Go source file
│   │   ├── filter_test.go  # Go test file
│   │   ├── go.mod  # Go module definition
│   │   ├── go.sum  # Go dependency checksums
│   │   ├── http_client_test.go  # Go test file
│   │   ├── local_targets.go  # Go source file
│   │   ├── main.go  # Go source file
│   │   ├── output.go  # Go source file
│   │   ├── output_test.go  # Go test file
│   │   ├── project_config.go  # Go source file
│   │   ├── project_mode.go  # Go source file
│   │   ├── project_root.go  # Go source file
│   │   ├── run.go  # Go source file
│   │   ├── sse.go  # Go source file
│   │   ├── tee_output.go  # Go source file
│   │   ├── test_helpers_test.go  # Go test file
│   │   ├── threshold.go  # Go source file
│   │   ├── threshold_test.go  # Go test file
│   │   └── types.go  # Go source file
│   ├── job-status-cli/  # CLI for querying job status
│   │   ├── api.go  # Go source file
│   │   ├── commands.go  # Go source file
│   │   ├── format.go  # Go source file
│   │   ├── format_test.go  # Go test file
│   │   ├── go.mod  # Go module definition
│   │   ├── job-status-cli  # executable binary or script
│   │   ├── main.go  # Go source file
│   │   ├── run.go  # Go source file
│   │   ├── run_test.go  # Go test file
│   │   ├── types.go  # Go source file
│   │   └── usage.go  # Go source file
│   ├── suite-runner/  # CLI for running scan suites
│   │   ├── api.go  # Go source file
│   │   ├── evaluate.go  # Go source file
│   │   ├── format.go  # Go source file
│   │   ├── go.mod  # Go module definition
│   │   ├── go.sum  # Go dependency checksums
│   │   ├── main.go  # Go source file
│   │   ├── poll_test.go  # Go test file
│   │   ├── run.go  # Go source file
│   │   ├── sse.go  # Go source file
│   │   ├── sse_test.go  # Go test file
│   │   ├── stream.go  # Go source file
│   │   ├── suite-runner  # executable binary or script
│   │   ├── suite.go  # Go source file
│   │   ├── suite.sample.yml  # YAML configuration
│   │   ├── suite_test.go  # Go test file
│   │   └── types.go  # Go source file
│   └── README.md  # project overview and quickstart
├── .dockerignore  # docker build ignore patterns
├── .editorconfig  # editor formatting defaults
├── .env  # local environment variables
├── .env.example  # example local environment variables
├── .env.staging  # local staging environment variables
├── .env.staging.example  # example staging environment variables
├── .gitignore  # git ignore patterns
├── .gitleaks.toml  # secret scanning configuration
├── .golangci.yml  # Go lint configuration
├── AGENTS.md  # agent instructions and repository conventions
├── CHANGELOG.md  # release and change history
├── CODE_OF_CONDUCT.md  # community behavior policy
├── CONTRIBUTING.md  # contribution workflow and standards
├── go.work  # Go workspace module links
├── go.work.sum  # Go workspace checksums
├── justfile  # task runner recipes
├── LICENSE  # project license
├── plan.md  # working plan notes
├── ralph.md  # agent workflow notes
├── README.md  # project overview and quickstart
├── SECURITY.md  # security policy and reporting process
├── SUPPORT.md  # support and troubleshooting notes
└── tsconfig.strict.json  # shared strict TypeScript rules
```
