# StageFlow

[![CI](https://github.com/mattboback/stageflow/actions/workflows/ci.yml/badge.svg)](https://github.com/mattboback/stageflow/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

**Live demo:** [stageflow.org](https://stageflow.org) | **Run locally:** `cp .env.example .env && just diagnose && just demo`

Open-source, self-hosted frontend quality gate for accessibility, SEO, security, performance, and content checks, designed to give developers and AI agents fast regression-aware feedback.

StageFlow accepts live URLs or static-site ZIP archives, runs eight scanners, streams execution over Server-Sent Events, and merges heterogeneous results into one normalized report for both the web UI and the Go CLI. The long-term product center is the CLI: run StageFlow after a frontend edit, get structured terminal output, and use hosted project baselines and diffs to decide whether the change regressed quality.

This repository is designed to be both runnable and reviewable. Start from `.env.example`, replace every `change-me` value before using a public domain, and treat `stageflow.org` references as project examples rather than defaults to reuse unchanged.

![StageFlow dashboard and scan pipeline](docs/images/hero.png)

## Start Here

- **Fastest smoke test:** `cp .env.example .env && just diagnose && just demo`
- **Reviewer path (5–15 minutes):** [docs/evaluators-guide.md](docs/evaluators-guide.md)
- **Contributor setup:** [CONTRIBUTING.md](CONTRIBUTING.md)
- **System design entrypoint:** [ARCHITECTURE.md](ARCHITECTURE.md)
- **System design deep dive:** [docs/architecture/system.md](docs/architecture/system.md)
- **High-signal code entrypoints:** [Web app](clients/web/README.md), [Platform API](services/platform-api/README.md), [Scanner Runner](services/scanner-runner/README.md), [CLI](clients/cli/README.md)

## Why This Repo Is Worth Reviewing

- **Real distributed system** — Go services, NATS JetStream, Podman-isolated job execution, MinIO, PostgreSQL, and SQLite
- **Contract-driven integration** — shared JSON Schema contracts generate types for Go and TypeScript consumers
- **Two client surfaces over one backend** — a SvelteKit web app and a Go CLI both consume the same Platform API
- **Strong verification story** — Go race tests, Vitest, Storybook interaction and accessibility checks, end-to-end flows, and golden regression tests
- **Security-aware by default** — SSRF protections, archive safety checks, rootless containers, API middleware, and secrets scanning in CI

## What StageFlow Does

StageFlow solves the problem of **gating frontend changes with one regression-aware quality check**.

It is meant to answer a practical question after an edit:

> Did this change make accessibility, performance, SEO, security, or content quality worse?

It supports two input modes:

1. **URL scans** — scan one or more live URLs
2. **ZIP scans** — upload a static-site archive, extract it safely, discover HTML, and scan it locally

The system normalizes scanner output into one contract-driven report with:

- unified severity scoring
- stable issue IDs for regression diffing
- page-level evidence including screenshots
- scanner summaries and per-page rollups
- streaming job progress for the browser and CLI

The most important product loop is:

1. Make a frontend change locally.
2. Run StageFlow from the terminal.
3. Compare against a baseline or promoted project state.
4. Let the CLI tell you whether the edit passed, regressed, or needs review.

## Architecture at a Glance

| Surface / service            | Responsibility                                                                       |
| ---------------------------- | ------------------------------------------------------------------------------------ |
| `clients/web`                | SvelteKit UI for job submission, live status, and report exploration                 |
| `clients/cli`                | Go CLI for automation, CI gating, Project Mode, and JSON output                      |
| `services/platform-api`      | URL/ZIP intake, project CRUD, report APIs, diffing, SSE hub, and API boundary        |
| `services/orchestrator`      | Job state machine, NATS event handling, Podman pod lifecycle, and report aggregation |
| `services/archive-extractor` | Safe ZIP extraction, provenance generation, and static file serving inside a job pod |
| `services/scanner-runner`    | TypeScript/Bun runtime that loads scanners, captures artifacts, and publishes events |
| `libs/contracts`             | JSON Schema contracts with generated Go and TypeScript types                         |

The design decisions worth inspecting first are:

- **SSE over WebSocket** for simple, proxy-friendly job progress streaming
- **NATS JetStream** for durable orchestration events and replay after restarts
- **Typed event envelopes** with lenient envelope parsing, strict payload parsing, event metadata propagation, and explicit ACK/NAK behavior
- **Podman per-job isolation** instead of long-lived shared worker containers
- **Schema-first contracts** so scanners, API, CLI, and web UI all share the same report shape
- **Stable content-based issue IDs** so baseline diffing works across reruns

## Environment Modes

| Mode                                  | Primary use                                           | Web                       | API                       | Grafana                   | Source of truth                                                              |
| ------------------------------------- | ----------------------------------------------------- | ------------------------- | ------------------------- | ------------------------- | ---------------------------------------------------------------------------- |
| `dev` via `just demo` / `just dev up` | Fastest local smoke test                              | `http://localhost:3000`   | `http://localhost:8080`   | `http://localhost:3001`   | `infra/compose/podman-compose.yml` + `infra/compose/podman-compose.test.yml` |
| `local` overlay                       | Scan localhost/private targets during development     | `http://localhost:3010`   | `http://localhost:8080`   | `http://localhost:3001`   | `infra/compose/podman-compose.local.yml`                                     |
| repo-managed staging overlay          | Domain-like staging on alternate loopback ports       | `http://127.0.0.1:3300`   | `http://127.0.0.1:8300`   | `http://127.0.0.1:3301`   | `infra/compose/podman-compose.staging.yml`                                   |
| optional self-hosted edge             | Public-domain routing and TLS for your own deployment | proxied by host Caddy     | proxied by host Caddy     | proxied by host Caddy     | `infra/caddy/Caddyfile`                                                      |
| hosted `stageflow.org` production     | Shared public demo                                    | managed outside this repo | managed outside this repo | managed outside this repo | external deployment control plane                                            |

These modes intentionally use different port layouts. The self-hosted Caddy edge proxies to bridge-bound service ports (`3100` frontend, `8100` API, `3101` Grafana, `9100` MinIO), while the local and staging overlays expose different loopback ports for developer convenience. Use one topology per environment rather than mixing configs across them.

The hosted `stageflow.org` demo uses the same application code, but its
production release, verification, monitoring, and rollback process is managed
outside this public repository.

## Quick Start

### Prerequisites

- [Go 1.26.3](https://go.dev/dl/)
- [Node.js 22](https://nodejs.org/)
- [Bun](https://bun.sh/)
- [Podman](https://podman.io/) with `podman compose`
- [just](https://github.com/casey/just)
- [golangci-lint v2](https://golangci-lint.run/)

### Fastest local smoke test

```bash
git clone https://github.com/mattboback/stageflow.git
cd stageflow
cp .env.example .env
just diagnose
just demo
```

`just demo` runs the full bootstrap: prerequisite checks, dependency install, image builds, stack restart, health waits, and MinIO initialization.

### Manual bootstrap

```bash
just setup
just images
just dev up
just dev init
```

### Run a first scan

```bash
just cli-install
stageflow scan https://example.com
```

To scan `localhost` or other private targets during development:

```bash
just setup && just images && just dev up local && just dev init local
stageflow project init
stageflow project doctor .
stageflow project .
```

`.stageflow/config.yaml` still drives a local Project Mode run, but it can now
also carry the hosted project link you use for baseline memory via
`stageflow.remote_project` and `stageflow.remote_api_url`. That keeps the
terminal loop as: edit locally, run `stageflow project`, then follow with
`stageflow scan --project <slug> --api https://stageflow.org` when you want the
hosted regression-memory step.

### Self-hosting notes

- Start from `.env.example`; never commit `.env`, `.env.staging`, or real credentials
- Replace every `change-me` value before using a public domain or shared environment
- Set `STAGEFLOW_PUBLIC_DOMAIN`, `PLATFORM_API_CORS_ALLOW_ORIGINS`, `GF_SERVER_ROOT_URL`, and the frontend `VITE_*` URLs for your deployment
- `infra/caddy/Caddyfile` is an optional host-level edge example for self-hosted public domains and TLS
- Public self-hosted scanner deployments should pair runtime URL validation with host/container egress controls; see [infra/security/egress-policy.example.md](infra/security/egress-policy.example.md)
- The hosted `stageflow.org` demo uses the same application code, but its production control plane is intentionally managed outside this repository

For detailed environment variables and overlay-specific notes, see [docs/reference/configuration.md](docs/reference/configuration.md) and [docs/operations/deployment.md](docs/operations/deployment.md).

## Screenshots

<table>
  <tr>
    <td><img src="docs/images/scan-in-progress.png" alt="Live scan progress with scanner status and event streaming" /></td>
    <td><img src="docs/images/report-overview.png" alt="Unified report overview with score, issue totals, and scanner summary" /></td>
  </tr>
  <tr>
    <td align="center"><em>Live job progress streamed over SSE</em></td>
    <td align="center"><em>Contract-driven unified report output</em></td>
  </tr>
</table>

## Security and Operations Posture

- URL intake enforces SSRF protections and private-target controls
- ZIP handling is isolated into a dedicated extractor with archive safety limits
- Scan execution runs inside rootless Podman job pods
- API boundaries use middleware for request IDs, logging, panic recovery, CORS, auth, and rate limiting where appropriate
- CI includes secrets scanning, `govulncheck`, Go race tests, web tests, Storybook interaction checks, and image builds

## Testing Strategy

The project uses layered verification rather than a single happy-path build:

CI-backed checks:

- **Go services:** `go build ./...`, `go test -race ./...`, `golangci-lint run`, `govulncheck ./...`
- **Web app:** `bun run ci` in `clients/web` for format checks, strict linting, type checks, and coverage-backed unit tests
- **Storybook:** separate CI job builds Storybook and runs interaction/accessibility checks with the Storybook test runner
- **Scanner Runner:** `bun run ci` in `services/scanner-runner` for format checks, strict linting, type checks, and coverage-backed tests
- **Containers/security:** image builds, SBOM generation, Trivy scanning, `bun audit`, and gitleaks
- **Dead code:** web and scanner-runner dead-code analysis currently runs as a non-blocking CI job

Acceptance checks:

- **Golden regression flow:** `qa/e2e/project-scan-golden.sh` runs baseline, promote, regression diff, and exit-code assertions. It is wired to scheduled/manual GitHub Actions and is available locally with `just project-golden`.

## Agent-facing workflow

StageFlow is increasingly optimized for **terminal-first quality gating**:

- **Local loop:** `stageflow project init`, `stageflow project doctor`, then `stageflow project --format json` to run a local dev-server scan with structured output. `.stageflow/config.yaml` can optionally record the hosted project slug for the follow-up regression-memory step.
- **Setup loop:** `stageflow project init --format json` and `stageflow project doctor --format json` let agents bootstrap and validate project wiring with parseable terminal output, including the hosted project association when one is configured.
- **Hosted regression memory:** hosted baselines still run in a separate StageFlow API context. After the local loop, run `stageflow scan --project <slug> --format json --api https://stageflow.org` against the associated hosted project to get one parseable envelope with the current report plus baseline diff metadata.
- **Automation decision:** agents can inspect exit codes for pass/fail and parse JSON output to decide whether to stop, retry, or escalate.

## Where Reviewers Should Look First

If you want the strongest engineering signals quickly, start here:

1. [services/orchestrator](services/orchestrator) — explicit job FSM, NATS-driven execution, and E2E-style tests
2. [services/scanner-runner](services/scanner-runner/README.md) — scanner plugin runtime, Playwright integration, artifact handling, and contract enforcement
3. [libs/contracts](libs/contracts) — schema-first design and generated cross-language types
4. [clients/cli](clients/cli/README.md) — streaming CLI UX, Project Mode, JSON output, and severity-based exit codes
5. [clients/web](clients/web/README.md) — report UX, live status flows, accessibility checks, and component tests
6. [docs/architecture/system.md](docs/architecture/system.md) — trust boundaries, data flow, failure modes, and topology

For a guided 5–15 minute walkthrough, use [docs/evaluators-guide.md](docs/evaluators-guide.md).

## Docs

| Document                                                               | Description                                                               |
| ---------------------------------------------------------------------- | ------------------------------------------------------------------------- |
| [docs/evaluators-guide.md](docs/evaluators-guide.md)                   | Structured review path for hiring managers and senior engineers           |
| [ARCHITECTURE.md](ARCHITECTURE.md)                                     | Short root entrypoint to the architecture material                        |
| [docs/architecture/system.md](docs/architecture/system.md)             | Full system design, service boundaries, event flow, and threat model      |
| [docs/reference/configuration.md](docs/reference/configuration.md)     | Environment variables, compose overlays, and topology guidance            |
| [docs/operations/deployment.md](docs/operations/deployment.md)         | Local, self-hosted, and hosted-demo deployment boundaries                 |
| [clients/web/README.md](clients/web/README.md)                         | Frontend routes, architecture, commands, and tests                        |
| [services/platform-api/README.md](services/platform-api/README.md)     | Intake API boundary, routes, middleware, and local verification           |
| [services/scanner-runner/README.md](services/scanner-runner/README.md) | Scanner runtime responsibilities, plugin loading, outputs, and validation |
| [clients/cli/README.md](clients/cli/README.md)                         | CLI install, commands, and output formats                                 |
| [docs/PROJECT_MODE.md](docs/PROJECT_MODE.md)                           | Local dev server lifecycle scanning                                       |
| [docs/operations/devtools.md](docs/operations/devtools.md)             | Repo-local tooling and QA helpers                                         |
| [docs/operations/cli_cheatsheet.md](docs/operations/cli_cheatsheet.md) | Common CLI workflows                                                      |

## Support

- Use repository issue templates for bug reports and setup questions
- Use [SECURITY.md](SECURITY.md) for private vulnerability reporting

## License

[MIT](LICENSE)
