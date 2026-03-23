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

interface BaseScreenshotArtifact {
	artifact_id: string;
	scanner_id: string;
	page_id: string;
	page_url?: string;
	url: string;
	thumbnail_url?: string;
	captured_at?: string;
}

export interface ViolationScreenshotArtifact extends BaseScreenshotArtifact {
	kind: 'violation';
	issue_id: string;
	occurrence_index: number;
}

export interface PageOverviewScreenshotArtifact extends BaseScreenshotArtifact {
	kind: 'page_overview';
	issue_id?: never;
	occurrence_index?: never;
}

export type ScreenshotArtifact = ViolationScreenshotArtifact | PageOverviewScreenshotArtifact;

interface ScannerArtifactBundle {
	scanner_type: string;
	results_json?: string;
	report_html?: string;
	scan_stage_log?: string;
	scan_recipe?: string;
}

interface ScanArtifacts {
	report_json: string;
	report_html: string;
	scan_stage_log?: string;
	scan_recipe?: string;
	extraction_stage_log?: string;
	extraction_recipe?: string;
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
	| 'loading'
	| 'error';

interface ScannerCapabilities {
	outputFormats: string[];
	supportsScreenshots: boolean;
	supportsConcurrency: boolean;
	requiresBrowser: boolean;
	supportsOffline: boolean;
	maxConcurrency: number;
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
