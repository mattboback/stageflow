# StageFlow CLI

Submit URL scan jobs to a StageFlow API, wait for completion, and render the
unified report in shell-friendly formats. Supports severity-based exit codes
for CI gating, structured JSON output for automation, and Project Mode for
local dev server scanning.

Need the broader product overview first? Start with the [repository README](../../README.md).

## Install

```bash
just cli-install
stageflow version
```

Or build in place:

```bash
cd clients/cli
go build -o stageflow .
./stageflow version
```

## Commands

| Command | Description |
| --- | --- |
| `scan` | Submit a scan job, wait for completion, print results |
| `project` | Run a Project Mode scan using `.stageflow/config.yaml` |
| `project init` | Scaffold `.stageflow/config.yaml` and `.stageflow/README.md` |
| `project doctor` | Validate project config and dev readiness without scanning |
| `ai` | Run the AI Navigator with natural language objectives |
| `diff` | Compare a saved baseline against another report or a live URL |
| `report` | Fetch and display results for an existing job ID |
| `scanners` | List scanners available on the API |
| `version` | Print version information |
| `completion` | Generate shell completion scripts |
| `docs` | Generate Markdown reference docs |

## Scan examples

```bash
# Scan a public URL (text output, default scanners)
stageflow scan https://example.com --api https://stageflow.org

# Pick scanners and output format
stageflow scan https://example.com --scanners axe,seo --format json --api https://stageflow.org

# Scan multiple routes in one job
stageflow scan https://example.com https://example.com/about --format markdown

# Scan a local dev server
stageflow scan http://127.0.0.1:5173 --allow-private-targets

# Save JSON report to file
stageflow scan https://example.com --format json > report.json
```

## Output formats

| Format | Flag | Best for |
| --- | --- | --- |
| Text | `--format text` (default) | Terminal review — human-readable, compact |
| Markdown | `--format markdown` | PR comments, agent-friendly structured sections |
| JSON | `--format json` | Automation — full report envelope with metadata |

## Quality gates

### Exit codes

| Exit code | Meaning |
| --- | --- |
| 0 | Scan completed, no issues at or above `--fail-on` threshold |
| 1 | Issues meet or exceed `--fail-on` severity threshold |
| 2 | CLI or API error |

### `--fail-on` severity gate

```bash
# Pass — only moderate issues, threshold is serious
stageflow scan https://example.com --scanners axe --fail-on serious   # exit 0

# Fail — moderate issues exist, threshold is moderate
stageflow scan https://example.com --scanners axe --fail-on moderate  # exit 1
```

Severity hierarchy (highest to lowest): `critical` > `serious` > `moderate` > `minor` > `info`.

### Filtering flags

| Flag | Effect |
| --- | --- |
| `--fail-on <sev>` | Exit 1 if any displayed issue meets this severity |
| `--severity <csv>` | Only show issues matching these severities |
| `--category <csv>` | Only show issues matching these categories |
| `--max-issues <n>` | Cap returned issues (default 200, 0 = unlimited) |
| `--summary-only` | Summary counts only, skip individual findings |
| `--group-by <mode>` | Group by `category`, `scanner`, or `none` |

## Compare against a baseline

`stageflow diff` compares a saved JSON report against either another saved
report or a live URL.

```bash
# Compare two saved reports
stageflow diff baseline.json current.json

# Compare a saved baseline against a live URL
stageflow diff baseline.json https://example.com --api https://stageflow.org
```

Use `--fail-on-new` to gate on newly introduced issues and
`--fail-on-regression` to fail when scores drop or new issues appear.

## JSON report envelope

`--format json` outputs a versioned envelope (`stageflow-cli/report@v1`):

```jsonc
{
  "schema": "stageflow-cli/report@v1",
  "cli":    { "version": "...", "commit": "...", "date": "..." },
  "api":    { "base_url": "https://stageflow.org" },
  "job":    { "id": "...", "state": "DONE" },
  "links":  { "job": "https://...", "results": "https://..." },
  "urls":   ["https://example.com"],
  "filters": {
    "max_issues": 200,
    "issues_returned": 2,
    "issues_total": 2,
    "truncated": false
  },
  "report": {
    "summary": {
      "score": 85,
      "scoreGrade": "B",
      "totalIssues": 2,
      "bySeverity": { "critical": 0, "serious": 0, "moderate": 2, "minor": 0 },
      "byScanner":  { "axe": 2 }
    },
    "issues": [{
      "id": "672859f7e59a",           // stable content-based hash
      "ruleId": "landmark-one-main",
      "scanner": "axe",
      "severity": "moderate",
      "title": "Document should have one main landmark",
      "description": "Each page should contain exactly one <main> landmark.",
      "howToFix": "Fix all of the following: ...",
      "wcagTags": ["WCAG 1.3.1"],
      "helpUrl": "https://dequeuniversity.com/rules/axe/...",
      "occurrences": [{
        "selector": "html",
        "html": "<html lang=\"en\">",
        "target": ["html"],
        "contextHtml": "...",
        "ancestorPath": "html"
      }]
    }],
    "scanners": [{ "id": "axe", "status": "success", "issueCount": 2, "durationMs": 1788 }],
    "pages":    [{ "url": "https://example.com", "issueCount": 2, "durationMs": 1500 }]
  }
}
```

Issue `id` fields are content-based hashes — the same violation on the same page produces the same `id` across runs, making them reliable for regression diffing.

## Project Mode

`stageflow project` (without remote CRUD subcommands) is the local Project Mode workflow. The remote commands such as `stageflow project create`, `list`, `show`, `update`, `delete`, and `promote` manage named project records on a StageFlow API instead.

Project Mode automates the full scan lifecycle for local development:
start dev server, wait for readiness, submit scan, stream results, stop server.

```bash
stageflow project init          # scaffold config
stageflow project doctor        # validate config without scanning
stageflow project               # full lifecycle
```

### Example config (`.stageflow/config.yaml`)

```yaml
version: 1

stageflow:
  api_url: "http://localhost:8080"

scan:
  urls:
    - http://127.0.0.1:5173/
    - http://127.0.0.1:5173/login
  scanners: axe,lighthouse,seo,link-checker
  screenshot: true
  allow_private_targets: true

dev:
  start:
    cmd: ["bun", "run", "dev"]
    cwd: .
  ready:
    url: http://127.0.0.1:5173
```

Cover the routes you care about in `scan.urls` — scanning only `/` misses
route-specific regressions.

See [Project Mode docs](../../docs/PROJECT_MODE.md) for the full configuration reference.

## Environment variables

| Variable | Default | Description |
| --- | --- | --- |
| `STAGEFLOW_API_URL` | `http://localhost:8080` | Platform API base URL (overridden by `--api`) |
| `STAGEFLOW_API_KEY` | *(unset)* | API key (sent as `X-Api-Key`, overridden by `--api-key`) |
