export type ReviewDecision = 'pass' | 'fail';

export interface ReviewMeasurement {
	fg: string;
	bg: string;
	ratio: number | null;
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
	return { fg: value.fg, bg: value.bg, ratio: value.ratio };
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
