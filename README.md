# StageFlow

Podman-native web accessibility and quality scanning platform.

StageFlow runs multi-scanner audits against live URLs or static-site ZIP archives, then aggregates outputs into one normalized report stream. It is built for self-hosting, strict intake validation, and operational transparency.

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

Full design details: [ARCHITECTURE.md](ARCHITECTURE.md).

## Built-In Scanners

| Scanner | Focus |
| --- | --- |
| `axe` | Accessibility (WCAG rule violations) |
| `lighthouse` | Performance and quality audits |
| `seo` | SEO best-practice checks |
| `security-headers` | HTTP security header posture |
| `link-checker` | Broken link detection |
| `ai-navigator` | Goal-driven browser flow evaluation |

## Tech Stack

| Layer | Technology |
| --- | --- |
| Backend services | Go 1.25 |
| Scanner runtime | TypeScript + Bun + Playwright |
| Frontend | SvelteKit 5 + Tailwind v4 |
| Messaging | NATS JetStream |
| Storage | MinIO (artifacts) + PostgreSQL (job state/events) |
| Container runtime | Podman |
| Edge/proxy | Caddy |
| Monitoring | Grafana |

## Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [Bun](https://bun.sh/)
- [Podman](https://podman.io/) (with `podman compose`)
- [just](https://github.com/casey/just)
- [golangci-lint v2](https://golangci-lint.run/)

## Quick Start

```bash
git clone https://github.com/mattboback/stageflow.git
cd stageflow

cp .env.example .env

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

## Day-to-Day Commands

Run `just help` for the full recipe list.

| Area | Command | Purpose |
| --- | --- | --- |
| Setup | `just setup` | Install deps, sync workspace, create network |
| Local stack | `just dev up` / `just dev down` / `just dev logs` | Start, stop, or inspect local compose stack |
| Staging stack | `just staging up` / `just staging down` | Manage staging compose environment |
| Build | `just build` | Build Go services, frontend, scanner-runner |
| Images | `just images` | Build container images |
| Quality | `just ci` | Lint, typecheck, tests, audits |
| Service run | `just run frontend` / `just run api` / `just run orchestrator` | Run one service locally |
| Production | `just prod <cmd>` / `just deploy <mode>` | Quadlet and deploy workflows |

## Scanner Plugin System

Scanners are discovered via manifest files and loaded dynamically by the scanner runtime.

Discovery paths (in order):

1. Built-in scanners in `platform/scanner-runner/src/scanners`
2. Mounted `/plugins`
3. User plugins at `~/.stageflow/plugins`
4. Extra paths from `PLUGIN_PATHS`

To add a custom scanner:

1. Implement a scanner module (extends `ScannerBase`).
2. Add a valid `manifest.json` (schema-backed).
3. Make the plugin available in a discovery path.

Reference docs:

- [ARCHITECTURE.md](ARCHITECTURE.md#scanner-plugin-system)
- `packages/contracts/scanner-manifest/schema/README.md`

## Security and Runtime Boundaries

- URL intake blocks private/loopback/link-local/metadata targets.
- ZIP extraction enforces archive safety limits and path sanitization.
- Scanner execution is containerized per job.
- API status streaming uses SSE with reconnect-safe behavior.
- Edge rate limiting is expected at proxy/load-balancer/CDN layers.

See [SECURITY.md](SECURITY.md) and [ARCHITECTURE.md](ARCHITECTURE.md#security-and-trust-boundaries).

## Repository Layout

```text
stageflow/
|- platform/              # API, orchestrator, extractor, scanner-runner
|- frontend/              # SvelteKit app
|- packages/              # Contracts + shared Go modules
|- infra/                 # Compose, Caddy, Quadlets, monitoring, scanner config
|- tools/                 # job-status-cli, suite-runner
|- tests/                 # End-to-end tests
`- scripts/               # Build/deploy scripts
```

## Documentation Map

- [ARCHITECTURE.md](ARCHITECTURE.md): deep system design, flows, and constraints
- [OPERATIONS.md](OPERATIONS.md): runbook for startup, health checks, and incident response
- [CONFIGURATION.md](CONFIGURATION.md): environment and deployment configuration guide
- [CONTRIBUTING.md](CONTRIBUTING.md): local workflow, standards, and PR checklist
- [SECURITY.md](SECURITY.md): vulnerability reporting policy
- [SUPPORT.md](SUPPORT.md): help channels and debugging checklist
- [tools/README.md](tools/README.md): operational tooling (`job-status-cli`, `suite-runner`)
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md): community conduct standards
- [CHANGELOG.md](CHANGELOG.md): release history

## Contributing

Contributions are welcome. Start with [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE) © 2025 Matthew Boback
