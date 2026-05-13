# StageFlow CLI Cheatsheet

Quick reference for the day-to-day `stageflow` CLI.

For the broader product overview, start with the [repository README](../../README.md). For the
fuller CLI narrative, see the [CLI README](../../clients/cli/README.md). For
local dev-server scanning, see [Project Mode](../PROJECT_MODE.md). For
repo-local helpers such as `job-status-cli` and `suite-runner`, see
[Devtools](devtools.md).

## 1. Setup

### Start the local stack

Use the local overlay when you want to scan private targets such as `localhost` or `127.0.0.1`:

```bash
cp .env.example .env
just setup
just dev up local
just dev init local
just images
```

`just images` builds the scanner images the stack uses during scans. The first run is the slowest.

### Install the CLI

Recommended install loop:

```bash
just cli-install
stageflow version
```

Build in place instead:

```bash
cd clients/cli
go build -o stageflow .
./stageflow version
```

## 2. Basic commands

### Help and version

```bash
stageflow --help
stageflow version
```

### List scanners

```bash
stageflow scanners
stageflow scanners --format markdown
stageflow scanners --format json
```

## 3. Run scans

### Scan a public URL

```bash
stageflow scan https://example.com
```

### Scan multiple URLs in one job

```bash
stageflow scan https://example.com https://example.com/login --format markdown
```

### Scan a local target

```bash
stageflow scan http://127.0.0.1:5173 --allow-private-targets
```

### Output formats

```bash
stageflow scan https://example.com --format text
stageflow scan https://example.com --format markdown
stageflow scan https://example.com --format json > report.json
```

## 4. AI Navigator

```bash
stageflow ai https://example.com "Navigate to the pricing page and confirm it loads"
stageflow ai https://example.com "Submit the contact form" --expand-provenance
```

Useful flags:

- `--expand-provenance`
- `--allow-private-targets`
- `--timeout 5m`

## 5. Project Mode

Project Mode starts your local app, waits for readiness, runs the scan, and shuts the app down when finished.

### Initialize Project Mode files

```bash
stageflow project init
```

This creates:

- `.stageflow/config.yaml`
- `.stageflow/README.md`

### Example `.stageflow/config.yaml`

```yaml
version: 1

stageflow:
  api_url: http://localhost:8080
  remote_project: my-app # Optional hosted project slug for follow-up remote scans
  remote_api_url: https://stageflow.org # Optional hosted API for the remote project

scan:
  urls:
    - http://127.0.0.1:5173
  scanners: axe,lighthouse,seo,link-checker
  allow_private_targets: true

dev:
  start:
    cmd: ["bun", "run", "dev"]
    cwd: .
  ready:
    url: http://127.0.0.1:5173
```

### Validate before scanning

```bash
stageflow project doctor
stageflow project doctor --skip-dev
```

### Run a project scan

```bash
stageflow project
stageflow project --format markdown
```

These commands use a local `.stageflow/config.yaml`. The optional
`stageflow.remote_project` / `stageflow.remote_api_url` link records which
hosted project to use next, but the local run is still separate from the hosted
regression-memory step.

### Follow with hosted regression memory

```bash
stageflow project --format json
stageflow project hosted --format json
```

Use the first command for the fast local edit-check loop. Use the second when
you want the hosted baseline/diff memory for the associated project without
starting the local dev server.

## 6. Remote projects

Remote projects are named records stored on a StageFlow API. Use them when you want saved targets, baselines, and regression tracking.

```bash
stageflow project create my-app --url https://example.com --scanner axe
stageflow project list
stageflow project show my-app
stageflow project update my-app --url https://example.com/v2
stageflow scan --project my-app --format json
stageflow project hosted --format json
stageflow project promote my-app --job-id <job-id>
stageflow project delete my-app
```

If your local `.stageflow/config.yaml` already declares `stageflow.remote_project:
my-app`, run `stageflow project hosted` from the repo instead of repeating the
slug at the shell.

## 7. Reports

### Fetch an existing report

```bash
stageflow report <job-id>
stageflow report <job-id> --format markdown
stageflow report <job-id> --format json
```

## 8. Compare scans

```bash
stageflow diff baseline.json current.json
stageflow diff baseline.json https://example.com --api https://stageflow.org
```

## 9. Environment variables

- `STAGEFLOW_API_URL` (default `http://localhost:8080`)
- `STAGEFLOW_API_KEY` (optional, sent as `X-Api-Key`)

## 10. Troubleshooting quick hits

- Private or loopback targets require the local overlay: `just dev up local`
- If `stageflow` resolves to the wrong binary, rerun `just cli-install`
- Use `stageflow project doctor` to debug Project Mode readiness problems
- Use `just dev logs` to inspect local stack logs
