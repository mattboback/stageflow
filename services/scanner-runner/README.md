# Scan Worker (`scanner-runner`)

This service is the **scan worker runtime** that runs *inside per-job pods*.

It is responsible for:
- Loading job provenance (the list of pages/URLs to scan) from the shared job workspace.
- Resolving a scanner module (Axe, Lighthouse, SEO, etc.) via a filesystem plugin + manifest model.
- Writing scanner artifacts (`results.json`, `report.html`, and any per-page assets like screenshots) into the worker results directory.
- Uploading artifacts to object storage and publishing scan lifecycle events over NATS.

It is **not** responsible for:
- Accepting job submissions (handled by Platform API).
- Pod/container orchestration (handled by the Orchestrator service).
- Aggregating multi-scanner output into a job-level report (handled by the Orchestrator).

## Scanner resolution (plugin loader)

On startup the worker discovers plugin manifests and then loads the scanner selected by `SCANNER_TYPE`.

- Manifest filenames: `manifest.json` or `scanner.json`
- IDs and aliases: resolution is case-insensitive for aliases; IDs are expected to be lowercase/hyphenated
- Default search paths:
  - `dist/scanners` (built-ins; populated during `bun run build` via `scripts/copy-builtin-manifests.mjs`)
  - `/plugins` (volume-mounted plugins)
  - `${HOME}/.stageflow/plugins` (dev convenience)
  - plus `PLUGIN_PATHS` (colon-separated)

## Outputs

The worker writes into `RESULTS_DIR` (default: `${SCANNER_DATA_DIR}/results`):

- Note: the scanner identifier used in artifact keys is `scanner.metadata.name` at runtime; for built-ins it matches the manifest `id`. For third-party plugins, keep them the same.
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
