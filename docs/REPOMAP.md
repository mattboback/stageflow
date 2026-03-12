# Repository Map

This document provides a high-level overview of the StageFlow repository structure, helping contributors navigate the codebase.

## Root Directory

- `platform/` - Core backend microservices and scanner runtime
- `frontend/` - SvelteKit web application
- `packages/` - Shared code, JSON schemas, and generated contracts
- `infra/` - Deployment configuration, Podman compose files, Caddy configs, and Quadlets
- `tools/` - CLI tools for interacting with the platform
- `tests/` - End-to-end integration tests
- `scripts/` - Build, test, and utility scripts
- `docs/` - Architecture, configuration, and tooling documentation

## Platform Services (`/platform`)

### `api/`
The Platform API is the public-facing REST interface. It handles job submission, validation, SSRF protections, and SSE streaming for live status.
- **Language:** Go
- **Key Files:** `internal/api/router.go`, `internal/api/handlers_*.go`

### `orchestrator/`
The Orchestrator is the state machine engine. It consumes NATS events, manages Podman containers for scanners, and aggregates scanner outputs into unified reports.
- **Language:** Go
- **Key Files:** `internal/orchestrator/events.go`, `internal/application/jobs/`, `internal/domain/jobs/`

### `extractor/`
The Extractor handles secure decompression of uploaded ZIP archives before scanning. It enforces strict limits to prevent zip bombs and path traversal.
- **Language:** Go
- **Key Files:** `internal/extractor/extractor.go`

### `scanner-runner/`
The Scanner Runner is the execution environment for individual scanner plugins. It loads manifests, runs Playwright browsers, and uploads artifacts.
- **Language:** TypeScript / Bun
- **Key Files:** `src/worker.ts`, `src/core/scanner-base.ts`, `src/scanners/`

## Frontend (`/frontend`)

The web UI for StageFlow, built with SvelteKit and Tailwind CSS.
- **Key Directories:**
  - `src/routes/` - Application pages (Playground, Report, Status)
  - `src/lib/components/` - Reusable UI components
  - `src/lib/api/` - API client and SSE integration

## Shared Packages (`/packages`)

### `contracts/`
JSON Schemas that define the data structures shared across the platform (e.g., Unified Report, Scanner Manifests).

### `shared-go/`
Common Go utilities used by the API and Orchestrator.
- **Key Areas:** NATS messaging (`messaging/`), Event structures (`events/`), Configuration loaders (`config/`)

## Infrastructure (`/infra`)

- `compose/` - Local development and staging Podman Compose definitions.
- `quadlets/` - Systemd Quadlet templates for production deployment.
- `caddy/` - Reverse proxy configurations.
- `minio/` - Object storage initialization scripts.
- `grafana/` - Monitoring dashboards and provisioning.

## Tooling (`/tools`)

- `stageflow-cli/` - The primary command-line interface for submitting jobs and fetching reports.
- `job-status-cli/` - Operator tool for inspecting job state and NATS events.
- `suite-runner/` - Tool for running batch scans across multiple domains with threshold validation.
