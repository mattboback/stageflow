import type { ScanResult } from '../../types/scan';

import type { SSEUpdate } from './types';

function cloneScanners(values?: string[]): string[] | undefined {
	if (!values || values.length === 0) {
		return undefined;
	}

	return [...values];
}

export function deriveRemainingScanners(
	expected?: string[],
	completed?: string[]
): string[] | undefined {
	if (!expected || expected.length === 0) {
		return undefined;
	}

	const completedSet = new Set((completed ?? []).filter(Boolean));
	const remaining = expected.filter((scannerType) => scannerType && !completedSet.has(scannerType));

	return remaining.length > 0 ? remaining : undefined;
}

export function normalizeScannerProgress(result: ScanResult): ScanResult {
	const expected = cloneScanners(result.expected_scanners);
	const completed = cloneScanners(result.completed_scanners);
	const settled = [
		...(completed ?? []),
		...Object.entries(result.scanner_statuses ?? {})
			.filter(([, scanner]) => scanner.state === 'failed')
			.map(([name]) => name)
	];
	const remaining = deriveRemainingScanners(expected, settled);

	const normalized = { ...result };
	delete normalized.remaining_scanners;
	return {
		...normalized,
		...(expected !== undefined ? { expected_scanners: expected } : {}),
		...(completed !== undefined ? { completed_scanners: completed } : {}),
		...(remaining !== undefined ? { remaining_scanners: remaining } : {})
	};
}

export function applyScannerCompletionUpdate(result: ScanResult, update: SSEUpdate): ScanResult {
	const next = normalizeScannerProgress(result);

	if ((update.type !== 'scanner_complete' && update.type !== 'progress') || !update.scanner_type) {
		return next;
	}

	const previous = next.scanner_statuses?.[update.scanner_type];
	const pages = update.scanner_progress ?? update.progress;
	const scanner = {
		state:
			update.type === 'progress'
				? ('running' as const)
				: update.failed
					? ('failed' as const)
					: ('completed' as const),
		pages_scanned: update.pages_scanned ?? pages?.currentPage ?? previous?.pages_scanned ?? 0,
		total_pages: pages?.totalPages ?? previous?.total_pages ?? next.progress?.total_pages ?? 0,
		violations: update.scanner_violations ?? previous?.violations ?? 0,
		...(update.timing ? { total_ms: update.timing.total_ms } : {}),
		...(update.failed && update.error ? { error: update.error } : {})
	};
	const completed = [...(next.completed_scanners ?? [])];
	if (scanner.state === 'completed' && !completed.includes(update.scanner_type))
		completed.push(update.scanner_type);
	return normalizeScannerProgress({
		...next,
		completed_scanners: completed,
		scanner_statuses: { ...next.scanner_statuses, [update.scanner_type]: scanner }
	});
}
