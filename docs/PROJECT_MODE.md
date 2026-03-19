# StageFlow Project Mode

Project Mode is a powerful feature of the `stageflow` CLI that integrates web accessibility and quality scanning directly into your local development workflow.

Instead of manually starting your app and running scans against public URLs, Project Mode automates the entire lifecycle: it starts your local dev server, waits for it to become ready, runs the StageFlow scanners against your local environment, streams the results, and then gracefully shuts down the dev server.

## Prerequisites

Before using Project Mode, you need:
1. The `stageflow` CLI installed.
2. A running local StageFlow stack configured to allow private target scans (like `127.0.0.1` or `localhost`).

To start the local stack with the necessary permissions, run the following commands from the StageFlow repository:

```bash
just dev up local
just dev init local
just images
```

*Note: The `local` environment flag is crucial. It tells the Platform API to permit scanning private loopback targets and configures the job pods to use the host network.*

## Initialization

To set up Project Mode for your web application, navigate to your project's root directory and run:

```bash
stageflow project init
```

This command inspects your project (looking for `package.json`, `Justfile`, etc.) and scaffolds a configuration directory containing:
- `.stageflow/config.yaml` (The main configuration file)
- `.stageflow/README.md` (A quick-start guide)

## Configuration (`.stageflow/config.yaml`)

The generated `config.yaml` file controls how StageFlow interacts with your project. You will likely need to adjust it to match your specific setup.

Here is an example configuration:

```yaml
version: 1

stageflow:
  api_url: "http://localhost:8080"

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

## Troubleshooting and Validation

If you are unsure whether your configuration is correct, you can use the `doctor` command:

```bash
stageflow project doctor
```

The doctor command will:
- Validate your `config.yaml` syntax and structure.
- Verify that the StageFlow API is reachable.
- Start your dev server, wait for the readiness URL to respond, and then shut it down **without** actually running a scan.

This is the best way to debug issues with your `dev.start.cmd` or `dev.ready.url`.

### Common Issues

- **Port already in use**: If your dev server fails to start because a port is occupied, update your `dev.start.cmd` to use a different port (e.g., `["npm", "run", "dev", "--", "--port", "5174"]`) and update your `scan.urls` and `dev.ready.url` to match.
- **ENOENT errors**: If you see an `ENOENT` error when starting the dev server, verify that the first element in your `dev.start.cmd` array is a valid executable in your system's PATH (like `npm`, `bun`, or `just`).
- **Readiness timeouts**: If StageFlow times out waiting for your app, ensure the `dev.ready.url` is exactly what your dev server binds to. Some frameworks bind to `localhost` while others bind to `127.0.0.1` or `0.0.0.0`.
