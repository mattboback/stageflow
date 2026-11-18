import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { ScannerConfig } from '../../src/core/types';

import { loadConfigFromEnv, validateConfig } from '../../src/core/config-loader';

const ORIGINAL_ENV = process.env;

function resetEnv(): void {
	process.env = { ...ORIGINAL_ENV };
	delete process.env.NODE_ENV;
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
		it('succeeds when all required env vars are set', () => {
			process.env.JOB_ID = 'job-123';

			const config = loadConfigFromEnv({ scannerName: 'axe' });
			expect(config.provenancePath).toBe('/job/provenance.json');
			expect(config.resultsDir).toBe('/out');
		});
	});

	it('throws when scanner options JSON is invalid', () => {
		process.env.JOB_ID = 'job-123';
		process.env.SCANNER_OPTIONS = '{nope';

		expect(() => loadConfigFromEnv({ scannerName: 'axe' })).toThrow(
			'Failed to parse SCANNER_OPTIONS'
		);
	});

	it('throws when scanner options are not an object', () => {
		process.env.JOB_ID = 'job-123';
		process.env.SCANNER_OPTIONS = JSON.stringify(['bad']);

		expect(() => loadConfigFromEnv({ scannerName: 'axe' })).toThrow(
			'SCANNER_OPTIONS must be a JSON object'
		);
	});

	it('throws when explicit numeric env values are malformed', () => {
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
			}
		} satisfies ScannerConfig;

		expect(() => {
			validateConfig(config);
		}).toThrow('Invalid scanner configuration');
	});
});
