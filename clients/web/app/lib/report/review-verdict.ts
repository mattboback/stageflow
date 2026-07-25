export type ReviewDecision = 'pass' | 'fail';

export interface ReviewMeasurement {
	fg: string;
	bg: string;
	ratio: number | null;
	largeText?: boolean;
	requiredThreshold?: number;
}

export interface ReviewVerdict {
	verdict: ReviewDecision;
	at: string;
	measurement?: ReviewMeasurement;
}

export type ReviewVerdictsByJob = Record<string, Record<string, ReviewVerdict>>;

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
	if (value.verdict !== 'pass' && value.verdict !== 'fail') return null;
	if (typeof value.at !== 'string' || value.at.length === 0) return null;

	const measurement =
		normalizeMeasurement(value.measurement) ??
		normalizeMeasurement({ fg: value.fg, bg: value.bg, ratio: value.ratio });

	return {
		verdict: value.verdict,
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
			if (verdict) verdicts[issueId] = verdict;
		}
		if (Object.keys(verdicts).length > 0) jobs[jobId] = verdicts;
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
