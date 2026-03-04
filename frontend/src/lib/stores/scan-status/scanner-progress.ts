import type { ScanResult } from '$lib/types/scan';

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

	return {
		...result,
		expected_scanners: expected,
		completed_scanners: completed,
		remaining_scanners: deriveRemainingScanners(expected, completed)
	};
}

export function applyScannerCompletionUpdate(result: ScanResult, update: SSEUpdate): ScanResult {
	const next = normalizeScannerProgress(result);

	if (update.type !== 'scanner_complete' || !update.scanner_type) {
		return next;
	}

	const completed = [...(next.completed_scanners ?? [])];
	if (!completed.includes(update.scanner_type)) {
		completed.push(update.scanner_type);
	}

	return {
		...next,
		completed_scanners: completed,
		remaining_scanners: deriveRemainingScanners(next.expected_scanners, completed)
	};
}
