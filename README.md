# StageFlow

[![CI](https://github.com/mattboback/stageflow/actions/workflows/ci.yml/badge.svg)](https://github.com/mattboback/stageflow/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Security Policy](https://img.shields.io/badge/security-policy-blue.svg)](SECURITY.md)

**[Live Demo](https://stageflow.org)** | Try the scanning pipeline against any public URL

Podman-native web accessibility and quality scanning platform.

StageFlow runs multi-scanner audits against live URLs or static-site ZIP archives, then aggregates outputs into one normalized report stream. It is built for self-hosting, strict intake validation, and operational transparency.

![StageFlow — live scan pipeline dashboard](docs/images/hero.png)

## Why StageFlow

- Run accessibility and quality scans in infrastructure you control.
- Submit one job and run multiple scanner modules in parallel.
- Track job progress in real time through SSE (`/api/v1/jobs/{id}/stream`).
- Keep scanner execution isolated in per-job pods.
- Produce one deduplicated report from heterogeneous scanner outputs.

## At a Glance

```text
Client/UI -> Platform API -> NATS JetStream -> Orchestrator -> Podman job pod
                                                      |            |- Extractor (ZIP jobs)
                                                      |            `- Scanner runners
                                                      `-> Status + artifacts -> MinIO -> unified report
```

- API validates intake, applies SSRF guardrails, and publishes job events.
- Orchestrator owns the job FSM and scanner lifecycle.
- Scanner runner loads plugins by manifest and validates scanner options.
- Frontend receives live status with SSE and fallback refresh logic.

Full design details: [Architecture](docs/architecture/system.md).

## Screenshots

### Scanners

Eight powerful built-in scanner modules run in parallel inside isolated containers. Choose scanners per run and get one normalized report.

![Six scanners — Axe, Lighthouse, SEO, Security Headers, Link Checker, AI Navigator](docs/images/landing-scanners.png)

### Workflow

Submit a target, run isolated scanners, and ship one unified report.

![Workflow — configure scope, run scanners, ship report](docs/images/landing-workflow.png)

### Playground

Configure input type, target URLs, scanner selection, and run options from a single control surface.

![Playground — configure scan input and scanners](docs/images/playground.png)

Scanner presets (Coverage, Quick, Custom) let you select modules with one click. Each scanner card shows its focus area and current state.

![Playground — scanner selection with presets and module cards](docs/images/playground-scanners.png)

### Scan Execution

Real-time SSE event stream shows scanner progress, container status, and log output as the scan runs.

<table>
  <tr>
    <td><img src="docs/images/scan-in-progress.png" alt="Live scan in progress — SCANNING state with progress pipeline and scanner activity" /></td>
    <td><img src="docs/images/scan-complete.png" alt="Scan complete — all pipeline stages green with report and artifact links" /></td>
  </tr>
  <tr>
    <td align="center"><em>Live progress stream during execution</em></td>
    <td align="center"><em>Completed scan with report and artifact links</em></td>
  </tr>
</table>

### Report

The unified report aggregates findings from all scanners into one view with severity breakdown, Lighthouse scores, and scanner status.

![Report overview — risk snapshot, severity breakdown, Lighthouse scores, scanner status](docs/images/report-overview.png)

<table>
  <tr>
    <td><img src="docs/images/report-issues.png" alt="Issues tab — all findings with severity and remediation" /></td>
    <td><img src="docs/images/report-scanners.png" alt="Scans tab — per-scanner results and timing" /></td>
  </tr>
  <tr>
    <td align="center"><em>Issues tab — grouped findings with severity and fix guidance</em></td>
    <td align="center"><em>Scans tab — per-scanner results and timing</em></td>
  </tr>
</table>

### Page-Level Evidence

The Pages tab renders an annotated screenshot of each scanned page with bounding boxes highlighting issue locations. Click any marker to open full remediation detail.

<table>
  <tr>
    <td><img src="docs/images/report-pages.png" alt="Pages tab — annotated screenshot with bounding box overlays" /></td>
    <td><img src="docs/images/report-pages-detail.png" alt="Issue detail — evidence crop, selector, and fix guidance" /></td>
  </tr>
  <tr>
    <td align="center"><em>Annotated page screenshot with bounding boxes</em></td>
    <td align="center"><em>Issue detail with evidence crop, selector, and fix guidance</em></td>
  </tr>
</table>

![Full report — complete scan output](docs/images/scan-report.png)

### CLI

The `stageflow` CLI submits scans, lists scanners, and fetches reports from the terminal.

<table>
  <tr>
    <td><img src="docs/images/cli-help.png" alt="stageflow --help output" /></td>
    <td><img src="docs/images/cli-scanners.png" alt="stageflow scanners output" /></td>
  </tr>
  <tr>
    <td align="center"><em>CLI commands and flags</em></td>
    <td align="center"><em>Available scanners with categories and versions</em></td>
  </tr>
</table>

## Built-In Scanners

| Scanner | Focus |
| --- | --- |
| `axe` | Accessibility (WCAG rule violations) |
| `lighthouse` | Performance and quality audits |
| `seo` | SEO best-practice checks |
| `security-headers` | HTTP security header posture |
| `link-checker` | Broken link detection |
| `ai-navigator` | Goal-driven browser flow evaluation |
| `open-graph` | Social sharing metadata and rich preview validation |
| `spelling-grammar` | AI-powered content quality, spelling, and grammar analysis |

### New Advanced Scanners

Take your quality checks to the next level with our newest additions:

- **Open Graph Scanner (`open-graph`)**: Never ship a broken social preview again! This scanner digs deep into your metadata, validating Twitter cards, OG tags, and rich preview data to guarantee your links look absolutely flawless when shared on social media and messaging platforms.
- **Spelling & Grammar Scanner (`spelling-grammar`)**: Supercharge your content quality! Our intelligent text analysis thoroughly checks your pages for embarrassing typos and grammatical errors, ensuring your copy is polished, professional, and perfectly readable.

## AI Navigator

The `ai-navigator` scanner is a vision-model-powered browser automation agent that navigates websites to achieve user-defined goals. It uses OpenRouter as the AI provider gateway, routing requests to models from OpenAI, Anthropic, and Google.

### How It Works

1. User defines a navigation goal (e.g., "find the pricing page and verify it loads"), selects a model, and sets constraints (max steps, timeout, success criteria).
2. The agent runs an iterative loop: **screenshot page → send to vision model for analysis → decide next action → execute action → repeat**.
3. Available actions include click, fill, scroll, hover, select, keyboard input, and wait.
4. The loop ends when success criteria are met, max steps are reached, or the timeout expires.

### Supported Models (via OpenRouter)

| Provider | Models |
| --- | --- |
| OpenAI | `gpt-4o`, `gpt-4o-mini` |
| Anthropic | `claude-3.5-sonnet`, `claude-3-haiku` |
| Google | `gemini-pro-vision` |

### Configuration

The AI Navigator requires an OpenRouter API key set via the `OPENROUTER_API_KEY` environment variable. See [CONFIGURATION.md](docs/reference/configuration.md) for the full list of AI-related environment variables.

From the Playground UI, users can configure the goal objective, model selection, input values for form fills, max steps, timeout, and success criteria.

## Tech Stack

| Layer | Technology |
| --- | --- |
| Backend services | Go 1.26.1 |
| Scanner runtime | TypeScript + Bun + Playwright |
| Frontend | SvelteKit 5 + Tailwind v4 |
| Messaging | NATS JetStream |
| Storage | MinIO (artifacts) + PostgreSQL (job state/events) |
| Container runtime | Podman |
| Edge/proxy | Caddy |
| Monitoring | Grafana |

## Quality & Testing

StageFlow is built to be production-ready and deeply tested across the stack:
- **100% Type Safety**: Frontend and scanner-runner enforce strict TypeScript (`noUncheckedIndexedAccess`, zero `any` usage).
- **Backend Quality**: Go codebase is fully linted (golangci-lint), race-tested, and audited via `govulncheck`.
- **Frontend Testing**: 220+ unit tests with Vitest, comprehensive component testing with Storybook, and Playwright a11y interaction tests.
- **Continuous Integration**: The entire CI pipeline enforces stringent quality gates on every commit.

### Running Tests Locally

You can run the full test suite locally using the `just ci` command, which includes linting, typechecking, go tests, shell regression tests, and Storybook tests.

```bash
# Run the entire CI suite locally
just ci

# Run only frontend Storybook interaction + accessibility tests
just storybook-test

# Run repo shell regression tests
just shell-tests
```

## Prerequisites

- [Go 1.26.1](https://go.dev/dl/)
- [Bun](https://bun.sh/)
- [Podman](https://podman.io/) (with `podman compose`)
- [just](https://github.com/casey/just)
- [golangci-lint v2](https://golangci-lint.run/)

## Quick Start

```bash
git clone https://github.com/mattboback/stageflow.git
cd stageflow

cp .env.example .env

# Keep .env files local and out of version control.

just setup
just dev up
just dev init
just images
```

After startup:

- Frontend: `http://localhost:3000`
- API: `http://localhost:8080`
- Orchestrator admin API: `http://localhost:8081`

Tip: `just demo` runs setup, starts the stack, initializes buckets, builds images, and prints a URL-scan command.

## Production Deployment

This repo does not own standalone production deployment on the shared VPS.

Production operations for `stageflow.org` run from a shared root control
plane (configured via `STAGEFLOW_PROD_DEPLOY_DIR`, defaults to `/home/matt/Deployment`). Use the root `justfile` there when you need
to operate the live stack:

```bash
cd ${STAGEFLOW_PROD_DEPLOY_DIR:-/home/matt/Deployment}
just deploy stageflow
just restart stageflow
just logs stageflow
just stop stageflow
just health
just status
```

Repo-local `just prod ...` and `just deploy ...` intentionally stop and point
back to the external control plane. Keep local and staging work in this repo, but
keep VPS production ownership at the control plane directory.

## First URL Scan (API)

```bash
job_id="$({
  curl -sS -X POST http://localhost:8080/api/v1/jobs/urls \
    -H 'content-type: application/json' \
    -d '{"urls":["https://example.com"]}'
} | jq -r .job_id)"

curl -N "http://localhost:8080/api/v1/jobs/$job_id/stream"
```

SSRF protections reject loopback/private/link-local/metadata destinations for URL jobs.

## Optional CLI Mode

StageFlow also includes an optional CLI client (`clients/cli/`) that
talks to the existing Platform API. It submits URL jobs, waits for completion,
and fetches the aggregated report via `GET /api/v1/jobs/{id}/results`.

Public URL scans:

```bash
go run ./clients/cli scan https://example.com
```

For a local install that avoids stale binaries:

```bash
just cli-install
stageflow scanners
stageflow scan https://example.com
stageflow scan https://example.com --format json > report.json
```

Or build a local binary in place:

```bash
cd clients/cli
go build -o stageflow .
./stageflow scanners
./stageflow scan https://example.com
./stageflow scan https://example.com --format json > report.json
```

Local project mode (starts a dev server and scans `localhost`):

See the full [Project Mode Guide](docs/PROJECT_MODE.md) for detailed setup instructions and configuration references.

- Run the local stack with the local-only overlay (enables private targets + host-network job pods):
  - `just dev up local`
  - `just dev init local`
  - `just images`
- In your web project repo, run `stageflow project init` (optionally pass a
  project path as an arg). This generates:
  - `.stageflow/config.yaml`
  - `.stageflow/README.md`
- Run `stageflow project doctor` to validate config and local dev readiness
  before scanning.
- Run `stageflow project` to start dev, wait for readiness, and run scans.

Environment variables:

- `STAGEFLOW_API_URL` (default `http://localhost:8080`)
- `STAGEFLOW_API_KEY` (optional, sent as `X-Api-Key`)

Notes:

- Text output is the default. Use `--format json` for machine-readable output.
- `--json` remains available for backward compatibility, but `--format json`
  is the preferred form.
- `localhost`/private target submissions require the API instance to allow
  private scans (`PLATFORM_API_ALLOW_PRIVATE_TARGETS=true`), which is enabled
  by `just dev up local`.
- The CLI auto-enables `allow_private_targets=true` when targets are loopback
  or private literals (`localhost`, `127.0.0.1`, RFC1918, IPv6 ULA).
- The CLI refuses to submit private/loopback targets to a non-loopback `--api`
  URL to avoid accidentally scanning the server's own localhost.
- New project templates use a placeholder `dev.start.cmd`; `stageflow project`
  exits with clear setup guidance until you replace it.
- Use `stageflow project doctor --skip-dev` to validate config and scan
  preflight only.
- Project mode executes commands from your repo config; only run it on trusted repos.
- On macOS/Windows with Podman VM, `POD_NETNS_MODE=host` typically refers to the VM, not your host machine.

## Refreshing changed local services

Use `just dev-refresh` when you change `platform-api`, `orchestrator`, or
`frontend` and want to rebuild only those services in the local compose stack.
The command uses the same compose overlays as `just dev`, rebuilds the selected
services, and retries automatically if Podman hits the common container
name-collision state.

Refresh the default local trio:

```bash
just dev-refresh ENV=local SERVICES='platform-api orchestrator frontend'
```

Refresh only the API:

```bash
just dev-refresh ENV=local SERVICES='platform-api'
```

If Podman still refuses to recreate the service, run the manual fallback:

```bash
podman compose -p stageflow \
  -f infra/compose/podman-compose.yml \
  -f infra/compose/podman-compose.local.yml \
  rm -sf platform-api orchestrator frontend

podman compose -p stageflow \
  -f infra/compose/podman-compose.yml \
  -f infra/compose/podman-compose.local.yml \
  up -d --build --force-recreate --no-deps platform-api orchestrator frontend
```

Refresh `platform-api` before `frontend`, or refresh both together, when the
frontend depends on new live-status fields.

## Day-to-Day Commands

Run `just help` for the full recipe list.

| Area | Command | Purpose |
| --- | --- | --- |
| Setup | `just setup` | Install deps, sync workspace, create network |
| Local stack | `just dev up` / `just dev down` / `just dev logs` | Start, stop, or inspect local compose stack |
| Local refresh | `just dev-refresh ENV=local SERVICES='platform-api orchestrator frontend'` | Rebuild and recreate selected local compose services |
| Staging stack | `just staging up` / `just staging down` | Manage staging compose environment |
| Build | `just build` | Build Go services, frontend, scanner-runner |
| Images | `just images` | Build container images |
| Quality | `just ci` | Lint, typecheck, tests, audits |
| Service run | `just run clients/web` / `just run storybook` / `just run api` / `just run orchestrator` | Run one service locally |
| Component testing | `just storybook-test` | Run Storybook interaction + a11y tests |
| Production | `cd ${STAGEFLOW_PROD_DEPLOY_DIR:-/home/matt/Deployment} && just deploy stageflow` | Shared VPS production control plane |

Storybook testing conventions: `docs/testing/storybook-component-testing.md`.

## Scanner Plugin System

Scanners are discovered via manifest files and loaded dynamically by the scanner runtime.

Discovery paths (in order):

1. Built-in scanners in `services/scanner-runner/src/scanners`
2. Mounted `/plugins`
3. User plugins at `~/.stageflow/plugins`
4. Extra paths from `PLUGIN_PATHS`

To add a custom scanner:

1. Implement a scanner module (extends `ScannerBase`).
2. Add a valid `manifest.json` (schema-backed).
3. Make the plugin available in a discovery path.

Reference docs:

- [Architecture](docs/architecture/system.md#scanner-plugin-system)
- `libs/contracts/scanner-manifest/schema/README.md`

## Security and Runtime Boundaries

- URL intake blocks private/loopback/link-local/metadata targets.
- ZIP extraction enforces archive safety limits and path sanitization.
- Scanner execution is containerized per job.
- API status streaming uses SSE with reconnect-safe behavior.
- Edge rate limiting is expected at proxy/load-balancer/CDN layers.

See [SECURITY.md](SECURITY.md) and [Architecture](docs/architecture/system.md#security-and-trust-boundaries).

## Repository Layout

```text
stageflow/
|- clients/               # Web frontend, CLI
|- services/              # API, orchestrator, extractor, scanner-runner
|- libs/                  # Contracts + shared Go modules
|- infra/                 # Compose, Caddy, Quadlets, monitoring, scanner config
|- devtools/              # job-status-cli, suite-runner, scripts
|- qa/                    # End-to-end tests
```

> **Naming convention:** The web frontend source lives at `clients/web` and is referenced as `clients/web` in `justfile` recipes (e.g. `just run clients/web`), CI, and build scripts. The name `frontend` is used only as a Compose service name, container image tag, and systemd/quadlet unit name. These are intentionally separate: `clients/web` is the repo-level identity, `frontend` is the runtime identity.

## Documentation Map

- [Architecture](docs/architecture/system.md): deep system design, flows, and constraints
- [Configuration](docs/reference/configuration.md): environment and deployment configuration guide
- [CLI Tools](docs/operations/devtools.md): CLI tooling and common workflows
- [PROJECT_MODE.md](docs/PROJECT_MODE.md): guide to setting up and using the local development project scanner
- [CONTRIBUTING.md](CONTRIBUTING.md): local workflow, standards, and PR checklist
- [SECURITY.md](SECURITY.md): vulnerability reporting policy
- [SUPPORT.md](SUPPORT.md): help channels and debugging checklist
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md): community conduct standards
- [CHANGELOG.md](CHANGELOG.md): release history

## Contributing

Contributions are welcome. Start with [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE) © 2025-2026 Matthew Boback
