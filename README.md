# StageFlow

**Podman-native web accessibility scanning platform.**

StageFlow is a polyglot microservices platform that orchestrates web accessibility and quality scans. Submit a URL or upload a ZIP of static HTML, and StageFlow spins up containerized scanners — [axe-core](https://github.com/dequelabs/axe-core), [Lighthouse](https://github.com/GoogleChrome/lighthouse), SEO, security headers, link checking, and an optional AI-powered navigator — then aggregates results into a unified report delivered via Server-Sent Events.

## Project Write-Up (Implementation Snapshot)

I built StageFlow to run accessibility and quality scans in infrastructure I control, with clear runtime boundaries between API intake, orchestration, scanner execution, and reporting. In practice, the frontend submits jobs to a small Go API surface (`/api/v1/jobs/zip`, `/api/v1/jobs/urls`, `/api/v1/jobs/{id}`, `/api/v1/jobs/{id}/stream`, `/api/v1/scanners`) and then follows progress over SSE.

The intake path is intentionally strict. URL submissions are capped at 2 MB, limited to 100 URLs, and validate each target as `http`/`https` with host checks that block loopback, private, link-local, and metadata-address destinations. ZIP uploads are capped at 100 MB and support scanner module selection plus per-scanner config payloads. For ZIP jobs, files land in a staging bucket first; extraction is handled out-of-process by the extractor service.

The extractor enforces defensive archive handling before any scan runs: entry-count limits, max expansion ratio (ZIP bomb protection), max uncompressed size limits, per-entry size limits, and path sanitization to reject traversal/absolute-path tricks. For URL jobs, the orchestrator skips extraction and starts scanners directly. For ZIP jobs, it creates a pod, runs extraction, waits for `extraction.ready`, then transitions into scanning.

Scanning itself is plugin-driven. The scanner-runner discovers scanner manifests, loads the requested module, validates scanner identity against its manifest, and validates `SCANNER_OPTIONS` against the manifest schema. The orchestrator starts one container per selected scanner module and applies resource limits from scanner definitions (with defaults when not specified). Current built-ins are `axe`, `lighthouse`, `seo`, `security-headers`, `link-checker`, plus `ai-navigator` as an optional module.

What made this project interesting was not just running tools, but normalizing their output into one coherent report. On completion, the orchestrator downloads scanner outputs, merges page and issue data, deduplicates overlapping rules across scanners (for example `axe`/`lighthouse`/`seo` overlap on some checks) using explicit scanner priority, recalculates severity totals, and publishes a unified report artifact set. On the frontend, the scan status store keeps live progress readable with SSE reconnect behavior and a fetch fallback, so users still get a trustworthy terminal state if streaming drops.

### Implementation References

- API routes and intake validation: `platform/api/internal/api/router.go`, `platform/api/internal/api/handlers_jobs_url_submit.go`, `platform/api/internal/api/handlers_jobs_zip_upload.go`
- URL security policy (SSRF guardrails): `platform/api/internal/api/security.go`
- ZIP validation and safe extraction: `platform/extractor/internal/extractor/extractor.go`
- Job lifecycle and scanner startup: `platform/orchestrator/internal/orchestrator/events.go`, `platform/orchestrator/internal/orchestrator/extraction.go`, `platform/orchestrator/internal/orchestrator/scanning.go`
- Aggregation and cross-scanner deduplication: `platform/orchestrator/internal/orchestrator/report_aggregator_aggregate.go`, `platform/orchestrator/internal/orchestrator/rule_deduplication.go`
- Plugin loader and schema validation: `platform/scanner-runner/src/worker.ts`
- Scanner manifests (capabilities/config schemas): `packages/shared-go/scannercatalog/manifests/*/manifest.json`
- Frontend submit + live status flow: `frontend/src/lib/api/client.ts`, `frontend/src/lib/stores/scan-status.svelte.ts`

## Docs

- Architecture: [ARCHITECTURE.md](ARCHITECTURE.md)
- Contributing: [CONTRIBUTING.md](CONTRIBUTING.md)
- Code of Conduct: [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- Security: [SECURITY.md](SECURITY.md)
- Support: [SUPPORT.md](SUPPORT.md)
- Changelog: [CHANGELOG.md](CHANGELOG.md)

## Architecture

```
 Client → SvelteKit SPA → Go API → NATS JetStream → Go Orchestrator
                                                          ↓
                                              Podman containers
                                        ┌─────────┬─────────────┐
                                        │Extractor│  Scanners   │
                                        │  (Go)   │(TS/Bun/PW)  │
                                        └─────────┴─────────────┘
                                              ↓
                                     MinIO (results) → Unified Report
```

See [ARCHITECTURE.md](ARCHITECTURE.md) for the full system design, data flows, state machine, plugin system, and database schemas.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.25 (API, Orchestrator, Extractor) |
| Scanner Runtime | TypeScript / Bun / Playwright |
| Frontend | SvelteKit 5 (runes, Tailwind v4) |
| Messaging | NATS JetStream |
| Object Storage | MinIO (S3-compatible) |
| Containers | Podman (pods, volumes, networking) |
| Database | PostgreSQL |
| Reverse Proxy | Caddy (auto-HTTPS) |
| Monitoring | Grafana |

## Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [Bun](https://bun.sh/)
- [Podman](https://podman.io/) (with `podman compose`)
- [just](https://github.com/casey/just) (command runner)
- [golangci-lint v2](https://golangci-lint.run/)

## Quick Start

```bash
# Clone
git clone https://github.com/mattboback/stageflow.git
cd stageflow

# Copy environment config
cp .env.example .env
# The defaults are local-dev friendly; you can use any non-empty values for local credentials.

# Install dependencies and create Podman network
just setup

# Start the local stack (NATS, MinIO, Grafana, API, orchestrator, frontend)
just dev up

# Initialize MinIO buckets
just dev init

# Build dynamic job images (extractor + scanner-runner)
just images
```

Tip: `just demo` runs the full local setup + prints a ready-to-run URL scan command.

### Local Demo (URL Scan)

1. Open the UI: `http://localhost:3000`
2. Or submit a URL job via API:

```bash
job_id="$(
  curl -sS -X POST http://localhost:8080/api/v1/jobs/urls \
  -H 'content-type: application/json' \
  -d '{"urls":["https://example.com"]}' \
  | jq -r .job_id
)"

echo "$job_id"
```

3. Stream progress (SSE):

```bash
curl -N "http://localhost:8080/api/v1/jobs/$job_id/stream"
```

Note: URL scans enforce SSRF protections and will reject loopback/private/metadata targets.

## Justfile Commands

StageFlow uses [just](https://github.com/casey/just) as its command runner. Run `just help` to see all available recipes.

| Group | Command | Description |
|-------|---------|-------------|
| **setup** | `just setup` | One-time setup: Podman network, Go/Bun deps |
| **dev** | `just dev [up\|down\|restart\|logs\|init]` | Local dev stack via compose |
| **staging** | `just staging [up\|down\|restart\|logs\|init\|ps]` | Staging stack via compose |
| **build** | `just build` | Build all artifacts (Go + frontend + runner) |
| **build** | `just images` | Build container images |
| **quality** | `just ci` | Run local CI: lint + typecheck + test |
| **run** | `just run [frontend\|api\|orchestrator]` | Run a service locally |
| **prod** | `just prod [install\|up\|down\|restart\|logs\|ps\|health]` | Manage production Quadlets |
| **prod** | `just deploy [full\|quick]` | Deploy production |
| **cleanup** | `just clean [all\|deep]` | Remove build artifacts |

## Scanner Plugins

StageFlow uses a plugin system for scanners. Each scanner is a self-contained module discovered via a `manifest.json`:

| Scanner | Category | Description |
|---------|----------|-------------|
| **axe** | Accessibility | axe-core WCAG 2.x violations |
| **lighthouse** | Performance/Quality | Google Lighthouse audits |
| **seo** | SEO | SEO best practices |
| **security-headers** | Security | HTTP security header analysis |
| **link-checker** | Quality | Broken link detection |
| **ai-navigator** | Accessibility | LLM-powered page navigation |

Custom scanners can be added by:
1. Creating a directory with a `manifest.json` and scanner module
2. Mounting it into the scanner container at `/plugins`
3. Or placing it in `~/.stageflow/plugins`

See [ARCHITECTURE.md § Scanner Plugin System](ARCHITECTURE.md#scanner-plugin-system) for the full manifest schema and lifecycle.

## Self-Hosting

StageFlow is designed for self-hosting. All domain-specific configuration is driven by environment variables:

```bash
# .env
STAGEFLOW_PUBLIC_DOMAIN=your-domain.com    # Used by Caddy and presigned URLs
PLATFORM_API_CORS_ALLOW_ORIGINS=https://your-domain.com,https://www.your-domain.com
VITE_API_URL=https://your-domain.com       # Frontend API endpoint
VITE_SITE_URL=https://your-domain.com      # Frontend site URL
```

**Deployment options:**
- **Compose** (`just dev` / `just staging`) — for development and staging
- **Quadlets** (`just prod`) — systemd-managed Podman containers for production

## Project Structure

```
stageflow/
├── frontend/              # SvelteKit 5 SPA
├── platform/
│   ├── api/               # Go REST API + SSE
│   ├── orchestrator/      # Go job coordination + container management
│   ├── extractor/         # Go ZIP extraction service
│   └── scanner-runner/    # TypeScript/Bun scanner runtime
├── packages/
│   ├── contracts/         # Shared schemas (report, scanner-manifest, events)
│   └── shared-go/         # Shared Go libraries
├── infra/
│   ├── compose/           # Podman Compose files
│   ├── caddy/             # Reverse proxy config
│   ├── quadlets/          # Systemd Quadlet templates
│   ├── minio/             # Bucket initialization
│   ├── grafana/           # Dashboards and datasources
│   └── scanners/          # Scanner configuration overrides
├── tools/
│   ├── job-status-cli/    # CLI for inspecting jobs and system status
│   └── suite-runner/      # Test suite runner for integration testing
├── scripts/               # Build and deployment scripts
└── tests/                 # E2E tests and fixtures
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, code style, and PR guidelines.

## License

[MIT](LICENSE) © 2025 Matthew Boback
