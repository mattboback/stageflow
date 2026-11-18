import { describe, expect, it } from 'vitest';

import { buildScanTiming } from '../../src/core/scan-lifecycle';

describe('scan timing', () => {
	it('assigns the unmeasured remainder to finalization', () => {
		expect(buildScanTiming({ totalMs: 1000, pageIterationMs: 400, writeResultsMs: 100 })).toEqual({
			totalMs: 1000,
			pageIterationMs: 400,
			writeResultsMs: 100,
			finalizationMs: 500
		});
	});

	it('does not emit negative finalization timing when measured phases exceed total', () => {
		const timing = buildScanTiming({ totalMs: 100, pageIterationMs: 80, writeResultsMs: 40 });

		expect(timing.finalizationMs).toBe(0);
	});
});
