# Contributing to StageFlow

Thanks for improving StageFlow. This guide covers local dev, quality checks, and PR readiness.

## Ground Rules

Source of truth for coding conventions: [AGENTS.md](AGENTS.md).

Highlights:

- Use `just` commands from repository root for normal workflows.
- Keep changes focused; avoid drive-by refactors in bugfix PRs.
- Never suppress type or lint failures to make CI pass.
- Update docs when behavior, public interfaces, or operations change.

## Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [Bun](https://bun.sh/)
- [Podman](https://podman.io/) (with `podman compose`)
- [just](https://github.com/casey/just)
- [golangci-lint v2](https://golangci-lint.run/)

## Getting Started

```bash
git clone https://github.com/mattboback/stageflow.git
cd stageflow
cp .env.example .env
just setup
```

Start the local stack:

```bash
just dev up
just dev init
just images
```

## Development Workflows

### Service Workflows

- `just dev [up|down|restart|logs|init]`
- `just staging [up|down|restart|logs|init|ps]`
- `just run frontend`
- `just run frontend preview`
- `just run api`
- `just run orchestrator`

### Build and Quality Workflows

- `just build`
- `just images`
- `just ci`

`just ci` includes:

- Go build/lint/test (`-race`) across workspace modules.
- Frontend CI checks.
- Scanner-runner CI checks.
- Frontend and scanner-runner `bun audit --audit-level=high`.

## Coding Standards

### Go

- Always handle errors.
- Pass `context.Context` through call chains.
- Wrap errors with context (`fmt.Errorf("...: %w", err)`).

### TypeScript

- Prefer `unknown` + narrowing over `any`.
- Avoid unsafe casts; validate at boundaries.
- Keep scanner-runner code Bun-native where practical.

### Svelte 5

- Use runes (`$state`, `$derived`, `$effect`) in new code.
- Prefer factory stores (`.svelte.ts`) for cross-component lifecycle and async state.

### General

- Fail fast with guard clauses.
- Make illegal states unrepresentable where possible.
- Keep public contracts schema-first where applicable.

## Testing Guidance

Run `just ci` before opening a PR.

When iterating locally, targeted commands are useful:

```bash
# Go module example
go -C platform/api test -race ./...

# Frontend
bun --cwd frontend run test
bun --cwd frontend run test:coverage

# Scanner-runner
bun --cwd platform/scanner-runner run test
bun --cwd platform/scanner-runner run test:coverage
```

## Pull Request Checklist

Before submitting:

1. Keep PR scope to one coherent feature/fix.
2. Add or update tests for behavior changes.
3. Run `just ci` and ensure it passes.
4. Update documentation and examples impacted by your changes.
5. Include a clear PR description with the reason for change.

## Writing Scanner Plugins

To add a scanner plugin:

1. Implement scanner module logic compatible with scanner-runner lifecycle.
2. Create a valid `manifest.json` under plugin directory.
3. Ensure manifest schema compliance.
4. Load plugin through supported discovery paths.

References:

- `packages/contracts/scanner-manifest/schema/scanner-manifest.schema.json`
- `packages/contracts/scanner-manifest/schema/README.md`
- [ARCHITECTURE.md](ARCHITECTURE.md#scanner-plugin-system)

## Repository Map

| Directory | Purpose |
| --- | --- |
| `platform/api` | REST API and SSE endpoint surface |
| `platform/orchestrator` | Job FSM, container lifecycle, aggregation |
| `platform/extractor` | ZIP validation and extraction |
| `platform/scanner-runner` | Scanner runtime and plugin loader |
| `frontend` | SvelteKit UI |
| `packages/contracts` | JSON Schemas and generated contracts |
| `packages/shared-go` | Shared Go packages and events |
| `tools` | Operational and suite utilities |

## Need Help?

- For bugs/features/questions: open a GitHub issue.
- For security reports: follow [SECURITY.md](SECURITY.md).
