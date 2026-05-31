# StageFlow Master Audit

Generated: 2026-05-30 UTC

This file concatenates the section audit files produced during the pre-fix review and their post-fix status notes.

---

# Source: audit.md

# Root, Infrastructure, CI, and Documentation Audit

Date: 2026-05-30 UTC

Scope: repository root, `infra`, `qa`, `devtools`, `.github`, and top-level docs. Existing worktree changes were treated as user-owned. This is a pre-fix audit.

## Findings

### High

No high-severity findings were identified in this slice during the initial pass.

### Medium

1. Scanner configuration documentation names the wrong setup source.

   The configuration reference says `just setup` creates `infra/scanners/scanners.yaml` from `services/orchestrator/config/scanners.yaml` (`docs/reference/configuration.md:138`). The actual setup path calls `infra/scripts/ensure-scanner-config.sh` from `just deps` (`justfile:49`, `justfile:54`, `justfile:55`), and that script copies `infra/scanners/scanners.example.yaml` when the local config is missing (`infra/scripts/ensure-scanner-config.sh:4`, `infra/scripts/ensure-scanner-config.sh:6`, `infra/scripts/ensure-scanner-config.sh:23`, `infra/scripts/ensure-scanner-config.sh:24`).

   Impact: first-time operators and reviewers get conflicting guidance about which scanner override file is authoritative.

   Remediation: update the docs to name `infra/scanners/scanners.example.yaml` as the setup template and clarify that `services/orchestrator/config/scanners.yaml` is a service-local example/default.

2. The public `just deploy` recipe has a personal default control-plane path.

   `just deploy` defaults `STAGEFLOW_DEPLOY_CONTROL_PLANE` to `/home/matt/Deployment` (`justfile:20`, `justfile:25`). The recipe is guarded and fails with a useful message when that path is absent (`justfile:29`, `justfile:32`), but the default still exposes personal machine layout in the main public command surface.

   Impact: for an employer-facing open-source portfolio repo, a personal absolute path in the root task runner looks less polished than the surrounding documentation.

   Remediation: remove the personal default and require `STAGEFLOW_DEPLOY_CONTROL_PLANE` for hosted-demo deployment, or move hosted deploy delegation into a local/private justfile.

3. Dead-code analysis is now blocking in CI.

   The `dead_code` workflow job runs web and scanner-runner dead-code analysis and now fails CI when any command fails.

   Impact: the prior portfolio-polish gap is closed; unused dependencies/exports now block pull requests.

   Remediation status: completed after removing scanner-runner's unused provenance dependency and internal-only exported type noise.

4. Standalone MinIO bootstrap defaults differ from `.env.example`.

   `.env.example` uses `MINIO_ROOT_USER=minioadmin` and `MINIO_ROOT_PASSWORD=change-me` (`.env.example:6`, `.env.example:7`). The standalone bootstrap script falls back to `admin/password` when environment variables are missing (`infra/minio/init-buckets.sh:9`, `infra/minio/init-buckets.sh:10`, `infra/minio/init-buckets.sh:11`).

   Impact: the compose path requires credentials, but direct script invocation can accidentally target a local MinIO with unrelated default credentials. This is mainly an operator-safety and documentation consistency issue.

   Remediation: require explicit root credentials in the script, or source `.env`/match `.env.example` defaults intentionally.

### Low

5. Local ignored scanner config can contradict its own comment.

   The ignored local `infra/scanners/scanners.yaml` in this worktree says only the "core two" scanners are enabled, but the file enables several additional scanners (`infra/scanners/scanners.yaml:3`, `infra/scanners/scanners.yaml:23`, `infra/scanners/scanners.yaml:26`, `infra/scanners/scanners.yaml:29`, `infra/scanners/scanners.yaml:32`, `infra/scanners/scanners.yaml:35`). The tracked example file is consistent (`infra/scanners/scanners.example.yaml:19`, `infra/scanners/scanners.example.yaml:23`, `infra/scanners/scanners.example.yaml:35`).

   Impact: this does not affect a fresh clone because the file is gitignored, but it can confuse local verification.

   Remediation: refresh the ignored local file from the tracked example or update its comment.

## Strengths

- CI covers workflow linting, secrets scanning, Go build/lint/test/vuln checks, web CI, Storybook checks, scanner-runner CI, image builds, SBOM generation, and Trivy high/critical scans.
- Local setup commands are consolidated through `just`, with prerequisite diagnosis and protected-host guardrails for shared machines.
- `.env.example` uses placeholder values and documents that real secrets should not be committed.
- Infra docs clearly separate the repo-managed local/staging/self-hosted examples from the hosted production control plane.

## Validation

- `just --list`: passed and printed 39 recipes.
- Shell syntax checks for tracked `*.sh` scripts under `infra`, `qa`, and `devtools`: passed.
- `git diff --check`: passed.

## Finding Counts

- Critical: 0
- High: 0
- Medium: 4
- Low: 1

---

# Source: clients/web/audit.md

# Web Client Audit

Date: 2026-05-30 UTC

Scope: `clients/web` only. Existing worktree changes were treated as user-owned. This is a pre-fix audit written before the high-risk web fixes were implemented.

## Findings

### High

1. Scanner catalog failures are hidden behind enabled hardcoded fallback scanners.

   `fetchScanners` returns `getFallbackScannersResponse()` for both non-OK API responses and thrown fetch/JSON errors (`clients/web/src/lib/api/client.ts:271`, `clients/web/src/lib/api/client.ts:283`, `clients/web/src/lib/api/client.ts:296`). The fallback definitions are all marked `enabled: true` and use `version: "local-fallback"` (`clients/web/src/lib/api/client.ts:26`, `clients/web/src/lib/api/client.ts:30`, `clients/web/src/lib/api/client.ts:34`). `PlaygroundPage` expects `fetchScanners` to reject so it can populate `error`, but the catch block is unreachable for catalog outages because `fetchScanners` swallows the error (`clients/web/src/lib/components/playground/PlaygroundPage.svelte:103`, `clients/web/src/lib/components/playground/PlaygroundPage.svelte:110`).

   Impact: the UI can submit scans using stale built-in scanner IDs while the live registry is unavailable, misconfigured, or intentionally disabling scanners. That makes production state invisible to users and can create confusing 422/server-side errors later.

   Remediation: remove the silent enabled fallback from the production request path. Surface a scanner-catalog load error, keep the submit button disabled until the catalog is loaded, and reserve fallback definitions only for explicit tests/story fixtures.

### Medium

2. Report fetch rejections can leave report fetching stuck in-flight.

   `fetchReport` sets `reportFetchInFlight = true` before awaiting `resolvedDeps.reportPort.fetch` (`clients/web/src/lib/stores/scan-monitor/create.ts:263`, `clients/web/src/lib/stores/scan-monitor/create.ts:273`, `clients/web/src/lib/stores/scan-monitor/create.ts:274`). The function clears the flag on handled response states (`clients/web/src/lib/stores/scan-monitor/create.ts:276`, `clients/web/src/lib/stores/scan-monitor/create.ts:281`, `clients/web/src/lib/stores/scan-monitor/create.ts:287`, `clients/web/src/lib/stores/scan-monitor/create.ts:304`), but it does not use `try`/`finally` around the awaited port call.

   Impact: the default browser `createReportPort` catches normal fetch errors (`clients/web/src/lib/stores/scan-monitor/ports.ts:42`, `clients/web/src/lib/stores/scan-monitor/ports.ts:69`), but the store contract itself is brittle. A rejected injected port, future port implementation, or unexpected exception can leave `reportFetchInFlight` true and suppress retries.

   Remediation: wrap report fetching in `try`/`catch`/`finally`, clear in-flight state on every path, and surface a retryable report error through the snapshot.

3. ZIP upload promises a 100 MB limit but only checks the filename before upload.

   `PlaygroundZipUpload` accepts files when the name ends in `.zip` (`clients/web/src/lib/components/playground/PlaygroundZipUpload.svelte:22`, `clients/web/src/lib/components/playground/PlaygroundZipUpload.svelte:45`) and displays "Maximum 100MB" (`clients/web/src/lib/components/playground/PlaygroundZipUpload.svelte:116`). The size error is only mapped after the server returns 413 (`clients/web/src/lib/api/client.ts:249`, `clients/web/src/lib/api/client.ts:250`).

   Impact: users can select and start uploading a clearly oversized archive before receiving feedback. The backend still protects the service, but the frontend experience is less polished than the copy implies.

   Remediation: add client-side size validation aligned to the backend limit while keeping the server-side 413 handling.

4. Coverage thresholds omit the main API and scan-monitoring paths.

   Vitest coverage includes selected utility/components/store helper files, but omits `src/lib/api/client.ts`, `src/lib/api/sse.ts`, `src/lib/stores/scan-monitor/create.ts`, and `PlaygroundPage.svelte` (`clients/web/vitest.config.ts:17`, `clients/web/vitest.config.ts:39`). Targeted tests exist for some of these files, but the coverage gate does not require them.

   Impact: product-critical scan submission, scanner loading, SSE/poll fallback, and report fetching can regress without affecting the coverage threshold.

   Remediation: add the primary API/store files to the coverage include list after the high-risk store/client tests are strengthened.

## Post-Fix Status

- High finding 1 was addressed by surfacing scanner-catalog load failures instead of returning enabled fallback scanners in production request paths.
- Medium finding 2 was addressed by clearing report-fetch in-flight state on rejected report-port calls and surfacing a retryable report error.
- Medium finding 3 was addressed with client-side ZIP extension and 100 MB size validation before upload, while retaining server-side 413 handling.
- Medium finding 4 was addressed by adding the primary playground/API/SSE/scan-monitor paths to the Vitest coverage include set and strengthening tests until the existing coverage threshold passes.

