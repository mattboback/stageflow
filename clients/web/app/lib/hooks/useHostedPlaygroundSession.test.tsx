import type { ReactNode } from 'react';
import { act, renderHook, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { usePlaygroundSession } from './usePlaygroundSession';

vi.mock('../scan-policy', () => ({ MAX_SCAN_PAGES: 25, ZIP_UPLOAD_ENABLED: false }));
afterEach(() => vi.restoreAllMocks());

function mountSession() {
	vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
		const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
		if (url.endsWith('/api/v1/scanners'))
			return Promise.resolve(new Response(JSON.stringify({ scanners: [], categories: [] })));
		if (url.endsWith('/api/v1/discover'))
			return Promise.resolve(
				new Response(
					JSON.stringify({
						urls: Array.from({ length: 25 }, (_, i) => `https://example.com/${i}`),
						source: 'crawl',
						complete: false,
						truncated: true
					})
				)
			);
		throw new Error('Unexpected API request');
	});
	return renderHook(
		() => usePlaygroundSession({ projectId: null, seedUrl: 'https://example.com' }),
		{ wrapper: ({ children }: { children: ReactNode }) => <MemoryRouter>{children}</MemoryRouter> }
	);
}

describe('hosted configure intake', () => {
	it('rejects oversized pasted lists without dropping URLs and caps manual rows', async () => {
		const { result } = mountSession();
		await waitFor(() => expect(result.current.catalogLoading).toBe(false));
		const before = result.current.urls;
		act(() => {
			result.current.pasteUrls(
				0,
				Array.from({ length: 26 }, (_, i) => `https://example.com/${i}`).join('\n')
			);
		});
		expect(result.current.urls).toEqual(before);
		expect(result.current.error).toContain('25 URLs');
		for (let i = 0; i < 30; i++) act(() => result.current.addUrlRow());
		expect(result.current.urls).toHaveLength(25);
		act(() => result.current.setMode('zip'));
		expect(result.current.mode).toBe('url');
	});
	it('rejects discovery plus manual URLs above the cap without silently removing manual URLs', async () => {
		const { result } = mountSession();
		await waitFor(() => expect(result.current.catalogLoading).toBe(false));
		act(() => result.current.updateUrl(1, 'https://example.com/manual?explicit=1'));
		await act(async () => result.current.discoverUrls());
		const selected = result.current.discoveryPreview!.result.urls;
		act(() => result.current.applyDiscoveredUrls(selected));
		expect(result.current.discoveryPreview).not.toBeNull();
		expect(result.current.discoverNotice).toContain('25 URLs');
		expect(result.current.urls).toContain('https://example.com/manual?explicit=1');
		act(() => result.current.applyDiscoveredUrls(selected.slice(0, 24)));
		expect(result.current.urls).toHaveLength(25);
		expect(result.current.urls).toContain('https://example.com/manual?explicit=1');
	});
});
