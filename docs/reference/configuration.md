# StageFlow Configuration

Reference for environment variables used by local, staging, and production StageFlow deployments.

Use `.env.example` as the local baseline. It now favors localhost-friendly defaults, so override the domain-facing values before staging or production use. Keep secrets in your secret manager or host-level env, never in git.

If you are orienting yourself for the first time, start with the [repository README](../../README.md) for the quick start. This page is the detailed configuration reference once you know which environment you are setting up.

## Quick Start

```bash
cp .env.example .env
just diagnose
just demo
```

`just demo` is the fastest end-to-end smoke test. It is a guided wrapper around
the local `dev` stack: it runs setup, builds images, starts MinIO first,
initializes buckets, starts the full stack, and waits for readiness. For manual
control, run the individual steps yourself with `just setup`, `just images`,
`just dev up dev`, and `just dev init`.
The local `just` recipes default to compose project `stageflow_dev` and Podman
network `stageflow_dev_net`; set `COMPOSE_PROJECT_NAME` or
`STAGEFLOW_NETWORK_NAME` to isolate additional local stacks.

## Environment Topology

> The repo contains more than one valid deployment layout. Keep the compose overlay, public URLs, and edge proxy config aligned for the environment you are actually running.

| Mode                                  | When to use it                                        | Web                     | API                     | Grafana                 | Primary files                                                                |
| ------------------------------------- | ----------------------------------------------------- | ----------------------- | ----------------------- | ----------------------- | ---------------------------------------------------------------------------- |
| `dev` via `just demo` / `just dev up` | Fastest local smoke test                              | `http://localhost:3000` | `http://localhost:8080` | `http://localhost:3001` | `infra/compose/podman-compose.yml` + `infra/compose/podman-compose.test.yml` |
| `local` overlay                       | Localhost/private-target scans during development     | `http://localhost:3020` | `http://localhost:8080` | `http://localhost:3001` | `infra/compose/podman-compose.local.yml`                                     |
| repo-managed staging overlay          | Domain-like staging on alternate loopback ports       | `http://127.0.0.1:3300` | `http://127.0.0.1:8300` | `http://127.0.0.1:3301` | `infra/compose/podman-compose.staging.yml`                                   |
| optional self-hosted edge             | Public-domain routing and TLS for your own deployment | proxied by host Caddy   | proxied by host Caddy   | proxied by host Caddy   | `infra/caddy/Caddyfile`                                                      |
| hosted `stageflow.org` production     | Shared VPS deployment for the live demo               | gateway-managed         | gateway-managed         | gateway-managed         | separate deployment workspace (Quadlets + ingress network)                   |

The `dev` and `local` rows are both local development modes, not competing
deployment targets. Use `dev` for the normal demo/UI stack. Use `local` when a
scan must reach localhost or private-network targets from scanner pods; that
overlay serves the frontend on `3020` and switches job pods to host networking.

The repo-managed staging overlay, the optional self-hosted Caddy edge, and the hosted `stageflow.org` deployment intentionally use different topologies. `podman-compose.staging.yml` binds `3300/3301/8300/9300`, while the live demo runs behind a separate gateway + Quadlet-managed ingress network rather than repo-managed host port bindings. Use one topology per environment rather than mixing them.

The hosted `stageflow.org` demo uses the same application code, but this repository should be treated as the source for local development and self-hosted layouts rather than as the authoritative deployment automation for that public instance.

The optional Caddy edge expects a separate host-level loopback layout where the app is already exposed on `127.0.0.1:3100` (frontend), `127.0.0.1:3101` (Grafana), `127.0.0.1:8100` (API), and `127.0.0.1:9100` (MinIO). Those are not the same ports used by the local (`3000/3020/8080/3001`) or staging (`3300/8300/3301/9300`) compose overlays.

## Variable Reference

### MinIO

| Variable              | Required | Default in `.env.example` | Purpose                                          |
| --------------------- | -------- | ------------------------- | ------------------------------------------------ |
| `MINIO_ROOT_USER`     | yes      | `minioadmin`              | MinIO root user used by MinIO service bootstrap. |
| `MINIO_ROOT_PASSWORD` | yes      | `change-me`               | MinIO root password.                             |
| `MINIO_ACCESS_KEY`    | yes      | `stageflow`               | App credential used by services to access MinIO. |
| `MINIO_SECRET_KEY`    | yes      | `change-me`               | Secret for `MINIO_ACCESS_KEY`.                   |

