import type { ScannerSelection } from '$lib/types/scan';

import {
	applyScannerPreset,
	detectScannerPreset
} from '$lib/components/playground/scanner-presets';
import { describe, expect, it } from 'vitest';

describe('scanner presets', () => {
	it('applies coverage by enabling all non-AI scanners', () => {
		const selections: ScannerSelection[] = [
			{ id: 'axe', enabled: false },
			{ id: 'lighthouse', enabled: false },
			{ id: 'ai-navigator', enabled: false }
		];

		const result = applyScannerPreset(selections, 'coverage');

		expect(result.find((scanner) => scanner.id === 'axe')?.enabled).toBe(true);
		expect(result.find((scanner) => scanner.id === 'lighthouse')?.enabled).toBe(true);
		expect(result.find((scanner) => scanner.id === 'ai-navigator')?.enabled).toBe(false);
	});

	it('applies quick by enabling only axe when present', () => {
		const selections: ScannerSelection[] = [
			{ id: 'axe', enabled: true },
			{ id: 'lighthouse', enabled: true }
		];

		const result = applyScannerPreset(selections, 'quick');

		expect(result.find((scanner) => scanner.id === 'axe')?.enabled).toBe(true);
		expect(result.find((scanner) => scanner.id === 'lighthouse')?.enabled).toBe(false);
	});

	it('detects custom when selection does not match a preset', () => {
		const selections: ScannerSelection[] = [
			{ id: 'axe', enabled: false },
			{ id: 'lighthouse', enabled: true },
			{ id: 'ai-navigator', enabled: false }
		];

		expect(detectScannerPreset(selections)).toBe('custom');
	});
});
