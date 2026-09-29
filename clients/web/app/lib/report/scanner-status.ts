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

/**
 * Scanners that did not finish: they failed, or the job expected them and no
 * result came back. A report with any of them is partial. `partial` means the
 * scanner ran and some pages failed, so it still counts as finished. Matches
 * `diff.IncompleteScanners` in libs/go/diff.
 */
export function incompleteScanners(scanners: readonly ScannerSummary[] | undefined): string[] {
	return (scanners ?? [])
		.filter((scanner) => scanner.status === 'failed' || scanner.status === 'skipped')
		.map((scanner) => scanner.id)
		.sort();
}
