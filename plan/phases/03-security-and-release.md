# Phase 03 — Security, Releases, and Supply Chain

## Goal

Treat StageFlow like a real public project: clear security posture, dependency hygiene, and a repeatable release pipeline.

## Exit Criteria

- Security reporting path exists and is linked everywhere appropriate.
- Dependency/vulnerability scanning runs in CI.
- Releases are versioned, tagged, and reproducible.

## Work Items

- Security posture and hardening:
  - Validate SSRF defenses and URL/ZIP trust boundaries (size limits, timeouts, network controls).
  - Ensure scanner containers run with least privilege (capabilities, filesystem, network, seccomp/apparmor where applicable).
  - Document threat model assumptions and operational guidance for running in untrusted environments.
- Supply-chain hygiene:
  - Add automated dependency update tooling (Go + Bun).
  - Add vulnerability scanning (Go vuln checks, JS vuln checks) and fail CI on high severity where appropriate.
  - Consider SBOM generation for releases and container images.
- Release process:
  - Decide versioning policy (`v0.x` until stable; semantic versioning expectations).
  - Automate release creation (GitHub Releases), changelog updates, and build artifacts.
  - If publishing container images, define registries, tags, and signing policy.

## Artifacts (Deliverables)

- Security:
  - `SECURITY.md` (if not already complete)
  - A short threat model / operational hardening doc (in `docs/` or `ARCHITECTURE.md`)
- Automation:
  - `.github/dependabot.yml` (or an equivalent dependency update mechanism)
  - Code scanning workflow (CodeQL or equivalent)
  - Vulnerability scan workflow(s) for Go and Bun ecosystems
- Releases:
  - `CHANGELOG.md` kept up to date
  - Release workflow(s) that build and attach artifacts
  - Container publishing workflow(s) (optional, if you want public images)

