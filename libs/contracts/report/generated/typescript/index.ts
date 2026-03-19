/**
 * @stageflow/contracts-report
 *
 * Generated TypeScript types from JSON Schema.
 * DO NOT MODIFY - regenerate with `bun run generate:ts`
 */

// Export all generated types
export * from "./unified-report.v2";

// Export validation functions
export {
	validateReport,
	isValidReport,
	assertValidReport,
	parseReport,
	type ValidationResult,
	type ValidationError,
	type IntegrityError,
} from "./validator";

// Canonical, stable name for the aggregated report type.
// Keep `UnifiedReportV2` as the schema/versioned name; point `UnifiedReport` at the latest.
export type { UnifiedReportV2 as UnifiedReport } from "./unified-report.v2";

// NOTE: Avoid legacy aliases like `ScanResults`/`ScanArtifact` here to keep the
// canonical aggregate report type name unambiguous.
