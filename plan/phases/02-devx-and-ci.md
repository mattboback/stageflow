# Phase 02 — Developer Experience & CI Hardening

## Goal

Ensure local development is reliable and CI is a faithful, fast representation of real usage.

## Exit Criteria

- A clean clone can run `just setup` → `just dev up` → `just dev init` → run a scan without surprises.
- CI failures are actionable and high-signal.
- Tests are deterministic and cover core workflows.

## Work Items

- Local dev reliability:
  - Make “first run” paths resilient (clear errors on missing tools, missing env vars, missing Podman capabilities).
  - Reduce the number of manual `.env` edits for local dev (safe defaults and explicit overrides).
  - Document and test Podman rootless assumptions and volume/network behavior.
- CI improvements:
  - Add dedicated jobs for Go, frontend, and integration/E2E to improve feedback speed.
  - Add caching where it helps (Go build cache, Bun install, Playwright browsers if used).
  - Make CI output more readable (grouped logs, clear failure summaries).
- Test strategy:
  - Identify the critical “public promise” flows (URL scan, ZIP scan, SSE progress, report aggregation).
  - Ensure each has at least one stable integration test and one “happy path” E2E test.
  - Add smoke tests for scanner plugin loading and manifest validation.

## Artifacts (Deliverables)

- Developer tooling:
  - A documented “one command local demo” recipe (README + `just` target if appropriate)
  - Optional devcontainer support (`.devcontainer/`) if it materially reduces setup friction
- CI:
  - Expanded `.github/workflows/ci.yml` (or split workflows: `backend.yml`, `frontend.yml`, `e2e.yml`)
  - A short “CI contract” section in `CONTRIBUTING.md` (what must pass and how to run locally)
- Tests:
  - Integration/E2E test coverage for the critical flows above

