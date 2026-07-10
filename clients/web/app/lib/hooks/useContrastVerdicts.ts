import { useCallback, useSyncExternalStore } from 'react';

export interface ContrastVerdict {
	verdict: 'pass' | 'fail';
	fg: string;
	bg: string;
	ratio: number | null;
	at: string;
}

type VerdictsByJob = Record<string, Record<string, ContrastVerdict>>;

const STORAGE_KEY = 'contrast-verdicts';
const MAX_JOBS = 20;

let cache: VerdictsByJob = {};
let cacheLoaded = false;
const listeners = new Set<() => void>();

function load(): VerdictsByJob {
	if (typeof window === 'undefined') return {};
	try {
		const stored = window.localStorage.getItem(STORAGE_KEY);
		return stored ? (JSON.parse(stored) as VerdictsByJob) : {};
	} catch {
		return {};
	}
}

function latestTimestamp(verdicts: Record<string, ContrastVerdict>): string {
	return Object.values(verdicts).reduce((max, v) => (v.at > max ? v.at : max), '');
}

function persist(next: VerdictsByJob) {
	const jobIds = Object.keys(next);
	if (jobIds.length > MAX_JOBS) {
		const byRecency = jobIds.sort((a, b) =>
			latestTimestamp(next[b] ?? {}).localeCompare(latestTimestamp(next[a] ?? {}))
		);
		next = Object.fromEntries(
			byRecency.slice(0, MAX_JOBS).map((id) => [id, next[id] ?? {}])
		);
	}
	cache = next;
	try {
		window.localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
	} catch {
		// Quota or private-mode failures just lose persistence, not the session state.
	}
	for (const listener of listeners) listener();
}

function subscribe(listener: () => void): () => void {
	listeners.add(listener);
	return () => listeners.delete(listener);
}

function getSnapshot(): VerdictsByJob {
	if (!cacheLoaded && typeof window !== 'undefined') {
		cache = load();
		cacheLoaded = true;
	}
	return cache;
}

function getServerSnapshot(): VerdictsByJob {
	return cache;
}

export function useContrastVerdicts(jobId: string) {
	const verdicts = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);

	const getVerdict = useCallback(
		(issueId: string): ContrastVerdict | null => verdicts[jobId]?.[issueId] ?? null,
		[verdicts, jobId]
	);

	const setVerdict = useCallback(
		(issueId: string, verdict: Omit<ContrastVerdict, 'at'>) => {
			const next: VerdictsByJob = {
				...getSnapshot(),
				[jobId]: {
					...getSnapshot()[jobId],
					[issueId]: { ...verdict, at: new Date().toISOString() }
				}
			};
			persist(next);
		},
		[jobId]
	);

	const clearVerdict = useCallback(
		(issueId: string) => {
			const current = getSnapshot();
			if (!current[jobId]?.[issueId]) return;
			const jobVerdicts = { ...current[jobId] };
			delete jobVerdicts[issueId];
			persist({ ...current, [jobId]: jobVerdicts });
		},
		[jobId]
	);

	return { getVerdict, setVerdict, clearVerdict };
}