## Strengths

- The frontend uses SvelteKit 5 with strict type checking and a CI script that combines formatting, strict linting, type checking, and coverage.
- URL input normalization/validation is clear and user-facing before scan submission.
- Scan monitor tests cover streaming fallback, polling fallback, terminal side effects, and report retry behavior.
- UI component tests check accessible labels and core playground interactions.

## Validation

- `bun run format:check` from `clients/web`: passed.
- `bun run lint:strict` from `clients/web`: passed.
- `bun run type-check` from `clients/web`: passed with 0 errors and 0 warnings.
- `bun run test:coverage` from `clients/web`: passed, 59 test files and 420 tests. Branch coverage was 80.26%, clearing the existing 80% threshold.

Not run: Storybook tests, browser E2E, or production API smoke tests.

## Finding Counts

- Critical: 0
- High: 1
- Medium: 3
- Low: 0

---

# Source: clients/cli/audit.md

# StageFlow CLI Audit

Date: 2026-05-30 UTC  
Scope: `clients/cli` only. Existing dirty worktree changes were treated as user-owned; this audit did not modify code.

## Findings

### High

1. Live `diff` scans bypass the target normalization and private-target guard used by `scan`.

   `stageflow scan` normalizes targets and refuses private or loopback targets against a non-local API before submission (`clients/cli/cobra_scan.go:52`, `clients/cli/cobra_scan.go:62`). The live URL path in `stageflow diff`, however, builds a `SubmitJobRequest` directly from `currentTarget` and sends it to `runScanJob` without calling `urlcheck.NormalizeTargets` or `urlcheck.ValidateLocalTargets` (`clients/cli/cobra_diff.go:156`, `clients/cli/cobra_diff.go:163`). `IsRemoteTarget` also only treats strings prefixed with `http://` or `https://` as live targets (`clients/cli/internal/diffrender/diffrender.go:91`), so the diff command has different URL semantics from scan.

   Impact: a command such as `stageflow diff baseline.json http://127.0.0.1:5173 --api https://stageflow.org` can submit a loopback target to a remote API path that direct `scan` explicitly blocks. This is a correctness and trust-boundary regression in a command intended for CI regression checks.

   Remediation: route live diff targets through the same normalization and private-target validation helper path as `scan`; add tests covering loopback/private targets with local and non-local API URLs, plus bare-host behavior if parity with `scan example.com` is intended.

2. Several API commands can hang indefinitely because they use `http.DefaultClient` through a background command context.

   `apiclient.NewClient` substitutes nil clients with `http.DefaultClient` (`clients/cli/internal/apiclient/client.go:19`, `clients/cli/internal/apiclient/client.go:21`). Non-scan commands then call API methods with `cmd.Context()` and no per-command timeout, for example `scanners` (`clients/cli/cobra_scanners.go:18`, `clients/cli/cobra_scanners.go:21`), `report` (`clients/cli/cobra_report.go:25`, `clients/cli/cobra_report.go:28`, `clients/cli/cobra_report.go:41`), and remote project CRUD/list commands (`clients/cli/cobra_project_remote.go:75`, `clients/cli/cobra_project_remote.go:77`, `clients/cli/cobra_project_update.go:46`, `clients/cli/cobra_project_update.go:48`).

   Impact: a stalled TCP connection or API handler can leave common portfolio-demo commands waiting forever. The scan/project paths have explicit operation timeouts (`clients/cli/scan_job.go:148`, `clients/cli/scan_job.go:201`), so the behavior is inconsistent across the CLI.

   Remediation: add a shared API operation timeout, either through a default `http.Client{Timeout: ...}` or through command-level `context.WithTimeout`; expose a `--timeout` flag where long API operations are expected. Add `httptest.Server` cases that accept a connection and delay the response.

### Medium

3. Project readiness configuration can panic the CLI process for non-positive intervals.

   `configDuration` accepts any duration parsed by `time.ParseDuration`, including `0s` and negative values (`clients/cli/project_config.go:276`, `clients/cli/project_config.go:282`). `waitForReady` then passes the configured interval directly to `time.NewTicker` (`clients/cli/dev_stack.go:108`, `clients/cli/dev_stack.go:119`), which panics for a non-positive duration.

   Impact: a malformed `.stageflow/config.yaml` can crash `stageflow project` or `stageflow project doctor` instead of producing a clean exit-code-2 configuration error.

   Remediation: validate positive durations for `dev.ready.interval`, `dev.ready.timeout`, `dev.stop.timeout`, and scan timeouts during config validation or immediately after parsing. Add tests for `interval: 0s` and negative durations.

4. Local Project Mode does not expose the same quality-gate flags as scan/report/hosted.

   The local `project` command defines only `--timeout`, `--max-issues`, and hidden `--no-stream` (`clients/cli/cobra_project.go:31`, `clients/cli/cobra_project.go:38`). When rendering, it passes only `Format` and `MaxIssues` into `renderUnifiedReport` (`clients/cli/project_run.go:233`, `clients/cli/project_run.go:236`). By contrast, `scan`, `report`, and `project hosted` bind the shared report flags including `--fail-on`, severity/category filters, summary mode, and grouping (`clients/cli/cobra_scan.go:148`, `clients/cli/cobra_report.go:56`, `clients/cli/cobra_project.go:78`).

   Impact: the README positions Project Mode as the local agent gating loop (`clients/cli/README.md:63`, `clients/cli/README.md:71`), but the primary local command cannot use the same severity gate documented for CI-style quality gates (`clients/cli/README.md:95`, `clients/cli/README.md:121`). Users must switch commands or parse output manually.

   Remediation: bind the shared report flags to `project` as well, or document that local Project Mode intentionally supports only truncation. Add command tests for `stageflow project --fail-on serious`.

5. The public `ai` command is under-specified and not covered by tests.

   The README lists `ai` as a first-class command (`clients/cli/README.md:36`), but the implementation hardcodes provider/model values in the scanner config (`clients/cli/cobra_ai.go:81`, `clients/cli/cobra_ai.go:83`) and always writes JSON directly (`clients/cli/cobra_ai.go:115`, `clients/cli/cobra_ai.go:120`) despite inheriting the global `--format` flag from the root command (`clients/cli/cobra_root.go:40`, `clients/cli/cobra_root.go:46`). A targeted search found no `_test.go` coverage for `newAICmd`, `ai-navigator`, `AINavigator`, or `fetchProvenance`.

   Impact: this command looks product-visible but behaves unlike the rest of the CLI and carries API/provider assumptions with no regression coverage. For an employer-facing portfolio project, that reads as experimental code living in the shipped command tree.

   Remediation: either mark the command experimental/hidden, or make provider/model/output behavior explicit and tested. Add table tests for request shaping, output shape, success detection, and provenance-fetch failure handling.

### Low

6. The `--auth-state` help text renders an accidental metavar.

   The flag usage string includes backticks around `stageflow auth capture` (`clients/cli/cobra_scan.go:133`, `clients/cli/cobra_scan.go:139`). pflag treats a back-quoted phrase in usage text as the displayed value name, so `go run . scan --help` renders `--auth-state stageflow auth capture` instead of a path-like metavar. This is easy to miss but noticeably unpolished in command help.

   Remediation: remove backticks from flag usage strings or set explicit annotations/metavars consistently. Add a help-output smoke assertion for important flags.

7. The API client package has no direct unit tests.

   `internal/apiclient` owns URL construction, API-key header injection, JSON request/response handling, and error-body propagation (`clients/cli/internal/apiclient/client.go:33`, `clients/cli/internal/apiclient/client.go:47`, `clients/cli/internal/apiclient/client.go:55`, `clients/cli/internal/apiclient/client.go:124`). `go test -count=1 ./internal/apiclient` reports `[no test files]`.

   Impact: command smoke tests exercise some happy paths, but API contract edge cases such as path resolution, auth headers, non-2xx bodies, malformed JSON, and delete/promote responses are not pinned close to the code that owns them.

   Remediation: add focused `httptest.Server` tests for `BuildURL`, `Do`, `GetJSON`, `SendJSON`, `DeleteJSON`, and project endpoint path escaping.

8. Remote project create/update accept unnormalized URL and scanner inputs while scan/project paths validate them locally.

   `project create` sends repeated `--url` and `--scanner` values directly to the API (`clients/cli/cobra_project_remote.go:62`, `clients/cli/cobra_project_remote.go:64`, `clients/cli/cobra_project_remote.go:48`). `project update` similarly sends changed URL/scanner slices directly (`clients/cli/cobra_project_update.go:31`, `clients/cli/cobra_project_update.go:36`, `clients/cli/cobra_project_update.go:48`). Direct scans and Project Mode use local normalization/parsing first (`clients/cli/cobra_scan.go:52`, `clients/cli/cobra_scan.go:57`, `clients/cli/project_run.go:125`, `clients/cli/project_run.go:130`).

   Impact: users get earlier, clearer errors in scan flows than in project-management flows, and API-side validation becomes part of normal CLI UX for these commands.

   Remediation: reuse `urlcheck.NormalizeTargets` and scanner parsing/validation before project create/update requests. Add tests for empty scanner names, bare hosts, unsupported schemes, and duplicate/replacement semantics.

## Post-Fix Status

