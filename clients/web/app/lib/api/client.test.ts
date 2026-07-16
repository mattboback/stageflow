import { afterEach, describe, expect, it, vi } from 'vitest';

import { submitScanJob } from './client';

const baseParams = {
	mode: 'url' as const,
	file: null,
	urls: ['https://example.com'],
	scanners: [{ id: 'axe', enabled: true }],
	highlightStyle: 'solid' as const
};

afterEach(() => {
	vi.unstubAllGlobals();
});

describe('browser URL submission boundary', () => {
	it('uses the anonymous route when no authentication recipe is present', async () => {
		const fetchMock = vi.fn<(input: RequestInfo | URL, init?: RequestInit) => Promise<Response>>(async () =>
			new Response(JSON.stringify({ job_id: 'job-anonymous' }), {
				status: 200,
				headers: { 'Content-Type': 'application/json' }
			})
		);
		vi.stubGlobal('fetch', fetchMock);

		await submitScanJob(baseParams);

		expect(String(fetchMock.mock.calls[0]?.[0]).endsWith('/api/v1/jobs/urls/anonymous')).toBe(true);
	});

	it('uses the constrained browser-auth route when form authentication is present', async () => {
		const fetchMock = vi.fn<(input: RequestInfo | URL, init?: RequestInit) => Promise<Response>>(async () =>
			new Response(JSON.stringify({ job_id: 'job-auth' }), {
				status: 200,
				headers: { 'Content-Type': 'application/json' }
			})
		);
		vi.stubGlobal('fetch', fetchMock);
		const auth = { mode: 'form', form: { login_url: 'https://example.com/login', steps: [] } };

		await submitScanJob({ ...baseParams, auth });

		expect(String(fetchMock.mock.calls[0]?.[0]).endsWith('/api/v1/jobs/urls/browser-auth')).toBe(true);
		const request = fetchMock.mock.calls[0]?.[1] as RequestInit;
		expect(JSON.parse(String(request.body))).toMatchObject({ auth });
	});
});
