# StageFlow CLI

Submit URL scan jobs to a StageFlow API, wait for completion, and render the
unified report in shell-friendly formats.

### Usage

```bash
cd clients/cli
go run . scan https://example.com
```

For the recommended local install loop:

```bash
just cli-install
stageflow scan https://example.com
```

Or build and run in place:

```bash
cd clients/cli
go build -o stageflow .
./stageflow scan https://example.com
```

### Commands

| Command | Description |
| --- | --- |
| `scan` | Submit a scan job, wait for completion (SSE by default), then print results |
| `project` | Run a project-mode scan using `.stageflow/config.yaml` |
| `ai` | Run the AI Navigator against a project with natural language objectives |
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

# Run an AI Navigator session
./stageflow ai https://example.com "Navigate to the contact page and submit the form" --expand-provenance

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
  scanners: axe,lighthouse,seo,link-checker
  screenshot: true
  allow_private_targets: true
```
