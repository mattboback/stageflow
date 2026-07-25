# Dependency Exceptions

StageFlow keeps dependency exceptions narrow, documented, and temporary. An
audit exception never disables the surrounding audit threshold: every advisory
other than the exact identifier listed below continues to fail CI. A version
ceiling is always scoped to the workspaces that actually need it, never applied
repository-wide.

## Version Ceilings

These are recorded as `ignore` entries in [`.github/dependabot.yml`](../.github/dependabot.yml)
so the bot stops reopening upgrades that cannot merge.

### `typescript` — pinned below 6.1 in ESLint-linted workspaces

| Field | Value |
| --- | --- |
| Status | Upstream-blocked |
| Reviewed | 2026-07-25 |
| Owner | StageFlow maintainers |
| Scope | `clients/web`, `services/scanner-runner` (both pin `~6.0.3`) |
| Not scoped to | `libs/contracts/{report,provenance,scanner-manifest}`, which run `^7.0.2` |
| Removal trigger | Remove once `@typescript-eslint/typescript-estree` widens its `typescript` peer range past `<6.1.0`. |

`@typescript-eslint/typescript-estree` declares `typescript: ">=4.8.4 <6.1.0"`.
Installing TypeScript 7 in either workspace makes type-aware linting fail at
`typescript-estree/dist/create-program/shared.js` with
`TypeError: Cannot read properties of undefined (reading 'Cjs')`, because the
native TypeScript 7 port no longer exposes the compiler internals the parser
reads. The three contract packages have no ESLint configuration, so they are
already on 7.x and are deliberately outside this ceiling.

Note that `react-router` is *not* the blocker. It was, at 8.2.0
(`typescript: ^5.1.0 || ^6.0.0`), but 8.3.0 widened its peer to include
`^7.0.0`.

### `@types/node` — pinned to the Node runtime major

| Field | Value |
| --- | --- |
| Status | Intentional, indefinite |
| Reviewed | 2026-07-25 |
| Owner | StageFlow maintainers |
| Scope | `clients/web`, `services/scanner-runner` (both pin `24`) |
| Removal trigger | Not a defect to remove. Raise the pin whenever the Node runtime major moves — the two must always match. |

Type definitions describe the runtime they are compiled against, so
`@types/node` must equal the Node major that `.node-version`, every
`engines.node` field, and `NODE_VERSION` in the workflows declare. Accepting a
major bump here independently would typecheck the code against a Node API
surface that neither CI nor the containers provide.

StageFlow targets the **Active LTS** line rather than the newest release: Node
22 entered maintenance in October 2025, and Node 26 does not reach LTS until
October 2026.

## Security Advisories

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
