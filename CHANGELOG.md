# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.0] - 2026-07-03

### Added

- `stageflow scan <dir|zip>`: point the CLI at a local build directory (or an
  existing ZIP archive) and it is zipped, uploaded to the platform's ZIP
  intake, served from the job's isolated static server, and scanned — local
  project scanning with zero local stack, including against the hosted API
- Web client test suite: vitest unit/component tests driven by the committed
  v2 report fixture, plus a Playwright e2e smoke against the preview build
  with a mocked API; both run in CI alongside a new web-client README
- CLI demo GIF in the README, recorded against the hosted API with a committed
  recording script (`devtools/qa/record-demo.sh`)

### Changed

- README now leads with a single golden path (CLI scan against the hosted
  demo); `PRODUCT.md` and `DESIGN.md` moved under `docs/`
- `devtools/qa/report-preview.mjs` uses repo-relative paths and the
  Playwright dependency declared in `clients/web`, so it runs from any checkout

- **Breaking (CLI):** redesigned the command grammar so each noun means one thing
  - The local dev-server scan loop moved from bare `stageflow project` to `stageflow dev scan`; `project init` and `project doctor` moved to `stageflow dev init` and `stageflow dev doctor`. Bare `stageflow dev` and `stageflow project` print help instead of acting
  - `project` now means only the remote API entity. The remote baseline/regression scan is unified as `stageflow project scan [slug]` — pass the slug explicitly, or omit it to use `stageflow.project` from `.stageflow/config.yaml`
  - `--scanners` (comma string) on `scan` is renamed to `--scanner` and unified with `project create`/`project update`: one flag, repeatable and comma-tolerant
- **Breaking (config):** `.stageflow/config.yaml` is now `version: 2` — `stageflow.remote_project` renamed to `stageflow.project`, `stageflow.remote_api_url` removed (use `stageflow.api_url`, `--api`, or `STAGEFLOW_API_URL`), and `scan.scanners` is a YAML list instead of a comma string
- **Breaking (JSON):** CLI envelope schemas `stageflow-cli/project-init@v1` and `project-doctor@v1` renamed to `dev-init@v1` and `dev-doctor@v1`; the doctor envelope's `hostedMemory` field is now `remoteProject` (with `slug` instead of `projectSlug`) and its `recommendedScanCommand` emits the new `stageflow project scan` grammar. `report@v1` and `project-scan@v1` are unchanged
- `stageflow ai` is hidden from help output (still functional; experimental)
- Internal: the CLI's flat `package main` (60 files) is decomposed into
  focused internal packages (`render`, `projectmode`, `authintake`,
  `exitcode`, `buildinfo`, `testsupport`); release ldflags now stamp
  `internal/buildinfo` instead of `main`
- Rewrote root and per-command help text around the three scan entry points (`scan <url>`, `dev scan`, `project scan`); exit-code semantics are unchanged (0 pass, 1 policy failure, 2 error)
- `docs/PROJECT_MODE.md` renamed to `docs/dev-mode.md` to match the new `dev` namespace

### Removed

- The repo-managed staging deployment mode (`just staging`, the staging
  compose/Caddy/env examples, and its docs); hosted deployment is managed
  outside this repository, and self-hosting uses the base compose files
- `stageflow project hosted` (superseded by slug-free `stageflow project scan`)
- `--project` flag on `stageflow scan` (use `stageflow project scan <slug>`)
- Deprecated `--json` global flag (use `--format json`) and the hidden `--fail-severity` alias (use `--fail-on`)

### Fixed

- Refreshed the project-scan golden fixtures from the hermetic CI stack: report
  doc v2.1.0 fields (page overview artifacts, `locationInfo`, real scanner
  `toolVersion`), localhost URLs from the local golden stack, and the rescored
  summary

- ZIP job pages are no longer blocked by the scanner-side private-target
  guard: the intake now marks ZIP jobs `allow_private_targets`, since their
  scan target is the job's own loopback static server, not a user-supplied
  URL (previously every page after the entry URL failed with
  "IP 127.0.0.1 is in a blocked network range")

- ZIP scans no longer fail at the extraction stage on rootless Podman
  deployments: the freshly created per-job workspace volume was owned by
  container-root, so the non-root extractor could not write to it. The
  orchestrator now mounts the workspace with Podman's `U` option, which
  chowns it to the extraction worker's runtime user at start

- Projects now honor `PLATFORM_API_ALLOW_PRIVATE_TARGETS` end to end: instances
  that opt in accept private/localhost project URLs (creation previously always
  enforced public-only validation), and project scan jobs carry the opt-in to
  scanner pods so the runner's own target guard no longer blocks those scans

