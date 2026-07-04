# StageFlow

[![CI](https://github.com/mattboback/stageflow/actions/workflows/ci.yml/badge.svg)](https://github.com/mattboback/stageflow/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

StageFlow is a self-hostable **frontend quality platform**. It ships eight
built-in scanners — accessibility, performance, SEO, links, security headers,
social metadata, content quality, and agent-driven navigation — behind a single
report contract, and remembers a **baseline per project** so every scan can
answer the question that matters in CI: *did this change make the frontend
worse?*

**▶ Live demo: [stageflow.org](https://stageflow.org)** — scan any URL and explore a
real, interactive report in the browser. No account required.

![StageFlow CLI scanning example.com and failing on serious issues](docs/images/demo.gif)

Reviewing this codebase? The
[evaluator guide](docs/evaluators-guide.md) maps the most interesting surfaces —
contracts, SSRF guards, the job FSM, and the test strategy — into a 5–15 minute
tour.

## Try It in 60 Seconds

The CLI is a single Go binary that submits scans to any StageFlow API and
streams live results to your terminal. Point it at the hosted demo — nothing to
deploy:

```bash
export PATH="$HOME/.local/bin:$PATH"
just cli-install          # or download a binary from GitHub Releases
stageflow scan https://example.com --api https://stageflow.org
```

Prefer a browser? Submit the same scan at [stageflow.org](https://stageflow.org)
and watch it stream live.

Common CLI workflows:

```bash
# Choose scanners and return JSON for automation.
stageflow scan https://example.com --scanner axe,seo --format json --api https://stageflow.org

# Fail the command (exit 1) when serious-or-worse issues are found — a CI gate.
stageflow scan https://example.com --fail-on serious --api https://stageflow.org

# Scan several routes in one job.
stageflow scan https://example.com https://example.com/about --format markdown --api https://stageflow.org

# Scan a local build before it ships anywhere: the directory is zipped,
# uploaded, and served by the platform — no dev server or public URL needed.
stageflow scan ./dist --api https://stageflow.org
```

Exit codes are machine-readable: `0` clean, `1` threshold tripped, `2` CLI/API
error. Release binaries are on
[GitHub Releases](https://github.com/mattboback/stageflow/releases); the full
command reference is in [clients/cli/README.md](clients/cli/README.md) and
[docs/reference/cli/stageflow](docs/reference/cli/stageflow). If `stageflow`
doesn't resolve after `just cli-install`, put `~/.local/bin` earlier on `PATH`
and rerun — see [docs/operations/devtools.md](docs/operations/devtools.md).

## The Report

Every scan — CLI or browser — produces one unified report built for triage:
score and severity distribution at a glance, issues grouped by scanner/rule,
page screenshots and occurrence evidence, remediation details for engineers,
and JSON/HTML downloads.

![StageFlow report overview](docs/images/report-overview.png)

Every flagged element is overlaid directly on a full-page screenshot — click a
bounding box on the Pages tab to jump straight to that issue's fix guidance,
WCAG references, and evidence:

![Clicking a bounding-box overlay on the Pages tab opens the issue detail card](docs/images/report-bounding-box.gif)

The report shape is a versioned contract. To see it without running a scan,
open the committed fixture
[`libs/contracts/report/fixtures/unified-report.v2.all-scans.json`](libs/contracts/report/fixtures/unified-report.v2.all-scans.json)
— a full multi-scanner report validated against the v2 schema.

## Regression Memory: The Headline Capability

Register a project once, promote a known-good scan as its baseline, and every
later scan diffs against it — new issues or a tripped severity gate exit
non-zero, so it drops straight into CI:

```bash
stageflow project create marketing-site --url https://example.com --scanner axe --scanner seo
stageflow project scan marketing-site --format json     # streams live, then diffs vs. baseline
stageflow project promote marketing-site --job-id <id>  # accept a run as the new reference
```

The full lifecycle, a CI gate snippet, and auth details are in
[docs/remote.md](docs/remote.md).

## Going Deeper

- **Dev mode** — `stageflow dev` turns the CLI into a repeatable local
  regression loop: start your dev server, wait for readiness, scan, stream
  progress, emit JSON for agents or CI. See [docs/dev-mode.md](docs/dev-mode.md).
- **Architecture** — clients → Platform API → NATS JetStream → Orchestrator →
  per-job rootless Podman pods, with SSRF and ZIP-bomb guards at the trust
  boundaries. Start with [ARCHITECTURE.md](ARCHITECTURE.md), then
  [docs/architecture/system.md](docs/architecture/system.md).
- **Product & design rationale** — who the platform serves and why the report
  UI looks the way it does: [docs/product.md](docs/product.md) and
  [docs/design.md](docs/design.md).

## Self-Host Locally

The local stack uses Podman, NATS JetStream, PostgreSQL, MinIO, the Go Platform
API and Orchestrator, the TypeScript scanner runner, and the React Router web app.

Prerequisites:

- Go `1.26.4` or newer in the `1.26` line
- Node.js `22.x` or newer (the repo pins `22` in `.node-version`)
- Bun `1.3.8` or newer
- **Podman with Compose support — Docker is not supported.** Per-job rootless
  Podman pods are core to the isolation model, not an interchangeable runtime.
- `just`

Run the guided smoke test:

```bash
cp .env.example .env
just diagnose
just demo
```

When the demo is ready:

- Web UI: `http://localhost:3000`
- API: `http://localhost:8080`
- Grafana: `http://localhost:3001`

`just demo` uses an isolated local compose project and Podman network by
default (`stageflow_dev` / `stageflow_dev_net`). `just dev up dev` is the
lower-level default compose mode on port `3000`; `just dev up local` applies
the local overlay on port `3020` and enables localhost/private-target scanning
for the CLI dev loop. Manual stack commands (`just setup`, `just images`,
`just dev up`, `just dev init`, `just dev logs`) are available when you want
more control. Once the stack is up, `stageflow stack up|down|status` (the CLI
binary itself) can drive its day-to-day lifecycle in place of `just dev`.

Before exposing StageFlow on a public domain, replace every `change-me` value
in `.env`, set explicit CORS origins, keep API authentication enabled, and
review [docs/reference/configuration.md](docs/reference/configuration.md) and
[infra/security/egress-policy.example.md](infra/security/egress-policy.example.md).

Hosted production deployment, monitoring, rollback, and VPS control-plane
automation are intentionally managed outside this public repository. This repo
contains the application source, local development stack, and self-hosting
examples.

## What StageFlow Runs

Built-in scanners:

- `axe` for accessibility
- `lighthouse` for performance, best-practices, and PWA signals
- `seo` for title, metadata, heading, and structured-data checks
- `security-headers` for common HTTP response-header policy
- `link-checker` for internal and external link health
- `open-graph` for social preview metadata
- `spelling-grammar` for content quality
- `ai-navigator` for natural-language navigation objectives through an LLM

Scanner enablement and resource overrides are configured through
[infra/scanners/scanners.example.yaml](infra/scanners/scanners.example.yaml).
The local example enables a smaller default set for resource use; request or
enable additional scanners when you need them. The AI Navigator is optional and
requires `OPENROUTER_API_KEY` when enabled.

## Repository Map

| Path                         | Purpose                                                                  |
| ---------------------------- | ------------------------------------------------------------------------ |
| `clients/cli`                | Go CLI for scans, the dev loop, baselines, reports, and docs generation  |
| `clients/web`                | React Router/Vite app for hosted/local browser workflows                 |
| `services/platform-api`      | Public HTTP API, auth, CORS, URL/ZIP intake, SSE, projects, reports      |
| `services/orchestrator`      | Job state machine, Podman pod lifecycle, aggregation, persistence        |
| `services/archive-extractor` | Safe ZIP extraction and static serving inside job pods                   |
| `services/scanner-runner`    | Bun/TypeScript Playwright scanner runtime                                |
| `libs/contracts`             | JSON Schema contracts; Go/TypeScript types are generated during setup/CI |
| `infra`                      | Compose, Caddy, MinIO, Grafana, scanner, and security examples           |
| `docs`                       | Architecture, remote projects, dev mode, configuration, and CLI reference |

## Development

Install dependencies and local config:

```bash
just setup
```

From a fresh checkout, run `just setup` or `just generate-contracts` before
running focused Go or TypeScript build commands directly; generated contract
code is intentionally ignored.

Run the repository quality gate:

```bash
just ci
```

Pre-commit hooks are a fast local guard for common mistakes before a commit.
`just ci` is the full repo quality gate and is the command to use before
opening a PR or handing work to CI.

Useful focused checks:

```bash
just shell-tests
(cd clients/web && bun run ci)
(cd services/scanner-runner && bun run ci)
just dead-code
just project-golden
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for pull request expectations and
workspace-specific validation commands.

Committed screenshots and reviewer-facing images live under `docs/images`.
Temporary QA/build evidence belongs under ignored artifact, output, or cache
paths such as `artifacts/`, `output/`, and `.cache/`; `just clean` may remove
those ephemeral files.

## Security

StageFlow scans websites and can run browser automation, so deployment
boundaries matter. Public deployments should keep API authentication enabled,
avoid private-target scans unless explicitly intended, isolate scanner pods, and
use environment-specific credentials.

Report vulnerabilities privately through [SECURITY.md](SECURITY.md). Do not
open public issues for vulnerabilities.

## License

StageFlow is released under the [MIT License](LICENSE).
