# StageFlow CLI tools

StageFlow includes a small set of command-line tools for local development and
operations. These tools talk to the running StageFlow services (API and
orchestrator); they don't replace the web UI.

For the full command surface, see `tools/README.md`.

## stageflow

`stageflow` submits URL scan jobs to the Platform API, waits for completion, and
prints the unified report.

The source lives in `tools/stageflow-cli/`, but the built binary is named
`stageflow`.

### Build

Build the CLI by running:

```bash
cd tools/stageflow-cli
go build -o stageflow .
```

### Run a URL scan

Run a scan against a public URL by running:

```bash
./stageflow run --url https://example.com
```

### Environment variables

Configure the CLI using:

- `STAGEFLOW_API_URL` (default `http://localhost:8080`)
- `STAGEFLOW_API_KEY` (optional, sent as `X-Api-Key`)

### Local project mode (EXPERIMENTAL)

When you run `stageflow run` without `--url`, the CLI uses
`.stageflow/config.yaml` to start a local dev server and scan `scan.urls`.
Optionally pass a single positional argument to `stageflow run` to set the
project path; if omitted, the CLI uses the git root of the current working
directory.

> **Warning:** Project mode can execute commands from your repo config. Only run
> it on trusted repositories.

#### Prerequisites for scanning `localhost`

Localhost scans are only intended for local/self-hosted stacks.

- The API must be configured to accept private targets (`PLATFORM_API_ALLOW_PRIVATE_TARGETS=true`).
- The orchestrator must run job pods in the host network namespace on Linux (`POD_NETNS_MODE=host`) so scanners can reach `http://localhost:<port>`.
- The CLI refuses to submit loopback targets (for example `localhost`, `127.0.0.1`) to a non-loopback `--api` base URL.

The repo includes a local-only compose override that sets these defaults:

```bash
just dev up local
just dev init local
```

#### Example `.stageflow/config.yaml`

```yaml
version: 1

stageflow:
  api_url: http://localhost:8080
  api_key_env: STAGEFLOW_API_KEY

scan:
  urls:
    - http://localhost:1337
  scanners: axe,lighthouse
  allow_private_targets: true
  timeout: 5m
  format: summary
  severity: minor
  thresholds:
    critical: 0

dev:
  up:
    - ["bun", "install"]
  start:
    cmd: ["bun", "run", "dev", "--port", "1337"]
  ready:
    url: http://localhost:1337
```

#### Notes / limitations

- `.stageflow/config.yaml` is parsed strictly; unknown keys will fail.
- `POD_NETNS_MODE=host` only works for Linux hosts. On macOS/Windows with a Podman VM, host networking refers to the VM, not your workstation.

## job-status-cli

`job-status-cli` queries the orchestrator admin API to inspect jobs, events,
pods, and system metrics.

Build the CLI by running:

```bash
cd tools/job-status-cli
go build -o job-status-cli .
```

Set `ORCHESTRATOR_ADMIN_URL` to point at the admin API (default
`http://localhost:8081`).

## suite-runner

`suite-runner` runs scans across multiple domains, evaluates threshold
compliance, and exits non-zero when any domain fails.

Build the CLI by running:

```bash
cd tools/suite-runner
go build -o suite-runner .
```

Set `PLATFORM_API_BASE_URL` to point at the Platform API (default
`http://localhost:8080`).
