# Dependency Security Exceptions

StageFlow keeps dependency-audit exceptions narrow, documented, and temporary.
An exception never disables the surrounding audit threshold: every advisory
other than the exact identifier listed below continues to fail CI.

## GHSA-8988-4f7v-96qf

| Field | Value |
| --- | --- |
| Status | Temporarily accepted |
| Reviewed | 2026-07-16 |
| Owner | StageFlow maintainers |
| Dependency path | `lighthouse` → `@sentry/node` → OpenTelemetry 1.x |
| CI scope | Scanner-runner moderate audit only; exact `--ignore GHSA-8988-4f7v-96qf` waiver |
| Removal trigger | Remove as soon as a compatible Lighthouse/Sentry release adopts OpenTelemetry 2.8 or later. |

The advisory affects OpenTelemetry baggage propagation before 2.8. StageFlow's
scanner does not accept or propagate inbound OpenTelemetry baggage, and Node's
HTTP header limits provide an additional bound. Forcing `@opentelemetry/core`
2.8 into Sentry's 1.30.x graph is not a safe patch: it breaks Sentry at runtime.
The compatible 1.30.1 line is therefore retained and guarded by a Sentry
initialization smoke test until the upstream dependency graph can be upgraded.

Dependabot checks the scanner-runner weekly. When Lighthouse/Sentry updates,
remove the exact ignore from CI and `just ci`, refresh the lockfile, and verify
that an unfiltered `bun audit --audit-level=moderate` passes.
