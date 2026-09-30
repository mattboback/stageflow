export type ReviewDecision = 'pass' | 'fail';

export interface ReviewMeasurement {
	fg: string;
	bg: string;
	ratio: number | null;
	largeText?: boolean;
	requiredThreshold?: number;
}

export interface ReviewVerdict {
	verdict?: ReviewDecision;
	note?: string;
	subjectKey?: string;
	evidenceSignature?: string;
	executionSignature?: string;
	scannerVersion?: string;
	screenshotDigest?: string;
	sourceJobId?: string;
	carriedFrom?: string;
	at: string;
	measurement?: ReviewMeasurement;
}

export type ReviewVerdictsByJob = Record<string, Record<string, ReviewVerdict>>;

export const MAX_REVIEW_NOTE_LENGTH = 4000;

export const REVIEW_VERDICTS_STORAGE_KEY = 'review-verdicts-v1';
export const LEGACY_CONTRAST_VERDICTS_STORAGE_KEY = 'contrast-verdicts';

function isRecord(value: unknown): value is Record<string, unknown> {
	return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function normalizeMeasurement(value: unknown): ReviewMeasurement | undefined {
	if (!isRecord(value)) return undefined;
	if (typeof value.fg !== 'string' || typeof value.bg !== 'string') return undefined;
	if (value.ratio !== null && typeof value.ratio !== 'number') return undefined;
	const largeText = typeof value.largeText === 'boolean' ? value.largeText : undefined;
	const requiredThreshold =
		typeof value.requiredThreshold === 'number' &&
		Number.isFinite(value.requiredThreshold) &&
		value.requiredThreshold > 0
			? value.requiredThreshold
			: undefined;
	return {
		fg: value.fg,
		bg: value.bg,
		ratio: value.ratio,
		...(largeText !== undefined ? { largeText } : {}),
		...(requiredThreshold !== undefined ? { requiredThreshold } : {})
	};
}

function normalizeVerdict(value: unknown): ReviewVerdict | null {
	if (!isRecord(value)) return null;
	if (value.verdict !== undefined && value.verdict !== 'pass' && value.verdict !== 'fail')
		return null;
	const note =
		typeof value.note === 'string' ? value.note.slice(0, MAX_REVIEW_NOTE_LENGTH) : undefined;
	if (!value.verdict && !note?.trim()) return null;
	if (typeof value.at !== 'string' || value.at.length === 0) return null;

	const measurement =
		normalizeMeasurement(value.measurement) ??
		normalizeMeasurement({ fg: value.fg, bg: value.bg, ratio: value.ratio });

	return {
		...(value.verdict === 'pass' || value.verdict === 'fail' ? { verdict: value.verdict } : {}),
		...(note ? { note } : {}),
		...Object.fromEntries(
			[
				'subjectKey',
				'evidenceSignature',
				'executionSignature',
				'scannerVersion',
				'screenshotDigest',
				'sourceJobId',
				'carriedFrom'
			].flatMap((key) => (typeof value[key] === 'string' ? [[key, value[key]]] : []))
		),
		at: value.at,
		...(measurement ? { measurement } : {})
	};
}

/** Validate both the current shape and the legacy contrast-verdict shape. */
export function normalizeReviewVerdicts(value: unknown): ReviewVerdictsByJob {
	if (!isRecord(value)) return {};

	const jobs: ReviewVerdictsByJob = {};
	for (const [jobId, rawJob] of Object.entries(value)) {
		if (!jobId || !isRecord(rawJob)) continue;
		const verdicts: Record<string, ReviewVerdict> = {};
		for (const [issueId, rawVerdict] of Object.entries(rawJob)) {
			if (!issueId) continue;
			const verdict = normalizeVerdict(rawVerdict);
			if (verdict)
				Object.defineProperty(verdicts, issueId, {
					value: verdict,
					enumerable: true,
					writable: true,
					configurable: true
				});
		}
		if (Object.keys(verdicts).length > 0)
			Object.defineProperty(jobs, jobId, {
				value: verdicts,
				enumerable: true,
				writable: true,
				configurable: true
			});
	}
	return jobs;
}

/**
 * Human-readable timestamp for a review decision.
 *
 * Shared by ManualReviewTab and VerifyContrastTab, which held identical private
 * copies — the two surfaces must format a verdict the same way.
 * Returns the raw ISO string unchanged when it is unparseable.
 */
export function formatVerdictTime(iso: string): string {
	const date = new Date(iso);
	if (Number.isNaN(date.getTime())) return iso;
	return date.toLocaleString(undefined, {
		month: 'short',
		day: 'numeric',
		hour: 'numeric',
		minute: '2-digit'
	});
}