### PostgreSQL

| Variable            | Required | Default in `.env.example`                                                | Purpose                           |
| ------------------- | -------- | ------------------------------------------------------------------------ | --------------------------------- |
| `POSTGRES_USER`     | yes      | `stageflow`                                                              | Postgres username.                |
| `POSTGRES_PASSWORD` | yes      | `change-me`                                                              | Postgres password.                |
| `POSTGRES_DB`       | yes      | `stageflow`                                                              | Primary Postgres database name.   |
| `DATABASE_URL`      | yes      | `postgres://stageflow:change-me@postgres:5432/stageflow?sslmode=disable` | DSN used by API and orchestrator. |

### Grafana

| Variable                     | Required | Default in `.env.example` | Purpose                                            |
| ---------------------------- | -------- | ------------------------- | -------------------------------------------------- |
| `GF_SECURITY_ADMIN_USER`     | yes      | `admin`                   | Grafana admin user.                                |
| `GF_SECURITY_ADMIN_PASSWORD` | yes      | `change-me`               | Grafana admin password.                            |
| `GF_SERVER_ROOT_URL`         | yes      | `http://localhost:3001`   | External URL Grafana uses for redirects and links. |

### Platform API

| Variable                             | Required | Default in `.env.example`        | Purpose                                                                                                             |
| ------------------------------------ | -------- | -------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| `PORT`                               | no       | `8080`                           | Platform API HTTP listen port inside the container.                                                                 |
| `NATS_URL`                           | yes      | `nats://nats:4222`               | NATS server URL used to publish job events and subscribe to lifecycle events.                                       |
| `ORCHESTRATOR_API_URL`               | yes      | `http://orchestrator:8081`       | Internal orchestrator admin API URL used for status snapshots.                                                      |
| `ORCHESTRATOR_API_TOKEN`             | yes      | `change-me-orchestrator-token`   | Inter-service token for calls from Platform API to Orchestrator.                                                    |
| `PLATFORM_API_TOKEN`                 | yes      | `change-me-platform-api-token`   | Public API token accepted via `X-Api-Key` or `Authorization: Bearer`.                                               |
| `PLATFORM_API_AUTH_DISABLED`         | no       | `false`                          | Explicit local-only opt-out when running the API without `PLATFORM_API_TOKEN`. Do not enable in public deployments. |
| `PLATFORM_API_ALLOW_PRIVATE_TARGETS` | no       | `false`                          | Controls whether the API accepts private/local targets.                                                             |
| `PLATFORM_API_TRUSTED_PROXIES`       | no       | empty                            | Comma-separated trusted proxy CIDRs/IPs allowed to supply `X-Forwarded-For` for rate-limit keys.                    |
| `SCANNER_CONFIG_PATH`                | no       | `/data/scanners.yaml` in compose | Optional YAML scanner override file for enablement, image, and resource tweaks.                                     |
| `PROJECT_DB_PATH`                    | no       | `./projects.db` in code          | SQLite database path for project records, project/job mappings, and baselines.                                      |

`PLATFORM_API_TOKEN` is required at startup unless `PLATFORM_API_AUTH_DISABLED=true` is set. The disabled mode exists for local development only and should not be used on a public domain.

### Orchestrator

