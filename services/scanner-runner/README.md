# Scanner Runner (`scanner-runner`)

This service is the **Scanner Runner** runtime that runs *inside per-job pods*.

It is responsible for:
- Loading job provenance (the list of pages/URLs to scan) from the shared job workspace.
- Resolving a built-in scanner module (Axe, Lighthouse, SEO, etc.) via the shared manifest catalog.
- Writing scanner artifacts (`results.json`, `report.html`, and any per-page assets like screenshots) into the worker results directory.
- Uploading artifacts to object storage and publishing scan lifecycle events over NATS.

It is **not** responsible for:
- Accepting job submissions (handled by Platform API).
- Pod/container orchestration (handled by the Orchestrator service).
- Aggregating multi-scanner output into a job-level report (handled by the Orchestrator).

## Runtime security notes

For URL jobs, the worker validates browser network requests when `SCAN_URLS` is set. Initial targets, redirects, final navigated URLs, and HTTP(S) subresources are checked against the same blocked-network policy. `ALLOW_PRIVATE_TARGETS=true` is reserved for local/private scanning and still does not allow link-local metadata targets.

Chromium currently launches with `--no-sandbox`, `--disable-setuid-sandbox`, and `chromiumSandbox: false`. This is acceptable only inside the intended StageFlow job boundary: rootless Podman job pods, `no-new-privileges`, per-job containers, constrained resources, and host/container egress controls for hosted or self-hosted public deployments. Enabling Chromium sandboxing in the runtime image is a future hardening task; do not treat the browser process as the primary isolation boundary.

## Scanner resolution

On startup the worker loads the built-in manifest catalog and resolves the scanner selected by `SCANNER_TYPE`.

- IDs and aliases: resolution is case-insensitive for aliases; IDs are expected to be lowercase/hyphenated
- Built-in manifests live in `libs/go/scannercatalog/manifests/*/manifest.json`.
- `bun run build` copies those manifests into `dist/scanners` for runtime images.

## Contract types

`bun run prepare:contracts` regenerates TypeScript contracts and writes local package shims under `services/scanner-runner/node_modules/@stageflow/*`. This is intentional: the source tree typechecks against generated contract files before the publishable contract packages have been built into `dist`.

## Outputs

The worker writes into `RESULTS_DIR` (default: `${SCANNER_DATA_DIR}/results`):

- Note: the scanner identifier used in artifact keys is `scanner.metadata.name` at runtime; for built-ins it matches the manifest `id`.
- `results.json` (always; web-server results format, version `2.0.0`)
- `report.html` (best-effort; may be missing if report generation fails)
- `<pageId>/...` (per-page directories created during scanning)
  - `<pageId>/screenshots/*` (only when a scanner captures screenshots)
  - scanner-specific per-page files (only when implemented)

Uploads (MinIO / artifacts bucket):

- `${JOB_ID}/${scannerId}/results.json`
- `${JOB_ID}/${scannerId}/report.html` (only if it exists)
- `${JOB_ID}/${scannerId}/${pageId}/screenshots/*` (only if `<pageId>/screenshots/` exists)

Some scanners upload additional per-page files (for example `ai-navigator` uploads `${JOB_ID}/ai-navigator/${pageId}/ai-trace.json`).

## SCANNER_OPTIONS validation

If a manifest provides `configSchema`, the worker validates `SCANNER_OPTIONS` against it.

- If `SCANNER_OPTIONS` is missing, it is treated as `{}` for validation (so scanners with optional config can run).
- In `NODE_ENV=production` a schema mismatch fails the run; in non-production it logs a warning.
