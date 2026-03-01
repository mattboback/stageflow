# StageFlow Optional CLI Mode — Implementation Plan

## Summary

Add an optional CLI client so StageFlow can be used from the shell in addition to the existing web/demo workflow. This is an additive secondary mode, not a product redefinition.

The initial CLI MVP must be fully truthful and implementable against the current architecture:

- The existing web frontend remains the primary experience.
- The existing `POST /api/v1/jobs/urls` endpoint remains the canonical submission path.
- Public URL scanning behavior stays unchanged by default.
- `stageflow.org` must keep its current public-safe behavior.
- Phase 1 private-target support is preparatory API-side validation work only.
- End-to-end scanning of a developer workstation's `localhost` is available only in local/self-hosted mode (see Phase 4); it must not be implied for hosted/public deployments.

## Goals

- Add a thin `stageflow` CLI that submits jobs, waits for completion, and formats results for shell/agent use.
- Make the CLI usable against both local and hosted StageFlow APIs for existing API-compatible URL scans.
- Lay the groundwork for future local-target workflows without weakening the default public SSRF posture.

## Non-Goals

- Embedding the orchestrator, NATS, MinIO, or Podman into the CLI.
- Replacing the frontend or changing the core StageFlow positioning.
- Adding a new orchestrator path or a new API subsystem for CLI mode.
- Shipping end-to-end developer-workstation localhost scanning in the initial version.
- Adding config files, watch mode, diff mode, daemon/proxy mode, artifact mode, or MCP wrapping in the initial version.

---

## Product Boundary

- StageFlow remains a web accessibility + quality scanning platform.
- The CLI is a convenience client for the existing API, not a new platform component.
- The live demo and public deployments must not change behavior unless explicitly configured.
- The initial CLI MVP is public-URL-first and must not imply a localhost transport capability that does not exist yet.

---

## Public API Changes

### 1. URL Submission Request

**File:** `platform/api/internal/api/handlers_jobs_url_submit.go`

Extend the existing `POST /api/v1/jobs/urls` request body with one new optional field:

```json
{
  "urls": ["http://localhost:3000"],
  "modules": ["axe"],
  "screenshot": false,
  "allow_private_targets": true
}
```

Rules:

- `allow_private_targets` is optional.
- Default is `false`.
- If omitted or `false`, behavior is exactly the same as today.
- If `true`, the API may allow loopback and broader private targets, but only when the server is also configured to permit them.
- This request flag changes only API-side validation acceptance. It does not guarantee scanner-side reachability to the caller's workstation.

### 2. New API Environment Variable

**API env var:** `PLATFORM_API_ALLOW_PRIVATE_TARGETS`

Rules:

- Default is `false`.
- It is only read by the API service.
- Public/shared deployments, including `stageflow.org`, must leave it unset or `false`.
- Private/local target acceptance is only available when this is `true`.

---

## Security / SSRF Design

### Current Problem to Avoid

Do **not** implement a blanket SSRF bypass. A global early return that disables validation for all URL submissions would be too broad and would be easy to misconfigure on shared stacks.

### Required Design

Refactor URL validation so private-target allowance is a scoped validation mode, not an SSRF off-switch.

**Files:**

- `platform/api/internal/api/security.go`
- `platform/api/internal/api/handlers_jobs_url_submit.go`

Implementation rules:

- Always keep URL parsing and scheme validation.
- Always keep host presence validation.
- Even in private-target mode, still reject:
  - invalid or unparseable URLs
  - non-HTTP(S) schemes
  - metadata IP `169.254.169.254`
  - link-local addresses
  - unspecified addresses
  - multicast addresses
- Only relax the "public targets only" restriction when **both** are true:
  1. request body sets `allow_private_targets=true`
  2. API process has `PLATFORM_API_ALLOW_PRIVATE_TARGETS=true`

When private-target mode is enabled for a request, allow the broader local/dev targets needed for future local workflows:

