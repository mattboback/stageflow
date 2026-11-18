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
	now: () => string = () => new Date().toISOString(),
	initialReviews: ReviewVerdictsByJob = {},
	onChange?: (reviews: ReviewVerdictsByJob) => void
) {
	let cache: ReviewVerdictsByJob = normalizeReviewVerdicts(initialReviews);
	let cacheLoaded = storage === null;
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
		const updated = normalizeReviewVerdicts(mutate(readPersisted()));
		const next = storage ? pruneJobs(updated) : updated;
		cache = next;
		cacheLoaded = true;
		try {
			storage?.setItem(REVIEW_VERDICTS_STORAGE_KEY, JSON.stringify(next));
		} catch {
			// Quota or private-mode failures lose persistence, not session state.
		}
		onChange?.(next);
		notify();
	};

	return {
		subscribe(this: void, listener: () => void): () => void {
			listeners.add(listener);
			return () => listeners.delete(listener);
		},

		getSnapshot(this: void): ReviewVerdictsByJob {
			if (!cacheLoaded) {
				cache = readPersisted();
				cacheLoaded = true;
			}
			return cache;
		},

		getServerSnapshot(this: void): ReviewVerdictsByJob {
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

export type ReviewVerdictStore = ReturnType<typeof createReviewVerdictStore>;

/** Rebase this tab's edits on fresh persisted records without erasing another tab. */
export function mergeReviewChanges(
	persisted: ReviewVerdictsByJob,
	previous: ReviewVerdictsByJob,
	next: ReviewVerdictsByJob
): ReviewVerdictsByJob {
	const merged = normalizeReviewVerdicts(persisted);
	for (const jobId of new Set([...Object.keys(previous), ...Object.keys(next)])) {
		const prior = previous[jobId] ?? {};
		const updated = next[jobId] ?? {};
		const records = { ...merged[jobId] };
		for (const issueId of new Set([...Object.keys(prior), ...Object.keys(updated)])) {
			if (JSON.stringify(prior[issueId]) === JSON.stringify(updated[issueId])) continue;
			if (updated[issueId])
				Object.defineProperty(records, issueId, {
					value: updated[issueId],
					enumerable: true,
					configurable: true,
					writable: true
				});
			else delete records[issueId];
		}
		if (Object.keys(records).length)
			Object.defineProperty(merged, jobId, {
				value: records,
				enumerable: true,
				configurable: true,
				writable: true
			});
		else delete merged[jobId];
	}
	return merged;
}
