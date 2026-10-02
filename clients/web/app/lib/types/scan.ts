interface ScanProgress {
	current_page: number;
	total_pages: number;
	percentage: number;
}

export interface ScannerTiming {
	total_ms: number;
	page_iteration_ms: number;
	write_results_ms: number;
	upload_artifacts_ms: number;
	publish_completed_ms: number;
	finalization_ms: number;
}

/** One full-page overview per scanned page; the report crops it per finding. */
export interface ScreenshotArtifact {
	kind: 'page_overview';
	artifact_id: string;
	scanner_id: string;
	page_id: string;
	page_url?: string;
	url: string;
	captured_at?: string;
}

interface ScannerArtifactBundle {
	scanner_type: string;
	results_json?: string;
}

interface ScanArtifacts {
	report_json: string;
	screenshots?: ScreenshotArtifact[];
	scanner_artifacts?: Record<string, ScannerArtifactBundle>;
}

export interface ScanResult {
	id: string;
	state: string;
	error?: string;
	error_details?: string;
	last_stage?: string;
	violations?: number;
	progress?: ScanProgress;
	expected_scanners?: string[];
	completed_scanners?: string[];
	remaining_scanners?: string[];
	scanner_statuses?: Record<string, ScannerStatus>;
	artifacts?: ScanArtifacts;
	created_at: string;
	updated_at: string;
}

export type ScanStatus =
	| 'pending'
	| 'processing'
	| 'extracting'
	| 'scanning'
	| 'completing'
	| 'complete'
	| 'failed'
	| 'cancelled'
	| 'loading'
	| 'error';

interface ScannerCapabilities {
	outputFormats: string[];
	supportsScreenshots: boolean;
	supportsConcurrency: boolean;
	requiresBrowser: boolean;
	supportsOffline: boolean;
	maxConcurrency: number;
	estimatedTimePerPage?: number;
}

export interface ScannerDefinition {
	id: string;
	name: string;
	version: string;
	description: string;
	categories: string[];
	aliases: string[];
	enabled: boolean;
	builtIn: boolean;
	capabilities: ScannerCapabilities;
	image?: string;
}

export interface ScannerSelection {
	id: string;
	enabled: boolean;
	config?: Record<string, unknown>;
}

export interface ScannersResponse {
	scanners: ScannerDefinition[];
	categories: string[];
}

export interface ScannerStatus {
	state: 'pending' | 'running' | 'completed' | 'failed';
	pages_scanned: number;
	total_pages: number;
	violations: number;
	total_ms?: number;
	error?: string;
}
