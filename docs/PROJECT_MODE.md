# StageFlow Project Mode

Project Mode integrates StageFlow scans directly into your local development
workflow.

Instead of manually starting your app and then running scans, Project Mode
automates the full lifecycle: start the dev server, wait for readiness, run the
StageFlow scanners, stream the results, and shut the dev server down when the
scan finishes.

If you need the bigger picture first, start with the [repository README](../README.md), the [CLI README](../clients/cli/README.md), or the [CLI cheatsheet](operations/cli_cheatsheet.md).

## Local Project Mode vs. remote projects

This document covers **local Project Mode**:

- `stageflow project`
- `stageflow project init`
- `stageflow project doctor`

These commands use a local `.stageflow/config.yaml` to start your dev server, scan it, and stop it again.

That same config can now also declare an optional hosted project association so
the local loop knows which remote project to use for the later baseline-memory
step.

The StageFlow CLI also supports **remote project management**:

- `stageflow project create`
- `stageflow project list`
- `stageflow project show`
- `stageflow project update`
- `stageflow project delete`
- `stageflow project promote`

Those commands manage named project records on a running StageFlow API for baselines and regression tracking. They are related, but they are not the same execution context: `stageflow project` is still local dev-server scanning, while `stageflow scan --project ...` is still the hosted baseline/regression-memory step. The remote project lifecycle — create, scan, promote a baseline, diff for regressions, and gate CI — is documented in full in [docs/remote.md](remote.md).

## Prerequisites

Before using Project Mode, you need:

1. The `stageflow` CLI installed.
2. A running local StageFlow stack configured to allow private target scans (like `127.0.0.1` or `localhost`).

To start the local stack with the necessary permissions, run the following commands from the StageFlow repository:

```bash
just setup
just images
just dev up local
just dev init local
```

_Note: The `local` environment flag is crucial. It tells the Platform API to permit scanning private loopback targets and configures the job pods to use the host network._

`just images` builds the scanner images used by local runs. On a fresh setup it can take a while, so treat it as part of environment setup rather than a per-scan command.

After the local overlay is up, run `stageflow project init` to scaffold a `.stageflow/config.yaml`, then `stageflow project doctor .` to verify wiring before running a scan.

If you are integrating StageFlow into an automated agent workflow, both commands
also support `--format json` so setup and readiness checks can be consumed as
structured terminal output instead of scraped from text.

## Initialization

To set up Project Mode for your web app, navigate to your project's root directory and run:

```bash
stageflow project init
stageflow project init --format json
```

This command inspects your project (looking for `package.json`, `Justfile`,
etc.) and creates a configuration directory containing:

- `.stageflow/config.yaml` (The main configuration file)
- `.stageflow/README.md` (A quick-start guide)

If StageFlow cannot infer a startup command, the generated config keeps a
placeholder for `dev.start.cmd`. Replace it before running `stageflow project`.

If you already have a hosted project registered on a StageFlow API, keep its
slug in the generated config as the optional `stageflow.remote_project`
association. Add `stageflow.remote_api_url` when the hosted project lives on a
different API base URL. That link powers `stageflow project hosted` and surfaces
in `stageflow project doctor --format json`.

If you run the command from a subdirectory, StageFlow resolves the git root first and creates `.stageflow/` there so the config stays attached to the repository instead of one nested folder.