## [0.2.0] - 2026-06-09

### Changed

- Consolidated the report UI: the standalone Scanners and page-overview views are merged into the Overview dashboard and a single `VisualReviewPanel` Pages overlay; report sections are now Overview, Issues, Pages, and Downloads
- Playground form-auth: auto-detect the login form and make the post-login success selector optional
- Generalized mobile "bento" styling and improved the scan-status layout
- Refreshed all README screenshots against a public demo target (Deque "Mars Commuter"), captured from the consolidated report UI
- Consolidated the four duplicated orchestrator PostgreSQL test harnesses into a single shared helper that caches the embedded-Postgres binary across runs and honors a `TEST_DATABASE_URL` override for running against an existing database
- Aligned CI Go version with the `go.mod` toolchain pin (1.26.4)
- Extracted MinIO bucket/app-user provisioning into a shared `provision.sh` used by every compose overlay
- Isolated the default local compose stack under its own project and network (`stageflow_dev` / `stageflow_dev_net`)
- Rewrote the README and remote-projects guide to lead with the platform workflow; added per-service READMEs and an evaluator guide

### Added

- Panic recovery on the orchestrator container-monitor goroutine so a monitor panic can no longer crash the process
- Test coverage for the CLI SSE wire-format parser and the report-aggregator guard paths

### Fixed

- Keep small issue boxes clickable under full-page overlays in the report
- `scan.completed` timing no longer exceeds the reported `total_ms`
- Made the platform-api auth-disabled startup warning explicit and unmissable so it cannot be silently overlooked
- MinIO policy volume mount uses the `:z` SELinux label so local provisioning works on SELinux-enforcing hosts (e.g. Fedora)
- Web container builds no longer fail prerendering when `VITE_SITE_URL` (or other site vars) arrive as empty build args — blank values now fall back to defaults
- The CLI now cross-compiles for Windows: Project Mode's dev-server process-group handling moved behind platform build tags instead of Unix-only `syscall` calls

## [0.1.0] - 2026-05-28

Initial public release.

### Scanners

- `axe`: accessibility checks via axe-core with page-level evidence and screenshots
- `lighthouse`: performance, best-practices, and PWA signals
- `seo`: title, meta, heading, and structured-data validation
- `security-headers`: HTTP response-header policy checks
- `link-checker`: in-page link health and cross-page reachability
- `open-graph`: social-preview metadata (og:\*, Twitter cards)
- `spelling-grammar`: rule-based spelling and content-quality checks
- `ai-navigator`: agent-driven navigation traces for guided audits

### Surfaces

- SvelteKit web app for scan submission, live status, and report exploration
- Go CLI (`stageflow`) for terminal-first scans, Project Mode, JSON output, and severity-based exit codes
- Platform API for URL/ZIP intake, projects, reports, diffing, and SSE streaming
- Authenticated scanning: interactive Playwright capture and deterministic form recipes flow through the runtime so every scanner runs against the post-login surface
- Auth-wall detector emits a unified report issue when a scan lands on a login redirect, login form, or recognized 401/403 page

### Orchestration

- Job FSM driven by NATS JetStream with durable consumers and event replay
- Per-job Podman pod isolation for scanner runtime, archive extractor, and orchestrator
- MinIO-backed artifact storage with presigned access, PostgreSQL-backed orchestrator job/event state, and SQLite-backed Platform API project metadata

### Contracts

- JSON Schema contracts in `libs/contracts/` with generated Go and TypeScript types
- Stable, content-derived issue IDs so reports compare cleanly against promoted baselines

### Observability

- Authenticated Prometheus-compatible `/metrics` endpoint on the orchestrator admin API: job-state and Podman pod gauges, `event_handled_total{event,status}` counters, an event-handler duration histogram, and `http_requests_total{status}` counters — collected in-process with no metrics-client dependency
- Structured panic recovery and per-client rate limiting on the orchestrator admin API
- Grafana dashboards (optional) for job overview and provenance validation

### Verification

- Go services: `go build`, `go test -race`, `golangci-lint`, `govulncheck`
- Web app: Vitest unit tests, Storybook interaction and accessibility checks, type-check, lint
- Scanner runner: Bun-based unit and integration suites, including real-browser authenticated flows
- Golden regression flow (`qa/e2e/project-scan-golden.sh`) covering baseline → promote → regression diff with exit-code assertions
- CI includes secrets scanning (gitleaks), SBOM generation, and Trivy image scanning
