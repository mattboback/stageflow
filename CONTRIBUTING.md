# Contributing to StageFlow

Thank you for your interest in contributing to StageFlow! This guide will help you get set up and productive.

## Development Setup

### Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [Bun](https://bun.sh/)
- [Podman](https://podman.io/) (with `podman compose`)
- [just](https://github.com/casey/just)
- [golangci-lint v2](https://golangci-lint.run/)

### Getting Started

```bash
git clone https://github.com/mattboback/stageflow.git
cd stageflow
cp .env.example .env
# Edit .env with local credentials
just setup
```

This installs Go and Bun dependencies and creates the Podman network.

### Running Locally

```bash
# Start infrastructure (NATS, MinIO, Grafana)
just dev up
just dev init    # one-time: create MinIO buckets

# Run services individually for development
just run frontend    # SvelteKit dev server
just run api         # Platform API
just run orchestrator
```

## Code Style

StageFlow follows the conventions in [AGENTS.md](AGENTS.md). Key rules:

### Go
- Always handle errors — never `_ = err`
- Pass `ctx` through call chains
- Wrap errors: `fmt.Errorf("doing thing: %w", err)`

### TypeScript
- No `any` — use `unknown` + narrowing
- Avoid `as` casts — prefer runtime checks
- Prefer Bun APIs when running on Bun

### Svelte 5
- Use runes: `$state`, `$derived`, `$effect`
- No stores in new code — runes replace them

### General
- Fail fast with guard clauses
- Validate at trust boundaries
- Never disable lint rules to pass — fix the underlying issue

## Testing

Run the full CI suite locally before submitting a PR:

```bash
just ci
```

This runs:
- Go build, lint (`golangci-lint`), and test (`-race`) across all workspace modules
- Frontend lint, type-check, and test with coverage

### Running Tests Individually

```bash
# Go tests for a specific module
cd platform/api && go test -race ./...

# Frontend tests
cd frontend && bun run test

# Frontend with coverage
cd frontend && bun run test:coverage
```

## Pull Request Process

1. Fork the repo and create a feature branch from `main`
2. Make your changes with clear, focused commits
3. Ensure `just ci` passes locally
4. Open a PR against `main` with a clear description of what and why
5. Address any review feedback

### PR Guidelines

- Keep PRs focused — one feature or fix per PR
- Include tests for new functionality
- Update documentation if your change affects the public API or architecture

## Writing Scanner Plugins

StageFlow's scanner system is designed for extensibility. To create a new scanner:

1. Create a directory under `platform/scanner-runner/src/scanners/your-scanner/`
2. Add a `manifest.json` following the schema in `packages/contracts/scanner-manifest/`
3. Implement the scanner class extending `ScannerBase`
4. Export it from `index.ts`

See [ARCHITECTURE.md § Scanner Plugin System](ARCHITECTURE.md#scanner-plugin-system) for the full manifest schema, lifecycle hooks, and configuration options.

## Project Layout

| Directory | Language | Purpose |
|-----------|----------|---------|
| `platform/api` | Go | REST API + SSE |
| `platform/orchestrator` | Go | Job coordination, container management |
| `platform/extractor` | Go | ZIP extraction service |
| `platform/scanner-runner` | TypeScript | Scanner runtime with Playwright |
| `frontend` | Svelte/TS | SPA frontend |
| `packages/shared-go` | Go | Shared libraries |
| `packages/contracts` | JSON Schema | Shared type contracts |
| `tools/` | Go | CLI utilities |

## Questions?

Open an issue for bugs, feature requests, or questions about the codebase.
