# Contracts (`libs/contracts`)

JSON Schema contracts are the spine of StageFlow: every scanner, service, and
client agrees on these shapes, and the Go and TypeScript types used across the
monorepo are **generated from them** (`just generate-contracts`, also run in
CI and `just setup`). Generated code is intentionally not committed.

| Contract           | What it defines                                                            | Consumed by                                    |
| ------------------ | -------------------------------------------------------------------------- | ---------------------------------------------- |
| `report`           | The unified report (v2): summary, scanners, pages, issues, artifacts       | CLI output, web report UI, orchestrator        |
| `scanner-manifest` | Scanner metadata: id, aliases, resources, capabilities (+ Go validator)    | scanner catalog/registry, orchestrator, runner |
| `provenance`       | The per-job scan input: pages/URLs, auth, options handed to scanner pods   | platform-api, orchestrator, scanner-runner     |
| `events`           | NATS JetStream event envelopes for the job lifecycle                       | platform-api, orchestrator                     |

## The report contract

`report/schema/unified-report.v2.schema.json` is the single shape every
scanner's findings are aggregated into, whatever the scanner measures —
axe accessibility violations, Lighthouse scores, broken links, or LLM
navigation results. That single contract is what lets one CLI renderer, one
web UI, and one baseline-diff engine serve all eight scanners.

Committed fixtures are contract documentation you can execute:

- [`report/fixtures/unified-report.v2.all-scans.json`](report/fixtures/unified-report.v2.all-scans.json)
  — a full multi-scanner report, validated against the schema. It drives the
  web client's component tests and e2e smoke, and the report-preview QA
  harness renders the real UI from it.
- `report/MIGRATION.md` records the v1 → v2 migration.

## Codegen flow

```
schema/*.schema.json ──► just generate-contracts ──► Go types (libs/go/…)
                                        └──────────► TS types (clients/web, scanner-runner)
```

Change a schema, regenerate, and every consumer typechecks against the new
shape — contract drift fails the build rather than surfacing at runtime.
