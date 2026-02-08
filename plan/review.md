# Repo Review (Open-Source Readiness)

This review summarizes what’s already in good shape for a public repository and what remains to reach a “professional open-source” standard.

## What’s Already Strong

- Clear project identity and scope in `README.md` (problem statement, architecture, quick start, structure).
- Deep technical documentation in `ARCHITECTURE.md` (flows, schemas, security notes).
- Community baseline files already present: `LICENSE`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`.
- CI exists and covers both stacks (Go workspace + frontend) via `.github/workflows/ci.yml`.
- Standardized local workflow via `justfile` with sensible grouping (`setup`, `dev`, `staging`, `quality`).
- Config templates exist (`.env.example`, `.env.staging.example`) and `.env*` is ignored by git.

## Gaps to Close for a Public OSS Release

### Community & Project Hygiene

- Add missing community health files commonly expected by contributors:
  - Security reporting policy (`SECURITY.md`)
  - Support policy (`SUPPORT.md`)
  - Changelog and release notes conventions (`CHANGELOG.md`)
- Add GitHub issue/PR templates to raise the quality of incoming reports and reduce triage overhead.
- Decide whether to enable GitHub Discussions for Q&A (if not, route questions to Issues explicitly).

### Public vs Private Operations

- Keep personal/VPS/operator-only instructions out of the public “mainline” docs.
- Ensure any deployment guide is generic, reproducible, and does not depend on personal paths, hosts, or credentials.

### Security & Trust Boundaries (High-Impact for This Domain)

StageFlow’s core function (scanning arbitrary URLs and processing uploaded HTML) is security-sensitive by default.

- Make the “safe defaults” story obvious:
  - URL validation and SSRF controls
  - ZIP extraction and path traversal protections
  - Timeouts, size limits, and resource limits
  - Container sandboxing and least-privilege execution for scanner pods
- Add a clear threat model summary and operational guidance for running against untrusted inputs.

### CI Depth and Supply Chain

- Expand CI beyond build/lint/test:
  - Dependency update automation (Go + Bun)
  - Vulnerability scanning for both ecosystems
  - Optional code scanning (CodeQL or equivalent)
  - SBOM generation for release artifacts (optional but increasingly expected)

### Releases

- Define versioning and compatibility expectations (especially around scanner plugin contracts).
- Automate release creation (tags, changelog updates, artifacts).
- Decide if you want to publish container images, and if so: registries, tag strategy, and signing.

## Quick Wins (High Value / Low Risk)

- Land community health files (`SECURITY.md`, `SUPPORT.md`, `CHANGELOG.md`).
- Add issue templates + PR template.
- Add dependency update tooling (Dependabot or equivalent) and a basic vuln scan in CI.
- Document a single “happy path demo” that proves the product in <10 minutes on a clean machine.

## Open Decisions to Make Explicit

- Supported platforms (Linux-only vs Linux/macOS; rootless Podman expectations).
- Whether Docker is supported (officially, unofficially, or not at all).
- Whether to publish public container images and where.
- Whether to accept scanner plugins from third parties as a stable extension point, and the compatibility policy for plugin schemas.

