import type { ScannerSelection } from './types/scan';
import { incompleteScanners } from './report/scanner-status';
import type { IssueDetail, UnifiedReport } from './types/unified-report';

export const LOCAL_PROJECT_SCHEMA_VERSION = 1 as const;
export const LOCAL_PROJECT_DIFF_SCHEMA = 'stageflow/diff@v1' as const;

export interface LocalProjectConfiguration {
	urls: string[];
	scanners: ScannerSelection[];
	/**
	 * Always 'chromium': scans run on one engine. The key stays in the stored
	 * shape and in the fingerprint so saved baselines keep matching.
	 */
	browser: 'chromium';
	/** Always 'solid'; kept in the stored shape and fingerprint like `browser`. */
	highlightStyle: 'solid';
}

export interface LocalProject {
	id: string;
	schemaVersion: typeof LOCAL_PROJECT_SCHEMA_VERSION;
	name: string;
	configuration: LocalProjectConfiguration;
	createdAt: string;
	updatedAt: string;
}

export interface LocalBaseline {
	projectId: string;
	jobId: string;
	configFingerprint: string;
	report: UnifiedReport;
	createdAt: string;
}

export type LocalRunStatus = 'submitted' | 'complete' | 'failed';

export interface LocalRun {
	jobId: string;
	projectId: string;
	configFingerprint: string;
	status: LocalRunStatus;
	createdAt: string;
	completedAt?: string;
	score?: number;
	totalIssues?: number;
	/** Full report kept so a hosted job can be reopened after artifacts expire. */
	report?: UnifiedReport;
}

export function buildArchivedLocalRun(
	run: LocalRun,
	report: UnifiedReport,
	jobId: string,
	completedAt = new Date().toISOString()
): LocalRun | null {
	if (
		run.jobId !== jobId ||
		report.meta.jobId !== jobId ||
		(run.status === 'complete' && run.report)
	) {
		return null;
	}

	return {
		...run,
		status: 'complete',
		completedAt: run.completedAt ?? completedAt,
		...(report.summary.score == null ? {} : { score: report.summary.score }),
		totalIssues: report.summary.totalIssues,
		report
	};
}

export interface ProjectDiffSide {
	jobId: string;
	score?: number;
	totalIssues: number;
}

export interface ProjectDiff {
	schema: typeof LOCAL_PROJECT_DIFF_SCHEMA;
	baseline: ProjectDiffSide;
	current: ProjectDiffSide;
	delta: {
		scoreDelta?: number;
		newIssues: number;
		fixedIssues: number;
		unchangedIssues: number;
	};
	new: IssueDetail[];
	fixed: IssueDetail[];
	/** Did not finish in one of the reports, so their issues are not compared. */
	skippedScanners?: string[];
	incompatibleIdentity?: boolean;
}

const SECRET_KEYS = new Set([
	'apikey',
	'auth',
	'authorization',
	'inputvalues',
	'password',
	'secret',
	'storagestate',
	'token',
	'username'
]);

function isSecretKey(key: string): boolean {
	return SECRET_KEYS.has(key.toLowerCase().replaceAll(/[^a-z0-9]/g, ''));
}

function collectExecutionOnlyStrings(
	value: unknown,
	secretValues: Set<string>,
	executionOnly = false
): void {
	if (typeof value === 'string') {
		if (executionOnly && value.length > 0) secretValues.add(value);
		return;
	}
	if (Array.isArray(value)) {
		value.forEach((nested) => collectExecutionOnlyStrings(nested, secretValues, executionOnly));
		return;
	}
	if (!value || typeof value !== 'object') return;

	Object.entries(value).forEach(([key, nested]) =>
		collectExecutionOnlyStrings(nested, secretValues, executionOnly || isSecretKey(key))
	);
}

