import type { ScannerSelection } from '../../types/scan';

export type ScannerPreset = 'coverage' | 'quick' | 'custom';

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

	// 'coverage': every scanner the catalog offered. It used to carve out the
	// opt-in AI navigator, which no longer exists.
	return scanners.map((scanner) => ({ ...scanner, enabled: true }));
}
