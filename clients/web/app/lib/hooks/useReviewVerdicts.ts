import { useCallback, useSyncExternalStore } from 'react';

import {
	LEGACY_CONTRAST_VERDICTS_STORAGE_KEY,
	REVIEW_VERDICTS_STORAGE_KEY,
	normalizeReviewVerdicts,
	type ReviewVerdict,
	type ReviewVerdictsByJob
} from '../report/review-verdict';

const MAX_JOBS = 20;

let cache: ReviewVerdictsByJob = {};
let cacheLoaded = false;
const listeners = new Set<() => void>();

function parseStored(raw: string | null): ReviewVerdictsByJob {
	if (!raw) return {};
	try {
		return normalizeReviewVerdicts(JSON.parse(raw));
	} catch {
		return {};
	}
}

function load(): ReviewVerdictsByJob {
	if (typeof window === 'undefined') return {};
	try {
		const currentRaw = window.localStorage.getItem(REVIEW_VERDICTS_STORAGE_KEY);
		if (currentRaw !== null) return parseStored(currentRaw);

		const legacy = parseStored(window.localStorage.getItem(LEGACY_CONTRAST_VERDICTS_STORAGE_KEY));
		if (Object.keys(legacy).length > 0) {
			window.localStorage.setItem(REVIEW_VERDICTS_STORAGE_KEY, JSON.stringify(legacy));
		}
		return legacy;
	} catch {
		return {};
	}
}

function latestTimestamp(verdicts: Record<string, ReviewVerdict>): string {
	return Object.values(verdicts).reduce(
		(max, verdict) => (verdict.at > max ? verdict.at : max),
		''
	);
}

function persist(next: ReviewVerdictsByJob) {
	const jobIds = Object.keys(next);
	if (jobIds.length > MAX_JOBS) {
		const byRecency = jobIds.sort((a, b) =>
			latestTimestamp(next[b] ?? {}).localeCompare(latestTimestamp(next[a] ?? {}))
		);
		next = Object.fromEntries(byRecency.slice(0, MAX_JOBS).map((id) => [id, next[id] ?? {}]));
	}
	cache = next;
	try {
		window.localStorage.setItem(REVIEW_VERDICTS_STORAGE_KEY, JSON.stringify(next));
	} catch {
		// Quota or private-mode failures lose persistence, not session state.
	}
	for (const listener of listeners) listener();
}

function subscribe(listener: () => void): () => void {
	listeners.add(listener);
	return () => listeners.delete(listener);
}

function getSnapshot(): ReviewVerdictsByJob {
	if (!cacheLoaded && typeof window !== 'undefined') {
		cache = load();
		cacheLoaded = true;
	}
	return cache;
}

function getServerSnapshot(): ReviewVerdictsByJob {
	return cache;
}

export function useReviewVerdicts(jobId: string) {
	const verdicts = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);

	const getVerdict = useCallback(
		(issueId: string): ReviewVerdict | null => verdicts[jobId]?.[issueId] ?? null,
		[verdicts, jobId]
	);

	const setVerdict = useCallback(
		(issueId: string, verdict: Omit<ReviewVerdict, 'at'>) => {
			const current = getSnapshot();
			persist({
				...current,
				[jobId]: {
					...current[jobId],
					[issueId]: { ...verdict, at: new Date().toISOString() }
				}
			});
		},
		[jobId]
	);

	const clearVerdict = useCallback(
		(issueId: string) => {
			const current = getSnapshot();
			if (!current[jobId]?.[issueId]) return;
			const jobVerdicts = { ...current[jobId] };
			delete jobVerdicts[issueId];
			const next = { ...current };
			if (Object.keys(jobVerdicts).length === 0) delete next[jobId];
			else next[jobId] = jobVerdicts;
			persist(next);
		},
		[jobId]
	);

	return { getVerdict, setVerdict, clearVerdict };
}
