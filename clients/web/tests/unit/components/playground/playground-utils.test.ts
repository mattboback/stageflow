import {
	buildAiNavigatorConfig,
	isZipFilename,
	normalizeUrlInput,
	normalizeUrlListText,
	parseUrlList,
	validateHttpUrls
} from '$lib/components/playground/playground-utils';
import { describe, expect, it } from 'vitest';

describe('playground-utils', () => {
	it('parses newline-separated URLs', () => {
		expect(parseUrlList('a.com\n\n https://b.com  \n')).toEqual(['https://a.com', 'https://b.com']);
	});

	it('normalizes URL input by defaulting to https', () => {
		expect(normalizeUrlInput('example.com')).toBe('https://example.com');
		expect(normalizeUrlInput('https://example.com')).toBe('https://example.com');
		expect(normalizeUrlInput('http://example.com')).toBe('http://example.com');
		expect(normalizeUrlInput('')).toBeNull();
	});

	it('normalizes URL text blocks and reports if changed', () => {
		expect(normalizeUrlListText('example.com\nhttps://b.com')).toEqual({
			text: 'https://example.com\nhttps://b.com',
			changed: true
		});

		expect(normalizeUrlListText('https://a.com\nhttps://b.com')).toEqual({
			text: 'https://a.com\nhttps://b.com',
			changed: false
		});
	});

	it('validates http(s) urls', () => {
		const { valid, invalid } = validateHttpUrls([
			'https://example.com',
			'http://example.com',
			'ftp://example.com',
			'https://'
		]);

		expect(valid).toEqual(['https://example.com', 'http://example.com']);
		expect(invalid).toHaveLength(2);
	});

	it('validates zip filenames', () => {
		expect(isZipFilename('site.zip')).toBe(true);
		expect(isZipFilename(' SITE.ZIP ')).toBe(true);
		expect(isZipFilename('site.tar.gz')).toBe(false);
	});

	it('builds AI navigator config with optional fields', () => {
		const config = buildAiNavigatorConfig({
			objective: '  do the thing ',
			maxSteps: 10,
			maxWallTimeMs: 120_000,
			model: 'openai/gpt-4o-mini',
			inputValues: [
				{ key: ' email ', value: ' test@example.com ' },
				{ key: '', value: 'ignored' }
			],
			successCriteria: [
				{ type: 'url-contains', value: ' /done ' },
				{ type: 'custom', value: '' }
			]
		});

		expect(config).toMatchObject({
			goal: {
				objective: 'do the thing',
				maxSteps: 10,
				maxWallTimeMs: 120_000,
				inputValues: { email: 'test@example.com' },
				successCriteria: [{ type: 'url-contains', value: '/done' }]
			},
			vision: { provider: 'openrouter', model: 'openai/gpt-4o-mini' }
		});
	});
});
