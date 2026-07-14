import { describe, expect, it, vi } from 'vitest';

import { REVIEW_VERDICTS_STORAGE_KEY } from './review-verdict';
import { createReviewVerdictStore, type ReviewVerdictStorage } from './review-verdict-store';

class MemoryStorage implements ReviewVerdictStorage {
	private values = new Map<string, string>();

	getItem(key: string): string | null {
		return this.values.get(key) ?? null;
	}

	setItem(key: string, value: string): void {
		this.values.set(key, value);
	}
}

function persisted(storage: ReviewVerdictStorage) {
	return JSON.parse(storage.getItem(REVIEW_VERDICTS_STORAGE_KEY) ?? '{}');
}

describe('createReviewVerdictStore', () => {
	it('merges writes from independently cached stores in the same job', () => {
		const storage = new MemoryStorage();
		const storeA = createReviewVerdictStore(storage, () => '2026-07-14T10:00:00.000Z');
		const storeB = createReviewVerdictStore(storage, () => '2026-07-14T10:01:00.000Z');
		storeA.getSnapshot();
		storeB.getSnapshot();

		storeA.setVerdict('job-1', 'issue-a', { verdict: 'pass' });
		storeB.setVerdict('job-1', 'issue-b', { verdict: 'fail' });

		expect(persisted(storage)['job-1']).toEqual({
			'issue-a': { verdict: 'pass', at: '2026-07-14T10:00:00.000Z' },
			'issue-b': { verdict: 'fail', at: '2026-07-14T10:01:00.000Z' }
		});
	});

	it('preserves other jobs and fresh decisions while clearing', () => {
		const storage = new MemoryStorage();
		const storeA = createReviewVerdictStore(storage);
		const storeB = createReviewVerdictStore(storage);
		storeA.getSnapshot();
		storeB.getSnapshot();

		storeA.setVerdict('job-a', 'issue-a', { verdict: 'pass' });
		storeB.setVerdict('job-b', 'issue-b', { verdict: 'fail' });
		storeA.clearVerdict('job-a', 'issue-a');

		expect(persisted(storage)).toEqual({
			'job-b': {
				'issue-b': expect.objectContaining({ verdict: 'fail' })
			}
		});
	});

	it('refreshes cached subscribers from a storage event value', () => {
		const storage = new MemoryStorage();
		const store = createReviewVerdictStore(storage);
		const listener = vi.fn();
		store.getSnapshot();
		store.subscribe(listener);

		store.sync(
			JSON.stringify({
				'job-1': {
					'issue-1': { verdict: 'pass', at: '2026-07-14T10:00:00.000Z' }
				}
			})
		);

		expect(listener).toHaveBeenCalledOnce();
		expect(store.getSnapshot()['job-1']?.['issue-1']?.verdict).toBe('pass');
	});

	it('refreshes from the latest persisted value instead of a stale event payload', () => {
		const storage = new MemoryStorage();
		const storeA = createReviewVerdictStore(storage);
		const storeB = createReviewVerdictStore(storage);
		storeA.setVerdict('job-1', 'issue-a', { verdict: 'pass' });
		const staleEventValue = storage.getItem(REVIEW_VERDICTS_STORAGE_KEY);
		storeB.setVerdict('job-1', 'issue-b', { verdict: 'fail' });

		storeB.refresh();

		expect(staleEventValue).not.toBe(storage.getItem(REVIEW_VERDICTS_STORAGE_KEY));
		expect(Object.keys(storeB.getSnapshot()['job-1'] ?? {})).toEqual(['issue-a', 'issue-b']);
	});
});
