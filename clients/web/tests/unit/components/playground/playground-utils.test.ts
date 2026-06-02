import {
	buildAiNavigatorConfig,
	buildFormAuthConfig,
	isAuthConfigComplete,
	isZipFilename,
	MAX_ZIP_UPLOAD_BYTES,
	normalizeUrlInput,
	normalizeUrlListText,
	parseUrlList,
	validateZipUploadFile,
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

	it('validates zip upload file names and sizes', () => {
		expect(validateZipUploadFile(new File(['ok'], 'site.zip'))).toBeNull();
		expect(validateZipUploadFile(new File(['ok'], 'site.txt'))).toBe('Please select a ZIP file');

		const oversized = new File([new Uint8Array(1)], 'site.zip');
		Object.defineProperty(oversized, 'size', { value: MAX_ZIP_UPLOAD_BYTES + 1 });
		expect(validateZipUploadFile(oversized)).toBe('ZIP file must be 100MB or smaller');
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

	it('auto-detects login (no success selector) when only credentials are provided', () => {
		const config = {
			enabled: true,
			loginUrl: ' https://app.example.com/login ',
			username: 'demo@example.com',
			password: 'secret',
			usernameSelector: '',
			passwordSelector: '',
			submitSelector: '',
			successStrategy: 'auto' as const,
			successSelector: ''
		};

		// Login URL + username + password are the only required fields now.
		expect(isAuthConfigComplete(config)).toBe(true);
		expect(buildFormAuthConfig(config)).toEqual({
			mode: 'form',
			form: {
				login_url: 'https://app.example.com/login',
				steps: [
					{ type: 'fill', selector: 'auto:username', value: 'demo@example.com' },
					{ type: 'fill', selector: 'auto:password', value: 'secret' },
					{ type: 'click', selector: 'auto:submit' }
				],
				success: { type: 'networkidle' }
			}
		});
	});

	it('still requires login URL, username, and password', () => {
		const base = {
			enabled: true,
			loginUrl: '',
			username: 'demo@example.com',
			password: 'secret',
			usernameSelector: '',
			passwordSelector: '',
			submitSelector: '',
			successStrategy: 'auto' as const,
			successSelector: ''
		};

		expect(isAuthConfigComplete(base)).toBe(false);
		expect(buildFormAuthConfig(base)).toBeNull();
	});

	it('builds form auth with selector-based success detection', () => {
		const config = buildFormAuthConfig({
			enabled: true,
			loginUrl: ' https://app.example.com/login ',
			username: 'demo@example.com',
			password: 'secret',
			usernameSelector: '',
			passwordSelector: '',
			submitSelector: '',
			successStrategy: 'selector',
			successSelector: ' a[href="/dashboard"] '
		});

		expect(config).toEqual({
			mode: 'form',
			form: {
				login_url: 'https://app.example.com/login',
				steps: [
					{ type: 'fill', selector: 'auto:username', value: 'demo@example.com' },
					{ type: 'fill', selector: 'auto:password', value: 'secret' },
					{ type: 'click', selector: 'auto:submit' }
				],
				success: { type: 'selector', selector: 'a[href="/dashboard"]' }
			}
		});
	});
});
