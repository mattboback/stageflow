import { describe, expect, it } from 'vitest';

import type { IssueDetail } from '../types/unified-report';
import type { ReviewVerdict } from './review-verdict';
import { getReviewGroupStatus, needsHumanReview } from './review';

function issue(overrides: Partial<IssueDetail> = {}): IssueDetail {
	return {
		id: 'issue-1',
		scanner: 'axe',
		ruleId: 'button-name',
		severity: 'serious',
		title: 'Buttons must have names',
		description: 'Check the button name.',
		pageId: 'page-1',
		pageUrl: 'https://example.com',
		elementCount: 0,
		occurrences: [],
		...overrides
	};
}

describe('needsHumanReview', () => {
	it('includes Lighthouse informational checks without concrete targets', () => {
		expect(
			needsHumanReview(
				issue({ scanner: 'lighthouse', ruleId: 'custom-controls-labels', severity: 'info' })
			)
		).toBe(true);
	});

	it('includes axe-incomplete contrast findings even when they have a DOM occurrence', () => {
		expect(
			needsHumanReview(
				issue({
					ruleId: 'color-contrast',
					scannerData: { axeIncomplete: true },
					occurrences: [{ selector: '.hero', elementId: 'hero' }]
				})
			)
		).toBe(true);
	});

	it('excludes automatically determined contrast failures and ordinary findings', () => {
		expect(needsHumanReview(issue({ ruleId: 'color-contrast' }))).toBe(false);
		expect(needsHumanReview(issue())).toBe(false);
	});
});

describe('getReviewGroupStatus', () => {
	const issues = [
		issue({
			id: 'manual-1',
			scanner: 'lighthouse',
			ruleId: 'custom-controls-labels',
			severity: 'info'
		}),
		issue({
			id: 'manual-2',
			scanner: 'lighthouse',
			ruleId: 'custom-controls-labels',
			severity: 'info'
		})
	];
	const verdict = (decision: 'pass' | 'fail'): ReviewVerdict => ({
		verdict: decision,
		at: '2026-07-14T10:00:00.000Z'
	});

	it('distinguishes pending, completed, and mixed grouped decisions', () => {
		expect(getReviewGroupStatus(issues, () => null)).toEqual({
			label: '2 need review',
			tone: 'pending'
		});
		expect(
			getReviewGroupStatus(issues, (id) => (id === 'manual-1' ? verdict('pass') : null))
		).toEqual({ label: 'needs review', tone: 'pending' });
		expect(getReviewGroupStatus(issues, () => verdict('pass'))).toEqual({
			label: 'reviewed · pass',
			tone: 'pass'
		});
		expect(getReviewGroupStatus(issues, () => verdict('fail'))).toEqual({
			label: 'reviewed · fail',
			tone: 'fail'
		});
		expect(
			getReviewGroupStatus(issues, (id) => verdict(id === 'manual-1' ? 'pass' : 'fail'))
		).toEqual({ label: 'reviewed · mixed', tone: 'mixed' });
	});
});
