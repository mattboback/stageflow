# StageFlow Evaluator Guide

This guide is for hiring managers, staff+ engineers, and reviewers who want to evaluate
StageFlow quickly as an open-source portfolio project.

## Two-minute path

1. Open the live demo at [stageflow.org](https://stageflow.org).
2. Run a scan against any public URL.
3. Skim the generated report: overview, grouped issues, visual evidence, and artifacts.
4. Open the committed fixture at
   [`libs/contracts/report/fixtures/unified-report.v2.all-scans.json`](../libs/contracts/report/fixtures/unified-report.v2.all-scans.json)
   to see the output contract without running the stack.

## Five-to-fifteen minute code tour

| Area | Start here | What to look for |
| --- | --- | --- |
| Product overview | `README.md` | CLI, hosted demo, self-hosting path, scanner list |
| Architecture | `docs/architecture/system.md` | Trust boundaries, job flow, baseline memory, reviewer code map |
| API intake | `services/platform-api/internal/api/security.go` | URL validation, private-target policy, SSRF guard |
| Job lifecycle | `services/orchestrator/internal/domain/jobs/transitions.go` | Explicit state machine and transition tests |
| Scanner runtime | `services/scanner-runner` | Plugin execution, Playwright integration, scanner contracts |
| Report contract | `libs/contracts/report` | Schema-first report design and generated Go/TypeScript types |
| CLI | `clients/cli` | SSE streaming, rendering, exit codes, project/baseline commands |
| Web UI | `clients/web/app` | React Router report UI, filters, evidence, artifacts |
| CI | `.github/workflows/ci.yml` | Go/web/scanner checks, secrets scan, images, SBOM, Trivy |

## Interesting engineering surfaces

### 1. Safe intake boundary

StageFlow accepts URLs and static-site ZIP archives. Good review points:

- URL scheme and private-network checks in `services/platform-api/internal/api/security.go`;
- explicit `--allow-private-targets` behavior for trusted local scans;
- ZIP extraction limits and path traversal protection in `services/archive-extractor`;
- API-key auth defaults and local-only auth disablement.

### 2. Durable orchestration

The orchestrator consumes NATS events and owns job state transitions. Good review points:

- transition rules and tests in `services/orchestrator/internal/domain/jobs`;
- per-job Podman lifecycle planning;
- scanner completion tracking by expected scanner;
- report aggregation after all required scanner outputs arrive.

### 3. Contract-driven reports

The repository uses schemas as the integration point between services:

- `libs/contracts/report/schema/unified-report.v2.schema.json`;
- generated Go and TypeScript types;
- stable issue IDs for baseline diffing;
- one report shape consumed by the CLI and web app.

### 4. CLI as a CI surface

The CLI is not just a wrapper around the browser UI. It adds:

- terminal progress from SSE;
- text, Markdown, and JSON rendering;
- `--fail-on` severity gates with machine-readable exit codes;
- remote project workflows for baseline promotion and regression detection;
- `stageflow dev` for local dev-server scan loops.

### 5. Web report UX

The React Router app focuses on triage:

- score and severity summaries;
- grouped issues by scanner/rule;
- occurrence evidence and screenshots;
- visual/page review overlays;
- report artifacts and scanner errors.

## Testing and quality gates

The main quality gates are:

- workflow linting plus stale-vocabulary checks;
- gitleaks secret scanning;
- Go build, lint, race tests, and vulnerability checks;
- web lint, typecheck, and build;
- scanner-runner CI with browser-backed checks;
- container image builds, SBOM generation, and Trivy scanning;
- golden project-scan regression tests.

## How this represents the author's work

This is a solo-authored project covering product direction, Go services, TypeScript scanner
runtime, React UI, CLI UX, contracts, infra examples, tests, and documentation.

The most useful files for assessing depth are:

1. `services/orchestrator/internal/domain/jobs/transitions.go`
2. `services/platform-api/internal/api/security.go`
3. `services/scanner-runner`
4. `libs/contracts/report`
5. `clients/cli/internal/jobstream`
6. `clients/cli/report_output.go`
7. `clients/web/app/components/report`
8. `qa/e2e/project-scan-golden.sh`