function redactExecutionOnlyStrings(value: string, secretValues: string[]): string {
	let remainder = value;
	let redacted = '';
	while (remainder.length > 0) {
		let matchIndex = -1;
		let matchLength = 0;
		for (const secretValue of secretValues) {
			const candidateIndex = remainder.indexOf(secretValue);
			if (
				candidateIndex >= 0 &&
				(matchIndex < 0 ||
					candidateIndex < matchIndex ||
					(candidateIndex === matchIndex && secretValue.length > matchLength))
			) {
				matchIndex = candidateIndex;
				matchLength = secretValue.length;
			}
		}
		if (matchIndex < 0) return redacted + remainder;
		redacted += `${remainder.slice(0, matchIndex)}[redacted]`;
		remainder = remainder.slice(matchIndex + matchLength);
	}
	return redacted;
}

function copySafeValue(value: unknown, secretValues: string[]): unknown {
	if (Array.isArray(value)) {
		return value.map((nested) => copySafeValue(nested, secretValues));
	}
	if (typeof value === 'string') return redactExecutionOnlyStrings(value, secretValues);
	if (!value || typeof value !== 'object') {
		return value;
	}

	return Object.fromEntries(
		Object.entries(value).flatMap(([key, nested]) =>
			isSecretKey(key) ? [] : [[key, copySafeValue(nested, secretValues)]]
		)
	);
}

/**
 * Return the only scanner configuration that may be stored in the browser.
 * Authentication and AI input values are execution-only, even if a caller
 * accidentally includes them in a scanner config object.
 */
export function sanitizeScannerSelections(selections: ScannerSelection[]): ScannerSelection[] {
	const executionOnlyValues = new Set<string>();
	selections.forEach((selection) =>
		collectExecutionOnlyStrings(selection.config, executionOnlyValues)
	);
	// Replace longer values first so overlapping secrets cannot leave a suffix
	// behind when the shorter value is encountered first.
	const secretValues = [...executionOnlyValues].sort((left, right) => right.length - left.length);

	return selections.map((selection) => ({
		id: selection.id,
		enabled: selection.enabled,
		...(selection.config
			? { config: copySafeValue(selection.config, secretValues) as Record<string, unknown> }
			: {})
	}));
}

export function sanitizeProjectConfiguration(
	configuration: LocalProjectConfiguration
): LocalProjectConfiguration {
	return {
		urls: [...new Set(configuration.urls.map((url) => url.trim()).filter(Boolean))].sort(),
		scanners: sanitizeScannerSelections(configuration.scanners).sort((a, b) =>
			a.id.localeCompare(b.id)
		),
		browser: 'chromium',
		highlightStyle: 'solid'
	};
}

function stableValue(value: unknown): unknown {
	if (Array.isArray(value)) return value.map(stableValue);
	if (!value || typeof value !== 'object') return value;

	return Object.fromEntries(
		Object.entries(value)
			.sort(([left], [right]) => left.localeCompare(right))
			.map(([key, nested]) => [key, stableValue(nested)])
	);
}

function bytesToHex(bytes: Uint8Array): string {
	return Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('');
}

/** A deterministic, non-secret fingerprint for baseline compatibility. */
export async function fingerprintProjectConfiguration(
	configuration: LocalProjectConfiguration
): Promise<string> {
	const safe = sanitizeProjectConfiguration(configuration);
	const fingerprintInput = {
		urls: safe.urls,
		scanners: safe.scanners
			.filter((scanner) => scanner.enabled)
			.map((scanner) => ({
				id: scanner.id,
				...(scanner.config ? { config: scanner.config } : {})
			})),
		browser: safe.browser,
		highlightStyle: safe.highlightStyle
	};
	const canonical = JSON.stringify(stableValue(fingerprintInput));
	const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(canonical));
	return `identity-v2:${bytesToHex(new Uint8Array(digest))}`;
}

function reportScore(report: UnifiedReport): number | undefined {
	return report.summary.score ?? undefined;
}