- `localhost`
- `127.0.0.0/8`
- `::1`
- RFC1918 IPv4 ranges (`10/8`, `172.16/12`, `192.168/16`)
- hostnames that resolve to those addresses

Keep blocking all other currently blocked reserved ranges, including:

- IPv6 unique local ranges
- metadata and other reserved non-public ranges already blocked today

Important implementation note:

- The active handler path currently calls `validateTargetURLsWithResolver(...)` directly.
- The validation mode must be applied in the code path used by `handleJobURLSubmit`, not only in a wrapper that is not used there.

### Important Limitation

This phase changes API acceptance rules only.

- It does **not** change scanner container networking.
- URL scans still execute inside orchestrator-managed containers/pods.
- In the current architecture, `localhost` resolves in the scanner runtime's network namespace, not automatically on the CLI caller's machine.
- Therefore, enabling broader private-target validation in the API is groundwork only and does not by itself deliver end-to-end workstation localhost or private-network scanning.

A later phase must add one of:

- a daemon/proxy bridge
- explicit host-network or host-mapping support
- artifact-mode scanning via ZIP/static hosting

### Failure Behavior

If `allow_private_targets=true` is requested but the API is not configured to allow it:

- Return `400 Bad Request`
- Use a structured validation-style error response
- Message should clearly state that this API instance does not permit private target scans

This keeps the endpoint stable while making the restriction explicit to future CLI users.

---

## Deployment Scoping

To ensure the live demo is unaffected:

- Do **not** enable `PLATFORM_API_ALLOW_PRIVATE_TARGETS` in:
  - `infra/compose/podman-compose.yml`
  - production/staging Quadlet templates
  - any public deployment defaults
- If local compose wiring is added, scope it to:
  - `infra/compose/podman-compose.local.yml`

Local compose behavior:

- Expose the env var only in the local-only compose override, if needed.
- Default it to `false` unless explicitly set by the developer.

This keeps private/local target support available for local experimentation without silently changing the default stack behavior.

---

## CLI Tool

### Location and Module

- **Location:** `tools/stageflow-cli/`
- **Module:** `github.com/mattboback/stageflow/tools/stageflow-cli`
- **Binary name:** `stageflow`

Add to `go.work`:

```txt
use ./tools/stageflow-cli
```

### Dependencies

- `packages/contracts/report/generated/go` for `UnifiedReportV2`
- Standard library HTTP + SSE handling
- No NATS, Podman, MinIO, or orchestrator-specific imports

### Day-One Command Scope

Initial commands:

- `stageflow run`
- `stageflow report`
- `stageflow scanners`

Deferred:

- localhost transport bridge / daemon mode (for hosted/remote callers)
- artifact mode for deterministic local scans
- `summarize`
- `watch`
- `diff`
- MCP wrapper
- release automation

---

## CLI Command Contract

### `stageflow run`

Submit a scan job, wait for completion, fetch results, and print an agent-friendly result.

```txt
stageflow run [flags]

Flags:
  --url <url>                Target URL to scan (required, repeatable)
  --scanners <list>          Comma-separated scanner modules (default: axe)
  --screenshot               Capture screenshots (default: false)
  --api <url>                API base URL (default: $STAGEFLOW_API_URL or http://localhost:8080)
  --api-key <key>            API key (default: $STAGEFLOW_API_KEY, omitted if empty)
  --timeout <duration>       Max wait time (default: 5m)
  --format <fmt>             Output format: summary, json, quiet (default: summary)
  --max <n>                  Max issues to output (default: 0 = unlimited)
  --severity <level>         Minimum severity to include: critical, serious, moderate, minor, info (default: minor)
  --threshold-critical <n>   Fail if critical issues exceed N
  --threshold-serious <n>    Fail if serious issues exceed N
  --threshold-total <n>      Fail if total issues exceed N
  --no-stream                Poll instead of SSE (fallback for restricted environments)
```

Behavior:

