# Unified Report Schema v2.0.0

## Overview

This is the **single source of truth** for the Unified Scan Report schema. All TypeScript and Go type definitions are generated from this JSON Schema.

The Unified Report format aggregates results from multiple accessibility and quality scanners (Axe, Lighthouse, etc.) into a single, consistent structure.

## Schema Location

- **JSON Schema**: `unified-report.v2.schema.json` (this directory)
- **Validation Script**: `validate.js` (this directory)
- **Fixtures**: `../fixtures/` directory

## Schema Version

- **Version**: 2.0.0
- **Format**: JSON Schema Draft-07
- **Compatibility**: TypeScript, Go, JavaScript

## Key Features

### 1. Comprehensive Type Safety

- All fields are strongly typed with validation rules
- Required fields are enforced
- Format validation for URIs, dates, and patterns
- Enum constraints for status and severity values

### 2. Data Integrity Validation

The validation script (`validate.js`) performs additional checks beyond schema validation:

- **Count Consistency**: Ensures `summary.totalIssues` matches `issues.length`
- **Severity Accuracy**: Validates `summary.bySeverity` matches actual issue counts
- **Reference Integrity**: Verifies all scanner/page/artifact IDs are valid references
- **Page Counts**: Confirms `pagesScanned` matches `pages.length`

### 3. Scanner Extensibility

Supports any number of scanners through:
- Generic `scannerData` field for scanner-specific metadata
- Flexible `category` field for issue categorization
- Standardized severity normalization

## Schema Structure

```
UnifiedReportV2 (Root)
├── version: string (semver)
├── meta: ReportMeta
│   ├── jobId: string (required)
│   ├── baseUrl?: string (URI)
│   ├── scannedAt?: string (ISO 8601)
│   ├── completedAt?: string (ISO 8601)
│   └── durationMs?: number
├── summary: ReportSummary
│   ├── score?: number (0-100)
│   ├── scoreGrade?: string (A-F)
│   ├── totalIssues: number (required)
│   ├── bySeverity: SeverityCounts (required)
│   ├── byScanner?: Record<string, number>
│   ├── pagesScanned: number (required)
│   ├── pagesWithIssues: number (required)
│   └── lighthouseCategories?: LighthouseCategorySummary[]
├── scanners: ScannerSummary[] (required)
├── pages: PageSummary[] (required)
├── issues: IssueDetail[] (required)
├── artifacts?: ReportArtifact[]
└── errors?: ReportError[]
```

## Usage

### Validating a Report

```bash
# Using the validation script
node validate.js /path/to/report.json

# Using ajv-cli directly
npx ajv-cli validate \
  -s unified-report.v2.schema.json \
  -d /path/to/report.json \
  --strict=false
```

### Validation Exit Codes

- `0`: Valid with no data integrity issues
- `1`: Invalid schema or data integrity violations

### Example Output

```bash
✅ unified-report.v2.json is valid

# Or with issues:
❌ report.json is invalid

Validation errors:
  • /summary/totalIssues: must be integer
  • /issues/0/severity: must be equal to one of the allowed values

⚠️  Data integrity warnings:
   - summary.totalIssues (10) does not match issues.length (9)
   - Issues reference non-existent scanner IDs: custom-scanner
```

## Type Definitions

### Core Types

#### IssueSeverity (enum)
- `critical` - Blocking accessibility or critical failures
- `serious` - Major issues affecting many users
- `moderate` - Moderate impact issues
- `minor` - Minor issues or improvements
- `info` - Informational items

#### UserGroup (enum)
User groups affected by accessibility issues:
- `blind` - Screen reader users
- `low-vision` - Low vision users
- `motor` - Motor disability users
- `cognitive` - Cognitive disability users
- `deaf` - Deaf or hard of hearing users
- `vestibular` - Vestibular disorder users
- `all` - All users

#### Scanner Status (enum)
- `success` - Scanner completed successfully
- `failed` - Scanner failed to complete
- `skipped` - Scanner was skipped

### Key Interfaces

#### IssueDetail

Represents a specific issue found by a scanner.

**Required Fields:**
- `id` - Unique issue identifier
- `scanner` - Scanner that detected this issue
- `ruleId` - Scanner-specific rule ID
- `severity` - Normalized severity level
- `title` - Short issue title
- `description` - Detailed description
- `pageId` - Reference to page
- `pageUrl` - Page URL
- `elementCount` - Number of affected elements

