# StageFlow Configuration

Reference for environment variables used by local, staging, and production StageFlow deployments.

Use `.env.example` as the baseline. Keep secrets in your secret manager or host-level env, never in git.

## Quick Start

```bash
cp .env.example .env
just setup
just dev up
just dev init
just images
```

## Variable Reference

### MinIO

| Variable | Required | Default in `.env.example` | Purpose |
| --- | --- | --- | --- |
| `MINIO_ROOT_USER` | yes | `minioadmin` | MinIO root user used by MinIO service bootstrap. |
| `MINIO_ROOT_PASSWORD` | yes | `change-me` | MinIO root password. |
| `MINIO_ACCESS_KEY` | yes | `stageflow` | App credential used by services to access MinIO. |
| `MINIO_SECRET_KEY` | yes | `change-me` | Secret for `MINIO_ACCESS_KEY`. |

### PostgreSQL

| Variable | Required | Default in `.env.example` | Purpose |
| --- | --- | --- | --- |
| `POSTGRES_USER` | yes | `stageflow` | Postgres username. |
| `POSTGRES_PASSWORD` | yes | `change-me` | Postgres password. |
| `POSTGRES_DB` | yes | `stageflow` | Primary Postgres database name. |
| `DATABASE_URL` | yes | `postgres://stageflow:change-me@postgres:5432/stageflow?sslmode=disable` | DSN used by API and orchestrator. |

### Grafana

| Variable | Required | Default in `.env.example` | Purpose |
| --- | --- | --- | --- |
| `GF_SECURITY_ADMIN_USER` | yes | `admin` | Grafana admin user. |
| `GF_SECURITY_ADMIN_PASSWORD` | yes | `change-me` | Grafana admin password. |
| `GF_SERVER_ROOT_URL` | yes | `https://your-domain.com/monitoring` | External URL Grafana uses for redirects and links. |

### Platform API

| Variable | Required | Default in `.env.example` | Purpose |
| --- | --- | --- | --- |
| `PLATFORM_API_TOKEN` | no | empty | Used for API authentication. |
| `PLATFORM_API_ALLOW_PRIVATE_TARGETS` | no | `false` | Controls whether the API accepts private/local targets. |

### Public Domain and CORS

| Variable | Required | Default in `.env.example` | Purpose |
| --- | --- | --- | --- |
| `STAGEFLOW_PUBLIC_DOMAIN` | yes | `your-domain.com` | Public domain used in generated URLs and edge config. |
| `PLATFORM_API_CORS_ALLOW_ORIGINS` | yes | `https://your-domain.com,https://www.your-domain.com` | Browser origin allowlist for API requests. |

### Frontend

| Variable | Required | Default in `.env.example` | Purpose |
| --- | --- | --- | --- |
| `VITE_API_URL` | yes | `https://your-domain.com` | Frontend API base URL. |
| `VITE_SITE_TITLE` | no | `StageFlow` | Site title shown in UI metadata. |
| `VITE_SITE_URL` | yes | `https://your-domain.com` | Canonical site URL used for metadata/share cards. |
| `VITE_GITHUB_URL` | no | `https://github.com/mattboback/stageflow` | Repository link shown in UI. |
| `VITE_TAGLINE` | no | `Podman-native web accessibility scanning platform` | Marketing tagline in UI surfaces. |
| `VITE_AI_NAVIGATOR_DEFAULT_MODEL` | no | `openai/gpt-4o-mini` | Default model shown for AI navigator flows. |

### AI Navigator (Optional)

| Variable | Required | Default in `.env.example` | Purpose |
| --- | --- | --- | --- |
| `OPENROUTER_API_KEY` | only if AI navigator enabled | empty | API key for OpenRouter model calls. |
| `OPENROUTER_APP_TITLE` | no | `StageFlow` | OpenRouter request attribution title. |
| `OPENROUTER_APP_REFERER` | no | `https://your-domain.com` | OpenRouter request attribution referer. |
| `AI_NAVIGATOR_DEFAULT_MODEL` | no | `openai/gpt-4o-mini` | Default backend model when scanner options do not override. |

### Caddy (Edge)

| Variable | Required | Default in `.env.example` | Purpose |
| --- | --- | --- | --- |
| `CADDY_EMAIL` | no | `admin@your-domain.com` | Email address used for Let's Encrypt TLS registration. |

### Advanced / Internal Overrides

These variables are primarily used internally or for advanced infrastructure setups. They typically do not need modification by standard operators.

| Variable | Purpose |
| --- | --- |
| `MINIO_ENDPOINT` | Internal host/port for MinIO API. |
| `MINIO_USE_SSL` | Whether internal MinIO connections use SSL. |
| `MINIO_USE_PROXY_URLS` | Whether to use proxy URLs for MinIO presigned URLs. |
| `MINIO_REGION` | MinIO region configuration. |
| `MINIO_ARTIFACT_BUCKET` | Name of the bucket used for storing artifacts. |
| `GF_AUTH_ANONYMOUS_ENABLED` | Enables anonymous access to Grafana dashboards. |
| `GF_SERVER_HTTP_PORT` | Internal HTTP port for Grafana. |
| `GF_SERVER_SERVE_FROM_SUB_PATH` | Whether Grafana serves from a sub-path. |

## Environment Guidance

- Use distinct credentials for local, staging, and production.
- Keep domains and CORS origins environment-specific.
- If you already run a shared Caddy on the host, route StageFlow through that existing process and avoid binding a second edge proxy.

## Pre-Deploy Validation

1. `just build` passes.
2. `just ci` passes.
3. One URL scan and one ZIP scan complete successfully.
4. SSE stream updates and final report retrieval both work.

## Related Docs

- [README](../../README.md)
- [Architecture](../architecture/system.md)
- [Security](../../.github/SECURITY.md)