1. `POST /api/v1/jobs/urls`
2. Wait on `GET /api/v1/jobs/{id}/stream` unless `--no-stream` is set
3. Print progress/state updates to `stderr` only
4. Fetch final status from `GET /api/v1/jobs/{id}`
5. Fetch the aggregated report JSON from `GET /api/v1/jobs/{id}/results`
6. Decode `UnifiedReportV2`
7. Evaluate thresholds against the full, unfiltered report summary and full issue set
8. Filter issues for display by severity and `--max`
9. Print formatted output to `stdout`

### `stageflow report`

Fetch and display results for an existing job ID.

```txt
stageflow report [flags] <job-id>

Flags:
  --api
  --api-key
  --format
  --max
  --severity
```

Behavior:

- `GET /api/v1/jobs/{id}`
- Require terminal `DONE`
- Fetch report via `GET /api/v1/jobs/{id}/results`
- Reuse the same filtering and formatting as `run`
- Threshold evaluation remains based on the full report, not display filters

### `stageflow scanners`

List available scanners from the existing API.

```txt
stageflow scanners [flags]

Flags:
  --api
  --api-key
  --format <summary|json>
```

Behavior:

- `GET /api/v1/scanners`
- Default to a compact summary list
- `--format json` prints machine-friendly structured output

---

## Results Fetching Contract

Use `GET /api/v1/jobs/{id}/results` as the canonical way for the CLI to fetch the aggregated report JSON.

Rationale:

- It is an existing stable API path.
- It already redirects to the presigned artifact URL.
- It avoids depending on top-level artifact field names in the status payload.

Do **not** design the CLI around `artifacts.results_json` in the top-level job status payload.

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Scan completed and thresholds passed (or no thresholds were set) |
| 1 | Scan completed but one or more thresholds failed |
| 2 | Scan failed, timed out, returned an invalid state, or results could not be fetched/parsed |

---

## Output Contract

### `summary` (default)

Human- and agent-readable text including:

- job id
- URL(s)
- duration
- score and grade when present
- severity totals
- filtered issue list
- threshold outcome when thresholds are set

### `json`

Machine-oriented JSON wrapper generated by the CLI, not a raw dump of `UnifiedReportV2`.

Required top-level fields:

```json
{
  "job_id": "abc123",
  "state": "DONE",
  "urls": ["https://example.com"],
  "duration_ms": 47000,
  "score": 72,
  "score_grade": "C",
  "summary": {
    "total_issues": 27,
    "by_severity": {
      "critical": 3,
      "serious": 7,
      "moderate": 12,
      "minor": 5
    },
    "by_scanner": {
      "axe": 20,
      "lighthouse": 7
    },
    "pages_scanned": 3
  },
  "issues": [],
  "threshold_result": "fail",
  "threshold_detail": "critical: 3 > 0",
  "errors": []
}
```

Rules:

- This wrapper is generated by the CLI.
- It is not a raw dump of the report schema.
- `threshold_result` and exit codes are derived from the full report, not from filtered display output.

### `quiet`

Single-line pass/fail intended for automation:

```txt
PASS: critical=0 serious=0 total=0
FAIL: critical: 3 > 0
```

---

## File Structure

```txt
tools/stageflow-cli/
├── main.go
├── run.go
├── cmd_run.go
├── cmd_report.go
├── cmd_scanners.go
├── client.go
├── sse.go
├── output.go
├── filter.go
├── threshold.go
├── types.go
├── constants.go
├── go.mod
└── go.sum (only if needed)
```

Reuse patterns from:

- `tools/suite-runner/`
- `tools/job-status-cli/`

Do not import those tools directly as libraries. Copy/adapt their patterns into the new module.

---

## Current Status & Audit (March 2026)

