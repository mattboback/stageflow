import type { ScannerStatus } from '../types/unified-report';

export type ScannerStatusTone = 'success' | 'danger' | 'muted' | 'warning';

export function getScannerStatusTone(status?: ScannerStatus | null): ScannerStatusTone {
	switch (status) {
		case 'success':
			return 'success';
		case 'partial':
			return 'warning';
		case 'failed':
			return 'danger';
		case 'skipped':
			return 'muted';
		default:
			return 'warning';
	}
}

export function formatScannerStatus(status?: ScannerStatus | null): string {
	switch (status) {
		case 'success':
			return 'Success';
		case 'partial':
			return 'Partial';
		case 'failed':
			return 'Failed';
		case 'skipped':
			return 'Skipped';
		default:
			return 'Unknown';
	}
}
