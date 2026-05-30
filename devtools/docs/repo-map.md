# Devtools Repo Map

This map documents the `devtools` slice: the ops job-status CLI, QA
suite-runner, repository helper scripts, and their integration with root
commands and CI. Claims are source-cited in `path:line` form.

## Slice Boundary

| Path | Role | Evidence |
| --- | --- | --- |
| `devtools/ops/job-status-cli` | Go CLI for querying the orchestrator admin API. | The command description says it queries the orchestrator admin API for job and pod details; `main` exits with the result of `run` (`devtools/ops/job-status-cli/main.go:1`, `devtools/ops/job-status-cli/main.go:9`). |
| `devtools/qa/suite-runner` | Go CLI for submitting URL scans across domains and evaluating accessibility thresholds. | `main` uses an HTTP client and calls `run`; README says it runs accessibility scans across multiple domains and generates pass/fail output (`devtools/qa/suite-runner/main.go:9`, `devtools/qa/suite-runner/README.md:3`). |
| `devtools/scripts/install-cli.sh` | Builds and installs the repo-local `stageflow` CLI. | It accepts install directory/name arguments, builds `clients/cli` with version ldflags, installs the binary, checks PATH resolution, and runs `version` (`devtools/scripts/install-cli.sh:7`, `devtools/scripts/install-cli.sh:20`, `devtools/scripts/install-cli.sh:21`, `devtools/scripts/install-cli.sh:24`, `devtools/scripts/install-cli.sh:26`, `devtools/scripts/install-cli.sh:42`). |
| `devtools/scripts/tests/cli-install.test.sh` | Shell regression tests for install script edge cases. | It exercises a shadowed PATH case and a missing PATH case, asserting exit code 1 and explanatory output (`devtools/scripts/tests/cli-install.test.sh:32`, `devtools/scripts/tests/cli-install.test.sh:43`, `devtools/scripts/tests/cli-install.test.sh:48`, `devtools/scripts/tests/cli-install.test.sh:51`, `devtools/scripts/tests/cli-install.test.sh:60`, `devtools/scripts/tests/cli-install.test.sh:65`). |
| `devtools/scripts/precommit/run.mjs` | Staged-file precommit helper for generated contract checks. | It inspects argv files, maps report/provenance contract prefixes to cwd, and runs `bun run check` only for affected contracts (`devtools/scripts/precommit/run.mjs:9`, `devtools/scripts/precommit/run.mjs:13`, `devtools/scripts/precommit/run.mjs:22`). |
| `devtools/scripts/a11y/test-axe-local.js` | Local standalone Playwright + axe smoke helper. | It resolves dependencies from `services/scanner-runner`, defaults target URL to `https://stageflow.org`, runs axe, and prints violation/pass/incomplete/inapplicable counts (`devtools/scripts/a11y/test-axe-local.js:6`, `devtools/scripts/a11y/test-axe-local.js:13`, `devtools/scripts/a11y/test-axe-local.js:24`, `devtools/scripts/a11y/test-axe-local.js:39`, `devtools/scripts/a11y/test-axe-local.js:43`). |
| `devtools/scripts/go` | Go workspace command fan-out helper scripts. | `go-work-dirs.sh` reads `go work edit -json` and prints `DiskPath`; `run-in-work-dirs.sh` runs a label and command in each printed dir (`devtools/scripts/go/go-work-dirs.sh:14`, `devtools/scripts/go/go-work-dirs.sh:20`, `devtools/scripts/go/run-in-work-dirs.sh:9`, `devtools/scripts/go/run-in-work-dirs.sh:14`, `devtools/scripts/go/run-in-work-dirs.sh:19`). |

## Ops Job Status CLI

### Commands And Endpoints

