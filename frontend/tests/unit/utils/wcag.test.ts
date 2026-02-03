/**
 * WCAG Utility Tests
 */

import { extractWcagCriteria, getIssueWcagConformanceLevel } from '$lib/utils/wcag';
import { describe, expect, it } from 'vitest';

describe('wcag', () => {
	describe('getIssueWcagConformanceLevel', () => {
		it('extracts level A from wcag2a tag', () => {
			expect(getIssueWcagConformanceLevel(['wcag2a'], undefined)).toBe('A');
		});

		it('extracts level A from wcag21a tag', () => {
			expect(getIssueWcagConformanceLevel(['wcag21a'], undefined)).toBe('A');
		});

		it('extracts level A from wcag22a tag', () => {
			expect(getIssueWcagConformanceLevel(['wcag22a'], undefined)).toBe('A');
		});

		it('extracts level AA from tags', () => {
			expect(getIssueWcagConformanceLevel(['wcag21aa'], undefined)).toBe('AA');
			expect(getIssueWcagConformanceLevel(['wcag2aa'], undefined)).toBe('AA');
			expect(getIssueWcagConformanceLevel(['wcag22aa'], undefined)).toBe('AA');
		});

		it('extracts level AAA from tags', () => {
			expect(getIssueWcagConformanceLevel(['wcag21aaa'], undefined)).toBe('AAA');
			expect(getIssueWcagConformanceLevel(['wcag2aaa'], undefined)).toBe('AAA');
			expect(getIssueWcagConformanceLevel(['wcag22aaa'], undefined)).toBe('AAA');
		});

		it('returns null for non-WCAG tags', () => {
			expect(getIssueWcagConformanceLevel(['best-practice'], undefined)).toBeNull();
			expect(getIssueWcagConformanceLevel(['cat.color'], undefined)).toBeNull();
		});

		it('returns null for empty tags', () => {
			expect(getIssueWcagConformanceLevel([], undefined)).toBeNull();
		});

		it('returns null for undefined tags', () => {
			expect(getIssueWcagConformanceLevel(undefined, undefined)).toBeNull();
		});

		it('falls back to wcagRef when tags dont match', () => {
			expect(getIssueWcagConformanceLevel([], 'level AA')).toBe('AA');
			expect(getIssueWcagConformanceLevel(['best-practice'], 'Level A')).toBe('A');
		});

		it('parses level from wcagRef case-insensitively', () => {
			expect(getIssueWcagConformanceLevel([], 'LEVEL AAA')).toBe('AAA');
			expect(getIssueWcagConformanceLevel([], 'level aa')).toBe('AA');
		});

		it('tags take priority over wcagRef', () => {
			expect(getIssueWcagConformanceLevel(['wcag2a'], 'level AAA')).toBe('A');
		});

		it('handles mixed tags and extracts highest priority', () => {
			// AAA has lowest priority, should return first match
			expect(getIssueWcagConformanceLevel(['wcag2aaa', 'wcag2aa'], undefined)).toBe('AAA');
		});
	});

	describe('extractWcagCriteria', () => {
		it('extracts criteria from compact wcag tags', () => {
			expect(extractWcagCriteria(['wcag111'], undefined)).toEqual(['1.1.1']);
		});

		it('extracts criteria from dotted wcag tags', () => {
			expect(extractWcagCriteria(['wcag1.4.3'], undefined)).toEqual(['1.4.3']);
		});

		it('extracts criteria from wcagRef', () => {
			expect(extractWcagCriteria([], 'WCAG 2.4.7')).toEqual(['2.4.7']);
		});

		it('deduplicates criteria across tags and refs', () => {
			expect(extractWcagCriteria(['wcag111', 'wcag1.1.1'], 'WCAG 1.1.1')).toEqual(['1.1.1']);
		});

		it('returns empty list when no criteria are present', () => {
			expect(extractWcagCriteria(undefined, undefined)).toEqual([]);
		});
	});
});
