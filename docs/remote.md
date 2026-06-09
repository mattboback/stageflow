# Remote Projects & Regression Memory

StageFlow is a **platform with a thin CLI client**, not just a local scanner.
Every scan runs on a StageFlow API — your own local stack or the hosted service
at `stageflow.org` — and the CLI talks to it over plain HTTP, streaming live
progress back over Server-Sent Events. The same binary that scans a one-off URL
also manages long-lived **projects** with promoted **baselines**, so a scan can
answer the question that actually matters in CI: *did this change make the
frontend worse?*

This guide covers the remote/platform workflow. For the local dev-server loop
(`stageflow project`), see [PROJECT_MODE.md](PROJECT_MODE.md).

## The mental model

| Concept | What it is |
| --- | --- |
| **API** | A running StageFlow Platform API. Local (`http://localhost:8080`) or hosted (`https://stageflow.org`). |
| **Project** | A named record on the API holding target URLs, a scanner selection, and a pointer to a promoted baseline job. |
| **Scan** | A job the API runs against a project's URLs. Streamed live; ends with a unified report. |
| **Baseline** | A completed job promoted as the reference point. Later scans are diffed against it. |
| **Regression** | New issues (or a worse score) relative to the baseline. Surfaces as a non-zero exit code. |

The CLI is stateless: it reads the API base URL and key from flags or
environment and never holds project state locally.

## Pointing the CLI at an API

Every command accepts a persistent `--api` flag (and `--api-key` when the API
requires auth). Both have environment fallbacks:

```bash
export STAGEFLOW_API_URL="https://stageflow.org"
export STAGEFLOW_API_KEY="<your-key>"     # only if the API enforces auth

# ...or per-command:
stageflow project list --api https://stageflow.org --api-key "$KEY"
```

`--api` defaults to `http://localhost:8080`, so against a local stack you can
omit it entirely.

## The lifecycle

### 1. Create a project

```bash
stageflow project create marketing-site \
  --name "Marketing Site" \
  --url https://example.com \
  --url https://example.com/pricing \
  --scanner axe --scanner seo --scanner link-checker
```

`--url` is repeatable. `--scanner` is repeatable; omit it to run all enabled
scanners. `--name` defaults to the slug.

### 2. Run a project scan

```bash
stageflow scan --project marketing-site
```

This submits a job built from the project's stored URLs and scanners, streams
progress over SSE, and — once a baseline exists — fetches the diff. With
`--format json` the CLI emits a single envelope containing the project metadata,
a pass/fail decision (`passed`, `severityFailed`, `regressed`), the full report,
and the diff:

```bash
stageflow scan --project marketing-site --format json
```

### 3. Promote a baseline

Pick a known-good job and make it the reference:

```bash
stageflow project promote marketing-site --job-id <job-id>
```

Every subsequent `scan --project` is now diffed against it. Re-promote whenever
you want to accept a new baseline.

### 4. Inspect and maintain

```bash
stageflow project list                 # slug, name, baseline, URL count
stageflow project show marketing-site  # full record
stageflow project update marketing-site --scanner axe --scanner seo  # replaces scanners
stageflow project delete marketing-site
```

`update` flags replace the corresponding field wholesale: `--url` replaces all
URLs, `--scanner` replaces all scanners.

## From a repository: `stageflow project hosted`

If a repo has a `.stageflow/config.yaml` that declares a `remote_project` slug,
`stageflow project hosted` runs that hosted project **without** starting a local
dev server, then reports the regression diff:

```yaml
# .stageflow/config.yaml
stageflow:
  remote_project: marketing-site
  remote_api_url: https://stageflow.org   # optional; falls back to api_url
```

```bash
stageflow project hosted --format json
```

This is the bridge between local Project Mode and the hosted platform: the local
loop (`stageflow project`) gives you fast validation against your dev server,
while `project hosted` asks the hosted API for baseline and regression memory.
See [PROJECT_MODE.md](PROJECT_MODE.md) for the config reference.

## Gating CI on regressions

Exit codes are machine-readable, so a project scan is a one-line gate:

| Exit code | Meaning |
| --- | --- |
| `0` | Scan completed; no displayed issue met `--fail-on` and no regression |
| `1` | A displayed issue met the severity threshold, or the scan regressed from baseline |
| `2` | CLI or API error |

```yaml
# .github/workflows/frontend-quality.yml
- name: StageFlow regression gate
  env:
    STAGEFLOW_API_URL: https://stageflow.org
    STAGEFLOW_API_KEY: ${{ secrets.STAGEFLOW_API_KEY }}
  run: stageflow scan --project marketing-site --fail-on serious
```

The job fails if the deploy introduced new serious-or-worse issues, or if the
report regressed against the promoted baseline.

## Targets, auth, and boundaries

- Private/loopback targets (`localhost`, `127.0.0.1`, RFC1918) are rejected
  against a non-loopback API unless the API explicitly runs in private-target
  mode. Configure auth on the remote project rather than passing `--auth-state`
  to `scan --project`.
- When the API enforces authentication, supply `--api-key` or `STAGEFLOW_API_KEY`.
- The hosted `stageflow.org` API runs the same application code as this repo;
  self-host the identical stack with the [local instructions](../README.md#self-host-locally).
