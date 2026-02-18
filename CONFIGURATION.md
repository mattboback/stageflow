# StageFlow Configuration

Reference for environment configuration across StageFlow services.

This document covers variable groups and deployment intent. Use `.env.example` and service config sources as canonical defaults.

## Configuration Principles

- Keep secrets out of source control.
- Use separate env files per environment (`.env`, `.env.staging`, production secrets).
- Prefer explicit values for domains, CORS origins, and storage endpoints.
- Validate scanner/plugin settings before production rollout.

## Environment Groups

### Public and Domain Settings

- `STAGEFLOW_PUBLIC_DOMAIN`: public domain used by edge/proxy and URL generation paths.
- `VITE_API_URL`: frontend API base URL.
- `VITE_SITE_URL`: frontend site URL.

### API Service Settings

- API token/auth settings (optional access control for API requests).
- CORS allowlist settings, including production domains.
- Upload and request limits used at intake boundaries.
- SSE behavior and timeouts.

### Orchestrator Settings

- NATS connection and consumer behavior.
- PostgreSQL connection and schema location.
- MinIO bucket and artifact storage settings.
- Podman runtime and scanner container lifecycle controls.

### Extractor Settings

- MinIO staging/artifact access settings.
- ZIP extraction limits and validation controls.
- Workspace/results path conventions.

### Scanner Runner Settings

- Scanner selection/module identity.
- `SCANNER_OPTIONS` payload (validated against manifest schema).
- Plugin discovery paths (`/plugins`, `~/.stageflow/plugins`, `PLUGIN_PATHS`).
- Browser/runtime execution settings.

### Infrastructure Service Settings

- NATS service endpoint/credentials.
- MinIO endpoint/access credentials and SSL mode.
- PostgreSQL endpoint/credentials/database names.
- Grafana/admin credentials and provisioning settings.

## Required vs Optional Settings

Required in most environments:

- Service connectivity settings (NATS, MinIO, Postgres).
- Public URLs/domain config for frontend/API routing.
- CORS allow origins for browser clients.

Usually optional or environment-dependent:

- API auth token enforcement.
- Extra plugin search paths.
- Advanced scanner options and AI-specific scanner settings.

## Local Development Defaults

Recommended flow:

```bash
cp .env.example .env
just setup
just dev up
just dev init
just images
```

## Staging Overrides

- Start from `.env.staging.example` where available.
- Ensure staging uses distinct domains, buckets, and credentials.
- Verify `just staging` flows before promoting images.

## Production Guidance

- Inject secrets via your deployment secret manager.
- Restrict CORS origins to production frontend domains.
- Ensure edge rate limiting/WAF policy is configured.
- Rotate credentials and audit access regularly.

## Configuration Validation Checklist

Before deployment:

1. `just build` passes.
2. `just ci` passes.
3. Environment-specific stack boots cleanly.
4. One URL job and one ZIP job complete successfully.
5. SSE updates and final report retrieval both work.

## Related Docs

- [README.md](README.md)
- [ARCHITECTURE.md](ARCHITECTURE.md)
- [OPERATIONS.md](OPERATIONS.md)
- [SECURITY.md](SECURITY.md)
