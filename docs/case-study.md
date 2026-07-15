# Engineering Case Study

StageFlow is a portfolio-scale example of owning a product from browser interaction through distributed execution, storage, contracts, operations, and release engineering. This case study highlights decisions that can be verified directly in the repository rather than relying on private metrics or unsupported claims.

## The Problem

Frontend quality checks usually arrive as unrelated outputs: an accessibility report, a Lighthouse score, broken-link logs, SEO warnings, and screenshots stored somewhere else. That fragmentation makes review slow and makes regression gating brittle.

StageFlow turns those tools into one workflow:

1. Accept a live URL or static-site archive.
2. Run isolated scanners against the same target and provenance.
3. Normalize their results into one versioned report.
4. Put evidence, remediation, and human review decisions together.
5. Compare registered projects with a promoted baseline so CI can distinguish existing debt from new regressions.

## Decisions and Tradeoffs

### One report contract, not scanner-specific clients

JSON Schema is the source of truth for report, event, provenance, and scanner-manifest contracts. Generated Go and TypeScript types keep services and clients aligned, while committed fixtures serve as executable examples.

The tradeoff is generation overhead: a fresh checkout must generate ignored language bindings before focused builds. In return, scanners can evolve behind a stable client-facing shape and contract drift becomes a CI failure.

### Events for work, HTTP and SSE for users

The Platform API accepts work over HTTP, publishes lifecycle events through NATS JetStream, and exposes progress through SSE. The Orchestrator owns durable job transitions and scanner coordination.

SSE was chosen because progress is predominantly server-to-client and does not need a bidirectional WebSocket protocol. The current fanout is deliberately single-instance; horizontal scaling would require a shared subscription/fanout layer.

### Explicit state transitions

Jobs move through a small state machine rather than being inferred from whichever artifact exists. Domain transition rules are shared, and SQL guards prevent delayed or replayed events from moving a job backward.

This adds ceremony to every lifecycle change, but makes retries, terminal failures, progress, and operational inspection predictable.

### Treat every target as hostile

URL intake classifies resolved addresses, the browser runtime repeats target checks across navigation and subresources, ZIP extraction enforces path, size, entry-count, and compression-ratio limits, and scanners run in rootless per-job Podman pods.

These controls reduce risk but do not eliminate DNS rebinding. The repository documents that residual boundary and expects public operators to add network egress controls rather than presenting application validation as a complete sandbox.

### Stable identity for useful baselines

Issue identifiers are derived from stable content such as rule, page, and element context. A promoted report can therefore distinguish new findings from unchanged debt without depending on array order or runtime IDs.

The tradeoff is that identity rules become part of product behavior: normalization changes must be tested against committed golden results.

## Verification Strategy

The repository uses layers that match the risks:

- Unit and race tests for Go services and state policies.
- Contract fixtures validated across generated Go and TypeScript types.
- Scanner lifecycle and browser-integration tests.
- React component and Playwright user-flow tests.
- Real PostgreSQL and NATS integration coverage.
- A Golden Regression workflow that creates a project, scans, promotes a baseline, introduces a known defect, rescans, and proves the CLI exits non-zero with the expected normalized diff.
- Secret, dependency, vulnerability, dead-code, container, SBOM, and image checks in CI.

Start with the [code tour](code-tour.md) for a short path through the corresponding source, or read the [architecture](architecture.md) for the complete data flow and trust boundaries.

## What This Demonstrates

- Full-stack product judgment: the browser, CLI, API, orchestration, storage, and reports form one coherent user workflow.
- Backend and platform design: durable events, explicit state, isolated execution, contracts, retention, metrics, and operational failure modes.
- Frontend quality: responsive evidence-heavy reports, keyboard interaction, accessibility gates, live progress, and recovery states.
- Security reasoning: explicit trust boundaries, scoped credentials, hostile-input handling, documented residual risks, and fail-closed public configuration.
- Delivery discipline: reproducible setup, generated references, release artifacts, CI gates, and end-to-end regression proof.

The project intentionally does not claim production scale or private business impact. Its value as evidence is that the important decisions, limitations, and verification are inspectable.
