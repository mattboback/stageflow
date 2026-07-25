// The app's single entry point for report contract types. Re-exports only what
// the app actually consumes — ReportArtifact, ReportMeta, and ReportSummary are
// deliberately absent, and should be added back here when something needs them.
import type {
	IssueDetail,
	IssueOccurrence,
	IssueSeverity,
	LighthouseCategorySummary,
	PageOverviewElement,
	PageSummary,
	ReportError,
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
	ReportError,
	ScannerStatus,
	ScannerSummary,
	UnifiedReportV2
};

export type UnifiedReport = UnifiedReportV2;

export interface IssueGroup {
	fingerprint: string;
	ruleId: string;
	scanner: string;
	title: string;
	description: string;
	helpUrl?: string;
	wcagTags?: string[];
	category?: string;
	severity: IssueSeverity;
	occurrences: IssueDetail[];
	pageIds: string[];
	baselineFingerprint?: string;
	status?: 'new' | 'persisted' | 'resolved';
}
