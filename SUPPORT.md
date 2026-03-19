# Support

Thank you for using StageFlow. This document points you to the right docs, the right issue path, and the fastest local debugging checks.

## Start with the docs

Use the source that matches your question:

- Product overview and local quick start: `README.md`
- Architecture and service boundaries: `docs/architecture/system.md`
- Configuration and environment variables: `docs/reference/configuration.md`
- CLI and developer tooling: `docs/operations/devtools.md`
- Project mode: `docs/PROJECT_MODE.md`

## Before opening an issue

1. Search existing issues: <https://github.com/mattboback/stageflow/issues>
2. Confirm whether the problem is local-only, staging-only, or production-only.
3. Gather the exact command, target URL, and error output that reproduces the problem.

## Opening the right issue

Open a GitHub issue when you have:

- a reproducible bug
- a feature request for the platform, scanners, or CLI
- a docs gap that blocks setup or usage

For security reports, do not open a public issue. Follow `SECURITY.md` instead.

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

Use the local overlay form when you are debugging private-target scans:

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

### 4. Verify local configuration

- Confirm `.env` exists and is based on `.env.example`.
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

This repository documents the app and local workflows, but live production operations for `stageflow.org` are intentionally managed from the external deployment workspace. If your question is about live production operations, use the external control plane described in `AGENTS.md`.
