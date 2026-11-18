import type { ScannerStatus, ScannerSummary } from '../types/unified-report';

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

/** Incomplete page coverage is excluded on both sides of a comparison. Matches Go diff. */
export function incompleteScanners(scanners: readonly ScannerSummary[] | undefined): string[] {
	return (scanners ?? [])
		.filter(
			(scanner) =>
				scanner.status === 'failed' || scanner.status === 'skipped' || scanner.status === 'partial'
		)
		.map((scanner) => scanner.id)
		.sort();
}
