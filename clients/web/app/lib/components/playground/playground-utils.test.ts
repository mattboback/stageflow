import { describe, expect, it } from 'vitest';
import { normalizeUrlInput, validateHttpUrls } from './playground-utils';

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
		const result = validateHttpUrls(['https://example.com', 'http://google.com', 'https://sub.domain.co.uk']);
		expect(result.valid).toEqual(['https://example.com', 'http://google.com', 'https://sub.domain.co.uk']);
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
