# StageFlow Open-Source Readiness Plan

This folder contains a phased, artifact-driven plan to bring StageFlow to a professional, public open-source release standard.

## Current State (Already Strong)

StageFlow already has many “open-source baseline” pieces in place:

- `LICENSE` (MIT)
- `README.md` with architecture + quick start
- `ARCHITECTURE.md` with system design, flows, and schemas
- `CONTRIBUTING.md` and `CODE_OF_CONDUCT.md`
- GitHub Actions CI (`.github/workflows/ci.yml`) covering Go + frontend
- `justfile` for a consistent dev/operator CLI surface
- Example environment files (`.env.example`, `.env.staging.example`)

## Definition of Done (Public OSS “Professional”)

StageFlow is considered ready for open source when:

- A new developer can run a local stack end-to-end with minimal manual setup.
- The repo has clear community expectations (support channels, security reporting, contribution workflow).
- CI is reproducible, fast, and enforces quality gates (lint, tests, type-check, vuln scans).
- Security posture is documented and defaults are safe (SSRF, sandboxing, limits, secrets handling).
- Releases are versioned and repeatable (changelog, tags, artifacts, container images if applicable).
- Private/operator-specific details are separated from public docs (no personal paths/domains or secrets).

## Phases

1. Phase 00 — Audit & Public/Private Split: `plan/phases/00-audit-and-sanitize.md`
2. Phase 01 — Community & Documentation: `plan/phases/01-community-and-docs.md`
3. Phase 02 — Developer Experience & CI Hardening: `plan/phases/02-devx-and-ci.md`
4. Phase 03 — Security, Releases, and Supply Chain: `plan/phases/03-security-and-release.md`

## Artifact Drafts

`plan/artifacts/` includes copy-ready draft files for common open-source “community health” and GitHub templates. The intent is to review and move the approved versions into the repository root / `.github/` when executing the phases.

## Tracking

Recommended workflow:

- Create GitHub milestones matching the phases above.
- Convert each “Work item” into an issue with a clear acceptance checklist.
- Land changes phase-by-phase to keep PRs small and reviewable.
