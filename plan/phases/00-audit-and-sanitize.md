# Phase 00 — Audit & Public/Private Split

## Goal

Identify anything that would block public release (secrets, private ops details, unclear licensing) and make the public repo “safe to clone”.

## Exit Criteria

- No secrets in git history or the current tree.
- No personal infrastructure details required to understand the project.
- Third-party licensing and data privacy implications are understood and documented.

## Work Items

- Run automated secret scans on the full git history (and document remediation steps if anything is found).
- Inventory public-facing configuration surfaces (env vars, config files, CLI flags) and ensure examples are safe and generic.
- Identify and move private/operator-only notes (personal VPS paths, internal hostnames, private dashboards) into a private location that is not part of the public repo.
- Validate that all bundled scanners and major dependencies are compatible with `LICENSE` (MIT) and that any required attribution is handled.
- Confirm the project name, module path, and repo location are final (or document a migration plan if they will change).

## Artifacts (Deliverables)

- Root docs (public):
  - `SECURITY.md` (vulnerability reporting + supported versions)
  - `SUPPORT.md` (where questions go; what is/isn’t supported)
- GitHub config:
  - `.github/ISSUE_TEMPLATE/` (bug report, feature request, question)
  - `.github/PULL_REQUEST_TEMPLATE.md`
- Compliance and hygiene:
  - A documented “secrets & ops split” policy (can live in `CONTRIBUTING.md` or a short `docs/` page)
  - License inventory output and a decision on attribution approach (tooling choice recorded in an issue)

## Notes / Review Findings (Initial)

- Baseline OSS docs exist (`LICENSE`, `README.md`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `ARCHITECTURE.md`).
- There are environment templates in-tree; ensure they stay example-only and never include real credentials.

