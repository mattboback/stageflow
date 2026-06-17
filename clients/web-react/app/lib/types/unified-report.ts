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
