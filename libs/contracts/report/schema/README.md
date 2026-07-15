# Unified Report Schema

`unified-report.v2.schema.json` is the source of truth for StageFlow report documents. It accepts compatible `2.x` report versions; current producers may emit a newer minor version than the original `2.0.0` fixtures.

Do not duplicate field definitions here. Read the schema descriptions for the contract and use the committed fixtures as executable examples:

- `../fixtures/unified-report.v2.json` is the minimal report.
- `../fixtures/unified-report.v2.all-scans.json` exercises all built-in scanner shapes and drives web tests.

## Validate

From this directory:

```bash
node validate.js ../fixtures/unified-report.v2.json
node validate.js ../fixtures/unified-report.v2.all-scans.json
```

Or from `libs/contracts/report`:

```bash
bun run validate:fixtures
```

Validation checks both JSON Schema constraints and cross-field integrity such as summary counts, scanner/page references, and issue totals.

## Generate Types

From the repository root:

```bash
just generate-contracts
```

The generator emits Go and TypeScript types used by services and clients. Generated code is a build artifact and is intentionally not committed.

## Versioning

- Preserve compatibility within major version 2 when adding optional fields or enum values supported by existing consumers.
- Use a new major schema for incompatible field or semantic changes.
- Update schema descriptions and fixtures with every contract change.
- Prefer `scannerData` for scanner-specific payloads instead of adding core fields that only one scanner understands.

The [contracts README](../../README.md) describes package ownership and the code-generation flow.
