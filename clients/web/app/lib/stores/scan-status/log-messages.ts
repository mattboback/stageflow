import type { ScanStatus } from '../../types/scan';
import type { ScannerTiming } from '../../types/scan';

export interface StatusProgressLike {
	current_page?: number;
	total_pages?: number;
	currentPage?: number;
	totalPages?: number;
}

export interface StatusLike {
	type?: string;
	progress?: StatusProgressLike;
	scanner_type?: string;
	pages_scanned?: number;
	violations?: number;
	timing?: ScannerTiming;
	error?: string;
	error_details?: string;
}

function formatScannerTiming(timing?: ScannerTiming): string | null {
	if (!timing || timing.total_ms <= 0) return null;

	return `${(timing.total_ms / 1000).toFixed(1)}s`;
}

export function formatErrorDetails(details?: string): string | null {
	if (!details) return null;

	const compact = details.replace(/\s+/g, ' ').trim();
	if (!compact) return null;

	return compact.length > 160 ? `${compact.slice(0, 157)}…` : compact;
}

export function normalizeStatus(raw: string | undefined): ScanStatus {
	if (!raw) return 'pending';

	const value = raw.toLowerCase();
	if (value === 'done') return 'complete';

	if (value === 'failed') return 'failed';

	if (value === 'pending' || value === 'ready_to_scan') return 'pending';

	if (value === 'scanning' || value === 'extracting' || value === 'completing') {
		return value as ScanStatus;
	}

	return 'processing';
}

export function getLogMessage(normalizedState: string, data: StatusLike): string | null {
	switch (normalizedState) {
		case 'PENDING':
			return 'Job queued. Waiting for worker allocation...';
		case 'EXTRACTING':
			return 'Worker allocated. Extracting site artifacts...';
		case 'READY_TO_SCAN':
			return 'Extraction complete. Analyzing directory structure...';
		case 'SCANNING': {
			if (data.type === 'scanner_complete' && data.scanner_type) {
				const timing = formatScannerTiming(data.timing);
				const pages = data.pages_scanned;
				const issues = data.violations;
				const details = [
					timing,
					pages !== undefined ? `${pages} page${pages === 1 ? '' : 's'}` : null,
					issues !== undefined ? `${issues} issue${issues === 1 ? '' : 's'}` : null
				].filter(Boolean);
				if (details.length > 0) {
					return `[${data.scanner_type}] Complete in ${details.join(', ')}.`;
				}
				return `[${data.scanner_type}] Scanner complete. Waiting for remaining scanners...`;
			}

			const currentPage = data.progress?.current_page ?? data.progress?.currentPage;
			const totalPages = data.progress?.total_pages ?? data.progress?.totalPages;
			const scannerTag = data.scanner_type ? `[${data.scanner_type}]` : '[Scanner]';
			if (currentPage !== undefined && totalPages !== undefined) {
				if (currentPage <= 0 && totalPages > 0) {
					return `${scannerTag} Starting scan (1/${totalPages})...`;
				}
				return `${scannerTag} Visiting page ${currentPage}/${totalPages}...`;
			}
			return `${scannerTag} Scan is running. Waiting for page progress...`;
		}
		case 'COMPLETING':
			if (data.scanner_type) {
				return `[${data.scanner_type}] Scanner complete. Waiting for remaining scanners and aggregation...`;
			}
			return 'Aggregating scanner outputs and generating reports...';
		case 'DONE':
			return 'Workflow complete. Cleaning up ephemeral resources.';
		case 'FAILED': {
			const details = formatErrorDetails(data.error_details);
			const errorMessage = data.error ?? 'Unknown error';
			return `CRITICAL: Job failed - ${errorMessage}${details ? ` (${details})` : ''}`;
		}
		default:
			return null;
	}
}