- High finding 1 was addressed by routing live `diff` targets through the shared normalization and private-target validation path used by `scan`, with regression tests for local and remote API behavior.
- High finding 2 was addressed by adding a shared bounded API command context/client path and a hung-response regression test.
- Medium finding 3 was addressed by rejecting non-positive project readiness, stop, and scan timeouts during config/command handling.
- Medium finding 4 was addressed by binding the shared report quality-gate flags to local Project Mode and rendering through the shared report options.
- Medium finding 5 was addressed by hiding the experimental `ai` command from public help/README command listings.
- Low finding 6 was addressed by removing the accidental pflag metavar from `--auth-state` help.
- Low finding 7 was addressed with focused `internal/apiclient` tests for URL construction, API-key injection, JSON requests, error propagation, and project slug escaping.
- Low finding 8 was addressed by normalizing and validating remote project create/update URLs and scanner inputs before API submission.

## Strengths

- Command execution is testable through `run(args, getenv, stdout, stderr)` and consistently maps quality-gate failures to exit code 1 and command/API failures to exit code 2 (`clients/cli/run.go:12`, `clients/cli/report_flags.go:80`).
- Report rendering has meaningful coverage for text, Markdown, JSON, filtering, truncation, occurrence rendering, grouping, and fail-on behavior.
- Project Mode uses YAML `KnownFields(true)` for configuration parsing, which is a strong guard against misspelled config keys (`clients/cli/project_config.go:117`, `clients/cli/project_config.go:118`).
- Auth-state intake has size limits, JSON validation, base64 shaping, and mutually exclusive auth modes, which is a solid foundation for safe CLI-side handling (`clients/cli/auth_intake.go:47`, `clients/cli/auth_intake.go:70`, `clients/cli/cobra_scan.go:147`).

## Validation

- `go version`  
  Outcome: passed; `go version go1.26.3 linux/amd64`.

- `go test -count=1 ./...`  
  Outcome: passed. Packages tested include root CLI, `internal/apiclient`, `internal/diffrender`, `internal/jobstream`, `internal/manifesttmpl`, `internal/projectmode`, and `internal/urlcheck`.

- `go vet ./...`  
  Outcome: passed with no output.

- `go test -count=1 ./internal/apiclient`  
  Outcome: passed with focused API client tests.

- `go run . scan --help`  
  Outcome: exited 0; confirmed `--auth-state` now renders as a string/path-style flag instead of the malformed `stageflow auth capture` metavar.

## Documentation Checked

- Local README and repo map were inspected for command promises and module boundaries.
- Cobra and pflag behavior was checked against current package documentation/source for command execution, inherited flags, and pflag back-quoted usage metavars.

## Blockers and Uncertainty

- No live StageFlow API was exercised, so API compatibility findings are based on CLI request shaping and local tests, not end-to-end server behavior.
- The worktree was already dirty in `clients/cli/internal/apiclient/types.go` and `clients/cli/docs/`; those changes were not altered.

---

# Source: services/platform-api/audit.md

# Platform API Audit

Date: 2026-05-30 UTC

Scope: `services/platform-api` only, with read-only inspection of shared contracts/libs and downstream callers where needed to verify Platform API behavior. The repo was dirty at audit time; existing changes were treated as user-owned. This audit writes only this file.

## Findings

### High

1. Lifecycle subscription failure is non-fatal, so the API can start with broken realtime status/SSE.

   `cmd/server/main.go:114` builds the job-status event handler, but `cmd/server/main.go:115` to `cmd/server/main.go:117` only logs `SubscribeToStatusEvents` errors and continues. `cmd/server/main.go:119` then logs "Subscribed to lifecycle events" even after the warning path. Since `internal/messaging/service.go:40` to `internal/messaging/service.go:98` wires all NATS lifecycle subjects into the in-memory projection, a partial or total subscription failure leaves `GET /api/v1/jobs/{id}/stream` and cache-first status behavior stale or dependent on cold orchestrator reads. This is a ship-readiness risk because a production instance can look healthy while missing its live event feed.

   Remediation: fail startup when required lifecycle subscriptions cannot be established, or expose a degraded health/readiness state and disable SSE until subscriptions are active. Add a startup test that asserts subscription errors propagate instead of being swallowed.

2. Project scans are published before the project-job mapping is durable, and mapping failures are ignored.

   `internal/api/handlers_projects.go:326` publishes `job.created`; only after that, `internal/api/handlers_projects.go:340` records the project mapping. If `RecordProjectJob` fails, `internal/api/handlers_projects.go:341` to `internal/api/handlers_projects.go:342` only logs the error and still returns `201`. That mapping is later required by promotion and diff flows: `internal/api/handlers_projects.go:388` to `internal/api/handlers_projects.go:400` rejects promotion when the job is not associated with the project, and `internal/api/handlers_jobs_status.go:210` to `internal/api/handlers_jobs_status.go:220` rejects diffs for unmapped jobs.

   Remediation: make job creation and project-job mapping an atomic application operation from the API's point of view. A small scoped fix is to create the job ID, record a pending mapping before publish, delete/mark it failed if publish fails, and return an error if the mapping cannot be written. A more durable design is an outbox table for project scan requests.

3. ZIP uploads can leak staged objects when queueing fails after upload.

   `internal/api/handlers_jobs_zip_upload.go:319` uploads the ZIP to `scanner-staging`, then `internal/api/handlers_jobs_zip_upload.go:432` publishes `job.created`. If publish fails, `internal/api/handlers_jobs_zip_upload.go:432` to `internal/api/handlers_jobs_zip_upload.go:434` returns an error without deleting `req.zipPath`; the HTTP handler then returns `500` at `internal/api/handlers_jobs_zip_upload.go:127` to `internal/api/handlers_jobs_zip_upload.go:129`. The URL auth path already models the expected cleanup behavior: `internal/api/handlers_jobs_url_submit.go:209` to `internal/api/handlers_jobs_url_submit.go:210` deletes uploaded auth state on publish failure.

   Remediation: delete `scanner-staging` object `req.zipPath` on publish failure or request cancellation after upload. Add a ZIP-specific test equivalent to `TestHandleJobURLSubmitStorageStateAuthCleanupOnPublishFailure` in `internal/api/handlers_test.go:292`.

### Medium

4. Project CRUD does not enforce the URL intake contract, so saved projects can later queue empty or oversized scans.

   URL submissions enforce non-empty URLs, max 100 URLs, and max URL length in `internal/api/handlers_jobs_url_submit.go:233` to `internal/api/handlers_jobs_url_submit.go:267`. Project create only checks `len(req.URLs) == 0` at `internal/api/handlers_projects.go:119` to `internal/api/handlers_projects.go:123`, while update forwards `req.URLs` directly at `internal/api/handlers_projects.go:212` to `internal/api/handlers_projects.go:216`. The store treats an empty JSON array as a real update at `internal/project/store.go:156` to `internal/project/store.go:165`. `handleProjectScan` validates DNS/SSRF at `internal/api/handlers_projects.go:290`, but it does not re-run count/length/non-empty validation before publishing the saved URL list at `internal/api/handlers_projects.go:302` to `internal/api/handlers_projects.go:310`.

   Remediation: reuse `validateURLSubmitRequest` and target validation for project create/update, and re-check before project scan publish. Add handler tests for `PATCH {"urls":[]}`, too many URLs, overlong URLs, and invalid/private targets.

5. Browser CORS preflight blocks advertised project methods.

   The router supports `PATCH` and `DELETE` for `/api/v1/projects/{slug}` at `internal/api/handlers_projects.go:56` to `internal/api/handlers_projects.go:66`, and the README documents `GET/PATCH/DELETE` at `README.md:54`. The CORS middleware advertises only `GET, POST, OPTIONS` at `internal/api/middleware.go:291`, and `internal/api/middleware_test.go:39` locks in that incomplete header. Per MDN's current CORS guide, preflight uses `Access-Control-Request-Method` and the server's `Access-Control-Allow-Methods` to decide whether the actual method may be sent: https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/CORS.

   Remediation: include every browser-facing method registered by this API, at least `GET, POST, PATCH, DELETE, OPTIONS`, and update the CORS test to assert route coverage rather than the current incomplete string.

6. SSE can silently omit the initial status snapshot when response construction fails.

   `handleJobStream` calls `sendInitialStatus` at `internal/api/handlers_sse.go:399`. If `buildJobStatusResponse` fails, `sendInitialStatus` returns `true` without logging or writing an error at `internal/api/handlers_sse.go:202` to `internal/api/handlers_sse.go:205`, so the stream continues into `writeTerminalStatus` or `streamJobUpdates` at `internal/api/handlers_sse.go:403` to `internal/api/handlers_sse.go:407`. One concrete failure path is required artifact presigning for completed jobs: `internal/api/job_status_response.go:189` to `internal/api/job_status_response.go:191` returns an error.

   Remediation: log the build error and send a deterministic SSE `error` event or fail the request before streaming begins. Add a test where storage presign fails for a completed job and assert the first stream response is not an empty/ambiguous stream.

7. Job route parsing accepts unknown suffixes instead of returning 404.

   `handleJobStatus` switches on known suffixes at `internal/api/handlers_jobs_status.go:43` to `internal/api/handlers_jobs_status.go:57`, but an unknown suffix falls through and returns the base job status from `internal/api/handlers_jobs_status.go:60` onward. Similarly, `jobIDFromJobPath` accepts any path with `{jobID}/stream/...` because it checks only `len(parts) < 2` and `parts[1]` at `internal/api/handlers_sse.go:18` to `internal/api/handlers_sse.go:27`.

   Remediation: require exact route shapes for `/api/v1/jobs/{id}`, `/report`, `/results`, `/diff`, and `/stream`; return `404` for extra segments. Add table tests for typo and extra-segment paths.

