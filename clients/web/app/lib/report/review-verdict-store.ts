import {
	LEGACY_CONTRAST_VERDICTS_STORAGE_KEY,
	REVIEW_VERDICTS_STORAGE_KEY,
	normalizeReviewVerdicts,
	type ReviewVerdict,
	type ReviewVerdictsByJob
} from './review-verdict';

const MAX_JOBS = 20;

export interface ReviewVerdictStorage {
	getItem(key: string): string | null;
	setItem(key: string, value: string): void;
}

function parseStored(raw: string | null): ReviewVerdictsByJob {
	if (!raw) return {};
	try {
		return normalizeReviewVerdicts(JSON.parse(raw));
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

function pruneJobs(next: ReviewVerdictsByJob): ReviewVerdictsByJob {
	const jobIds = Object.keys(next);
	if (jobIds.length <= MAX_JOBS) return next;
	const byRecency = jobIds.sort((a, b) =>
		latestTimestamp(next[b] ?? {}).localeCompare(latestTimestamp(next[a] ?? {}))
	);
	return Object.fromEntries(byRecency.slice(0, MAX_JOBS).map((id) => [id, next[id] ?? {}]));
}

export function createReviewVerdictStore(
	storage: ReviewVerdictStorage | null,
	now: () => string = () => new Date().toISOString()
) {
	let cache: ReviewVerdictsByJob = {};
	let cacheLoaded = false;
	const listeners = new Set<() => void>();

	const readPersisted = (): ReviewVerdictsByJob => {
		if (!storage) return cacheLoaded ? cache : {};
		try {
			const currentRaw = storage.getItem(REVIEW_VERDICTS_STORAGE_KEY);
			if (currentRaw !== null) return parseStored(currentRaw);

			const legacy = parseStored(storage.getItem(LEGACY_CONTRAST_VERDICTS_STORAGE_KEY));
			if (Object.keys(legacy).length > 0) {
				storage.setItem(REVIEW_VERDICTS_STORAGE_KEY, JSON.stringify(legacy));
			}
			return legacy;
		} catch {
			return cacheLoaded ? cache : {};
		}
	};

	const notify = () => {
		for (const listener of listeners) listener();
	};

	const persist = (mutate: (fresh: ReviewVerdictsByJob) => ReviewVerdictsByJob) => {
		const next = pruneJobs(mutate(readPersisted()));
		cache = next;
		cacheLoaded = true;
		try {
			storage?.setItem(REVIEW_VERDICTS_STORAGE_KEY, JSON.stringify(next));
		} catch {
			// Quota or private-mode failures lose persistence, not session state.
		}
		notify();
	};

	return {
		subscribe(listener: () => void): () => void {
			listeners.add(listener);
			return () => listeners.delete(listener);
		},

		getSnapshot(): ReviewVerdictsByJob {
			if (!cacheLoaded) {
				cache = readPersisted();
				cacheLoaded = true;
			}
			return cache;
		},

		getServerSnapshot(): ReviewVerdictsByJob {
			return cache;
		},

		refresh() {
			cache = readPersisted();
			cacheLoaded = true;
			notify();
		},

		sync(raw: string | null) {
			cache = parseStored(raw);
			cacheLoaded = true;
			notify();
		},

		setVerdict(jobId: string, issueId: string, verdict: Omit<ReviewVerdict, 'at'>) {
			persist((fresh) => ({
				...fresh,
				[jobId]: {
					...fresh[jobId],
					[issueId]: { ...verdict, at: now() }
				}
			}));
		},

		clearVerdict(jobId: string, issueId: string) {
			persist((fresh) => {
				if (!fresh[jobId]?.[issueId]) return fresh;
				const jobVerdicts = { ...fresh[jobId] };
				delete jobVerdicts[issueId];
				const next = { ...fresh };
				if (Object.keys(jobVerdicts).length === 0) delete next[jobId];
				else next[jobId] = jobVerdicts;
				return next;
			});
		}
	};
}
