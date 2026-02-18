# AGENTS.md — packages/contracts

JSON Schema → generated multi-language type bindings. No business logic. Never edit `generated/` directly.

## STRUCTURE

```
packages/contracts/
├── scanner-manifest/
│   ├── schema/scanner-manifest.schema.json  ← source of truth
│   ├── scanner_manifest.go                  ← generated Go (DO NOT EDIT)
│   ├── generated/typescript/                ← generated TS (DO NOT EDIT)
│   └── Makefile                             ← `make` regenerates both
├── report/
│   ├── generated/go/                        ← generated Go (DO NOT EDIT)
│   ├── generated/typescript/                ← generated TS + validator (DO NOT EDIT)
│   └── Makefile
└── events/fixtures/                         ← example event payloads for tests
```

## CONVENTIONS

- **Never edit `generated/`** — all changes go via schema → `make`.
- Change scanner manifest shape: edit `scanner-manifest/schema/scanner-manifest.schema.json` → `make`.
- Change report shape: edit report schema → `make` in `report/`.
- Generated Go files carry `// Code generated ... DO NOT EDIT.` header.
- Generated TS files carry `/* eslint-disable */` header — intentional.
- `scanner-runner` copies TS types via `bun run prepare:contracts` (in `platform/scanner-runner/scripts/`).

## CONSUMERS

| Consumer | Uses |
|----------|------|
| `platform/scanner-runner` | TS scanner-manifest + TS report types (via prepare:contracts) |
| `packages/shared-go` | Go report types (go.work) |
| `platform/orchestrator` | Go scanner-manifest types (via shared-go/scannercatalog) |
| `frontend` | TS report validator |
