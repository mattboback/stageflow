# Contributing to StageFlow

Thanks for taking an interest in StageFlow. This repository contains the product itself: the web app, CLI, backend services, shared contracts, and the local development stack used to work on them.

Small, focused fixes are welcome. If you are planning a larger feature, architecture change, or workflow shift, start with an issue or draft pull request first so the work can be scoped before you invest heavily.

## Start here

These docs give the fastest orientation:

- `docs/README.md` for the docs map and fastest path to the right guide
- `README.md` for the product overview, screenshots, and local quick start
- `docs/architecture/system.md` for service boundaries and data flow
- `docs/reference/configuration.md` for environment variables and runtime configuration
- `docs/operations/devtools.md` for repo tooling and contributor workflows

## Repo map

Use this map to find the right entry point:

- `clients/web` — SvelteKit frontend
- `clients/cli` — `stageflow` CLI
- `services/platform-api` — intake API, SSE stream, and report APIs
- `services/orchestrator` — job lifecycle and scanner orchestration
- `services/scanner-runner` — scanner runtime and Playwright-based scanners
- `libs/contracts` — shared schemas and generated contracts
- `devtools` — development, QA, and operational helpers used from this repo
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

For the default local stack:

```bash
just setup
just dev up
just dev init
just images
```

For localhost or other private-target scans, use the local overlay:

```bash
just setup
just dev up local
just dev init local
just images
```

`just images` builds the scanner images used by local scans. Expect the first run to take the longest.

After startup, the main local endpoints are:

| Service                | `dev` mode (default)    | `local` overlay mode    |
| ---------------------- | ----------------------- | ----------------------- |
| Frontend               | `http://localhost:3000` | `http://localhost:3010` |
| Platform API           | `http://localhost:8080` | `http://localhost:8080` |
| Orchestrator Admin API | `http://localhost:8081` | `http://localhost:8081` |

## Choose the right stack

| If you want to... | Use | Why |
| --- | --- | --- |
| Work against public URLs during normal development | `just dev up` | Default local stack, simplest path |
| Scan `localhost`, `127.0.0.1`, or other private targets | `just dev up local` | Enables the private-target path required for loopback scanning |
| Verify a repo-managed staging-style environment | `just staging up` | Separate compose stack for staging verification |

## Working in the repo

- Keep pull requests scoped. It is easier to review a focused scanner, API, UI, or docs change than a mixed refactor.
- Update the relevant docs and contracts in the same pull request when behavior crosses service boundaries.
- Include screenshots for meaningful UI changes and terminal snippets for notable CLI changes.
- If you touch scanner execution, intake validation, or report schemas, explain the behavior change clearly in the PR description.

## Validation before review

Run the broad quality gate before opening a pull request:

```bash
just ci
```

When iterating on a narrower area, these targeted commands are useful:

```bash
just storybook-test
just shell-tests
```

If you only need one surface locally:

```bash
just run clients/web
just run storybook
just run api
just run orchestrator
```

And for stack inspection:

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

## Pull request expectations

A pull request is ready for review when all of the following are true:

- the relevant local commands succeed
- new behavior is covered by tests or clearly justified manual verification
- user-facing docs are updated when behavior changes
- screenshots are included for meaningful UI changes
- CLI output examples are refreshed when command behavior changes

Every PR should make it easy for a reviewer to answer three questions quickly:

1. What changed?
2. How was it validated?
3. Are there follow-ups, limits, or rollout notes to keep in mind?

## Pre-commit hooks

This repo uses `pre-commit` (the Python-based framework) as the canonical hook workflow to ensure formatting, secrets, and commit-message checks run before push. Do not use Husky or other JS-based hook managers.

If you use `pre-commit`, install the hooks locally:

```bash
pre-commit install
pre-commit install --hook-type commit-msg
```

## Reporting bugs or asking questions

- Search existing issues first: <https://github.com/mattboback/stageflow/issues>
- Use the issue templates for reproducible bugs and feature requests
- Use [SUPPORT.md](SUPPORT.md) for troubleshooting steps and docs links
- Follow [SECURITY.md](SECURITY.md) for private security disclosures

## Production boundary

Do not add or depend on repo-local production deployment commands for the live `stageflow.org` environment. Production operations are intentionally managed from the external deployment workspace described in `AGENTS.md`.

## Code of Conduct

By participating in this project, you agree to follow [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
