




# AGENTS.md

---

## Prime Directives

> These rules are non-negotiable. Violating them is a critical failure.

| Rule                                | Rationale                                                                                |
| ----------------------------------- | ---------------------------------------------------------------------------------------- |
| **Never bypass lint/test failures** | Fix the issue. Never disable rules, skip tests, or hide errors.                          |
| **No breadcrumbs**                  | No `# TODO: add X later`, no `summary.md`, no orphaned docs unless explicitly requested. |

---

## Concurrency Model

Assume other agents or humans may commit during your session.

- Don't delete or revert changes just because you didn't make them.

- If you see unexpected changes to the the working-tree don't panic. Your
  changes have just been committed

- Only worry about your task stop focusing on working tree changes that aren't
  related to your task. The other agents will stay out of your way if you stay
  out of their way

---

## Defensive Programming

```
Fail fast → Guard clauses → Validate at boundaries → Make illegal states unrepresentable
```

**Validate at trust boundaries:**

- Incoming request payloads
- Environment variables / config
- Queue messages
- Database rows
- External API responses

---

## Language-Specific

### TypeScript

| Constraint       | Alternative                            |
| ---------------- | -------------------------------------- |
| No `any`         | `unknown` + narrowing                  |
| Avoid `as` casts | Runtime checks; justify if unavoidable |

- Prefer Bun APIs when running on Bun.
- Assume modern browsers unless stated otherwise.

### Python

- **Package management:** `uv` + `pyproject.toml` only. No pip venvs, Poetry, or `requirements.txt`.
- Type hints on all signatures. Validate with `mypy` if in CI.
- Prefer Pydantic models or dataclasses over raw dicts.

### Go

- Always handle errors. Never `_ = err`.
- Pass `ctx` through call chains.
- Wrap errors: `fmt.Errorf("doing thing: %w", err)`

### Svelte 5

> ⚠️ Training data for Svelte 5 is limited. **Always fetch current docs** via MCP or web.

- Use runes: `$state`, `$derived`, `$effect`
- No stores in new code—runes replace them.

---

## Anti-Patterns

| Don't                      | Do Instead               |
| -------------------------- | ------------------------ |
| Disable lint rules to pass | Fix the underlying issue |
| Assume docs are current    | Verify against code      |
| Ignore test failures       | Investigate and fix      |

---

## Deployment (VPS)

StageFlow supports production deployments via **systemd --user + Quadlets** (Podman).

### Quick commands (from repo root)

- Install units: `just prod install`
- Start: `just prod up`
- Stop: `just prod down`
- Restart: `just prod restart`
- Logs: `just prod logs`
- Health: `just prod health`

### Reverse proxy note

If you already run a shared reverse proxy (Caddy/Nginx/Traefik), avoid deploying an additional
StageFlow-managed Caddy instance that binds `80/443`. Prefer routing to StageFlow services on
loopback (or within your Podman network) from your existing gateway.
