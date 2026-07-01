import type { ScanTiming } from './types';

export type ScanLifecyclePhaseDurations = Pick<
	ScanTiming,
	'pageIterationMs' | 'writeResultsMs' | 'uploadArtifactsMs'
>;

export function buildScanCompletedTiming(
	input: ScanLifecyclePhaseDurations & { totalMs: number }
): ScanTiming {
	const measuredMs = input.pageIterationMs + input.writeResultsMs + input.uploadArtifactsMs;

	return {
		totalMs: input.totalMs,
		pageIterationMs: input.pageIterationMs,
		writeResultsMs: input.writeResultsMs,
		uploadArtifactsMs: input.uploadArtifactsMs,
		publishCompletedMs: 0,
		finalizationMs: Math.max(0, input.totalMs - measuredMs)
	};
}

export function withPublishCompletedTiming(
	timing: ScanTiming,
	publishCompletedMs: number
): ScanTiming {
	return {
		...timing,
		publishCompletedMs,
		finalizationMs: timing.finalizationMs + publishCompletedMs
	};
}
