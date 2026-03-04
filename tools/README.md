# StageFlow Tools

Developer and operations CLI utilities for StageFlow.

## stageflow-cli

Submit URL scan jobs to a StageFlow API, wait for completion, and render the
unified report in shell-friendly formats.

### Usage

```bash
cd tools/stageflow-cli
go run . scan https://example.com
```

For the recommended local install loop:

```bash
just cli-install
stageflow scan https://example.com
```

Or build and run in place:

```bash
cd tools/stageflow-cli
go build -o stageflow .
./stageflow scan https://example.com
```

### Commands

| Command | Description |
| --- | --- |
| `scan` | Submit a scan job, wait for completion (SSE by default), then print results |
| `project` | Run a project-mode scan using `.stageflow/config.yaml` |
| `report` | Fetch and display results for an existing job ID |
| `scanners` | List scanners available on the API |
| `version` | Print version information |
| `completion` | Generate shell completion scripts |
| `docs` | Generate Markdown docs for the CLI |

If `.stageflow/config.yaml` is missing, `stageflow project` creates:

- `.stageflow/config.yaml`
- `.stageflow/README.md`

It then prints setup instructions and exits.

Starter configs include a placeholder `dev.start.cmd`; `stageflow project`
fails fast with setup guidance until you replace it.

Use `stageflow project init` for explicit bootstrap and
`stageflow project doctor` to validate setup before scanning.

### Environment Variables

| Variable | Default | Description |
| --- | --- | --- |
| `STAGEFLOW_API_URL` | `http://localhost:8080` | Platform API base URL |
| `STAGEFLOW_API_KEY` | *(unset)* | Optional API key (sent as `X-Api-Key`) |

### Output formats

`stageflow-cli` supports `text`, `markdown`, and `json` output.

- Use `--format markdown` when you want stable section headings and
  agent-friendly semantic output.
- Use `--format json` when you need a machine-readable envelope.
- Use `--json` only for backward compatibility. It behaves the same as
  `--format json`.

### Examples

```bash
# Plain-text output (default)
./stageflow scan https://example.com

# Markdown output for human and agent review
./stageflow scan https://example.com --format markdown

# JSON output for automation
./stageflow scan https://example.com --format json > report.json

# Project-mode scan (uses .stageflow/config.yaml)
./stageflow project

# Project-mode scan in Markdown
./stageflow project --format markdown

# Review an existing job in Markdown
./stageflow report <job-id> --format markdown

# Scan multiple routes in one job
./stageflow scan https://example.com https://example.com/login --format markdown

# Explicitly scaffold project mode files
./stageflow project init

# Validate config and readiness without scanning
./stageflow project doctor

# List scanners (plain text default)
./stageflow scanners

# List scanners in Markdown
./stageflow scanners --format markdown

# List scanners in JSON
./stageflow scanners --format json
```

### Project-mode route coverage

Project mode is most useful when `scan.urls` covers the public routes you care
about. Start with a short curated list instead of only scanning `/`.

```yaml
scan:
  urls:
    - http://127.0.0.1:5173/
    - http://127.0.0.1:5173/login
  scanners: axe,lighthouse,seo,security-headers,link-checker
  screenshot: true
  allow_private_targets: true
```

---

## job-status-cli

Query the orchestrator admin API to inspect jobs, events, pods, and system metrics.

### Usage

```bash
cd tools/job-status-cli
go run . <command> [flags]
```

Or build and run:

```bash
go build -o job-status-cli .
./job-status-cli <command> [flags]
```

### Commands

| Command | Description |
| --- | --- |
| `jobs` | List jobs with optional state filtering and pagination |
| `events` | Show the event timeline for a specific job |
| `pods` | List all orchestrator-managed pods |
| `status` | Display system-wide job and pod metrics |

### Flags

**jobs:**
- `-state <STATE>` — Filter by job state (PENDING, EXTRACTING, READY_TO_SCAN, SCANNING, COMPLETING, DONE, FAILED)
- `-limit <N>` — Maximum jobs to display (default: 20)
- `-offset <N>` — Skip N jobs for pagination (default: 0)

**events:**
- First positional argument: job ID (required)
- `-limit <N>` — Maximum events to display (default: 500)
- `-offset <N>` — Skip N events (default: 0)
- `-payload` — Print full event payload JSON

### Environment Variables

| Variable | Default | Description |
| --- | --- | --- |
| `ORCHESTRATOR_ADMIN_URL` | `http://localhost:8081` | Orchestrator admin API endpoint |

### Examples

```bash
# List all jobs
./job-status-cli jobs

# List failed jobs
./job-status-cli jobs -state FAILED

# Show events for a specific job
./job-status-cli events abc123-def456

# Show events with full payloads
./job-status-cli events abc123-def456 -payload

# Check system status
./job-status-cli status
```

---

## suite-runner

Run accessibility scans across multiple domains, evaluate threshold compliance, and generate a pass/fail report.

### Usage

```bash
cd tools/suite-runner
go run . [flags]
```

Or build and run:

```bash
go build -o suite-runner .
./suite-runner [flags]
```

### Flags

| Flag | Default | Description |
| --- | --- | --- |
| `-suite <path>` | `suite.yml` | Path to YAML suite definition |
| `-api <url>` | `http://localhost:8080` | Platform API base URL (or `PLATFORM_API_BASE_URL` env var) |

### Suite YAML Format

```yaml
domains:
  - https://example.com
  - https://blog.example.com

modules:
  - axe
  - lighthouse

screenshot: false

thresholds:
  max_critical: 0      # Max allowed critical violations
  max_serious: 5       # Max allowed serious violations
  max_total: 50        # Max allowed total violations

timeout_seconds: 900    # Overall test timeout (default: 900)
stream_retry_seconds: 3 # SSE reconnect delay (default: 3)
```

### How It Works

1. Loads the YAML suite configuration
2. Submits a scan job for each domain via the Platform API
3. Streams results using Server-Sent Events with auto-reconnect
4. Evaluates each domain's results against violation thresholds
5. Prints a summary table with PASS/FAIL per domain
6. Exits with code 0 if all pass, 1 if any fail

### Examples

```bash
# Run with default suite.yml
./suite-runner

# Run a specific suite against staging
./suite-runner -suite tests/a11y-suite.yml -api https://staging.example.com

# Using environment variable for API URL
PLATFORM_API_BASE_URL=https://staging.example.com ./suite-runner -suite suite.yml
```