If you only need copy/paste command examples, the [CLI cheatsheet](operations/cli_cheatsheet.md#5-project-mode) is the shortest companion doc.

## Configuration (`.stageflow/config.yaml`)

The generated `config.yaml` file controls how StageFlow interacts with your project. You will likely need to adjust it to match your specific setup.

Here is an example configuration:

```yaml
version: 1

stageflow:
  api_url: "http://localhost:8080"
  remote_project: "my-frontend" # Optional hosted project slug for follow-up remote scans
  remote_api_url: "https://stageflow.org" # Optional hosted API for the remote project

scan:
  # Set this to the page URLs your dev server serves.
  urls:
    - http://127.0.0.1:5173
  scanners: axe,lighthouse,seo,link-checker
  allow_private_targets: true

dev:
  start:
    # The command to start your dev server
    cmd: ["bun", "run", "dev"]
    cwd: .
  ready:
    # The URL StageFlow will poll to know your app is ready to scan
    url: http://127.0.0.1:5173
```

### Configuration Reference

#### `stageflow`

- `api_url`: The URL of your StageFlow Platform API (defaults to `http://localhost:8080`).
- `api_key_env`: (Optional) The name of the environment variable containing your StageFlow API key, if authentication is required.
- `remote_project`: (Optional for local mode, required for hosted mode) Hosted project slug associated with this repo. `stageflow project hosted` uses it to run the registered project against the configured remote API.
- `remote_api_url`: (Optional) Hosted API base URL for that remote project when it differs from the local Project Mode API.

#### `scan`

- `urls`: A list of URLs to scan. These should point to your local dev server (e.g., `http://127.0.0.1:5173`).
- `scanners`: A comma-separated list of scanners to run (e.g., `axe,lighthouse,seo,link-checker`).
- `screenshot`: (Optional) Boolean indicating whether to capture screenshots during the scan.
- `allow_private_targets`: Must be `true` when scanning local addresses like `127.0.0.1`.
- `timeout`: (Optional) The maximum duration for the scan (e.g., `5m`).

#### `dev`

- `up`: (Optional) A list of commands to run before starting the dev server (e.g., `[["docker-compose", "up", "-d"]]`).
- `start.cmd`: The command array used to start your application (e.g., `["npm", "run", "dev"]`). **Ensure this is split into an array of strings.**
- `start.cwd`: The working directory from which to run the start command (defaults to `.`).
- `start.env`: (Optional) A map of environment variables to inject into the start command.
- `ready.url`: The URL StageFlow will continuously poll. The scan will only begin once this URL returns an HTTP 2xx or 3xx status code.
- `ready.timeout`: (Optional) Maximum time to wait for the readiness URL to respond (e.g., `30s`).
- `ready.interval`: (Optional) Polling interval for the readiness URL (e.g., `1s`).
- `down`: (Optional) A list of commands to run after the dev server shuts down (e.g., `[["docker-compose", "down"]]`).
- `stop.signal`: (Optional) The signal to send to the dev server to stop it (e.g., `SIGTERM`).
- `stop.timeout`: (Optional) The maximum time to wait for the dev server to shut down gracefully before forcing it.

## The Dev Loop

Once configured, you can run a full scan cycle by simply executing:

```bash
stageflow project
```

When you run this command, StageFlow will:

1. Read `.stageflow/config.yaml`.
2. Execute your `dev.start.cmd` in the background.
3. Poll the `dev.ready.url` until your app responds.
4. Submit a scan job to your local StageFlow stack for the targets defined in `scan.urls`.
5. Stream the live progress and results back to your terminal.
6. Automatically send an interrupt signal to cleanly shut down your dev server when the scan finishes.

Recommended terminal-first loop:

```bash
# Local edit / validate / scan loop
stageflow project doctor --format json
stageflow project --format json

# Follow-up hosted baseline-memory step
stageflow project hosted --format json
```

If `.stageflow/config.yaml` includes `stageflow.remote_project`,
`stageflow project hosted` skips the dev-server lifecycle and scans the hosted
project through `stageflow.remote_api_url` or `stageflow.api_url`. The local and
hosted runs remain separate on purpose: the first is fast local validation
against your dev server, the second asks the hosted API for baseline and
regression memory.

## Troubleshooting and Validation

If you are unsure whether your configuration is correct, use the `doctor`
command:

```bash
stageflow project doctor
stageflow project doctor --skip-dev
stageflow project doctor --format json
```

`stageflow project doctor` will:

- Validate your `config.yaml` syntax and structure.
- Verify that the StageFlow API is reachable.
- Start your dev server, wait for the readiness URL to respond, and then shut it down **without** actually running a scan.

With `--format json`, doctor also reports the hosted project association when
one is configured, which makes it easier for agents to carry the local result
into the follow-up `stageflow scan --project ...` step.

This is the best way to debug issues with your `dev.start.cmd` or `dev.ready.url`.

### Common Issues

- **Port already in use**: If your dev server fails to start because a port is occupied, update your `dev.start.cmd` to use a different port (e.g., `["npm", "run", "dev", "--", "--port", "5174"]`) and update your `scan.urls` and `dev.ready.url` to match.
- **ENOENT errors**: If you see an `ENOENT` error when starting the dev server, verify that the first element in your `dev.start.cmd` array is a valid executable in your system's PATH (like `npm`, `bun`, or `just`).
- **Readiness timeouts**: If StageFlow times out waiting for your app, ensure the `dev.ready.url` is exactly what your dev server binds to. Some frameworks bind to `localhost` while others bind to `127.0.0.1` or `0.0.0.0`.
