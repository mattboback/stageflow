# AGENTS.md

## Purpose

This repo contains StageFlow (`stageflow.org`), a Podman-native web accessibility and quality scanning platform.

## Production deployment

Production for this site is **not** managed from this repo. It is managed
from a separate root deployment workspace. Set `STAGEFLOW_PROD_DEPLOY_DIR` to
the path of that workspace before running any production commands.

Read `$STAGEFLOW_PROD_DEPLOY_DIR/DEPLOYMENT_STRATEGY.md` before making any
deployment, routing, or topology changes. The root
`$STAGEFLOW_PROD_DEPLOY_DIR/justfile` is the canonical operator interface.

### How to deploy

```bash
cd $STAGEFLOW_PROD_DEPLOY_DIR
just deploy stageflow
```

This command:

1. Compiles Go/Node artifacts and triggers building all required container images.
2. Installs quadlet unit files from `$STAGEFLOW_PROD_DEPLOY_DIR/stageflow-quadlets/templates/` into `~/.config/containers/systemd/`.
3. Restarts the systemd user service target (`stageflow.target`).

Quadlet templates live in the deployment workspace, not in this app repo.

**Important:** `systemctl --user start` is a no-op if the service is
already running. If you rebuild the image and need the container to pick
it up, restart the app service explicitly via the target:

```bash
systemctl --user restart stageflow.target
```

### Other operator commands

```bash
cd $STAGEFLOW_PROD_DEPLOY_DIR

just status                 # Show running state of all services
just health                 # Health check all services
just domain-check           # Public surface curl checks
just logs stageflow         # Tail container logs
just restart stageflow      # Restart service
just stop stageflow         # Stop service
```

### Topology

| Item | Value |
| --- | --- |
| Public hostname | `stageflow.org` |
| Pod name | `stageflow` |
| Quadlets | `$STAGEFLOW_PROD_DEPLOY_DIR/stageflow-quadlets/templates/` |
| Network | Shared `ingress.network` |
| Gateway | Shared Caddy at `$STAGEFLOW_PROD_DEPLOY_DIR/gateway/Caddyfile` |
| Public API routes | `/api/v1/*` passthrough, `/api/healthz` rewrite to internal `/healthz` |

### Runtime stack

- **App:** StageFlow runs multiple services (NATS, MinIO, Postgres, Orchestrator, Platform API, Frontend, Grafana) inside a single Podman Pod.
- **Ingress:** Shared Caddy gateway terminates TLS and routes `stageflow.org` traffic directly to the Pod via `ingress.network` using aliases (`stageflow-web`, `stageflow-api`, `stageflow-artifacts`). The app no longer binds to loopback ports on the host.

## Anti-drift rules

These rules match the root deployment workspace. Treat them as hard
constraints.

- Do not create standalone production deploy scripts, Makefiles, or
  justfile recipes in this repo that target the VPS.
- Do not add Caddy, nginx, or any reverse proxy config to this repo that
  binds public ports.
- Do not bind the app to public interfaces. The app listens on the
  container's internal port and Caddy handles public ingress.
- Route all production deployment work through
  `$STAGEFLOW_PROD_DEPLOY_DIR/justfile`.

## Local development

See `README.md` and `justfile` for full details.

```bash
just setup
just dev up
```

## Conflict resolution

If deployment instructions in this repo conflict with the root deployment
workspace:

1. `$STAGEFLOW_PROD_DEPLOY_DIR/DEPLOYMENT_STRATEGY.md` wins.
2. `$STAGEFLOW_PROD_DEPLOY_DIR/justfile` is the canonical operator interface.
3. This file is subordinate and must be updated to match.
