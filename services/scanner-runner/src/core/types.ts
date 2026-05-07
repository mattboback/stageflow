/**
 * Core type definitions for the Scanner Runner (scanner-runner).
 */

import type { BrowserContext, Page } from 'playwright';

import type { TargetValidationPolicy } from './target-validation';

export type WaitStrategyType = 'load' | 'domcontentloaded' | 'networkidle' | 'selector' | 'timeout';

export interface WaitStrategyLoad {
	type: 'load';
}

export interface WaitStrategyDOMContentLoaded {
	type: 'domcontentloaded';
}

export interface WaitStrategyNetworkIdle {
	type: 'networkidle';
}

export interface WaitStrategySelector {
	type: 'selector';
	selector: string;
	timeout?: number;
}

export interface WaitStrategyTimeout {
	type: 'timeout';
	ms: number;
}

export type WaitStrategy =
	| WaitStrategyLoad
	| WaitStrategyDOMContentLoaded
	| WaitStrategyNetworkIdle
	| WaitStrategySelector
	| WaitStrategyTimeout;

export const DEFAULT_WAIT_STRATEGY: WaitStrategy = { type: 'load' };

export type ActionType = 'click' | 'fill' | 'select' | 'hover' | 'wait' | 'scroll' | 'keyboard';

export interface ActionClick {
	type: 'click';
	selector: string;
	timeout?: number;
}

export interface ActionFill {
	type: 'fill';
	selector: string;
	value: string;
	timeout?: number;
}

export interface ActionSelect {
	type: 'select';
	selector: string;
	value: string;
	timeout?: number;
}

export interface ActionHover {
	type: 'hover';
	selector: string;
	timeout?: number;
}

export interface ActionWait {
	type: 'wait';
	ms: number;
}

export interface ActionScroll {
	type: 'scroll';
	selector?: string;
	direction?: 'up' | 'down';
	pixels?: number;
}

export interface ActionKeyboard {
	type: 'keyboard';
	key: string;
}

export type PreScanAction =
	| ActionClick
	| ActionFill
	| ActionSelect
	| ActionHover
	| ActionWait
	| ActionScroll
	| ActionKeyboard;

export type ProvenanceMode = 'static' | 'spa' | 'live';

export interface PageEntry {
	id: string;
	path: string;
	url: string;
	wait_for?: WaitStrategy;
	pre_scan_actions?: PreScanAction[];
	viewport?: { width: number; height: number };
	skip?: boolean;
	metadata?: Record<string, unknown>;
}

export interface Provenance {
	version: string;
	job_id: string;
	base_url: string;
	mode?: ProvenanceMode;
	default_wait_for?: WaitStrategy;
	default_viewport?: { width: number; height: number };
	pages: PageEntry[];
	metadata?: Record<string, unknown>;
}

export interface BrowserConfig {
	headless: boolean;
	args: string[];
	defaultViewport: { width: number; height: number };
	deviceScaleFactor: number;
	defaultTimeout: number;
	pageLoadTimeout: number;
	bypassCSP?: boolean;
}

export interface StorageConfig {
	endpoint: string;
	accessKey: string;
	secretKey: string;
	useSSL: boolean;
	bucket: string;
}

export interface MessagingConfig {
	url: string;
	subjects: {
		pageCompleted: string;
		scanCompleted: string;
		scanFailed: string;
	};
}

export interface ScannerConfig {
	jobId: string;
	requestId?: string;
	runId?: string;
	provenancePath: string;
	resultsDir: string;
	scannerName: string;
	concurrency: number;
	maxRetries: number;
	browser: BrowserConfig;
	storage: StorageConfig;
	messaging: MessagingConfig;
	options?: Record<string, unknown>;
}

export type IssueSeverity = 'critical' | 'serious' | 'moderate' | 'minor' | 'info';

export interface IssueLocation {
	selector?: string;
	xpath?: string;
	html?: string;
	line?: number;
	column?: number;
}

export interface Issue {
	id: string;
	scanner: string;
	severity: IssueSeverity;
	category: string;
	title: string;
	description: string;
	helpUrl?: string;
	location?: IssueLocation;
	screenshot?: string;
	metadata?: Record<string, unknown>;
}

export interface PageScanResult {
	pageId: string;
	url: string;
	path?: string;
	success: boolean;
	issues: Issue[];
	durationMs: number;
	startedAt: string;
	finishedAt: string;
	error?: string;
	retryable?: boolean;
	artifacts?: string[];
	rawResults?: unknown;
}

export interface LighthouseCategorySummary {
	id: 'performance' | 'accessibility' | 'best-practices' | 'seo' | 'pwa';
	title: string;
	avgScore: number;
}

export interface ScanSummary {
	totalIssues: number;
	bySeverity: Record<IssueSeverity, number>;
	byCategory: Record<string, number>;
	pagesScanned: number;
	pagesFailed: number;
	pagesWithIssues: number;
	avgDurationMs: number;
	lighthouseCategories?: LighthouseCategorySummary[];
}

export interface ScanResults {
	jobId: string;
	scanner: string;
	version: string;
	totalPages: number;
	pages: PageScanResult[];
	summary: ScanSummary;
	startedAt: string;
	completedAt: string;
	durationMs: number;
}

export interface ScanContext {
	page: Page;
	context: BrowserContext;
	pageEntry: PageEntry;
	resultsDir: string;
	config: ScannerConfig;
	logger: ScannerLogger;
	targetValidationPolicy?: TargetValidationPolicy;
}

export interface ScannerLifecycleHooks {
	onScanStart?(config: ScannerConfig): Promise<void>;
	onScanEnd?(results: ScanResults): Promise<void>;
	onPageStart?(pageEntry: PageEntry): Promise<void>;
	onPageEnd?(result: PageScanResult): Promise<void>;
	onError?(error: Error, context?: { pageEntry?: PageEntry }): Promise<void>;
}

export interface ScannerLogger {
	info(message: string, meta?: Record<string, unknown>): void;
	warn(message: string, meta?: Record<string, unknown>): void;
	error(message: string, meta?: Record<string, unknown>): void;
	debug(message: string, meta?: Record<string, unknown>): void;
}

export interface ScanEventPublisher {
	publishPageCompleted(result: PageScanResult, index: number, total: number): Promise<void>;
	publishScanCompleted(
		results: ScanResults,
		timing?: ScanTiming,
		artifacts?: {
			stageLogPath?: string;
			recipePath?: string;
			reportPath?: string;
		}
	): Promise<void>;
	publishScanFailed(
		error: string,
		details?: string,
		artifacts?: { stageLogPath?: string; recipePath?: string }
	): Promise<void>;
	close(): Promise<void>;
}

export interface ScanTiming {
	totalMs: number;
	pageIterationMs: number;
	writeResultsMs: number;
	uploadArtifactsMs: number;
	publishCompletedMs: number;
	finalizationMs: number;
}

export interface StorageProvider {
	ensureBucket(bucket: string): Promise<void>;
	upload(bucket: string, key: string, filePath: string, contentType?: string): Promise<void>;
	uploadBuffer(bucket: string, key: string, data: Buffer, contentType?: string): Promise<void>;
	uploadDirectory(bucket: string, prefix: string, localDir: string): Promise<number>;
	download(bucket: string, key: string, destPath: string): Promise<void>;
	exists(bucket: string, key: string): Promise<boolean>;
}

export interface ScannerMetadata {
	name: string;
	version: string;
	description?: string;
}

export type ScannerCategory =
	| 'accessibility'
	| 'performance'
	| 'security'
	| 'seo'
	| 'quality'
	| 'custom';

export type OutputFormat = 'json' | 'html' | 'csv';
