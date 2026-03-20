# StageFlow

[![CI](https://github.com/mattboback/stageflow/actions/workflows/ci.yml/badge.svg)](https://github.com/mattboback/stageflow/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Security Policy](https://img.shields.io/badge/security-policy-blue.svg)](.github/SECURITY.md)

**Live demo:** [stageflow.org](https://stageflow.org)  
**Run locally:** `cp .env.example .env && just setup && just dev up && just dev init && just images`

Podman-native web accessibility and quality scanning platform.

StageFlow runs multi-scanner audits against live URLs or static-site ZIP archives, streams job progress in real time, and merges heterogeneous scanner outputs into one normalized report. The project is designed for self-hosting, strict intake validation, and operational transparency.

![StageFlow — live scan pipeline dashboard](docs/images/hero.png)

## Why this project is interesting

StageFlow is a strong showcase project because it combines:

- a real multi-service architecture
- real-time SSE job streaming
- isolated scanner execution in per-job pods
- contract-driven report normalization across multiple scanners
- both a web UI and CLI surface for the same backend platform

## What you can do with it

- Submit one job and run multiple scanners in parallel.
- Track live progress through SSE at `/api/v1/jobs/{id}/stream`.
- Scan public URLs, static-site ZIPs, and local projects through CLI project mode.
- Keep scanner execution isolated from the main app runtime.
- Review one unified report with findings, evidence, and per-scanner results.

## Architecture at a glance

```text
Client/UI -> Platform API -> NATS JetStream -> Orchestrator -> Podman job pod
                                                     |            |- Extractor (ZIP jobs)
                                                     |            `- Scanner runners
                                                     `-> Status + artifacts -> MinIO -> unified report
```

- `services/platform-api` validates intake, applies SSRF guardrails, and exposes scan/report APIs.
- `services/orchestrator` owns the job state machine and scanner lifecycle.
- `services/scanner-runner` loads scanner manifests and executes browser-driven checks.
- `clients/web` renders live status, report views, and operational workflow screens.
- `clients/cli` submits scans, drives project mode, and renders reports in terminal-friendly formats.

For the full system design, see [docs/architecture/system.md](docs/architecture/system.md).

## Repo map

- `clients/web` — SvelteKit frontend
- `clients/cli` — `stageflow` CLI
- `services/platform-api` — intake API, SSE stream, report APIs
- `services/orchestrator` — job FSM and scanner orchestration
- `services/scanner-runner` — scanner runtime and Playwright-based execution
- `libs/contracts` — shared schemas and generated contracts
- `devtools` — internal ops and QA helpers
- `qa` — end-to-end and verification assets

## Screenshots

### Scanner selection

![Six scanners — Axe, Lighthouse, SEO, Security Headers, Link Checker, AI Navigator](docs/images/landing-scanners.png)

### Live scan execution

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

### Unified report

<table>
  <tr>
    <td><img src="docs/images/report-overview.png" alt="Report overview — risk snapshot, severity breakdown, Lighthouse scores, scanner status" /></td>
    <td><img src="docs/images/report-issues.png" alt="Issues tab — grouped findings with severity and remediation" /></td>
  </tr>
  <tr>
    <td align="center"><em>Overview with severity, scores, and scanner status</em></td>
    <td align="center"><em>Issues tab with grouped findings and remediation detail</em></td>
  </tr>
</table>

### Page-level evidence

<table>
  <tr>
    <td><img src="docs/images/report-pages.png" alt="Pages tab — annotated screenshot with bounding box overlays" /></td>
    <td><img src="docs/images/report-pages-detail.png" alt="Issue detail — evidence crop, selector, and fix guidance" /></td>
  </tr>
  <tr>
    <td align="center"><em>Annotated screenshots for page-level evidence</em></td>
    <td align="center"><em>Issue detail with evidence crop and fix guidance</em></td>
  </tr>
</table>

### CLI

<table>
  <tr>
    <td><img src="docs/images/cli-help.png" alt="stageflow --help output" /></td>
    <td><img src="docs/images/cli-scanners.png" alt="stageflow scanners output" /></td>
  </tr>
  <tr>
    <td align="center"><em>CLI command surface</em></td>
    <td align="center"><em>Scanner discovery from the terminal</em></td>
  </tr>
</table>

## Built-in scanners

| Scanner | Focus |
| --- | --- |
| `axe` | Accessibility and WCAG rule violations |
| `lighthouse` | Performance and quality audits |
| `seo` | Search and metadata best practices |
| `security-headers` | HTTP security header posture |
| `link-checker` | Broken link detection |
| `ai-navigator` | Goal-driven browser flow evaluation |
| `open-graph` | Social preview and metadata validation |
| `spelling-grammar` | AI-assisted content quality analysis |

## Quick start

### Prerequisites

- [Go 1.26.1](https://go.dev/dl/)
- [Bun](https://bun.sh/)
- [Podman](https://podman.io/) with `podman compose`
- [just](https://github.com/casey/just)
- [golangci-lint v2](https://golangci-lint.run/)

### Start the local stack

```bash
git clone https://github.com/mattboback/stageflow.git
cd stageflow
cp .env.example .env

just setup
just dev up
just dev init
just images
```

After startup, the endpoints depend on your environment mode:

| Service | `dev` mode (default) | `local` overlay mode |
| --- | --- | --- |
| Frontend | `http://localhost:3000` | `http://localhost:3010` |
| Platform API | `http://localhost:8080` | `http://localhost:8080` |
| Orchestrator Admin API | `http://localhost:8081` | `http://localhost:8081` |

### Run a first scan

```bash
job_id="$({
  curl -sS -X POST http://localhost:8080/api/v1/jobs/urls \
    -H 'content-type: application/json' \
    -d '{"urls":["https://example.com"]}'
} | jq -r '.job_id')"

curl -N "http://localhost:8080/api/v1/jobs/$job_id/stream"
```

### Scan localhost or private targets

Use the local overlay when scanners must reach `127.0.0.1`, `localhost`, or other private addresses:

```bash
just dev up local
just dev init local
just images
```

### Install the CLI

```bash
just cli-install
stageflow version
stageflow scan https://example.com
```

## Demo flows

If you want a quick portfolio walkthrough, start with these:

1. Submit a public URL scan from the web UI.
2. Watch live progress over SSE while scanners run.
3. Open the unified report and inspect findings, screenshots, and per-scanner status.
4. Run the same platform through the CLI with `stageflow scan` or `stageflow project`.

## Release model

StageFlow follows two release streams:
- **Application stack**: Continuous deployment from the `main` branch. Commits to `main` are considered production-ready for the control plane.
- **CLI (`stageflow`)**: Tagged releases (e.g., `clients/cli/v0.1.0`). GitHub Actions automatically cross-compiles and attaches binary assets to GitHub Releases when a tag is pushed.

See [CHANGELOG.md](CHANGELOG.md) for the history of notable changes.

## Quality and testing

StageFlow keeps the quality story visible and reproducible:

- strict TypeScript in the frontend and scanner runtime
- Go build, lint, race-test, and vulnerability checks
- Vitest coverage plus Storybook interaction and accessibility testing
- repo-level CI that runs the major quality gates together

Run the main validation flows locally with:

```bash
just ci
just storybook-test
just shell-tests
```

## Docs map

Use the shortest path to the detail you need:

- [Architecture](docs/architecture/system.md)
- [Configuration reference](docs/reference/configuration.md)
- [CLI and developer tooling](docs/operations/devtools.md)
- [CLI cheatsheet](docs/operations/cli_cheatsheet.md)
- [Project mode](docs/PROJECT_MODE.md)
- [CLI README](clients/cli/README.md)
- [Contributing](.github/CONTRIBUTING.md)
- [Support](.github/SUPPORT.md)

## Operating modes

### Local development

Use this repo directly for local development and iteration:

```bash
just setup
just dev up
just dev init
just images
```

### Staging verification

Use the staging recipes when you need a repo-managed verification stack:

```bash
just staging up
just staging init
just staging ps
just staging logs
```

### Production boundary

The live site at `stageflow.org` is real, but production operations are intentionally managed from an external deployment workspace, not from this repo.

Use the external control plane for live operations:

```bash
# Set STAGEFLOW_PROD_DEPLOY_DIR to your deployment workspace root first
cd $STAGEFLOW_PROD_DEPLOY_DIR
just deploy stageflow
just restart stageflow
just logs stageflow
just health
just status
```

This repository remains the source of truth for the application code, local development flow, staging configuration, and documentation.

## Contributing

If you want to work on StageFlow, start with the [Contributing guide](.github/CONTRIBUTING.md). For support and troubleshooting, see the [Support guide](.github/SUPPORT.md).
