# StageFlow Architecture

StageFlow is a self-hostable frontend quality platform. It accepts URLs or static-site ZIP
archives, runs a scanner pipeline in isolated containers, normalizes results into one report
contract, and exposes the result through both a browser UI and a Go CLI.

For a fast reviewer path, start with the [repository README](../../README.md) and then use
this document as the system map.

## System goals

1. **Safe intake** — treat every URL and uploaded archive as untrusted input.
2. **Isolated execution** — run extraction and scanners in scoped rootless Podman jobs.
3. **Deterministic orchestration** — move jobs through an explicit finite-state machine.
4. **Actionable output** — merge scanner findings into one stable, diffable report.
5. **CI usefulness** — remember promoted baselines so a later scan can report regressions.

## Platform shape

```text
Web UI / Go CLI
      │
      ▼
Platform API ── job.created ──► NATS JetStream
      ▲                              │
      │                              ▼
      │                         Orchestrator
      │                              │
      │                              ▼
      │                 per-job Podman pod / containers
      │                 ├─ archive-extractor for ZIP jobs
      │                 └─ scanner-runner for URL or extracted-site scans
      │                              │
      ▼                              ▼
SQLite status/project store     MinIO artifacts + PostgreSQL job state
```

## Primary repository areas

| Path | Responsibility |
| --- | --- |
| `services/platform-api` | HTTP intake, SSRF guard, auth middleware, project CRUD, SSE status, artifact redirects |
| `services/orchestrator` | Job FSM, NATS consumers, Podman lifecycle, aggregation, PostgreSQL persistence |
| `services/archive-extractor` | ZIP validation, traversal protection, provenance generation, static serving |
| `services/scanner-runner` | TypeScript/Bun scanner runtime, browser automation, plugin execution, artifact upload |
| `clients/web` | React Router browser UI for scan submission, status, and report review |
| `clients/cli` | Go CLI for terminal scans, project/baseline workflows, JSON and Markdown output |
| `libs/contracts` | JSON Schemas plus generated Go and TypeScript contract types |
| `libs/go/*` | Shared Go packages for messaging, config, models, storage, and scanner catalogs |

## Runtime services

| Service | Runtime | Role |
| --- | --- | --- |
| `platform-api` | Go | Public API boundary and local status projection |
| `orchestrator` | Go | Durable job controller and report aggregation |
| `archive-extractor` | Go | Safe ZIP extraction and provenance |
| `scanner-runner` | TypeScript/Bun/Playwright | Scanner execution runtime |
| `frontend-react` | React Router static build served by nginx | Browser UI |
| `nats` | NATS JetStream | Durable event streams |
| `minio` | S3-compatible object storage | Scanner outputs, reports, screenshots |
| `postgres` | PostgreSQL | Orchestrator job/event state |
| `grafana` | Grafana | Optional local observability surface |

## Job lifecycle

The orchestrator owns the job state machine:

```text
PENDING
  ├─ URL job ─────────────► READY_TO_SCAN
  └─ ZIP job ─► EXTRACTING ─► READY_TO_SCAN

READY_TO_SCAN ─► SCANNING ─► COMPLETING ─► DONE
       │             │             │
       └─────────────┴─────────────┴──► FAILED
```

Important properties:

- state transitions are explicit and tested;
- scanner completion is tracked per expected scanner, not by elapsed time;
- every terminal job publishes a final status event;
- aggregated reports are uploaded as artifacts instead of being stored inline in the API.

## URL scan flow

1. The web UI or CLI submits `POST /api/v1/jobs/urls`.
2. Platform API validates request shape, scanner IDs, target URL schemes, and private-address policy.
3. Platform API publishes `job.created` to NATS JetStream and seeds its local status projection.
4. Orchestrator consumes the event, creates the per-job Podman pod, and starts scanner-runner containers.
5. Scanner Runner emits page and scanner completion events while writing artifacts to MinIO.
6. Orchestrator waits for the expected scanners, downloads their results, aggregates a unified report, and publishes `job.completed`.
7. Platform API updates connected SSE clients and redirects report/result requests to the stored artifacts.
8. CLI and web clients render the same unified report contract.

