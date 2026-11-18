import type { ReactNode } from 'react';
import { act, renderHook, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { usePlaygroundSession } from './usePlaygroundSession';

afterEach(() => vi.restoreAllMocks());
function mountSession() {
	return renderHook(
		() => usePlaygroundSession({ projectId: null, seedUrl: 'https://example.com' }),
		{ wrapper: ({ children }: { children: ReactNode }) => <MemoryRouter>{children}</MemoryRouter> }
	);
}

describe('configure discovery preview', () => {
	it('requires preview acceptance and preserves explicit manual queries', async () => {
		vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
			const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
			if (url.endsWith('/api/v1/scanners'))
				return Promise.resolve(new Response(JSON.stringify({ scanners: [], categories: [] })));
			if (url.endsWith('/api/v1/discover'))
				return Promise.resolve(
					new Response(
						JSON.stringify({
							urls: ['https://example.com/', 'https://example.com/about'],
							source: 'crawl',
							truncated: false,
							complete: true
						})
					)
				);
			throw new Error('Unexpected API request');
		});
		const { result } = mountSession();
		await waitFor(() => expect(result.current.catalogLoading).toBe(false));
		act(() => result.current.updateUrl(1, 'https://example.com/search?manual=1'));
		await act(async () => result.current.discoverUrls());
		expect(result.current.urls).toEqual([
			'https://example.com',
			'https://example.com/search?manual=1'
		]);
		expect(result.current.discoveryPreview?.result.urls).toHaveLength(2);
		act(() => result.current.applyDiscoveredUrls(['https://example.com/about']));
		expect(result.current.urls).toEqual([
			'https://example.com/about',
			'https://example.com/search?manual=1'
		]);
		expect(result.current.discoveryPreview).toBeNull();
	});
	it('does not send authenticated discovery requests', async () => {
		const mock = vi
			.spyOn(globalThis, 'fetch')
			.mockResolvedValue(new Response(JSON.stringify({ scanners: [], categories: [] })));
		const { result } = mountSession();
		await waitFor(() => expect(result.current.catalogLoading).toBe(false));
		act(() => result.current.setAuthConfig({ ...result.current.authConfig, enabled: true }));
		await act(async () => result.current.discoverUrls());
		expect(mock).toHaveBeenCalledTimes(1);
		expect(result.current.discoverNotice).toContain('public pages only');
	});
});
