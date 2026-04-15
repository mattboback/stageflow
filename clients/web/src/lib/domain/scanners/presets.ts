import type { ScannerSelection } from '$lib/types/scan';

export type ScannerPreset = 'coverage' | 'quick' | 'custom';

const aiNavigatorId = 'ai-navigator';
const axeId = 'axe';

function getQuickScannerId(scanners: ScannerSelection[]): string | undefined {
	if (scanners.length === 0) {
		return undefined;
	}

	if (scanners.some((scanner) => scanner.id === axeId)) {
		return axeId;
	}

	return scanners[0]?.id;
}

function hasNonAiScanner(scanners: ScannerSelection[]): boolean {
	return scanners.some((scanner) => scanner.id !== aiNavigatorId);
}

export function applyScannerPreset(
	scanners: ScannerSelection[],
	preset: ScannerPreset
): ScannerSelection[] {
	if (preset === 'custom' || scanners.length === 0) {
		return scanners;
	}

	if (preset === 'quick') {
		const targetId = getQuickScannerId(scanners);
		return scanners.map((scanner) => ({
			...scanner,
			enabled: scanner.id === targetId
		}));
	}

	const allowNonAiOnly = hasNonAiScanner(scanners);
	let updated = scanners.map((scanner) => ({
		...scanner,
		enabled: allowNonAiOnly ? scanner.id !== aiNavigatorId : false
	}));

	if (!updated.some((scanner) => scanner.enabled)) {
		const fallbackId = scanners[0]?.id;
		updated = scanners.map((scanner) => ({
			...scanner,
			enabled: scanner.id === fallbackId
		}));
	}

	return updated;
}

export function detectScannerPreset(scanners: ScannerSelection[]): ScannerPreset {
	if (scanners.length === 0) {
		return 'custom';
	}

	const quickId = getQuickScannerId(scanners);
	const quickMatch = scanners.every((scanner) => scanner.enabled === (scanner.id === quickId));
	if (quickMatch) {
		return 'quick';
	}

	const allowNonAiOnly = hasNonAiScanner(scanners);
	if (allowNonAiOnly) {
		const coverageMatch = scanners.every(
			(scanner) => scanner.enabled === (scanner.id !== aiNavigatorId)
		);
		if (coverageMatch) {
			return 'coverage';
		}
	} else {
		const fallbackId = scanners[0]?.id;
		const fallbackMatch = scanners.every(
			(scanner) => scanner.enabled === (scanner.id === fallbackId)
		);
		if (fallbackMatch) {
			return 'coverage';
		}
	}

	return 'custom';
}