## ZIP scan flow

ZIP scans add an extraction phase before scanning:

1. Platform API validates the multipart request and stores the ZIP in MinIO.
2. Orchestrator starts `archive-extractor` in the job pod.
3. Archive Extractor enforces entry-count, total-size, per-entry-size, compression-ratio, and path-traversal limits.
4. Archive Extractor discovers HTML pages, writes `provenance.json`, and serves the extracted site inside the job network.
5. Scanner Runner scans the internal static-server URLs using the same scanner pipeline as URL jobs.

## Remote projects and baseline memory

Remote projects are named API records containing URLs, selected scanners, and optional auth/scanner settings.
A promoted job becomes the baseline for that project. Later project scans compare stable issue IDs against the
baseline and mark new findings as regressions, fixed findings as resolved, and matching findings as unchanged.

This is the path that makes StageFlow useful in CI: `stageflow project scan` can exit non-zero when regressions
appear or when a severity gate is exceeded.

## Unified report contract

Scanner outputs are normalized into `libs/contracts/report/schema/unified-report.v2.schema.json`.
The report includes:

- job metadata and scanner summaries;
- page summaries, screenshots, and page-overview evidence;
- issue details with stable content-derived IDs;
- normalized severity/category fields;
- occurrence-level evidence such as selectors, snippets, and bounding boxes;
- artifacts and scanner/global errors.

Generated Go and TypeScript types keep the API, orchestrator, CLI, scanner runner, and web UI aligned.

## Scanner system

Built-in scanners are cataloged and validated through the scanner manifest contract:

- `axe`
- `lighthouse`
- `seo`
- `security-headers`
- `link-checker`
- `open-graph`
- `spelling-grammar`
- `ai-navigator`

The scanner registry supports strict resolution for API validation and lenient catalog lookup for discovery. Scanner
Runner owns browser automation, scanner plugin execution, result writing, and artifact upload.

## Security model

The highest-risk surfaces are target intake, archive extraction, browser automation, and artifact access.
The main controls are:

- URL scheme and private-address validation at the API boundary;
- optional explicit private-target opt-in for trusted local/self-host scenarios;
- ZIP bomb, size, and path-traversal protections in archive extraction;
- rootless Podman job isolation with resource limits and no-new-privileges;
- scanner identity and scanner-option validation against manifests;
- API-key auth by default for API routes, with local-only auth disablement;
- secrets kept out of repo-owned examples and checked with gitleaks in CI.

## Client architecture

### Web UI

`clients/web` is a React Router app built into static assets and served by nginx in the production image.
It focuses on:

- scan submission and live status;
- report overview and severity distribution;
- issue grouping and occurrence evidence;
- visual/page review overlays;
- JSON/HTML artifact access.

### CLI

`clients/cli` is a Go binary that submits scans, consumes SSE, fetches the unified report, and renders text,
Markdown, or JSON. It also owns the developer loop (`stageflow dev`) and remote project workflows
(`stageflow project create|scan|promote|update|delete`).

## Testing and CI

The repository uses layered checks:

- Go build/lint/test/race/vulnerability checks across workspace modules;
- web lint, typecheck, and build;
- scanner-runner lint/typecheck/unit/integration checks;
- workflow linting and stale-vocabulary checks;
- gitleaks secret scanning;
- container image builds, SBOM generation, and Trivy scanning;
- golden project-scan regression coverage for the baseline/diff workflow.

## Reviewer code map

Good places to evaluate the engineering depth:

| Topic | Start here |
| --- | --- |
| URL intake and SSRF protection | `services/platform-api/internal/api/security.go` |
| Job FSM | `services/orchestrator/internal/domain/jobs/transitions.go` |
| NATS event handling | `libs/go/messaging` and orchestrator consumers |
| Report aggregation | `services/orchestrator/internal/application/report` |
| Scanner runtime | `services/scanner-runner` |
| Unified schema | `libs/contracts/report` |
| CLI streaming and rendering | `clients/cli/internal/jobstream` and `clients/cli/report_output.go` |
| Web report UI | `clients/web/app/components/report` |
