# StageFlow CLI

Submit URL scan jobs to a StageFlow API, wait for completion, and render the
unified report in shell-friendly formats. Supports severity-based exit codes
for CI gating, structured JSON output for automation, a dev-server scan loop
for local development, and baseline-aware project diffs for regression gating
after frontend edits.

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

## The three ways to scan

| Command | What it scans | When to use it |
| --- | --- | --- |
| `stageflow scan <url>` | Any URL you give it | One-off checks, ad-hoc audits |
| `stageflow dev scan` | Your local dev server (started for you) | Local edit-then-check loops |
| `stageflow project scan [slug]` | A project registered on the API, diffed against its baseline | CI regression gating |

## Commands

| Command | Description |
| --- | --- |
| `scan` | Scan one or more URLs and report the results |
| `dev init` | Scaffold `.stageflow/config.yaml` and `.stageflow/README.md` |
| `dev doctor` | Validate the config and dev-server readiness without scanning |
| `dev scan` | Start the dev server, scan it, and report results |
| `project create/list/show/update/delete` | Manage remote projects on the API |
| `project scan` | Scan a remote project and diff against its baseline |
| `project promote` | Promote a scan job to be the project baseline |
| `diff` | Compare a saved baseline against another report or a live URL |
| `report` | Fetch and display results for an existing job ID |
| `auth capture` | Launch Chromium for interactive login and write Playwright storage state |
| `scanners` | List scanners available on the API |
| `version` | Print version information |
| `completion` | Generate shell completion scripts |
| `docs` | Generate Markdown reference docs |

## Scan examples

```bash
# Scan a public URL (text output, default scanners)
stageflow scan https://example.com --api https://stageflow.org

# Pick scanners and output format
stageflow scan https://example.com --scanner axe,seo --format json --api https://stageflow.org

# Scan multiple routes in one job
stageflow scan https://example.com https://example.com/about --format markdown

# Scan a local dev server
stageflow scan http://127.0.0.1:5173 --allow-private-targets

# Save JSON report to file
stageflow scan https://example.com --format json > report.json
```

`--scanner` is repeatable and comma-tolerant: `--scanner axe --scanner seo`
and `--scanner axe,seo` are equivalent.

## Agent gating loop

StageFlow is designed to fit an edit-then-check terminal workflow:

```bash
# Local dev loop
stageflow dev init --format json > dev-init.json
stageflow dev doctor --format json > dev-doctor.json
stageflow dev scan --format json > local-scan.json

# Follow-up remote regression loop
stageflow project scan --format json > project-scan.json
```

Use the local dev loop when you want fast feedback against a dev server. Use
`stageflow project scan` when you want baseline memory and regression diffs
for a project already registered on a StageFlow API, without starting the
local dev server.

`dev init` and `dev doctor` also support JSON output so agents can bootstrap
and validate the local loop without scraping human-oriented text.
`dev doctor --format json` also exposes the linked remote project when one is
configured.

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
| 1 | Issues meet or exceed `--fail-on` severity threshold, or a baseline regression |
| 2 | CLI or API error |

### `--fail-on` severity gate

```bash
# Pass — only moderate issues, threshold is serious
stageflow scan https://example.com --scanner axe --fail-on serious   # exit 0

# Fail — moderate issues exist, threshold is moderate
stageflow scan https://example.com --scanner axe --fail-on moderate  # exit 1
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

The full filter set is documented on `stageflow report --help`.

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

For registered remote projects, `stageflow project scan <slug> --format json`
emits a single envelope that includes both the scan report and any available
regression diff, which makes it easier for agents to parse one terminal
payload instead of correlating separate commands.

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

## The dev loop

`stageflow dev` automates the full scan lifecycle for local development:
start dev server, wait for readiness, submit scan, stream results, stop server.

```bash
stageflow dev init          # scaffold config
stageflow dev doctor        # validate config without scanning
stageflow dev scan          # full lifecycle
```

### Example config (`.stageflow/config.yaml`)

```yaml
version: 2

stageflow:
  api_url: "http://localhost:8080"
  project: "my-frontend" # Optional remote project slug for `stageflow project scan`

scan:
  urls:
    - http://127.0.0.1:5173/
    - http://127.0.0.1:5173/login
  scanners: [axe, lighthouse, seo, link-checker]
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

`stageflow.project` is optional for local runs; it lets `stageflow project scan`
(run with no slug) find the remote project from the repo config.

See the [dev mode guide](../../docs/dev-mode.md) for the full configuration reference.

## Remote projects and baselines

Remote project commands manage the regression-memory layer hosted on the API:

```bash
stageflow project create my-frontend --url https://staging.example.com --api https://stageflow.org
stageflow project scan my-frontend --api https://stageflow.org
stageflow project promote my-frontend --job-id <job-id> --api https://stageflow.org
stageflow project scan my-frontend --api https://stageflow.org   # now diffs against the baseline
```

That flow is what lets StageFlow answer "did this frontend change regress from
the last known-good baseline?" rather than only "what issues exist right now?"
If `.stageflow/config.yaml` already records `stageflow.project: my-frontend`,
run `stageflow project scan` from the repo with no slug instead of repeating
it at the shell.

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

`--auth-state` and `--auth-recipe` are mutually exclusive, and they apply only
to `stageflow scan` (for `project scan`, configure auth on the registered
project instead).

The full design, trust boundaries, and threat model live in
[docs/architecture/system.md#authenticated-scanning](../../docs/architecture/system.md#authenticated-scanning).
