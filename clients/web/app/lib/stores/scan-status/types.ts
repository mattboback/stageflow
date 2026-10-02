import type { ScannerTiming } from '../../types/scan';

export interface SSEUpdate {
	type: 'status' | 'queue' | 'progress' | 'scanner_complete' | 'complete' | 'failed';
	state: string;
	queue_position?: number;
	scanner_type?: string;
	failed?: boolean;
	scanner_progress?: { currentPage: number; totalPages: number };
	progress?: { currentPage: number; totalPages: number };
	totalPages?: number;
	pages_scanned?: number;
	violations?: number;
	scanner_violations?: number;
	timing?: ScannerTiming;
	error?: string;
	error_details?: string;
	stage?: string;
}
