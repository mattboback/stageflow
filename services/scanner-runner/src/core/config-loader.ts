/**
 * Config Loader
 *
 * Loads scanner configuration from environment variables.
 */

import { join } from 'node:path';

import type { BrowserConfig, MessagingConfig, ScannerConfig, StorageConfig } from './types';

export interface ConfigLoaderOptions {
	scannerName: string;
	defaults?: Partial<ScannerConfig>;
}

const DEFAULT_DATA_DIR = join(process.cwd(), 'data');

export function loadConfigFromEnv(options: ConfigLoaderOptions): ScannerConfig {
	const { scannerName, defaults } = options;
	const { requestId: _defaultRequestId, runId: _defaultRunId, ...defaultConfig } = defaults ?? {};

	const jobId = requireEnv('JOB_ID');
	const requestId = process.env.REQUEST_ID;
	const runId = process.env.RUN_ID;
	const dataDir = process.env.SCANNER_DATA_DIR ?? DEFAULT_DATA_DIR;

	const browser = loadBrowserConfig();
	const storage = loadStorageConfig();
	const messaging = loadMessagingConfig();
	const trimmedRequestId = requestId?.trim();
	const trimmedRunId = runId?.trim();

	const config: ScannerConfig = {
		...defaultConfig,
		jobId,
		provenancePath:
			process.env.PROVENANCE_PATH ?? defaults?.provenancePath ?? join(dataDir, 'provenance.json'),
		resultsDir: process.env.RESULTS_DIR ?? defaults?.resultsDir ?? join(dataDir, 'results'),
		scannerName,
		concurrency: getEnvInt('SCAN_CONCURRENCY', defaults?.concurrency ?? 4),
		maxRetries: getEnvInt('MAX_RETRIES', defaults?.maxRetries ?? 3),
		browser,
		storage,
		messaging,
		...(trimmedRequestId !== undefined ? { requestId: trimmedRequestId } : {}),
		...(trimmedRunId !== undefined ? { runId: trimmedRunId } : {})
	};

	const scannerOptions = loadScannerOptionsFromEnv();
	if (scannerOptions) {
		config.options = scannerOptions;
	}

	return config;
}

function loadBrowserConfig(): BrowserConfig {
	return {
		headless: getEnvBool('BROWSER_HEADLESS', true),
		args: [
			'--no-sandbox',
			'--disable-setuid-sandbox',
			'--disable-dev-shm-usage',
			'--disable-gpu',
			...(process.env.BROWSER_ARGS?.split(',').filter(Boolean) ?? [])
		],
		defaultViewport: {
			width: getEnvInt('VIEWPORT_WIDTH', 1280),
			height: getEnvInt('VIEWPORT_HEIGHT', 720)
		},
		deviceScaleFactor: getEnvNumber('DEVICE_SCALE_FACTOR', 2),
		defaultTimeout: getEnvInt('DEFAULT_TIMEOUT', 30_000),
		pageLoadTimeout: getEnvInt('PAGE_LOAD_TIMEOUT', 15_000),
		bypassCSP: getEnvBool('BROWSER_BYPASS_CSP', false)
	};
}

function loadStorageConfig(): StorageConfig {
	const endpoint = requireEnv('MINIO_ENDPOINT');
	const accessKey = requireAnyEnv(['MINIO_ACCESS_KEY', 'MINIO_ROOT_USER']);
	const secretKey = requireAnyEnv(['MINIO_SECRET_KEY', 'MINIO_ROOT_PASSWORD']);
	const bucket = requireEnv('MINIO_ARTIFACT_BUCKET');

	return {
		endpoint,
		accessKey,
		secretKey,
		useSSL: getEnvBool('MINIO_USE_SSL', false),
		bucket
	};
}

function loadMessagingConfig(): MessagingConfig {
	const url = requireEnv('NATS_URL');

	return {
		url,
		subjects: {
			pageCompleted: process.env.NATS_SUBJECT_PAGE_COMPLETED ?? 'scan.events.page.completed',
			scanCompleted: process.env.NATS_SUBJECT_SCAN_COMPLETED ?? 'scan.events.completed',
			scanFailed: process.env.NATS_SUBJECT_SCAN_FAILED ?? 'scan.events.failed'
		}
	};
}

function requireEnv(name: string): string {
	const value = process.env[name];
	if (!value) {
		throw new Error(`Required environment variable ${name} is not set`);
	}
	return value;
}

function requireAnyEnv(names: readonly string[]): string {
	for (const name of names) {
		const value = process.env[name];
		if (value) {
			return value;
		}
	}

	throw new Error(`Required environment variable not set (expected one of: ${names.join(', ')})`);
}

function getEnvBool(name: string, defaultValue: boolean): boolean {
	const raw = process.env[name];
	if (!raw) {
		return defaultValue;
	}
	const normalized = raw.trim().toLowerCase();
	if (['0', 'false', 'no', 'off'].includes(normalized)) {
		return false;
	}
	if (['1', 'true', 'yes', 'on'].includes(normalized)) {
		return true;
	}
	throw new Error(`Invalid boolean environment variable ${name}: ${raw}`);
}

function getEnvNumber(name: string, defaultValue: number): number {
	const raw = process.env[name];
	if (!raw) {
		return defaultValue;
	}
	const n = Number(raw);
	if (!Number.isFinite(n)) {
		throw new Error(`Invalid numeric environment variable ${name}: ${raw}`);
	}

	return n;
}

function getEnvInt(name: string, defaultValue: number): number {
	const raw = process.env[name];
	if (!raw) {
		return defaultValue;
	}
	const n = Number.parseInt(raw, 10);
	if (!Number.isInteger(n) || String(n) !== raw.trim() || n < 0) {
		throw new Error(`Invalid integer environment variable ${name}: ${raw}`);
	}

	return n;
}

function loadScannerOptionsFromEnv(): Record<string, unknown> | undefined {
	const raw = process.env.SCANNER_OPTIONS;
	if (!raw) {
		return undefined;
	}

	let parsed: unknown;
	try {
		parsed = JSON.parse(raw);
	} catch (err) {
		const message = err instanceof Error ? err.message : String(err);
		throw new Error(`Failed to parse SCANNER_OPTIONS as JSON: ${message}`, { cause: err });
	}

	if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
		throw new Error('SCANNER_OPTIONS must be a JSON object');
	}

	return parsed as Record<string, unknown>;
}

export function validateConfig(config: ScannerConfig): void {
	const errors: string[] = [];

	if (!config.jobId) {
		errors.push('jobId is required');
	}

	if (!config.provenancePath) {
		errors.push('provenancePath is required');
	}

	if (!config.resultsDir) {
		errors.push('resultsDir is required');
	}

	if (!config.scannerName) {
		errors.push('scannerName is required');
	}

	if (config.concurrency < 1) {
		errors.push('concurrency must be at least 1');
	}

	if (config.maxRetries < 1) {
		errors.push('maxRetries must be at least 1');
	}

	if (errors.length === 0) {
		return;
	}

	throw new Error(`Invalid scanner configuration:\n  - ${errors.join('\n  - ')}`);
}
