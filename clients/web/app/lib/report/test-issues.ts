import type { IssueDetail } from '../types/unified-report';

/**
 * Builds a contract-complete IssueDetail so tests cannot accidentally assert
 * against a shape the API would never send. Every required field has a default;
 * override only what the test is actually about.
 *
 * Motivated by a real defect: the Playwright specs built issues without the
 * required pageId/pageUrl/elementCount, and one assertion had been calibrated
 * against that malformed data rather than against the contract.
 */
export function makeIssue(overrides: Partial<IssueDetail> = {}): IssueDetail {
	return {
		id: 'issue-1',
		scanner: 'axe',
		ruleId: 'color-contrast',
		severity: 'serious',
		title: 'Elements must have sufficient color contrast',
		description: 'Ensure the contrast between foreground and background colors meets AA.',
		pageId: 'page-1',
		pageUrl: 'https://example.com/',
		elementCount: 1,
		...overrides
	};
}
