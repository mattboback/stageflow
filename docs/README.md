# StageFlow Docs

Use this page as the shortest path to the right StageFlow doc.

## Choose your path

### I am evaluating the project for the first time

Start here if you want to decide whether StageFlow is credible, understandable, and worth running:

1. [Repository README](../README.md) for the product overview, screenshots, and local quick start.
2. [Architecture](architecture/system.md) for the service map, trust boundaries, and job flow.
3. [CLI README](../clients/cli/README.md) if you want to see how the terminal workflow maps to the platform.

### I want to run StageFlow locally

1. [Repository README](../README.md) for the product overview and local quick start.
2. [Configuration reference](reference/configuration.md) for `.env` variables and environment-specific details.
3. [.github/SUPPORT.md](../.github/SUPPORT.md) if setup works partially but something still fails.

### I want to use the CLI

1. [CLI README](../clients/cli/README.md) for the main concepts, output formats, and examples.
2. [CLI cheatsheet](operations/cli_cheatsheet.md) for common day-to-day commands.
3. [Generated CLI reference](reference/cli/stageflow/stageflow.md) for the full command tree.

### I want to scan a local app during development

1. [Project mode](PROJECT_MODE.md) for `.stageflow/config.yaml`, readiness checks, and the local dev-server workflow.
2. [CLI cheatsheet](operations/cli_cheatsheet.md#5-project-mode) for copy/paste examples.
3. [.github/SUPPORT.md](../.github/SUPPORT.md#6-debug-project-mode-separately) if project mode fails before the scan starts.

### I want to contribute or operate the repo

1. [Contributing guide](../.github/CONTRIBUTING.md) for workflow expectations and repo entry points.
2. [Developer tools and operational CLIs](operations/devtools.md) for maintainer-oriented tooling.
3. [Testing docs](testing/storybook-component-testing.md) if you are working in frontend or UI verification flows.

## Terminology: Remote vs. local projects

StageFlow uses the word “project” in two related but different ways:

- `stageflow project`, `stageflow project init`, and `stageflow project doctor` are **local project mode** commands. They work from a local `.stageflow/config.yaml`, start your dev server, scan it, and stop it again.
- `stageflow project create`, `list`, `show`, `update`, `delete`, and `promote` are **remote project management** commands. They manage named project records on a running StageFlow API for baselines and regression tracking.

If you keep that distinction in mind, the rest of the docs are much easier to navigate.

## Docs by area

| Doc | Best for |
| --- | --- |
| [README](../README.md) | Product overview, screenshots, quick local setup, repo map |
| [Architecture](architecture/system.md) | System shape, service responsibilities, lifecycle details |
| [Configuration reference](reference/configuration.md) | Environment variables and deployment/runtime configuration |
| [CLI cheatsheet](operations/cli_cheatsheet.md) | Common CLI commands and everyday copy/paste examples |
| [CLI README](../clients/cli/README.md) | CLI concepts, JSON contract, project mode, quality gates |
| [Project mode](PROJECT_MODE.md) | Local dev-server scanning workflow |
| [Developer tools](operations/devtools.md) | Maintainer and operator tooling beyond the main CLI |
| [Contributing](../.github/CONTRIBUTING.md) | How to work in the repo and what validation is expected |
| [Support](../.github/SUPPORT.md) | Troubleshooting flow and issue-routing guidance |

## Examples and deeper reads

- [AI quality gate case study](case-study-ai-quality-gate.md) — current capabilities plus next-step automation ideas
- [Multi-site monitoring case study](case-study-multi-site-monitoring.md) — current project/baseline flow plus proposed monitoring features
- [Storybook component testing](testing/storybook-component-testing.md)
