import { describe, expect, it } from 'vitest';
import {
	DEFAULT_AI_CONFIG,
	estimateScanRuntime,
	validatePlaygroundConfiguration,
	type AuthFormConfig
} from './playground-utils';
import { normalizeUrlInput, validateHttpUrls } from '../url';
import type { ScannerDefinition, ScannerSelection } from '../types/scan';

const auth: AuthFormConfig = {
	enabled: false,
	loginUrl: '',
	username: '',
	password: '',
	usernameSelector: '',
	passwordSelector: '',
	submitSelector: '',
	successStrategy: 'auto',
	successSelector: ''
};
const selections: ScannerSelection[] = [{ id: 'axe', enabled: true }];

function scanner(id: string, estimatedTimePerPage?: number): ScannerDefinition {
	return {
		id,
		name: id,
		version: '1',
		description: '',
		categories: [],
		aliases: [],
		enabled: true,
		builtIn: true,
		capabilities: {
			outputFormats: [],
			supportsScreenshots: false,
			supportsConcurrency: true,
			requiresBrowser: false,
			supportsOffline: true,
			maxConcurrency: 1,
			// Omit the key entirely when unset: under exactOptionalPropertyTypes an
			// optional field will not accept an explicit undefined.
			...(estimatedTimePerPage === undefined ? {} : { estimatedTimePerPage })
		}
	};
}

describe('normalizeUrlInput', () => {
	it('normalizes URLs correctly', () => {
		expect(normalizeUrlInput('example.com')).toBe('https://example.com');
		expect(normalizeUrlInput('https://example.com')).toBe('https://example.com');
		expect(normalizeUrlInput('http://example.com')).toBe('http://example.com');
		expect(normalizeUrlInput('//example.com')).toBe('https://example.com');
		expect(normalizeUrlInput('   ')).toBeNull();
	});
});

describe('validateHttpUrls', () => {
	it('accepts valid public domain names', () => {
		const result = validateHttpUrls([
			'https://example.com',
			'http://google.com',
			'https://sub.domain.co.uk'
		]);
		expect(result.valid).toEqual([
			'https://example.com',
			'http://google.com',
			'https://sub.domain.co.uk'
		]);
		expect(result.invalid).toHaveLength(0);
	});

	it('accepts localhost', () => {
		const result = validateHttpUrls(['http://localhost', 'https://localhost:3000']);
		expect(result.valid).toEqual(['http://localhost', 'https://localhost:3000']);
		expect(result.invalid).toHaveLength(0);
	});

	it('accepts IP addresses', () => {
		const result = validateHttpUrls(['http://127.0.0.1', 'http://192.168.1.1', 'http://[::1]']);
		expect(result.valid).toEqual(['http://127.0.0.1', 'http://192.168.1.1', 'http://[::1]']);
		expect(result.invalid).toHaveLength(0);
	});

	it('rejects URLs without protocol or non-HTTP protocols', () => {
		const result = validateHttpUrls(['ftp://example.com', 'file:///bin/bash']);
		expect(result.valid).toHaveLength(0);
		expect(result.invalid).toEqual([
			{ url: 'ftp://example.com', reason: 'URL must start with http:// or https://.' },
			{ url: 'file:///bin/bash', reason: 'URL must start with http:// or https://.' }
		]);
	});

	it('rejects bare words and hostnames without a dot', () => {
		const result = validateHttpUrls(['https://not-a-url', 'http://invalid-host']);
		expect(result.valid).toHaveLength(0);
		expect(result.invalid).toEqual([
			{ url: 'https://not-a-url', reason: 'Hostname must contain a dot or be localhost.' },
			{ url: 'http://invalid-host', reason: 'Hostname must contain a dot or be localhost.' }
		]);
	});
});

describe('validatePlaygroundConfiguration', () => {
	it('returns the normalized URLs used by the submitted payload', () => {
		const result = validatePlaygroundConfiguration({
			mode: 'url',
			urls: [' example.com ', ''],
			file: null,
			selections,
			auth,
			ai: DEFAULT_AI_CONFIG,
			aiEnabled: false
		});
		expect(result.ready).toBe(true);
		expect(result.validUrls).toEqual(['https://example.com']);
	});

	it('blocks invalid targets, incomplete form auth, and invalid AI bounds', () => {
		const invalidUrl = validatePlaygroundConfiguration({
			mode: 'url',
			urls: ['ftp://example.com'],
			file: null,
			selections,
			auth,
			ai: DEFAULT_AI_CONFIG,
			aiEnabled: false
		});
		expect(invalidUrl.focusId).toBe('url-input-0');

		const incompleteAuth = validatePlaygroundConfiguration({
			mode: 'url',
			urls: ['https://example.com'],
			file: null,
			selections,
			auth: { ...auth, enabled: true, loginUrl: 'https://example.com/login' },
			ai: DEFAULT_AI_CONFIG,
			aiEnabled: false
		});
		expect(incompleteAuth.focusId).toBe('auth-username');

		const invalidAi = validatePlaygroundConfiguration({
			mode: 'url',
			urls: ['https://example.com'],
			file: null,
			selections: [{ id: 'ai-navigator', enabled: true }],
			auth,
			ai: { ...DEFAULT_AI_CONFIG, objective: 'Checkout', maxSteps: 51 },
			aiEnabled: true
		});
		expect(invalidAi.focusId).toBe('ai-max-steps');
	});
});

describe('estimateScanRuntime', () => {
	it('uses the slowest parallel scanner and target count to show a range', () => {
		const estimate = estimateScanRuntime(
			[scanner('axe', 5_000), scanner('lighthouse', 30_000)],
			[
				{ id: 'axe', enabled: true },
				{ id: 'lighthouse', enabled: true }
			],
			2,
			'url'
		);
		expect(estimate).toEqual({
			label: '48s–1m 30s',
			detail: '2 pages; scanners run in parallel'
		});
	});

	it('does not invent a numeric estimate for archives or missing catalog data', () => {
		expect(estimateScanRuntime([], selections, 1, 'zip').label).toBe('Varies by archive');
		expect(estimateScanRuntime([scanner('axe')], selections, 1, 'url').label).toBe(
			'Varies by site'
		);
	});
});