| Command | Flags/Input | Endpoint | Output Shape | Evidence |
| --- | --- | --- | --- | --- |
| `job-status-cli jobs` | `-state`, `-limit` default 20, `-offset` default 0. | `GET /api/v1/jobs?limit=&offset=&state=` | Heading plus tabular `JOB ID`, `STATE`, `INPUT TYPE`, `CREATED`, `COMPLETED`, `ERROR`. | `devtools/ops/job-status-cli/run.go:125`, `devtools/ops/job-status-cli/run.go:128`, `devtools/ops/job-status-cli/run.go:133`, `devtools/ops/job-status-cli/run.go:134`, `devtools/ops/job-status-cli/commands.go:43`, `devtools/ops/job-status-cli/commands.go:64`, `devtools/ops/job-status-cli/commands.go:78` |
| `job-status-cli events <job-id>` | Required job ID; `-limit` default 500, `-offset` default 0, `-payload` optional. | `GET /api/v1/jobs/{job_id}/events?limit=&offset=` | Heading plus event table; optional payload JSON blocks. | `devtools/ops/job-status-cli/run.go:80`, `devtools/ops/job-status-cli/run.go:83`, `devtools/ops/job-status-cli/run.go:85`, `devtools/ops/job-status-cli/run.go:103`, `devtools/ops/job-status-cli/commands.go:155`, `devtools/ops/job-status-cli/commands.go:180`, `devtools/ops/job-status-cli/commands.go:217` |
| `job-status-cli pods` | No command-specific flags. | `GET /api/v1/pods` | Heading plus tabular `POD ID`, `NAME`, `STATUS`, `JOB ID`, `JOB STATE`. | `devtools/ops/job-status-cli/run.go:164`, `devtools/ops/job-status-cli/run.go:173`, `devtools/ops/job-status-cli/commands.go:265`, `devtools/ops/job-status-cli/commands.go:299` |
| `job-status-cli status` | No command-specific flags. | `GET /api/v1/status` | `System Status` summary with job totals/by-state and pod totals/by-status. | `devtools/ops/job-status-cli/run.go:191`, `devtools/ops/job-status-cli/run.go:200`, `devtools/ops/job-status-cli/commands.go:313`, `devtools/ops/job-status-cli/commands.go:335`, `devtools/ops/job-status-cli/commands.go:347`, `devtools/ops/job-status-cli/commands.go:376` |

### Ops Job Status Flow

```mermaid
flowchart TD
  A[job-status-cli argv] --> B[run]
  B --> C[read ORCHESTRATOR_ADMIN_URL or default localhost:8081]
  B --> D[read ORCHESTRATOR_API_TOKEN]
  C --> E{command}
  E -->|jobs| F[parse state limit offset]
  E -->|events| G[parse job_id limit offset payload]
  E -->|pods| H[parse pods flags]
  E -->|status| I[parse status flags]
  F --> J[buildAPIURL /api/v1/jobs]
  G --> K[buildAPIURL /api/v1/jobs/{id}/events]
  H --> L[buildAPIURL /api/v1/pods]
  I --> M[buildAPIURL /api/v1/status]
  J --> N[GET with optional Bearer token]
  K --> N
  L --> N
  M --> N
  N --> O[require HTTP 200]
  O --> P[decode JSON]
  P --> Q[format table or summary]
```

| Concern | Behavior | Evidence |
| --- | --- | --- |
| Default admin URL | `ORCHESTRATOR_ADMIN_URL` overrides `http://localhost:8081`. | `devtools/ops/job-status-cli/run.go:11`, `devtools/ops/job-status-cli/run.go:24`, `devtools/ops/job-status-cli/run.go:26` |
| Auth token | Non-empty `ORCHESTRATOR_API_TOKEN` is sent as `Authorization: Bearer <token>`. | `devtools/ops/job-status-cli/run.go:29`, `devtools/ops/job-status-cli/api.go:25` |
| URL safety | API URLs must parse and use `http` or `https`; invalid schemes return errors. | `devtools/ops/job-status-cli/api.go:32`, `devtools/ops/job-status-cli/api.go:39` |
| JSON response handling | Non-200 responses are returned as errors with response body; 200 responses are decoded as JSON into typed structs. | `devtools/ops/job-status-cli/api.go:55`, `devtools/ops/job-status-cli/api.go:65`, `devtools/ops/job-status-cli/api.go:74` |
| Response contracts | Types map job, pod, job event, and system status JSON fields. | `devtools/ops/job-status-cli/types.go:5`, `devtools/ops/job-status-cli/types.go:17`, `devtools/ops/job-status-cli/types.go:37`, `devtools/ops/job-status-cli/types.go:81` |
| Usage text | Usage lists commands, flags, env vars, and examples. | `devtools/ops/job-status-cli/usage.go:13`, `devtools/ops/job-status-cli/usage.go:17`, `devtools/ops/job-status-cli/usage.go:23`, `devtools/ops/job-status-cli/usage.go:32`, `devtools/ops/job-status-cli/usage.go:37` |

