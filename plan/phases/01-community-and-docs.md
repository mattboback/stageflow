# Phase 01 — Community & Documentation

## Goal

Make the repository self-explanatory for new users and contributors, with clear expectations and a polished “front door”.

## Exit Criteria

- “What is this, who is it for, how do I run it” is obvious from the README and docs.
- Contribution flow is predictable (issues, PRs, review expectations, and communication channels).
- Documentation reflects real behavior and stays close to the code.

## Work Items

- Tighten the README for first-run success:
  - Confirm prerequisites are minimal and accurate.
  - Ensure quick start matches current `just` recipes exactly.
  - Add a “minimal local demo flow” (submit a URL → watch SSE → view report).
- Add/standardize community health files:
  - Security policy and support policy (if not done in Phase 00).
  - Changelog + release notes conventions (even if the first release is `v0.x`).
- Add GitHub issue and PR templates with required fields:
  - Repro steps, environment, logs, expected vs actual.
  - Security-related reports explicitly routed to the security policy.
- Add a “docs map” so information is easy to find (either keep root docs or introduce a `docs/` folder intentionally, but avoid duplicates).

## Artifacts (Deliverables)

- Root docs:
  - `SECURITY.md`
  - `SUPPORT.md`
  - `CHANGELOG.md` (Keep a stable structure; start at `v0.1.0` or `Unreleased`)
- GitHub community:
  - `.github/ISSUE_TEMPLATE/bug_report.yml`
  - `.github/ISSUE_TEMPLATE/feature_request.yml`
  - `.github/ISSUE_TEMPLATE/question.yml` (or direct to Discussions if enabled)
  - `.github/PULL_REQUEST_TEMPLATE.md`
- Optional (recommended):
  - `.github/labels.yml` (or documented label taxonomy)
  - `CODEOWNERS` (only if you want enforced review ownership)

