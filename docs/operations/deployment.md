# Deployment Notes

StageFlow keeps repo-managed environments separate from the hosted `stageflow.org` demo.

## Local Demo

Use this mode for the fastest runnable stack on a development machine:

```bash
cp .env.example .env
just diagnose
just demo
```

This uses the compose files under `infra/compose/`, builds local images, starts the stack on loopback ports, and initializes MinIO buckets.

## Repo-Managed Staging and Self-Hosting

Use the checked-in compose and Caddy examples for deployments you manage from this repository:

- `infra/compose/podman-compose.yml` is the base service topology.
- `infra/compose/podman-compose.local.yml` enables local/private target scanning for development.
- `infra/compose/podman-compose.staging.yml` provides a domain-like staging overlay on alternate loopback ports.
- `infra/caddy/Caddyfile` is an optional host-level edge example for public-domain routing and TLS.

Before exposing a public domain, replace every `change-me` value in `.env.example`, set explicit CORS origins, and review the scanner egress guidance in `infra/security/egress-policy.example.md`.

The compose orchestrator mounts the rootless Podman socket so it can create scanner job pods. In compose deployments it runs as UID/GID `0:0` inside the container to access that socket; keep the host Podman socket rootless and keep `no-new-privileges` enabled.

## Hosted Demo

The hosted `stageflow.org` demo runs the same application code, but its production release, verification, monitoring, rollback, and host-level deployment control plane are intentionally managed outside this public repository.

The repo `just deploy` recipe is a guarded delegate for that control plane:

- On the VPS, it uses `${STAGEFLOW_DEPLOY_CONTROL_PLANE:-/home/matt/Deployment}/justfile` and runs `just deploy stageflow`.
- Outside that host, it fails with a neutral message and leaves local, staging, and self-hosted deployment to the checked-in compose/Caddy files.
- Set `STAGEFLOW_DEPLOY_CONTROL_PLANE=/path/to/control-plane` only on hosts that intentionally provide the external deployment justfile.