**Optional Fields:**
- `severityRaw` - Original scanner severity before normalization
- `helpUrl` - Link to documentation
- `wcagTags` - WCAG success criteria (e.g., ["wcag2aa", "1.4.3"])
- `occurrences` - List of specific DOM elements
- `category` - Issue category (accessibility, performance, security, etc.)
- `friendlyNode` - User-friendly element metadata
- `locationInfo` - Scroll position information
- `userImpact` - Description of user impact
- `howToFix` - Remediation guidance
- `scannerData` - Scanner-specific additional data

#### LighthouseCategorySummary

**Added in v2.0.0** - Previously missing from Go models.

Represents aggregated Lighthouse category scores:
- `id` - Category ID (performance, accessibility, best-practices, seo, pwa)
- `title` - Human-readable category name
- `avgScore` - Average score across pages (0.0 to 1.0)

## Schema Improvements (v2.0.0)

### Fixed Issues

1. ✅ **Added LighthouseCategorySummary**: Previously only in TypeScript, now in schema
2. ✅ **True Optionality**: Nullable types properly specified with `["type", "null"]`
3. ✅ **Data Integrity**: Custom validators for business logic constraints
4. ✅ **Consistent Naming**: Single schema eliminates TypeScript/Go discrepancies

### Breaking Changes from v1.x

- None - v2.0.0 is backward compatible with existing data
- New validation rules may catch previously undetected inconsistencies

## Code Generation

### TypeScript

```bash
just generate-contracts
```

Generated types use:
- Interfaces (not types) for better compatibility
- JSDoc comments from schema descriptions
- Proper optional fields

### Go

```bash
just generate-contracts
```

Generated code uses:
- Pointer types for optional fields (`*int`, `*string`)
- Proper struct tags for JSON marshaling
- Validation method stubs

## Migration Guide

### From Separate TypeScript/Go Schemas

**Before:**
```typescript
// Multiple definitions
import { ScanArtifact } from '../types/scan-results';
import { UnifiedReportV2 } from '../contracts/unified-report.v2';
```

**After:**
```typescript
// Single source
import { UnifiedReportV2 } from '@stageflow/contracts-report';
```

Prefer the stable alias name `UnifiedReport` for application code (it currently points at `UnifiedReportV2`).

### Updating Go Models

**Before:**
```go
type ReportSummary struct {
    Score      int    `json:"score,omitempty"`  // ❌ defaults to 0
}
```

**After:**
```go
type ReportSummary struct {
    Score      *int   `json:"score,omitempty"`  // ✅ truly optional
}
```

## Testing

### Fixture Files

Two comprehensive fixtures demonstrate the schema:

1. **unified-report.v2.json** - Minimal valid report
2. **unified-report.v2.all-scans.json** - Full example with all scanner types

### Continuous Integration

Add to your CI pipeline:

```yaml
# .github/workflows/validate-schema.yml
- name: Validate Schema
  run: |
    bun add -d ajv-cli ajv-formats
    node libs/contracts/report/schema/validate.js \
      libs/contracts/report/fixtures/*.json
```

## Best Practices

### Creating Reports

1. **Use Validation Early**: Validate during generation, not just at the end
2. **Maintain Data Integrity**: Ensure counts match before serializing
3. **Include Metadata**: Always populate timing and version info
4. **Reference Carefully**: Verify all IDs exist before referencing

### Extending the Schema

1. **Use `scannerData`**: For scanner-specific fields, don't modify schema
2. **Version Bumps**: Increment version for breaking changes
3. **Backward Compatibility**: Maintain compatibility within major versions

### Performance

- Keep `occurrences` arrays reasonable (<100 items per issue)
- Use `elementsTruncated` flag when limiting occurrences
- Consider artifact size when using `dataUri` (prefer `path`)

## Support

### Issues Found in This Schema

Please report via:
- GitHub Issues: https://github.com/mattboback/stageflow/issues
- Label: `schema`, `contracts`

### Questions

Check existing documentation:
- Migration guide: `libs/contracts/report/MIGRATION.md`
- Type definitions: See generated code comments

## License

Part of the Stageflow project. See root LICENSE file.

## Changelog

### v2.0.0 (2026-01-02)

- **Added**: LighthouseCategorySummary type definition
- **Added**: Comprehensive validation script with data integrity checks
- **Added**: JSON Schema as single source of truth
- **Fixed**: Optional field handling (proper nullable types)
- **Improved**: Documentation and examples
- **Improved**: Format validation for URIs and dates
