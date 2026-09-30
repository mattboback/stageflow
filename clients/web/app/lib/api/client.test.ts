import { afterEach, describe, expect, it, vi } from 'vitest';

import { deleteScanJob, submitScanJob } from './client';

const baseParams = {
	mode: 'url' as const,
	file: null,
	urls: ['https://example.com'],
	scanners: [{ id: 'axe', enabled: true }]
};

/**
 * The URL a fetch mock was called with, for whichever RequestInfo form was used.
 *
 * `String(input)` would stringify a Request object to "[object Request]" and make
 * every endsWith assertion below silently false, so the shape is narrowed instead.
 */
function requestUrl(input: RequestInfo | URL | undefined): string {
	if (input === undefined) throw new Error('fetch was not called');
	if (typeof input === 'string') return input;
	if (input instanceof URL) return input.href;
	return input.url;
}

/** The JSON body a fetch mock was called with. */
function requestBody(init: RequestInit | undefined): string {
	const body = init?.body;
	if (typeof body !== 'string') {
		throw new Error('expected a string request body');
	}
	return body;
}

afterEach(() => {
	vi.unstubAllGlobals();
});

describe('browser URL submission boundary', () => {
	it('uses the canonical route without auth or an API key when no recipe is present', async () => {
		const fetchMock = vi.fn<(input: RequestInfo | URL, init?: RequestInit) => Promise<Response>>(
			() =>
				Promise.resolve(
					new Response(JSON.stringify({ job_id: 'job-anonymous' }), {
						status: 200,
						headers: { 'Content-Type': 'application/json' }
					})
				)
		);
		vi.stubGlobal('fetch', fetchMock);

		await submitScanJob(baseParams);

		expect(new URL(requestUrl(fetchMock.mock.calls[0]?.[0]), 'https://web.test').pathname).toBe(
			'/api/v1/jobs/urls'
		);
		const request = fetchMock.mock.calls[0]?.[1];
		expect(request?.method).toBe('POST');
		expect(request?.headers).toEqual({ 'Content-Type': 'application/json' });
		expect(JSON.parse(requestBody(request))).toEqual({
			urls: baseParams.urls,
			modules: ['axe'],
			profile: 'production',
			scanner_configs: {},
			screenshot: true
		});
	});

	it('uses the canonical route with the unchanged literal form payload', async () => {
		const fetchMock = vi.fn<(input: RequestInfo | URL, init?: RequestInit) => Promise<Response>>(
			() =>
				Promise.resolve(
					new Response(JSON.stringify({ job_id: 'job-auth' }), {
						status: 200,
						headers: { 'Content-Type': 'application/json' }
					})
				)
		);
		vi.stubGlobal('fetch', fetchMock);
		const auth = {
			mode: 'form',
			form: {
				login_url: 'https://example.com/login',
				steps: [{ type: 'fill', selector: '#password', value: 'literal p@ss+word' }]
			}
		};

		await submitScanJob({ ...baseParams, auth });

		expect(new URL(requestUrl(fetchMock.mock.calls[0]?.[0]), 'https://web.test').pathname).toBe(
			'/api/v1/jobs/urls'
		);
		expect(fetchMock.mock.calls[0]?.[1]?.headers).toEqual({ 'Content-Type': 'application/json' });
		expect(JSON.parse(requestBody(fetchMock.mock.calls[0]?.[1]))).toMatchObject({ auth });
	});
});

describe('deleteScanJob', () => {
	it('treats 204 as success', async () => {
		const fetchMock = vi.fn<(input: RequestInfo | URL, init?: RequestInit) => Promise<Response>>(
			() => Promise.resolve(new Response(null, { status: 204 }))
		);
		vi.stubGlobal('fetch', fetchMock);

		await expect(deleteScanJob('job-1')).resolves.toBeUndefined();
		expect(requestUrl(fetchMock.mock.calls[0]?.[0]).endsWith('/api/v1/jobs/job-1')).toBe(true);
		expect(fetchMock.mock.calls[0]?.[1]?.method).toBe('DELETE');
	});

	it('maps 409 to a still-running error', async () => {
		vi.stubGlobal('fetch', () =>
			Promise.resolve(
				new Response(JSON.stringify({ error: 'Job is still running' }), {
					status: 409,
					headers: { 'Content-Type': 'application/json' }
				})
			)
		);

		await expect(deleteScanJob('job-running')).rejects.toThrow(/still running/i);
	});
});