8. Form-auth `login_url` is accepted with only a non-empty check at the Platform API boundary.

   Top-level scan URLs go through `validateTargetURLsWithResolver` at `internal/api/handlers_jobs_url_submit.go:115`, but form auth only checks `auth.form.login_url` is non-empty at `internal/api/handlers_jobs_url_submit.go:399` to `internal/api/handlers_jobs_url_submit.go:400`. Downstream scanner-runner later navigates to that value at `services/scanner-runner/src/core/auth-hydrator.ts:118` to `services/scanner-runner/src/core/auth-hydrator.ts:123`, where runtime validation may reject it. That turns a boundary validation issue into a queued job that fails later.

   Remediation: parse and validate `auth.form.login_url` with the same URL/SSRF policy used for scan targets, or require it to share an origin with one of the submitted URLs unless private-target mode is explicitly allowed. Add tests for bad scheme, metadata/private IP, and same-origin success.

### Low

9. Multipart field limits truncate silently instead of returning a clear validation error.

   `readFormValue` uses `io.LimitReader(part, limit)` at `internal/api/handlers_jobs_zip_upload.go:28` to `internal/api/handlers_jobs_zip_upload.go:34` and never checks whether more bytes were present. That affects `modules` at `internal/api/handlers_jobs_zip_upload.go:331` to `internal/api/handlers_jobs_zip_upload.go:340`, `scanner_configs` at `internal/api/handlers_jobs_zip_upload.go:346` to `internal/api/handlers_jobs_zip_upload.go:369`, `highlight_style` at `internal/api/handlers_jobs_zip_upload.go:374` to `internal/api/handlers_jobs_zip_upload.go:382`, and `screenshot` at `internal/api/handlers_jobs_zip_upload.go:387` to `internal/api/handlers_jobs_zip_upload.go:396`.

   Remediation: read `limit+1`, reject oversized fields with a structured `400` or `413`, and test boundary values for each multipart field.

## Strengths

- The URL submission path has meaningful boundary protections: body size, URL count/length, SSRF/DNS checks, module normalization, scanner config validation, and cleanup for uploaded auth state.
- Object key presigning is consistently job-scoped through `jobScopedKey` and `jobScopedJoin`, reducing artifact path traversal risk.
- The jobstatus reducer has useful terminal-state guards, cache-first behavior, watcher tests, and NATS integration coverage.
- SQLite opens with WAL, busy timeout, foreign keys, and single-connection mode, which is a reasonable default for this service's project store.

## Validation

Commands run from `/home/matt/Deployment/stageflow/services/platform-api`:

```bash
go test ./...
```

Outcome: passed. Packages included `cmd/server`, `internal/api`, `internal/jobstatus`, `internal/project`, `internal/status`, `internal/statussource`, and `tests/integration`; no-test packages reported as such.

```bash
go test -race ./internal/api ./internal/jobstatus ./internal/project ./tests/integration
```

Outcome: passed. The race check completed successfully for the highest-risk handler, event projection, project persistence, and NATS integration packages.

## Post-Fix Status

- High finding 1 was addressed by making lifecycle subscription setup fail startup when required status subscriptions cannot be established.
- High finding 2 was addressed by making project-job mapping durable before publish and cleaning it up when publish fails.
- High finding 3 was addressed by deleting staged ZIP objects on publish failure or request cancellation after upload.
- Medium finding 4 was addressed by reusing URL intake and target validation for project create/update/scan paths, including empty-list and scanner-module validation.
- Medium finding 5 was addressed by including `PATCH` and `DELETE` in CORS preflight methods.
- Medium finding 6 was addressed by sending a deterministic SSE error event when the initial status snapshot cannot be built.
- Medium finding 7 was addressed by requiring exact job route shapes and returning 404 for unknown suffixes/extra segments.
- Medium finding 8 was addressed by validating form-auth `login_url` through the same target URL policy used for scan targets.
- Low finding 9 was addressed by reading multipart form fields with a `limit+1` guard and returning structured validation errors for oversized fields.

## Suggested Next Work

1. Keep the targeted API handler regressions in the normal test path.
2. For release confidence, rerun `go test ./...` and the targeted race command above after future API changes. Also run the repo-level Go command documented in `README.md:136` to `README.md:143` once unrelated dirty changes are controlled.

---

# Source: services/orchestrator/audit.md

# StageFlow Orchestrator Audit

Scope: `services/orchestrator` only, reviewed from the current dirty worktree on 2026-05-30 UTC. Existing changes were treated as user-owned. This audit focuses on code quality, correctness, maintainability, job lifecycle/FSM/runtime/storage/event risks, test gaps, and ship-readiness for an open-source portfolio project.

## Findings

### High: terminal events can be lost after the database reaches `DONE` or `FAILED`

`CompleteJob` persists the terminal state before publishing `job.completed` (`services/orchestrator/internal/application/jobs/service.go:353`, `services/orchestrator/internal/application/jobs/service.go:365`). `FailJob` does the same for `FAILED` before publishing `job.failed` (`services/orchestrator/internal/application/jobs/service.go:391`, `services/orchestrator/internal/application/jobs/service.go:403`). If NATS publishing fails after the database write, the handler returns an error and JetStream can redeliver, but redelivery will not retry the terminal publish: `scan.completed` is ignored once the job is terminal (`services/orchestrator/internal/application/jobs/service.go:274`), and `FailJob` returns immediately for terminal jobs (`services/orchestrator/internal/application/jobs/service.go:385`). The result is a job that looks final in Postgres but never emits the terminal event downstream.

