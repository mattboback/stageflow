import { fireEvent, render, screen } from '@testing-library/react';
import { createRoutesStub } from 'react-router';
import { afterEach, describe, expect, it, vi } from 'vitest';
import Home from './home';

afterEach(() => vi.restoreAllMocks());
function mountHome() {
	const Stub = createRoutesStub([
		{ path: '/', Component: Home },
		{ path: '/scan/:jobId', Component: () => <h1>Submitted scan</h1> }
	]);
	render(<Stub initialEntries={['/']} />);
}

describe('homepage discovery', () => {
	it('requires selection confirmation before submission', async () => {
		let submission: Record<string, unknown> | null = null;
		const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation((input, init) => {
			const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
			if (url.endsWith('/api/v1/discover'))
				return Promise.resolve(
					new Response(
						JSON.stringify({
							urls: ['https://example.com/', 'https://example.com/about'],
							source: 'sitemap',
							truncated: false,
							complete: true,
							warnings: []
						}),
						{ status: 200 }
					)
				);
			if (url.endsWith('/api/v1/jobs/urls')) {
				submission = JSON.parse(typeof init?.body === 'string' ? init.body : '') as Record<
					string,
					unknown
				>;
				return Promise.resolve(
					new Response(JSON.stringify({ job_id: 'discovery-job' }), { status: 201 })
				);
			}
			throw new Error(`Unexpected API request ${url}`);
		});
		mountHome();
		fireEvent.change(screen.getByRole('textbox', { name: 'URL to scan' }), {
			target: { value: 'https://example.com' }
		});
		fireEvent.click(screen.getByRole('button', { name: 'Scan website' }));
		await screen.findByRole('region', { name: 'Website discovery preview' });
		expect(submission).toBeNull();
		expect(fetchMock).toHaveBeenCalledTimes(1);
		fireEvent.click(screen.getByRole('checkbox', { name: 'https://example.com/about' }));
		fireEvent.click(screen.getByRole('button', { name: 'Scan selected pages' }));
		await screen.findByRole('heading', { name: 'Submitted scan' });
		expect(submission).toMatchObject({
			urls: ['https://example.com/'],
			modules: ['axe', 'lighthouse', 'link-checker', 'security-headers', 'seo'],
			screenshot: true
		});
	});
	it('preserves explicit query URLs without discovery', async () => {
		const calls: string[] = [];
		vi.spyOn(globalThis, 'fetch').mockImplementation((input, init) => {
			calls.push(typeof input === 'string' ? input : input instanceof URL ? input.href : input.url);
			expect(JSON.parse(typeof init?.body === 'string' ? init.body : '')).toMatchObject({
				urls: ['https://example.com/?explicit=1']
			});
			return Promise.resolve(
				new Response(JSON.stringify({ job_id: 'manual-job' }), { status: 201 })
			);
		});
		mountHome();
		fireEvent.change(screen.getByRole('textbox', { name: 'URL to scan' }), {
			target: { value: 'https://example.com/?explicit=1' }
		});
		fireEvent.click(screen.getByRole('button', { name: 'Scan' }));
		await screen.findByRole('heading', { name: 'Submitted scan' });
		expect(calls).toHaveLength(1);
		expect(calls[0]).toContain('/api/v1/jobs/urls');
	});
});
