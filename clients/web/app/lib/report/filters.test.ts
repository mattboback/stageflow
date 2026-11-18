import { describe, expect, it } from 'vitest';

import { filterIssues } from './filters';
import { makeIssue } from './test-issues';

const issues = [
	makeIssue({
		id: 'a',
		scanner: 'axe',
		ruleId: 'color-contrast',
		severity: 'serious',
		title: 'Insufficient contrast',
		description: 'Foreground and background are too close.',
		pageId: 'home',
		pageUrl: 'https://example.com/',
		category: 'color',
		wcagTags: ['wcag2aa', 'wcag143']
	}),
	makeIssue({
		id: 'b',
		scanner: 'lighthouse',
		ruleId: 'image-alt',
		severity: 'critical',
		title: 'Image missing alt text',
		description: 'Screen readers announce nothing for this image.',
		pageId: 'pricing',
		pageUrl: 'https://example.com/pricing',
		category: 'text-alternatives',
		wcagTags: ['wcag2a']
	}),
	makeIssue({
		id: 'c',
		scanner: 'axe',
		ruleId: 'label',
		severity: 'moderate',
		title: 'Form field has no label',
		description: 'The input cannot be identified.',
		pageId: 'home',
		pageUrl: 'https://example.com/'
	})
];

const ids = (result: { id: string }[]) => result.map((issue) => issue.id);

describe('filterIssues', () => {
	it('returns everything when no filter is set', () => {
		expect(filterIssues(issues, {})).toEqual(issues);
	});

	// Callers pass URLSearchParams.get() results straight through, so `null` is the
	// shape an absent filter actually arrives in. exactOptionalPropertyTypes means
	// an omitted key and an explicit `undefined` are not interchangeable, so
	// omission is covered by the case above and `null` is covered here.
	it('treats null filter values as absent', () => {
		expect(
			filterIssues(issues, {
				scannerId: null,
				pageId: null,
				severity: null,
				category: null,
				query: null
			})
		).toEqual(issues);
	});

	it('filters by each field independently', () => {
		expect(ids(filterIssues(issues, { scannerId: 'axe' }))).toEqual(['a', 'c']);
		expect(ids(filterIssues(issues, { pageId: 'home' }))).toEqual(['a', 'c']);
		expect(ids(filterIssues(issues, { severity: 'critical' }))).toEqual(['b']);
		expect(ids(filterIssues(issues, { category: 'color' }))).toEqual(['a']);
	});

	it('intersects filters rather than unioning them', () => {
		expect(ids(filterIssues(issues, { scannerId: 'axe', severity: 'moderate' }))).toEqual(['c']);
		expect(filterIssues(issues, { scannerId: 'axe', severity: 'critical' })).toEqual([]);
	});

	it('matches an unknown value against nothing instead of falling back to everything', () => {
		expect(filterIssues(issues, { scannerId: 'pa11y' })).toEqual([]);
	});

	describe('query', () => {
		it('searches title, description, ruleId, pageUrl, and scanner', () => {
			expect(ids(filterIssues(issues, { query: 'contrast' }))).toEqual(['a']);
			expect(ids(filterIssues(issues, { query: 'screen readers' }))).toEqual(['b']);
			expect(ids(filterIssues(issues, { query: 'image-alt' }))).toEqual(['b']);
			expect(ids(filterIssues(issues, { query: '/pricing' }))).toEqual(['b']);
			expect(ids(filterIssues(issues, { query: 'lighthouse' }))).toEqual(['b']);
		});

		it('searches optional category and wcagTags without tripping on their absence', () => {
			expect(ids(filterIssues(issues, { query: 'text-alternatives' }))).toEqual(['b']);
			expect(ids(filterIssues(issues, { query: 'wcag143' }))).toEqual(['a']);
			// Issue 'c' has neither field; a query that matches nothing must not throw.
			expect(filterIssues(issues, { query: 'nonexistent-token' })).toEqual([]);
		});

		it('is case-insensitive and ignores surrounding whitespace', () => {
			expect(ids(filterIssues(issues, { query: '  CONTRAST  ' }))).toEqual(['a']);
		});

		it('treats a whitespace-only query as no query at all', () => {
			expect(filterIssues(issues, { query: '   ' })).toEqual(issues);
			expect(filterIssues(issues, { query: '' })).toEqual(issues);
		});
	});

	it('does not mutate the input array', () => {
		const input = [...issues];
		filterIssues(input, { scannerId: 'axe', query: 'contrast' });
		expect(input).toEqual(issues);
	});
});