**Status:**
- **Phase 1 (Safe API Groundwork)** has been fully implemented (via a `ralph` plan). `security.go` and `handlers_jobs_url_submit.go` correctly enforce the private-target validation mode. The config plumbing for `PLATFORM_API_ALLOW_PRIVATE_TARGETS` is complete.
- **Phase 2 (CLI MVP)** has been implemented. The `tools/stageflow-cli/` module includes `run`, `report`, and `scanners`, an SSE wait path with `--no-stream` polling fallback, `summary`/`json`/`quiet` output formats, threshold evaluation, and unit tests.

**Potential Issues & Ongoing Audit:**
- **Network Reachability:** The Phase 1 groundwork allows API validation for loopback/RFC1918 addresses, and Phase 4 adds a local-only workstation `localhost` path via host-network job pods. This is not appropriate for shared/hosted deployments, and should remain clearly documented as local-only.
- **Results Fetching:** We need to keep strictly to the planned `GET /api/v1/jobs/{id}/results` path for fetching results in the CLI, rather than depending on internal artifact paths in the job struct, which could break in the future as internal architectures shift.

---

## Implementation Order

### Phase 1 — Safe API Groundwork (COMPLETE)

1. [x] Add `allow_private_targets` to the URL submission request struct.
2. [x] Introduce scoped private-target validation mode in `security.go`.
3. [x] Gate that mode on both request flag and `PLATFORM_API_ALLOW_PRIVATE_TARGETS=true`.
4. [x] Add/update API tests for public, private, loopback, and metadata-target scenarios.
5. [x] Keep this phase explicitly limited to API acceptance rules; do not treat it as end-to-end localhost or private-network scanning support.

### Phase 2 — CLI MVP (COMPLETE)

1. [x] Scaffold `tools/stageflow-cli/` and add it to `go.work`.
2. [x] Implement shared CLI entrypoint and flag parsing.
3. [x] Implement HTTP client with optional `X-Api-Key`.
4. [x] Implement SSE wait path and `--no-stream` polling fallback.
5. [x] Implement `run`.
6. [x] Implement `report`.
7. [x] Implement `scanners`.
8. [x] Implement `summary`, `json`, and `quiet` formatters.
9. [x] Add unit tests for filtering, thresholds, output, and client behavior.

### Phase 3 — Docs / Local Wiring (COMPLETE)

1. [x] Update `README.md` with an "optional CLI mode" section.
2. [x] Surface local-only env vars in `infra/compose/podman-compose.local.yml`.
3. [x] Leave public/shared deployment defaults unchanged.
4. [x] Document Phase 4 local project mode and limitations.

### Phase 4 — Local Project Mode / Workstation `localhost` (EXPERIMENTAL)

This phase is a local-only extension on top of the CLI MVP so agents can run dynamic regression scans against a dev server on the same machine.

- CLI `stageflow run` project mode:
  - If `--url` is omitted, the CLI resolves a project root (git root from PWD by default) and loads `.stageflow/config.yaml`.
  - It runs `dev.up` steps, starts `dev.start.cmd`, waits for `dev.ready.url`, then submits a scan against `scan.urls`.
  - It supports `--out` for writing results to a file in addition to stdout.
  - It supports `--allow-private-targets` (and config `scan.allow_private_targets`) for localhost/private scans on local stacks.
- Orchestrator local-only enablement:
  - `POD_NETNS_MODE=host` runs job pods in the host network namespace so scanners can reach `http://localhost:<port>` on the same machine.
  - Scanner/extractor containers use `127.0.0.1` endpoints for NATS/MinIO when running in host netns.
- Local compose wiring:
  - `infra/compose/podman-compose.local.yml` enables `PLATFORM_API_ALLOW_PRIVATE_TARGETS=true` and `POD_NETNS_MODE=host` for local use.

Safety notes:

- Do not enable `PLATFORM_API_ALLOW_PRIVATE_TARGETS` or `POD_NETNS_MODE=host` on shared/public deployments.
- Project mode executes commands from `.stageflow/config.yaml`; treat it as trusted input.
- On macOS/Windows with Podman VM, `POD_NETNS_MODE=host` refers to the VM, not your host machine.

---

## Test Cases and Scenarios

### API Tests

