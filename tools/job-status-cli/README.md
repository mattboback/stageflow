# job-status-cli

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
