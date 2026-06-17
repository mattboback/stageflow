import type { ScannerTiming } from '../../types/scan';

export interface SSEUpdate {
	type: 'status' | 'progress' | 'scanner_complete' | 'complete' | 'failed';
	state: string;
	scanner_type?: string;
	progress?: { currentPage: number; totalPages: number };
	totalPages?: number;
	pages_scanned?: number;
	violations?: number;
	timing?: ScannerTiming;
	error?: string;
	error_details?: string;
	stage?: string;
}
