# StageFlow Tools

Developer and operations CLI utilities for StageFlow.

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

| Command  | Description |
|----------|-------------|
| `jobs`   | List jobs with optional state filtering and pagination |
| `events` | Show the event timeline for a specific job |
| `pods`   | List all orchestrator-managed pods |
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
|----------|---------|-------------|
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

Run accessibility compliance scans across multiple domains in parallel, evaluate results against thresholds, and generate a pass/fail report.

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
|------|---------|-------------|
| `-suite <path>` | `suite.yml` | Path to YAML suite definition |
| `-api <url>` | `http://localhost:8080` | Platform API base URL (or `PLATFORM_API_BASE_URL` env var) |

### Suite YAML Format

```yaml
domains:
  - https://example.com
  - https://blog.example.com

modules:
  - axe
  - keyboard

screenshot: false

thresholds:
  max_critical: 0      # Max allowed critical violations
  max_serious: 5        # Max allowed serious violations
  max_total: 50         # Max allowed total violations

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
