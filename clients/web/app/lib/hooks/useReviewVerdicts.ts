import {
	createContext,
	createElement,
	useContext,
	useCallback,
	useSyncExternalStore,
	type ReactNode
} from 'react';

import { REVIEW_VERDICTS_STORAGE_KEY, type ReviewVerdict } from '../report/review-verdict';
import {
	createReviewVerdictStore,
	type ReviewVerdictStore,
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
export { createReviewContext } from '../report/review-context';
import { findPriorReview, type ReviewContext } from '../report/review-context';
const ProviderContext = createContext<{
	store: ReviewVerdictStore;
	context?: ReviewContext;
} | null>(null);
export function ReviewVerdictProvider({
	store,
	context,
	children
}: {
	store: ReviewVerdictStore;
	context?: ReviewContext;
	children: ReactNode;
}) {
	return createElement(
		ProviderContext.Provider,
		{ value: { store, ...(context ? { context } : {}) } },
		children
	);
}
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
	const provider = useContext(ProviderContext);
	const activeStore = provider?.store ?? store;
	const context = provider?.context;
	// Wrapped rather than passed as bare method references: unbound-method flags
	// those because a detached method could observe the wrong `this`.
	const verdicts = useSyncExternalStore(
		activeStore === store ? subscribe : activeStore.subscribe,
		() => activeStore.getSnapshot(),
		() => activeStore.getServerSnapshot()
	);

	const getVerdict = useCallback(
		(issueId: string): ReviewVerdict | null => verdicts[jobId]?.[issueId] ?? null,
		[verdicts, jobId]
	);

	const setVerdict = useCallback(
		(issueId: string, verdict: Omit<ReviewVerdict, 'at'>) => {
			activeStore.setVerdict(jobId, issueId, {
				...verdicts[jobId]?.[issueId],
				...verdict,
				...context?.subjects[issueId],
				sourceJobId: jobId
			});
		},
		[jobId, activeStore, context, verdicts]
	);

	const clearVerdict = useCallback(
		(issueId: string) => {
			const current = verdicts[jobId]?.[issueId];
			if (current?.note) {
				const record = { ...current };
				delete record.verdict;
				delete record.measurement;
				activeStore.setVerdict(jobId, issueId, record);
			} else activeStore.clearVerdict(jobId, issueId);
		},
		[jobId, activeStore, verdicts]
	);

	const getPriorReview = (issueId: string) =>
		context?.projectId
			? findPriorReview(verdicts, jobId, issueId, context.subjects[issueId])
			: null;
	const confirmPriorReview = (issueId: string) => {
		const prior = getPriorReview(issueId);
		if (!prior || prior.stale) return;
		setVerdict(issueId, { ...prior.review, carriedFrom: prior.jobId });
	};
	const setNote = (issueId: string, note: string) => {
		const previous = getVerdict(issueId);
		if (!note.trim() && !previous?.verdict) {
			activeStore.clearVerdict(jobId, issueId);
			return;
		}
		setVerdict(issueId, { ...previous, note: note.slice(0, 4000) });
	};
	return { getVerdict, setVerdict, clearVerdict, getPriorReview, confirmPriorReview, setNote };
}

export function useReviewReportContext() {
	return useContext(ProviderContext)?.context;
}

export function getBrowserReviewStore() {
	return store;
}
export function useReviewSnapshot() {
	const provider = useContext(ProviderContext);
	const activeStore = provider?.store ?? store;
	return useSyncExternalStore(
		activeStore === store ? subscribe : activeStore.subscribe,
		() => activeStore.getSnapshot(),
		() => activeStore.getServerSnapshot()
	);
}