## QA Suite Runner

### Suite Runner Polling/SSE Flow

```mermaid
flowchart TD
  A[suite-runner flags] --> B[load suite YAML]
  B --> C[apply defaults]
  C --> D[context timeout]
  D --> E[for each domain]
  E --> F[POST /api/v1/jobs/urls]
  F --> G[open GET /api/v1/jobs/{job_id}/stream]
  G --> H[read SSE event blocks]
  H --> I{event}
  I -->|done| J[GET /api/v1/jobs/{job_id}]
  I -->|update complete or DONE| J
  I -->|update failed or FAILED| J
  I -->|read error| K[retry after stream_retry_seconds]
  K --> G
  J --> L{state}
  L -->|DONE| M[GET artifacts.results_json]
  L -->|FAILED| N[failed outcome]
  M --> O[evaluate thresholds]
  N --> P[append outcome]
  O --> P
  P --> Q[print summary]
  Q --> R[exit 0 if all passed else 1]
```

| Step | Source Behavior | Evidence |
| --- | --- | --- |
| Flags | `-suite` defaults to `suite.yml`; `-api` defaults to `PLATFORM_API_BASE_URL` or `http://localhost:8080`. | `devtools/qa/suite-runner/run.go:12`, `devtools/qa/suite-runner/run.go:21`, `devtools/qa/suite-runner/run.go:82` |
| YAML contract | Suite has `domains`, `modules`, `screenshot`, `thresholds`, `timeout_seconds`, and `stream_retry_seconds`. | `devtools/qa/suite-runner/types.go:3`, `devtools/qa/suite-runner/types.go:8`, `devtools/qa/suite-runner/types.go:9`, `devtools/qa/suite-runner/types.go:10` |
| Required domains | Loading fails if no domain is configured. | `devtools/qa/suite-runner/suite.go:11`, `devtools/qa/suite-runner/suite.go:22` |
| Defaults | Empty `modules` becomes `[axe, keyboard]`; timeout defaults to 900 seconds; stream retry defaults to 3 seconds. | `devtools/qa/suite-runner/suite.go:29`, `devtools/qa/suite-runner/suite.go:30`, `devtools/qa/suite-runner/suite.go:34`, `devtools/qa/suite-runner/suite.go:38` |
| Submission | Each domain is submitted as a URL scan with `urls`, `modules`, and `screenshot`. | `devtools/qa/suite-runner/run.go:46`, `devtools/qa/suite-runner/api.go:13`, `devtools/qa/suite-runner/api.go:21`, `devtools/qa/suite-runner/api.go:35` |
| SSE wait | The runner opens `/api/v1/jobs/{jobID}/stream` with `Accept: text/event-stream`. | `devtools/qa/suite-runner/stream.go:137`, `devtools/qa/suite-runner/stream.go:144` |
| SSE parsing | The parser skips comments/keepalives, captures `event:` and `data:` lines, and returns on blank-line event boundaries. | `devtools/qa/suite-runner/sse.go:8`, `devtools/qa/suite-runner/sse.go:18`, `devtools/qa/suite-runner/sse.go:22`, `devtools/qa/suite-runner/sse.go:26`, `devtools/qa/suite-runner/sse.go:32` |
| Terminal events | `done`, update type `complete`/`failed`, or state `DONE`/`FAILED` are treated as terminal stream signals. | `devtools/qa/suite-runner/stream.go:170`, `devtools/qa/suite-runner/stream.go:176`, `devtools/qa/suite-runner/stream.go:185` |
| Terminal status fetch | After the stream terminates, the runner fetches `/api/v1/jobs/{jobID}` and then `artifacts.results_json` for `DONE` jobs. | `devtools/qa/suite-runner/stream.go:53`, `devtools/qa/suite-runner/api.go:68`, `devtools/qa/suite-runner/stream.go:60`, `devtools/qa/suite-runner/api.go:103` |
| Threshold evaluation | `max_critical`, `max_serious`, and `max_total` fail when actual values exceed configured limits. | `devtools/qa/suite-runner/evaluate.go:3`, `devtools/qa/suite-runner/evaluate.go:4`, `devtools/qa/suite-runner/evaluate.go:8`, `devtools/qa/suite-runner/evaluate.go:12` |
| Summary and exit code | The runner prints a tabular summary and exits 1 if any outcome failed. | `devtools/qa/suite-runner/format.go:8`, `devtools/qa/suite-runner/format.go:10`, `devtools/qa/suite-runner/run.go:71`, `devtools/qa/suite-runner/run.go:85` |

