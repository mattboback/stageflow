/**
 * Canonical aggregated report types.
 *
 * Source of truth:
 * - JSON Schema: `packages/contracts/report/schema/unified-report.v2.schema.json`
 * - Generated TS: `packages/contracts/report/generated/typescript/unified-report.v2.ts`
 *
 * IMPORTANT:
 * - This is the *aggregated job report* (`{jobId}/report.json`).
 * - It is not the per-scanner `results.json` type used inside `scanner-runner`.
 */

import type {
	IssueSeverity,
	IssueDetail,
	LighthouseCategorySummary,
	PageOverviewElement,
	PageSummary,
	SeverityCounts,
	ReportArtifact,
	ReportError,
	ReportMeta,
	ReportSummary,
	ScannerStatus,
	ScannerSummary,
	UnifiedReportV2
} from '../../../../../packages/contracts/report/generated/typescript/unified-report.v2';

export type {
	IssueSeverity,
	IssueDetail,
	LighthouseCategorySummary,
	PageOverviewElement,
	PageSummary,
	SeverityCounts,
	ReportArtifact,
	ReportError,
	ReportMeta,
	ReportSummary,
	ScannerStatus,
	ScannerSummary,
	UnifiedReportV2
};

export type UnifiedReport = UnifiedReportV2;