| Variable                            | Required | Default in compose/code                     | Purpose                                                                              |
| ----------------------------------- | -------- | ------------------------------------------- | ------------------------------------------------------------------------------------ |
| `DATABASE_URL`                      | yes      | from `.env.example`                         | PostgreSQL DSN for orchestrator jobs, events, metrics, and state.                    |
| `PODMAN_SOCKET`                     | yes      | `/run/podman/podman.sock`                   | Podman API socket mounted into the orchestrator container.                           |
| `EXTRACTION_IMAGE`                  | yes      | `localhost/stageflow/extractor:latest`      | Image used for ZIP extraction job containers.                                        |
| `SCANNER_IMAGE`                     | yes      | `localhost/stageflow/scanner-runner:latest` | Default scanner runner image.                                                        |
| `SCANNER_IMAGE_OVERRIDE`            | no       | empty                                       | Global scanner image override when set.                                              |
| `API_PORT`                          | yes      | `8081`                                      | Orchestrator admin API port.                                                         |
| `ORCHESTRATOR_API_TOKEN`            | yes      | `change-me-orchestrator-token`              | Admin API token accepted via `X-Api-Key` or bearer auth.                             |
| `NATS_HOST`                         | yes      | `nats`                                      | Hostname injected into job pods so scanners can reach NATS.                          |
| `MINIO_HOST`                        | yes      | `minio`                                     | Hostname injected into job pods so scanners can reach MinIO.                         |
| `POD_NETWORK`                       | no       | effective `STAGEFLOW_NETWORK_NAME`          | Podman network for job pods in bridge mode.                                          |
| `POD_NETNS_MODE`                    | no       | `bridge` (`host` in local overlay)          | Network namespace mode for job pods. Use `host` only for local/private target scans. |
| `PAGE_LOAD_TIMEOUT`                 | no       | `15000`                                     | Browser page-load timeout in milliseconds.                                           |
| `A11Y_SCROLL_TIMEOUT`               | no       | `300`                                       | Accessibility scan scroll timeout in milliseconds.                                   |
| `JOB_EVENTS_RETENTION_DAYS`         | no       | `30`                                        | Retention window for stored job events.                                              |
| `JOB_EVENTS_PRUNE_INTERVAL_MINUTES` | no       | `60`                                        | Background job-event pruning interval.                                               |
| `JOB_EVENTS_PRUNE_BATCH_SIZE`       | no       | implementation default                      | Maximum number of old job events pruned per batch.                                   |
| `ORCHESTRATOR_ADMIN_RATE_LIMIT_RPS` | no       | `0` (disabled)                              | Per-client request/sec limit on the admin API. `0` disables rate limiting.           |
| `ORCHESTRATOR_ADMIN_RATE_LIMIT_BURST` | no     | falls back to RPS                           | Token-bucket burst for the admin API rate limiter.                                   |

> **Rate limiter scope:** admin API rate limiting is **in-memory, per orchestrator process** — an accepted single-instance tradeoff. The orchestrator admin API is not public; it is reached only by the Platform API over the internal network. If you ever run multiple orchestrator replicas *and* expose the admin API publicly, enforce limits at a shared edge proxy (e.g. Caddy) instead of relying on per-process counters.

### Public Domain and CORS

| Variable                          | Required | Default in `.env.example`                                                                 | Purpose                                               |
| --------------------------------- | -------- | ----------------------------------------------------------------------------------------- | ----------------------------------------------------- |
| `STAGEFLOW_PUBLIC_DOMAIN`         | yes      | `localhost`                                                                               | Public domain used in generated URLs and edge config. |
| `PLATFORM_API_CORS_ALLOW_ORIGINS` | yes      | `http://localhost:3000,http://127.0.0.1:3000,http://localhost:8080,http://localhost:3020,http://127.0.0.1:3020` | Browser origin allowlist for API requests. |

### Web App

| Variable                          | Required | Default in `.env.example`                                       | Purpose                                           |
| --------------------------------- | -------- | --------------------------------------------------------------- | ------------------------------------------------- |
| `VITE_API_URL`                    | yes      | `http://localhost:8080`                                         | Web app API base URL.                             |
| `VITE_SITE_TITLE`                 | no       | `StageFlow`                                                     | Site title shown in UI metadata.                  |
| `VITE_SITE_URL`                   | yes      | `http://localhost:3000`                                         | Canonical site URL used for metadata/share cards. |
| `VITE_GITHUB_URL`                 | no       | `https://github.com/mattboback/stageflow`                       | GitHub link shown in UI.                          |
| `VITE_TAGLINE`                    | no       | `Podman-native web accessibility and quality scanning platform` | Marketing tagline in UI surfaces.                 |
| `VITE_AI_NAVIGATOR_DEFAULT_MODEL` | no       | `openai/gpt-4o-mini`                                            | Default model shown for AI navigator flows.       |

