# StageFlow Configuration

Reference for environment variables used by local and production StageFlow deployments.

Use `.env.example` as the local baseline. It favors localhost-friendly defaults, so override domain-facing values before public use. Keep secrets in a secret manager or host-level environment, never in git.

Start with [Self-hosting](../self-hosting.md) for setup, topology, and pre-deploy checks. This page only defines configuration variables.

## Variable Reference

### MinIO

| Variable              | Required | Default in `.env.example` | Purpose                                          |
| --------------------- | -------- | ------------------------- | ------------------------------------------------ |
| `MINIO_ROOT_USER`     | yes      | `minioadmin`              | MinIO root user used by MinIO service bootstrap. |
| `MINIO_ROOT_PASSWORD` | yes      | `change-me`               | MinIO root password.                             |
| `MINIO_ACCESS_KEY`    | yes      | `stageflow`               | App credential used by services to access MinIO. |
| `MINIO_SECRET_KEY`    | yes      | `change-me`               | Secret for `MINIO_ACCESS_KEY`.                   |
| `MINIO_STAGING_RETENTION_DAYS` | no | `1`                     | Positive-integer expiry window for submitted ZIP archives. |
| `MINIO_ARTIFACT_RETENTION_DAYS` | no | `1`                    | Positive-integer expiry window for authentication state, screenshots, logs, scanner output, and reports. |
| `MINIO_APPLY_LIFECYCLES` | no | `true` | Set `false` only during a baseline migration preparation pass so buckets and policy are updated before artifact lifecycles. |

MinIO applies lifecycle expiration asynchronously, so an object may remain briefly after its retention window. Provisioning imports deterministic lifecycle rules for the staging and artifact buckets and is safe to rerun. Promoted project reports are copied into the private `scanner-baselines` bucket, which has no lifecycle expiry; each remains until it is replaced or its project is deleted. This release does not provide immediate user-triggered deletion.

When upgrading a deployment that already has promoted baselines, run
`MINIO_APPLY_LIFECYCLES=false ./infra/minio/init-buckets.sh` first. This creates
the baseline bucket and updates the restricted application policy without
shortening existing artifact retention. Start the updated Platform API and
verify its baseline reconciliation summary before rerunning provisioning with
the default `MINIO_APPLY_LIFECYCLES=true`. A project listed under
`missing_legacy_projects` refers to an artifact that expired before the copy;
its diff endpoint returns HTTP 409 until a new completed job is promoted as the
baseline.

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
| `PLATFORM_API_TOKEN`                 | yes      | `change-me-platform-api-token`   | Server-side API token accepted via `X-Api-Key` or `Authorization: Bearer`; never expose it through `VITE_*` variables. |
| `PLATFORM_API_AUTH_DISABLED`         | no       | `false`                          | Explicit local-only opt-out when running the API without `PLATFORM_API_TOKEN`. Do not enable in public deployments. |
| `PLATFORM_API_PUBLIC_SUBMISSION_RATE_LIMIT_RPM` | no | `6` | Per-client refill rate shared by public URL and ZIP submissions. |
| `PLATFORM_API_PUBLIC_SUBMISSION_RATE_LIMIT_BURST` | no | `3` | Maximum public submissions a client may burst before receiving `429`. |
| `PLATFORM_API_ALLOW_PRIVATE_TARGETS` | no       | `false`                          | Controls whether the API accepts private/local targets.                                                             |
| `PLATFORM_API_TRUSTED_PROXIES`       | no       | `127.0.0.1/32,::1/128`          | Comma-separated trusted proxy CIDRs/IPs allowed to supply `X-Forwarded-For` for rate-limit keys.                    |
| `SCANNER_CONFIG_PATH`                | no       | `/data/scanners.yaml` in compose | Optional YAML scanner override file for enablement, image, and resource tweaks.                                     |
| `PROJECT_DB_PATH`                    | no       | `./projects.db` in code          | SQLite database path for project records, project/job mappings, and baselines.                                      |

`PLATFORM_API_TOKEN` is required at startup unless `PLATFORM_API_AUTH_DISABLED=true` is set. The disabled mode exists for local development only and should not be used on a public domain. The supplied Caddy reference injects this token only into an exact allowlist: anonymous and browser-auth submissions, ZIP uploads, status/report/results/SSE reads, and the scanner catalog. Browser clients never receive the credential. Projects, baselines, diffs, the caller-authenticated URL route, and unmatched API paths remain protected. Set `PLATFORM_API_TRUSTED_PROXIES` to only Caddy's exact source address (loopback in the reference topology) before relying on forwarded client IPs for rate limiting.

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
| `VITE_TAGLINE`                    | no       | `Self-hostable frontend quality regression scanning`             | Product-capability phrase used in marketing and metadata; omit the product name and trailing punctuation. |
| `VITE_DEPLOYMENT_MODE`            | no       | `self-hosted`                                                   | Set to `hosted-demo` only for the public-demo privacy and retention notices. |

The frontend container builds the React Router app to static files and serves them with nginx on port `3020`.

### Scanner Registry Overrides

Built-in scanner metadata is embedded from `libs/go/scannercatalog/manifests/*/manifest.json`. The checked-in compose files mount `infra/scanners/scanners.yaml` into both Platform API and Orchestrator as `/data/scanners.yaml`; `just setup` creates that local file from `infra/scanners/scanners.example.yaml` when needed. Use that tracked example as the public template for deployments that need to disable scanners, override images, or tune resource limits without editing embedded manifests. `services/orchestrator/config/scanners.yaml` is a service-local example/default, not the setup template.

Current built-in scanner IDs are: `axe`, `lighthouse`, `seo`, `security-headers`, `link-checker`, `open-graph`, and `spelling-grammar`.

### Caddy (Edge)

| Variable      | Required | Default in `.env.example` | Purpose                                                |
| ------------- | -------- | ------------------------- | ------------------------------------------------------ |
| `CADDY_EMAIL` | no       | not set                   | Email address used for Let's Encrypt TLS registration. |
| `PLATFORM_API_TOKEN` | yes | `change-me-platform-api-token` | Server-side credential Caddy overwrites onto all proxied API requests. |

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

## Related Docs

- [Self-hosting](../self-hosting.md)
- [Architecture](../architecture.md)
