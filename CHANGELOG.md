# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Consolidated the report UI: the standalone Scanners and page-overview views are merged into the Overview dashboard and a single `VisualReviewPanel` Pages overlay; report sections are now Overview, Issues, Pages, and Downloads
- Playground form-auth: auto-detect the login form and make the post-login success selector optional
- Generalized mobile "bento" styling and improved the scan-status layout
- Refreshed all README screenshots against a public demo target (Deque "Mars Commuter"), captured from the consolidated report UI

### Removed

- Removed the internal AlchemizeCV dogfooding harness, its live integration test, the `dogfood-alchemizecv` justfile recipe, and all documentation references

### Fixed

- Keep small issue boxes clickable under full-page overlays in the report
- `scan.completed` timing no longer exceeds the reported `total_ms`

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
