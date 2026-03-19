# suite-runner

Run accessibility scans across multiple domains, evaluate threshold compliance, and generate a pass/fail report.

### Usage

```bash
cd qa/suite-runner
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
