# Contributing

[Overview](README.md) · [Architecture](docs/architecture.md) · [Self-hosting](docs/self-hosting.md) · [Security](SECURITY.md)

Start with the component that owns the behavior you are changing. Keep fixes focused, preserve public CLI/API contracts and run the existing checks for the affected area. The [architecture guide](docs/architecture.md) maps the runtime and shared contracts.

## Local setup

The repository uses Go 1.26.10 or later, Node.js 24.x, Bun 1.3.8 and `just`. Rootless Podman with Compose support is required for the local stack and live isolation checks. See the [self-hosting guide](docs/self-hosting.md) for socket and environment setup.

From the repository root, install the Go, web and scanner dependencies:

```bash
just deps
```

To work on the frontend:

```bash
cd clients/web
bun run dev
```

Open `http://localhost:3020`. The committed `/demo` works without a backend; submitting scans and watching live progress require a [running local stack](docs/self-hosting.md). The development client uses `http://localhost:8080` for the API by default. For another API address, set `VITE_API_URL` and allow the frontend origin in `PLATFORM_API_CORS_ALLOW_ORIGINS`.

Keep `.env`, credentials, browser state, scan output and local audit notes out of commits.

## Focused checks

Run these from the repository root. The shell subshells keep subsequent commands in the correct directory.

| Area                                                     | Existing check                                                     |
| -------------------------------------------------------- | ------------------------------------------------------------------ |
| Documentation                                            | `node devtools/scripts/check-markdown-links.mjs`                   |
| Go compilation                                           | `go build ./...`                                                   |
| Go unit and integration tests                            | `go test -race ./...`                                              |
| Archive extraction integration                           | `go test -race -tags integration ./services/archive-extractor/...` |
| Web formatting, lint, types, tests and build             | `(cd clients/web && bun run ci)`                                   |
| Scanner formatting, lint, types, runtime smoke and tests | `(cd services/scanner-runner && bun run ci)`                       |
| Production reachability                                  | `just dead-code`                                                   |
| Shell tooling                                            | `just shell-tests`                                                 |
| Browser end-to-end flows                                 | `just web-e2e`                                                     |
| Sanitized backup/restore canaries                        | `just backup-tests`                                                |

Go suites that use the embedded PostgreSQL harness need a non-root user and network access for the first server download, or a disposable test database supplied through `TEST_DATABASE_URL`. The backup checks separately need PostgreSQL server/client binaries on `PATH`. Browser checks need Chromium: install the web build with `(cd clients/web && bun x playwright install --with-deps chromium)` and the scanner build with `(cd services/scanner-runner && bun x playwright install chromium)`. Each package pins its own Playwright version.

## Full quality gate

The contributor gate is `just ci`. In addition to the toolchain above, install Python 3 with venv and pip, ShellCheck, jq, ripgrep, zip, tar, sha256sum, and PostgreSQL server/client binaries. Use a full-history checkout, a non-root user and an available port 4173.

```bash
just deps
(cd clients/web && bun x playwright install --with-deps chromium)
just ci
```

The gate checks Markdown links, contract generation and fixtures, workflow syntax, secrets, pre-commit hooks, Go build/lint/race tests and vulnerabilities, shell and backup regressions, web and scanner checks, production dead code, and browser end-to-end flows. Its secret scan reads the full Git history, so a shallow checkout cannot run the complete gate.

The live rootless Podman contracts and the full-stack scan-diff regression are separate and need a running stack:

```bash
just test-podman-live
just scan-diff-golden
```

## Generated files and fixtures

| Path                                                                                                                                   | Role and update process                                                                                               |
| -------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| `libs/contracts/` generated Go/TypeScript and `clients/web/public/schemas/`                                                            | Edit the source JSON Schemas, then run `just generate-contracts`. Review the generated diff.                          |
| `clients/web/public/demo/` and the generated `demo-summary.ts`, `demo-crop.ts`, `demo-terminal.json` under `clients/web/app/lib/demo/` | Committed sample-site reports and derived comparison data. Regenerate with `just demo-fixture` against a running API. |
| `qa/fixtures/` and component `testdata/` directories                                                                                   | Test inputs and expected outputs. Some are compared byte for byte; update only for an intentional behavior change.    |
| `assets/`                                                                                                                              | Screenshots and the diagram displayed in the documentation.                                                           |
| `node_modules/`, `build/`, `dist/`, `.react-router/`                                                                                   | Local dependencies or generated build output; excluded from Git.                                                      |
| `.local/`, `artifacts/`, `output/`                                                                                                     | Local investigation and command output; excluded from Git.                                                            |

The web app, scanner runtime and contract packages have independent install contexts. Preserve their lockfiles. An empty dead-code allowance file is still configuration consumed by a quality check. Generated files, dynamic routes, embedded assets and fixtures need producer and consumer checks before removal.

## Pull requests

Explain the problem, resulting behavior and validation performed. State any checks you could not run. Keep generated-output changes tied to their source changes, and include screenshots when a PR intentionally changes the interface. Report vulnerabilities through the [private security channel](SECURITY.md#reporting-a-vulnerability).
