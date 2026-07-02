# StageFlow CLI and operational tools

StageFlow ships one primary public CLI, `stageflow`, plus a couple of repo-local
helpers for operations and QA. These tools talk to running StageFlow services;
they complement the web UI rather than replace it.

If you need the broader product overview first, start with the [repository README](../../README.md). For everyday CLI usage, pair this page with the [CLI cheatsheet](cli_cheatsheet.md) and the [CLI README](../../clients/cli/README.md).

## stageflow

`stageflow` submits URL scan jobs to the Platform API, waits for completion, and
prints the unified report.

The source lives in `clients/cli/`, but the built binary is named
`stageflow`.

### Build

For the recommended local install loop, run:

```bash
export PATH="$HOME/.local/bin:$PATH"
just cli-install
stageflow version
```

`just cli-install` installs the binary to `~/.local/bin` by default, then
verifies that `stageflow` on your shell `PATH` resolves to that installed
binary. If `~/.local/bin` is missing from `PATH`, or another `stageflow`
earlier on `PATH` shadows it, the command exits with a fix-it message instead
of silently leaving a stale CLI active.

To build the CLI in place, run:

```bash
cd clients/cli
go build -o stageflow .
./stageflow version
```

### Run a URL scan

Run a scan against a public URL by running:

```bash
stageflow scan https://example.com --api https://stageflow.org
```

By default, `stageflow` prints plain text. Add `--format json` for
machine-readable output. `--json` remains available for backward compatibility.

If you built the binary in place instead of installing it, use `./stageflow ...` from `clients/cli/`.

### Run an experimental AI Navigator session

`stageflow ai` is hidden from normal help output while the workflow is still
experimental. Maintainers can run the AI Navigator against a project with
natural-language objectives:

```bash
stageflow ai https://example.com "Navigate to the contact page and submit the form" --api https://stageflow.org
```

Supported flags for the `ai` command:
- `--expand-provenance`: Wait for the scan to finish and expand the step-by-step trace in the output.
- `--allow-private-targets`: Allow scanning private or local network targets.
- `--timeout`: Maximum time to wait for the job to complete (default `5m`).

### Compare against a saved baseline

Use a saved JSON report as the baseline for either another report file or a
live URL:

```bash
stageflow diff baseline.json current.json
stageflow diff baseline.json https://example.com --api https://stageflow.org
```

### Environment variables

Configure the CLI using:

- `STAGEFLOW_API_URL` (default `http://localhost:8080`)
- `STAGEFLOW_API_KEY` (optional, sent as `X-Api-Key`)

### Local dev loop

When you run `stageflow dev scan`, the CLI uses `.stageflow/config.yaml` to
start a local dev server and scan `scan.urls`. Optionally pass a single
positional argument to set the project path; if omitted, the CLI uses the git
root of the current working directory.

Use these companion subcommands:

- `stageflow dev init [path]` to create `.stageflow/config.yaml` and
  `.stageflow/README.md`.
- `stageflow dev doctor [path]` to validate config and preflight checks
  without submitting a scan job.

> **Warning:** The dev loop can execute commands from your repo config. Only run
> it on trusted repositories.

#### Prerequisites for scanning `localhost`

Localhost scans are only intended for local/self-hosted stacks.

- The API must be configured to accept private targets (`PLATFORM_API_ALLOW_PRIVATE_TARGETS=true`).
- The orchestrator must run job pods in the host network namespace on Linux (`POD_NETNS_MODE=host`) so scanners can reach `http://localhost:<port>`.
- The CLI auto-enables `allow_private_targets=true` for private literal targets (for example `localhost`, `127.0.0.1`, RFC1918 ranges, and IPv6 ULA).
- The CLI refuses to submit private/loopback targets to a non-loopback `--api` base URL.

The repo includes a local-only compose override that sets these defaults:

```bash
just images
just dev up local
just dev init local
```

This is intentionally separate from `just demo`. `just demo` is the guided
smoke test for the default local `dev` stack on port `3000`; `just dev up local`
uses the local overlay on port `3020` for localhost/private-target scans.

#### Example `.stageflow/config.yaml`

```yaml
version: 2

stageflow:
  api_url: http://localhost:8080
  project: my-frontend # Optional remote project slug for `stageflow project scan`

scan:
  urls:
    - http://127.0.0.1:5173
  scanners: [axe, lighthouse, seo, link-checker]
  allow_private_targets: true

dev:
  start:
    cmd: ["bun", "run", "dev"]
    cwd: .
  ready:
    url: http://127.0.0.1:5173
```

> **Note:** If `stageflow dev init` cannot infer your dev command, it leaves
> a placeholder in the generated config. Replace that placeholder before running
> `stageflow dev scan`.

> **Note:** `stageflow dev doctor --skip-dev` validates config and scan
> preflight only, without starting your dev process.

#### Notes / limitations

- `.stageflow/config.yaml` is parsed strictly; unknown keys will fail.
- `.stageflow/README.md` includes first-run setup and troubleshooting.
- `POD_NETNS_MODE=host` only works for Linux hosts. On macOS/Windows with a Podman VM, host networking refers to the VM, not your workstation.

## Repo-local helpers

These helpers are mainly for maintainers working against a local StageFlow
stack.

### Contract Generation

`devtools/scripts/generate-contracts.sh` regenerates JSON Schema-derived
contracts under `libs/contracts/`.

Run all generated contract artifacts from the repo root:

```bash
./devtools/scripts/generate-contracts.sh
```

The script accepts mode arguments when you need only one language surface:

```bash
./devtools/scripts/generate-contracts.sh ts
./devtools/scripts/generate-contracts.sh go
```

The generated Go artifacts are type definitions with `UnmarshalJSON` checks
emitted by `go-jsonschema`. Use each contract package's `schema/validate.js`
or dedicated Go validator for full schema validation of JSON fixtures or
incoming payloads.

### job-status-cli

`job-status-cli` queries the orchestrator admin API to inspect jobs, events,
pods, and system metrics.

Build the CLI by running:

```bash
cd devtools/ops/job-status-cli
go build -o job-status-cli .
```

Set `ORCHESTRATOR_ADMIN_URL` to point at the admin API (default
`http://localhost:8081`).

### suite-runner

`suite-runner` runs scans across multiple domains, evaluates threshold
compliance, and exits non-zero when any domain fails.

Build the CLI by running:

```bash
cd devtools/qa/suite-runner
go build -o suite-runner .
```

Set `PLATFORM_API_BASE_URL` to point at the Platform API (default
`http://localhost:8080`).