export function computeProjectDiff(
	baselineJobId: string,
	baselineReport: UnifiedReport,
	currentJobId: string,
	currentReport: UnifiedReport
): ProjectDiff {
	// Same rule as diff.ComputeDiff in libs/go/diff: a scanner that did not
	// finish in either report is compared in neither, or its baseline findings
	// would all look fixed.
	const skipped = new Set([
		...incompleteScanners(baselineReport.scanners),
		...incompleteScanners(currentReport.scanners)
	]);
	const baselineScanners = new Set(baselineReport.scanners?.map((scanner) => scanner.id));
	const currentScanners = new Set(currentReport.scanners?.map((scanner) => scanner.id));
	for (const id of baselineScanners) if (!currentScanners.has(id)) skipped.add(id);
	for (const id of currentScanners) if (!baselineScanners.has(id)) skipped.add(id);

	const compared = (issue: IssueDetail) => !skipped.has(issue.scanner);
	const baselineById = new Map(baselineReport.issues.filter(compared).flatMap(occurrenceFindings));
	const currentById = new Map(currentReport.issues.filter(compared).flatMap(occurrenceFindings));

	const incompatibleIdentity =
		hasStableReportIdentity(baselineReport) !== hasStableReportIdentity(currentReport);
	if (incompatibleIdentity) {
		baselineById.clear();
		currentById.clear();
	}
	const newIssues = [...currentById.entries()]
		.filter(([key]) => !baselineById.has(key))
		.map(([, issue]) => issue)
		.sort((left, right) => findingSortKey(left).localeCompare(findingSortKey(right)));
	const fixedIssues = [...baselineById.entries()]
		.filter(([key]) => !currentById.has(key))
		.map(([, issue]) => issue)
		.sort((left, right) => findingSortKey(left).localeCompare(findingSortKey(right)));
	const unchangedIssues = [...currentById.keys()].filter((issueId) =>
		baselineById.has(issueId)
	).length;
	const baselineScore =
		skipped.size === 0 && !incompatibleIdentity ? reportScore(baselineReport) : undefined;
	const currentScore =
		skipped.size === 0 && !incompatibleIdentity ? reportScore(currentReport) : undefined;
	// A score covers every scanner, so it cannot be compared across a skip.
	const scoreComparable =
		!incompatibleIdentity &&
		skipped.size === 0 &&
		baselineScore !== undefined &&
		currentScore !== undefined;

	return {
		schema: LOCAL_PROJECT_DIFF_SCHEMA,
		baseline: {
			jobId: baselineJobId,
			...(baselineScore === undefined ? {} : { score: baselineScore }),
			totalIssues: baselineReport.summary.totalIssues
		},
		current: {
			jobId: currentJobId,
			...(currentScore === undefined ? {} : { score: currentScore }),
			totalIssues: currentReport.summary.totalIssues
		},
		delta: {
			...(scoreComparable ? { scoreDelta: currentScore - baselineScore } : {}),
			newIssues: newIssues.length,
			fixedIssues: fixedIssues.length,
			unchangedIssues
		},
		new: newIssues,
		fixed: fixedIssues,
		...(skipped.size === 0 ? {} : { skippedScanners: [...skipped].sort() }),
		...(incompatibleIdentity ? { incompatibleIdentity: true } : {})
	};
}

export function isUnifiedReport(value: unknown): value is UnifiedReport {
	if (!value || typeof value !== 'object') return false;
	const candidate = value as Partial<UnifiedReport>;
	return (
		typeof candidate.version === 'string' &&
		candidate.version.startsWith('2.') &&
		Array.isArray(candidate.issues) &&
		Boolean(candidate.summary && typeof candidate.summary.totalIssues === 'number')
	);
}

export function hasStableReportIdentity(report: UnifiedReport): boolean {
	return (
		report.pages.some((page) => page.id.startsWith('page-v2-')) ||
		report.issues.some((issue) => issue.pageId.startsWith('page-v2-'))
	);
}

function occurrenceFindings(issue: IssueDetail): [string, IssueDetail][] {
	if (!issue.pageId.startsWith('page-v2-') || !issue.occurrences?.length)
		return [[issue.id, issue]];
	return issue.occurrences.map((occurrence) => [
		occurrence.elementId ?? issue.id,
		{
			...issue,
			occurrences: [occurrence],
			elementCount: 1,
			elementsTruncated: false
		}
	]);
}

function findingSortKey(issue: IssueDetail): string {
	return `${issue.id}:${issue.occurrences?.[0]?.elementId ?? ''}`;
}