### Suite Inputs And Outputs

| Field/Output | Meaning | Evidence |
| --- | --- | --- |
| `domains` | One or more scan targets; sample includes `https://example.com` and `https://blog.example.com`. | `devtools/qa/suite-runner/suite.sample.yml:1` |
| `modules` | Scanner module list; sample uses `[axe, keyboard]`. | `devtools/qa/suite-runner/suite.sample.yml:4` |
| `screenshot` | Boolean passed through in URL job submission. | `devtools/qa/suite-runner/suite.sample.yml:5`, `devtools/qa/suite-runner/api.go:24` |
| `thresholds` | `max_critical`, `max_serious`, and `max_total` gates. | `devtools/qa/suite-runner/suite.sample.yml:6`, `devtools/qa/suite-runner/types.go:13` |
| `timeout_seconds` | Overall context timeout. | `devtools/qa/suite-runner/suite.sample.yml:10`, `devtools/qa/suite-runner/run.go:50` |
| `stream_retry_seconds` | Delay between SSE retry attempts. | `devtools/qa/suite-runner/suite.sample.yml:11`, `devtools/qa/suite-runner/run.go:65` |
| Summary rows | `domain`, `state`, `violations(total/critical/serious)`, `job_id`, and `passed`, with optional error. | `devtools/qa/suite-runner/format.go:10`, `devtools/qa/suite-runner/format.go:15`, `devtools/qa/suite-runner/format.go:25` |

## Helper Scripts

| Script | Command Shape | Inputs | Outputs/Side Effects | Evidence |
| --- | --- | --- | --- | --- |
| `devtools/scripts/install-cli.sh` | `devtools/scripts/install-cli.sh [bin_dir] [bin_name]` | Optional install dir default `$HOME/.local/bin`; optional name default `stageflow`; Git metadata; Go toolchain. | Builds `clients/cli`, installs executable, verifies PATH resolution, runs `<bin_name> version`. | `devtools/scripts/install-cli.sh:7`, `devtools/scripts/install-cli.sh:8`, `devtools/scripts/install-cli.sh:17`, `devtools/scripts/install-cli.sh:21`, `devtools/scripts/install-cli.sh:24`, `devtools/scripts/install-cli.sh:26`, `devtools/scripts/install-cli.sh:42` |
| `devtools/scripts/tests/cli-install.test.sh` | `bash devtools/scripts/tests/cli-install.test.sh` | Temporary directories and PATH manipulation. | Fails if install script does not reject a shadowed binary or an off-PATH install dir; prints `cli-install tests passed` on success. | `devtools/scripts/tests/cli-install.test.sh:32`, `devtools/scripts/tests/cli-install.test.sh:51`, `devtools/scripts/tests/cli-install.test.sh:74`, `devtools/scripts/tests/cli-install.test.sh:82` |
| `devtools/scripts/precommit/run.mjs` | `node devtools/scripts/precommit/run.mjs <staged files...>` | Staged file paths from the caller. | Runs `bun run check` in `libs/contracts/report` and/or `libs/contracts/provenance` only when matching prefixes are staged. | `devtools/scripts/precommit/run.mjs:9`, `devtools/scripts/precommit/run.mjs:13`, `devtools/scripts/precommit/run.mjs:22` |
| `devtools/scripts/a11y/test-axe-local.js` | `node devtools/scripts/a11y/test-axe-local.js [url]` | Scanner-runner-installed Playwright and `@axe-core/playwright`; optional URL default `https://stageflow.org`. | Launches headless Chromium, runs axe, prints counts and first-node snippets for violations. | `devtools/scripts/a11y/test-axe-local.js:6`, `devtools/scripts/a11y/test-axe-local.js:13`, `devtools/scripts/a11y/test-axe-local.js:24`, `devtools/scripts/a11y/test-axe-local.js:28`, `devtools/scripts/a11y/test-axe-local.js:39`, `devtools/scripts/a11y/test-axe-local.js:43`, `devtools/scripts/a11y/test-axe-local.js:50` |
| `devtools/scripts/go/go-work-dirs.sh` | `bash devtools/scripts/go/go-work-dirs.sh` | Must run from a directory containing `go.work`; requires `go` and `python3`. | Prints each `go.work` `DiskPath`. | `devtools/scripts/go/go-work-dirs.sh:4`, `devtools/scripts/go/go-work-dirs.sh:9`, `devtools/scripts/go/go-work-dirs.sh:14`, `devtools/scripts/go/go-work-dirs.sh:20` |
| `devtools/scripts/go/run-in-work-dirs.sh` | `bash devtools/scripts/go/run-in-work-dirs.sh <label> <command> [args...]` | Label and command; directory list from `go-work-dirs.sh`. | Runs the command in every Go workspace directory and prefixes progress with `==> <label> <dir>`. | `devtools/scripts/go/run-in-work-dirs.sh:5`, `devtools/scripts/go/run-in-work-dirs.sh:9`, `devtools/scripts/go/run-in-work-dirs.sh:14`, `devtools/scripts/go/run-in-work-dirs.sh:19` |

