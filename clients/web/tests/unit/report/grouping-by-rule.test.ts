import { describe, expect, it } from 'vitest';

import { groupIssuesByRule } from '$lib/report/grouping';
import type { IssueDetail } from '$lib/types/unified-report';

function issue(overrides: Partial<IssueDetail>): IssueDetail {
	return {
		id: 'i1',
		scanner: 'axe',
		ruleId: 'image-alt',
		title: 'Images must have alternative text',
		description: 'desc',
		severity: 'serious',
		pageId: 'p1',
		impact: 'serious',
		occurrences: [],
		...overrides
	} as IssueDetail;
}

describe('groupIssuesByRule', () => {
	it('returns empty array on empty input', () => {
		expect(groupIssuesByRule([])).toEqual([]);
	});

	it('aggregates occurrences with stable fingerprint', () => {
		const groups = groupIssuesByRule([
			issue({ id: 'a', pageId: 'p1' }),
			issue({ id: 'b', pageId: 'p2' }),
			issue({ id: 'c', pageId: 'p1' })
		]);
		expect(groups).toHaveLength(1);
		expect(groups[0].fingerprint).toBe('axe:image-alt');
		expect(groups[0].occurrences).toHaveLength(3);
		expect(groups[0].pageIds.sort()).toEqual(['p1', 'p2']);
	});

	it('separates groups by scanner:ruleId', () => {
		const groups = groupIssuesByRule([
			issue({ ruleId: 'image-alt', scanner: 'axe' }),
			issue({ ruleId: 'image-alt', scanner: 'lighthouse' }),
			issue({ ruleId: 'label', scanner: 'axe' })
		]);
		expect(groups.map((g) => g.fingerprint).sort()).toEqual([
			'axe:image-alt',
			'axe:label',
			'lighthouse:image-alt'
		]);
	});

	it('rolls up max severity', () => {
		const groups = groupIssuesByRule([
			issue({ id: 'a', severity: 'minor' }),
			issue({ id: 'b', severity: 'critical' }),
			issue({ id: 'c', severity: 'moderate' })
		]);
		expect(groups[0].severity).toBe('critical');
	});

	it('sorts by severity desc, then occurrence count desc, then fingerprint asc', () => {
		const groups = groupIssuesByRule([
			issue({ id: '1', ruleId: 'r-low', severity: 'minor' }),
			issue({ id: '2', ruleId: 'r-low', severity: 'minor' }),
			issue({ id: '3', ruleId: 'r-high', severity: 'critical' }),
			issue({ id: '4', ruleId: 'r-mid-a', severity: 'serious' }),
			issue({ id: '5', ruleId: 'r-mid-b', severity: 'serious' }),
			issue({ id: '6', ruleId: 'r-mid-b', severity: 'serious' })
		]);
		expect(groups.map((g) => g.fingerprint)).toEqual([
			'axe:r-high',
			'axe:r-mid-b',
			'axe:r-mid-a',
			'axe:r-low'
		]);
	});

	it('produces stable fingerprints across reordered inputs', () => {
		const a = groupIssuesByRule([
			issue({ id: '1', ruleId: 'foo' }),
			issue({ id: '2', ruleId: 'bar' })
		]);
		const b = groupIssuesByRule([
			issue({ id: '2', ruleId: 'bar' }),
			issue({ id: '1', ruleId: 'foo' })
		]);
		expect(a.map((g) => g.fingerprint)).toEqual(b.map((g) => g.fingerprint));
	});

	it('deduplicates pageIds', () => {
		const groups = groupIssuesByRule([
			issue({ id: 'a', pageId: 'p1' }),
			issue({ id: 'b', pageId: 'p1' }),
			issue({ id: 'c', pageId: 'p2' })
		]);
		expect(groups[0].pageIds.sort()).toEqual(['p1', 'p2']);
	});

	it('unions WCAG tags across occurrences', () => {
		const groups = groupIssuesByRule([
			issue({ id: 'a', wcagTags: ['wcag2aa', '1.1.1'] }),
			issue({ id: 'b', wcagTags: ['1.1.1', 'best-practice'] })
		]);
		expect(groups[0].wcagTags?.sort()).toEqual(['1.1.1', 'best-practice', 'wcag2aa']);
	});
});
