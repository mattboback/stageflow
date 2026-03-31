# AGENTS.md

## Purpose

This repo contains StageFlow, a Podman-native web accessibility and quality scanning platform.

Production deployment is managed from a separate workspace and is not handled from this repo.

## Local development

See `README.md` and `justfile` for full details.

```bash
just setup
just dev up
just dev init
just images
```

## Key paths

| Area | Location |
| --- | --- |
| Web app | `clients/web` (SvelteKit) |
| CLI | `clients/cli` (Go) |
| Platform API | `services/platform-api` (Go) |
| Orchestrator | `services/orchestrator` (Go) |
| Scanner Runner | `services/scanner-runner` (TypeScript/Bun) |
| Shared Go libs | `libs/go/*` |
| Contracts/schemas | `libs/contracts` |
| Compose files | `infra/compose` |
| Architecture docs | `docs/architecture/system.md` |
