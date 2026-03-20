# StageFlow CLI Cheatsheet

Quick reference for the `stageflow` CLI. For full command docs, see `clients/cli/README.md`, `docs/operations/devtools.md`, and `docs/PROJECT_MODE.md`.

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

## 5. Project mode

Project mode starts your local app, waits for readiness, runs the scan, and shuts the app down when finished.

### Initialize project mode files

```bash
stageflow project init
```

This scaffolds:

- `.stageflow/config.yaml`
- `.stageflow/README.md`

### Example `.stageflow/config.yaml`

```yaml
version: 1

stageflow:
  api_url: http://localhost:8080

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

## 6. Reports

### Fetch an existing report

```bash
stageflow report <job-id>
stageflow report <job-id> --format markdown
stageflow report <job-id> --format json
```

## 7. Environment variables

- `STAGEFLOW_API_URL` (default `http://localhost:8080`)
- `STAGEFLOW_API_KEY` (optional, sent as `X-Api-Key`)

## 8. Troubleshooting quick hits

- Private or loopback targets require the local overlay: `just dev up local`
- If `stageflow` resolves to the wrong binary, rerun `just cli-install`
- Use `stageflow project doctor` to debug project-mode readiness problems
- Use `just dev logs` to inspect local stack logs

