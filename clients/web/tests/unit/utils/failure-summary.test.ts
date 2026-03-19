/**
 * Failure Summary Utility Tests
 */

import {
	extractFailureDetails,
	extractPrimaryFailureDetail,
	formatHeadingOrderFlow,
	parseHeadingOrderContext
} from '$lib/utils/failure-summary';
import { describe, expect, it } from 'vitest';

describe('failure-summary', () => {
	it('strips headers, bullets, and numbered items', () => {
		const summary = 'Fix all of the following:\n- Missing alt text\n1. Missing form label';
		expect(extractFailureDetails(summary)).toEqual(['Missing alt text', 'Missing form label']);
	});

	it('falls back to original lines when details collapse to empty', () => {
		const summary = 'Fix any of the following:';
		expect(extractFailureDetails(summary)).toEqual(['Fix any of the following:']);
	});

	it('extracts actionable lines from axe-style summaries', () => {
		const summary = 'Fix any of the following:\n  Element does not have an alt attribute';
		expect(extractFailureDetails(summary)).toEqual(['Element does not have an alt attribute']);
	});

	it('preserves single-line summaries', () => {
		const summary =
			'Element has insufficient color contrast of 2.85:1 (foreground: #999999, background: #ffffff)';
		expect(extractFailureDetails(summary)).toEqual([summary]);
	});

	it('returns null when no failure summary exists', () => {
		expect(extractPrimaryFailureDetail(undefined)).toBeNull();
	});

	it('returns first detail when failure summary has multiple details', () => {
		const summary = 'Fix any of the following:\n- Missing alt text\n- Color contrast is too low';
		expect(extractPrimaryFailureDetail(summary)).toBe('Missing alt text');
	});

	it('returns null when heading order context is absent', () => {
		expect(parseHeadingOrderContext('No heading info here')).toBeNull();
	});

	it('parses heading order context from axe detail line', () => {
		const detail = 'Heading order invalid (h4 follows h1)';
		expect(parseHeadingOrderContext(detail)).toEqual({ previousLevel: 1, currentLevel: 4 });
	});

	it('formats heading order flow without missing headings when adjacent', () => {
		const flow = formatHeadingOrderFlow({ previousLevel: 2, currentLevel: 3 });
		expect(flow).toBe('H2 -> H3');
	});

	it('formats heading order flow with exactly one missing level', () => {
		const flow = formatHeadingOrderFlow({ previousLevel: 1, currentLevel: 3 });
		expect(flow).toBe('H1 -> H3 (missing H2)');
	});

	it('formats heading order flow with missing levels', () => {
		const flow = formatHeadingOrderFlow({ previousLevel: 1, currentLevel: 4 });
		expect(flow).toBe('H1 -> H4 (missing H2-H3)');
	});
});
