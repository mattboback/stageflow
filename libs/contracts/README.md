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
axe accessibility violations, Lighthouse scores, broken links, or social
metadata. That single contract is what lets one CLI renderer, one
web UI, and one baseline-diff engine serve all seven scanners.

Committed fixtures are contract documentation you can execute:

- [`report/fixtures/unified-report.v2.all-scans.json`](report/fixtures/unified-report.v2.all-scans.json)
  — a full multi-scanner report, validated against the schema. It drives the
  web client's component tests and e2e smoke.

## Codegen flow

```
schema/*.schema.json ──► just generate-contracts ──► Go types (libs/go/…)
                                        └──────────► TS types (clients/web, scanner-runner)
```

Change a schema, regenerate, and every consumer typechecks against the new
shape — contract drift fails the build rather than surfacing at runtime.

## Why the four packages are not identical

They share what should be shared: all four are `private`, all four validate their
fixtures, and all four are covered by the drift hook
(`devtools/scripts/precommit/run.mjs`) and by CI. The differences that remain are
consequences of what each package *is*, and are listed here so the next reader does
not "fix" one of them:

- **`events` has no `generated/`, `tsconfig.json`, or `exports`.** It defines NATS
  envelopes consumed from Go, so there is no TypeScript package to build and
  nothing to generate. Its only job is to validate fixtures against the schema.
- **`events/schema/validate.mjs` is `.mjs`; the other three are `.js`.** It is the
  one validator written as ESM, and no package sets `"type": "module"`, so the
  extension is required rather than stylistic.
- **`provenance` has no `generate:go`.** It emits TypeScript only. It used to write
  a Go module too, but `go.work` never included it and nothing imported it, so it
  was never compiled — see the note in `devtools/scripts/generate-contracts.sh`.
- **`report` nests its `go.mod` under `generated/go/`, `scanner-manifest` keeps one
  at the package root.** `report`'s Go is generated from the schema;
  `scanner-manifest` additionally ships a hand-written Go validator
  (`validator.go`) that `libs/go/scannercatalog` imports. Different kinds of code,
  so different placement.
