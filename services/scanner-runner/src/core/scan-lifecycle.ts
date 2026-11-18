import type { ScanTiming } from './types';

/**
 * finalizationMs is whatever the measured phases leave over (provenance load,
 * teardown), so the phases never add up to more than totalMs. The orchestrator
 * rejects an outcome whose phases do.
 */
export function buildScanTiming(input: {
	totalMs: number;
	pageIterationMs: number;
	writeResultsMs: number;
}): ScanTiming {
	return {
		totalMs: input.totalMs,
		pageIterationMs: input.pageIterationMs,
		writeResultsMs: input.writeResultsMs,
		finalizationMs: Math.max(0, input.totalMs - input.pageIterationMs - input.writeResultsMs)
	};
}
