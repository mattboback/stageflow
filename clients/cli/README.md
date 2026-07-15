# StageFlow CLI

The Go CLI submits scans to a StageFlow API, streams progress, renders reports, manages project baselines, and returns stable exit codes for CI.

User workflows live in the [CLI guide](../../docs/cli.md). Exact commands and flags live in the [generated reference](../../docs/reference/cli/stageflow/stageflow.md).

The default scanner selection is `scanners: [axe, lighthouse, seo, link-checker]`.

## Build and Test

From the repository root:

```bash
just generate-contracts
go build -o /tmp/stageflow ./clients/cli
go test ./clients/cli/...
```

Install the current checkout with `just cli-install`.

## Source Layout

| Path | Responsibility |
| --- | --- |
| `main.go` / `run.go` | Process entry point and testable `run(args, getenv, stdout, stderr)` |
| `cobra_*.go`, `scan_*.go`, `project_run.go`, `dev_*.go` | Cobra command tree and scan, project, dev, diff, and stack workflows |
| `internal/apiclient` | Platform API transport and DTOs |
| `internal/authintake` | Storage-state and form-recipe validation |
| `internal/jobstream` | SSE progress with polling fallback |
| `internal/projectmode` | Repository config, bootstrap detection, and dev-process lifecycle |
| `internal/render` | Report filtering and text, Markdown, and JSON output |
| `internal/stack` | Local Podman Compose lifecycle |
| `internal/staticsite` | Directory and ZIP packaging |
| `internal/urlcheck` | Target normalization and private-target policy |

Keep Cobra constructors focused on flags and dependency wiring; put reusable policy and transport code in focused packages under `internal`.

## Behavioral Contracts

CLI changes must preserve or deliberately version:

- exit codes: `0` pass, `1` quality gate, `2` CLI/API error;
- progress on stderr and report output on stdout;
- versioned JSON envelope schemas;
- private-target and authenticated-scan safety checks;
- SSE fallback behavior;
- API path escaping and report filtering semantics.

Run `just project-golden` when project scans, baseline diffs, report normalization, or exit decisions change.

## Generated Reference

Command docs are generated from Cobra and checked for drift:

```bash
go run ./clients/cli docs --out-dir docs/reference/cli/stageflow
```

Do not edit generated command pages by hand.
