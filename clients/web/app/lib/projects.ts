import type { ScannerSelection } from './types/scan';
import type { IssueDetail, UnifiedReport } from './types/unified-report';

export const LOCAL_PROJECT_SCHEMA_VERSION = 1 as const;
export const LOCAL_PROJECT_DIFF_SCHEMA = 'stageflow/diff@v1' as const;

export interface LocalProjectConfiguration {
	urls: string[];
	scanners: ScannerSelection[];
	browser: 'chromium' | 'firefox' | 'webkit';
	highlightStyle: 'solid' | 'dashed';
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
		browser: configuration.browser,
		highlightStyle: configuration.highlightStyle
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
	return bytesToHex(new Uint8Array(digest));
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
	const baselineById = new Map(baselineReport.issues.map((issue) => [issue.id, issue]));
	const currentById = new Map(currentReport.issues.map((issue) => [issue.id, issue]));

	const newIssues = [...currentById.values()]
		.filter((issue) => !baselineById.has(issue.id))
		.sort((left, right) => left.id.localeCompare(right.id));
	const fixedIssues = [...baselineById.values()]
		.filter((issue) => !currentById.has(issue.id))
		.sort((left, right) => left.id.localeCompare(right.id));
	const unchangedIssues = [...currentById.keys()].filter((issueId) =>
		baselineById.has(issueId)
	).length;
	const baselineScore = reportScore(baselineReport);
	const currentScore = reportScore(currentReport);

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
			...(baselineScore === undefined || currentScore === undefined
				? {}
				: { scoreDelta: currentScore - baselineScore }),
			newIssues: newIssues.length,
			fixedIssues: fixedIssues.length,
			unchangedIssues
		},
		new: newIssues,
		fixed: fixedIssues
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
