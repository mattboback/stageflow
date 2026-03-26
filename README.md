# StageFlow

[![CI](https://github.com/mattboback/stageflow/actions/workflows/ci.yml/badge.svg)](https://github.com/mattboback/stageflow/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Security Policy](https://img.shields.io/badge/security-policy-blue.svg)](.github/SECURITY.md)

**Live demo:** [stageflow.org](https://stageflow.org)  
**Run locally:** `cp .env.example .env && just setup && just dev up && just dev init && just images`

Podman-native web accessibility and quality scanning platform.

StageFlow runs multi-scanner audits against live URLs or static-site ZIP archives, streams job progress in real time, and merges heterogeneous scanner outputs into one normalized report. The project is designed for self-hosting, strict intake validation, and operational transparency.

![StageFlow — live scan pipeline dashboard](docs/images/hero.png)

## Engineering highlights

- Multi-service Go/TypeScript architecture coordinated through NATS JetStream
- Real-time SSE streaming from orchestrator through API to browser and CLI
- Per-job Podman pod isolation for scanner execution
- Contract-driven report normalization across 8 heterogeneous scanners
- Dual-surface API: same backend drives both a SvelteKit web UI and a Go CLI

## What you can do with it

- Submit one job and run multiple scanners in parallel.
- Track live progress through SSE at `/api/v1/jobs/{id}/stream`.
- Scan public URLs, static-site ZIPs, and local projects through CLI project mode.
- Track projects with baselines and detect regressions across scans via automatic diffing.
- Keep scanner execution isolated from the main app runtime.
- Review one unified report with findings, evidence, and per-scanner results.
- Use structured JSON output and exit codes as an automated quality gate in CI or agentic coding workflows.

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

![Eight scanners — Axe, Lighthouse, SEO, Security Headers, Link Checker, AI Navigator, Open Graph, Spelling & Grammar](docs/images/landing-scanners.png)

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

## CLI

The `stageflow` CLI submits scan jobs, streams live progress, and renders unified reports in text, markdown, or JSON. It also supports project mode for local dev server scanning.

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

### Scan a URL

```bash
# Default text output
stageflow scan https://example.com --api https://stageflow.org

# Choose specific scanners
stageflow scan https://example.com --scanners axe,seo,open-graph --api https://stageflow.org

# JSON for automation
stageflow scan https://example.com --scanners axe --format json --api https://stageflow.org
```

### Exit codes and quality gates

The `--fail-on` flag sets a severity threshold. The CLI exits 1 if any issue meets or exceeds it, exit 0 otherwise — no output parsing required.

```bash
# Pass: only moderate issues exist, threshold is serious
stageflow scan https://example.com --scanners axe --fail-on serious  # exit 0

# Fail: moderate issues exist, threshold is moderate
stageflow scan https://example.com --scanners axe --fail-on moderate # exit 1
```

Severity levels from highest to lowest: `critical`, `serious`, `moderate`, `minor`, `info`.

### JSON output structure

The `--format json` envelope (`stageflow-cli/report@v1`) is designed for programmatic consumption:

```jsonc
{
  "schema": "stageflow-cli/report@v1",
  "job":    { "id": "...", "state": "DONE" },
  "report": {
    "summary": {
      "score": 85,
      "scoreGrade": "B",
      "totalIssues": 2,
      "bySeverity": { "critical": 0, "serious": 0, "moderate": 2, "minor": 0 },
      "byScanner":  { "axe": 2 }
    },
    "issues": [
      {
        "id": "672859f7e59a",           // stable content-based hash
        "ruleId": "landmark-one-main",  // axe rule identifier
        "scanner": "axe",
        "severity": "moderate",
        "title": "Document should have one main landmark",
        "howToFix": "Fix all of the following: ...",
        "wcagTags": ["WCAG 1.3.1"],
        "occurrences": [
          { "selector": "html", "html": "<html lang=\"en\">", "target": ["html"] }
        ]
      }
    ],
    "scanners": [ { "id": "axe", "status": "success", "issueCount": 2 } ],
    "pages":    [ { "url": "https://example.com", "issueCount": 2 } ]
  }
}
```

Each issue includes a CSS selector, HTML snippet, and remediation guidance — enough for automated tooling to locate and fix violations.

### Remote project management

Projects are named entities on the platform that track target URLs, scanner configuration, and a baseline scan for regression detection.

```bash
# Create a project
stageflow project create my-app --url https://example.com --scanner axe

# Scan the project (compares against baseline if one is set)
stageflow scan --project my-app --format json

# Promote a scan as the baseline for future comparisons
stageflow project promote my-app --job-id <job-id>

# Update project configuration
stageflow project update my-app --url https://example.com/v2

# Other CRUD
stageflow project list
stageflow project show my-app
stageflow project delete my-app
```

When a baseline is set, `stageflow scan --project` outputs two JSON documents: the scan report followed by a diff showing new, fixed, and unchanged issues. The CLI exits 1 if regressions are detected, making it a drop-in CI quality gate.

### Local project mode

Local project mode integrates scanning into local development. It starts your dev server, waits for readiness, runs the scan, and shuts down the server.

```bash
stageflow project init          # scaffold .stageflow/config.yaml
stageflow project doctor        # validate config without scanning
stageflow project               # full lifecycle: start → scan → stop
```

See [Project Mode docs](docs/PROJECT_MODE.md) for configuration reference.

## Built-in scanners

| Scanner | Categories | Focus |
| --- | --- | --- |
| `axe` | accessibility | WCAG violations — landmarks, ARIA, color contrast, alt text, keyboard nav |
| `lighthouse` | performance, accessibility, seo, quality | Google Lighthouse audits — Core Web Vitals, best practices, scores |
| `seo` | seo | Meta tags, canonical URLs, structured data, content depth, title length |
| `security-headers` | security | HTTP header posture — CSP, HSTS, X-Frame-Options, Permissions-Policy |
| `link-checker` | quality | Broken links, redirect chains, link quality |
| `open-graph` | seo | Open Graph and Twitter Card metadata validation |
| `spelling-grammar` | quality | AI-assisted spelling and grammar analysis |
| `ai-navigator` | custom | LLM-powered Playwright agent — goal-driven browser flow evaluation |

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

With the CLI:

```bash
just cli-install
stageflow scan https://example.com
```

Or with curl:

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

Quickest way to see the system end-to-end:

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
- golden E2E test for the project scan → baseline → diff pipeline
- repo-level CI that runs the major quality gates together

Run the main validation flows locally with:

```bash
just ci
just storybook-test
just shell-tests
```

### Golden E2E test

`qa/e2e/project-scan-golden.sh` exercises the full project lifecycle against a live stack: create project, scan a clean page, promote baseline, swap to a page with a known violation, rescan, and verify the diff output matches committed golden files.

```bash
# Run against prod (default)
bash qa/e2e/project-scan-golden.sh

# Run against a local stack
STAGEFLOW_API_URL=http://localhost:8080 bash qa/e2e/project-scan-golden.sh
```

Golden files live in `qa/fixtures/project-golden/`. On first run the script auto-creates them from actual output; subsequent runs compare against them with `diff -u`.

## Docs map

Use the shortest path to the detail you need:

- [Architecture overview](ARCHITECTURE.md)
- [Architecture deep-dive](docs/architecture/system.md)
- [Configuration reference](docs/reference/configuration.md)
- [CLI and developer tooling](docs/operations/devtools.md)
- [CLI cheatsheet](docs/operations/cli_cheatsheet.md)
- [Project mode](docs/PROJECT_MODE.md)
- [CLI README](clients/cli/README.md)
- [Contributing](.github/CONTRIBUTING.md)
- [Support](.github/SUPPORT.md)

## Operating modes

**Local development:** `just setup && just dev up && just dev init && just images` (see [Quick start](#quick-start))

**Staging:** `just staging up && just staging init && just staging ps`

## Contributing

If you want to work on StageFlow, start with the [Contributing guide](.github/CONTRIBUTING.md). For support and troubleshooting, see the [Support guide](.github/SUPPORT.md).
