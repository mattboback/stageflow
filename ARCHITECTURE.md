# StageFlow Architecture

StageFlow is a self-hosted frontend quality gate built from Go services, a
TypeScript scanner runtime, a SvelteKit web UI, and a Go CLI. The system is
designed around safe intake, isolated execution, deterministic orchestration,
and one normalized report contract for accessibility, SEO, security,
performance, and content checks.

For the full architecture guide, read
[docs/architecture/system.md](docs/architecture/system.md). It covers service
boundaries, event flow, trust boundaries, storage, deployment topology, failure
modes, and the code reference map.

## System Shape

| Area                         | Responsibility                                                                             |
| ---------------------------- | ------------------------------------------------------------------------------------------ |
| `clients/web`                | SvelteKit UI for scan submission, live status, and report review                           |
| `clients/cli`                | Go CLI for terminal-first scans, Project Mode, JSON output, and regression gating          |
| `services/platform-api`      | Public API boundary for URL/ZIP intake, SSRF validation, projects, reports, diffs, and SSE |
| `services/orchestrator`      | Job state machine, NATS event consumption, Podman pod lifecycle, and report aggregation    |
| `services/archive-extractor` | Safe ZIP extraction, provenance generation, and static serving for ZIP jobs                |
| `services/scanner-runner`    | TypeScript/Bun scanner runtime with Playwright, artifacts, and contract validation         |
| `libs/contracts`             | JSON Schema contracts and generated Go/TypeScript types                                    |

## Core Decisions

- **Server-Sent Events over WebSocket** for one-way, proxy-friendly job status
  streams.
- **NATS JetStream** for durable job, extraction, and scan events.
- **Podman job pods** for per-job isolation instead of a shared long-lived
  scanner worker.
- **Schema-first contracts** so Go services, the CLI, the web UI, and scanners
  share the same report and scanner-manifest shapes.
- **Stable content-derived issue IDs** so scans can be compared against
  promoted baselines.

## Start Here

- [docs/architecture/system.md](docs/architecture/system.md) for the full
  system design.
- [docs/evaluators-guide.md](docs/evaluators-guide.md) for a 5-15 minute review
  path through the highest-signal parts of the repo.
- [docs/reference/configuration.md](docs/reference/configuration.md) for
  environment variables and local/self-hosted deployment topology.
- [docs/operations/deployment.md](docs/operations/deployment.md) for local,
  staging, self-hosted, and hosted-demo deployment boundaries.
