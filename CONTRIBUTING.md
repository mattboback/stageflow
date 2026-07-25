# Contributing

## Development Setup

Choose the smallest loop that matches the change:

- **Web UI only:** `cd clients/web && bun install --frozen-lockfile && bun run dev`; unit and mocked-browser tests do not require the service stack.
- **Scanner runtime only:** `cd services/scanner-runner && PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 bun install --frozen-lockfile`, then use `bun run ci` (install Chromium when running browser-backed suites).
- **One Go module:** run `just generate-contracts` once, `cd` into that module, then use `go test -race ./...` and `golangci-lint run`.
- **End-to-end or infrastructure:** use the full-stack steps below.

1. Copy `.env.example` to `.env`.
2. Run `just setup`.
3. Run `just images`.
4. Run `just dev up` and `just dev init`.
5. Run `pre-commit install --hook-type pre-commit --hook-type commit-msg` if you want the repo hooks locally.

Use `just demo` when you want the guided smoke test that performs the setup,
image build, MinIO initialization, and stack start in sequence. Use
`just dev up local` instead of the default `just dev up` when you need the
localhost/private-target overlay for CLI dev-loop scans.

Generated contract code is intentionally ignored. From a fresh checkout, run
`just setup` or `just generate-contracts` before running focused Go or
TypeScript commands directly inside a workspace.

Go modules pin the `1.26.5` toolchain in `go.mod` so local builds, CI, and
release binaries all compile with the same Go version; a newer Go on your
machine will download that toolchain automatically rather than drift from CI.

## Quality Gates

Pre-commit is a fast local guard for whitespace, secrets, generated contract
drift, internal documentation links, and other common mistakes. It is not the
full repo gate. Installing the hooks locally is optional, but the same config
runs in CI (`Workflow Lint` job), so a change that fails it fails your PR
whether or not you installed them.

### Formatting

Prettier is the only formatter, configured once at the repo root in `.prettierrc`
and enforced by `format:check` inside each Bun workspace's `ci` script — so it
runs from `just ci` and from CI without a separate step to keep in sync. Run
`bun run format` in `clients/web` or `services/scanner-runner` to fix findings.

The gate covers TypeScript. Two deliberate exclusions, both recorded in
`.prettierignore`:

- **Generated and byte-compared files.** `docs/reference/cli/` is byte-compared
  against `go run ./clients/cli docs` by `check-cli-docs-generated.sh`, and
  `qa/fixtures/` holds golden-regression inputs. Formatting either breaks CI.
- **Stylesheets.** `clients/web`'s CSS is written one rule per line for short
  declarations, consistently. Prettier cannot preserve that and would rewrite
  4,621 lines to expand it. Changing the convention is a decision for its own
  commit, not a side effect of adding a gate.

Run `just ci` before opening a PR when feasible. It runs the stale-vocabulary
check, Go build/lint/test/vuln checks, generated CLI-doc drift check, shell
regression tests, frontend CI/audit, and scanner-runner CI/audit. For focused
iteration, run the relevant workspace checks directly:

- `go build ./...`, `go test -race ./...`, `golangci-lint run`, `govulncheck ./...` for Go modules
- `bun run ci` in `clients/web`; include browser screenshots or scan/report evidence for UI changes when relevant
- `bun run ci` in `services/scanner-runner`
- `just shell-tests`, or run the scripts individually from `devtools/scripts/tests/`
- `just project-golden` when project-mode baseline, diff, CLI exit-code, or report normalization behavior changes
- `just dead-code` when removing or moving TypeScript exports/components in either Bun workspace
- `just coverage` when changing Go tests. It measures each `go.work` module with
  `-coverpkg=./...` and fails only when a module drops more than 1.5 points below
  `devtools/coverage-baseline.json`. If a drop is intentional, re-record it with
  `just coverage-update` and say why in the PR

Committed screenshots and reviewer-facing images belong in `docs/images`.
Ephemeral QA/build evidence should stay under ignored artifact, output, or
cache paths such as `artifacts/`, `output/`, and `.cache/`; `just clean` may
delete those local files.

### Orchestrator database tests

The `services/orchestrator` tests run against a real PostgreSQL. By default they
start an embedded PostgreSQL automatically — the binaries download once and are
cached under your user cache dir, so first run is slower and later runs are fast.
To run against an existing PostgreSQL instead (no download), set
`TEST_DATABASE_URL`, e.g. `TEST_DATABASE_URL=postgres://user:pass@localhost:5432/db?sslmode=disable go test ./...`.

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
- Contract schemas and generated Go/TypeScript types are regenerated locally when `libs/contracts` changes.
- CLI reference docs are regenerated with `go run ./clients/cli docs --out-dir docs/reference/cli/stageflow` when CLI commands or flags change.
- Screenshots or rendered scan/report evidence are included for UI changes.
- Security-sensitive paths are called out, especially URL intake, ZIP extraction, browser/network behavior, auth, CORS, tokens, and deployment config.
- `qa/e2e/project-scan-golden.sh` was run or explicitly judged not relevant.

Useful orientation links:

- [README.md](README.md)
- [docs/README.md](docs/README.md)
- [docs/architecture.md](docs/architecture.md)
- [docs/self-hosting.md](docs/self-hosting.md)
- [services/platform-api/README.md](services/platform-api/README.md)
- [services/scanner-runner/README.md](services/scanner-runner/README.md)
