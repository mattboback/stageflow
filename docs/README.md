# StageFlow Docs

Use this page as the shortest path to the right StageFlow doc.

## Choose your path

### I am evaluating the project for the first time

Start here if you want to decide whether StageFlow is credible, understandable, and worth running:

1. [Repository README](../README.md) for the product overview, screenshots, and local quick start.
2. [Architecture](architecture/system.md) for the service map, trust boundaries, and job flow.
3. [Evaluator guide](evaluators-guide.md) if you are reviewing this as a portfolio project.
4. [CLI README](../clients/cli/README.md) if you want to see how the terminal workflow maps to the platform.

### I want to run StageFlow locally

1. [Repository README](../README.md) for the product overview and local quick start.
2. [Configuration reference](reference/configuration.md) for `.env` variables and environment-specific details.

### I want to use the CLI

1. [CLI README](../clients/cli/README.md) for the main concepts, output formats, and examples.
2. [CLI cheatsheet](operations/cli_cheatsheet.md) for common day-to-day commands.
3. [Generated CLI reference](reference/cli/stageflow/stageflow.md) for the full command tree.

### I want to scan a local app during development

1. [Project mode](PROJECT_MODE.md) for `.stageflow/config.yaml`, readiness checks, and the local dev-server workflow.
2. [CLI cheatsheet](operations/cli_cheatsheet.md#5-project-mode) for copy/paste examples.

### I want to add a new scanner

1. [Scanner manifest schema](../libs/contracts/scanner-manifest/schema/README.md) for the full manifest field reference.
2. [Scanner runner tests](../services/scanner-runner/tests/) for examples of how existing scanners are tested.

## Terminology: Remote vs. local projects

StageFlow uses the word "project" in two related but different ways:

- `stageflow project`, `stageflow project init`, and `stageflow project doctor` are **local project mode** commands. They work from a local `.stageflow/config.yaml`, start your dev server, scan it, and stop it again.
- `stageflow project create`, `list`, `show`, `update`, `delete`, and `promote` are **remote project management** commands. They manage named project records on a running StageFlow API for baselines and regression tracking.

If you keep that distinction in mind, the rest of the docs are much easier to navigate.

## Docs by area

| Doc                                                   | Best for                                                   |
| ----------------------------------------------------- | ---------------------------------------------------------- |
| [README](../README.md)                                | Product overview, screenshots, quick local setup, repo map |
| [Architecture](architecture/system.md)                | System shape, service responsibilities, lifecycle details  |
| [Configuration reference](reference/configuration.md) | Environment variables and deployment/runtime configuration |
| [CLI cheatsheet](operations/cli_cheatsheet.md)        | Common CLI commands and everyday copy/paste examples       |
| [CLI README](../clients/cli/README.md)                | CLI concepts, JSON contract, project mode, quality gates   |
| [Project mode](PROJECT_MODE.md)                       | Local dev-server scanning workflow                         |
| [Developer tools](operations/devtools.md)             | Maintainer and operator tooling beyond the main CLI        |
| [Evaluator guide](evaluators-guide.md)                | How to review StageFlow as a portfolio project             |
