export interface SSEUpdate {
	type: 'status' | 'progress' | 'complete' | 'failed';
	state: string;
	scanner_type?: string;
	progress?: { currentPage: number; totalPages: number };
	totalPages?: number;
	error?: string;
	error_details?: string;
	stage?: string;
}
