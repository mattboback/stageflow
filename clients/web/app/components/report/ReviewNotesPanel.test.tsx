import { describe, expect, it, vi } from 'vitest';
import { act, fireEvent, render, screen } from '@testing-library/react';
import { getBrowserReviewStore, ReviewVerdictProvider } from '../../lib/hooks/useReviewVerdicts';
import { REVIEW_VERDICTS_STORAGE_KEY } from '../../lib/report/review-verdict';
import { createReviewVerdictStore } from '../../lib/report/review-verdict-store';
import type { ReviewContext } from '../../lib/report/review-context';
import { getReviewGroupStatus } from '../../lib/report/review';
import { ReviewNotesPanel } from './ReviewNotesPanel';

const subject = {
	subjectKey: 'occ-v2-one',
	evidenceSignature: 'same',
	executionSignature: 'same',
	scannerVersion: '1'
};
const context: ReviewContext = {
	jobId: 'current',
	projectId: 'project',
	subjects: { issue: subject }
};

describe('review notes and confirmation', () => {
	it('requires a click before a prior pass is active in a new run', () => {
		const store = createReviewVerdictStore(null, () => 'now', {
			previous: { issue: { ...subject, verdict: 'pass', note: 'Checked manually', at: 'before' } }
		});
		render(
			<ReviewVerdictProvider store={store} context={context}>
				<ReviewNotesPanel jobId="current" issueId="issue" />
			</ReviewVerdictProvider>
		);
		expect(screen.getByText(/Previously reviewed/)).toBeTruthy();
		expect(store.getSnapshot().current).toBeUndefined();
		fireEvent.click(screen.getByRole('button', { name: 'Confirm previous review' }));
		expect(store.getSnapshot().current?.issue).toMatchObject({
			verdict: 'pass',
			carriedFrom: 'previous',
			sourceJobId: 'current',
			at: 'now'
		});
	});
	it('shows changed evidence as stale and offers no confirmation action', () => {
		const store = createReviewVerdictStore(null, () => 'now', {
			previous: { issue: { ...subject, evidenceSignature: 'old', verdict: 'pass', at: 'before' } }
		});
		render(
			<ReviewVerdictProvider store={store} context={context}>
				<ReviewNotesPanel jobId="current" issueId="issue" />
			</ReviewVerdictProvider>
		);
		expect(screen.queryByRole('button', { name: 'Confirm previous review' })).toBeNull();
		expect(screen.getByText(/Review this run again/)).toBeTruthy();
	});
	it('can save a note without asserting that a human check passed', () => {
		const store = createReviewVerdictStore(null, () => 'now');
		render(
			<ReviewVerdictProvider store={store} context={context}>
				<ReviewNotesPanel jobId="current" issueId="issue" />
			</ReviewVerdictProvider>
		);
		fireEvent.change(screen.getByLabelText('Review note (optional)'), {
			target: { value: 'Assigned to frontend' }
		});
		fireEvent.click(screen.getByRole('button', { name: 'Save note' }));
		expect(store.getSnapshot().current?.issue).toMatchObject({ note: 'Assigned to frontend' });
		expect(store.getSnapshot().current?.issue?.verdict).toBeUndefined();
	});
	it('does not remove a note-only human check from the pending queue', () => {
		const issue = {
			id: 'human',
			scanner: 'lighthouse',
			ruleId: 'manual-check',
			severity: 'info' as const,
			title: 'Manual review',
			description: 'Check manually',
			pageId: 'page',
			pageUrl: 'https://example.com',
			wcagTags: [],
			elementCount: 0
		};
		const status = getReviewGroupStatus([issue], () => ({ at: 'now', note: 'TODO' }));
		expect(status?.tone).toBe('pending');
	});
});

it('displays notes when a project store arrives after the modal opens', () => {
	const initiallyEmpty = createReviewVerdictStore(null);
	const loaded = createReviewVerdictStore(null, () => 'now', {
		current: { issue: { note: 'Saved in the project', at: 'before' } }
	});
	const view = render(
		<ReviewVerdictProvider store={initiallyEmpty} context={context}>
			<ReviewNotesPanel jobId="current" issueId="issue" />
		</ReviewVerdictProvider>
	);
	view.rerender(
		<ReviewVerdictProvider store={loaded} context={context}>
			<ReviewNotesPanel jobId="current" issueId="issue" />
		</ReviewVerdictProvider>
	);
	expect(screen.getByLabelText<HTMLTextAreaElement>('Review note (optional)').value).toBe(
		'Saved in the project'
	);
});

it('keeps browser-backed providers synchronized with reviews saved in another tab', () => {
	let stored: string | null = null;
	vi.stubGlobal('localStorage', {
		getItem: () => stored,
		setItem: (_key: string, value: string) => {
			stored = value;
		},
		removeItem: () => {
			stored = null;
		}
	});
	const browser = getBrowserReviewStore();
	browser.sync(null);
	const view = render(
		<ReviewVerdictProvider store={browser} context={context}>
			<ReviewNotesPanel jobId="current" issueId="issue" />
		</ReviewVerdictProvider>
	);
	const saved = JSON.stringify({ current: { issue: { note: 'Other tab note', at: 'today' } } });
	act(() => {
		window.localStorage.setItem(REVIEW_VERDICTS_STORAGE_KEY, saved);
		window.dispatchEvent(
			new StorageEvent('storage', {
				key: REVIEW_VERDICTS_STORAGE_KEY,
				newValue: saved
			})
		);
	});
	expect(screen.getByLabelText<HTMLTextAreaElement>('Review note (optional)').value).toBe(
		'Other tab note'
	);
	view.unmount();
	window.localStorage.removeItem(REVIEW_VERDICTS_STORAGE_KEY);
	browser.sync(null);
	vi.unstubAllGlobals();
});
