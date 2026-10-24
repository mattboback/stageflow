# Self-hosting

[Overview](../README.md) · [Architecture](architecture.md) · [Security](../SECURITY.md)

StageFlow runs on rootless Podman. Docker is not supported because the Podman runtime and socket model are part of the job-isolation design.

## Requirements

- Go 1.26.10 or later
- Node.js 24.x
- Bun 1.3.8
- Rootless Podman with Compose support and a running user Podman socket
- `just`
- Git, Bash and `curl`

On Linux with systemd, start the socket with `systemctl --user start podman.socket`. From the repository root, copy the local defaults, check the runtime and configuration, then start the stack:

```bash
cp .env.example .env
just diagnose
just demo
```

`just demo` builds the local images, starts the Compose project with its shared artifact volume, and waits for readiness.

| Surface | URL                     |
| ------- | ----------------------- |
| Web     | `http://localhost:3000` |
| API     | `http://localhost:8080` |

Use `just dev up`, `just dev logs`, `just dev restart` and `just dev down` for the individual lifecycle steps. `.env.example` contains the local defaults; `infra/compose/podman-compose.yml` defines the stack wiring. For another isolated stack, choose a unique `COMPOSE_PROJECT_NAME` and non-conflicting `FRONTEND_PORT`, `PLATFORM_API_PORT` and `ORCHESTRATOR_PORT`.

## Private-target development

Set these values in `.env` when scanner pods must reach localhost or a private-network target:

```dotenv
PLATFORM_API_ALLOW_PRIVATE_TARGETS=true
POD_NETNS_MODE=host
POD_NETWORK=
```

Then restart the stack with `just dev restart`, set `STAGEFLOW_API_KEY` to the same value as `PLATFORM_API_TOKEN`, and pass `--allow-private-targets`:

```bash
STAGEFLOW_API_KEY='your-local-api-token' stageflow scan http://127.0.0.1:5173 \
  --api http://localhost:8080 --allow-private-targets
```

Host mode reaches the workstation on Linux; on macOS and Windows it reaches the Podman VM.

## Public deployment checklist

Before exposing StageFlow:

1. Replace every `change-me` value in `.env`.
2. Set explicit CORS origins and `VITE_API_URL`/`VITE_SITE_URL`; rebuild the frontend after changing its build-time URLs.
3. Put a TLS reverse proxy in front: `/api/*` to platform-api and everything else to the frontend. Set `PLATFORM_API_TRUSTED_PROXIES` to the proxy's source address. The proxy should inject no credentials.
4. Keep private-target scanning disabled unless it is an intentional trust boundary, and keep the Podman socket rootless with `no-new-privileges`.
5. Apply host egress policy to reduce DNS-rebinding and metadata-service risk. The isolated job network blocks other Podman networks, not ports bound on the host.
6. Run `just ci` ([contributor prerequisites](../CONTRIBUTING.md#full-quality-gate)), then complete one URL scan and one ZIP scan through the public edge, including SSE progress and final report retrieval.

Self-hosted deployments accept up to 100 URLs per scan and enable ZIP uploads by default; `PLATFORM_API_MAX_PAGES` and `PLATFORM_API_ZIP_UPLOAD_ENABLED` control intake. The hosted service accepts up to 25 public URLs and does not accept ZIP uploads. Terminal job rows and artifact directories are removed 24 hours after completion.
