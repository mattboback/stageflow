import type { IssueDetail, UnifiedReport } from './types/unified-report';

export const VERIFICATION_STORAGE_KEY = 'stageflow-verifications-v1';
const MAX_REQUESTS = 20;
const MAX_RECORD_BYTES = 256 * 1024;
const SUPPORTED_SCANNERS = new Set([
	'axe',
	'lighthouse',
	'seo',
	'security-headers',
	'link-checker'
]);

type ScanContext = NonNullable<UnifiedReport['meta']['scanContext']>;
export interface VerificationRequest {
	jobId: string;
	sourceJobId: string;
	createdAt: string;
	issue: Pick<IssueDetail, 'id' | 'title' | 'scanner' | 'ruleId' | 'pageId' | 'pageUrl'>;
	subjectId?: string;
	sourceContext: ScanContext;
	sourceScannerVersion: string;
	expectedFinalUrl: string;
}

export type VerificationStatus = 'fixed' | 'still-present' | 'needs-review' | 'unverified';
export interface VerificationResult {
	status: VerificationStatus;
	explanation: string;
}

export function verificationUnavailableReason(
	report: UnifiedReport,
	issue: IssueDetail
): string | null {
	const context = report.meta.scanContext;
	if (!SUPPORTED_SCANNERS.has(issue.scanner) || issue.category === 'auth')
		return 'Run a complete scan to inspect this finding.';
	if (
		!context ||
		!context.scannerConfigurations?.[issue.scanner] ||
		!context.scannerConfigFingerprints[issue.scanner] ||
		!context.pageConfigFingerprints?.[issue.pageId]
	)
		return 'This report lacks the original scan settings. Run a new complete scan before verifying a fix.';
	if (
		context.verificationEligiblePages?.[issue.pageId] !== true ||
		context.viewport?.width !== 1280 ||
		context.viewport.height !== 720
	)
		return 'This page uses custom viewport, wait or pre-scan actions that targeted verification cannot replay. Run a complete scan using those settings.';
	const scanner = report.scanners.find((entry) => entry.id === issue.scanner);
	const coverage = scanner?.pages?.find((page) => page.pageId === issue.pageId);
	if (!scanner?.toolVersion || coverage?.status !== 'success' || !coverage.finalUrl)
		return 'The original page was not successfully evaluated with recorded scanner coverage. Run a new complete scan.';
	if (!report.pages.some((page) => page.id === issue.pageId))
		return 'The original page is unavailable.';
	return null;
}

export function buildVerificationRequest(
	report: UnifiedReport,
	issue: IssueDetail,
	jobId: string
): VerificationRequest {
	const reason = verificationUnavailableReason(report, issue);
	if (reason) throw new Error(reason);
	const context = report.meta.scanContext;
	const scanner = report.scanners.find((entry) => entry.id === issue.scanner);
	const finalUrl = scanner?.pages?.find((page) => page.pageId === issue.pageId)?.finalUrl;
	if (!context || !scanner?.toolVersion || !finalUrl)
		throw new Error('Original scan context is missing.');
	const subjectId = issue.occurrences?.[0]?.elementId;
	return {
		jobId,
		sourceJobId: report.meta.jobId,
		createdAt: new Date().toISOString(),
		issue: {
			id: issue.id,
			title: issue.title,
			scanner: issue.scanner,
			ruleId: issue.ruleId,
			pageId: issue.pageId,
			pageUrl: issue.pageUrl
		},
		...(subjectId ? { subjectId } : {}),
		sourceContext: {
			expectedPages: [issue.pageId],
			identityScope: context.identityScope,
			scannerConfigFingerprints: {
				[issue.scanner]: context.scannerConfigFingerprints[issue.scanner] ?? ''
			},
			pageConfigFingerprints: {
				[issue.pageId]: context.pageConfigFingerprints?.[issue.pageId] ?? ''
			},
			...(context.viewport ? { viewport: context.viewport } : {}),
			purpose: context.purpose,
			authenticated: context.authenticated,
			inputType: context.inputType
		},
		sourceScannerVersion: scanner.toolVersion,
		expectedFinalUrl: finalUrl
	};
}

function pageLocation(url: string, inputType: ScanContext['inputType']): string | null {
	try {
		const parsed = new URL(url);
		return `${inputType === 'zip' ? '' : parsed.origin}${parsed.pathname}${parsed.search}${parsed.hash}`;
	} catch {
		return null;
	}
}

function sameViewport(a: ScanContext, b: ScanContext): boolean {
	return a.viewport?.width === b.viewport?.width && a.viewport?.height === b.viewport?.height;
}

