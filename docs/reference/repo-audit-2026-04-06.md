# Repo Audit 2026-04-06

This document records a corrected follow-up audit of repo clutter, tooling drift, and architectural cleanup opportunities.

It intentionally separates:

- Confirmed repo issues
- Local workspace clutter that is safe to delete but is not tracked in git
- Earlier claims that did not hold up after direct verification
- Highest-value fixes in recommended order

## Scope

Verified directly against the current repo state, including:

- root config files
- GitHub workflows
- `justfile`
- `clients/web`
- `services/*`
- `devtools/*`
- selected docs and support files

## Corrected Summary

### Confirmed High-Value Issues

1. Hook/tooling configuration is internally inconsistent.
   - Evidence:
     - root `package.json` still contains `precommit:*` scripts
     - `.pre-commit-config.yaml` is present and `CHANGELOG.md` says pre-commit replaced Husky
     - root `bun.lock` still lists `husky` and `lint-staged` under the root workspace even though root `package.json` does not
     - this clone has `core.hooksPath=.husky/_`, but `.husky/` does not exist
   - Files:
     - `package.json`
     - `bun.lock`
     - `.pre-commit-config.yaml`
     - `CHANGELOG.md`
   - Impact:
     - the repo sends mixed signals about the canonical hook system
     - the root lockfile is stale relative to the manifest
     - local hook behavior is easy to misconfigure

2. One devtools test script is dead.
   - Evidence:
     - `devtools/scripts/tests/verify-justfile.test.sh` targets `scripts/verify-justfile.sh`
     - that target script does not exist anywhere in the repo
     - the test script is not referenced by CI or the `justfile`
   - Files:
     - `devtools/scripts/tests/verify-justfile.test.sh`
   - Impact:
     - dead test surface
     - misleading maintenance burden

3. Two web components are dead code, and one likely keeps an unnecessary dependency alive.
   - Evidence:
     - `CopyEmailButton.svelte` has no consumers
     - `MermaidDiagram.svelte` has no consumers
     - `clients/web/package.json` still includes `mermaid`
   - Files:
     - `clients/web/src/lib/components/CopyEmailButton.svelte`
     - `clients/web/src/lib/components/MermaidDiagram.svelte`
     - `clients/web/package.json`
   - Impact:
     - dead UI surface
     - extra dependency and maintenance cost

4. `scan-status` and `scan-report` duplicate core job-stream logic.
   - Evidence:
     - both stores independently implement job fetch, SSE startup, SSE update handling, log message generation, and scanner-progress application
     - both call the same helpers (`createSSEStream`, `buildApiUrl`, `getLogMessage`, `normalizeStatus`, `applySseUpdate`, `applyScannerCompletionUpdate`, `normalizeScannerProgress`)
   - Files:
     - `clients/web/src/lib/stores/scan-status.svelte.ts`
     - `clients/web/src/lib/stores/scan-report.svelte.ts`
   - Impact:
     - duplicated bug surface in a critical realtime flow
     - harder future changes to job status and streaming behavior

5. Web coverage thresholds only measure a narrow handpicked subset of the app.
   - Evidence:
     - `vitest.config.ts` only includes a few specific components plus `src/lib/utils/**/*.ts`
   - File:
     - `clients/web/vitest.config.ts`
   - Impact:
     - reported coverage does not reflect actual frontend coverage

### Confirmed Medium-Value Issues

6. CI contains clear duplication and a small unused step.
   - Evidence:
     - repeated Bun setup/install blocks across `clients_web`, `clients_web_storybook`, and `scanner-runner`
     - repeated `while read dir` loops for Go build, lint, test, and vulncheck
     - `ci.yml` installs `just` and then runs the shell test directly instead of using it
   - File:
     - `.github/workflows/ci.yml`
   - Impact:
     - noisy workflow maintenance
     - higher drift risk between jobs

7. `suite-runner` carries unused module replacement entries.
   - Evidence:
     - `devtools/qa/suite-runner/go.mod` contains many `replace` directives for internal libraries
     - no Go source in that module imports those libraries
   - File:
     - `devtools/qa/suite-runner/go.mod`
   - Impact:
     - stale module metadata
     - misleading dependency surface

8. `platform-api` has a small but real duplicate helper.
   - Evidence:
     - `cloneStringSlice` in `internal/status/model.go`
     - `CloneStrings` in `internal/status/slices.go`
     - both implement the same copy-or-nil behavior
   - Files:
     - `services/platform-api/internal/status/model.go`
     - `services/platform-api/internal/status/slices.go`
   - Impact:
     - low-level duplication in a shared internal package

9. Go service startup and logging patterns are inconsistent.
   - Evidence:
     - `platform-api` uses `bootstrap.SetupLogging` and `run() error`
     - `orchestrator` uses `bootstrap.SetupLogging` and `run() int`
     - `archive-extractor` still uses the legacy `log` package directly in `main.go`
   - Files:
     - `services/platform-api/cmd/server/main.go`
     - `services/orchestrator/cmd/orchestrator/main.go`
     - `services/archive-extractor/cmd/server/main.go`
   - Impact:
     - inconsistent service entrypoint conventions
     - weaker structured logging in `archive-extractor`

10. Root and package formatter metadata drift.

- Evidence:
  - root `.prettierrc` sets `useTabs: false`
  - many tracked JSON manifests currently use tabs
  - Prettier versions differ across root, `clients/web`, and `services/scanner-runner`
