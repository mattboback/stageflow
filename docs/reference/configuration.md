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

`just demo` is the fastest end-to-end smoke test. For the individual bootstrap steps behind it, run `just setup && just images && just dev up && just dev init`.

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

| Variable                             | Required | Default in `.env.example` | Purpose                                                 |
| ------------------------------------ | -------- | ------------------------- | ------------------------------------------------------- |
| `PLATFORM_API_TOKEN`                 | no       | empty                     | Used for API authentication.                            |
| `PLATFORM_API_ALLOW_PRIVATE_TARGETS` | no       | `false`                   | Controls whether the API accepts private/local targets. |

### Public Domain and CORS

| Variable                          | Required | Default in `.env.example`                                                                 | Purpose                                               |
| --------------------------------- | -------- | ----------------------------------------------------------------------------------------- | ----------------------------------------------------- |
| `STAGEFLOW_PUBLIC_DOMAIN`         | yes      | `localhost`                                                                               | Public domain used in generated URLs and edge config. |
| `PLATFORM_API_CORS_ALLOW_ORIGINS` | yes      | `http://localhost:3000,http://127.0.0.1:3000,http://localhost:3010,http://localhost:8080` | Browser origin allowlist for API requests.            |

### Web App

| Variable                          | Required | Default in `.env.example`                                       | Purpose                                           |
| --------------------------------- | -------- | --------------------------------------------------------------- | ------------------------------------------------- |
| `VITE_API_URL`                    | yes      | `http://localhost:8080`                                         | Web app API base URL.                             |
| `VITE_SITE_TITLE`                 | no       | `StageFlow`                                                     | Site title shown in UI metadata.                  |
| `VITE_SITE_URL`                   | yes      | `http://localhost:3000`                                         | Canonical site URL used for metadata/share cards. |
| `VITE_GITHUB_URL`                 | no       | `https://github.com/mattboback/stageflow`                       | GitHub link shown in UI.                          |
| `VITE_TAGLINE`                    | no       | `Podman-native web accessibility and quality scanning platform` | Marketing tagline in UI surfaces.                 |
| `VITE_AI_NAVIGATOR_DEFAULT_MODEL` | no       | `openai/gpt-4o-mini`                                            | Default model shown for AI navigator flows.       |

### AI Navigator (Optional)

| Variable                     | Required                     | Default in `.env.example` | Purpose                                                     |
| ---------------------------- | ---------------------------- | ------------------------- | ----------------------------------------------------------- |
| `OPENROUTER_API_KEY`         | only if AI navigator enabled | empty                     | API key for OpenRouter model calls.                         |
| `OPENROUTER_APP_TITLE`       | no                           | `StageFlow`               | OpenRouter request attribution title.                       |
| `OPENROUTER_APP_REFERER`     | no                           | `http://localhost:3000`   | OpenRouter request attribution referer.                     |
| `AI_NAVIGATOR_DEFAULT_MODEL` | no                           | `openai/gpt-4o-mini`      | Default backend model when scanner options do not override. |

### Caddy (Edge)

| Variable      | Required | Default in `.env.example` | Purpose                                                |
| ------------- | -------- | ------------------------- | ------------------------------------------------------ |
| `CADDY_EMAIL` | no       | not set                   | Email address used for Let's Encrypt TLS registration. |

### Advanced infrastructure overrides

Most first-time local setups can ignore this section. These variables are mainly for custom or advanced self-hosted infrastructure layouts.

| Variable                        | Purpose                                             |
| ------------------------------- | --------------------------------------------------- |
| `MINIO_ENDPOINT`                | Internal host/port for MinIO API.                   |
| `MINIO_USE_SSL`                 | Whether internal MinIO connections use SSL.         |
| `MINIO_USE_PROXY_URLS`          | Whether to use proxy URLs for MinIO presigned URLs. |
| `MINIO_REGION`                  | MinIO region configuration.                         |
| `MINIO_ARTIFACT_BUCKET`         | Name of the bucket used for storing artifacts.      |
| `GF_AUTH_ANONYMOUS_ENABLED`     | Enables anonymous access to Grafana dashboards.     |
| `GF_SERVER_HTTP_PORT`           | Internal HTTP port for Grafana.                     |
| `GF_SERVER_SERVE_FROM_SUB_PATH` | Whether Grafana serves from a sub-path.             |

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
