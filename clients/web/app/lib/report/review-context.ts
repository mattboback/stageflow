import type { UnifiedReport } from '../types/unified-report';
import { buildOccurrenceModeReport } from './occurrence-mode';
import type { ReviewVerdict, ReviewVerdictsByJob } from './review-verdict';

export interface ReviewSubject {
	subjectKey: string;
	evidenceSignature?: string;
	executionSignature?: string;
	scannerVersion?: string;
	screenshotDigest?: string;
}

export interface ReviewContext {
	projectId?: string;
	jobId: string;
	pageLabels?: Record<string, string>;
	subjects: Record<string, ReviewSubject>;
}

function canonical(value: unknown): string {
	if (Array.isArray(value)) return `[${value.map(canonical).join(',')}]`;
	if (value && typeof value === 'object') {
		return `{${Object.entries(value)
			.sort(([a], [b]) => a.localeCompare(b))
			.map(([key, item]) => `${JSON.stringify(key)}:${canonical(item)}`)
			.join(',')}}`;
	}
	return JSON.stringify(value) ?? 'null';
}

async function signature(value: unknown): Promise<string | undefined> {
	if (!globalThis.crypto?.subtle) return undefined;
	const hash = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(canonical(value)));
	return `review-v1:${Array.from(new Uint8Array(hash), (byte) => byte.toString(16).padStart(2, '0')).join('')}`;
}

/** No inferred execution settings: older reports remain historical only. */
export async function createReviewContext(
	report: UnifiedReport,
	projectId?: string,
	configurationFingerprint?: string,
	screenshotDigests: Record<string, string> = {}
): Promise<ReviewContext> {
	const meta = report.meta as typeof report.meta & {
		scanContext?: unknown;
		profile?: string;
		policyVersion?: string;
	};
	const executionSignature = meta.scanContext
		? await signature({
				context: meta.scanContext,
				baseUrl: report.meta.scanContext?.inputType === 'urls' ? meta.baseUrl : undefined,
				profile: meta.profile,
				policyVersion: meta.policyVersion,
				configurationFingerprint
			})
		: undefined;
	const subjects: Record<string, ReviewSubject> = {};
	for (const issue of buildOccurrenceModeReport(report).issues) {
		const occurrence = issue.occurrences?.[0];
		const stable =
			issue.pageId.startsWith('page-v2-') &&
			(!occurrence || occurrence.elementId?.startsWith('occ-v2-'));
		const subjectKey = stable
			? `${issue.scanner}|${issue.ruleId}|${issue.pageId}|${occurrence?.elementId ?? 'page'}`
			: `legacy:${issue.id}`;
		const summary = report.scanners.find((scanner) => scanner.id === issue.scanner);
		const scannerVersion = summary?.toolVersion;
		const screenshotDigest =
			screenshotDigests[`${issue.scanner}|${issue.pageId}`] ?? screenshotDigests[issue.pageId];
		const evidence =
			occurrence?.html ||
			occurrence?.contextHtml ||
			issue.scannerData?.details ||
			issue.scannerData?.contrastData;
		const evidenceSignature =
			stable && evidence
				? await signature({
						ruleId: issue.ruleId,
						occurrence: occurrence && {
							selector: occurrence.selector,
							html: occurrence.html,
							contextHtml: occurrence.contextHtml,
							textSnippet: occurrence.textSnippet,
							ancestorPath: occurrence.ancestorPath,
							failureSummary: occurrence.failureSummary,
							boundingBox: occurrence.boundingBox
						},
						scannerData: issue.scannerData,
						screenshotDigest
					})
				: undefined;
		subjects[issue.id] = {
			subjectKey,
			...(evidenceSignature ? { evidenceSignature } : {}),
			...(executionSignature ? { executionSignature } : {}),
			...(scannerVersion ? { scannerVersion } : {}),
			...(screenshotDigest ? { screenshotDigest } : {})
		};
	}
	for (const check of report.manualChecks ?? []) {
		for (const pageId of check.pageIds ?? []) {
			if (!report.pages.some((page) => page.id === pageId)) continue;
			const key = `manual:${check.ruleId}:${pageId}`;
			const screenshotDigest =
				screenshotDigests[`lighthouse|${pageId}`] ?? screenshotDigests[pageId];
			const scannerVersion = report.scanners.find(
				(scanner) => scanner.id === 'lighthouse'
			)?.toolVersion;
			const evidenceSignature =
				screenshotDigest && pageId.startsWith('page-v2-')
					? await signature({
							screenshotDigest,
							ruleId: check.ruleId,
							description: check.description
						})
					: undefined;
			subjects[key] = {
				subjectKey: key,
				...(executionSignature ? { executionSignature } : {}),
				...(scannerVersion ? { scannerVersion } : {}),
				...(screenshotDigest ? { screenshotDigest } : {}),
				...(evidenceSignature ? { evidenceSignature } : {})
			};
		}
	}
	return {
		jobId: report.meta.jobId,
		subjects,
		pageLabels: Object.fromEntries(report.pages.map((page) => [page.id, page.path ?? page.url])),
		...(projectId ? { projectId } : {})
	};
}

export interface PriorReview {
	review: ReviewVerdict;
	jobId: string;
	stale: boolean;
}

export function findPriorReview(
	reviews: ReviewVerdictsByJob,
	jobId: string,
	issueId: string,
	subject?: ReviewSubject
): PriorReview | null {
	let latest: PriorReview | null = null;
	for (const [previousJob, records] of Object.entries(reviews)) {
		if (previousJob === jobId) continue;
		for (const [previousIssue, review] of Object.entries(records)) {
			if (
				!(subject && review.subjectKey === subject.subjectKey) &&
				!(previousIssue === issueId && !review.subjectKey)
			)
				continue;
			// Screenshot-derived measurements have no image digest in legacy/live reports.
			const stale =
				(Boolean(review.measurement) &&
					(!subject?.screenshotDigest || review.screenshotDigest !== subject.screenshotDigest)) ||
				!subject?.evidenceSignature ||
				!subject.executionSignature ||
				!subject.scannerVersion ||
				review.evidenceSignature !== subject.evidenceSignature ||
				review.executionSignature !== subject.executionSignature ||
				review.scannerVersion !== subject.scannerVersion;
			if (!latest || review.at > latest.review.at) latest = { review, jobId: previousJob, stale };
		}
	}
	return latest;
}
