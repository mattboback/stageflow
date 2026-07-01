# StageFlow

[![CI](https://github.com/mattboback/stageflow/actions/workflows/ci.yml/badge.svg)](https://github.com/mattboback/stageflow/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**Live demo: [stageflow.org](https://stageflow.org)** — scan any URL and explore an
interactive report in the browser. No account required.

StageFlow is a self-hostable **frontend quality platform**. It runs accessibility,
performance, SEO, link-health, security-header, social-metadata, content-quality, and
guided-navigation scanners behind one API, then normalizes the results into a single
report contract that the web UI, CLI, and baseline diff engine all consume.

Reviewing this as a portfolio project? Start with the
[evaluator guide](docs/evaluators-guide.md) for a 5–15 minute map of the most interesting
code paths.

![StageFlow report overview](docs/images/report-overview.png)

## Why it exists

Most frontend checks produce separate outputs: one Lighthouse result, one accessibility
report, one link checker run, one security-header check. StageFlow turns those into one
evidence-rich report and remembers a promoted baseline per project so CI can answer:

> Did this change make the frontend worse?

## Surfaces

| Surface | What it does |
| --- | --- |
| **Web UI** | Submit scans, watch live status, triage grouped issues, inspect screenshots/evidence |
| **CLI** | Run scans from a terminal or CI, stream SSE progress, emit text/Markdown/JSON |
| **Remote projects** | Register URLs/scanners once, promote a baseline, and detect regressions |
| **Self-host stack** | Podman compose stack with API, orchestrator, scanner runner, NATS, MinIO, PostgreSQL, Grafana |

## Start with the CLI

Download a release asset from
[GitHub Releases](https://github.com/mattboback/stageflow/releases), or build from source:

```bash
export PATH="$HOME/.local/bin:$PATH"
just cli-install
stageflow version
```

Run a hosted scan:

```bash
stageflow scan https://example.com --api https://stageflow.org
```

Common workflows:

```bash
# Choose scanners and return JSON for automation.
stageflow scan https://example.com --scanner axe,seo --format json --api https://stageflow.org

# Fail when serious-or-worse issues are found.
stageflow scan https://example.com --fail-on serious --api https://stageflow.org

# Scan several routes in one job.
stageflow scan https://example.com https://example.com/about --format markdown --api https://stageflow.org

# Scan a local dev server through a local StageFlow API.
stageflow scan http://127.0.0.1:5173 --allow-private-targets
```

Exit codes are intentionally machine-readable:

| Exit code | Meaning |
| --- | --- |
| `0` | Scan completed and no displayed issue met the `--fail-on` threshold |
| `1` | Scan completed and at least one displayed issue met the threshold |
| `2` | CLI or API error |

Full CLI docs live in [clients/cli/README.md](clients/cli/README.md) and
[docs/reference/cli/stageflow](docs/reference/cli/stageflow).

## Remote projects and regression memory

Register a project once, promote a known-good run, and let later scans compare against it:

```bash
stageflow project create marketing-site \
  --url https://example.com --scanner axe --scanner seo

stageflow project scan marketing-site --format json

stageflow project promote marketing-site --job-id <job-id>
```

A project scan exits non-zero when new issues appear or the severity gate trips, so it can
drop directly into CI. See [docs/remote.md](docs/remote.md) for the full workflow.

## Dev mode

`stageflow dev` turns the CLI into a local regression loop. It can start your dev server,
wait for readiness, run a scan, stream progress, stop the server, and emit machine-readable
JSON:

```bash
stageflow dev init
stageflow dev doctor
stageflow dev scan --format json
```

See [docs/dev-mode.md](docs/dev-mode.md) for configuration.

## Self-host locally

Prerequisites:

- Go `1.26.4` or newer in the `1.26` line
- Node.js `22.x`
- Bun `1.3.8` or newer
- Podman with Compose support
- `just`

Run the local demo stack:

```bash
cp .env.example .env
just diagnose
just demo
```

When the demo is ready:

| Service | Local URL |
| --- | --- |
| Web UI | `http://localhost:3000` |
| API | `http://localhost:8080` |
| Grafana | `http://localhost:3001` |

Manual stack commands are available when you want more control:

```bash
just setup
just images
just dev up
just dev init
just dev logs
```

Before exposing StageFlow on a public domain, replace every `change-me` value in `.env`,
set explicit CORS origins, keep API authentication enabled, and review
[docs/reference/configuration.md](docs/reference/configuration.md).

## Built-in scanners

- `axe` — accessibility checks via axe-core
- `lighthouse` — performance, best-practices, and PWA signals
- `seo` — title, metadata, heading, and structured-data checks
- `security-headers` — common HTTP response-header policy
- `link-checker` — internal and external link health
- `open-graph` — social preview metadata
- `spelling-grammar` — spelling and content-quality checks
- `ai-navigator` — agent-driven browser navigation traces

![StageFlow issue list](docs/images/report-issues.png)

## Code tour

- [docs/architecture/system.md](docs/architecture/system.md) — system design, flows, security model, and reviewer code map
- [libs/contracts/report](libs/contracts/report) — unified report schema and generated contract types
- [services/platform-api/internal/api](services/platform-api/internal/api) — intake, auth, SSRF guards, project APIs, SSE
- [services/orchestrator](services/orchestrator) — job FSM, container lifecycle, aggregation
- [services/scanner-runner](services/scanner-runner) — scanner runtime and Playwright integrations
- [clients/cli](clients/cli) — terminal UX, SSE streaming, report rendering, project mode
- [clients/web](clients/web) — React Router report UI
