# StageFlow CLI

Submit URL scan jobs to a StageFlow API, wait for completion, and render the
unified report in shell-friendly formats. Supports severity-based exit codes
for CI gating, structured JSON output for automation, Project Mode for
local dev server scanning, and baseline-aware project diffs for regression
gating after frontend edits.

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
| `auth capture` | Launch Chromium for interactive login and write Playwright storage state |
| `project` | Run a Project Mode scan using `.stageflow/config.yaml` |
| `project init` | Scaffold `.stageflow/config.yaml` and `.stageflow/README.md` |
| `project doctor` | Validate project config and dev readiness without scanning |
| `project hosted` | Run the configured hosted project scan without starting local dev |
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

## Agent gating loop

StageFlow is designed to fit an edit-then-check terminal workflow:

```bash
# Local dev loop
stageflow project init --format json > project-init.json
stageflow project doctor --format json > project-doctor.json
stageflow project --format json > local-scan.json

# Follow-up hosted regression loop
stageflow project hosted --format json > hosted-scan.json
```

Use the local Project Mode loop when you want fast feedback against a dev
server. Use `stageflow project hosted` when you want hosted baseline memory and
regression diffs for a project already registered on a StageFlow API, without
starting the local dev server.

`project init` and `project doctor` also support JSON output so agents can
bootstrap and validate the local loop without scraping human-oriented text.
`project doctor --format json` also exposes the hosted project association when
one is configured.

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

For registered remote projects, `stageflow scan --project <slug> --format json`
now emits a single envelope that includes both the scan report and any
available regression diff, which makes it easier for agents to parse one
terminal payload instead of correlating separate commands.

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
stageflow project hosted        # hosted scan from remote_project
```

### Example config (`.stageflow/config.yaml`)

```yaml
version: 1

stageflow:
  api_url: "http://localhost:8080"
  remote_project: "my-frontend" # Optional hosted project slug for follow-up remote scans
  remote_api_url: "https://stageflow.org" # Optional hosted API for the remote project

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

`stageflow.remote_project` is optional for local runs, but required by
`stageflow project hosted`. Hosted mode reads only the repository config needed
to find the remote project and API, then delegates to the same baseline/diff
flow used by `stageflow scan --project ...`.

See [Project Mode docs](../../docs/PROJECT_MODE.md) for the full configuration reference.

## Remote projects and baselines

Remote project commands manage the hosted regression-memory layer:

```bash
stageflow project create my-frontend --url https://staging.example.com --api https://stageflow.org
stageflow scan --project my-frontend --api https://stageflow.org
stageflow project hosted --format json
stageflow project promote my-frontend --job-id <job-id> --api https://stageflow.org
```

That flow is what lets StageFlow answer "did this frontend change regress from
the last known-good baseline?" rather than only "what issues exist right now?"
If `.stageflow/config.yaml` already records `stageflow.remote_project: my-frontend`,
run `stageflow project hosted` from the repo instead of repeating the slug and
API URL at the shell.

## Environment variables

| Variable | Default | Description |
| --- | --- | --- |
| `STAGEFLOW_API_URL` | `http://localhost:8080` | Platform API base URL (overridden by `--api`) |
| `STAGEFLOW_API_KEY` | *(unset)* | API key (sent as `X-Api-Key`, overridden by `--api-key`) |



## Authenticated scans

Scan beyond a marketing landing page or login redirect by attaching a session
to the scan job. Two flows, both fronted by `stageflow auth …` and
`stageflow scan --auth-state | --auth-recipe`:

### Capture a developer session (one-off / personal apps)

```bash
# Local interactive login. Writes a Playwright storage-state JSON with mode 0600.
stageflow auth capture https://app.example.com/login --output ./auth/state.json

# Attach it to a scan. The CLI base64-encodes the file, the orchestrator
# uploads it to MinIO under the job's prefix, and Provenance.auth references
# it as `{mode: storage_state, artifact_key: ...}`.
stageflow scan https://app.example.com/profile --auth-state ./auth/state.json
```

`stageflow auth capture` shells out to `npx playwright open --save-storage`,
so the developer machine needs Node.js and Playwright (e.g. `npm install -g
playwright`, or run from a project that already has `playwright` as a
dependency).

### Declarative form recipe (CI-driven)

Save a YAML/JSON recipe that mirrors the `Provenance.auth.form` shape. Step
values are either literal strings or `{from_env: NAME}` references — the CLI
never resolves names, the orchestrator forwards exactly the named env vars
into the scanner pod and nothing else.

```yaml
# auth/recipe.yaml
mode: form
login_url: https://app.example.com/login
steps:
  - type: fill
    selector: input[name=email]
    value:
      from_env: STAGEFLOW_AUTH_USER
  - type: fill
    selector: input[name=password]
    value:
      from_env: STAGEFLOW_AUTH_PASSWORD
  - type: click
    selector: button[type=submit]
success:
  type: selector
  selector: '[data-test=signed-in]'
  timeout: 15000
```

```bash
# CI runs this with STAGEFLOW_AUTH_USER and STAGEFLOW_AUTH_PASSWORD set in
# the orchestrator host's environment.
stageflow scan https://app.example.com/profile --auth-recipe ./auth/recipe.yaml
```

`--auth-state` and `--auth-recipe` are mutually exclusive, and neither is
supported alongside `--project` (configure auth on the registered project
instead).

The full design, trust boundaries, and threat model live in
[docs/architecture/system.md#authenticated-scanning](../../docs/architecture/system.md#authenticated-scanning).
