# Project Context

## Vision

StageFlow is a self-hosted, Podman-native web accessibility and quality scanning platform. Users submit a URL or ZIP archive and receive a unified, deduplicated report from multiple scanners (axe, lighthouse, SEO, security-headers, link-checker, ai-navigator) delivered via SSE. The goal is transparency, operational control, and correctness — scans run in isolated per-job pods, and SSRF guardrails are strict. Target users: developers and teams who need auditing infrastructure they control, without giving URLs to third-party SaaS services.

## Current Priorities

1. **Stability and correctness** — job FSM, scanner lifecycle, report aggregation, deduplication
2. **Test coverage** — expand unit + e2e coverage, enforce CI thresholds
3. **Docs and onboarding** — reduce setup friction, fill architecture/ops gaps
4. **Scanner ecosystem** — make plugin authoring clear and well-documented

## Success Metrics

- CI passes cleanly on every push to main
- Contributors can run `just demo` and have a working scan in < 5 minutes
- Test coverage stays above thresholds defined in vitest config
- Zero tracked build artifacts / secrets in git history

## Areas

| Area | Status | Notes |
|------|--------|-------|
| `platform/api/` | Stable | SSRF hardening done; timeout/cancellation hardened in PR#14 |
| `platform/orchestrator/` | Active | Postgres cutover (PR#13); FSM hardening ongoing |
| `platform/extractor/` | Stable | ZIP safety limits in place |
| `platform/scanner-runner/` | Active | Plugin system working; test coverage expanding |
| `frontend/` | Active | SvelteKit 5 runes; presets added in PR#1 |
| `packages/shared-go/` | Stable | Shared models, NATS, MinIO, httputil |
| `packages/contracts/` | Stable | JSON Schema → generated types |
| `infra/` | Stable | Compose, Quadlets, Caddy, Grafana |
| `tests/e2e/` | Growing | Go e2e tests against live stack |
| `docs/` | Needs work | Some gaps in OPERATIONS.md and CONFIGURATION.md |

## Contribution Guidelines

- All code must pass `just ci` (Go build/lint/test + bun lint/typecheck/test)
- Never use `_ = err` in Go or `any`/`as` casts in TypeScript
- Match existing patterns in the module you're editing
- PRs should include tests for new behavior

## Tone

Technical and direct. Friendly but not effusive. Assume contributors are competent. Acknowledge effort and explain decisions clearly.

## Out of Scope

- Adding cloud/SaaS hosting or centralized telemetry
- Supporting Docker as a first-class alternative to Podman (Podman-first)
- GUI scanner plugin builder (CLI/manifest-based only)