The frontend container builds the React Router app to static files and serves them with nginx on port `3020`.

### AI Navigator (Optional)

| Variable                     | Required                     | Default in `.env.example` | Purpose                                                     |
| ---------------------------- | ---------------------------- | ------------------------- | ----------------------------------------------------------- |
| `OPENROUTER_API_KEY`         | only if AI navigator enabled | empty                     | API key for OpenRouter model calls.                         |
| `OPENROUTER_APP_TITLE`       | no                           | `StageFlow`               | OpenRouter request attribution title.                       |
| `OPENROUTER_APP_REFERER`     | no                           | `http://localhost:3000`   | OpenRouter request attribution referer.                     |
| `AI_NAVIGATOR_DEFAULT_MODEL` | no                           | `openai/gpt-4o-mini`      | Default backend model when scanner options do not override. |

### Scanner Registry Overrides

Built-in scanner metadata is embedded from `libs/go/scannercatalog/manifests/*/manifest.json`. The checked-in compose files mount `infra/scanners/scanners.yaml` into both Platform API and Orchestrator as `/data/scanners.yaml`; `just setup` creates that local file from `infra/scanners/scanners.example.yaml` when needed. Use that tracked example as the public template for deployments that need to disable scanners, override images, or tune resource limits without editing embedded manifests. `services/orchestrator/config/scanners.yaml` is a service-local example/default, not the setup template.

Current built-in scanner IDs are: `axe`, `lighthouse`, `seo`, `security-headers`, `link-checker`, `open-graph`, `spelling-grammar`, and `ai-navigator`.

### Caddy (Edge)

| Variable      | Required | Default in `.env.example` | Purpose                                                |
| ------------- | -------- | ------------------------- | ------------------------------------------------------ |
| `CADDY_EMAIL` | no       | not set                   | Email address used for Let's Encrypt TLS registration. |

This section refers to the optional host-level edge proxy example under `infra/caddy/` for self-hosted installs. The hosted `stageflow.org` production gateway is managed from a separate deployment workspace rather than from this repository.

### Advanced infrastructure overrides

Most first-time local setups can ignore this section. These variables are mainly for custom or advanced self-hosted infrastructure layouts.

| Variable                        | Purpose                                             |
| ------------------------------- | --------------------------------------------------- |
| `MINIO_ENDPOINT`                | Internal host/port for MinIO API.                   |
| `MINIO_USE_SSL`                 | Whether internal MinIO connections use SSL.         |
| `MINIO_USE_PROXY_URLS`          | Legacy compatibility flag; artifact access is still signed through the configured public endpoint. |
| `MINIO_REGION`                  | MinIO region configuration.                         |
| `MINIO_ARTIFACT_BUCKET`         | Name of the bucket used for storing artifacts.      |
| `GF_AUTH_ANONYMOUS_ENABLED`     | Enables anonymous access to Grafana dashboards.     |
| `GF_SERVER_HTTP_PORT`           | Internal HTTP port for Grafana.                     |
| `GF_SERVER_SERVE_FROM_SUB_PATH` | Whether Grafana serves from a sub-path.             |

## Environment Guidance

- Use distinct credentials for local, staging, and production.
- Keep domains and CORS origins environment-specific.
- For self-hosted public domains, either route StageFlow through an existing host-level gateway or use `infra/caddy/Caddyfile` as the starting point for your own edge config.
- The hosted `stageflow.org` demo runs on a separate Quadlet-managed ingress topology; do not expect the repo-local compose overlays to mirror that production host layout 1:1.
- Keep committed screenshots and reviewer-facing images under `docs/images`.
  Ephemeral QA/build evidence belongs under ignored artifact, output, or cache
  paths such as `artifacts/`, `output/`, and `.cache/`; `just clean` may remove
  those files.

## Pre-Deploy Validation

1. `just build` passes.
2. `just ci` passes.
3. One URL scan and one ZIP scan complete successfully.
4. SSE stream updates and final report retrieval both work.

## Related Docs

- [README](../../README.md)
- [Architecture](../architecture/system.md)
