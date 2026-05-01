# Contributing

## Development Setup

1. Copy `.env.example` to `.env`.
2. Run `just setup`.
3. Run `just images`.
4. Run `just dev up` and `just dev init`.
5. Run `pre-commit install --hook-type pre-commit --hook-type commit-msg` if you want the repo hooks locally.

## Quality Gates

Run checks directly from each workspace before opening a PR.

- `go build ./...`, `go test -race ./...`, `golangci-lint run`, `govulncheck ./...` for Go modules
- `bun run ci` in `clients/web`
- `bun run ci` in `services/scanner-runner`
- `bash devtools/scripts/tests/cli-install.test.sh`
- `just project-golden` when project-mode baseline, diff, CLI exit-code, or report normalization behavior changes
- `just dead-code` when removing or moving TypeScript exports/components

## Getting Help

- Use the bug report or question issue templates for public support requests.
- Use [SECURITY.md](SECURITY.md) for vulnerabilities or anything sensitive.

## Pull Requests

- Keep PRs focused and small.
- Include test evidence for changed areas.
- Update docs/contracts when behavior changes.

Before opening a PR, check:

- Changed behavior is described in the PR body.
- Tests run are listed with pass/fail results.
- Contract schemas and generated Go/TypeScript types are regenerated when `libs/contracts` changes.
- CLI reference docs are regenerated with `go run ./clients/cli docs --out-dir docs/reference/cli/stageflow` when CLI commands or flags change.
- Screenshots or Storybook evidence are included for UI changes.
- Security-sensitive paths are called out, especially URL intake, ZIP extraction, browser/network behavior, auth, CORS, tokens, and deployment config.
- `qa/e2e/project-scan-golden.sh` was run or explicitly judged not relevant.

Useful orientation links:

- [README.md](README.md)
- [ARCHITECTURE.md](ARCHITECTURE.md)
- [docs/architecture/system.md](docs/architecture/system.md)
- [docs/operations/deployment.md](docs/operations/deployment.md)
- [services/platform-api/README.md](services/platform-api/README.md)
- [services/scanner-runner/README.md](services/scanner-runner/README.md)