NATS JetStream is explicitly an at-least-once system with redelivery on missing acknowledgements, so terminal side effects need their own idempotency boundary. See the official NATS JetStream docs on [consumers](https://docs.nats.io/nats-concepts/jetstream/consumers) and [JetStream publish acknowledgements](https://docs.nats.io/nats-concepts/jetstream).

Remediation: add a small transactional outbox table for `job.completed`/`job.failed` with a unique key such as `(job_id, event)`, write the outbox row in the same DB transaction/state transition that marks the job terminal, and run a retrying publisher that marks rows published only after NATS publish succeeds. Keep handler redelivery idempotent by consulting that outbox rather than trying to republish from terminal event handlers.

### High: duplicate or concurrent scanner terminal events can publish duplicate `job.completed`

The repository treats a duplicate scanner completion as successful and returns `allComplete=true` when all expected scanners are already present (`services/orchestrator/internal/adapters/repository/job_scanners.go:112`, `services/orchestrator/internal/adapters/repository/job_scanners.go:116`, `services/orchestrator/internal/adapters/repository/job_scanners.go:121`). The service completes the job whenever `allComplete` is true (`services/orchestrator/internal/application/jobs/service.go:296`, `services/orchestrator/internal/application/jobs/service.go:309`). `CompleteJob` also accepts an already-`COMPLETING` job without acquiring exclusive ownership of completion work (`services/orchestrator/internal/application/jobs/service.go:317`, `services/orchestrator/internal/application/jobs/service.go:327`) and publishes after `CompleteJob` returns (`services/orchestrator/internal/application/jobs/service.go:365`). The DB `CompleteJob` helper returns nil when the row is already terminal (`services/orchestrator/internal/adapters/repository/job_updates.go:203`), so a second handler can still aggregate, clean up, and publish a second `job.completed`.

Remediation: make entry into `COMPLETING` an ownership claim. For example, add a repository method that atomically updates `SCANNING -> COMPLETING` and returns `claimed bool`; only the claimant aggregates, cleans up, and writes/publishes the terminal event. Combine this with the outbox uniqueness above so duplicate completions become harmless no-ops.

### High: inbound event payloads are decoded but not contract-validated before side effects

The shared subscriber path strictly decodes the envelope payload but does not call payload `Validate()` (`libs/go/messaging/subscribe.go:189`, `libs/go/messaging/subscribe.go:195`, `libs/go/messaging/subscribe.go:234`). Orchestrator handlers then pass payloads straight into lifecycle logic, e.g. `job.created` calls `CreateJob` without validation (`services/orchestrator/internal/application/jobs/handle_job_created.go:10`, `services/orchestrator/internal/application/jobs/handle_job_created.go:12`). `CreateJob` treats only `urls` specially and sends every other input type through extraction (`services/orchestrator/internal/application/jobs/startup_lifecycle.go:71`, `services/orchestrator/internal/application/jobs/startup_lifecycle.go:75`). The event contracts already define required fields and valid input types (`libs/go/events/types.go:44`, `libs/go/events/types.go:53`, `libs/go/events/types.go:71`), but the orchestrator does not enforce them at its trust boundary.

This can persist malformed jobs, start extraction with an empty input path, or accept incomplete extraction/scan payloads before downstream failures make the root cause harder to diagnose.

Remediation: call `payload.Validate()` at the first orchestrator-owned handler boundary for every inbound event, before logging/auditing side effects where feasible. Add tests that malformed NATS-decoded payloads are rejected without creating jobs, pods, DB state transitions, or storage writes.

### Medium: setup failures can leave runtime resources or jobs in limbo

Extraction creates/persists pod state before moving the job to `EXTRACTING` and starting the worker (`services/orchestrator/internal/application/jobs/startup_lifecycle.go:181`, `services/orchestrator/internal/application/jobs/startup_lifecycle.go:188`, `services/orchestrator/internal/application/jobs/startup_lifecycle.go:199`). `ensureJobPod` creates the pod, then separately persists the pod ID (`services/orchestrator/internal/application/jobs/startup_lifecycle.go:211`, `services/orchestrator/internal/application/jobs/startup_lifecycle.go:216`). If pod creation succeeds but `UpdateJobPodID` fails, the DB has no pod ID for cleanup. If a later setup step fails, `startExtraction` returns an error without compensating cleanup or marking a setup failure (`services/orchestrator/internal/application/jobs/startup_lifecycle.go:188`, `services/orchestrator/internal/application/jobs/startup_lifecycle.go:194`, `services/orchestrator/internal/application/jobs/startup_lifecycle.go:199`).

Remediation: add compensating cleanup when runtime setup succeeds but persistence or worker launch fails. Prefer an explicit setup state/transaction boundary or a helper that records the pod ID before any operation that can create more resources, then calls `FailJob` plus `CleanupJob` on unrecoverable setup failures.

### Medium: rejected or unknown events are not durably auditable

Inbound audit rows are inserted after every handler attempt (`services/orchestrator/internal/orchestrator/event_trace.go:137`, `services/orchestrator/internal/orchestrator/event_trace.go:161`), but `job_events.job_id` has a foreign key to `jobs(id)` (`services/orchestrator/internal/adapters/repository/schema.sql:69`, `services/orchestrator/internal/adapters/repository/schema.sql:89`). If an event references a missing job, or if `job.created` fails before inserting the job, the audit insert fails and is only logged (`services/orchestrator/internal/orchestrator/event_trace.go:97`, `services/orchestrator/internal/orchestrator/event_trace.go:103`). That removes exactly the evidence needed to debug bad producers, out-of-order delivery, or validation failures.

Remediation: introduce an `event_inbox`/`event_audit` table that can store rejected events without a job FK, or make the FK nullable with a separate `payload_job_id` field. Keep the existing per-job event view by joining where a job row exists.

### Medium: `extraction.failed` missing-job handling is inconsistent with other inbound events

`extraction.ready`, scan completion, scan failure, and page-progress handlers explicitly ignore missing jobs in several paths, but `HandleExtractionFailed` attempts to update artifacts and then calls `FailJob` unconditionally (`services/orchestrator/internal/application/jobs/handle_extraction_failed.go:28`, `services/orchestrator/internal/application/jobs/handle_extraction_failed.go:43`). `FailJob` immediately returns an error when the job cannot be loaded (`services/orchestrator/internal/application/jobs/service.go:380`, `services/orchestrator/internal/application/jobs/service.go:382`). A stale or unknown `extraction.failed` event therefore redelivers until MaxDeliver instead of being classified consistently.

Remediation: apply the same missing-job policy used by the other event handlers, or route unknown failures to the event audit/dead-letter path above with a terminal "ignored/rejected" status.

### Low: API pagination and state filters silently accept bad input

`parsePaginationParams` ignores invalid `limit` and `offset` values and falls back to defaults (`services/orchestrator/internal/api/server.go:50`, `services/orchestrator/internal/api/server.go:62`). The jobs endpoint accepts any `state` string and passes it into the repository filter (`services/orchestrator/internal/api/server.go:242`, `services/orchestrator/internal/api/server.go:250`). This is not a runtime correctness issue, but for a portfolio-quality admin API it makes client mistakes harder to detect.

Remediation: return `400` for invalid pagination and reject unknown job states using the shared FSM/state constants.

## Post-Fix Status

- High terminal-event loss and duplicate-completion findings were addressed with an idempotent terminal-event outbox plus an atomic completion ownership claim.
- High inbound-validation finding was addressed by validating orchestrator-owned inbound event payloads before lifecycle side effects.
- Medium setup-compensation finding was addressed by cleaning up runtime resources and failing jobs when extraction setup fails after pod creation.
- Medium rejected/unknown-event audit finding was addressed by allowing nullable `job_events.job_id`, storing the original payload job ID separately, retrying inserts without a job FK on missing-job failures, and adding a legacy-schema migration guard that drops `job_events.job_id NOT NULL`.
- Medium `extraction.failed` missing-job handling was addressed by applying the same missing-job ignore policy used by the other inbound event handlers.
- Low API parameter validation was addressed by returning `400` for invalid pagination and unknown job state filters.

## Strengths

- The service is well-layered: domain policies, application use cases, infrastructure adapters, orchestration glue, API, and metrics are separated cleanly.
- The repository has meaningful transition guards and concurrency-aware scanner-result updates (`services/orchestrator/internal/adapters/repository/job_updates.go:57`, `services/orchestrator/internal/adapters/repository/job_scanners.go:62`).
- Runtime launch planning is centralized and testable, including auth/env handling and resource limits (`services/orchestrator/internal/application/jobs/scanner_launch_planner.go:104`).
- Config validation catches required infrastructure settings and dangerous empty admin tokens before startup (`services/orchestrator/cmd/orchestrator/config.go:92`).
- The Dockerfile uses a multi-stage build and non-root runtime user (`services/orchestrator/Dockerfile:3`, `services/orchestrator/Dockerfile:44`).
- The test surface is strong for a portfolio project: repository, runtime client, storage aggregation, domain policies, API, orchestrator flows, and e2e-style mocked flows are all represented.

## Test Gaps To Close

- Keep regression coverage around terminal publish retry/outbox behavior, duplicate scanner events, inbound validation before side effects, setup compensation after pod creation, and missing-job event audit insertion.
- For deployed databases created before nullable `job_events.job_id`, keep the schema migration test that proves startup drops the legacy `NOT NULL` constraint.

## Validation Run

- `go list ./...` from `services/orchestrator`: passed and listed all orchestrator packages.
- `go test ./...` from `services/orchestrator`: passed. Output included all packages under `cmd`, `internal`, and `test`; live Podman contract tests are build-tagged and were not included without `-tags podmanlive`.

## Ship-Readiness Summary

The orchestrator already presents as a serious, non-toy service: clean boundaries, substantial tests, explicit config, metrics, and operational audit intent. The main ship-readiness risk is not style; it is event lifecycle semantics around terminal state, publication, duplicate delivery, and rejected-event observability. Fixing the terminal outbox/idempotency boundary and adding validation-before-side-effects would materially improve correctness and make the repo much easier for employers to trust.

---

# Source: services/archive-extractor/audit.md

# Archive Extractor Audit

Date: 2026-05-30 UTC

Scope: `services/archive-extractor` Go source, tests, Dockerfile, testdata, and local docs. Existing worktree changes were treated as user-owned; this audit only writes this file.

## Findings

### High: ZIP objects are fully copied to local temp storage before any archive size policy is enforced

`Extractor.Extract` creates an unbounded temp ZIP (`services/archive-extractor/internal/extractor/extractor.go:41`), opens the MinIO object (`services/archive-extractor/internal/extractor/extractor.go:57`), copies the whole object with `io.Copy` (`services/archive-extractor/internal/extractor/extractor.go:68`), and only then runs `validateZIP` (`services/archive-extractor/internal/extractor/extractor.go:73`). The validation code caps entry count, declared uncompressed bytes, expansion ratio, and actual extracted bytes, but it does not prevent an oversized compressed object, malformed object, or very large non-ZIP object from filling `/tmp` or consuming substantial network/disk I/O before rejection.

Why it matters: for an employer-facing open-source project, archive ingestion should fail early and predictably. The current behavior makes the extractor vulnerable to resource exhaustion before the strongest safety checks execute.

Remediation: enforce a compressed-object byte limit before or during copy. Prefer carrying object size from storage stat into the extractor and rejecting over-limit inputs before reading; also wrap the copy with `io.LimitReader(maxCompressedBytes+1)` and fail if the limit is exceeded. Add focused tests around the copy boundary with an oversized reader/object, plus a documented config/default for the maximum accepted ZIP upload size.

### Medium: extraction into `destDir` is not atomic and does not clean stale or partial output

`mustExtractZIP` always extracts into `filepath.Join(cfg.Workspace, "site")` (`services/archive-extractor/cmd/server/main.go:134`). `extractZIPWithLimits` creates that destination if needed (`services/archive-extractor/internal/extractor/extractor.go:214`) but never clears existing contents before extraction. During extraction, files are opened with `O_TRUNC` (`services/archive-extractor/internal/extractor/extractor.go:330`) and errors return immediately (`services/archive-extractor/internal/extractor/extractor.go:221`), while aggregate size failures are detected only after an entry has been written (`services/archive-extractor/internal/extractor/extractor.go:226`). `extractFile` can also leave an over-limit file on disk because it writes `maxEntryBytes+1` bytes and then returns an error (`services/archive-extractor/internal/extractor/extractor.go:345`, `services/archive-extractor/internal/extractor/extractor.go:351`).

Why it matters: a retry or reused workspace can serve/discover stale HTML that was not in the current upload, and failed extraction attempts can leave partial payloads in the shared workspace. The current tests use fresh temp destinations, so they do not cover this idempotency and cleanup risk.

Remediation: extract into a fresh temp directory under the workspace, validate all extraction limits there, and atomically replace `site` only after success. On any extraction failure, remove the temp directory. Add tests proving stale files disappear on success and no partial files remain on validation or extraction failure.

### Medium: the static server exposes the whole extracted tree, not just discovered pages

The server wraps `http.FileServer(http.Dir(filepath.Clean(s.siteDir)))` (`services/archive-extractor/internal/server/server.go:61`) and forwards all non-OPTIONS requests to it (`services/archive-extractor/internal/server/server.go:64`, `services/archive-extractor/internal/server/server.go:75`). Discovery only selects `.html` and `.htm` files for provenance (`services/archive-extractor/internal/discovery/discovery.go:50`, `services/archive-extractor/internal/discovery/discovery.go:60`), but the HTTP server will serve any extracted file under `siteDir`, including assets, hidden files, and directories. Go's `http.Dir` documentation warns that it can expose sensitive files and directories, including dotfiles, when serving arbitrary directories.

Why it matters: even though the server binds loopback by default, every process sharing the pod/network namespace can request files outside provenance. Uploaded archives commonly include source maps, config files, build metadata, hidden files, or accidental secrets. Directory responses can also reveal structure that provenance intentionally omits.

Remediation: decide the intended contract explicitly. If scanners need assets, keep asset serving but block dotfiles, reject directory listings, and consider serving only paths under discovered HTML plus same-origin asset references. If scanners only need discovered pages, make the handler enforce provenance paths. Add tests for dotfile denial, directory listing denial, and serving a non-HTML asset only if that remains intended.

### Medium: ZIP path normalization allows ambiguous entries that should be rejected for reproducibility

`sanitizeZipEntryName` replaces backslashes and then calls `path.Clean` (`services/archive-extractor/internal/extractor/extractor.go:167`, `services/archive-extractor/internal/extractor/extractor.go:182`). It rejects only cleaned paths that are `..` or begin with `../` (`services/archive-extractor/internal/extractor/extractor.go:187`). As a result, entries like `a/../index.html` are accepted and rewritten to `index.html`. There is also no duplicate-path tracking in the validation loop (`services/archive-extractor/internal/extractor/extractor.go:127`), while extraction truncates existing output paths (`services/archive-extractor/internal/extractor/extractor.go:330`).

Why it matters: the extractor can silently collapse multiple archive names onto the same destination path. That makes provenance and served content dependent on ZIP entry order and makes tests less representative of adversarial archives.

Remediation: reject any `.` or `..` path segment before cleaning, and track sanitized destination names to reject duplicates and file/directory collisions. Add table-driven tests for `a/../b.html`, duplicate filenames, duplicate-after-normalization names, and backslash variants.

### Medium: the main job lifecycle has no focused unit tests

The core orchestration path in `runExtraction` wires NATS, MinIO, extraction, discovery, provenance upload, server startup, ready publication, and shutdown (`services/archive-extractor/cmd/server/main.go:42`). Failure reporting is centralized in `publishFailureAndExit` (`services/archive-extractor/cmd/server/main.go:470`), and stage artifacts are finalized in `stage_logger.go` (`services/archive-extractor/cmd/server/stage_logger.go:212`). `go test ./...` reports `cmd/server` as `[no test files]`, so this behavior is currently covered only indirectly by package-level unit tests and local integration tests that bypass MinIO/NATS.

Why it matters: event payloads and stage artifacts are the contract between extractor, orchestrator, and scanner runner. Regressions in ready/failed publication, stage log paths, or shutdown behavior would not be caught by the narrow unit suite.

Remediation: introduce test seams around storage, messaging, stage logging, and server startup, then add focused tests for success payload contents, failure payload contents, stage finalization behavior, and invalid config/port handling. Keep these as unit tests with fakes; reserve full MinIO/NATS coverage for a separate integration profile.

### Low: `PORT` is used in provenance before listener startup validates it

`loadConfig` accepts `PORT` as a raw string (`services/archive-extractor/cmd/server/main.go:427`), `Validate` does not check it (`services/archive-extractor/cmd/server/main.go:381`), and provenance is generated with `"http://localhost:" + cfg.Port` before the server starts (`services/archive-extractor/cmd/server/main.go:212`, `services/archive-extractor/cmd/server/main.go:215`). The listener will later reject invalid addresses (`services/archive-extractor/internal/server/server.go:87`), but local provenance and uploaded provenance may already have been written (`services/archive-extractor/cmd/server/main.go:234`, `services/archive-extractor/cmd/server/main.go:271`).

Remediation: parse and validate `PORT` during config validation, or build provenance from `siteServer.ListenerAddr()` after a successful bind. This would also make tests more representative of the runtime address actually in use.

### Low: `localhost` provenance does not match the server's explicit IPv4 bind

The server default binds `127.0.0.1:<port>` (`services/archive-extractor/internal/server/server.go:39`, `services/archive-extractor/internal/server/server.go:41`), while production provenance uses `http://localhost:<port>` (`services/archive-extractor/cmd/server/main.go:212`). The integration test avoids this mismatch by building `baseURL` from `ListenerAddr()` (`services/archive-extractor/integration_test.go:64`, `services/archive-extractor/integration_test.go:68`).

Remediation: use the bound listener address for provenance, or bind consistently to the same hostname that provenance publishes. Add a runtime-style test that starts the server and fetches every generated provenance URL.

### Low: server hardening is minimal beyond `ReadHeaderTimeout`

The HTTP server sets `ReadHeaderTimeout` (`services/archive-extractor/internal/server/server.go:81`) but leaves write, read, and idle timeouts unset. Because this service is intended to be loopback-only and short-lived, this is not a current blocker, but it is a ship-readiness gap if the server is ever exposed beyond same-pod scanner traffic.

Remediation: keep the loopback-only design documented and enforced, or add explicit `ReadTimeout`, `WriteTimeout`, and `IdleTimeout` values that fit scanner behavior. Add a regression test or config assertion if non-loopback binding is intentionally unsupported.

## Post-Fix Status

- The high compressed-ZIP resource-exhaustion finding was addressed with a 100 MiB compressed object policy, metadata pre-check when object size is available, and a hard `max+1` streaming copy guard.
- Medium extraction idempotency finding was addressed by extracting into a fresh temp directory and replacing the destination only after successful validation/extraction.
- Medium static-server exposure finding was addressed by denying dot-path segments and directory listings while still serving extracted assets needed by scanners.
- Medium ZIP path ambiguity finding was addressed by rejecting ambiguous `.`/`..` segments, duplicate normalized entries, and file/directory collisions.
- Low port/provenance findings were addressed by validating `PORT`, starting the static server before provenance generation, and building provenance URLs from the bound listener address.
- Low server hardening was addressed by adding read, write, and idle timeouts appropriate for the short-lived loopback server.
- The main `cmd/server` lifecycle still deserves broader fake-based unit coverage around ready/failed event payloads and stage artifact finalization.

## Strengths

- The extractor has meaningful ZIP defenses: entry count, name length, NUL rejection, absolute path rejection, Windows drive rejection, traversal rejection, per-entry declared size, aggregate declared size, compression-ratio check, destination containment, and actual-byte extraction limits (`services/archive-extractor/internal/extractor/extractor.go:19`, `services/archive-extractor/internal/extractor/extractor.go:121`, `services/archive-extractor/internal/extractor/extractor.go:140`, `services/archive-extractor/internal/extractor/extractor.go:152`, `services/archive-extractor/internal/extractor/extractor.go:255`, `services/archive-extractor/internal/extractor/extractor.go:345`).
- Discovery produces deterministic provenance ordering and ignores symlinks (`services/archive-extractor/internal/discovery/discovery.go:37`, `services/archive-extractor/internal/discovery/discovery.go:77`, `services/archive-extractor/internal/discovery/discovery.go:82`).
- Provenance writes use a temp file and rename rather than direct overwrite (`services/archive-extractor/internal/provenance/provenance.go:72`, `services/archive-extractor/internal/provenance/provenance.go:95`).
- The runtime image uses a multi-stage Dockerfile and a non-root runtime user (`services/archive-extractor/Dockerfile:3`, `services/archive-extractor/Dockerfile:27`, `services/archive-extractor/Dockerfile:29`, `services/archive-extractor/Dockerfile:31`).
- The current unit and integration tests cover many happy-path, traversal, discovery, provenance, and local server behaviors (`services/archive-extractor/internal/extractor/validate_zip_test.go:29`, `services/archive-extractor/internal/discovery/discovery_test.go:274`, `services/archive-extractor/internal/provenance/generator_generate_test.go:10`, `services/archive-extractor/internal/server/server_test.go:28`, `services/archive-extractor/integration_test.go:25`).

## Validation

Commands run from `/home/matt/Deployment/stageflow/services/archive-extractor`:

- `GOWORK=off go test ./...`
  - Outcome: passed.
  - Output summary: archive extractor packages passed.
- `GOWORK=off go test -tags=integration ./...`
  - Outcome: passed.
  - Output summary: build-tagged integration package passed, and package tests passed.
- `GOWORK=off go vet ./...`
  - Outcome: passed with no output.

Additional inspection commands:

- `unzip -l services/archive-extractor/testdata/path-traversal.zip`
  - Confirmed fixture contains `../../etc/evil.html` and `normal.html`.
- `unzip -l services/archive-extractor/testdata/absolute-path.zip`
  - Confirmed fixture contains `/etc/passwd.html` and `normal.html`.
- `unzip -l services/archive-extractor/testdata/nested-site.zip`
  - Confirmed fixture contains `page.html`, `subdir/`, and `subdir/deep.html`.

Recommended validation before shipping this section:

- `GOWORK=off go test ./...`
- `GOWORK=off go test -tags=integration ./...`
- `GOWORK=off go vet ./...`
- `docker build -f services/archive-extractor/Dockerfile .` from the repository root, if Docker/BuildKit is available.
- Add the remediation tests listed above, especially oversized compressed download, stale destination cleanup, duplicate normalized ZIP names, dotfile/directory-listing server behavior, and full `cmd/server` ready/failed event payloads.

## External Documentation Checked

- Go `archive/zip` documentation: `OpenReader` returns entries with central-directory metadata such as `CompressedSize64` and `UncompressedSize64`, and current docs note future insecure-path behavior changes: https://pkg.go.dev/archive/zip
- Go `net/http` documentation: `FileServer` serves contents rooted at the provided filesystem, and `http.Dir` warns that serving arbitrary directories can expose sensitive files, directories, and dotfiles: https://pkg.go.dev/net/http

## Finding Counts

- Critical: 0
- High: 1
- Medium: 4
- Low: 3

## Blockers And Uncertainty

- I did not inspect or change orchestrator/scanner-runner behavior beyond archive-extractor-local docs because this audit scope was limited to `services/archive-extractor`.
- The severity of full-tree serving depends on whether scanners intentionally need arbitrary assets. The current implementation and tests do not encode that contract, so the audit treats it as a medium ship-readiness risk rather than a confirmed vulnerability.
- Docker image build was not run; the Go test/vet checks were run locally and passed.

---

# Source: services/scanner-runner/audit.md

# Scanner Runner Audit

Scope: `services/scanner-runner` only, audited on 2026-05-30 UTC. Existing worktree changes were treated as user-owned. This audit is code-review oriented: findings first, ordered by severity, with concrete remediation scoped to this package.

## Findings

### High: Authenticated Lighthouse scans do not use the hydrated browser session

`PageIterator` hydrates authenticated state into the Playwright context for `storage_state` and `form` auth (`services/scanner-runner/src/core/page-iterator.ts:146`, `services/scanner-runner/src/core/page-iterator.ts:171`, `services/scanner-runner/src/core/page-iterator.ts:176`), then passes the authenticated `page` to scanners (`services/scanner-runner/src/core/page-iterator.ts:445`). `LighthouseScanner.scanPage`, however, calls `runLighthouse(page, pageEntry.url)` (`services/scanner-runner/src/scanners/lighthouse/index.ts:85`), and `runLighthouse` ignores the Playwright page parameter (`services/scanner-runner/src/scanners/lighthouse/index.ts:218`). It launches/uses a separate Chrome process through `ensureChrome()` and passes only the page URL plus the remote debugging port to Lighthouse (`services/scanner-runner/src/scanners/lighthouse/index.ts:225`, `services/scanner-runner/src/scanners/lighthouse/index.ts:229`, `services/scanner-runner/src/scanners/lighthouse/chrome-lifecycle.ts:56`). The Lighthouse invocation also explicitly allows storage reset (`services/scanner-runner/src/scanners/lighthouse/lighthouse-invoker.ts:42`).

Impact: for authenticated jobs, Axe/SEO/link-style scanners operate on the hydrated Playwright context, but Lighthouse audits a fresh unauthenticated Chrome session. Gated pages can be measured as login redirects or anonymous content, producing misleading portfolio-critical results. The local real-browser integration test proves the missing bridge indirectly: its gated Lighthouse helper has to pass a cookie header manually (`services/scanner-runner/tests/integration/auth-scan-real-browser.test.ts:512`, `services/scanner-runner/tests/integration/auth-scan-real-browser.test.ts:551`), while the production scanner has no equivalent.

Remediation: make Lighthouse consume authenticated state. Options include running Lighthouse against the already-authenticated Chrome/CDP session, exporting context cookies/storage state into a Lighthouse-compatible session before the run, or passing validated auth headers/cookies deliberately for same-origin targets. Add a production `LighthouseScanner` integration test for both `form` and `storage_state` auth that fails if the audited DOM is the login page.

### High: `security-headers` bypasses the target-validation pipeline for its out-of-band fetch

The runtime target policy is built once per provenance (`services/scanner-runner/src/core/page-iterator.ts:133`) and included in each `ScanContext` (`services/scanner-runner/src/core/types.ts:276`, `services/scanner-runner/src/core/types.ts:283`, `services/scanner-runner/src/core/page-iterator.ts:452`). Browser navigation and subresources are validated through `BrowserManager.navigateToPage` and routing (`services/scanner-runner/src/core/browser-manager.ts:147`, `services/scanner-runner/src/core/browser-manager.ts:197`, `services/scanner-runner/src/core/browser-manager.ts:208`), and link checking manually validates every redirect hop (`services/scanner-runner/src/scanners/link-checker/validation.ts:31`, `services/scanner-runner/src/scanners/link-checker/validation.ts:57`).

`SecurityHeadersScanner` does not use that policy. It picks `page.url()` or `pageEntry.url`, then calls `page.request.fetch(targetURL, { method: 'GET', timeout: 30_000 })` directly (`services/scanner-runner/src/scanners/security-headers/index.ts:96`, `services/scanner-runner/src/scanners/security-headers/index.ts:102`). Playwright's API request context follows redirects automatically by default, so this request can take a different redirect path than the already-validated page navigation without rechecking blocked/private targets.

Impact: this weakens the scanner-runner's SSRF/blocked-network story specifically in the security scanner. Even if initial navigation was protected, the scanner's second HTTP client can follow redirects outside the validated path.

Remediation: route `security-headers` through the same validation helper used by `link-checker`: validate the initial URL, perform the fetch with `maxRedirects: 0`/manual redirects, validate every `Location` hop, and preserve browser-context cookies where needed. Add a regression test for a public URL redirecting to `http://169.254.169.254/latest/meta-data`.

### Medium: Plugin validation constructs scanner instances twice

`loadPluginFromManifest` resolves a factory and then calls `validateFactory(factory)` (`services/scanner-runner/src/core/plugins/plugin-load.ts:60`, `services/scanner-runner/src/core/plugins/plugin-load.ts:61`). `validateFactory` instantiates the scanner only to check for `scanPage` (`services/scanner-runner/src/core/plugins/plugin-load.ts:131`, `services/scanner-runner/src/core/plugins/plugin-load.ts:133`). Later, `getScanner` calls the same factory again for the scanner that actually runs (`services/scanner-runner/src/worker.ts:57`, `services/scanner-runner/src/worker.ts:68`).

Impact: third-party plugin constructors run twice during normal startup. That is surprising plugin behavior and can duplicate side effects such as temp directory creation, telemetry setup, expensive model initialization, or open handles. Tests cover invalid factories and happy-path loading, but not constructor call count (`services/scanner-runner/tests/core/plugins/plugin-load.test.ts:291`).

Remediation: instantiate once and return the validated instance from the load path, or validate the module shape without executing constructors. Add a plugin-loader regression test with a constructor counter to lock the lifecycle contract.

### Medium: Artifact upload can silently produce reports that reference missing screenshots

The unified report formatter records screenshot artifact paths when issues or page overviews include screenshots (`services/scanner-runner/src/core/web-server-formatter.ts:64`, `services/scanner-runner/src/core/web-server-formatter.ts:141`). During upload, `ScannerBase.uploadArtifacts` uploads `results.json` and `report.html` as required artifacts, but screenshot directory upload errors are logged and swallowed, including non-filesystem storage/provider failures (`services/scanner-runner/src/core/scanner-base.ts:476`, `services/scanner-runner/src/core/scanner-base.ts:497`, `services/scanner-runner/src/core/scanner-base.ts:500`). Extra artifact upload failures are also warnings (`services/scanner-runner/src/core/scanner-base.ts:581`, `services/scanner-runner/src/core/scanner-base.ts:589`, `services/scanner-runner/src/core/scanner-base.ts:606`).

Impact: a scan can publish `scan.completed` with successful results while referenced screenshot or trace artifacts are absent from object storage. For a portfolio-grade scanner, broken artifact links make reports look unreliable even when the core scan succeeded.

Remediation: distinguish optional screenshot capture from required artifact publication. Either fail/mark partial when object-storage upload fails after artifact references have been written, or remove/degrade those artifact references and surface a report-level warning before publishing completion.

### Medium: The riskiest runtime paths are excluded from coverage or only gated

Coverage explicitly excludes `src/ai/**`, `src/screenshots/**`, `src/worker.ts`, `src/index.ts`, and `src/lib.ts` (`services/scanner-runner/vitest.config.ts:13`, `services/scanner-runner/vitest.config.ts:21`). The Lighthouse unit test file states that actual Lighthouse integration requires browser/CDP and is not unit-tested (`services/scanner-runner/tests/scanners/lighthouse/index.test.ts:4`). The real Lighthouse authenticated test is gated behind `E2E_LIGHTHOUSE=1` (`services/scanner-runner/tests/integration/auth-scan-real-browser.test.ts:811`).

Impact: the codebase has many good focused tests, but the portfolio-critical runtime edges named in this audit are exactly where coverage is weakest: worker orchestration, screenshots/artifacts, AI navigation, and production Lighthouse/CDP behavior.

Remediation: keep fast unit tests as default, but add a small smoke suite that runs in CI on the production scanner entry points with mocked MinIO/NATS and one local Playwright/Lighthouse fixture. Do not rely only on helper functions that duplicate production logic.

### Low: Invalid environment values are silently replaced with defaults

`getEnvBool`, `getEnvNumber`, and `getEnvInt` return defaults when inputs are malformed (`services/scanner-runner/src/core/config-loader.ts:125`, `services/scanner-runner/src/core/config-loader.ts:140`, `services/scanner-runner/src/core/config-loader.ts:149`). `validateConfig` catches only a few resulting fields such as `concurrency` and `maxRetries` (`services/scanner-runner/src/core/config-loader.ts:198`, `services/scanner-runner/src/core/config-loader.ts:202`).

Impact: typos like `BROWSER_HEADLESS=treu`, `PAGE_LOAD_TIMEOUT=abc`, or an unintended viewport value become silent operational defaults. That is friendly during development but poor for a job-runner where reproducibility matters.

Remediation: fail fast, or at least log structured warnings, for malformed explicit env values. Keep defaults only for missing values.

## Post-Fix Status

- High finding 1 was addressed by carrying hydrated Playwright browser context state into Lighthouse's Chrome/CDP session and preserving state during authenticated audits.
- High finding 2 was addressed by validating the initial security-headers URL and every redirect hop with manual redirect handling.
- Medium plugin lifecycle finding was addressed by validating and retaining a single scanner instance during plugin load.
- Medium artifact upload finding was addressed by propagating screenshot and extra artifact upload failures instead of publishing reports that reference missing artifacts.
- Low malformed-env finding was addressed by failing fast on malformed explicit boolean, number, and integer environment values.
- The broader runtime coverage hardening finding remains follow-up work; the dead-code gate is now blocking and passes locally.

## Strengths

- The package has strong TypeScript/ESLint settings and a useful strict CI script (`services/scanner-runner/eslint.config.mjs:40`, `services/scanner-runner/package.json:34`).
- Browser target validation is thoughtfully implemented for normal navigation, redirects, subresources, and static localhost exceptions (`services/scanner-runner/src/core/browser-manager.ts:147`, `services/scanner-runner/src/core/target-validation.ts:210`, `services/scanner-runner/tests/core/browser-manager.test.ts:379`).
- Auth redaction and real-browser auth flows have meaningful tests, which is a strong signal for employers reviewing correctness beyond happy paths (`services/scanner-runner/tests/integration/auth-scan-real-browser.test.ts:658`, `services/scanner-runner/tests/integration/auth-scan-real-browser.test.ts:827`).
- Built-in manifest drift is guarded against the shared scanner catalog (`services/scanner-runner/tests/core/plugins/builtin-manifests.test.ts:23`).

## Validation Run

- `bun run type-check`
  - Outcome: passed, exit 0.
- `bun run vitest run tests/scanners/lighthouse/index.test.ts tests/core/plugins/plugin-load.test.ts tests/core/plugins/plugin-discovery.test.ts tests/core/browser-manager.test.ts tests/core/target-validation.test.ts`
  - Outcome: passed, 5 test files, 136 tests.
- `bun run vitest run tests/core/page-iterator.test.ts tests/core/scanner-base.test.ts tests/scanners/security-headers/index.test.ts tests/scanners/link-checker/scanPage.test.ts tests/scanners/link-checker/index.test.ts tests/worker/worker-validation.test.ts`
  - Outcome: passed, 6 test files, 97 tests.
- `bun run find-dead-code`
  - Outcome: passed.
- `bun run analyze:dead`
  - Outcome: passed.
- `bun run ci`
  - Outcome: passed, including format, strict lint, type-check, and coverage. Coverage ran 62 test files with 673 passing tests and 1 skipped test.

Not run: Docker build, live MinIO/NATS worker run, or `E2E_LIGHTHOUSE=1` real Lighthouse integration.

## External Docs Checked

- Playwright `APIRequestContext` docs: `page.request` shares browser-context cookies, and `fetch` follows redirects automatically by default with `maxRedirects` defaulting to 20. <https://playwright.dev/docs/api/class-apirequestcontext>
- Playwright `page.route` docs: routing applies to page network requests and has redirect/service-worker caveats. <https://playwright.dev/docs/api/class-page#page-route>
- Lighthouse programmatic docs: Node usage launches Chrome separately and uses `port`; for authenticated sites, the documented workflow logs into the Chrome instance that Lighthouse targets. <https://github.com/GoogleChrome/lighthouse/blob/main/docs/readme.md>

## Finding Counts

- Critical: 0
- High: 2
- Medium: 3
- Low: 1

---

# Source: libs/audit.md

# Shared Libraries and Contracts Audit

Date: 2026-05-30 UTC

Scope: `libs/contracts` and `libs/go`. Existing worktree changes were treated as user-owned. This is a pre-fix audit for shared boundaries used by clients and services.

## Findings

### High

1. Event contracts drift across JSON Schema, Go structs, and TypeScript publishers.

   `scan.page.completed` schema requires only `job_id`, `page_id`, `page_index`, and `total_pages`, and allows `page_index` and `total_pages` minimum `0` (`libs/contracts/events/schema/scan.page.completed.schema.json:13`, `libs/contracts/events/schema/scan.page.completed.schema.json:17`). The Go payload includes optional `scanner_type`, documents `page_index` as 1-based, and rejects values below `1` (`libs/go/events/types.go:136`, `libs/go/events/types.go:140`, `libs/go/events/types.go:157`). Scanner Runner publishes `scanner_type` and passes the index directly (`services/scanner-runner/src/core/event-publisher.ts:96`, `services/scanner-runner/src/core/event-publisher.ts:99`, `services/scanner-runner/src/core/event-publisher.ts:101`), and its contract test expects the extra key (`services/scanner-runner/tests/core/event-publisher.contract.test.ts:47`).

   `scan.completed` schema requires `timing` (`libs/contracts/events/schema/scan.completed.schema.json:13`, `libs/contracts/events/schema/scan.completed.schema.json:20`), but Go marks timing `omitempty` and validates it only when present (`libs/go/events/types.go:176`, `libs/go/events/types.go:185`, `libs/go/events/types.go:213`). The Go test explicitly verifies timing is omitted when nil (`libs/go/events/types_test.go:197`, `libs/go/events/types_test.go:230`).

   Impact: the README and architecture docs present contracts as schema-first, but event producers/consumers can disagree on required fields and valid values. That weakens the trust boundary between scanner-runner, orchestrator, and platform-api.

   Remediation: pick one event contract source of truth and align schemas, fixtures, Go validation, and TypeScript publisher tests. Add a check that serializes representative Go/TS payloads and validates them against the schema.

### Medium

2. Event schema validation is a hand-rolled subset, not full JSON Schema validation.

   `libs/contracts/events/schema/validate.mjs` implements only `const`, scalar/object `type`, `minLength`, `minimum`, `required`, and recursive `properties` checks (`libs/contracts/events/schema/validate.mjs:21`, `libs/contracts/events/schema/validate.mjs:29`, `libs/contracts/events/schema/validate.mjs:54`, `libs/contracts/events/schema/validate.mjs:58`, `libs/contracts/events/schema/validate.mjs:62`, `libs/contracts/events/schema/validate.mjs:70`). Other contract families use Ajv with format support (`libs/contracts/report/schema/validate.js:11`, `libs/contracts/provenance/schema/validate.js:10`, `libs/contracts/scanner-manifest/schema/validate.js:10`).

   Impact: the event validator can pass fixtures that a real JSON Schema validator would reject once the schema grows to arrays, `additionalProperties`, `format`, `oneOf`, or other keywords.

   Remediation: replace the custom event validator with Ajv or clearly constrain event schemas to the supported subset and test that constraint.

3. Shared Go env helpers silently fall back on malformed explicit values.

   `GetEnvInt`, `GetEnvBool`, and `GetEnvDuration` return defaults when an environment variable is present but malformed (`libs/go/config/env.go:19`, `libs/go/config/env.go:30`, `libs/go/config/env.go:46`). That keeps startup resilient but makes typo-driven configuration drift hard to detect.

   Impact: services using these helpers can run with defaults after operators explicitly set invalid values.

   Remediation: add strict variants or structured warnings for malformed explicit values, then migrate infrastructure-critical settings to strict parsing.

4. Scanner override typos are silently ignored.

   `ApplyOverrides` skips unknown scanner IDs without warning or error (`libs/go/scannerregistry/config.go:164`, `libs/go/scannerregistry/config.go:174`, `libs/go/scannerregistry/config.go:176`). This is lenient for future/custom scanners, but a typo in deployment YAML can look successful while changing nothing.

   Impact: scanner enablement/image/resource overrides can silently fail in deployment configuration.

   Remediation: return or log unknown override IDs, with an explicit escape hatch if custom scanner IDs are intentionally allowed later.

## Post-Fix Status

- High finding 1 was addressed by aligning the event JSON Schemas, fixtures, and Go validation for `scan.page.completed` and `scan.completed`.
- Medium finding 2 was addressed by replacing the hand-rolled event schema validator with Ajv 2020 plus format support and wiring fixture validation into CI.
- Medium finding 3 was addressed by adding strict Go env helpers and migrating infrastructure-critical config loaders to fail fast on malformed explicit values.
- Medium finding 4 was addressed by adding checked scanner override application and making Platform API/Orchestrator startup fail on unknown scanner override IDs.

## Strengths

- The report, provenance, and scanner-manifest contract families have schemas, fixtures, generated types, and runtime validators.
- Go shared packages have meaningful focused tests for events, models, scanner catalog/registry, provenance, storage, diffing, logging, and response helpers.
- Scanner manifests are embedded and validated before registry initialization, which gives services a stable scanner catalog.
- The report contract includes additional integrity checks beyond structural schema validation in TypeScript.

## Validation

- `node libs/contracts/events/schema/validate.mjs`: passed.
- `cd libs/contracts/report && bun run validate:fixtures`: passed.
- `cd libs/contracts/provenance && bun install --frozen-lockfile && bun run validate:fixtures`: passed. Local install was needed so Node did not resolve Ajv packages through `/home/matt/node_modules`, which points at another project on this machine.
- `cd libs/contracts/scanner-manifest && bun install && bun run validate:fixtures && go test ./...`: passed. The temporary Bun lockfile generated by the install was removed afterward.
- Go tests for every `libs/go/*` module: passed.

## Finding Counts

- Critical: 0
- High: 1
- Medium: 3
- Low: 0