Add or update tests for:

1. Public URL submission still succeeds unchanged.
2. Loopback and RFC1918 targets without `allow_private_targets` still fail.
3. Loopback and RFC1918 targets with `allow_private_targets=true` and env disabled return `400`.
4. `http://localhost:3000` succeeds at the API validation layer when both request flag and env var are enabled.
5. `127.0.0.1` and `::1` succeed at the API validation layer when both request flag and env var are enabled.
6. RFC1918 targets succeed at the API validation layer when both request flag and env var are enabled.
7. Hostnames that resolve to public addresses and the explicitly allowed private/local addresses succeed in private-target mode.
8. `http://169.254.169.254` still fails even in private-target mode.
9. Invalid schemes like `file://...` still fail in all modes.
10. Existing scanner/module validation behavior is unchanged.

### CLI Tests

Add tests for:

1. `run` flag parsing and env defaults
2. public URL success path
3. exit `0` on success with passing thresholds
4. exit `1` on threshold failure
5. exit `2` on timeout
6. exit `2` on submit/fetch/parsing failures
7. `report` returns exit `2` when the job is not `DONE`
8. `scanners` summary formatting
9. `json` output shape
10. severity and `--max` filtering
11. severity and `--max` filtering do not affect threshold outcome

### Manual Verification

CLI MVP:

1. Run a local or hosted StageFlow API instance.
2. Run:

```bash
stageflow run \
  --api http://localhost:8080 \
  --url https://example.com \
  --scanners axe \
  --format summary
```

3. Confirm:
   - submission succeeds
   - progress is printed to `stderr`
   - final report is printed to `stdout`

API groundwork only:

1. Run API tests with `PLATFORM_API_ALLOW_PRIVATE_TARGETS=true` in a controlled local environment.
2. Confirm loopback and RFC1918 targets are accepted by the validator only when both the request flag and server toggle are enabled.
3. Confirm this validation success is treated as API groundwork only, not as proof of end-to-end workstation localhost or broader private-network reachability.

Public-safe mode:

1. Run against an API instance without the env var enabled.
2. Submit a public URL; confirm success.
3. Submit `http://localhost:3000` with `allow_private_targets=true`; confirm a clear `400` rejection.

---

## Verification

After implementation, run:

```bash
just ci
```

Useful targeted checks during development:

```bash
(cd platform/api && go test ./...)
(cd tools/stageflow-cli && go test ./...)
```

`justfile` changes are not required unless implementation reveals a concrete gap, because `just ci` already iterates Go modules from `go.work`.

---

## Documentation Changes

Update `README.md` with a short "Optional CLI mode" section that states:

- StageFlow still supports the current web/API workflow.
- The CLI is an additional interface.
- Public scans work the same way as today.
- Private/local target validation support is opt-in API groundwork only.
- End-to-end localhost or broader private-network scanning requires a later transport or artifact-mode phase.
- `stageflow.org` keeps the default public-safe behavior.

Do not rewrite the project narrative or replace the demo-first positioning.

---

## Explicit Assumptions and Defaults

- The CLI is a convenience client, not a new service.
- The existing API endpoint remains canonical.
- The initial version includes only `run`, `report`, and `scanners`.
- Binary name is `stageflow`.
- Module directory is `tools/stageflow-cli/`.
- `PLATFORM_API_ALLOW_PRIVATE_TARGETS` defaults to `false`.
- `summary` is the default output format.
- `GET /api/v1/jobs/{id}/results` is the canonical aggregated report fetch path for the CLI.
- The CLI MVP supports existing API-compatible URL scans.
- Phase 1 private-target work allows loopback plus RFC1918 IPv4 targets at the API validation layer.
- End-to-end workstation localhost scanning is not part of the MVP.
- `--allow-private-targets` is deferred from the CLI until a later transport solution or artifact-mode path exists.
- Thresholds are computed from the full report, not filtered output.
- `stageflow.org` and public deployments keep the current default behavior.