## Root And CI Integration

| Integration Point | What It Runs | Evidence |
| --- | --- | --- |
| Root `just ci` installs Go quality tools. | `golangci-lint` and `govulncheck`. | `justfile:422`, `justfile:423`, `justfile:463`, `justfile:464` |
| Root `just ci` runs Go lint/vuln across workspace dirs. | Looping commands over directories discovered from `go.work`. | `justfile:478`, `justfile:492` |
| Root `just ci` runs shell install regression tests. | `bash devtools/scripts/tests/cli-install.test.sh`. | `justfile:500` |
| Root `just cli-install` delegates to the install script. | Recipe accepts `BIN_DIR` and `BIN_NAME` and calls `devtools/scripts/install-cli.sh`. | `justfile:546` |
| Root `just shell-test` also runs install regression tests. | Separate quality recipe for shell regression tests. | `justfile:600`, `justfile:604` |
| Root `just project-golden` bridges QA slice into root commands. | Runs `bash qa/e2e/project-scan-golden.sh`. | `justfile:616`, `justfile:617`, `justfile:620` |
| GitHub CI Go job uses devtools Go fan-out script. | Build, lint, test, and vulncheck all run through `devtools/scripts/go/run-in-work-dirs.sh`; install script test runs in CI too. | `.github/workflows/ci.yml:98`, `.github/workflows/ci.yml:101`, `.github/workflows/ci.yml:104`, `.github/workflows/ci.yml:112`, `.github/workflows/ci.yml:115` |
| GitHub CI includes web Storybook a11y coverage. | `clients_web_storybook` is named "Web App Storybook (interaction + a11y)" and runs `bun run test-storybook:ci`. | `.github/workflows/ci.yml:153`, `.github/workflows/ci.yml:154`, `.github/workflows/ci.yml:184` |
| Golden regression workflow is separate from main CI. | Workflow is manual plus daily cron, builds CLI, runs `qa/e2e/project-scan-golden.sh`, exports golden artifacts, and uploads them. | `.github/workflows/golden-regression.yml:4`, `.github/workflows/golden-regression.yml:6`, `.github/workflows/golden-regression.yml:70`, `.github/workflows/golden-regression.yml:108`, `.github/workflows/golden-regression.yml:111`, `.github/workflows/golden-regression.yml:123` |
| Operations docs expose both devtools CLIs. | `docs/operations/devtools.md` has sections and build commands for `job-status-cli` and `suite-runner`. | `docs/operations/devtools.md:157`, `docs/operations/devtools.md:165`, `docs/operations/devtools.md:172`, `docs/operations/devtools.md:180` |

## Environment And Config Expectations

