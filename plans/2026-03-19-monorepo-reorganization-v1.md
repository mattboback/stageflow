# StageFlow Repository Reorganization Plan

## Goals
- Simplify the top-level directory structure.
- Consolidate fragmented domains (like QA, Testing, and Tooling).
- Adopt consistent naming for user-facing applications vs backend services.
- Make it easier for developers to locate code and understand architectural boundaries.
- Minimize disruption to existing CI/CD and deployment processes (`justfile` and `go.work` updates required).

## Current Structure Analysis
Currently, the repo is divided into:
- `clients/` (`cli/stageflow`, `web`)
- `devtools/` (`ops/job-status-cli`, `qa/suite-runner`, `scripts`)
- `services/` (`platform-api`, `orchestrator`, `scanner-runner`, `archive-extractor`)
- `libs/` (`contracts`, `go/*`)
- `qa/` (`e2e`)
- `infra/`, `docs/`, `plans/`

**Pain Points Identified:**
1. **Fragmented Tooling:** Internal dev tools are split between `devtools/ops` and `devtools/qa`, while tests exist in `qa/`.
2. **Ambiguous `clients` vs `services`:** `cli` and `web` are "clients", but they are essentially the user-facing "apps" of the platform.
3. **Deep Nesting:** The CLI tool is nested three levels deep (`clients/cli/stageflow`).

## Proposed New Structure

### 1. `apps/` (Replaces `clients/`)
User-facing applications and binaries.
- `apps/web/` (from `clients/web`)
- `apps/cli/` (from `clients/cli/stageflow`)

### 2. `services/` (Keep mostly as is)
Backend APIs, background workers, and system processes.
- `services/platform-api/`
- `services/orchestrator/`
- `services/scanner-runner/`
- `services/archive-extractor/`

### 3. `tools/` (Replaces `devtools/`)
Internal developer scripts and utility binaries not shipped to users.
- `tools/job-status-cli/` (from `devtools/ops/job-status-cli`)
- `tools/scripts/` (from `devtools/scripts`)

### 4. `qa/` (Consolidated Quality Assurance)
All end-to-end testing, test runners, and test suites.
- `qa/e2e/` (keep as is)
- `qa/suite-runner/` (from `devtools/qa/suite-runner`)

### 5. `libs/` (Keep mostly as is)
Shared packages, contracts, and internal modules.
- `libs/contracts/`
- `libs/go/` (We can keep this language-specific nesting since the repo uses both Go and TypeScript, or flatten if desired).

### 6. Keep `infra/`, `docs/`, `plans/` as is.

## Migration Steps

### Phase 1: Directory Moves
1. `mkdir -p apps/cli tools qa/suite-runner`
2. `mv clients/web apps/web`
3. `mv clients/cli/stageflow/* apps/cli/` (and remove empty `clients` dir)
4. `mv devtools/ops/job-status-cli tools/`
5. `mv devtools/qa/suite-runner/* qa/suite-runner/`
6. `mv devtools/scripts tools/`
7. `rm -rf devtools`

### Phase 2: Configuration Updates
1. **`go.work`**: Update all moved Go module paths:
   - `./clients/cli/stageflow` -> `./apps/cli`
   - `./devtools/ops/job-status-cli` -> `./tools/job-status-cli`
   - `./devtools/qa/suite-runner` -> `./qa/suite-runner`
2. **`justfile`**:
   - Update `web_dir := 'apps/web'`
   - Update `run` recipe paths for `clients/web` to `apps/web`
   - Update `build` and `cli-install` recipe paths
   - Update shell regression script paths to `tools/scripts/...`
3. **Dockerfile/CI configurations**: Scan `infra/` for any hardcoded paths to `clients/web` or `clients/cli` and update them.

### Phase 3: Validation
- Run `just setup` to ensure `go.work` syncs.
- Run `just ci` to execute all formatting, linting, tests, and builds.
- Run `just images` to verify container builds succeed with the new layout.

## Risk Mitigation
- Create the moves in a single atomic Git commit to preserve Git history (`git mv`).
- Make sure to update the root `README.md` and any documentation referencing old paths.
- Ensure the production deployment scripts (managed externally as per `AGENTS.md`) don't rely on exact internal folder paths, or if they do, coordinate the updates.