# Repo Fix Plan 2026-04-06

This plan is derived from `docs/reference/repo-audit-2026-04-06.md`.

## Objectives

- make the repo tooling story internally consistent
- remove confirmed dead code and dead tests
- reduce duplication in the frontend realtime scan flow
- clean up small but persistent maintenance drift

## Highest-Value Fixes

### 1. Resolve Hook System Drift

Why this is first:

- it affects day-to-day development behavior
- the repo currently contains conflicting signals across `package.json`, `bun.lock`, `.pre-commit-config.yaml`, and `CHANGELOG.md`
- it is the clearest correctness and maintainability issue at the repo root

Tasks:

- choose the canonical hook system
- remove stale configuration for the losing path
- align `bun.lock` with `package.json`
- update contributor-facing docs to match the chosen setup

Verification:

- root manifest and lockfile agree
- docs reference one hook path only
- hook bootstrap instructions are explicit and reproducible

### 2. Delete Confirmed Dead Files

Why this is second:

- low effort
- immediate reduction in noise and maintenance burden

Tasks:

- delete `devtools/scripts/tests/verify-justfile.test.sh`
- delete `clients/web/src/lib/components/CopyEmailButton.svelte`
- delete `clients/web/src/lib/components/MermaidDiagram.svelte` if no consumer is added first
- remove `mermaid` from `clients/web/package.json` if the component is deleted

Verification:

- no imports remain
- web install/build/test still pass

### 3. Consolidate `scan-status` and `scan-report`

Why this is third:

- it removes duplicated logic in a critical runtime flow
- it reduces future bug-fix cost for job streaming and state updates

Tasks:

- extract shared scan job stream behavior into a shared helper/module
- keep report-only retry/polling logic separate
- preserve existing UX and log messaging behavior

Verification:

- existing store tests pass
- no observable regression in scan status transitions

### 4. Clean Up Small Structural Drift

Tasks:

- remove unused `replace` directives from `devtools/qa/suite-runner/go.mod`
- replace `cloneStringSlice` with `CloneStrings` in `services/platform-api/internal/status`
- optionally remove the unused `Separator` barrel export if still unreferenced

Verification:

- Go build/test passes for touched modules

### 5. Deduplicate CI

Why this is last:

- useful, but less important than correctness and dead-code cleanup
- easier to do after repo tooling is settled

Tasks:

- remove unused `just` install step from `.github/workflows/ci.yml`
- centralize tool versions where practical
- reduce repeated setup blocks and repeated `go-work-dirs.sh` loops

Verification:

- workflow behavior remains unchanged
- YAML gets smaller and easier to maintain

## Recommended Implementation Sequence

1. tooling truth
2. dead-file deletions
3. small drift cleanup
4. web store consolidation
5. CI deduplication

## Notes

- `spec/` and `CODE_OF_CONDUCT.md` are intentionally not in the first-wave fix list.
- They are low-value candidates, but deleting them is a product/documentation choice rather than a correctness fix.
