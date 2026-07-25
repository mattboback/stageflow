import { describe, expect, it } from 'vitest';

import { ISSUE_SORTS, ISSUE_SORT_LABELS, isIssueSortKey, sortIssues } from './sorting';
import { makeIssue } from './test-issues';

const issues = [
	makeIssue({
		id: 'moderate-b',
		title: 'Beta',
		scanner: 'lighthouse',
		pageUrl: 'https://example.com/z',
		severity: 'moderate',
		elementCount: 9
	}),
	makeIssue({
		id: 'critical',
		title: 'Zulu',
		scanner: 'axe',
		pageUrl: 'https://example.com/a',
		severity: 'critical',
		elementCount: 1
	}),
	makeIssue({
		id: 'moderate-a',
		title: 'Alpha',
		scanner: 'pa11y',
		pageUrl: 'https://example.com/m',
		severity: 'moderate',
		elementCount: 4
	})
];

const ids = (result: { id: string }[]) => result.map((issue) => issue.id);

describe('isIssueSortKey', () => {
	it('accepts every advertised sort key', () => {
		for (const key of ISSUE_SORTS) {
			expect(isIssueSortKey(key)).toBe(true);
		}
	});

	it('rejects an unknown key', () => {
		expect(isIssueSortKey('by-vibes')).toBe(false);
	});

	it('accepts null and undefined, which fall back to the default sort', () => {
		expect(isIssueSortKey(null)).toBe(true);
		expect(isIssueSortKey(undefined)).toBe(true);
	});

	it('has a label for every key, so the UI cannot render undefined', () => {
		for (const key of ISSUE_SORTS) {
			expect(ISSUE_SORT_LABELS[key]).toBeTruthy();
		}
		expect(Object.keys(ISSUE_SORT_LABELS).sort()).toEqual([...ISSUE_SORTS].sort());
	});
});

describe('sortIssues', () => {
	it('orders by severity, worst first, then by title', () => {
		expect(ids(sortIssues(issues, 'severity'))).toEqual(['critical', 'moderate-a', 'moderate-b']);
	});

	it('orders by page URL', () => {
		expect(ids(sortIssues(issues, 'page'))).toEqual(['critical', 'moderate-a', 'moderate-b']);
	});

	it('orders by scanner name', () => {
		expect(ids(sortIssues(issues, 'scanner'))).toEqual(['critical', 'moderate-b', 'moderate-a']);
	});

	it('orders by element count, highest first', () => {
		expect(ids(sortIssues(issues, 'count'))).toEqual(['moderate-b', 'moderate-a', 'critical']);
	});

	it('orders by title', () => {
		expect(ids(sortIssues(issues, 'title'))).toEqual(['moderate-a', 'moderate-b', 'critical']);
	});

	it('returns a new array and leaves the input untouched', () => {
		const input = [...issues];
		const sorted = sortIssues(input, 'title');

		expect(sorted).not.toBe(input);
		expect(ids(input)).toEqual(ids(issues));
	});

	it('handles an empty list', () => {
		for (const key of ISSUE_SORTS) {
			expect(sortIssues([], key)).toEqual([]);
		}
	});
});
