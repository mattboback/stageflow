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

For the recommended local install loop, run:

```bash
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
cd tools/stageflow-cli
go build -o stageflow .
```

### Run a URL scan

Run a scan against a public URL by running:

```bash
./stageflow scan https://example.com
```

By default, `stageflow` prints plain text. Add `--format json` for
machine-readable output. `--json` remains available for backward compatibility.

### Run an AI Navigator session

Run the AI Navigator against a project with natural language objectives:

```bash
./stageflow ai https://example.com "Navigate to the contact page and submit the form"
```

Supported flags for the `ai` command:
- `--expand-provenance`: Wait for the scan to finish and expand the step-by-step trace in the output.
- `--allow-private-targets`: Allow scanning private or local network targets.
- `--timeout`: Maximum time to wait for the job to complete (default `5m`).

### Environment variables

Configure the CLI using:

- `STAGEFLOW_API_URL` (default `http://localhost:8080`)
- `STAGEFLOW_API_KEY` (optional, sent as `X-Api-Key`)

### Local project mode (EXPERIMENTAL)

When you run `stageflow project`, the CLI uses `.stageflow/config.yaml` to
start a local dev server and scan `scan.urls`. Optionally pass a single
positional argument to set the project path; if omitted, the CLI uses the git
root of the current working directory.

Use these companion subcommands:

- `stageflow project init [path]` to create `.stageflow/config.yaml` and
  `.stageflow/README.md`.
- `stageflow project doctor [path]` to validate config and preflight checks
  without submitting a scan job.

> **Warning:** Project mode can execute commands from your repo config. Only run
> it on trusted repositories.

#### Prerequisites for scanning `localhost`

Localhost scans are only intended for local/self-hosted stacks.

- The API must be configured to accept private targets (`PLATFORM_API_ALLOW_PRIVATE_TARGETS=true`).
- The orchestrator must run job pods in the host network namespace on Linux (`POD_NETNS_MODE=host`) so scanners can reach `http://localhost:<port>`.
- The CLI auto-enables `allow_private_targets=true` for private literal targets (for example `localhost`, `127.0.0.1`, RFC1918 ranges, and IPv6 ULA).
- The CLI refuses to submit private/loopback targets to a non-loopback `--api` base URL.

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

scan:
  urls:
    - http://127.0.0.1:1337
  scanners: axe,lighthouse,seo,security-headers,open-graph,spelling-grammar
  allow_private_targets: true

dev:
  start:
    cmd: ["__STAGEFLOW_SET_DEV_START_CMD__"] # replace this
    cwd: .
  ready:
    url: http://localhost:1337
```

> **Note:** `stageflow project` fails fast with setup guidance if
> `dev.start.cmd` still uses the scaffold placeholder.

> **Note:** `stageflow project doctor --skip-dev` validates config and scan
> preflight only, without starting your dev process.

#### Notes / limitations

- `.stageflow/config.yaml` is parsed strictly; unknown keys will fail.
- `.stageflow/README.md` includes first-run setup and troubleshooting.
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
