import { describe, expect, it } from 'vitest';
import type { ScanResult } from '../../types/scan';
import { applyScannerCompletionUpdate, normalizeScannerProgress } from './scanner-progress';
import { applySseUpdate } from './event-update';

describe('durable scanner states', () => {
	const result = {
		id: 'scan',
		created_at: '2026-10-02T00:00:00Z',
		updated_at: '2026-10-02T00:00:00Z',
		state: 'SCANNING',
		expected_scanners: ['axe', 'seo', 'lighthouse'],
		completed_scanners: ['axe'],
		scanner_statuses: {
			seo: {
				state: 'failed',
				pages_scanned: 0,
				total_pages: 30,
				violations: 0,
				error: 'browser closed'
			}
		}
	} as ScanResult;
	it('refresh excludes failures from remaining scanners while preserving successes', () => {
		expect(normalizeScannerProgress(result).remaining_scanners).toEqual(['lighthouse']);
		expect(normalizeScannerProgress(result).completed_scanners).toEqual(['axe']);
	});
	it('scanner failure settles the scanner without failing the job', () => {
		const update = {
			type: 'scanner_complete' as const,
			state: 'SCANNING',
			scanner_type: 'lighthouse',
			failed: true,
			error: 'timeout'
		};
		const next = applyScannerCompletionUpdate(applySseUpdate(result, update), update);
		expect(next.state).toBe('SCANNING');
		expect(next.error).toBeUndefined();
		expect(next.completed_scanners).toEqual(['axe']);
		expect(next.remaining_scanners).toBeUndefined();
		expect(next.scanner_statuses?.lighthouse?.state).toBe('failed');
	});
	it('tracks scanner page progress independently of aggregate progress', () => {
		const next = applyScannerCompletionUpdate(result, {
			type: 'progress',
			state: 'SCANNING',
			scanner_type: 'lighthouse',
			progress: { currentPage: 30, totalPages: 30 },
			scanner_progress: { currentPage: 28, totalPages: 30 }
		});
		expect(next.scanner_statuses?.lighthouse?.pages_scanned).toBe(28);
	});
});
