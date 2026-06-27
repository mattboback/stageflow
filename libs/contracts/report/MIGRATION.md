# Migration Guide: Unified Report Schema v2

This document tracks the move from hand-maintained report types to generated
types from the JSON Schema. The StageFlow repo has completed this migration;
use this guide for external consumers or new code.

## Current Source of Truth

- JSON Schema: `libs/contracts/report/schema/unified-report.v2.schema.json`
- Generated code: run `just generate-contracts` from the repo root.

### Repo Status (Jan 2026)

- `libs/go/models/results.go` has been removed.
- Services now import generated Go types from
  `github.com/mattboback/stageflow/libs/contracts/report/generated/go`.
- TypeScript consumers use `@stageflow/contracts-report` (or local generated
  files where bundlers do not load workspace packages directly).

---

## TypeScript Usage

### Imports

```typescript
import type { UnifiedReportV2, IssueDetail } from "@stageflow/contracts-report";
```

When working inside this repo, `services/scanner-runner` uses a build step to
copy generated types into `node_modules/@stageflow/contracts-report`:

```
services/scanner-runner/scripts/prepare-contracts-report-types.mjs
```

The StageFlow frontend currently imports directly from the generated file:

```
libs/contracts/report/generated/typescript/unified-report.v2.ts
```

## Go Usage

```go
import report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
```

### Differences (Old vs Generated)

- `ScanResults` → `UnifiedReportV2`
- `PageSummary.PageID` → `PageSummary.Id`
- `IssueDetail.HelpURL` → `IssueDetail.HelpUrl`
- `IssueDetail.PageID` → `IssueDetail.PageId`
- Enum fields now use generated constants:
  - `IssueSeverityCritical`, `IssueSeveritySerious`, `IssueSeverityModerate`,
    `IssueSeverityMinor`, `IssueSeverityInfo`
  - `ScannerStatusSuccess`, `ScannerStatusFailed`, `ScannerStatusSkipped`
  - `ErrorScopeScanner`, `ErrorScopePage`, `ErrorScopeGlobal`
  - `UserImpactSeverityBlocking`, `UserImpactSeverityDegraded`,
    `UserImpactSeverityInconvenient`

### Optional Fields

Generated types use pointers for optional fields. Plan for `nil`:

```go
if issue.HelpUrl != nil {
    log.Println(*issue.HelpUrl)
}
```

---

## Validation

### JSON Unmarshal (Go)

The generated Go types validate required fields and enum values during
`json.Unmarshal`:

```go
var report UnifiedReportV2
if err := json.Unmarshal(data, &report); err != nil {
    return err
}
```

### Schema CLI

```bash
node libs/contracts/report/schema/validate.js <file.json>
cd libs/contracts/report && bun run validate:fixtures
```

---

## Migration Checklist (External Consumers)

- Update imports to generated types
- Replace string comparisons with enum constants
- Handle pointer fields for optional values
- Validate fixtures (Go unmarshal or schema CLI)

---

## Rollback Plan

If issues arise during migration:

1. Keep old types available during transition
2. Use build tags to switch between implementations:
   ```go
   //go:build use_generated_types
   ```
3. Revert imports if needed
4. The schema and generated types remain available for future migration attempts

---

## Historical Migration Sequence

The in-repo migration is complete; this is the order it was carried out, kept as
a reference for external consumers adopting the contract in their own codebase:

1. Set up the contracts package and update TypeScript imports to `@stageflow/contracts-report`.
2. Generate the Go types and update the API/orchestrator services to consume them.
3. Update frontend consumers and remove any temporary adapter layer.
4. Delete the old hand-maintained type files and verify all tests pass.
