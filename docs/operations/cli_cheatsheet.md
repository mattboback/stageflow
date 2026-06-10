# StageFlow CLI Cheatsheet

Quick reference for the day-to-day `stageflow` CLI.

For the broader product overview, start with the [repository README](../../README.md). For the
fuller CLI narrative, see the [CLI README](../../clients/cli/README.md). For
local dev-server scanning, see [Dev Mode](../dev-mode.md). For
repo-local helpers such as `job-status-cli` and `suite-runner`, see
[Devtools](devtools.md).

## 1. Setup

### Start the local stack

Use the local overlay when you want to scan private targets such as `localhost` or `127.0.0.1`:

```bash
cp .env.example .env
just setup
just images
just dev up local
just dev init local
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

## 4. The dev loop

`stageflow dev` starts your local app, waits for readiness, runs the scan, and shuts the app down when finished.

### Initialize the dev loop

```bash
stageflow dev init
```

This creates:

- `.stageflow/config.yaml`
- `.stageflow/README.md`

### Example `.stageflow/config.yaml`

```yaml
version: 2

stageflow:
  api_url: http://localhost:8080
  project: my-app # Optional remote project slug for `stageflow project scan`

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

### Validate before scanning

```bash
stageflow dev doctor
stageflow dev doctor --skip-dev
```

### Run the dev-loop scan

```bash
stageflow dev scan
stageflow dev scan --format markdown
```

These commands use a local `.stageflow/config.yaml`. The optional
`stageflow.project` link records which remote project to use next, but the
local run is still separate from the remote regression-memory step.

### Follow with remote regression memory

```bash
stageflow dev scan --format json
stageflow project scan --format json
```

Use the first command for the fast local edit-check loop. Use the second when
you want the baseline/diff memory for the linked project without starting the
local dev server.

## 5. Remote projects

Remote projects are named records stored on a StageFlow API. Use them when you want saved targets, baselines, and regression tracking.

```bash
stageflow project create my-app --url https://example.com --scanner axe
stageflow project list
stageflow project show my-app
stageflow project update my-app --url https://example.com/v2
stageflow project scan my-app --format json
stageflow project promote my-app --job-id <job-id>
stageflow project delete my-app
```

If your local `.stageflow/config.yaml` already declares `stageflow.project:
my-app`, run `stageflow project scan` from the repo with no slug instead of
repeating it at the shell.

## 6. Reports

### Fetch an existing report

```bash
stageflow report <job-id>
stageflow report <job-id> --format markdown
stageflow report <job-id> --format json
```

## 7. Compare scans

```bash
stageflow diff baseline.json current.json
stageflow diff baseline.json https://example.com --api https://stageflow.org
```

## 8. Environment variables

- `STAGEFLOW_API_URL` (default `http://localhost:8080`)
- `STAGEFLOW_API_KEY` (optional, sent as `X-Api-Key`)

## 9. Troubleshooting quick hits

- Private or loopback targets require the local overlay: `just dev up local`
- If `stageflow` resolves to the wrong binary, rerun `just cli-install`
- Use `stageflow dev doctor` to debug dev-loop readiness problems
- Use `just dev logs` to inspect local stack logs