| Tool | Environment/Config | Default | Behavior | Evidence |
| --- | --- | --- | --- | --- |
| `job-status-cli` | `ORCHESTRATOR_ADMIN_URL` | `http://localhost:8081` | Selects orchestrator admin API base URL. | `devtools/ops/job-status-cli/run.go:11`, `devtools/ops/job-status-cli/run.go:24`, `devtools/ops/job-status-cli/run.go:26` |
| `job-status-cli` | `ORCHESTRATOR_API_TOKEN` | Empty | If non-empty, sent as bearer auth. | `devtools/ops/job-status-cli/run.go:29`, `devtools/ops/job-status-cli/api.go:25` |
| `suite-runner` | `PLATFORM_API_BASE_URL` or `-api` | `http://localhost:8080` | Selects Platform API base URL for submissions, streams, status, and result fetches. | `devtools/qa/suite-runner/run.go:12`, `devtools/qa/suite-runner/run.go:21`, `devtools/qa/suite-runner/api.go:35`, `devtools/qa/suite-runner/stream.go:137` |
| `suite-runner` | YAML suite file | `suite.yml` | Defines domains, modules, screenshots, thresholds, timeout, and stream retry. | `devtools/qa/suite-runner/run.go:20`, `devtools/qa/suite-runner/types.go:3`, `devtools/qa/suite-runner/suite.sample.yml:1` |
| `install-cli.sh` | Args `bin_dir`, `bin_name` | `$HOME/.local/bin`, `stageflow` | Controls install destination and command name. | `devtools/scripts/install-cli.sh:7`, `devtools/scripts/install-cli.sh:8` |
| `test-axe-local.js` | CLI URL arg | `https://stageflow.org` | Page tested by standalone axe run. | `devtools/scripts/a11y/test-axe-local.js:24` |
| Go workspace scripts | Current directory | Must contain `go.work` | They fail if `go.work` is absent, then discover directories from `go work edit -json`. | `devtools/scripts/go/go-work-dirs.sh:9`, `devtools/scripts/go/go-work-dirs.sh:14` |

## Tests And Package Dependencies

| Component | Dependencies/Tests | Evidence |
| --- | --- | --- |
| `job-status-cli` | Standalone Go module using Go `1.26.3`; test files cover run/format behavior. | `devtools/ops/job-status-cli/go.mod:1`, `devtools/ops/job-status-cli/go.mod:3`, `devtools/ops/job-status-cli/run_test.go:44`, `devtools/ops/job-status-cli/format_test.go:20` |
| `suite-runner` | Standalone Go module using Go `1.26.3` and `gopkg.in/yaml.v3`; tests cover defaults, threshold evaluation, and SSE parsing. | `devtools/qa/suite-runner/go.mod:1`, `devtools/qa/suite-runner/go.mod:3`, `devtools/qa/suite-runner/go.mod:5`, `devtools/qa/suite-runner/suite_test.go:8`, `devtools/qa/suite-runner/poll_test.go:10`, `devtools/qa/suite-runner/sse_test.go:26` |
| Install script tests | Shell tests are invoked by root `just ci`, root `just shell-test`, and GitHub CI. | `justfile:500`, `justfile:604`, `.github/workflows/ci.yml:112` |
| Standalone axe helper | Uses scanner-runner's Node dependency tree rather than declaring its own package manifest in `devtools/scripts/a11y`. | `devtools/scripts/a11y/test-axe-local.js:6`, `devtools/scripts/a11y/test-axe-local.js:13`, `devtools/scripts/a11y/test-axe-local.js:17` |

## Uncertainties And Follow-Up Checks

| Item | Current State | Evidence |
| --- | --- | --- |
| `job-status-cli` README documents `ORCHESTRATOR_ADMIN_URL` but omits `ORCHESTRATOR_API_TOKEN`; usage text and code include the token. | The more complete source of truth is code/usage. | `devtools/ops/job-status-cli/README.md:45`, `devtools/ops/job-status-cli/usage.go:32`, `devtools/ops/job-status-cli/usage.go:34`, `devtools/ops/job-status-cli/run.go:29` |
| `suite-runner` default modules include `keyboard`, but this map did not verify a scanner catalog entry named `keyboard`. | The default is documented as code behavior only; scanner availability should be checked in scanner catalog if changing suite defaults. | `devtools/qa/suite-runner/suite.go:30` |
| `test-axe-local.js` is not referenced by the cited root justfile/CI lines. | It exists as an ad hoc local helper in this slice, while CI a11y coverage is through Storybook and scanner-runner jobs. | `devtools/scripts/a11y/test-axe-local.js:24`, `.github/workflows/ci.yml:153`, `.github/workflows/ci.yml:184`, `.github/workflows/ci.yml:199` |
