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
Local `just` recipes isolate the default compose project and Podman network as
`stageflow_dev` and `stageflow_dev_net`, so repeat smoke tests do not share DNS
aliases with unrelated StageFlow stacks on the same host. Override
`COMPOSE_PROJECT_NAME` or `STAGEFLOW_NETWORK_NAME` only when you intentionally
want a different local identity.

## Repo-Managed Self-Hosting

Use the checked-in compose and Caddy examples for deployments you manage from this repository:

- `infra/compose/podman-compose.yml` is the base service topology.
- `infra/compose/podman-compose.local.yml` enables local/private target scanning for development.
- `infra/caddy/Caddyfile` is an optional host-level edge example for public-domain routing and TLS.

Before exposing a public domain, replace every `change-me` value in `.env.example`, set explicit CORS origins, and review the scanner egress guidance in `infra/security/egress-policy.example.md`.

The compose orchestrator mounts the rootless Podman socket so it can create scanner job pods. In compose deployments it runs as UID/GID `0:0` inside the container to access that socket; keep the host Podman socket rootless and keep `no-new-privileges` enabled.

## Hosted Demo

The hosted `stageflow.org` demo runs the same application code, but its production release, verification, monitoring, rollback, and host-level deployment control plane are intentionally managed outside this public repository. Local and self-hosted deployments use the checked-in compose/Caddy files above.
