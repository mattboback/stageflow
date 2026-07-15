# Self-Hosting

StageFlow uses rootless Podman for its long-running services and isolated per-job scanner pods. Docker is not supported because the Podman runtime and socket model are part of the isolation design.

## Requirements

- Go 1.26.5
- Node.js 22
- Bun 1.3.8
- Podman with Compose support
- `just`

Copy the local environment template before starting a stack:

```bash
cp .env.example .env
```

## Local Demo

Use the guided path for a first run:

```bash
just diagnose
just demo
```

This builds local images, starts the compose project, initializes MinIO, and waits for readiness.

| Surface | URL                     |
| ------- | ----------------------- |
| Web     | `http://localhost:3000` |
| API     | `http://localhost:8080` |
| Grafana | `http://localhost:3001` |

The default local identity is compose project `stageflow_dev` on network `stageflow_dev_net`. Override `COMPOSE_PROJECT_NAME` or `STAGEFLOW_NETWORK_NAME` only to isolate an additional stack.

## Private-Target Development

Use the `local` overlay when scanner pods must reach `localhost`, loopback, or private-network targets:

```bash
just setup
just images
just dev up local
just dev init local
```

This serves the web app on `http://localhost:3020`, enables private-target intake, and uses host networking for job pods. Host networking behaves as intended on Linux; on macOS or Windows it reaches the Podman VM rather than the workstation.

## Manual Lifecycle

Use `just dev` recipes when you need individual steps:

```bash
just setup
just images
just dev up dev
just dev init dev
just dev logs
```

After images and `.env` exist, the CLI can manage the same compose stack:

```bash
stageflow stack up
stageflow stack status
stageflow stack down
```

`stageflow stack up --env local` selects the private-target overlay. The command must run from a StageFlow checkout and does not replace initial setup or image builds.

## Public Self-Hosting

The checked-in files provide an application topology and an optional edge example:

- `infra/compose/podman-compose.yml` defines the base services.
- `infra/compose/podman-compose.local.yml` is development-only private-target configuration.
- `infra/caddy/Caddyfile` is a starting point for host-level TLS and routing.
- `infra/security/egress-policy.example.md` describes scanner egress controls.

The Caddy example expects services exposed on its documented loopback ports. It is not the same layout as the local `3000`/`3020` compose overlays; choose one topology and align its public URLs, ports, and proxy configuration.

Before exposing StageFlow:

1. Replace every `change-me` value in `.env`.
2. Set explicit CORS origins and keep API authentication enabled.
3. Keep private-target scanning disabled unless it is an intentional trust boundary.
4. Keep the host Podman socket rootless and `no-new-privileges` enabled.
5. Apply host or container egress policy to reduce DNS-rebinding and metadata-service risk.
6. Run `just ci`, then complete one URL scan and one ZIP scan.
7. Verify SSE updates and final report retrieval through the public edge.

See the [configuration reference](reference/configuration.md) for environment variables and [architecture](architecture.md#security-model) for the threat model.

Staging uploads and completed artifacts default to one-day object-store
lifecycles. Adjust `MINIO_STAGING_RETENTION_DAYS` and
`MINIO_ARTIFACT_RETENTION_DAYS` deliberately for your environment and reflect
the chosen values in your own data-handling notice. The hosted demo's policy is
documented separately in [Privacy](privacy.md). Reports promoted as project
baselines are copied into a private lifecycle-exempt bucket and persist until
they are replaced or their project is deleted.

For upgrades with existing promoted baselines, prepare storage before applying
short artifact lifecycles:

```bash
MINIO_APPLY_LIFECYCLES=false ./infra/minio/init-buckets.sh
# Start the updated Platform API and verify its baseline reconciliation log.
./infra/minio/init-buckets.sh
```

If the reconciliation summary lists a project under `missing_legacy_projects`,
its old artifact expired before it could be copied. The service stays
available, but diff requests for that project return HTTP 409 until you promote
a new completed job as its baseline.

## Hosted Demo Boundary

`stageflow.org` runs the application code in this repository, but its gateway, Quadlet units, release process, monitoring, and rollback automation live in a separate deployment workspace. The hosted service is not evidence that the repo-local compose and Caddy examples reproduce its host topology.
