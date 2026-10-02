import { afterEach, describe, expect, it, vi } from 'vitest';

afterEach(() => {
	vi.unstubAllEnvs();
	vi.resetModules();
});

describe('deployment scan policy', () => {
	it.each([
		['hosted-demo', 25, false],
		['self-hosted', 100, true]
	])('sets %s limits', async (mode, max, zip) => {
		vi.stubEnv('VITE_DEPLOYMENT_MODE', mode);
		vi.resetModules();
		const policy = await import('./scan-policy');
		expect(policy.MAX_SCAN_PAGES).toBe(max);
		expect(policy.ZIP_UPLOAD_ENABLED).toBe(zip);
	});
	it('rejects a hosted scan above 25 URLs and accepts the boundary', async () => {
		vi.stubEnv('VITE_DEPLOYMENT_MODE', 'hosted-demo');
		vi.resetModules();
		const { validatePlaygroundConfiguration } = await import('./playground/playground-utils');
		const config = {
			mode: 'url' as const,
			urls: Array.from({ length: 25 }, (_, i) => `https://example.com/${i}`),
			file: null,
			selections: [{ id: 'axe', enabled: true }],
			auth: {
				enabled: false,
				loginUrl: '',
				username: '',
				password: '',
				usernameSelector: '',
				passwordSelector: '',
				submitSelector: '',
				successStrategy: 'auto' as const,
				successSelector: ''
			}
		};
		expect(validatePlaygroundConfiguration(config).ready).toBe(true);
		const rejected = validatePlaygroundConfiguration({
			...config,
			urls: [...config.urls, 'https://example.com/extra']
		});
		expect(rejected.ready).toBe(false);
		expect(rejected.message).toContain('25 URLs');
		expect(
			validatePlaygroundConfiguration({
				...config,
				mode: 'zip',
				file: new File(['build'], 'site.zip', { type: 'application/zip' })
			}).ready
		).toBe(false);
	});
});
