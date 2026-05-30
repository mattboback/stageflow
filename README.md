# StageFlow

[![CI](https://github.com/mattboback/stageflow/actions/workflows/ci.yml/badge.svg)](https://github.com/mattboback/stageflow/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

**Live demo:** [stageflow.org](https://stageflow.org) &nbsp;|&nbsp; **Run locally:** `cp .env.example .env && just diagnose && just demo`

---

StageFlow is an open-source, self-hosted **frontend quality gate** that answers one practical question after every code change:

> *Did this edit make accessibility, performance, SEO, security, or content quality worse?*

It runs eight scanners — Axe, Lighthouse, SEO, Link Checker, Security Headers, Open Graph, Spelling/Grammar, and an AI Navigator — against live URLs or static-site ZIP archives, normalizes the results into a single contract-driven report, and streams job progress in real time. The primary interface is a Go CLI designed for terminal-first feedback loops and CI gating.

![StageFlow dashboard and scan pipeline](docs/images/hero.png)

---

## Why This Project

StageFlow is a portfolio-quality distributed system built to be both **runnable and reviewable**. It demonstrates:

- **Real distributed architecture** — multiple Go services, NATS JetStream event streaming, per-job Podman pod isolation, MinIO artifact storage, PostgreSQL, and SQLite running together as a coherent system
- **Contract-first design** — shared JSON Schema contracts generate types for both Go and TypeScript consumers, keeping services decoupled without leaking implementation details
- **Two client surfaces, one API** — a SvelteKit 5 web app and a Go CLI both consume the same Platform API with no server-side differences
- **Layered verification** — Go race tests, Vitest unit tests, Storybook interaction and accessibility checks, golden regression flows, and CI gates at every layer
- **Security by default** — SSRF protections on URL intake, archive bomb defense on ZIP handling, rootless Podman execution, API middleware stack, and CI secrets scanning

The codebase is intentionally structured to be read as well as run. Start from `.env.example`, replace every `change-me` before using a public domain.

---

## What It Does

StageFlow accepts two input modes:

| Input | How it works |
|-------|-------------|
| **URL scan** | Submit one or more live URLs; scanners hit them directly |
| **ZIP scan** | Upload a static-site archive; StageFlow extracts it safely, discovers HTML, and scans it locally in an isolated container |

The system produces one normalized report with:

- Unified severity scoring (critical → serious → moderate → minor → info)
- Stable content-based issue IDs for regression diffing across reruns
- Page-level evidence with screenshots
- Per-scanner summaries and per-page rollups
- Real-time job progress streamed over SSE

The primary product loop is:

```
1. Make a frontend change locally
2. Run: stageflow project
3. StageFlow starts your dev server, scans it, and streams results
4. CLI exits 0 (pass) or 1 (severity gate) — machines and humans both understand it
```

---

## Architecture at a Glance

```
┌─ Clients ─────────────────────────────────────────────────────────────────┐
│  Web (SvelteKit 5)            CLI (Go)                                    │
│       │                           │                                       │
│       └───────────────┬───────────┘                                       │
│                       ↓                                                   │
│              Platform API (Go)                                            │
│         ┌────────────────────────────┐                                    │
│         │  URL/ZIP intake + SSRF     │                                    │
│         │  Project CRUD + baselines  │                                    │
│         │  Report APIs + SSE hub     │                                    │
│         │  Diff engine               │                                    │
│         └────────────┬───────────────┘                                    │
└──────────────────────┼────────────────────────────────────────────────────┘
                       │ NATS JetStream (job.created → ...)
                       ↓
              Orchestrator (Go)
         ┌────────────────────────────┐
         │  Job FSM (state machine)   │
         │  Podman pod lifecycle      │
         │  Report aggregation        │
         │  PostgreSQL persistence    │
         └────────────┬───────────────┘
                      │ launches per-job Podman pod
                      ↓
         ┌────────────────────────────────────────┐
         │  Job Pod (ephemeral, rootless)         │
         │                                        │
         │  Archive Extractor (Go)                │
         │  ├─ Safe ZIP extraction                │
         │  └─ Static file serving                │
         │                                        │
         │  Scanner Runner (TypeScript/Bun)       │
         │  ├─ Axe (accessibility)                │
         │  ├─ Lighthouse (performance/SEO)       │
         │  ├─ Security Headers                   │
         │  ├─ SEO, Link Checker, Open Graph      │
         │  ├─ Spelling/Grammar                   │
         │  └─ AI Navigator                       │
         └────────────────────────────────────────┘
                      │ artifacts → MinIO
                      │ events → NATS JetStream
```

| Service | Language | Responsibility |
|---------|----------|---------------|
| `clients/web` | SvelteKit 5 | Scan submission, live status, report exploration |
| `clients/cli` | Go | Automation, CI gating, Project Mode, JSON output |
| `services/platform-api` | Go | HTTP boundary, intake, SSE hub, projects, diffs |
| `services/orchestrator` | Go | Job FSM, NATS events, Podman pods, aggregation |
| `services/archive-extractor` | Go | Safe ZIP extraction, provenance, static serving |
| `services/scanner-runner` | TypeScript/Bun | Scanner plugins, Playwright automation, artifacts |
| `libs/contracts` | JSON Schema | Shared report/event contracts → generated Go + TS types |

Full architecture details: [ARCHITECTURE.md](ARCHITECTURE.md) and [docs/architecture/system.md](docs/architecture/system.md).

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| API services | Go 1.26 |
| Scanner runtime | TypeScript, Bun 1.3.8, Node 22 |
| Web UI | SvelteKit 5, Svelte 5, Tailwind CSS |
| CLI | Go 1.26 |
| Message bus | NATS JetStream 2.12 |
| Job state | PostgreSQL 17 |
| Project baselines | SQLite |
| Artifact storage | MinIO (S3-compatible) |
| Browser automation | Playwright + Chromium |
| Container runtime | Podman (rootless) |
| Observability | Grafana 12 |

---

## Quick Start

### Prerequisites

- [Go 1.26.3](https://go.dev/dl/)
- [Node.js 22](https://nodejs.org/) + [Bun](https://bun.sh/)
- [Podman](https://podman.io/) with `podman compose`
- [just](https://github.com/casey/just)
- [golangci-lint v2](https://golangci-lint.run/)

### Fastest smoke test

```bash
git clone https://github.com/mattboback/stageflow.git
cd stageflow
cp .env.example .env
just diagnose   # checks prerequisites
just demo       # builds images, starts stack, initializes MinIO
```

Then open `http://localhost:3000` or run a scan from the terminal:

```bash
just cli-install
stageflow scan https://example.com
```

### Local Project Mode (scan your dev server)

```bash
just setup && just images && just dev up local && just dev init local
stageflow project init    # creates .stageflow/config.yaml
stageflow project doctor  # validates wiring
stageflow project         # starts dev server, scans it, streams results
```

### Manual bootstrap

```bash
just setup
just images
just dev up
just dev init
```

---

## Key Design Decisions

**SSE over WebSocket** — job progress is one-directional. Server-Sent Events are simpler, proxy-friendly, and require no upgrade handshake. Reconnection and event buffering are handled in the SSE hub.

**NATS JetStream** — the API publishes events; the Orchestrator subscribes. Durable consumers and explicit ACK/NAK mean the Orchestrator can replay missed events after a restart without polling the database.

**Per-job Podman pods** — every scan runs in an ephemeral rootless pod with `no-new-privileges` and CPU/memory limits. Isolation is per-job, not per-scanner, so a compromised scanner cannot affect the host or other jobs.

**Schema-first contracts** — one JSON Schema per concept (`unified-report`, `scanner-manifest`, `provenance`, events). Generated Go and TypeScript types are committed to the repo. Adding a scanner cannot silently break the web UI or CLI.

**Stable content-based issue IDs** — each issue is fingerprinted from `sha256(ruleId + context + occurrence)`. The same violation produces the same ID across reruns and scanner updates, which is what makes baseline diffing reliable.

**Explicit job FSM** — the Orchestrator tracks `PENDING → EXTRACTING → SCANNING → COMPLETING → DONE/FAILED` with defined transition rules and completion policies. Terminal states prevent duplicate events; the FSM is tested independently of real infrastructure.

---

## Testing Strategy

| Layer | Tools | What it covers |
|-------|-------|---------------|
| Go unit tests | `go test -race` | FSM transitions, domain logic, utilities |
| Go lint/vuln | `golangci-lint`, `govulncheck` | Code quality and known CVEs |
| Web unit tests | Vitest, Testing Library | Components, SSE stores, report logic |
| Storybook CI | Storybook test runner | Component interaction and accessibility |
| Scanner runner tests | Vitest | Plugin loading, scanner output validation |
| E2E golden flow | `qa/e2e/project-scan-golden.sh` | Baseline → promote → regression diff → exit code |
| Container security | Trivy, `bun audit`, gitleaks | Image CVEs, dependency audit, secrets |

---

## Environment Modes

| Mode | Use | Web | API |
|------|-----|-----|-----|
| `dev` (`just demo`) | Local smoke test | `localhost:3000` | `localhost:8080` |
| `local` overlay | Scan private/localhost targets | `localhost:3010` | `localhost:8080` |
| `staging` overlay | Domain-like alt ports | `127.0.0.1:3300` | `127.0.0.1:8300` |
| Self-hosted edge | Public domain + TLS | Caddy → bridge port | Caddy → bridge port |

Self-hosting notes:
- Start from `.env.example`; never commit `.env` or real credentials
- Replace every `change-me` value before using a public domain
- See [docs/reference/configuration.md](docs/reference/configuration.md) and [docs/operations/deployment.md](docs/operations/deployment.md) for full guidance
- The hosted `stageflow.org` demo runs the same code; its production control plane is managed outside this repo

---

## Where to Start Reviewing

The highest-signal parts of the codebase for a technical review:

1. [services/orchestrator](services/orchestrator) — explicit FSM, NATS-driven coordination, E2E-style tests without real infra
2. [services/scanner-runner](services/scanner-runner/README.md) — plugin system, Playwright integration, contract enforcement
3. [libs/contracts](libs/contracts) — schema-first design, generated cross-language types
4. [clients/cli](clients/cli/README.md) — Project Mode, streaming UX, JSON output, severity exit codes
5. [clients/web](clients/web/README.md) — SSE-driven report UX, accessibility, component tests
6. [ARCHITECTURE.md](ARCHITECTURE.md) — service boundaries, data flow, design rationale

Guided review path (5–15 min): [docs/evaluators-guide.md](docs/evaluators-guide.md)

---

## Screenshots

### Real-World Dogfooding: Auditing AlchemizeCV
StageFlow is used to continuously audit [AlchemizeCV](https://alchemizecv.com) (our resume-tailoring SaaS platform) using a secure, multi-step authenticated scanning flow:

<table>
  <tr>
    <td><img src="docs/images/scan-in-progress.png" alt="Live scan progress of AlchemizeCV with scanner status and event streaming" /></td>
    <td><img src="docs/images/report-overview.png" alt="Unified report overview of AlchemizeCV's private workspace dashboard" /></td>
  </tr>
  <tr>
    <td align="center"><em>Live job progress of AlchemizeCV scan streamed over SSE</em></td>
    <td align="center"><em>Unified report overview of AlchemizeCV's private dashboard after authentication</em></td>
  </tr>
  <tr>
    <td><img src="docs/images/report-issues.png" alt="Detailed issues view of AlchemizeCV dashboard audit" /></td>
    <td><img src="docs/images/report-pages.png" alt="Page-level visual evidence and contrast highlights on the AlchemizeCV dashboard" /></td>
  </tr>
  <tr>
    <td align="center"><em>Grouped scanner issues view with precise rule breakdowns</em></td>
    <td align="center"><em>Page-level interactive evidence overlay showing visual accessibility outlines</em></td>
  </tr>
</table>

---

## Docs

| Document | Description |
|----------|-------------|
| [docs/evaluators-guide.md](docs/evaluators-guide.md) | Guided 5–15 min review path |
| [ARCHITECTURE.md](ARCHITECTURE.md) | Architecture overview with diagrams and design rationale |
| [ROADMAP.md](ROADMAP.md) | Direction and planned work |
| [CHANGELOG.md](CHANGELOG.md) | Release notes (Keep a Changelog) |
| [docs/architecture/system.md](docs/architecture/system.md) | Full system design, threat model, failure modes |
| [docs/reference/configuration.md](docs/reference/configuration.md) | Environment variables and overlay topology |
| [docs/operations/deployment.md](docs/operations/deployment.md) | Local, self-hosted, and hosted deployment boundaries |
| [docs/PROJECT_MODE.md](docs/PROJECT_MODE.md) | Local dev server lifecycle scanning |
| [clients/web/README.md](clients/web/README.md) | Frontend architecture, routes, tests |
| [services/platform-api/README.md](services/platform-api/README.md) | API routes, middleware, SSRF |
| [services/scanner-runner/README.md](services/scanner-runner/README.md) | Scanner plugin system and outputs |
| [clients/cli/README.md](clients/cli/README.md) | CLI commands and output formats |

## Support

- Bug reports and setup questions: GitHub issue templates
- Vulnerability reports: [SECURITY.md](SECURITY.md)

## License

[MIT](LICENSE)
