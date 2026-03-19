# StageFlow CLI Cheatsheet

## 1. Setup & Build

### Prerequisites
Make sure you have Go installed, as well as the local dev stack running for full functionality.

### Start the Local Dev Stack
```bash
just setup
just dev up local
```
*(Note: `dev up local` enables full internal component communication needed for local scanning).*

### Build the CLI Binary
```bash
cd clients/cli
go build -o stageflow
```

---

## 2. Basic Commands

### Help & Version
```bash
./stageflow --help
./stageflow version
```

### List Scanners
View all available built-in scanners (e.g., a11y, seo, link-checker):
```bash
./stageflow scanners
```

---

## 3. Ad-Hoc Scanning (`scan`)

Run a one-off scan against any publicly accessible URL or local target.

### Scan a Public URL
```bash
./stageflow scan https://example.com
```

### Scan a Local Target
To scan a local instance (e.g., `localhost`), you must pass the `--allow-private-targets` flag:
```bash
./stageflow scan http://localhost:5173 --allow-private-targets
```

### Output Formats
Change the output format to JSON or Markdown using the `--format` flag:
```bash
./stageflow scan https://example.com --format json > report.json
./stageflow scan https://example.com --format markdown > report.md
```

---

## 4. Project Mode (`project`)

Project mode integrates StageFlow into your repository. It automatically manages your dev server lifecycle (starts it, waits for readiness, scans, and shuts it down).

### Initialize a Project
Run this in the root of your Git repository:
```bash
./stageflow project
```
*If this is the first time, it will automatically bootstrap a `.stageflow/config.yaml` file by detecting your package manager and framework.*

### Example `.stageflow/config.yaml`
```yaml
api: http://localhost:8080
scan:
  urls:
    - http://127.0.0.1:5173
  scanners:
    - a11y
    - seo
    - link-checker
    - open-graph
    - spelling-grammar
dev:
  cwd: frontend
  cmd:
    - npm
    - run
    - dev
  ready:
    url: http://127.0.0.1:5173
    timeout: 30s
```

### Run a Project Scan
Once configured, simply run:
```bash
./stageflow project
```
*StageFlow will start the dev server defined in `cmd`, wait for `ready.url` to be accessible, run the scan against `scan.urls`, and gracefully terminate the dev server when finished.*

---

## 5. Fetching Reports (`report`)

If you have an existing Job ID, you can fetch its report without re-scanning:
```bash
./stageflow report <job-id> --format markdown
```
