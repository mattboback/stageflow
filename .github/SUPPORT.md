# Support

StageFlow support currently happens through this repository. There is not a separate public help desk or discussion forum today, so the best route depends on the kind of problem you are hitting.

## Where to go

- **Reproducible bug** — open a GitHub issue with steps, logs, and the affected surface
- **Feature idea or product gap** — open a feature request issue
- **Docs problem or unclear setup step** — open an issue and point to the file or command that needs work
- **Security issue** — do **not** open a public issue; follow [SECURITY.md](SECURITY.md)

Before filing anything, search existing issues: <https://github.com/mattboback/stageflow/issues>

## Useful docs

Start with the source that matches your question:

- Docs landing page and path finder: `docs/README.md`
- Product overview and local quick start: `README.md`
- Architecture and service boundaries: `docs/architecture/system.md`
- Configuration and environment variables: `docs/reference/configuration.md`
- CLI and developer tooling: `docs/operations/devtools.md`
- Project mode: `docs/PROJECT_MODE.md`

## Before opening an issue

Capture the details that make a report actionable:

1. Confirm whether the problem is local-only, staging-only, or specific to the public demo.
2. Note the exact command, target URL or ZIP, scanner list, and commit or release you tested.
3. Save the relevant logs, screenshots, or terminal output.
4. If the issue is local, rerun with the checklist below so the report includes what you already tried.

## Local troubleshooting checklist

### 1. Confirm the right stack is running

For normal local development:

```bash
just dev up
just dev init
```

For localhost or private-target scans:

```bash
just dev up local
just dev init local
```

If you need to inspect running containers directly:

```bash
podman ps --format '{{.Names}}\t{{.Status}}'
```

### 2. Check service logs

```bash
just dev logs
```

Use the local overlay form when debugging private-target scans:

```bash
just dev logs local
```

### 3. Check health endpoints

```bash
curl -fsS http://localhost:8080/healthz
curl -fsS http://localhost:8081/healthz
```

- `http://localhost:8080/healthz` checks the Platform API
- `http://localhost:8081/healthz` checks the Orchestrator admin API

### 4. Verify local configuration and mode

- Confirm `.env` exists and is based on `.env.example`
- Ensure you are using the correct port for your environment mode (`:3000` in `dev`, `:3010` in `local`)
- Rebuild images after dependency or runtime changes:

```bash
just images
```

### 5. Reinstall the CLI if the wrong binary is on your `PATH`

```bash
just cli-install
stageflow version
```

### 6. Debug project mode separately

If `stageflow project` fails, validate the generated config and readiness flow first:

```bash
stageflow project doctor
stageflow project doctor --skip-dev
```

## What to include in an issue

The most useful reports include:

- the exact command or UI flow that failed
- the target being scanned (public URL, localhost target, ZIP upload, or project mode)
- the scanners involved
- the expected result versus the actual result
- job IDs, logs, screenshots, or terminal output when available

## Common problem patterns

### Private or localhost targets are rejected

Use the local overlay:

```bash
just dev up local
just dev init local
just images
```

This enables private-target scanning in the local stack and configures job pods so scanners can reach loopback targets.

### A service starts but scans still fail

Check logs first, then restart the stack:

```bash
just dev down
just dev up
just dev init
```

For the local overlay:

```bash
just dev down local
just dev up local
just dev init local
```

### Docs or command examples look wrong

Please open a docs issue and include:

- the page or file path
- the incorrect command or example
- the expected behavior

## Production support boundary

This repository documents the app and local workflows, but live production operations for `stageflow.org` are managed from an external deployment workspace. If you are reporting a public demo problem, describe the public symptom clearly, but avoid assuming access to production internals.