- Files:
  - `.prettierrc`
  - `package.json`
  - `clients/web/package.json`
  - `services/scanner-runner/package.json`
- Impact:
  - formatting churn risk
  - unclear canonical formatting source

### Low-Value or Judgment-Call Cleanup

11. `spec/` is tracked, referenced, and intentionally present, but still looks like low-value process ceremony.

- Evidence:
  - `spec/spec-process-cicd-*.md` are tracked
  - `README.md` explicitly documents `spec/`
- Files:
  - `spec/`
  - `README.md`
- Assessment:
  - this is not accidental clutter
  - deleting it would be a product/documentation choice, not a correctness fix

12. `CODE_OF_CONDUCT.md` is generic, but its removal is optional.

- Assessment:
  - low-value for a solo-maintainer repo
  - not a defect by itself

## Local Workspace Clutter

These items currently exist in the working tree and are safe to delete locally, but they are already ignored and are not tracked repo issues.

- `cli`
- `.cache/`
- `plans/`
- `clients/web/.svelte-kit/`
- `clients/web/coverage/`
- `clients/web/storybook-static/`
- `services/scanner-runner/coverage/`
- `services/scanner-runner/dist/`

Important correction: these are local ignored artifacts in this clone, not committed repo clutter.

## Earlier Claims That Did Not Hold Up

1. `scanner-runner` does not have zero tests.
   - False.
   - The repo has extensive tests under `services/scanner-runner/tests/`.

2. The build and coverage artifacts above are committed clutter.
   - False in this clone.
   - They are ignored local artifacts, not tracked files.

3. `.pre-commit-config.yaml` should simply be deleted.
   - Too strong.
   - The repo still signals an intended pre-commit migration. The actual issue is configuration drift, not the mere presence of the file.

4. `.editorconfig`, `go.work.sum`, `CHANGELOG.md`, `CONTRIBUTING.md`, and `SECURITY.md` should be treated as straightforward clutter.
   - Not supported by the repo state.
   - They are tracked, coherent, and actively relevant enough to keep unless the maintainer wants a more aggressive docs reduction.

5. `tsconfig.strict.json`, `.gitleaks.toml`, and `go.work` might be unused.
   - False.
   - `tsconfig.strict.json` is extended by `services/scanner-runner/tsconfig.json` and copied in its Dockerfile.
   - `.gitleaks.toml` supports the CI gitleaks job.
   - `go.work` is the active Go workspace definition.

## Highest-Value Fixes

Recommended order:

1. Normalize the hook and root JS tooling story.
   - Choose one canonical hook path: pre-commit or Husky.
   - Make the repo state consistent with that choice.
   - Regenerate or prune the root `bun.lock` so it matches `package.json`.
   - Update docs/changelog language to match reality.

2. Delete confirmed dead files.
   - Remove `devtools/scripts/tests/verify-justfile.test.sh`.
   - Remove `clients/web/src/lib/components/CopyEmailButton.svelte`.
   - Remove `clients/web/src/lib/components/MermaidDiagram.svelte` if there is no near-term consumer.
   - Remove `mermaid` from `clients/web/package.json` if the component is deleted.

3. Consolidate realtime scan-store logic.
   - Extract shared job-stream behavior out of `scan-status` and `scan-report`.
   - Keep the report-specific polling/retry logic layered on top.

4. Clean up stale module metadata and tiny duplications.
   - Remove unused `replace` directives from `devtools/qa/suite-runner/go.mod`.
   - Replace `cloneStringSlice` with `CloneStrings` inside `platform-api/internal/status`.

5. Simplify CI after the correctness fixes above land.
   - Remove unused `just` installation from `ci.yml`.
   - Centralize repeated Go/Bun versions.
   - Reduce repeated loop/setup blocks.

## Fix Plan

### Phase 1: Tooling Truth

- Decide whether pre-commit is the canonical hook system.
- If yes:
  - remove Husky/lint-staged residue from the root lockfile and docs
  - document `pre-commit install` in contributor setup
- If no:
  - remove `.pre-commit-config.yaml` and root precommit scripts
  - add a real `.husky/` setup to the repo
- Regenerate the root `bun.lock` after the decision so it matches the manifest.

### Phase 2: Dead-Code Sweep

- Delete `verify-justfile.test.sh`.
- Delete unused web components.
- Remove `mermaid` if no remaining consumer exists.
- Optionally remove the unused `Separator` barrel export if it remains unused.

### Phase 3: Web Store Consolidation

- Extract shared helpers for:
  - status fetch
  - SSE setup
  - SSE update application
  - log message accumulation
  - scanner-progress normalization
- Keep `scan-report` responsible only for:
  - aggregated report fetch/retry
  - screenshot refresh
  - polling fallback behavior specific to the report page

### Phase 4: Metadata and Workflow Cleanup

- Trim `suite-runner/go.mod` to the dependencies it really uses.
- Replace the local duplicate string-slice helper in `platform-api`.
- Remove the unused `just` install step from CI.
- Decide later whether `spec/` and `CODE_OF_CONDUCT.md` still earn their place.

## Suggested Execution Order

If fixing this in small, high-leverage commits:

1. hook strategy + root lockfile alignment
2. dead-file deletions
3. `suite-runner` module cleanup + `platform-api` helper cleanup
4. web store refactor
5. CI deduplication
