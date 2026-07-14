import { describe, expect, it } from 'vitest';

import type { IssueDetail } from '../types/unified-report';
import { needsHumanReview } from './review';

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
