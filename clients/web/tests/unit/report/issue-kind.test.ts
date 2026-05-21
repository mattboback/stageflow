import type { IssueDetail } from '$lib/types/unified-report';

import {
	getIssueKind,
	getIssueKindLabel,
	isManualReviewIssue,
	rewriteIssueTitle
} from '$lib/report';
import { describe, expect, it } from 'vitest';

function makeIssue(overrides: Partial<IssueDetail> = {}): IssueDetail {
	return {
		id: 'issue-1',
		scanner: 'axe',
		ruleId: 'color-contrast',
		severity: 'serious',
		title: 'Low contrast text',
		description: 'Text elements have insufficient contrast ratio',
		pageId: 'page-1',
		pageUrl: 'http://example.com',
		elementCount: 1,
		occurrences: [
			{
				selector: '.hero',
				html: '<h1 class="hero">Title</h1>',
				failureSummary: 'Increase contrast'
			}
		],
		...overrides
	};
}

describe('getIssueKind', () => {
	it('classifies issues with selector/html as element-level', () => {
		expect(getIssueKind(makeIssue())).toBe('element');
	});

	it('classifies issues with ancestorPath as element-level', () => {
		expect(
			getIssueKind(
				makeIssue({
					occurrences: [{ ancestorPath: 'main > section > h1' }]
				})
			)
		).toBe('element');
	});

	it('classifies Lighthouse info issues with no occurrences as manual review', () => {
		expect(
			getIssueKind(
				makeIssue({
					scanner: 'lighthouse',
					severity: 'info',
					title: 'Interactive controls are keyboard focusable',
					occurrences: []
				})
			)
		).toBe('manual');
	});

	it('classifies issues whose description mentions manual verification as manual review', () => {
		expect(
			getIssueKind(
				makeIssue({
					scanner: 'lighthouse',
					severity: 'moderate',
					title: 'Verify focus order',
					description: 'Manual review required to confirm focus order is logical.',
					occurrences: []
				})
			)
		).toBe('manual');
	});

	it('classifies link-checker scanner issues with no DOM target as URL-level', () => {
		expect(
			getIssueKind(
				makeIssue({
					scanner: 'link-checker',
					ruleId: 'broken-link',
					severity: 'serious',
					occurrences: [{ failureSummary: '404 Not Found' }]
				})
			)
		).toBe('url');
	});

	it('classifies issues with no occurrences and no manual hints as page-level', () => {
		expect(
			getIssueKind(
				makeIssue({
					scanner: 'security-headers',
					ruleId: 'missing-csp',
					severity: 'serious',
					title: 'Missing Content Security Policy',
					description: 'No CSP header present.',
					occurrences: []
				})
			)
		).toBe('page');
	});

	it('classifies issues without a pageId and no occurrences as scanner diagnostics', () => {
		const issue = makeIssue({
			scanner: 'lighthouse',
			ruleId: 'pipeline-error',
			severity: 'moderate',
			title: 'Pipeline error',
			description: 'Scanner failed to start.',
			occurrences: []
		});
		issue.pageId = '';
		expect(getIssueKind(issue)).toBe('scanner');
	});
});

describe('isManualReviewIssue', () => {
	it('returns true for manual review classifications', () => {
		expect(
			isManualReviewIssue(
				makeIssue({
					scanner: 'lighthouse',
					severity: 'info',
					occurrences: []
				})
			)
		).toBe(true);
	});

	it('returns false for element-level issues', () => {
		expect(isManualReviewIssue(makeIssue())).toBe(false);
	});
});

describe('rewriteIssueTitle', () => {
	it('prefixes positive-sounding manual titles with "Verify:"', () => {
		const issue = makeIssue({
			scanner: 'lighthouse',
			severity: 'info',
			title: 'Interactive controls are keyboard focusable',
			occurrences: []
		});
		expect(rewriteIssueTitle(issue)).toBe('Verify: Interactive controls are keyboard focusable');
	});

	it('does not double-prefix titles already starting with Verify/Review/Manual/Check/Confirm', () => {
		const cases = [
			'Verify focus order',
			'Review color contrast manually',
			'Manual review: keyboard focus',
			'Check color contrast',
			'Confirm landmark structure'
		];
		for (const title of cases) {
			const issue = makeIssue({
				scanner: 'lighthouse',
				severity: 'info',
				title,
				occurrences: []
			});
			expect(rewriteIssueTitle(issue)).toBe(title);
		}
	});

	it('leaves non-manual issue titles unchanged', () => {
		const issue = makeIssue({ title: 'Low contrast text' });
		expect(rewriteIssueTitle(issue)).toBe('Low contrast text');
	});
});

describe('getIssueKindLabel', () => {
	it('returns human-readable labels for each kind', () => {
		expect(getIssueKindLabel('element')).toBe('Element issue');
		expect(getIssueKindLabel('url')).toBe('URL issue');
		expect(getIssueKindLabel('manual')).toBe('Manual review');
		expect(getIssueKindLabel('page')).toBe('Page issue');
		expect(getIssueKindLabel('scanner')).toBe('Scanner diagnostic');
	});
});