export function evaluateVerification(
	request: VerificationRequest,
	report: UnifiedReport
): VerificationResult {
	const unverified = (explanation: string): VerificationResult => ({
		status: 'unverified',
		explanation
	});
	const context = report.meta.scanContext;
	const source = request.sourceContext;
	const scanner = report.scanners.find((entry) => entry.id === request.issue.scanner);
	if (report.meta.jobId !== request.jobId || context?.purpose !== 'verification')
		return unverified('This report is not the requested targeted verification.');
	if (
		context.identityScope !== source.identityScope ||
		context.inputType !== source.inputType ||
		context.authenticated !== source.authenticated ||
		!sameViewport(context, source)
	)
		return unverified(
			'The page identity, authentication, input type or viewport differs from the original scan.'
		);
	if (
		!scanner?.toolVersion ||
		scanner.toolVersion !== request.sourceScannerVersion ||
		context.scannerConfigFingerprints[request.issue.scanner] !==
			source.scannerConfigFingerprints[request.issue.scanner] ||
		!context.pageConfigFingerprints?.[request.issue.pageId] ||
		context.pageConfigFingerprints[request.issue.pageId] !==
			source.pageConfigFingerprints?.[request.issue.pageId]
	)
		return unverified(
			'The scanner version or execution settings changed. Run a complete scan with the new settings.'
		);
	const page = scanner.pages?.find((entry) => entry.pageId === request.issue.pageId);
	if (page?.status !== 'success')
		return unverified('The affected page did not complete evaluation.');
	const expectedLocation = pageLocation(request.expectedFinalUrl, source.inputType);
	if (
		!expectedLocation ||
		!page.finalUrl ||
		pageLocation(page.finalUrl, context.inputType) !== expectedLocation
	)
		return unverified('The scan landed on a different page, redirect or login screen.');
	if (
		report.issues.some(
			(issue) => issue.pageId === request.issue.pageId && issue.category === 'auth'
		)
	)
		return unverified(
			'The scanner encountered an authentication wall. Provide fresh working authentication.'
		);
	const rules = page.rules?.filter((rule) => rule.ruleId === request.issue.ruleId) ?? [];
	if (!rules.length || rules.some((rule) => rule.outcome === 'error'))
		return unverified('The relevant rule was disabled, unsupported or failed to evaluate.');
	const findings = report.issues.filter(
		(issue) =>
			issue.scanner === request.issue.scanner &&
			issue.pageId === request.issue.pageId &&
			issue.ruleId === request.issue.ruleId
	);
	const matchesSubject = (subjectIds?: string[]) =>
		request.subjectId && subjectIds?.includes(request.subjectId);
	const failed = rules.filter((rule) => rule.outcome === 'fail');
	const incomplete = rules.filter((rule) => rule.outcome === 'incomplete');
	if (
		failed.some((rule) => !request.subjectId || matchesSubject(rule.subjectIds)) ||
		findings.some(
			(issue) =>
				issue.scannerData?.axeIncomplete !== true &&
				(!request.subjectId ||
					issue.occurrences?.some((occurrence) => occurrence.elementId === request.subjectId))
		)
	)
		return {
			status: 'still-present',
			explanation: 'The same page, rule and affected subject still fail.'
		};
	if (incomplete.some((rule) => !request.subjectId || matchesSubject(rule.subjectIds)))
		return {
			status: 'needs-review',
			explanation:
				'The current evaluation requires a human decision. Open the finding and review its evidence.'
		};
	if (failed.length || incomplete.length) {
		if (
			request.subjectId &&
			rules.some((rule) => rule.outcome === 'pass' && matchesSubject(rule.subjectIds))
		)
			return {
				status: 'fixed',
				explanation:
					'The original subject passed the relevant rule. Other subjects on this page still need attention.'
			};
		return unverified(
			'The rule still reports other or ambiguous subjects. The original subject has no confirmed outcome.'
		);
	}
	if (findings.length)
		return unverified(
			'The captured findings and rule outcomes disagree. Review this run manually.'
		);
	if (rules.some((rule) => rule.outcome === 'pass'))
		return {
			status: 'fixed',
			explanation:
				'The relevant rule completed on the intended page and the original defect is absent.'
		};
	if (rules.every((rule) => rule.outcome === 'inapplicable'))
		return {
			status: 'fixed',
			explanation:
				'The rule completed and is no longer applicable; the affected element or condition may have been removed.'
		};
	return unverified('The relevant rule has no conclusive outcome.');
}

interface VerificationStorage {
	getItem(key: string): string | null;
	setItem(key: string, value: string): void;
}

