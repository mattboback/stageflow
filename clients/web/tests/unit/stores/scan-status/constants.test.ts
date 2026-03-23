/**
 * Scan Status Constants Tests
 */

import { MAX_LOG_LINES, MAX_SSE_PARSE_ERRORS } from '$lib/stores/scan-status/constants';
import { describe, expect, it } from 'vitest';

describe('scan-status constants', () => {
	it('MAX_LOG_LINES is 50', () => {
		expect(MAX_LOG_LINES).toBe(50);
	});

	it('MAX_SSE_PARSE_ERRORS is 3', () => {
		expect(MAX_SSE_PARSE_ERRORS).toBe(3);
	});
});
