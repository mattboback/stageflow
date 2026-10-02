import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { ScannerConfig } from '../../src/core/types';

import { loadConfigFromEnv, validateConfig } from '../../src/core/config-loader';

const ORIGINAL_ENV = process.env;

function resetEnv(): void {
	process.env = { ...ORIGINAL_ENV };
	delete process.env.NODE_ENV;
}

function setRequiredRuntimeEnv(): void {
	process.env.NATS_URL = 'nats://nats.example.com:4222';
}

describe('config-loader', () => {
	beforeEach(() => {
		resetEnv();
	});

	afterEach(() => {
		resetEnv();
		vi.restoreAllMocks();
	});

	it('throws when required env vars are missing', () => {
		delete process.env.JOB_ID;
		expect(() => loadConfigFromEnv({ scannerName: 'axe' })).toThrow(
			'Required environment variable JOB_ID is not set'
		);
	});

	it('parses scanner options from JSON and applies defaults', () => {
		setRequiredRuntimeEnv();
		process.env.JOB_ID = 'job-123';
		process.env.SCANNER_OPTIONS = JSON.stringify({ locale: 'en-US' });

		const config = loadConfigFromEnv({
			scannerName: 'axe',
			defaults: {
				concurrency: 2,
				maxRetries: 5,
				resultsDir: '/tmp/results',
				provenancePath: '/tmp/provenance.json'
			}
		});

		expect(config.jobId).toBe('job-123');
		expect(config.concurrency).toBe(2);
		expect(config.maxRetries).toBe(5);
		expect(config.options).toEqual({ locale: 'en-US' });
	});

	describe('runtime environment validation', () => {
		it('throws when NATS_URL is missing', () => {
			process.env.JOB_ID = 'job-123';
			setRequiredRuntimeEnv();
			delete process.env.NATS_URL;

			expect(() => loadConfigFromEnv({ scannerName: 'axe' })).toThrow(
				'Required environment variable NATS_URL is not set'
			);
		});

		it('succeeds when all required env vars are set', () => {
			process.env.JOB_ID = 'job-123';
			setRequiredRuntimeEnv();

			const config = loadConfigFromEnv({ scannerName: 'axe' });

			expect(config.messaging.url).toBe('nats://nats.example.com:4222');
		});
	});

	it('throws when scanner options JSON is invalid', () => {
		setRequiredRuntimeEnv();
		process.env.JOB_ID = 'job-123';
		process.env.SCANNER_OPTIONS = '{nope';

		expect(() => loadConfigFromEnv({ scannerName: 'axe' })).toThrow(
			'Failed to parse SCANNER_OPTIONS'
		);
	});

	it('throws when scanner options are not an object', () => {
		setRequiredRuntimeEnv();
		process.env.JOB_ID = 'job-123';
		process.env.SCANNER_OPTIONS = JSON.stringify(['bad']);

		expect(() => loadConfigFromEnv({ scannerName: 'axe' })).toThrow(
			'SCANNER_OPTIONS must be a JSON object'
		);
	});

	it('throws when explicit numeric env values are malformed', () => {
		setRequiredRuntimeEnv();
		process.env.JOB_ID = 'job-123';
		process.env.PAGE_LOAD_TIMEOUT = 'abc';

		expect(() => loadConfigFromEnv({ scannerName: 'axe' })).toThrow(
			'Invalid integer environment variable PAGE_LOAD_TIMEOUT'
		);
	});

	it('validateConfig reports invalid settings', () => {
		const config = {
			jobId: '',
			provenancePath: '',
			resultsDir: '',
			scannerName: '',
			concurrency: 0,
			maxRetries: 0,
			browser: {
				headless: true,
				args: [],
				defaultViewport: { width: 1280, height: 720 },
				deviceScaleFactor: 2,
				defaultTimeout: 1000,
				pageLoadTimeout: 1000
			},
			messaging: {
				url: 'nats://localhost:4222',
				subjects: {
					pageCompleted: 'page.completed',
					scanCompleted: 'scan.completed',
					scanFailed: 'scan.failed'
				}
			}
		} satisfies ScannerConfig;

		expect(() => {
			validateConfig(config);
		}).toThrow('Invalid scanner configuration');
	});
});
