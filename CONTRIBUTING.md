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

## Getting Help

- Use the bug report or question issue templates for public support requests.
- Use [SECURITY.md](SECURITY.md) for vulnerabilities or anything sensitive.

## Pull Requests

- Keep PRs focused and small.
- Include test evidence for changed areas.
- Update docs/contracts when behavior changes.
