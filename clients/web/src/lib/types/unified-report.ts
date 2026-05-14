/**
 * Canonical aggregated report types.
 *
 * Source of truth:
 * - JSON Schema: `libs/contracts/report/schema/unified-report.v2.schema.json`
 * - Generated TS: `libs/contracts/report/generated/typescript/unified-report.v2.ts`
 *
 * IMPORTANT:
 * - This is the *aggregated job report* (`{jobId}/report.json`).
 * - It is not the per-scanner `results.json` type used inside `scanner-runner`.
 */

import type {
	IssueDetail,
	IssueOccurrence,
	IssueSeverity,
	LighthouseCategorySummary,
	PageOverviewElement,
	PageSummary,
	ReportArtifact,
	ReportError,
	ReportMeta,
	ReportSummary,
	ScannerStatus,
	ScannerSummary,
	SeverityCounts,
	UnifiedReportV2
} from '../../../../../libs/contracts/report/generated/typescript/unified-report.v2';

export type {
	IssueSeverity,
	IssueDetail,
	IssueOccurrence,
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

/**
 * An aggregated group of issue occurrences that share the same rule fingerprint.
 * Sentry-style "fix this rule" rollup designed for both the grouped issue list
 * and future diff-scan support (compare runs, surface new/persisted/resolved).
 */
export interface IssueGroup {
	/** Stable identifier: `${scanner}:${ruleId}`. Survives reordering of inputs. */
	fingerprint: string;
	ruleId: string;
	scanner: string;
	title: string;
	description: string;
	helpUrl?: string;
	wcagTags?: string[];
	category?: string;
	/** Highest severity observed across all occurrences in the group. */
	severity: IssueSeverity;
	/** Original flat issues nested under the group. */
	occurrences: IssueDetail[];
	/** Distinct page ids touched by occurrences in this group. */
	pageIds: string[];
	/** Reserved for future diff-scan feature; not populated by current pipeline. */
	baselineFingerprint?: string;
	/** Reserved for future diff-scan feature; not populated by current pipeline. */
	status?: 'new' | 'persisted' | 'resolved';
}
