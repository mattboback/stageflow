# StageFlow

StageFlow is a frontend quality gate for accessibility, performance, SEO,
links, security headers, social metadata, content quality, and agent-driven
navigation checks. It gives developers a CLI-first workflow, a hosted report UI
at `stageflow.org`, and a self-hostable Podman stack for teams that want to run
the whole system themselves.

![StageFlow report overview](docs/images/report-overview.png)

## Start With the CLI

Use the CLI when you want a terminal-friendly scan result, JSON for automation,
or a CI gate based on issue severity.

Download a release asset from
[GitHub Releases](https://github.com/mattboback/stageflow/releases), or build
from this repository:

```bash
just cli-install
stageflow version
```

Run a hosted scan:

```bash
stageflow scan https://example.com --api https://stageflow.org
```

Common CLI workflows:

```bash
# Choose scanners and return JSON for automation.
stageflow scan https://example.com --scanners axe,seo --format json --api https://stageflow.org

# Fail the command when serious-or-worse issues are found.
stageflow scan https://example.com --fail-on serious --api https://stageflow.org

# Scan several routes in one job.
stageflow scan https://example.com https://example.com/about \
  --format markdown \
  --api https://stageflow.org

# Scan a local dev server through a local StageFlow API.
stageflow scan http://127.0.0.1:5173 --allow-private-targets
```

Exit codes are intentionally machine-readable:

| Exit code | Meaning                                                             |
| --------- | ------------------------------------------------------------------- |
| `0`       | Scan completed and no displayed issue met the `--fail-on` threshold |
| `1`       | Scan completed and at least one displayed issue met the threshold   |
| `2`       | CLI or API error                                                    |

For the full command reference, see [clients/cli/README.md](clients/cli/README.md)
and [docs/reference/cli/stageflow](docs/reference/cli/stageflow).

## Hosted StageFlow

The hosted `stageflow.org` service runs the same application code as this repo
and is the fastest way to try StageFlow without operating the stack. Use it from
the CLI with `--api https://stageflow.org`, or use the browser UI for scan
submission, live status, and report review.

The report UI is built for triage:

- score and severity distribution at a glance
- issue grouping by scanner/rule
- page screenshots and occurrence evidence
- remediation details for engineers
- downloads for JSON and HTML report artifacts

![StageFlow issue list](docs/images/report-issues.png)

Hosted production deployment, monitoring, rollback, and VPS control-plane
automation are intentionally managed outside this public repository. This repo
contains the application source, local development stack, and self-hosting
examples.

## Project Mode

Project Mode turns the CLI into a repeatable frontend regression loop. It can
start your dev server, wait for readiness, run a scan, stream progress, stop the
server, and emit JSON that agents or CI jobs can parse.

```bash
stageflow project init
stageflow project doctor
stageflow project --format json
```

After a project is associated with a hosted StageFlow project, run the hosted
baseline/diff loop from the same repo:

```bash
stageflow project hosted --format json
```

See [docs/PROJECT_MODE.md](docs/PROJECT_MODE.md) for configuration details.

## Self-Host Locally

The local stack uses Podman, NATS JetStream, PostgreSQL, MinIO, the Go Platform
API and Orchestrator, the TypeScript scanner runner, and the SvelteKit web app.

Prerequisites:

- Go `1.26.4` or newer in the `1.26` line
- Node.js `22.x`
- Bun `1.3.8` or newer
- Podman with Compose support
- `just`

Run the fastest local smoke test:

```bash
cp .env.example .env
just diagnose
just demo
```

When the demo is ready:

- Web UI: `http://localhost:3000`
- API: `http://localhost:8080`
- Grafana: `http://localhost:3001`

Manual stack commands are available when you want more control:

```bash
just setup
just images
just dev up
just dev init
just dev logs
```

Before exposing StageFlow on a public domain, replace every `change-me` value
in `.env`, set explicit CORS origins, keep API authentication enabled, and
review [docs/reference/configuration.md](docs/reference/configuration.md) and
[infra/security/egress-policy.example.md](infra/security/egress-policy.example.md).

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
The AI Navigator is optional and requires `OPENROUTER_API_KEY` when enabled.

## Repository Map

| Path                         | Purpose                                                                  |
| ---------------------------- | ------------------------------------------------------------------------ |
| `clients/cli`                | Go CLI for scans, Project Mode, baselines, reports, and docs generation  |
| `clients/web`                | SvelteKit app for hosted/local browser workflows                         |
| `services/platform-api`      | Public HTTP API, auth, CORS, URL/ZIP intake, SSE, projects, reports      |
| `services/orchestrator`      | Job state machine, Podman pod lifecycle, aggregation, persistence        |
| `services/archive-extractor` | Safe ZIP extraction and static serving inside job pods                   |
| `services/scanner-runner`    | Bun/TypeScript Playwright scanner runtime                                |
| `libs/contracts`             | JSON Schema contracts and generated Go/TypeScript types                  |
| `infra`                      | Compose, Caddy, MinIO, Grafana, scanner, and security examples           |
| `docs`                       | Architecture, Project Mode, configuration, operations, and CLI reference |

For the system architecture and trust boundaries, start with
[ARCHITECTURE.md](ARCHITECTURE.md), then read
[docs/architecture/system.md](docs/architecture/system.md).

## Development

Install dependencies and local config:

```bash
just setup
```

Run the repository quality gate:

```bash
just ci
```

Useful focused checks:

```bash
just shell-tests
just storybook-test
just dead-code
just project-golden
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for pull request expectations and
workspace-specific validation commands.

## Security

StageFlow scans websites and can run browser automation, so deployment
boundaries matter. Public deployments should keep API authentication enabled,
avoid private-target scans unless explicitly intended, isolate scanner pods, and
use environment-specific credentials.

Report vulnerabilities privately through [SECURITY.md](SECURITY.md). Do not
open public issues for vulnerabilities.

## License

StageFlow is released under the [MIT License](LICENSE).
