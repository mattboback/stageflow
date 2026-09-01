# CLI Guide

The `stageflow` Go binary submits scans to a StageFlow API, streams progress over SSE, renders reports, and returns stable exit codes for automation. Exact flags and defaults live in the [generated command reference](reference/cli/stageflow/stageflow.md).

## Install

```bash
just cli-install
stageflow version
```

Release binaries are available from [GitHub Releases](https://github.com/mattboback/stageflow/releases). To build from source, run `go build -o stageflow ./clients/cli` from the repository root.

## Choose a Workflow

| Command | Use it for |
| --- | --- |
| `stageflow scan <url...>` | One-off public or private URL scans |
| `stageflow scan <dir-or-zip>` | Uploading and scanning a static build |
| `stageflow dev scan` | Starting, scanning, and stopping a local dev server |
| `stageflow project scan [slug]` | Comparing a registered project with its promoted baseline |

All commands accept `--api`; it defaults to `http://localhost:8080`. Set `STAGEFLOW_API_URL` and, when required, `STAGEFLOW_API_KEY` to avoid repeating connection flags.

## One-Off Scans

```bash
stageflow scan https://example.com --api https://stageflow.org
stageflow scan https://example.com/about https://example.com/pricing --format markdown
stageflow scan ./dist --api https://stageflow.org
stageflow scan site.zip --api https://stageflow.org
```

Directory scans exclude VCS metadata and `node_modules`, require a top-level `index.html`, and enforce the upload limit documented by `stageflow scan --help`.

Private targets require a local API configured for private-target intake:

```bash
stageflow scan http://127.0.0.1:5173 --allow-private-targets
```

The CLI refuses to send a private target to a non-loopback API even when the flag is present.

## Output and Exit Codes

Use `--format text` for terminal output, `--format markdown` for review artifacts, and `--format json` for a versioned machine-readable envelope. Progress is written to stderr so stdout remains safe to redirect.

| Code | Meaning |
| --- | --- |
| `0` | Scan passed the configured severity and regression gates |
| `1` | A displayed issue met `--fail-on`, or a project regressed |
| `2` | CLI, configuration, or API error |

Common report controls are `--fail-on`, `--severity`, `--category`, `--max-issues`, `--summary-only`, and `--group-by`. Consult `stageflow report --help` for exact values.

## Local Dev Loop

`stageflow dev scan` reads a trusted repository's `.stageflow/config.yaml`, runs setup commands, starts the app, waits for readiness, submits the scan, and stops the process. Do not run repository-defined dev commands in an untrusted checkout.

```bash
stageflow dev init
stageflow dev doctor
stageflow dev scan --format json
```

`dev init` resolves the git root, detects common package-manager and Justfile commands, and creates `.stageflow/config.yaml` plus a local quick-start README. Replace any unresolved startup placeholder before scanning.

```yaml
version: 2

stageflow:
  api_url: http://localhost:8080
  project: my-frontend

scan:
  urls:
    - http://127.0.0.1:5173
  scanners: [axe, lighthouse, seo, link-checker]
  allow_private_targets: true

dev:
  start:
    cmd: ["bun", "run", "dev"]
    cwd: .
  ready:
    url: http://127.0.0.1:5173
```

Important fields:

| Field | Purpose |
| --- | --- |
| `stageflow.api_url` | API used by CLI workflows |
| `stageflow.api_key_env` | Environment variable containing the API key |
| `stageflow.project` | Optional registered project used by slug-free `project scan` |
| `scan.urls` | Routes served by the dev app; include every route that matters |
| `scan.scanners` | Scanner selection |
| `scan.allow_private_targets` | Required for local/private targets |
| `dev.up` / `dev.down` | Optional setup and teardown commands |
| `dev.start` | Long-running app command, working directory, and environment |
| `dev.ready` | URL, timeout, and interval used for readiness polling |
| `dev.stop` | Shutdown signal and graceful timeout |

Use `stageflow dev doctor --skip-dev` for config/API preflight without starting the app. Use normal `doctor` to test the full start/readiness/stop cycle without submitting a scan.

## Projects and Baselines

A project is API-owned state: a slug, URLs, scanner selection, completed jobs, and an optional promoted baseline.

```bash
stageflow project create marketing-site \
  --url https://example.com \
  --scanner axe --scanner seo
stageflow project scan marketing-site --format json
stageflow project promote marketing-site --job-id <job-id>
stageflow project scan marketing-site --format json
```

After promotion, each scan includes a server-side diff. A score decrease relative to the promoted baseline, any new issue, or a configured severity failure return exit code `1`. Re-promote a known-good completed job when intentionally accepting a new baseline.

Manage records with `project list`, `show`, `update`, and `delete`. Repeated `--url` or `--scanner` values on `update` replace the corresponding list.

When `.stageflow/config.yaml` contains `stageflow.project`, run this from the repository without repeating the slug:

```bash
stageflow project scan --format json
```

This does not start the local dev server. A useful sequence is local validation with `dev scan`, followed by the API-owned regression check with `project scan`.

## CI Gate

Anonymous one-off scans against the public demo work without a key:

```bash
stageflow scan https://example.com --fail-on serious --api https://stageflow.org
```

`project create`, `project scan`, `project promote`, and `/diff` need a caller API
key. The hosted demo does not issue those keys. Point CI project gates at your
own StageFlow API:

```yaml
- name: StageFlow regression gate
  env:
    STAGEFLOW_API_URL: https://stageflow.example.com
    STAGEFLOW_API_KEY: ${{ secrets.STAGEFLOW_API_KEY }}
  run: stageflow project scan marketing-site --fail-on serious
```

## Saved Reports and Live Diffs

```bash
stageflow report <job-id> --format json
stageflow diff baseline.json current.json
stageflow diff baseline.json https://example.com --api http://localhost:8080
```

Use `--fail-on-new` to gate newly introduced issues and `--fail-on-regression` to include score regressions.

## Authenticated Scans

Capture a developer session as Playwright storage state:

```bash
stageflow auth capture https://app.example.com/login --output ./auth/state.json
stageflow scan https://app.example.com/profile --auth-state ./auth/state.json
```

The capture file is written with mode `0600`. For CI, use `--auth-recipe` with a form recipe whose sensitive values reference allow-listed environment variables. The CLI validates recipes but does not resolve credentials. `--auth-state` and `--auth-recipe` are mutually exclusive and apply to one-off scans; configure authentication on registered projects for project scans.

See [Authenticated Scanning](architecture.md#authenticated-scanning) for credential lifecycle and trust boundaries.

## Troubleshooting

- Run `stageflow dev doctor` for config or readiness failures.
- Ensure `scan.urls` and `dev.ready.url` use the port and hostname the app actually binds.
- Run `just dev logs` to inspect the local stack.
- Re-run `just cli-install` if another binary shadows `~/.local/bin/stageflow`.
- Use the [self-hosting guide](self-hosting.md#private-target-development) when local scanner pods cannot reach the app.
