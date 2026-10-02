# StageFlow

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

StageFlow helps developers **audit a frontend and compare deployments**.
Run five built-in scanners to find accessibility, SEO, link, and security-header
problems, inspect page evidence, and choose what to fix first. Compare saved scan
reports to spot new regressions and verify fixes in the browser or CI.

**▶ Live demo: [stageflow.org](https://stageflow.org)** — scan any public URL and
explore a real, interactive report in the browser. No StageFlow account is
required on the demo; uploads and generated artifacts expire after 24 hours.

![StageFlow Review workspace for a scan of stageflow.org: site score, severity and scanner filters, page list, screenshot evidence, and the findings on the selected page](docs/images/report-review.png)

Reviewing this codebase? The [code tour](docs/code-tour.md) maps contracts, SSRF guards, the job state machine, and the test strategy into a 5–15 minute path.

## Try It in 60 Seconds

The CLI is a single Go binary that submits scans to any StageFlow API and
streams live results to your terminal. Download it from
[GitHub Releases](https://github.com/mattboback/stageflow/releases) and point
it at the hosted demo — nothing to clone, nothing to deploy:

```bash
# Linux amd64 shown; macOS (darwin) and Windows assets ship on the same release.
version=v0.5.0
curl -fsSL "https://github.com/mattboback/stageflow/releases/download/${version}/stageflow_${version}_linux_amd64.tar.gz" | tar -xz
install -m 0755 stageflow ~/.local/bin/stageflow   # any directory on your PATH

stageflow scan https://example.com
```

With Go 1.26 installed, `go install github.com/mattboback/stageflow/clients/cli/cmd/stageflow@latest`
installs it too. Working from a clone instead? `just cli-install` builds and
installs the same binary.

Prefer a browser? Submit the same scan at [stageflow.org](https://stageflow.org)
and watch it stream live.

Common CLI workflows:

```bash
# Choose scanners and return JSON for automation.
stageflow scan https://example.com --scanner axe,seo --format json

# Fail the command (exit 1) when serious-or-worse issues are found — a CI gate.
stageflow scan https://example.com --fail-on serious

# Scan several routes in one job.
stageflow scan https://example.com https://example.com/about --format markdown

# Scan a local build before it ships anywhere: the directory is zipped,
# uploaded, and served by the platform — no dev server or public URL needed.
stageflow scan ./dist
```

![StageFlow CLI streaming a live scan and printing the machine-readable JSON envelope](docs/images/demo.gif)

Exit codes are machine-readable: `0` clean, `1` threshold tripped, `2` CLI/API error, incomplete scanner coverage, or incompatible baseline identity. Release binaries are on [GitHub Releases](https://github.com/mattboback/stageflow/releases); see the [CLI guide](docs/cli.md) for workflows and run `stageflow --help` or `stageflow <command> --help` for exact flags.

## The Report

Every scan — CLI or browser — produces one unified report built for triage. Use
Review for page screenshots, overlays, and human decisions; Findings for
searchable issues grouped by scanner and rule; and Artifacts for owned HTML and
JSON report downloads.

When a scanner supplies element-location evidence, StageFlow overlays the
finding on a full-page screenshot. Select a bounding box in Review to open that
finding's guidance, references, and evidence. Page-level and manual-review
findings remain available for triage even when no visual overlay exists.

The report shape is a versioned contract. To see it without running a scan,
open the committed fixture
[`libs/contracts/report/fixtures/unified-report.v2.all-scans.json`](libs/contracts/report/fixtures/unified-report.v2.all-scans.json)
— a full multi-scanner report validated against the [public v2 schema](https://stageflow.org/schemas/unified-report.v2.json).

## Regression Memory: The Headline Capability

Save a known-good scan as a baseline, then compare each deployment's report:

```bash
stageflow scan ./dist --format json --max-issues 0 > baseline.json
# After changing and rebuilding the site:
stageflow scan ./dist --format json --max-issues 0 > current.json
stageflow diff baseline.json current.json --fail-on-new=any
```

The comparison runs locally and exits `1` when the selected regression gate
trips. Store baseline files in your repository or CI artifacts; replace them
explicitly when accepting a change. See the [CLI guide](docs/cli.md#baselines-and-comparisons).

For a GitHub step summary, after building the site and reviewing a baseline:

```bash
stageflow scan ./dist --format json --max-issues 0 > current.json && stageflow diff baseline.json current.json --format markdown --fail-on-new=any >> "$GITHUB_STEP_SUMMARY"
```

Incomplete scanner coverage exits `2`; accept it explicitly with `--allow-partial` on scan and diff. Baselines saved before stable identity support require a fresh reviewed scan. A [minimal composite action](action.yml) provides the same regression gate.

Prefer the browser? The `/projects` workspace keeps named projects, completed
runs, and manually promoted baselines in your browser's IndexedDB. It requires
no StageFlow account and performs no cloud sync; clearing site data removes that
local project history.

## Going Deeper

- **CLI workflows** — URL and static-build scans, saved baselines, and CI gates: [docs/cli.md](docs/cli.md).
- **Architecture** — clients → Platform API → PostgreSQL queue → Orchestrator →
  per-job rootless Podman pods, with SSRF and ZIP-bomb guards at the trust
  boundaries: [docs/architecture.md](docs/architecture.md).
- **Self-hosting** — local modes, public edge guidance, and the hosted-demo boundary: [docs/self-hosting.md](docs/self-hosting.md).
- **Product & design rationale** — who the platform serves and why the report
  UI looks the way it does: [docs/product.md](docs/product.md) and
  [docs/design.md](docs/design.md).
- **Documentation assets** — reproduce the Review screenshot and social card
  from committed local inputs: [docs/images/README.md](docs/images/README.md).
- **Engineering case study** — the major architecture, security, contract, and
  delivery tradeoffs, with links to inspectable evidence: [docs/case-study.md](docs/case-study.md).

![StageFlow clients submit through the Platform API and PostgreSQL queue to isolated scanner pods that produce one unified report](docs/images/architecture.svg)

## Engineering Evidence

| Signal                               | Repository evidence                                                                      |
| ------------------------------------ | ---------------------------------------------------------------------------------------- |
| End-to-end regression proof          | [Golden Regression workflow](.github/workflows/golden-regression.yml)                    |
| Schema-first Go/TypeScript contracts | [Contract fixture](libs/contracts/report/fixtures/unified-report.v2.all-scans.json)      |
| Explicit trust boundaries            | [Security model](docs/architecture.md#security-model) and [security policy](SECURITY.md) |
| Broad automated quality gate         | [CI workflow](.github/workflows/ci.yml) and `just ci`                                    |
| Reviewer-oriented source path        | [Five-to-fifteen-minute code tour](docs/code-tour.md)                                    |

## Self-Host Locally

With Go 1.26.6, Node.js 24, Bun 1.3.8, Podman Compose, and `just` installed:

```bash
cp .env.example .env
just diagnose
just demo
```

The web UI starts at `http://localhost:3000` and the API at `http://localhost:8080`. See [Self-hosting](docs/self-hosting.md) for private-target development, manual lifecycle commands, public edge guidance, and required hardening.

## What StageFlow Runs

Built-in scanners:

- `axe` for accessibility
- `lighthouse` for accessibility, best-practices, and SEO audits
- `seo` for title, metadata, heading, and structured-data checks
- `security-headers` for common HTTP response-header policy
- `link-checker` for internal and external link health

StageFlow does not measure performance: Lighthouse's performance category is
not run.

A scan runs all five unless you pick some with `--scanner` (CLI) or `modules`
(API). Each scanner's memory limit is set in its manifest under
`libs/go/scannercatalog/manifests/`.

## Repository Map

| Path                         | Purpose                                                                        |
| ---------------------------- | ------------------------------------------------------------------------------ |
| `clients/cli`                | Go CLI for scans, baselines, reports, and shell completion                     |
| `clients/web`                | React Router/Vite app for hosted/local browser workflows                       |
| `services/platform-api`      | Public HTTP API, auth, CORS, URL/ZIP intake, SSE, transactional job intake, reports |
| `services/orchestrator`      | Job state machine, Podman pod lifecycle, aggregation, persistence              |
| `services/archive-extractor` | Safe ZIP extraction and static serving inside job pods                         |
| `services/scanner-runner`    | Bun/TypeScript Playwright scanner runtime                                      |
| `libs/contracts`             | JSON Schema contracts with committed generated Go/TypeScript types             |
| `infra`                      | Compose, scanner, and security examples                                 |
| `docs`                       | Canonical architecture, CLI, self-hosting, design, and reference documentation |

## Development

Install dependencies and local config:

```bash
just setup
```

Generated contract code is committed. Run `just generate-contracts` after changing
a schema; `just ci` checks that the generated code is current.

Run the repository quality gate:

```bash
just ci
```

Pre-commit hooks are a fast local guard for common mistakes before a commit.
`just ci` is the full repo quality gate and is the command to use before
opening a PR or handing work to CI.

Useful focused checks:

```bash
just shell-tests
(cd clients/web && bun run ci)
(cd services/scanner-runner && bun run ci)
just dead-code
go test -cover ./...
just scan-diff-golden
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for pull request expectations and
workspace-specific validation commands.

Committed screenshots and reviewer-facing images live under `docs/images`.
Temporary QA/build evidence belongs under ignored artifact, output, or cache
paths such as `artifacts/`, `output/`, and `.cache/`; `just clean` may remove
those ephemeral files.

## Security

StageFlow scans websites and can run browser automation, so deployment
boundaries matter. Public deployments should keep API authentication enabled,
avoid private-target scans unless explicitly intended, isolate scanner pods, and
use environment-specific credentials.

Report vulnerabilities privately through [SECURITY.md](SECURITY.md). Do not
open public issues for vulnerabilities. Before using the hosted demo, review
its [data handling and 24-hour retention policy](docs/privacy.md).

## License

StageFlow is released under the [MIT License](LICENSE).
