# StageFlow

[![CI](https://github.com/mattboback/stageflow/actions/workflows/ci.yml/badge.svg)](https://github.com/mattboback/stageflow/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

StageFlow audits a deployed website and diffs each release against the last good one, so you can see what broke, what was fixed, and how the score moved.

![The /demo comparison of two versions of a sample site: the score down 3, 3 new findings and 1 fixed, then the report's prioritized fixes](assets/report-review.png)

Five scanners check every page: axe for accessibility, Lighthouse for accessibility, best practices and SEO, plus StageFlow's own SEO, security-header and link checks. Each scan runs in isolated containers, and the report marks findings on screenshots of each page with concrete fix guidance.

Scan any public URL at [stageflow.org](https://stageflow.org) — no account needed. The [/demo](https://stageflow.org/demo) report shows the comparison above. Hosted scans accept up to 25 public URLs, and scan records and reports expire 24 hours after completion. [Projects](https://stageflow.org/projects) keep runs and baselines in your browser, with no account.

## Quick start

### CLI

Download the archive for your platform from [GitHub Releases](https://github.com/mattboback/stageflow/releases) and verify it against `checksums.txt`, or install with:

```bash
go install github.com/mattboback/stageflow/clients/cli/cmd/stageflow@latest
stageflow version
```

Scan a public URL, keep a reviewed report as a baseline, and diff later deployments:

```bash
stageflow scan https://preview.example.com --profile staging --format json --max-issues 0 > baseline.json

# After deploying a change:
stageflow scan https://preview.example.com --profile staging --format json --max-issues 0 > current.json
stageflow diff baseline.json current.json --fail-on-new=any \
  --format markdown >> "$GITHUB_STEP_SUMMARY"
```

The diff exits `1` when a release adds a finding. The step-summary output looks like this:

![The diff's markdown digest in a GitHub step summary: score 75 to 70, 3 new issues and 1 fixed, each new one with its page and selector](assets/step-summary.png)

### CI

The repository ships a composite action. Deploy a public preview first, then compare it with a complete, reviewed baseline:

```yaml
- uses: actions/checkout@v7
# Deploy a public preview using your hosting provider before this step.
- uses: mattboback/stageflow@v1.0.0
  with:
    baseline: qa/stageflow-baseline.json
    target: https://preview.example.com
    profile: staging
```

Set `STAGEFLOW_API_URL` for a self-hosted API. `STAGEFLOW_API_KEY` is optional for public hosted targets. The action fails on new findings and incomplete coverage.

### CLI commands

| Command | Use it for |
| --- | --- |
| `stageflow check --project project.json --output run-001` | Repeat a saved browser project and compare its baseline |
| `stageflow scan <url...>` | One-off public or private URL scans |
| `stageflow scan <dir-or-zip>` | Static build uploads to a self-hosted API |
| `stageflow diff baseline.json current.json` | Compare a saved baseline with a current scan |
| `stageflow report <job-id>` | Fetch a saved report |
| `stageflow auth capture` | Capture Playwright storage state for an authenticated scan |
| `stageflow completion <shell>` | Shell completion for bash, zsh, fish or PowerShell |

Use `stageflow --help` and `stageflow <command> --help` for exact flags. Exit codes are stable: `0` passes the selected gates, `1` means a finding or regression tripped a gate, and `2` means an error or incomplete scanner coverage. Reports with partial coverage always exit `2` unless the comparison explicitly allows partial coverage.

## Self-hosting

StageFlow runs on rootless Podman. Docker is not supported because the Podman runtime and socket model are part of the job-isolation design.

### Requirements

- Go 1.26.9 or later
- Node.js 24.x
- Bun 1.3.8
- Rootless Podman with Compose support and a running user Podman socket
- `just`
- Git, Bash and `curl`

On Linux with systemd, start the socket with `systemctl --user start podman.socket`. Run `just diagnose` to check the runtime, socket and local configuration, then:

```bash
cp .env.example .env
just demo
```

`just demo` builds the local images, starts the Compose project with its shared artifact volume, and waits for readiness.

| Surface | URL |
| --- | --- |
| Web | `http://localhost:3000` |
| API | `http://localhost:8080` |

Use `just dev up`, `just dev logs`, `just dev restart` and `just dev down` for the individual lifecycle steps. `.env.example` contains the local defaults; `infra/compose/podman-compose.yml` defines the stack wiring. For another isolated stack, choose a unique `COMPOSE_PROJECT_NAME` and non-conflicting `FRONTEND_PORT`, `PLATFORM_API_PORT` and `ORCHESTRATOR_PORT`.

### Private-target development

Set these values in `.env` when scanner pods must reach localhost or a private-network target:

```dotenv
PLATFORM_API_ALLOW_PRIVATE_TARGETS=true
POD_NETNS_MODE=host
POD_NETWORK=
```

Then restart the stack with `just dev restart`, set `STAGEFLOW_API_KEY` to the same value as `PLATFORM_API_TOKEN`, and pass `--allow-private-targets`:

```bash
STAGEFLOW_API_KEY='your-local-api-token' stageflow scan http://127.0.0.1:5173 \
  --api http://localhost:8080 --allow-private-targets
```

Host mode reaches the workstation on Linux; on macOS and Windows it reaches the Podman VM.

### Public deployment checklist

Before exposing StageFlow:

1. Replace every `change-me` value in `.env`.
2. Set explicit CORS origins and `VITE_API_URL`/`VITE_SITE_URL`; rebuild the frontend after changing its build-time URLs.
3. Put a TLS reverse proxy in front: `/api/*` to platform-api and everything else to the frontend. Set `PLATFORM_API_TRUSTED_PROXIES` to the proxy's source address. The proxy should inject no credentials.
4. Keep private-target scanning disabled unless it is an intentional trust boundary, and keep the Podman socket rootless with `no-new-privileges`.
5. Apply host egress policy to reduce DNS-rebinding and metadata-service risk. The isolated job network blocks other Podman networks, not ports bound on the host.
6. Run `just ci`, then complete one URL scan and one ZIP scan through the public edge, including SSE progress and final report retrieval.

Self-hosted deployments accept up to 100 URLs per scan and enable ZIP uploads by default; `PLATFORM_API_MAX_PAGES` and `PLATFORM_API_ZIP_UPLOAD_ENABLED` control intake. The hosted service accepts up to 25 public URLs and does not accept ZIP uploads. Terminal job rows and artifact directories are removed 24 hours after completion.

## Architecture

![Clients submit over HTTP and SSE to the Platform API, which admits jobs into a PostgreSQL queue; the orchestrator claims one job at a time, runs it in rootless Podman pods and publishes one Unified Report v2](assets/architecture.svg)

| Component | Responsibility |
| --- | --- |
| `services/platform-api` | Public HTTP boundary: URL and ZIP intake, SSRF policy, queue admission, job status, SSE and report files |
| `services/orchestrator` | Claims jobs, runs each in a rootless Podman pod, settles scanner outcomes, aggregates the report, enforces deadlines and retention |
| `services/archive-extractor` | For ZIP jobs, validates and extracts the archive, then serves the site to the scanners |
| `services/scanner-runner` | Bun/TypeScript Playwright runtime; each container runs one scanner over the job's pages |
| `internal/jobstore` | PostgreSQL queue, job state machine and status snapshot shared by the API and orchestrator |
| `internal/artifacts` | Job file store on the artifact volume, confined with `os.Root` and atomic renames |
| `libs/go/scannercatalog` | The five scanners, read from embedded manifests with their resource limits and option schemas |
| `libs/contracts` | JSON Schemas with committed Go and TypeScript types, checked for drift in CI |
| `clients/web`, `clients/cli` | The React Router app and the Go CLI |

A job moves through `PENDING → EXTRACTING → READY_TO_SCAN → SCANNING → COMPLETING → DONE`, with `FAILED` reachable from any state. URL jobs skip extraction. PostgreSQL holds the queue, job state and scanner attempts; the artifact volume holds private inputs, the extracted site, screenshots and the unified report. Only one job runs at a time and five may wait.

`UnifiedReportV2` is one document per job: metadata, a summary with an optional 0–100 score, per-scanner status, per-page summaries with screenshot overlays, issues and structured errors. Each issue ID is a hash of its rule, page and element, so the same violation keeps its ID across runs for baselines and regression detection. JSON Schema files under `libs/contracts/` are the source of truth; `just generate-contracts` regenerates the committed types and CI fails on drift.

## Security and privacy

Report vulnerabilities privately through [GitHub Security Advisories](https://github.com/mattboback/stageflow/security/advisories/new). Do not open a public issue for a vulnerability.

The hosted demo is a no-account service for public URLs. A finished scan's job record and files are deleted 24 hours after it completes; the deletion endpoint removes them immediately. Job IDs are unguessable bearer-style references, so anyone who receives a job or report URL can retrieve it until expiry. Daily database backups are retained for 14 days, so a job record can outlive the 24-hour window in a backup. Browser-local projects and saved baseline files never leave your device.

The security model has five nested boundaries:

- **Edge proxy:** terminates TLS, routes `/api/*` to the API, and injects no credentials.
- **API intake:** http(s) only, SSRF classification of every resolved address, request and URL limits, literal-only authentication values, rate limits and timeouts.
- **Archive extraction:** entry-count, per-entry and total size limits, compression-ratio checks and path-traversal prevention in the job's own workspace.
- **Job containers:** unprivileged users, all Linux capabilities dropped, `no-new-privileges`, read-only root filesystem, capped CPU/memory and a disk-backed `/tmp` removed with the pod.
- **Job network:** job pods join a dedicated Podman network with `isolate=strict`; they reach the internet and no other Podman network.

Residual risks remain. URL classification is not connection-level DNS pinning, so DNS rebinding is not fully prevented. Chromium runs with `--no-sandbox`; the container is its sandbox. Public deployments should add host egress policy.

### Authenticated scanning

A job can carry a form-login recipe or Playwright storage state. Every caller sends literal values; the API and orchestrator reject environment references, so nothing from a server's environment reaches a scanner. The CLI resolves recipe references locally.

```bash
stageflow auth capture https://app.example.com/login --output ./auth/state.json
stageflow scan https://app.example.com/profile --auth-state ./auth/state.json
```

Capture files are written with mode `0600`. Storage state is mounted only into the scanners that need it and is deleted when the job ends. Credential values stay in the live job row until terminal cleanup, which scrubs them from job and scanner errors. Public provenance, reports and screenshots of sensitive controls are redacted. A failed login becomes a critical `auth-hydration-failed` finding instead of a silent logged-out scan. Never submit a production or reused credential to the hosted demo; use a controlled self-hosted deployment and a throwaway test account.

## Development

The contributor gate is `just ci`. In addition to the toolchain above, install Python 3 with venv and pip, ShellCheck, jq, ripgrep, zip, tar, and sha256sum. Install the project dependencies and Chromium before running the gate:

```bash
just deps
(cd clients/web && bun x playwright install --with-deps chromium)
just ci
```

`just ci` checks Markdown links, regenerates contracts and fails on drift, validates fixtures, runs actionlint and a full-history gitleaks scan, pre-commit, Go build/lint/race tests/govulncheck, shell and PostgreSQL backup tests, web and scanner CI plus audits, knip and Go dead-code checks, and the Playwright browser suite. It needs a full-history checkout and port 4173 free. It also needs PostgreSQL server/client binaries on `PATH` and must not run as root.

Useful focused commands:

```bash
go test -race ./...                              # Go unit and integration tests
cd clients/web && bun run ci                     # format, lint, typecheck, Vitest, build
cd services/scanner-runner && bun run ci         # scanner runtime checks and Vitest
just web-e2e                                     # Playwright against a production build
just test-podman-live                            # live Podman isolation contracts
just scan-diff-golden                            # ZIP scan + regression diff goldens
just backup-tests                               # PostgreSQL backup/restore canaries
```

`just ci` does not run the live Podman contracts or the full-stack golden regression; those require a running rootless Podman stack.

## Repository layout

| Path | Contents |
| --- | --- |
| `clients/` | React Router web app and Go CLI |
| `services/` | Platform API, orchestrator, archive extractor and scanner runtime |
| `libs/` | JSON contracts, generated types and the shared Go libraries |
| `internal/` | Job store, artifact store and shared Go internals |
| `infra/` | Compose file, Quadlet templates and image/runtime scripts |
| `qa/` | Golden fixtures, scan-diff regression scripts and test sites |
| `devtools/` | Contract generation, dead-code/knip gates and release tooling |
| `assets/` | README screenshots and the architecture diagram |

## License

StageFlow is released under the [MIT License](LICENSE).
