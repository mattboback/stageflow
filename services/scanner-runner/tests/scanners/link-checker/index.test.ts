/**
 * Link Checker Scanner Tests
 *
 * Tests for pure helper functions in the link checker scanner.
 * Note: Full scanner integration requires Playwright, these tests focus on testable utilities.
 */

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { LinkCheckerScanner } from '../../../src/scanners/link-checker';
import {
	checkSingleLink,
	getSeverityForStatus
} from '../../../src/scanners/link-checker/validation';

const scanner = new LinkCheckerScanner();

describe('LinkCheckerScanner', () => {
	describe('getSeverityForStatus', () => {
		it("returns 'serious' for connection errors (status 0)", () => {
			expect(getSeverityForStatus(0)).toBe('serious');
		});

		it("returns 'serious' for 404 not found", () => {
			expect(getSeverityForStatus(404)).toBe('serious');
		});

		it("returns 'critical' for 5xx server errors", () => {
			expect(getSeverityForStatus(500)).toBe('critical');
			expect(getSeverityForStatus(502)).toBe('critical');
			expect(getSeverityForStatus(503)).toBe('critical');
		});

		it('treats external server errors as moderate', () => {
			expect(getSeverityForStatus(503, false)).toBe('moderate');
		});

		it("returns 'moderate' for other 4xx client errors", () => {
			expect(getSeverityForStatus(400)).toBe('moderate');
			expect(getSeverityForStatus(401)).toBe('moderate');
			expect(getSeverityForStatus(403)).toBe('moderate');
		});

		it("returns 'minor' for success codes", () => {
			expect(getSeverityForStatus(200)).toBe('minor');
			expect(getSeverityForStatus(301)).toBe('minor');
			expect(getSeverityForStatus(302)).toBe('minor');
		});
	});

	describe('checkSingleLink with mocked fetch', () => {
		const originalFetch = globalThis.fetch;

		beforeEach(() => {
			vi.useFakeTimers();
		});

		afterEach(() => {
			globalThis.fetch = originalFetch;
			delete process.env.SCAN_URLS;
			vi.useRealTimers();
		});

		it('returns success for 200 response', async () => {
			globalThis.fetch = vi.fn().mockResolvedValue({
				status: 200,
				redirected: false,
				url: 'https://example.com'
			});

			const resultPromise = checkSingleLink('https://example.com');

			await vi.advanceTimersByTimeAsync(0);
			const result = await resultPromise;

			expect(result.status).toBe(200);
			expect(result.error).toBeNull();
		});

		it('reselects cookies for every redirect destination without leaking the first session', async () => {
			const fetchMock = vi
				.fn()
				.mockResolvedValueOnce({
					status: 302,
					headers: new Headers({ location: 'https://other.example/final' })
				})
				.mockResolvedValueOnce({ status: 200, headers: new Headers() });
			globalThis.fetch = fetchMock;
			const cookiesForURL = vi.fn((url: string) =>
				Promise.resolve(new URL(url).origin === 'https://example.com' ? 'session=secret' : '')
			);
			const result = await checkSingleLink(
				'https://example.com/start#anchor',
				{ allowedOrigins: ['https://example.com', 'https://other.example'] },
				cookiesForURL
			);
			expect(result.status).toBe(200);
			expect(fetchMock.mock.calls[0]?.[0]).toBe('https://example.com/start');
			expect(fetchMock.mock.calls[0]?.[1].headers.Cookie).toBe('session=secret');
			expect(fetchMock.mock.calls[1]?.[1].headers.Cookie).toBeUndefined();
			expect(cookiesForURL).toHaveBeenCalledWith('https://other.example/final');
		});

		it('returns error for network failure', async () => {
			globalThis.fetch = vi.fn().mockRejectedValue(new Error('Network error'));

			const resultPromise = checkSingleLink('https://example.com');

			await vi.advanceTimersByTimeAsync(0);
			const result = await resultPromise;

			expect(result.status).toBeNull();
			expect(result.error).toBe('Network error');
		});

		it('tracks redirects', async () => {
			globalThis.fetch = vi
				.fn()
				.mockResolvedValueOnce({
					status: 302,
					url: 'https://example.com',
					headers: {
						get: vi.fn().mockReturnValue('https://example.com/final')
					}
				})
				.mockResolvedValueOnce({
					status: 200,
					url: 'https://example.com/final',
					headers: {
						get: vi.fn().mockReturnValue(null)
					}
				});

			const resultPromise = checkSingleLink('https://example.com');

			await vi.advanceTimersByTimeAsync(0);
			const result = await resultPromise;

			expect(result.redirects).toContain('https://example.com/final');
		});

		it('falls back to GET when HEAD fails', async () => {
			const mockFetch = vi
				.fn()
				.mockRejectedValueOnce(new Error('HEAD not supported'))
				.mockResolvedValueOnce({
					status: 200,
					redirected: false,
					url: 'https://example.com'
				});

			globalThis.fetch = mockFetch;

			const resultPromise = checkSingleLink('https://example.com');

			await vi.advanceTimersByTimeAsync(0);
			const result = await resultPromise;

			expect(mockFetch).toHaveBeenCalledTimes(2);
			expect(result.status).toBe(200);
		});

		it('blocks non-public link targets during URL jobs', async () => {
			const mockFetch = vi.fn();
			globalThis.fetch = mockFetch;

			const resultPromise = checkSingleLink('https://127.0.0.1/private');
			await vi.advanceTimersByTimeAsync(0);
			const result = await resultPromise;

			expect(result.status).toBeNull();
			expect(result.error).toContain('Blocked target URL');
			expect(mockFetch).not.toHaveBeenCalled();
		});

		it('allows links on the exact static local origin from policy', async () => {
			const mockFetch = vi.fn().mockResolvedValue({
				status: 200,
				redirected: false,
				url: 'http://localhost:4173/page'
			});
			globalThis.fetch = mockFetch;

			const resultPromise = checkSingleLink('http://localhost:4173/page', {
				allowedOrigins: ['http://localhost:4173']
			});

			await vi.advanceTimersByTimeAsync(0);
			const result = await resultPromise;

			expect(result.status).toBe(200);
			expect(result.error).toBeNull();
			expect(mockFetch).toHaveBeenCalled();
		});
	});

	describe('metadata', () => {
		it('has correct scanner name', () => {
			expect(scanner.metadata.name).toBe('link-checker');
		});

		it('has version', () => {
			expect(scanner.metadata.version).toBeDefined();
		});
	});
});
