import { describe, expect, it, vi } from 'vitest';

import {
	MAX_VALIDATED_REDIRECTS,
	followValidatedRedirects,
	isRedirectStatus,
	type RedirectHop
} from '../../src/core/redirect-guard';
import { BlockedTargetError } from '../../src/core/target-validation';

/**
 * An allowedOrigins policy validates synchronously against the origin list, with no
 * DNS involved, so these cases stay hermetic. The SSRF invariant under test is
 * that every hop is checked — a public URL that redirects to a blocked origin must
 * be rejected at the hop, not after.
 */
const policy = { allowedOrigins: ['https://allowed.example'] };

/** A transport stub that replays a scripted chain and records requested URLs. */
function scriptedTransport(chain: { status: number; location?: string | null }[]) {
	const requested: string[] = [];
	const request = vi.fn((url: string, hop: number): Promise<RedirectHop<{ id: number }>> => {
		requested.push(url);
		const step = chain[Math.min(hop, chain.length - 1)];
		return Promise.resolve({
			response: { id: hop },
			status: step?.status ?? 200,
			location: step?.location ?? null
		});
	});
	return { request, requested };
}

describe('isRedirectStatus', () => {
	it('recognizes the statuses that carry a Location', () => {
		for (const status of [301, 302, 303, 307, 308]) {
			expect(isRedirectStatus(status)).toBe(true);
		}
		for (const status of [200, 204, 304, 400, 404, 500]) {
			expect(isRedirectStatus(status)).toBe(false);
		}
	});
});

describe('followValidatedRedirects', () => {
	it('returns immediately when the first response is not a redirect', async () => {
		const { request, requested } = scriptedTransport([{ status: 200 }]);

		const result = await followValidatedRedirects('https://allowed.example/a', policy, request);

		expect(result.finalURL).toBe('https://allowed.example/a');
		expect(result.redirects).toEqual([]);
		expect(requested).toEqual(['https://allowed.example/a']);
	});

	it('follows a chain and reports every hop it took', async () => {
		const { request, requested } = scriptedTransport([
			{ status: 302, location: 'https://allowed.example/b' },
			{ status: 302, location: '/c' },
			{ status: 200 }
		]);

		const result = await followValidatedRedirects('https://allowed.example/a', policy, request);

		expect(result.finalURL).toBe('https://allowed.example/c');
		expect(result.redirects).toEqual(['https://allowed.example/b', 'https://allowed.example/c']);
		// Relative Location values resolve against the URL that produced them.
		expect(requested).toEqual([
			'https://allowed.example/a',
			'https://allowed.example/b',
			'https://allowed.example/c'
		]);
	});

	it('rejects a redirect into a blocked origin instead of following it', async () => {
		const { request, requested } = scriptedTransport([
			{ status: 302, location: 'http://169.254.169.254/latest/meta-data' },
			{ status: 200 }
		]);

		await expect(
			followValidatedRedirects('https://allowed.example/a', policy, request)
		).rejects.toThrow(BlockedTargetError);

		// The blocked hop must never be requested: validation precedes the transport.
		expect(requested).toEqual(['https://allowed.example/a']);
	});

	it('bounds the chain rather than looping forever', async () => {
		// Alternates between two allowed URLs, so only the hop cap can stop it.
		const request = vi.fn((url: string): Promise<RedirectHop<{ url: string }>> =>
			Promise.resolve({
				response: { url },
				status: 302,
				location: url.endsWith('/a') ? 'https://allowed.example/b' : 'https://allowed.example/a'
			})
		);

		await expect(
			followValidatedRedirects('https://allowed.example/a', policy, request)
		).rejects.toThrow(/Too many redirects/);

		expect(request).toHaveBeenCalledTimes(MAX_VALIDATED_REDIRECTS + 1);
	});

	it('stops at a redirect with no Location rather than treating it as an error', async () => {
		const { request } = scriptedTransport([{ status: 302, location: null }]);

		const result = await followValidatedRedirects('https://allowed.example/a', policy, request);

		expect(result.finalURL).toBe('https://allowed.example/a');
		expect(result.redirects).toEqual([]);
	});

	it('stops at a malformed Location rather than throwing', async () => {
		const { request } = scriptedTransport([{ status: 301, location: 'http://[not-a-url' }]);

		const result = await followValidatedRedirects('https://allowed.example/a', policy, request);

		expect(result.finalURL).toBe('https://allowed.example/a');
		expect(result.redirects).toEqual([]);
	});
});