export function parseVerificationRequest(value: unknown): VerificationRequest | null {
	try {
		return parseRequest(value);
	} catch {
		return null;
	}
}

function parseRequest(value: unknown): VerificationRequest | null {
	if (!value || typeof value !== 'object') return null;
	const request = value as Partial<VerificationRequest>;
	const issue = request.issue;
	const context = request.sourceContext;
	if (
		typeof request.jobId !== 'string' ||
		typeof request.sourceJobId !== 'string' ||
		typeof request.createdAt !== 'string' ||
		typeof request.sourceScannerVersion !== 'string' ||
		typeof request.expectedFinalUrl !== 'string' ||
		!issue ||
		typeof issue !== 'object' ||
		Array.isArray(issue) ||
		!context ||
		typeof context !== 'object' ||
		Array.isArray(context) ||
		!['id', 'title', 'scanner', 'ruleId', 'pageId', 'pageUrl'].every(
			(key) => typeof Reflect.get(issue, key) === 'string'
		) ||
		!Array.isArray(context.expectedPages) ||
		context.expectedPages.some((page) => typeof page !== 'string') ||
		!SUPPORTED_SCANNERS.has(issue.scanner) ||
		!context.scannerConfigFingerprints ||
		typeof context.scannerConfigFingerprints[issue.scanner] !== 'string' ||
		!context.pageConfigFingerprints ||
		typeof context.pageConfigFingerprints[issue.pageId] !== 'string' ||
		!['path', 'origin'].includes(context.identityScope) ||
		!['urls', 'zip'].includes(context.inputType) ||
		!['full', 'verification'].includes(context.purpose) ||
		typeof context.authenticated !== 'boolean' ||
		(request.subjectId !== undefined && typeof request.subjectId !== 'string')
	)
		return null;
	if (
		context.viewport &&
		(typeof context.viewport.width !== 'number' || typeof context.viewport.height !== 'number')
	)
		return null;
	if (JSON.stringify(value).length > MAX_RECORD_BYTES) return null;
	// Whitelist serialized context: credentials, options and unrecognized fields never survive import.
	return {
		jobId: request.jobId,
		sourceJobId: request.sourceJobId,
		createdAt: request.createdAt,
		sourceScannerVersion: request.sourceScannerVersion,
		expectedFinalUrl: request.expectedFinalUrl,
		issue: {
			id: issue.id,
			title: issue.title,
			scanner: issue.scanner,
			ruleId: issue.ruleId,
			pageId: issue.pageId,
			pageUrl: issue.pageUrl
		},
		...(request.subjectId ? { subjectId: request.subjectId } : {}),
		sourceContext: {
			expectedPages: [issue.pageId],
			identityScope: context.identityScope,
			scannerConfigFingerprints: {
				[issue.scanner]: context.scannerConfigFingerprints[issue.scanner] ?? ''
			},
			pageConfigFingerprints: {
				[issue.pageId]: context.pageConfigFingerprints[issue.pageId] ?? ''
			},
			purpose: context.purpose,
			authenticated: context.authenticated,
			inputType: context.inputType,
			...(context.viewport
				? { viewport: { width: context.viewport.width, height: context.viewport.height } }
				: {})
		}
	};
}
function readRequests(storage: VerificationStorage): VerificationRequest[] {
	try {
		const raw = storage.getItem(VERIFICATION_STORAGE_KEY);
		if (!raw || raw.length > MAX_REQUESTS * MAX_RECORD_BYTES) return [];
		const value: unknown = JSON.parse(raw);
		if (!Array.isArray(value)) return [];
		return value
			.flatMap((item: unknown) => {
				const parsed = parseVerificationRequest(item);
				return parsed ? [parsed] : [];
			})
			.slice(0, MAX_REQUESTS);
	} catch {
		return [];
	}
}

export function saveVerificationRequest(
	request: VerificationRequest,
	storage?: VerificationStorage
): boolean {
	try {
		const activeStorage = storage ?? window.localStorage;
		const safe = parseVerificationRequest(request);
		if (!safe) return false;
		activeStorage.setItem(
			VERIFICATION_STORAGE_KEY,
			JSON.stringify(
				[
					safe,
					...readRequests(activeStorage).filter((existing) => existing.jobId !== request.jobId)
				].slice(0, MAX_REQUESTS)
			)
		);
		return true;
	} catch {
		return false;
	}
}

export function loadVerificationRequest(
	jobId: string,
	storage?: VerificationStorage
): VerificationRequest | null {
	try {
		return (
			readRequests(storage ?? window.localStorage).find((request) => request.jobId === jobId) ??
			null
		);
	} catch {
		return null;
	}
}
