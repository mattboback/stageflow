# AGENTS.md

## Purpose

This file defines how agents should work in the StageFlow repository.
Follow these rules by default unless a user instruction explicitly overrides them.

## Prime Directives

These rules are non-negotiable. Violating them is a critical failure.

| Rule | Rationale |
| --- | --- |
| **Never bypass lint/test failures** | Fix the issue. Never disable rules, skip tests, or hide errors. |
| **No breadcrumbs** | No `TODO` placeholders, ad-hoc status docs, or orphaned notes unless explicitly requested. |
| **Validate before claiming success** | Run relevant checks and report the real outcome. |

## Execution Defaults (StageFlow)

- Use `just` from the repo root for standard workflows.
- Prefer the smallest command that validates your change, then run broader checks when risk is high.
- Preferred verification ladder:
  1. Narrow checks for touched area (example: `bun run lint`, `bun run test` in the affected package)
  2. Broader package/service checks
  3. `just ci` for full-repo confidence when changes are cross-cutting

### Common Commands

- Setup: `just setup`
- Local stack: `just dev up`, `just dev down`, `just dev logs`, `just dev init`
- Build: `just build`, `just images`
- Quality gate: `just ci`
- Production (systemd user + Quadlets): `just prod up|down|restart|logs|health`

## Concurrency Model

Assume other agents or humans may commit during your session.

- Do not delete or revert changes just because you did not make them.
- If the working tree changes unexpectedly, continue focusing on files relevant to your task.
- Ignore unrelated diffs. Coordinate only when there is a real overlap or conflict.

## Defensive Programming

Apply this sequence:

`Fail fast -> Guard clauses -> Validate at boundaries -> Make illegal states unrepresentable`

Validate all trust boundaries:

- Incoming request payloads
- Environment variables and config
- Queue messages and event envelopes
- Database rows and persistence mappings
- External API responses
- Filesystem/archive inputs (size, path, format, limits)

## Language-Specific Rules

### TypeScript

| Constraint | Preferred Alternative |
| --- | --- |
| No `any` | `unknown` with narrowing |
| Avoid `as` casts | Runtime checks or schema validation |

- Prefer Bun-native APIs/runtime features when running on Bun.
- Assume modern browsers unless requirements say otherwise.

### Python

- Use `uv` with `pyproject.toml`.
- Do not introduce Poetry, pipenv, or `requirements.txt` workflows.
- Add type hints on all signatures.
- Prefer Pydantic models or dataclasses over raw untyped dicts.

### Go

- Always handle errors. Never ignore `err`.
- Pass `context.Context` through call chains.
- Wrap errors with context: `fmt.Errorf("doing thing: %w", err)`.

### Svelte 5

Svelte changes quickly. Verify current docs before implementing new patterns.

- Use runes: `$state`, `$derived`, `$effect`
- Do not introduce stores in new code where runes are appropriate.

## Anti-Patterns

| Avoid | Do Instead |
| --- | --- |
| Disabling lint rules to pass checks | Fix the underlying issue |
| Assuming docs are current | Verify against current code and tooling |
| Ignoring failing tests | Debug and resolve root cause |
| Broad refactors during targeted fixes | Keep changes scoped and intentional |

## Deployment (VPS)

StageFlow supports production deployments via systemd user services + Podman Quadlets.

### Quick Commands (Repo Root)

- Install units: `just prod install`
- Start: `just prod up`
- Stop: `just prod down`
- Restart: `just prod restart`
- Logs: `just prod logs`
- Health: `just prod health`

### Reverse Proxy Note

If you already operate a shared reverse proxy (Caddy/Nginx/Traefik), avoid deploying an additional StageFlow-managed proxy that binds `80/443`. Route traffic from your existing gateway to StageFlow services on loopback or the Podman network.
