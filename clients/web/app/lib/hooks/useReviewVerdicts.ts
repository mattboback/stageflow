import { useCallback, useSyncExternalStore } from 'react';

import { REVIEW_VERDICTS_STORAGE_KEY, type ReviewVerdict } from '../report/review-verdict';
import {
	createReviewVerdictStore,
	type ReviewVerdictStorage
} from '../report/review-verdict-store';

const browserStorage: ReviewVerdictStorage = {
	getItem(key) {
		return typeof window === 'undefined' ? null : window.localStorage.getItem(key);
	},
	setItem(key, value) {
		if (typeof window !== 'undefined') window.localStorage.setItem(key, value);
	}
};

const store = createReviewVerdictStore(browserStorage);
let activeSubscriptions = 0;

function handleStorage(event: StorageEvent) {
	if (event.key !== REVIEW_VERDICTS_STORAGE_KEY && event.key !== null) return;
	try {
		if (event.storageArea && event.storageArea !== window.localStorage) return;
	} catch {
		return;
	}
	if (event.key === null || event.newValue === null) store.sync(null);
	else store.refresh();
}

function subscribe(listener: () => void): () => void {
	const unsubscribe = store.subscribe(listener);
	if (activeSubscriptions === 0 && typeof window !== 'undefined') {
		window.addEventListener('storage', handleStorage);
	}
	activeSubscriptions += 1;
	return () => {
		unsubscribe();
		activeSubscriptions -= 1;
		if (activeSubscriptions === 0 && typeof window !== 'undefined') {
			window.removeEventListener('storage', handleStorage);
		}
	};
}

export function useReviewVerdicts(jobId: string) {
	// Wrapped rather than passed as bare method references: unbound-method flags
	// those because a detached method could observe the wrong `this`.
	const verdicts = useSyncExternalStore(
		subscribe,
		() => store.getSnapshot(),
		() => store.getServerSnapshot()
	);

	const getVerdict = useCallback(
		(issueId: string): ReviewVerdict | null => verdicts[jobId]?.[issueId] ?? null,
		[verdicts, jobId]
	);

	const setVerdict = useCallback(
		(issueId: string, verdict: Omit<ReviewVerdict, 'at'>) => {
			store.setVerdict(jobId, issueId, verdict);
		},
		[jobId]
	);

	const clearVerdict = useCallback(
		(issueId: string) => {
			store.clearVerdict(jobId, issueId);
		},
		[jobId]
	);

	return { getVerdict, setVerdict, clearVerdict };
}
