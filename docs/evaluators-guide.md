# StageFlow Evaluator Guide

This guide is for hiring managers, staff+ engineers, and other reviewers who want to quickly understand what StageFlow is and how to evaluate the codebase as a portfolio project.

If you have 5–15 minutes, follow the paths below. Each one points to concrete files and directories in this repository.

---

## 1. Product and Architecture at a Glance

- Start with the root README for the high-level picture:
  - `README.md` — screenshots, quick local setup, and a short architecture sketch.
- For system design and boundaries:
  - `docs/architecture/system.md` — service map, job lifecycle, contracts, and trust boundaries.

What to look for:

- Clear separation between the client surfaces (web UI and CLI) and the backend services.
- Use of contracts (`libs/contracts/*`) to keep scanners and consumers decoupled.
- Emphasis on isolation (per-job pods) and SSRF/archive guardrails.

---

## 2. Interesting Surfaces

Depending on your focus, you can dive into different parts of the system.

### 2.1 CLI and Developer Experience

- Entry points:
  - `clients/cli` — Go CLI implementation.
  - `clients/cli/README.md` — concepts, examples, JSON envelope, and Project Mode.
  - `docs/reference/cli/stageflow/*.md` — generated reference for each subcommand.

Highlights:

- Streaming scan status over SSE and structured JSON output for automation.
- Support for project baselines and regression diffing.
- Local Project Mode that drives a dev server lifecycle from the CLI.

### 2.2 Orchestrator and Job Lifecycle

- Entry points:
  - `services/orchestrator/internal/domain/jobs` — job model and state machine.
  - `services/orchestrator/internal/application/jobs` — application services and policies.
  - `services/orchestrator/internal/api` — admin API for inspection and tests.
  - `services/orchestrator/test` — E2E-style tests for job flow.

Highlights:

- Explicit job FSM and state transitions.
- Use of NATS JetStream for job events and coordination.
- Test coverage that exercises end-to-end flows without relying on a real Podman or Postgres.

### 2.3 Scanner Runner and Contracts

- Entry points:
  - `services/scanner-runner` — TypeScript/Bun runtime and Playwright-driven scanners.
  - `services/scanner-runner/tests` — tests for scanner pipeline, runtime contracts, and integrations.
  - `libs/contracts/report` — unified report schema and generated types.
  - `libs/contracts/scanner-manifest` — scanner manifest schema.

Highlights:

- Contract-driven normalization of eight different scanners into a single report shape.
- Use of Playwright for browser automation and screenshot evidence.
- Optional AI-powered scanners with clear boundaries around configuration (for example `OPENROUTER_API_KEY`).

### 2.4 Web App and Accessibility Testing

- Entry points:
  - `clients/web` — SvelteKit 5 web app.
  - `clients/web/tests/unit` — Vitest unit tests for stores, utilities, and components.
    Highlights:

- Focus on report UX, evidence visualization, and live job status.
- Storybook-based interaction and accessibility tests in CI.

---

## 3. Testing and Quality Gates

- CI workflows:
  - `.github/workflows/ci.yml` — combined Go, web app, Storybook, and Scanner Runner jobs.
  - `.github/workflows/release-stageflow-cli.yml` — multi-platform CLI release pipeline.

- Go quality gates:
  - `go test -race ./...` and `golangci-lint` across modules.
  - `govulncheck ./...` for vulnerability scanning.

- Web app and Scanner Runner:
  - `bun run ci` in `clients/web` and `services/scanner-runner` (typecheck, lint, tests).
  - Storybook interaction + axe-based a11y tests.

- End-to-end verification:
  - `qa/e2e/project-scan-golden.sh` and `qa/fixtures/project-golden/*` — project baseline and regression golden test.

What to look for:

- How tests are layered: unit, integration, E2E, and shell-based.
- How configuration and secrets are handled in tests (for example, test keys vs real API keys).

---

## 4. Operations

- Operations and tooling:
  - `justfile` — main dev and CI helper commands.
  - `docs/operations/devtools.md` — CLI and repo-local tooling (job-status CLI, QA suite-runner).
  - `AGENTS.md` — notes about the separate production deployment workspace.

What to look for:

- How local development is protected from accidental production changes (for example, guardrails around the `stageflow.org` environment).

---

## 5. Key Design Decisions Worth Examining

These decisions shaped the system most. Each is a good place to dig deeper.

### Podman over Docker

StageFlow uses rootless Podman for scanner pod isolation. Rootless containers run without a daemon and without privilege escalation. This simplifies the security model for self-hosted deployments: scanner containers cannot affect the host even if compromised.

### NATS JetStream over a traditional job queue

The API publishes job events to NATS JetStream and the orchestrator subscribes. JetStream provides durable streams with replay, so the orchestrator can recover missed events after a restart without database polling. This decouples submission from execution and avoids the API needing to know anything about the orchestrator's runtime state.

### SSE over WebSocket for job streaming

Server-sent events are unidirectional, work through most proxies and CDNs without configuration, and are sufficient for a job progress stream. The platform API maintains an in-memory SSE hub per job with event buffering for reconnecting clients. SSE keeps the protocol simple; WebSocket would add bidirectional complexity for no benefit here.

### Contract-driven report normalization

All eight scanners produce findings that conform to one JSON Schema (`libs/contracts/report`). Generated Go and TypeScript types are the only types either side uses. The web app, CLI, and diff engine never contain scanner-specific branches. Adding a new scanner is safe: it cannot break existing consumers.

### Stable content-based issue IDs

Each issue gets a stable hash derived from its content. This is what makes the baseline diff engine work: when you rescan a page, issues with the same ID are "unchanged", new IDs are "regressions", and missing IDs are "fixes". Without stable IDs the diff would be meaningless.

### Dual surface (web UI + CLI) from one API

The same platform API drives both the SvelteKit web UI and the Go CLI. The CLI adds `--fail-on` severity gating and `--format json` for CI pipelines. This demonstrates how the same backend can serve different clients with very different output requirements without duplicating logic.

---

## 6. How This Project Represents My Work

This is a solo-authored project. Everything in the repo — system design, Go services, TypeScript runtime, SvelteKit web app, CLI, infra, tests, and docs — was built and maintained by one person.

The most interesting parts to look at for engineering depth:

1. `services/orchestrator` — explicit job FSM, NATS-driven state transitions, E2E test harness with mock Podman.
2. `services/scanner-runner` — abstract `ScannerBase`, plugin system, Playwright integration, AI scanner boundaries.
3. `libs/contracts` — schema-first design, codegen for two languages, migration docs.
4. `clients/cli` — SSE streaming, Project Mode lifecycle, structured JSON output, `--fail-on` quality gate.
5. `qa/e2e/project-scan-golden.sh` — golden regression test for the full scan → baseline → diff pipeline.
6. `.github/workflows/ci.yml` — layered CI across Go, web app, Scanner Runner, and Storybook.

If you have questions about any part of the system, open an issue with the `question` label or reach out via the contact on my GitHub profile.
