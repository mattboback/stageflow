# Architecture

StageFlow is a multi-service scanning platform that runs browser-driven audits inside rootless Podman containers and merges the results into one normalized report. This document covers the system shape, data flow, and key decisions. For the full treatment, see [docs/architecture/system.md](docs/architecture/system.md).

## System shape

```text
Client (Web UI / CLI)
  -> Platform API (Go)              Intake validation, SSRF guards, SSE streaming
    -> NATS JetStream               Async job dispatch and event replay
      -> Orchestrator (Go)           Job state machine, container lifecycle
        -> Podman job pod
             |- Archive extractor    (Go, ZIP jobs only)
             `- Scanner runners      (TypeScript / Bun / Playwright)

Artifacts flow back:
  Scanner output  -> MinIO (object storage)
  Job state       -> PostgreSQL
  Unified report  -> Platform API -> Client
```

## Repository layout

| Directory | Contents | Language |
| --- | --- | --- |
| `clients/web` | SvelteKit frontend — submission, live status, report views | TypeScript, Svelte 5 |
| `clients/cli` | `stageflow` CLI — scan, project mode, report rendering | Go |
| `services/platform-api` | Intake API, SSRF validation, SSE stream, project CRUD | Go |
| `services/orchestrator` | Job FSM, Podman pod lifecycle, report aggregation | Go |
| `services/scanner-runner` | Scanner plugin runtime, Playwright-based execution | TypeScript, Bun |
| `services/archive-extractor` | Secure ZIP extraction and provenance generation | Go |
| `libs/contracts` | JSON schemas and generated TypeScript/Go types | JSON Schema |
| `libs/go/*` | Shared Go libraries (config, domain, events, messaging, models, storage) | Go |
| `infra/compose` | Podman compose files (dev, staging, test overlays) | YAML |
| `infra/scanners` | Scanner container configuration | YAML |
| `devtools/` | Internal ops and QA helpers | Go, Bash |
| `qa/` | E2E tests and golden fixtures | Go, Bash |

## Job lifecycle

```
PENDING -> EXTRACTING -> READY_TO_SCAN -> SCANNING -> COMPLETING -> DONE
                                                                     |
                                            (any stage) ----------> FAILED
```

| State | What happens |
| --- | --- |
| PENDING | Job created, URLs validated, creation event published to NATS |
| EXTRACTING | Archive extractor downloads ZIP and generates provenance (skipped for URL jobs) |
| READY_TO_SCAN | Pages extracted, orchestrator plans scanner launches |
| SCANNING | Scanner containers running in parallel inside a Podman pod, SSE events streaming |
| COMPLETING | All scanners finished, orchestrator aggregates findings into a unified report |
| DONE | Report uploaded to MinIO, terminal SSE event sent, job artifacts available |

## Scanner plugin system

Each scanner is defined by a JSON manifest (`libs/go/scannercatalog/manifests/*/manifest.json`) specifying its ID, description, categories, browser requirements, concurrency limits, and estimated timing.

The scanner-runner loads the manifest at container startup, launches Playwright, iterates through target pages, and publishes per-page and per-scanner completion events over NATS. All scanners produce findings in a normalized contract (`libs/contracts/report`) so the orchestrator and frontend never handle scanner-specific logic.

**Built-in scanners:** axe (WCAG accessibility), lighthouse (performance/SEO/best-practices), seo (meta/structured data), security-headers (HTTP header posture), link-checker (broken links), open-graph (social metadata), spelling-grammar (content quality), ai-navigator (LLM-driven journey simulation).

## Key design decisions

**Podman over Docker** — Rootless containers with no daemon dependency. Scanner pods run unprivileged, which simplifies the security model for self-hosted deployments.

**NATS JetStream for dispatch** — Decouples the API from the orchestrator. JetStream provides durable streams with replay, so the orchestrator can recover missed events after a restart.

**SSE over WebSocket** — Simpler protocol, works through proxies and CDNs, sufficient for unidirectional job progress streaming. The API maintains an in-memory hub per job; clients reconnect automatically.

**Contract-driven reports** — All 8 scanners conform to one JSON schema. The frontend, CLI, and diff engine operate on the unified contract without scanner-specific branches.

**Dual surface (web + CLI)** — Same API, same report contract, two rendering paths. The CLI adds `--fail-on` severity gating and `--format json` for CI integration.

## Tech stack

Go 1.26 | SvelteKit 5 / Svelte 5 | TypeScript | Bun | Podman | NATS JetStream | MinIO | PostgreSQL | Playwright | axe-core | Lighthouse

## Further reading

- [Full architecture deep-dive](docs/architecture/system.md)
- [Configuration reference](docs/reference/configuration.md)
- [CLI documentation](clients/cli/README.md)
- [Contributing guide](.github/CONTRIBUTING.md)
