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

	it('parses heading order context from axe detail line', () => {
		const detail = 'Heading order invalid (h4 follows h1)';
		expect(parseHeadingOrderContext(detail)).toEqual({ previousLevel: 1, currentLevel: 4 });
	});

	it('formats heading order flow with missing levels', () => {
		const flow = formatHeadingOrderFlow({ previousLevel: 1, currentLevel: 4 });
		expect(flow).toBe('H1 -> H4 (missing H2-H3)');
	});
});
