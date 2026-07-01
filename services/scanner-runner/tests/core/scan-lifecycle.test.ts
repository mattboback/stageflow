import { describe, expect, it } from 'vitest';

import {
	buildScanCompletedTiming,
	withPublishCompletedTiming
} from '../../src/core/scan-lifecycle';

describe('scan lifecycle timing', () => {
	it('builds scan.completed timing with publish duration pending', () => {
		const timing = buildScanCompletedTiming({
			totalMs: 1000,
			pageIterationMs: 400,
			writeResultsMs: 100,
			uploadArtifactsMs: 200
		});

		expect(timing).toEqual({
			totalMs: 1000,
			pageIterationMs: 400,
			writeResultsMs: 100,
			uploadArtifactsMs: 200,
			publishCompletedMs: 0,
			finalizationMs: 300
		});
	});

	it('does not emit negative finalization timing when measured phases exceed total', () => {
		const timing = buildScanCompletedTiming({
			totalMs: 100,
			pageIterationMs: 80,
			writeResultsMs: 40,
			uploadArtifactsMs: 20
		});

		expect(timing.finalizationMs).toBe(0);
	});

	it('adds publish duration to the timing used for lifecycle logs', () => {
		const timingForEvent = buildScanCompletedTiming({
			totalMs: 1000,
			pageIterationMs: 400,
			writeResultsMs: 100,
			uploadArtifactsMs: 200
		});

		const timingForLog = withPublishCompletedTiming(timingForEvent, 50);

		expect(timingForEvent.publishCompletedMs).toBe(0);
		expect(timingForEvent.finalizationMs).toBe(300);
		expect(timingForLog.publishCompletedMs).toBe(50);
		expect(timingForLog.finalizationMs).toBe(350);
	});
});
