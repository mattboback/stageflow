# Contributing to StageFlow

Thanks for contributing to StageFlow. This repository contains the application source, local development workflows, and staging tooling for the StageFlow platform.

Before you start, read these docs to understand the repo shape and operating model:

- `README.md` for the product overview and local quick start
- `docs/architecture/system.md` for service boundaries and data flow
- `docs/reference/configuration.md` for environment and runtime configuration
- `docs/operations/devtools.md` for CLI and developer tooling

## What lives where

Use this map to find the right entry point:

- `clients/web` — SvelteKit frontend
- `clients/cli` — `stageflow` CLI
- `services/platform-api` — intake API, SSE stream, report APIs
- `services/orchestrator` — job lifecycle and scanner orchestration
- `services/scanner-runner` — scanner runtime and Playwright-based scanners
- `libs/contracts` — shared schemas and generated contracts
- `devtools` — internal ops, QA, and contributor tooling
- `qa` — end-to-end and integration-style verification assets

## Development prerequisites

Install the same core tools used by the repo:

- Go 1.26.1
- Bun
- Podman with `podman compose`
- `just`
- `golangci-lint`

Then create your local environment file:

```bash
cp .env.example .env
```

## Local setup

For a normal local development stack:

```bash
just setup
just dev up
just dev init
just images
```

For localhost or other private-target scans, use the local overlay instead:

```bash
just setup
just dev up local
just dev init local
just images
```

After startup:

- Frontend: `http://localhost:3000`
- Platform API: `http://localhost:8080`
- Orchestrator admin API: `http://localhost:8081`

## Common workflows

### Run the full quality gate

Before opening a pull request, run the same broad validation flow used for local CI:

```bash
just ci
```

### Run targeted checks

Use these when iterating on a specific area:

```bash
just storybook-test
just shell-tests
```

### Run individual services

These commands are useful when you only need one surface locally:

```bash
just run clients/web
just run storybook
just run api
just run orchestrator
```

### Inspect the local stack

```bash
just dev logs
just dev down
just dev restart
```

Use the same commands with the `local` environment when you are working with private-target scans:

```bash
just dev logs local
just dev down local
just dev restart local
```

## Choosing a place to work

- Work in `clients/web` for UI, UX, Storybook, and frontend tests.
- Work in `clients/cli` for CLI behavior, project mode, and report rendering.
- Work in `services/platform-api` for intake validation, job submission, and streaming/report APIs.
- Work in `services/orchestrator` for job state transitions and runner coordination.
- Work in `services/scanner-runner` for scanner execution and browser automation behavior.
- Work in `libs/contracts` when API and report schemas change.

If a change crosses boundaries, update the relevant docs and contracts in the same pull request.

## Pull request expectations

A change is ready for review when all of the following are true:

- the relevant local commands succeed
- new behavior is covered by tests or clearly justified manual verification
- user-facing docs are updated when behavior changes
- screenshots are included for meaningful UI changes
- CLI output examples are refreshed when command behavior changes

When you open a pull request, include:

- a short summary of what changed
- the areas touched (`clients/web`, `clients/cli`, `services/*`, `libs/contracts`, docs)
- validation steps you ran
- follow-up work or known limitations

## Pre-commit hooks

This repo includes a `.pre-commit-config.yaml` with formatting, secrets, and commit-message checks. If you use `pre-commit`, install the hooks locally:

```bash
pre-commit install
pre-commit install --hook-type commit-msg
```

## Reporting bugs or asking questions

- Search existing issues first: <https://github.com/mattboback/stageflow/issues>
- Use `SUPPORT.md` when you need troubleshooting or doc links
- Follow `SECURITY.md` for private security disclosures

## Production boundary

Do not add or depend on repo-local production deployment commands for the live VPS. Production operations for `stageflow.org` are intentionally managed from the external deployment workspace described in `AGENTS.md` and the root deployment strategy.

## Code of Conduct

By participating in this project, you agree to follow `CODE_OF_CONDUCT.md`.
