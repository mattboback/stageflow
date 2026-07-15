# Dependency Security Exceptions

StageFlow keeps dependency-audit exceptions narrow, documented, and temporary.
An exception never disables the surrounding audit threshold: every advisory
other than the exact identifier listed below continues to fail CI.

## GHSA-8988-4f7v-96qf

| Field | Value |
| --- | --- |
| Status | Temporarily accepted |
| Reviewed | 2026-07-15 |
| Owner | StageFlow maintainers |
| Dependency path | `lighthouse` → `@sentry/node` → OpenTelemetry 1.x |
| CI scope | Scanner-runner moderate audit only |
| Removal trigger | Remove as soon as a compatible Lighthouse release adopts the patched OpenTelemetry line, or sooner if StageFlow begins invoking the affected behavior. |

Dependabot checks the scanner-runner weekly. When Lighthouse updates, remove
the `--ignore=GHSA-8988-4f7v-96qf` argument from CI, refresh the lockfile, and
verify that an unfiltered `bun audit --audit-level=moderate` passes.
