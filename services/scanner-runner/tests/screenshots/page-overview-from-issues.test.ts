import { describe, expect, it } from 'vitest';

import type { Issue } from '../../src/core/types';

import { issuesToPageOverviewViolations } from '../../src/screenshots/page-overview-from-issues';

function issue(overrides: Partial<Issue>): Issue {
	return {
		id: 'rule',
		scanner: 'link-checker',
		severity: 'moderate',
		category: 'links',
		title: 'title',
		description: 'desc',
		...overrides
	};
}

describe('issuesToPageOverviewViolations', () => {
	it('maps an issue with target nodes into a violation with targets', () => {
		const result = issuesToPageOverviewViolations([
			issue({
				id: 'broken-404',
				severity: 'serious',
				metadata: { nodes: [{ target: ['a:nth-of-type(2)'] }, { target: ['#nav > a'] }] }
			})
		]);

		expect(result).toEqual([
			{
				id: 'broken-404',
				impact: 'serious',
				nodes: [{ target: ['a:nth-of-type(2)'] }, { target: ['#nav > a'] }]
			}
		]);
	});

	it('falls back to node.selector when target is absent', () => {
		const result = issuesToPageOverviewViolations([
			issue({ metadata: { nodes: [{ selector: 'main > a' }] } })
		]);

		expect(result[0]?.nodes).toEqual([{ target: ['main > a'] }]);
	});

	it('omits nodes for page-global issues with no selector (page-level fallback)', () => {
		const result = issuesToPageOverviewViolations([
			issue({ id: 'missing-csp', metadata: { header: 'content-security-policy' } })
		]);

		expect(result).toEqual([{ id: 'missing-csp', impact: 'moderate' }]);
		expect(result[0]).not.toHaveProperty('nodes');
	});

	it('drops empty or non-string selectors', () => {
		const result = issuesToPageOverviewViolations([
			issue({ metadata: { nodes: [{ target: ['', '  '] }, { selector: '' }] } })
		]);

		expect(result[0]).not.toHaveProperty('nodes');
	});
});
